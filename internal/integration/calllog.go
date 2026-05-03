package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CallEntry mirrors one row of integration_call_log. Used both as the
// internal write payload and the JSON shape returned by the read API.
type CallEntry struct {
	ID            int64     `json:"id"`
	Time          time.Time `json:"ts"`
	IntegrationID string    `json:"integration_id"`
	Direction     string    `json:"direction"` // inbound | outbound
	Method        string    `json:"method"`
	Path          string    `json:"path"`
	StatusCode    int       `json:"status_code"`
	DurationMS    int       `json:"duration_ms"`
	BytesWritten  *int64    `json:"bytes_written,omitempty"`
	RequestID     string    `json:"request_id,omitempty"`
	ResourceKind  string    `json:"resource_kind,omitempty"`
	ResourceID    string    `json:"resource_id,omitempty"`
}

// CallQueryOpts is the filter set for CallLogger.Query.
type CallQueryOpts struct {
	IntegrationID string
	Direction     string // "inbound" | "outbound" | ""
	// StatusClass narrows by HTTP status family. 2 = 200-299, 4 = 400-499,
	// 5 = 500-599. Zero means no filter.
	StatusClass int
	Since       time.Time
	Until       time.Time
	// Cursor is the last id from the previous page; rows returned all
	// have id < Cursor (descending). Zero means "start from the top".
	Cursor int64
	Limit  int
}

const (
	callLogDefaultLimit = 100
	callLogMaxLimit     = 500
	// 256 — same buffer size as audit.Sink. A burst >256 is the only
	// thing that drops; per-request writes are sub-ms so the queue
	// drains fast enough to make this comfortably oversized.
	callLogQueueSize = 256
)

// CallLogger persists per-call API audit data for integration-attributed
// traffic. Pattern mirrors audit.Sink: a single goroutine consumes a
// buffered channel of writes; producers (HTTP middleware, proxy handler)
// never block on the database.
//
// TODO(adr-0010): batched writes. Right now every call is one INSERT.
// At sustained >10 writes/sec this becomes the bottleneck — switch to
// batching ~50–200 rows per transaction with a 1s flush.
//
// TODO(adr-0010): retention. The table grows forever. When row count
// crosses ~100k OR table size > 100MB, add a periodic worker that
// DELETEs rows older than N days (N likely 7–30 — much shorter than
// audit_log because per-call volume is much higher).
//
// TODO(adr-0010): stats helpers. When real traffic exists, add
// per-integration aggregation methods (calls/min, error rate, p95
// latency over last N hours) for the deferred KPI cards.
type CallLogger struct {
	pool *pgxpool.Pool
	log  *slog.Logger
	ch   chan writeMsg
	done chan struct{}
	wg   sync.WaitGroup
}

type writeMsg struct {
	integrationID string
	direction     string
	method        string
	path          string
	statusCode    int
	durationMS    int
	bytesWritten  int64
	hasBytes      bool
	requestID     string
	// TODO(adr-0010): when path-to-resource extraction lands, add
	// resourceKind/resourceID fields here, populate in the producers
	// (Middleware + LogOutbound), and extend write() to pass them.
}

func NewCallLogger(pool *pgxpool.Pool, log *slog.Logger) *CallLogger {
	if log == nil {
		log = slog.Default()
	}
	cl := &CallLogger{
		pool: pool,
		log:  log.With("component", "integration.calllog"),
		ch:   make(chan writeMsg, callLogQueueSize),
		done: make(chan struct{}),
	}
	cl.wg.Add(1)
	go cl.consume()
	return cl
}

// Close stops accepting new writes and waits for the queue to drain.
func (cl *CallLogger) Close() {
	close(cl.ch)
	<-cl.done
}

// record enqueues a write. Non-blocking; if the queue is full the
// entry is dropped and a warning is logged. Dropping is preferable
// to blocking the request hot path.
func (cl *CallLogger) record(msg writeMsg) {
	select {
	case cl.ch <- msg:
	default:
		cl.log.Warn("call log dropped: queue full",
			"integration", msg.integrationID, "path", msg.path)
	}
}

func (cl *CallLogger) consume() {
	defer cl.wg.Done()
	for msg := range cl.ch {
		cl.write(msg)
	}
	close(cl.done)
}

func (cl *CallLogger) write(msg writeMsg) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var bw any
	if msg.hasBytes {
		bw = msg.bytesWritten
	}
	var reqID any
	if msg.requestID != "" {
		reqID = msg.requestID
	}

	_, err := cl.pool.Exec(ctx, `
        INSERT INTO integration_call_log
            (integration_id, direction, method, path, status_code, duration_ms, bytes_written, request_id)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		msg.integrationID, msg.direction, msg.method, msg.path,
		msg.statusCode, msg.durationMS, bw, reqID,
	)
	if err != nil {
		cl.log.Error("insert call log",
			"err", err, "integration", msg.integrationID, "path", msg.path)
	}
}

// LogOutbound is called explicitly from ProxyHandlers after a proxied
// call completes. The principal is admin (browser), the integration
// is the proxy *target*.
func (cl *CallLogger) LogOutbound(
	ctx context.Context,
	integrationID, method, path string,
	statusCode, durationMS int,
	bytesWritten int64,
	requestID string,
) {
	cl.record(writeMsg{
		integrationID: integrationID,
		direction:     "outbound",
		method:        method,
		path:          path,
		statusCode:    statusCode,
		durationMS:    durationMS,
		bytesWritten:  bytesWritten,
		hasBytes:      true,
		requestID:     requestID,
	})
}

// Query returns entries newest-first. NextCursor is the id of the
// last returned row; pass it as opts.Cursor for the next page (zero
// means there are no more rows).
func (cl *CallLogger) Query(ctx context.Context, opts CallQueryOpts) ([]CallEntry, int64, error) {
	if opts.Limit <= 0 {
		opts.Limit = callLogDefaultLimit
	}
	if opts.Limit > callLogMaxLimit {
		opts.Limit = callLogMaxLimit
	}

	q, args := buildCallQuery(opts)
	rows, err := cl.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("calllog: query: %w", err)
	}
	defer rows.Close()

	out := make([]CallEntry, 0, opts.Limit)
	for rows.Next() {
		var e CallEntry
		var bytesWritten *int64
		var requestID, resourceKind, resourceID *string
		if err := rows.Scan(
			&e.ID, &e.Time, &e.IntegrationID, &e.Direction,
			&e.Method, &e.Path, &e.StatusCode, &e.DurationMS,
			&bytesWritten, &requestID, &resourceKind, &resourceID,
		); err != nil {
			return nil, 0, fmt.Errorf("calllog: scan: %w", err)
		}
		e.BytesWritten = bytesWritten
		if requestID != nil {
			e.RequestID = *requestID
		}
		if resourceKind != nil {
			e.ResourceKind = *resourceKind
		}
		if resourceID != nil {
			e.ResourceID = *resourceID
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("calllog: rows: %w", err)
	}

	var nextCursor int64
	if len(out) == opts.Limit {
		nextCursor = out[len(out)-1].ID
	}
	return out, nextCursor, nil
}

// MarshalJSON ensures the timestamp is always RFC3339 with millisecond
// precision regardless of pgx's ts/tstz formatting.
func (e CallEntry) MarshalJSON() ([]byte, error) {
	type alias CallEntry
	return json.Marshal(struct {
		alias
		Time string `json:"ts"`
	}{
		alias: alias(e),
		Time:  e.Time.UTC().Format(time.RFC3339Nano),
	})
}

// buildCallQuery is the SQL builder; split out for unit-testing
// without a live database. Returns parameterized SQL + args.
func buildCallQuery(opts CallQueryOpts) (string, []any) {
	var (
		conds []string
		args  []any
	)
	add := func(cond string, val any) {
		args = append(args, val)
		conds = append(conds, fmt.Sprintf(cond, len(args)))
	}

	if opts.IntegrationID != "" {
		add("integration_id = $%d", opts.IntegrationID)
	}
	if opts.Direction != "" {
		add("direction = $%d", opts.Direction)
	}
	switch opts.StatusClass {
	case 2:
		conds = append(conds, "status_code BETWEEN 200 AND 299")
	case 3:
		conds = append(conds, "status_code BETWEEN 300 AND 399")
	case 4:
		conds = append(conds, "status_code BETWEEN 400 AND 499")
	case 5:
		conds = append(conds, "status_code BETWEEN 500 AND 599")
	}
	if !opts.Since.IsZero() {
		add("ts >= $%d", opts.Since)
	}
	if !opts.Until.IsZero() {
		add("ts < $%d", opts.Until)
	}
	if opts.Cursor > 0 {
		add("id < $%d", opts.Cursor)
	}

	q := `SELECT id, ts, integration_id, direction, method, path, status_code,
                  duration_ms, bytes_written, request_id, resource_kind, resource_id
          FROM integration_call_log`
	if len(conds) > 0 {
		q += " WHERE " + strings.Join(conds, " AND ")
	}
	q += " ORDER BY id DESC"

	args = append(args, opts.Limit)
	q += fmt.Sprintf(" LIMIT $%d", len(args))
	return q, args
}

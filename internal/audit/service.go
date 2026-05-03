package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Entry mirrors one row of audit_log. Metadata is the raw JSON payload
// of the originating event — callers parse it as needed.
type Entry struct {
	ID          int64           `json:"id"`
	Time        time.Time       `json:"ts"`
	ActorKind   string          `json:"actor_kind"`
	ActorID     string          `json:"actor_id,omitempty"`
	Action      string          `json:"action"`
	SubjectKind string          `json:"subject_kind,omitempty"`
	SubjectID   string          `json:"subject_id,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
}

// QueryOpts is the filter set for Service.Query. Zero values mean
// "no filter" except Limit which gets a default.
type QueryOpts struct {
	SubjectKind string
	SubjectID   string
	// Action filters by topic. A value ending with ".*" is treated as
	// a prefix match (e.g. "session.*"); otherwise it's an exact match.
	Action string
	Since  time.Time
	Until  time.Time
	// Cursor is the last id from the previous page. Rows returned all
	// have id < Cursor (descending). Zero means "start from the top".
	Cursor int64
	Limit  int
}

const (
	defaultLimit = 100
	maxLimit     = 500
)

// Service exposes read-only queries over audit_log. The write side
// lives in Sink (sink.go) and runs in its own goroutine.
type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// Query returns entries newest-first. NextCursor is the id of the
// last returned row; pass it back as opts.Cursor for the next page,
// or 0 means "no more pages" (fewer rows than Limit returned).
func (s *Service) Query(ctx context.Context, opts QueryOpts) (entries []Entry, nextCursor int64, err error) {
	if opts.Limit <= 0 {
		opts.Limit = defaultLimit
	}
	if opts.Limit > maxLimit {
		opts.Limit = maxLimit
	}

	q, args := buildQuery(opts)

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("audit: query: %w", err)
	}
	defer rows.Close()

	out := make([]Entry, 0, opts.Limit)
	for rows.Next() {
		var e Entry
		var actorID, subjectKind, subjectID *string
		var metadata []byte
		if err := rows.Scan(
			&e.ID, &e.Time, &e.ActorKind, &actorID, &e.Action,
			&subjectKind, &subjectID, &metadata,
		); err != nil {
			return nil, 0, fmt.Errorf("audit: scan: %w", err)
		}
		if actorID != nil {
			e.ActorID = *actorID
		}
		if subjectKind != nil {
			e.SubjectKind = *subjectKind
		}
		if subjectID != nil {
			e.SubjectID = *subjectID
		}
		if len(metadata) > 0 {
			e.Metadata = json.RawMessage(metadata)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("audit: rows: %w", err)
	}

	// Only signal a next page when we filled the page exactly. A
	// short page means the caller has reached the tail.
	if len(out) == opts.Limit {
		nextCursor = out[len(out)-1].ID
	}
	return out, nextCursor, nil
}

// buildQuery is split out so the SQL can be exercised by unit tests
// without a live database. It returns a paramaterized query and the
// matching argument slice.
func buildQuery(opts QueryOpts) (string, []any) {
	var (
		conds []string
		args  []any
	)

	add := func(cond string, val any) {
		args = append(args, val)
		conds = append(conds, fmt.Sprintf(cond, len(args)))
	}

	if opts.SubjectKind != "" {
		add("subject_kind = $%d", opts.SubjectKind)
	}
	if opts.SubjectID != "" {
		add("subject_id = $%d", opts.SubjectID)
	}
	if opts.Action != "" {
		if strings.HasSuffix(opts.Action, ".*") {
			prefix := strings.TrimSuffix(opts.Action, "*")
			add("action LIKE $%d", prefix+"%")
		} else {
			add("action = $%d", opts.Action)
		}
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

	q := `SELECT id, ts, actor_kind, actor_id, action, subject_kind, subject_id, metadata
          FROM audit_log`
	if len(conds) > 0 {
		q += " WHERE " + strings.Join(conds, " AND ")
	}
	q += " ORDER BY id DESC"

	args = append(args, opts.Limit)
	q += fmt.Sprintf(" LIMIT $%d", len(args))

	return q, args
}

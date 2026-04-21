package host

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"sync"
)

// Notification is an inbound RPC that doesn't expect a response. LSP
// uses them heavily (publishDiagnostics, window/logMessage, etc.).
type Notification struct {
	Method string
	Params json.RawMessage
}

// RPCHandler resolves sidecar → host requests. The Mux routes inbound
// RPCs with an ID through this handler and writes the result back on
// the same stream. Handlers should return quickly — blocking handlers
// head-of-line-block the single reader goroutine.
//
// Returning an error marshalled as an RPCError with code = RPCErrInternal
// unless the error is a *RPCError itself (in which case the struct is
// sent verbatim).
type RPCHandler interface {
	Handle(ctx context.Context, method string, params json.RawMessage) (any, error)
}

// RPCHandlerFunc is a func-to-interface adapter.
type RPCHandlerFunc func(ctx context.Context, method string, params json.RawMessage) (any, error)

// Handle implements RPCHandler.
func (f RPCHandlerFunc) Handle(ctx context.Context, method string, params json.RawMessage) (any, error) {
	return f(ctx, method, params)
}

// nilHandler fails every inbound request with MethodNotFound. It's
// the default when a Mux is built with a nil RPCHandler, which the
// LSP proxy uses (sidecar → host calls are rare for language servers).
type nilHandler struct{}

func (nilHandler) Handle(_ context.Context, method string, _ json.RawMessage) (any, error) {
	return nil, &RPCError{Code: RPCErrMethodNotFound, Message: "method not found: " + method}
}

// ─────────────────────────────────────────────
// Mux
// ─────────────────────────────────────────────

// Mux is a JSON-RPC 2.0 multiplexer over a FramedReader/FramedWriter
// pair. It supports:
//
//   - Outbound requests (Call / Notify) — one goroutine-per-call;
//     responses are demuxed by ID and delivered to a per-call channel.
//   - Inbound requests — routed through RPCHandler.
//   - Inbound notifications — fanned out on Notifications().
//
// Lifecycle:
//
//	mux := NewMux(reader, writer, handler, log)
//	ctx, cancel := context.WithCancel(parent)
//	mux.Start(ctx)      // spawns the reader goroutine
//	defer cancel()      // cancelling the ctx stops the reader and
//	                    // rejects every in-flight Call with
//	                    // context.Canceled
//
// Concurrency: Call / Notify are safe for concurrent callers. The
// reader goroutine is single-threaded; handler invocations are
// spawned on a fresh goroutine per request so a slow handler doesn't
// block the stream.
type Mux struct {
	reader  *FramedReader
	writer  *FramedWriter
	handler RPCHandler
	log     *slog.Logger

	mu      sync.Mutex
	nextID  int64
	pending map[string]chan *RPC
	notifCh chan Notification

	startOnce sync.Once
	closeOnce sync.Once
	done      chan struct{}
}

// NewMux wraps a reader/writer pair. handler may be nil — defaults to
// a handler that fails every inbound request with MethodNotFound.
// log nil → slog.Default.
func NewMux(r *FramedReader, w *FramedWriter, handler RPCHandler, log *slog.Logger) *Mux {
	if handler == nil {
		handler = nilHandler{}
	}
	if log == nil {
		log = slog.Default()
	}
	return &Mux{
		reader:  r,
		writer:  w,
		handler: handler,
		log:     log,
		pending: make(map[string]chan *RPC),
		notifCh: make(chan Notification, 32),
		done:    make(chan struct{}),
	}
}

// Start spawns the reader goroutine. Cancelling ctx (or closing the
// underlying reader) ends the read loop and rejects every pending
// Call with ctx.Err() or io.EOF.
func (m *Mux) Start(ctx context.Context) {
	m.startOnce.Do(func() {
		go m.readLoop(ctx)
	})
}

// Call sends an outbound request and blocks until the response arrives
// or ctx is done. Returns the raw result bytes or a *RPCError wrapped
// in an error for server-side failures.
func (m *Mux) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	rawParams, err := marshalParams(params)
	if err != nil {
		return nil, err
	}
	id := m.nextIDString()
	ch := make(chan *RPC, 1)

	m.mu.Lock()
	m.pending[id] = ch
	m.mu.Unlock()

	// Clean up the pending entry unconditionally on exit so a late
	// response doesn't leak.
	defer func() {
		m.mu.Lock()
		delete(m.pending, id)
		m.mu.Unlock()
	}()

	req := RPC{
		JSONRPC: "2.0",
		ID:      json.RawMessage(strconv.Quote(id)),
		Method:  method,
		Params:  rawParams,
	}
	if err := m.writer.Write(req); err != nil {
		return nil, fmt.Errorf("mux: write request %s: %w", method, err)
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-m.done:
		return nil, io.ErrUnexpectedEOF
	case resp := <-ch:
		if resp.Error != nil {
			return nil, resp.Error
		}
		return resp.Result, nil
	}
}

// Notify sends a one-way message (no response expected).
func (m *Mux) Notify(method string, params any) error {
	rawParams, err := marshalParams(params)
	if err != nil {
		return err
	}
	return m.writer.Write(RPC{
		JSONRPC: "2.0",
		Method:  method,
		Params:  rawParams,
	})
}

// Notifications returns the read-only channel of inbound
// notifications. Callers must drain it — full buffer means oldest
// notifications are dropped with a warn log.
func (m *Mux) Notifications() <-chan Notification {
	return m.notifCh
}

// ─────────────────────────────────────────────
// readLoop
// ─────────────────────────────────────────────

func (m *Mux) readLoop(ctx context.Context) {
	defer m.closeAll(io.EOF)
	for {
		select {
		case <-ctx.Done():
			m.closeAll(ctx.Err())
			return
		default:
		}
		raw, err := m.reader.Read()
		if err != nil {
			if !errors.Is(err, io.EOF) {
				m.log.Warn("mux: read", "err", err)
			}
			m.closeAll(err)
			return
		}
		var msg RPC
		if uerr := json.Unmarshal(raw, &msg); uerr != nil {
			m.log.Warn("mux: malformed frame", "err", uerr)
			continue
		}
		m.dispatch(ctx, &msg)
	}
}

func (m *Mux) dispatch(ctx context.Context, msg *RPC) {
	switch {
	case msg.Method != "" && len(msg.ID) > 0:
		// Inbound request — route through the handler on a fresh
		// goroutine so a slow handler doesn't block the stream.
		go m.handleInbound(ctx, msg)
	case msg.Method != "" && len(msg.ID) == 0:
		// Inbound notification.
		m.pushNotification(Notification{Method: msg.Method, Params: msg.Params})
	case len(msg.ID) > 0:
		// Response to an outbound Call.
		id, perr := parseID(msg.ID)
		if perr != nil {
			m.log.Warn("mux: invalid response ID", "err", perr)
			return
		}
		m.mu.Lock()
		ch, ok := m.pending[id]
		m.mu.Unlock()
		if !ok {
			m.log.Warn("mux: response for unknown ID", "id", id)
			return
		}
		select {
		case ch <- msg:
		default:
			m.log.Warn("mux: response drop — caller gone", "id", id)
		}
	default:
		m.log.Warn("mux: unclassifiable frame", "raw", string(msg.Method))
	}
}

func (m *Mux) handleInbound(ctx context.Context, msg *RPC) {
	result, err := m.handler.Handle(ctx, msg.Method, msg.Params)
	resp := RPC{JSONRPC: "2.0", ID: msg.ID}
	if err != nil {
		var rerr *RPCError
		if errors.As(err, &rerr) {
			resp.Error = rerr
		} else {
			resp.Error = &RPCError{Code: RPCErrInternal, Message: err.Error()}
		}
	} else {
		raw, merr := json.Marshal(result)
		if merr != nil {
			resp.Error = &RPCError{Code: RPCErrInternal, Message: "marshal: " + merr.Error()}
		} else {
			resp.Result = raw
		}
	}
	if werr := m.writer.Write(resp); werr != nil {
		m.log.Warn("mux: write response", "id", string(msg.ID), "err", werr)
	}
}

func (m *Mux) pushNotification(n Notification) {
	select {
	case m.notifCh <- n:
	default:
		// Drop-oldest policy to keep the reader running under
		// backpressure.
		select {
		case dropped := <-m.notifCh:
			m.log.Warn("mux: notification buffer full; dropped oldest", "method", dropped.Method)
		default:
		}
		select {
		case m.notifCh <- n:
		default:
			m.log.Warn("mux: notification dropped", "method", n.Method)
		}
	}
}

// closeAll rejects every pending Call with err and closes the done
// channel. Safe for concurrent callers; only the first close wins.
func (m *Mux) closeAll(err error) {
	m.closeOnce.Do(func() {
		close(m.done)
		m.mu.Lock()
		entries := make([]chan *RPC, 0, len(m.pending))
		for id, ch := range m.pending {
			entries = append(entries, ch)
			delete(m.pending, id)
		}
		m.mu.Unlock()
		for _, ch := range entries {
			// Non-blocking send of a synthetic error response. The
			// Call side also watches m.done, so this is belt-and-
			// suspenders.
			select {
			case ch <- &RPC{Error: &RPCError{Code: RPCErrInternal, Message: err.Error()}}:
			default:
			}
		}
	})
}

// ─────────────────────────────────────────────
// helpers
// ─────────────────────────────────────────────

func (m *Mux) nextIDString() string {
	m.mu.Lock()
	m.nextID++
	id := m.nextID
	m.mu.Unlock()
	return strconv.FormatInt(id, 10)
}

func marshalParams(params any) (json.RawMessage, error) {
	if params == nil {
		return nil, nil
	}
	if raw, ok := params.(json.RawMessage); ok {
		return raw, nil
	}
	return json.Marshal(params)
}

// parseID accepts either a JSON string ("42") or a JSON number (42)
// and returns the canonical string form.
func parseID(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "", errors.New("empty id")
	}
	if raw[0] == '"' {
		s, err := strconv.Unquote(string(raw))
		return s, err
	}
	// JSON number — normalise to its decimal representation.
	var n json.Number
	if err := json.Unmarshal(raw, &n); err != nil {
		return "", err
	}
	return n.String(), nil
}

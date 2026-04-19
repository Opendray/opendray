package gateway

// T14 — SSE workbench stream.
//
// WorkbenchBus is the host → Flutter out-of-band channel. Go bridge
// namespaces (workbench.showMessage, workbench.openView, etc.) publish
// events here; every /api/workbench/stream subscriber receives a copy.
//
// Thread-safe: mu protects subs; Publish iterates a snapshot so it never
// holds the lock while writing to channels.

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/opendray/opendray/plugin/bridge"
)

const (
	// subBufferSize is the per-subscriber channel buffer. Events beyond this
	// capacity use drop-oldest semantics so the publisher is never blocked.
	subBufferSize = 32

	// defaultHeartbeatInterval is the production heartbeat cadence.
	defaultHeartbeatInterval = 20 * time.Second
)

// WorkbenchEvent is the wire shape sent to SSE subscribers.
//
// Kind is one of:
//
//	"showMessage" | "openView" | "updateStatusBar" | "contributionsChanged" | "theme"
//
// Plugin is the source plugin name (may be empty for host-originated events
// such as contributionsChanged).
type WorkbenchEvent struct {
	Kind    string          `json:"kind"`
	Plugin  string          `json:"plugin,omitempty"`
	Payload json.RawMessage `json:"payload"`
}

// subscription holds a single subscriber's channel and its unique id.
type subscription struct {
	id uint64
	ch chan WorkbenchEvent
}

// WorkbenchBus is the host → Flutter out-of-band channel.
//
//   - Thread-safe for concurrent Publish + Subscribe.
//   - Buffered channels per subscriber (size 32).
//   - Drop-oldest on overflow + log warn — never block the publisher thread.
//   - Implements bridge.ShowMessageSink, bridge.OpenViewSink, and bridge.StatusBarSink.
type WorkbenchBus struct {
	mu   sync.Mutex
	subs map[uint64]*subscription
	next uint64 // monotonically increasing subscription id
	log  *slog.Logger
}

// NewWorkbenchBus creates a new WorkbenchBus with the given logger.
func NewWorkbenchBus(log *slog.Logger) *WorkbenchBus {
	if log == nil {
		log = slog.Default()
	}
	return &WorkbenchBus{
		subs: make(map[uint64]*subscription),
		log:  log,
	}
}

// Publish broadcasts ev to all current subscribers. Never blocks.
// If a subscriber's buffer is full, the oldest event is dropped to make room.
func (b *WorkbenchBus) Publish(ev WorkbenchEvent) {
	b.mu.Lock()
	// Snapshot the current subscribers so we release the lock before writing.
	snapshot := make([]*subscription, 0, len(b.subs))
	for _, s := range b.subs {
		snapshot = append(snapshot, s)
	}
	b.mu.Unlock()

	for _, s := range snapshot {
		// Non-blocking send: if full, drop oldest to make room.
		select {
		case s.ch <- ev:
		default:
			// Channel full — drop oldest event, enqueue new one.
			select {
			case dropped := <-s.ch:
				b.log.Warn("workbench bus: subscriber channel full, dropping oldest event",
					"kind", dropped.Kind,
					"plugin", dropped.Plugin,
					"sub_id", s.id,
				)
			default:
			}
			// Best-effort enqueue after drop.
			select {
			case s.ch <- ev:
			default:
				b.log.Warn("workbench bus: failed to deliver event after drop, skipping",
					"kind", ev.Kind,
					"sub_id", s.id,
				)
			}
		}
	}
}

// Subscribe returns a channel receiving future events and a done func the
// caller must invoke (typically via defer) to free resources.
// done is idempotent — safe to call multiple times.
func (b *WorkbenchBus) Subscribe() (<-chan WorkbenchEvent, func()) {
	b.mu.Lock()
	id := b.next
	b.next++
	s := &subscription{
		id: id,
		ch: make(chan WorkbenchEvent, subBufferSize),
	}
	b.subs[id] = s
	b.mu.Unlock()

	var once sync.Once
	done := func() {
		once.Do(func() {
			b.mu.Lock()
			delete(b.subs, id)
			b.mu.Unlock()
			// Close the channel so any range/select on it unblocks.
			close(s.ch)
		})
	}
	return s.ch, done
}

// SubscriberCount returns the current number of active subscribers.
// Exposed for test visibility.
func (b *WorkbenchBus) SubscriberCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.subs)
}

// ─── ShowMessageSink / OpenViewSink / StatusBarSink ───────────────────────────

// ShowMessage implements bridge.ShowMessageSink.
// userID is ignored in M2 (single-user assumption).
func (b *WorkbenchBus) ShowMessage(userID, plugin string, opts bridge.ShowMessageOpts) error {
	payload, err := json.Marshal(opts)
	if err != nil {
		return fmt.Errorf("workbench bus ShowMessage: marshal payload: %w", err)
	}
	b.Publish(WorkbenchEvent{
		Kind:    "showMessage",
		Plugin:  plugin,
		Payload: json.RawMessage(payload),
	})
	return nil
}

// OpenView implements bridge.OpenViewSink.
// userID is ignored in M2 (single-user assumption).
func (b *WorkbenchBus) OpenView(userID, plugin, viewID string) error {
	payload, err := json.Marshal(viewID)
	if err != nil {
		return fmt.Errorf("workbench bus OpenView: marshal payload: %w", err)
	}
	b.Publish(WorkbenchEvent{
		Kind:    "openView",
		Plugin:  plugin,
		Payload: json.RawMessage(payload),
	})
	return nil
}

// UpdateStatusBar implements bridge.StatusBarSink.
// userID is ignored in M2 (single-user assumption).
func (b *WorkbenchBus) UpdateStatusBar(userID, plugin string, items []bridge.StatusBarOverride) error {
	payload, err := json.Marshal(items)
	if err != nil {
		return fmt.Errorf("workbench bus UpdateStatusBar: marshal payload: %w", err)
	}
	b.Publish(WorkbenchEvent{
		Kind:    "updateStatusBar",
		Plugin:  plugin,
		Payload: json.RawMessage(payload),
	})
	return nil
}

// PublishContributionsChanged is a convenience helper for the contribution
// registry hot-reload (T15). Publishes a contributionsChanged event with
// an empty payload.
func (b *WorkbenchBus) PublishContributionsChanged() {
	b.Publish(WorkbenchEvent{
		Kind:    "contributionsChanged",
		Payload: json.RawMessage(`{}`),
	})
}

// ─── HTTP handler ─────────────────────────────────────────────────────────────

// workbenchStream handles GET /api/workbench/stream.
//
// Streams events via SSE per
// https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events.
//
// Wire format:
//
//	data: {"kind":"showMessage","plugin":"kanban","payload":{...}}\n\n
//	:\n\n   — heartbeat (keeps intermediaries from idle-closing)
//
// Required headers:
//
//	Content-Type: text/event-stream
//	Cache-Control: no-cache
//	Connection: keep-alive
//	X-Accel-Buffering: no
//
// Lifecycle:
//  1. Subscribe to workbenchBus.
//  2. Loop: select on bus events, heartbeat ticker, r.Context().Done().
//  3. On each event: marshal JSON, write "data: <json>\n\n", Flush.
//  4. On heartbeat: write ":\n\n", Flush.
//  5. On context cancel (client disconnect): call bus unsubscribe, return.
//
// 503 EBUS if workbenchBus is nil.
// 401 handled by the protected-route middleware.
func (s *Server) workbenchStream(w http.ResponseWriter, r *http.Request) {
	if s.workbenchBus == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "EBUS",
			"workbench bus not wired")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSONError(w, http.StatusInternalServerError, "EFLUSH",
			"streaming not supported by this transport")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	evCh, unsubscribe := s.workbenchBus.Subscribe()
	defer unsubscribe()

	hbInterval := s.heartbeatInterval
	if hbInterval <= 0 {
		hbInterval = defaultHeartbeatInterval
	}
	ticker := time.NewTicker(hbInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return

		case ev, ok := <-evCh:
			if !ok {
				// Channel was closed (bus shut down or unsubscribed from another goroutine).
				return
			}
			data, err := json.Marshal(ev)
			if err != nil {
				// Marshal should never fail for our controlled struct — log and skip.
				if s.logger != nil {
					s.logger.Error("workbench stream: marshal event", "err", err)
				}
				continue
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				return
			}
			flusher.Flush()

		case <-ticker.C:
			// SSE comment — treated as heartbeat by the spec.
			if _, err := fmt.Fprint(w, ":\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

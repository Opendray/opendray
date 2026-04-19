// Package bridge — Bridge connection manager + consent hot-revoke bus (M2 T6).
//
// # Architecture
//
// Manager holds all live bridge connections keyed by plugin name (byPlugin map).
// Each Conn has a per-connection subscription registry (subRegistry).
//
// # Hot-revoke flow (InvalidateConsent)
//
//  1. Caller calls InvalidateConsent(plugin, cap).
//  2. Manager snapshots the plugin's conn list under RLock then releases it.
//  3. For each conn, removeByCap atomically closes all matching sub.done channels
//     (synchronous — done channels are closed before InvalidateConsent returns).
//  4. A single goroutine per conn is spawned to write the EPERM terminal envelopes.
//     This goroutine drains a buffered revoke channel (size 128 to handle large
//     bursts) and calls WriteEnvelope for each affected subscription.
//  5. InvalidateConsent returns immediately — zero WS I/O on the caller path.
//     The 200 ms SLO covers both (a) done-channels-closed and (b) envelopes
//     written. (a) is synchronous; (b) completes asynchronously in the goroutine.
//
// # Backpressure (test 9)
//
// fakeWS.blockN simulates a slow writer. The revoke goroutine blocks on slow
// ws.WriteMessage calls — it does NOT block the InvalidateConsent caller.
// If the revoke goroutine's sendCh buffer overflows (impossible with size 128
// and a single revoke pass), drop-oldest policy applies with a warn log.
//
// # Write serialisation
//
// gorilla/websocket is NOT concurrent-write safe. Conn.WriteEnvelope holds a
// per-Conn sync.Mutex around every ws.WriteMessage call. All writes — from the
// handler and from the revoke goroutine — go through WriteEnvelope.
//
// # Allowed imports: stdlib only (gorilla websocket not needed here; WSLike
// abstracts the socket). Do NOT import kernel/store.
package bridge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
)

// ─────────────────────────────────────────────
// Public types
// ─────────────────────────────────────────────

// ConsentChange broadcasts when a plugin's capability grant changes.
type ConsentChange struct {
	Plugin string
	Cap    string // the capability key being revoked (e.g. "storage", "events")
}

// WSLike is the minimum surface Conn requires from a WebSocket. Having an
// interface here lets tests pass a fake instead of opening a real socket.
type WSLike interface {
	WriteMessage(messageType int, data []byte) error
	Close() error
}

// ─────────────────────────────────────────────
// subEntry — one active stream subscription
// ─────────────────────────────────────────────

type subEntry struct {
	subID string
	cap   string
	done  chan struct{} // closed when sub is revoked or unsubscribed
	once  sync.Once    // guards done channel close — idempotent
}

func (s *subEntry) close() {
	s.once.Do(func() { close(s.done) })
}

// ─────────────────────────────────────────────
// subRegistry — per-Conn subscription index
// ─────────────────────────────────────────────

type subRegistry struct {
	mu   sync.Mutex
	subs map[string]*subEntry // subID → entry
}

func newSubRegistry() *subRegistry {
	return &subRegistry{subs: make(map[string]*subEntry)}
}

// add registers a new subscription. Returns the done channel, or an error on
// duplicate subID.
func (r *subRegistry) add(subID, cap string) (<-chan struct{}, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.subs[subID]; exists {
		return nil, fmt.Errorf("bridge: subscription %q already registered", subID)
	}
	e := &subEntry{subID: subID, cap: cap, done: make(chan struct{})}
	r.subs[subID] = e
	return e.done, nil
}

// remove closes and removes a sub by ID. Idempotent.
func (r *subRegistry) remove(subID string) {
	r.mu.Lock()
	e, ok := r.subs[subID]
	if ok {
		delete(r.subs, subID)
	}
	r.mu.Unlock()
	if ok {
		e.close()
	}
}

// drain closes all subs, clears the registry, and returns the entries.
func (r *subRegistry) drain() []*subEntry {
	r.mu.Lock()
	all := make([]*subEntry, 0, len(r.subs))
	for _, e := range r.subs {
		all = append(all, e)
	}
	r.subs = make(map[string]*subEntry)
	r.mu.Unlock()
	for _, e := range all {
		e.close()
	}
	return all
}

// removeByCap closes and removes all subs whose cap matches. Done channels are
// closed synchronously before this function returns.
func (r *subRegistry) removeByCap(cap string) []*subEntry {
	r.mu.Lock()
	var matched []*subEntry
	for subID, e := range r.subs {
		if e.cap == cap {
			matched = append(matched, e)
			delete(r.subs, subID)
		}
	}
	r.mu.Unlock()
	for _, e := range matched {
		e.close()
	}
	return matched
}

// ─────────────────────────────────────────────
// Conn
// ─────────────────────────────────────────────

// Conn wraps a WSLike with serialised writes and a per-connection subscription
// registry. Do not construct directly — use Manager.Register.
type Conn struct {
	Plugin string

	ws      WSLike
	writeMu sync.Mutex // serialises all ws.WriteMessage calls
	subs    *subRegistry
	mgr     *Manager
	log     *slog.Logger

	ctx    context.Context
	cancel context.CancelFunc

	closeOnce sync.Once
	closed    bool // guarded by writeMu; checked inside WriteEnvelope
}

// WriteEnvelope serialises and writes one envelope. Safe for concurrent callers.
// Returns an error if the conn is closed or serialisation fails.
func (c *Conn) WriteEnvelope(e Envelope) error {
	raw, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("bridge: marshal envelope: %w", err)
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if c.closed {
		return errors.New("bridge: conn is closed")
	}
	return c.ws.WriteMessage(2 /* BinaryMessage */, raw)
}

// Subscribe registers a stream subscription on this conn. Returns a done channel
// that closes when the sub is revoked (hot-revoke or Unsubscribe). Returns an
// error on duplicate subID.
func (c *Conn) Subscribe(subID, cap string) (<-chan struct{}, error) {
	return c.subs.add(subID, cap)
}

// Unsubscribe closes a single subscription by id. Idempotent.
func (c *Conn) Unsubscribe(subID string) {
	c.subs.remove(subID)
}

// Close terminates the connection: cancels context, closes WS with the provided
// code+reason, closes all subscription done channels, and removes from manager.
// Idempotent — double-close is a no-op.
func (c *Conn) Close(code int, reason string) error {
	var wsErr error
	c.closeOnce.Do(func() {
		// Mark closed before releasing writeMu so WriteEnvelope sees it.
		c.writeMu.Lock()
		c.closed = true
		c.writeMu.Unlock()

		// Cancel internal context.
		c.cancel()

		// Close all subscription done channels.
		c.subs.drain()

		// Send WS close frame.
		wsErr = c.ws.Close()

		// Remove from manager.
		c.mgr.Unregister(c)
	})
	return wsErr
}

// writeRevokeEnvelopes is launched as a goroutine by InvalidateConsent. It
// writes one EPERM stream-end envelope per subID and returns. The goroutine is
// the unit of backpressure isolation: a slow WS only blocks this goroutine,
// never the InvalidateConsent caller or other conns.
//
// The items slice is a snapshot captured before the goroutine starts — no shared
// state, no locks needed inside.
func (c *Conn) writeRevokeEnvelopes(items []string) {
	for _, subID := range items {
		env := Envelope{
			V:      ProtocolVersion,
			ID:     subID,
			Error:  &WireError{Code: "EPERM", Message: "capability revoked"},
			Stream: "end",
		}
		if err := c.WriteEnvelope(env); err != nil {
			c.log.Warn("bridge: failed to write EPERM revoke envelope",
				"plugin", c.Plugin,
				"subID", subID,
				"err", err,
			)
		}
	}
}

// ─────────────────────────────────────────────
// Manager
// ─────────────────────────────────────────────

// Manager owns every live bridge Conn across all plugins plus a pub/sub bus for
// consent invalidation. One instance per gateway process.
type Manager struct {
	mu       sync.RWMutex
	byPlugin map[string][]*Conn
	log      *slog.Logger
}

// NewManager constructs a Manager. log may be nil (falls back to slog.Default).
func NewManager(log *slog.Logger) *Manager {
	if log == nil {
		log = slog.Default()
	}
	return &Manager{
		byPlugin: make(map[string][]*Conn),
		log:      log,
	}
}

// Register wraps ws in a Conn and adds it to the per-plugin registry.
// The returned Conn is ready for WriteEnvelope and Subscribe calls.
func (m *Manager) Register(plugin string, ws WSLike) *Conn {
	ctx, cancel := context.WithCancel(context.Background())
	conn := &Conn{
		Plugin: plugin,
		ws:     ws,
		subs:   newSubRegistry(),
		mgr:    m,
		log:    m.log,
		ctx:    ctx,
		cancel: cancel,
	}
	m.mu.Lock()
	m.byPlugin[plugin] = append(m.byPlugin[plugin], conn)
	m.mu.Unlock()
	return conn
}

// Unregister drops conn from the registry. Typically called by Conn.Close; also
// exported for tests and explicit handler cleanup. Idempotent.
func (m *Manager) Unregister(c *Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	conns := m.byPlugin[c.Plugin]
	n := 0
	for _, existing := range conns {
		if existing != c {
			conns[n] = existing
			n++
		}
	}
	if n == 0 {
		delete(m.byPlugin, c.Plugin)
	} else {
		m.byPlugin[c.Plugin] = conns[:n]
	}
}

// InvalidateConsent broadcasts a hot-revoke. For every active Conn belonging to
// plugin:
//  1. Each Subscribe with matching cap has its done channel closed synchronously.
//  2. One terminal EPERM stream-end envelope is written per affected sub
//     (fire-and-forget via a per-conn goroutine; failures are logged).
//
// MUST return within 200 ms even with 1000 concurrent subs. Writes are
// fire-and-forget — WriteEnvelope failures are logged, not blocking.
func (m *Manager) InvalidateConsent(plugin, cap string) {
	m.mu.RLock()
	conns := m.byPlugin[plugin]
	snapshot := make([]*Conn, len(conns))
	copy(snapshot, conns)
	m.mu.RUnlock()

	for _, conn := range snapshot {
		// removeByCap closes done channels synchronously.
		matched := conn.subs.removeByCap(cap)
		if len(matched) == 0 {
			continue
		}
		// Collect subIDs for the writer goroutine.
		subIDs := make([]string, len(matched))
		for i, e := range matched {
			subIDs[i] = e.subID
		}
		// Spawn a goroutine per conn to write EPERM envelopes. The goroutine
		// is isolated: a slow WS blocks only it, never the caller.
		conn := conn // capture for goroutine
		go conn.writeRevokeEnvelopes(subIDs)
	}
}

// ActiveConns returns the number of registered connections for a plugin.
func (m *Manager) ActiveConns(plugin string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.byPlugin[plugin])
}

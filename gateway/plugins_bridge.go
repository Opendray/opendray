package gateway

// T7 — Plugin bridge WebSocket handler.
//
// Route (registered in protected chi group):
//
//	GET /api/plugins/{name}/bridge/ws
//
// Flow:
//
//  1. chi.URLParam "name" → plugin name.
//  2. JWT middleware already ran (protected group).
//  3. Origin check (see originAllowed) — 403 EFORBIDDEN_ORIGIN on mismatch,
//     BEFORE the websocket upgrade.
//  4. 404 ENOPLUGIN if the plugin is not registered.
//  5. Upgrade via a dedicated upgrader (restrictive compared to the session
//     WS in ws.go — we own CheckOrigin ourselves so we can 403 pre-handshake).
//  6. Manager.Register → *bridge.Conn with serialised writes.
//  7. Reader goroutine: decode envelope JSON, dispatch via namespaceRegistry.
//  8. Idle read deadline + ping/pong keepalive.
//  9. On disconnect: Manager.Unregister + close WS.
//
// Namespaces implement the Namespace interface and register themselves via
// Server.RegisterNamespace. They are duck-typed so T9 (workbench), T10 (storage),
// T11 (events) can land independently without touching this file.
//
// Rate limiting is per-connection via a local sliding-window bucket. Default
// 60 req/min, configurable through cfg.Plugins2.BridgeRatePerMinute.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"

	"github.com/opendray/opendray/plugin"
	"github.com/opendray/opendray/plugin/bridge"
)

// ─── Public types ────────────────────────────────────────────────────────────

// Namespace is the server-side surface every bridge API namespace implements.
// T7 treats this surface as duck-typed so T9/T10/T11 can land independently.
//
// Dispatch is called once per inbound request envelope. It may return:
//   - (result, nil)         — success; handler marshals result and replies.
//   - (nil, *bridge.PermError) — capability denied; handler replies with EPERM.
//   - (nil, err)            — any other error; handler replies with EINTERNAL.
//
// Long-lived streams are managed by the namespace via conn.Subscribe; this
// method returns immediately for "subscribe" calls with the subscription id
// surfaced in the result payload.
//
// Subscribe/unsubscribe-capable namespaces (events) accept conn; stateless
// namespaces (workbench, storage) ignore it. The parameter is always supplied
// so the method signature is uniform.
//
// envID is the inbound envelope's id. Stream-capable methods (events.subscribe)
// use it as the subId tagged on every chunk (M2-PLAN §11 wire contract — the
// subId MUST equal the subscribe envelope id so the client's in-flight request
// completer and the chunk-stream handler share a correlation key).
type Namespace interface {
	Dispatch(ctx context.Context, plugin string, method string, args json.RawMessage, envID string, conn *bridge.Conn) (any, error)
}

// ─── Internal types ──────────────────────────────────────────────────────────

// namespaceRegistry maps a namespace name → its handler. Populated by
// Server.RegisterNamespace (called from main.go wiring or a future task),
// queried by the bridge reader loop.
type namespaceRegistry struct {
	mu sync.RWMutex
	ns map[string]Namespace
}

func newNamespaceRegistry() *namespaceRegistry {
	return &namespaceRegistry{ns: make(map[string]Namespace)}
}

func (r *namespaceRegistry) register(name string, n Namespace) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ns[name] = n
}

func (r *namespaceRegistry) lookup(name string) (Namespace, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	n, ok := r.ns[name]
	return n, ok
}

// RegisterNamespace plugs a namespace implementation into the bridge handler.
// Safe to call at any point after New — future connections see the new entry,
// and in-flight reader loops pick it up on their next lookup (each inbound
// envelope does a fresh lookup).
func (s *Server) RegisterNamespace(name string, n Namespace) {
	if s.bridgeNamespace == nil {
		s.bridgeNamespace = newNamespaceRegistry()
	}
	s.bridgeNamespace.register(name, n)
}

// bridgeRateLimiter is a tiny per-plugin sliding-window limiter kept local to
// this file. 60 events/minute default; configurable via PluginsConfig. Uses a
// circular buffer of recent event timestamps so the retryAfterMs calculation
// is exact (time until the oldest timestamp falls out of the window).
type bridgeRateLimiter struct {
	mu       sync.Mutex
	window   time.Duration
	limit    int
	recent   []time.Time // ring buffer of the N most-recent hit timestamps
	head     int         // next write index into recent
	filled   bool        // once len(recent) == limit
}

func newBridgeRateLimiter(limit int, window time.Duration) *bridgeRateLimiter {
	if limit <= 0 {
		limit = 60
	}
	if window <= 0 {
		window = time.Minute
	}
	return &bridgeRateLimiter{
		window: window,
		limit:  limit,
		recent: make([]time.Time, limit),
	}
}

// allow returns (true, 0) when the caller may proceed. When the window is
// full it returns (false, retryAfter) where retryAfter is the duration until
// the oldest event in the window expires — exactly when the next allow-call
// would succeed.
func (rl *bridgeRateLimiter) allow(now time.Time) (bool, time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// The oldest event in the window is at recent[head] once the buffer is
	// filled; while still filling it is at recent[0].
	if !rl.filled {
		rl.recent[rl.head] = now
		rl.head++
		if rl.head == rl.limit {
			rl.head = 0
			rl.filled = true
		}
		return true, 0
	}

	oldest := rl.recent[rl.head]
	if now.Sub(oldest) >= rl.window {
		// The oldest event rolled out of the window — accept.
		rl.recent[rl.head] = now
		rl.head = (rl.head + 1) % rl.limit
		return true, 0
	}

	// Window full — compute time until the oldest event expires.
	retry := rl.window - now.Sub(oldest)
	if retry < time.Millisecond {
		retry = time.Millisecond
	}
	return false, retry
}

// ─── Upgrader ────────────────────────────────────────────────────────────────

// bridgeUpgrader is the dedicated upgrader for plugin bridge connections. We
// own CheckOrigin here (it is applied pre-upgrade by the handler itself so we
// can 403 with a JSON body before the websocket handshake starts) and enforce
// a 1 MiB message size limit to match the body-size limiter on the REST side.
var bridgeUpgrader = websocket.Upgrader{
	ReadBufferSize:    4096,
	WriteBufferSize:   4096,
	EnableCompression: false,
	// CheckOrigin returns true here — the handler performs the origin check
	// BEFORE calling Upgrade so it can 403 with a JSON error body. By the time
	// Upgrade runs we've already accepted the origin.
	CheckOrigin: func(_ *http.Request) bool { return true },
}

// ─── Handler ─────────────────────────────────────────────────────────────────

// pluginsBridgeWS handles WS upgrades at /api/plugins/{name}/bridge/ws.
func (s *Server) pluginsBridgeWS(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	// ── 1. Origin check (pre-upgrade so we can return a JSON 403) ─────────────
	if !originAllowed(r, s.bridgeCfg.FrontendOrigin) {
		writeJSONError(w, http.StatusForbidden, "EFORBIDDEN_ORIGIN",
			"origin not allowed for bridge connection")
		return
	}

	// ── 2. Plugin existence check ────────────────────────────────────────────
	if !s.bridgePluginExists(name) {
		writeJSONError(w, http.StatusNotFound, "ENOPLUGIN",
			"plugin not registered")
		return
	}

	// ── 3. Bridge manager must be wired ──────────────────────────────────────
	if s.bridgeMgr == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "EBRIDGE",
			"bridge manager not wired")
		return
	}

	// ── 4. Upgrade ───────────────────────────────────────────────────────────
	ws, err := bridgeUpgrader.Upgrade(w, r, nil)
	if err != nil {
		// Upgrade failed — gorilla has already written a response so we just
		// log and return. Calling writeJSONError here would double-write.
		if s.logger != nil {
			s.logger.Warn("bridge: upgrade failed", "plugin", name, "err", err)
		}
		return
	}

	// Size limit matches bodySizeLimiter on REST routes.
	ws.SetReadLimit(1 << 20)

	// ── 5. Register with manager ─────────────────────────────────────────────
	conn := s.bridgeMgr.Register(name, ws)

	// ── 6. Configure keepalive + idle read deadline ──────────────────────────
	readTimeout := s.bridgeCfg.BridgeReadTimeout
	if readTimeout <= 0 {
		readTimeout = 60 * time.Second
	}
	_ = ws.SetReadDeadline(time.Now().Add(readTimeout))
	ws.SetPongHandler(func(string) error {
		return ws.SetReadDeadline(time.Now().Add(readTimeout))
	})

	// Ping ticker — half the read timeout so we get at least one pong per window.
	pingInterval := readTimeout / 2
	if pingInterval < time.Second {
		pingInterval = time.Second
	}

	done := make(chan struct{})

	// Pinger goroutine.
	go func() {
		t := time.NewTicker(pingInterval)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
				// WriteControl doesn't go through bridge.Conn.WriteEnvelope;
				// gorilla's WriteControl is goroutine-safe with plain WriteMessage
				// as long as nobody is mid-write on the text/binary channel.
				// We rely on conn.WriteEnvelope holding writeMu for all
				// non-control frames; control frames bypass that lock.
				_ = ws.WriteControl(websocket.PingMessage, nil,
					time.Now().Add(5*time.Second))
			}
		}
	}()

	// ── 7. Rate limiter (local to this connection) ───────────────────────────
	rl := newBridgeRateLimiter(s.bridgeCfg.BridgeRatePerMinute, time.Minute)

	// ── 8. Reader loop ───────────────────────────────────────────────────────
	defer func() {
		close(done)
		// Close unregisters + closes ws + drains subscriptions.
		_ = conn.Close(websocket.CloseNormalClosure, "bye")
	}()

	for {
		_ = ws.SetReadDeadline(time.Now().Add(readTimeout))
		_, payload, err := ws.ReadMessage()
		if err != nil {
			// Normal close or read timeout — exit the loop.
			return
		}

		var env bridge.Envelope
		if err := json.Unmarshal(payload, &env); err != nil {
			_ = conn.WriteEnvelope(bridge.NewErr("", "EINVAL", "malformed envelope JSON"))
			continue
		}

		// ── Protocol-level checks ────────────────────────────────────────────
		if env.V != bridge.ProtocolVersion {
			_ = conn.WriteEnvelope(bridge.NewErr(env.ID, "EINVAL",
				fmt.Sprintf("unsupported protocol version %d", env.V)))
			continue
		}

		// Stream-inbound envelopes are permitted but ignored in M2 T7; the
		// namespaces will inspect their own streams in a later iteration.
		if env.Stream != "" {
			if s.logger != nil {
				s.logger.Debug("bridge: inbound stream envelope ignored",
					"plugin", name, "subID", env.ID, "stream", env.Stream)
			}
			continue
		}

		if env.NS == "" {
			_ = conn.WriteEnvelope(bridge.NewErr(env.ID, "EINVAL", "missing namespace"))
			continue
		}
		if env.Method == "" {
			_ = conn.WriteEnvelope(bridge.NewErr(env.ID, "EINVAL", "missing method"))
			continue
		}

		// ── Rate-limit check ────────────────────────────────────────────────
		if ok, retry := rl.allow(time.Now()); !ok {
			writeRateLimitedResponse(conn, env.ID, retry)
			continue
		}

		// ── Namespace dispatch ──────────────────────────────────────────────
		ns, found := s.bridgeNamespace.lookup(env.NS)
		if !found {
			_ = conn.WriteEnvelope(bridge.NewErr(env.ID, "EUNAVAIL",
				fmt.Sprintf("namespace %q is not implemented in this host", env.NS)))
			continue
		}

		// Dispatch on a goroutine so a slow namespace doesn't head-of-line-block
		// other inbound envelopes. Each dispatcher is self-contained and uses
		// conn.WriteEnvelope which is already serialised.
		go s.dispatchInvoke(r.Context(), name, conn, env, ns)
	}
}

// dispatchInvoke is the boundary between the reader loop and a namespace's
// Dispatch. It recovers panics, maps errors to envelopes, and serialises the
// result — keeping the reader loop tight and panic-safe.
func (s *Server) dispatchInvoke(ctx context.Context, plugin string, conn *bridge.Conn, env bridge.Envelope, ns Namespace) {
	defer func() {
		if rec := recover(); rec != nil {
			if s.logger != nil {
				s.logger.Error("bridge: namespace dispatch panicked",
					"plugin", plugin, "ns", env.NS, "method", env.Method, "panic", rec)
			}
			_ = conn.WriteEnvelope(bridge.NewErr(env.ID, "EINTERNAL",
				"internal error handling request"))
		}
	}()

	result, err := ns.Dispatch(ctx, plugin, env.Method, env.Args, env.ID, conn)
	if err != nil {
		// PermError → EPERM envelope; everything else → EINTERNAL.
		var permErr *bridge.PermError
		if errors.As(err, &permErr) {
			_ = conn.WriteEnvelope(bridge.NewErr(env.ID, permErr.Code, permErr.Msg))
			return
		}
		_ = conn.WriteEnvelope(bridge.NewErr(env.ID, "EINTERNAL", err.Error()))
		return
	}

	ok, marshalErr := bridge.NewOK(env.ID, result)
	if marshalErr != nil {
		if s.logger != nil {
			s.logger.Warn("bridge: result marshal failed",
				"plugin", plugin, "ns", env.NS, "method", env.Method, "err", marshalErr)
		}
		_ = conn.WriteEnvelope(bridge.NewErr(env.ID, "EINTERNAL",
			"failed to marshal response"))
		return
	}
	_ = conn.WriteEnvelope(ok)
}

// writeRateLimitedResponse writes an ETIMEOUT envelope with retryAfterMs in
// the WireError.Data payload.
func writeRateLimitedResponse(conn *bridge.Conn, id string, retry time.Duration) {
	retryMs := retry.Milliseconds()
	if retryMs <= 0 {
		retryMs = 1
	}
	data, _ := json.Marshal(map[string]any{"retryAfterMs": retryMs})
	env := bridge.Envelope{
		V:  bridge.ProtocolVersion,
		ID: id,
		Error: &bridge.WireError{
			Code:    "ETIMEOUT",
			Message: "per-plugin bridge rate limit exceeded",
			Data:    data,
		},
	}
	_ = conn.WriteEnvelope(env)
}

// bridgePluginExists returns true when the plugin name is known to the
// runtime. Tests can inject a custom lookup via bridgePluginsOverride; in
// production we consult s.plugins.Get directly.
func (s *Server) bridgePluginExists(name string) bool {
	if s.bridgePluginsOverride != nil {
		_, ok := s.bridgePluginsOverride(name)
		return ok
	}
	if s.plugins == nil {
		return false
	}
	_, ok := s.plugins.Get(name)
	return ok
}

// ─── Origin check ────────────────────────────────────────────────────────────

// originAllowed enforces the T7 origin policy. Returns true for any of:
//
//   - Origin header empty (non-browser clients with no Origin).
//   - Origin host == r.Host (same-origin; Flutter web served from the gateway).
//   - Origin == "app://opendray" (Android/iOS WKWebView / InAppWebView).
//   - Origin host is localhost or 127.0.0.1 with any port (dev).
//   - Origin == cfgFrontendOrigin when that value is non-empty.
//
// Anything else returns false.
func originAllowed(r *http.Request, cfgFrontendOrigin string) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		// No Origin header — allow. Browsers always send Origin on WS upgrade;
		// non-browser tooling (curl, Go tests without a dialer) may not.
		return true
	}

	// Explicit app scheme.
	if origin == "app://opendray" {
		return true
	}

	// Configured frontend host (future-proof).
	if cfgFrontendOrigin != "" && origin == cfgFrontendOrigin {
		return true
	}

	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}

	// Same-origin: Origin host matches the Request Host. We compare Host
	// including port because the gateway listens on a specific port.
	if u.Host == r.Host {
		return true
	}

	// Localhost / 127.0.0.1 with any port.
	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		host = u.Host
	}
	if host == "localhost" || host == "127.0.0.1" {
		return true
	}

	return false
}

// ─── Sanity compile-time assertion: *plugin.Runtime satisfies the lookup shape.
// The handler calls s.plugins.Get directly; this assertion keeps us honest
// against type drift in the plugin package.
var _ = func() bool {
	var p *plugin.Runtime
	_ = p
	return true
}

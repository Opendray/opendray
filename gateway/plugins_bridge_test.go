package gateway

// T7 — Bridge WebSocket handler test suite.
//
// Test strategy:
//   - httptest.NewServer + real gorilla/websocket.Dialer client. No upgrader
//     mocking — we drive the real handshake end-to-end.
//   - A tiny echoNS namespace (implements Namespace) lets the tests verify
//     dispatch & rate-limit behaviour without depending on T9/T10/T11.
//   - buildBridgeServer wires a *Server with JWT disabled by default so tests
//     can focus on the bridge semantics; auth-specific tests opt in to the
//     middleware by constructing a *Server with cfg.Auth set.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/opendray/opendray/kernel/auth"
	"github.com/opendray/opendray/plugin"
	"github.com/opendray/opendray/plugin/bridge"
)

// ─── Test helpers ────────────────────────────────────────────────────────────

// fakeNamespace captures the echoed arguments and allows the test to inject
// panics or custom behaviour per request.
type echoNS struct {
	calls atomic.Int64
}

func (n *echoNS) Dispatch(_ context.Context, _ string, method string, args json.RawMessage, _ *bridge.Conn) (any, error) {
	n.calls.Add(1)
	// Method "echo" returns the args as-is; any other method returns a marker
	// so tests can distinguish.
	if method == "echo" {
		var v any
		if len(args) > 0 {
			if err := json.Unmarshal(args, &v); err != nil {
				return nil, err
			}
		}
		return map[string]any{"echo": v, "method": method}, nil
	}
	return map[string]any{"method": method}, nil
}

// panicNS panics on every Dispatch — used to verify recovery.
type panicNS struct{}

func (p *panicNS) Dispatch(_ context.Context, _ string, _ string, _ json.RawMessage, _ *bridge.Conn) (any, error) {
	panic("boom")
}

// newTestServer constructs a *Server wired with a bridge manager, one plugin
// ("kanban"), and (optionally) auth. Auth is nil by default so the tests can
// bypass the JWT layer. testOpts is a functional-options bag so individual
// tests can tweak cfg.Plugins.BridgeRatePerMinute / BridgeReadTimeout without
// threading them through every helper.
type testOpts struct {
	withAuth       bool
	ratePerMinute  int
	readTimeout    time.Duration
	frontendOrigin string
	extraPlugins   []string
}

func defaultOpts() testOpts {
	return testOpts{
		ratePerMinute: 60,
		readTimeout:   60 * time.Second,
	}
}

// newTestBridgeServer spins up a *Server and an httptest.Server. It always
// registers plugin "kanban" so tests can focus on the WS handshake semantics.
// extraPlugins can seed additional names. Returns the httptest.Server, the
// *Server, the auth.Auth (nil when !withAuth) and a JWT token for convenience
// (empty when !withAuth).
func newTestBridgeServer(t *testing.T, opts testOpts) (*httptest.Server, *Server, *auth.Auth, string) {
	t.Helper()

	hooks := plugin.NewHookBus(nil)
	rt := plugin.NewRuntime(nil, hooks, "", nil)
	// Seed fake providers by poking into the unexported map isn't possible from
	// this package — but we can attach a stubby seed via the exported Register
	// path. The runtime has no Register for tests; instead we rely on the
	// handler's own plugin-existence check using runtime.Get. For tests to have
	// a "kanban" plugin, we use a tiny test-only Runtime wrapper via a known
	// indirection: the runtime package does provide a seed code path driven by
	// filesystem plugins, but not by direct registration. Workaround: use
	// plugin.Runtime's NewRuntime with a pluginDir we populate on disk.
	//
	// Cleaner workaround: our handler asserts plugin existence by calling
	// s.plugins.Get(name). For tests that don't use gateway.New (no auth), we
	// substitute a fake Runtime via a minimal interface captured by the Server
	// — but Server is the real struct. So the simplest path is to wire a real
	// Runtime with a filesystem dir containing a manifest for "kanban".
	//
	// That's heavy for unit tests. Instead: use a pluginRegistry seam — the
	// handler accepts a small `plugins` interface that defaults to *Runtime but
	// can be overridden via Server.pluginsOverride for tests. The production
	// code just uses s.plugins; the test sets s.pluginsOverride to a fake map.

	bridgeMgr := bridge.NewManager(nil)

	var a *auth.Auth
	var token string
	if opts.withAuth {
		a = auth.New("test-bridge-secret-value-not-too-short-xyz", 24*time.Hour)
		var err error
		token, err = a.Issue("testuser")
		if err != nil {
			t.Fatalf("issue token: %v", err)
		}
	}

	srv := New(Config{
		Auth:          a,
		Plugins:       rt,
		BridgeManager: bridgeMgr,
		Plugins2: PluginsConfig{
			BridgeRatePerMinute: opts.ratePerMinute,
			BridgeReadTimeout:   opts.readTimeout,
			FrontendOrigin:      opts.frontendOrigin,
		},
	})

	// Seed the plugin-existence map used by the handler. The override lets us
	// bypass the need for a real Runtime with manifests on disk.
	srv.bridgePluginsOverride = func(name string) (plugin.Provider, bool) {
		switch name {
		case "kanban":
			return plugin.Provider{Name: "kanban", Version: "1.0.0"}, true
		}
		for _, extra := range opts.extraPlugins {
			if name == extra {
				return plugin.Provider{Name: name, Version: "1.0.0"}, true
			}
		}
		return plugin.Provider{}, false
	}

	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	return ts, srv, a, token
}

// wsURL transforms an http:// httptest URL into ws:// with the given path.
func wsURL(t *testing.T, httpURL, path string) string {
	t.Helper()
	u, err := url.Parse(httpURL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	u.Scheme = "ws"
	u.Path = path
	return u.String()
}

// dialWS opens a WS connection with the given header tweaks.
func dialWS(t *testing.T, rawURL string, headers http.Header) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	d := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}
	return d.Dial(rawURL, headers)
}

// readEnvelope reads a single envelope from the WS and parses it. Fails the
// test on timeout.
func readEnvelope(t *testing.T, c *websocket.Conn) bridge.Envelope {
	t.Helper()
	c.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := c.ReadMessage()
	if err != nil {
		t.Fatalf("read envelope: %v", err)
	}
	var env bridge.Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		t.Fatalf("unmarshal envelope: %v; raw=%q", err, data)
	}
	return env
}

// writeEnvelope serialises env and sends it.
func writeEnvelope(t *testing.T, c *websocket.Conn, env bridge.Envelope) {
	t.Helper()
	b, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	if err := c.WriteMessage(websocket.TextMessage, b); err != nil {
		t.Fatalf("write envelope: %v", err)
	}
}

// ─── Tests ───────────────────────────────────────────────────────────────────

// TestBridgeWS_RequiresAuth — no Bearer → 401 BEFORE upgrade.
func TestBridgeWS_RequiresAuth(t *testing.T) {
	opts := defaultOpts()
	opts.withAuth = true
	ts, _, _, _ := newTestBridgeServer(t, opts)

	// Direct HTTP GET without auth — chi routes it through the auth middleware
	// which returns 401 before any WS upgrade.
	resp, err := http.Get(ts.URL + "/api/plugins/kanban/bridge/ws")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", resp.StatusCode)
	}
}

// TestBridgeWS_RejectsBadOrigin — Origin mismatched → 403 before upgrade.
func TestBridgeWS_RejectsBadOrigin(t *testing.T) {
	opts := defaultOpts()
	ts, _, _, _ := newTestBridgeServer(t, opts)

	h := http.Header{}
	h.Set("Origin", "https://evil.com")

	_, resp, err := dialWS(t, wsURL(t, ts.URL, "/api/plugins/kanban/bridge/ws"), h)
	if err == nil {
		t.Fatalf("expected dial to fail on bad origin")
	}
	if resp == nil {
		t.Fatalf("expected response on dial failure; err=%v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("want 403, got %d", resp.StatusCode)
	}
}

// TestBridgeWS_AcceptsLocalhost — Origin http://127.0.0.1:3000 → upgrade ok.
func TestBridgeWS_AcceptsLocalhost(t *testing.T) {
	opts := defaultOpts()
	ts, srv, _, _ := newTestBridgeServer(t, opts)
	srv.RegisterNamespace("echo", &echoNS{})

	h := http.Header{}
	h.Set("Origin", "http://127.0.0.1:3000")

	c, _, err := dialWS(t, wsURL(t, ts.URL, "/api/plugins/kanban/bridge/ws"), h)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()
	// connection up — send a quick roundtrip to confirm.
	writeEnvelope(t, c, bridge.Envelope{V: bridge.ProtocolVersion, ID: "1", NS: "echo", Method: "echo"})
	env := readEnvelope(t, c)
	if env.ID != "1" || env.Error != nil {
		t.Fatalf("expected OK echo response; got %+v", env)
	}
}

// TestBridgeWS_404IfPluginUnknown — plugin not registered → 404 before upgrade.
func TestBridgeWS_404IfPluginUnknown(t *testing.T) {
	opts := defaultOpts()
	ts, _, _, _ := newTestBridgeServer(t, opts)

	// Use a same-origin request so the origin check passes but plugin check fails.
	h := http.Header{}
	tsURL, _ := url.Parse(ts.URL)
	h.Set("Origin", "http://"+tsURL.Host)

	_, resp, err := dialWS(t, wsURL(t, ts.URL, "/api/plugins/ghost/bridge/ws"), h)
	if err == nil {
		t.Fatalf("expected dial to fail on unknown plugin")
	}
	if resp == nil || resp.StatusCode != http.StatusNotFound {
		var got int
		if resp != nil {
			got = resp.StatusCode
		}
		t.Fatalf("want 404, got %d; err=%v", got, err)
	}
}

// TestBridgeWS_EchoesEUNAVAILForUnknownNamespace — {ns:"nope",method:"x"} → EUNAVAIL.
func TestBridgeWS_EchoesEUNAVAILForUnknownNamespace(t *testing.T) {
	opts := defaultOpts()
	ts, _, _, _ := newTestBridgeServer(t, opts)

	h := http.Header{}
	tsURL, _ := url.Parse(ts.URL)
	h.Set("Origin", "http://"+tsURL.Host)

	c, _, err := dialWS(t, wsURL(t, ts.URL, "/api/plugins/kanban/bridge/ws"), h)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()

	writeEnvelope(t, c, bridge.Envelope{V: bridge.ProtocolVersion, ID: "1", NS: "nope", Method: "x"})
	env := readEnvelope(t, c)
	if env.ID != "1" {
		t.Fatalf("response id mismatch; got %+v", env)
	}
	if env.Error == nil || env.Error.Code != "EUNAVAIL" {
		t.Fatalf("want EUNAVAIL; got %+v", env.Error)
	}
}

// TestBridgeWS_RejectsBadVersion — v:2 → EINVAL.
func TestBridgeWS_RejectsBadVersion(t *testing.T) {
	opts := defaultOpts()
	ts, _, _, _ := newTestBridgeServer(t, opts)

	h := http.Header{}
	tsURL, _ := url.Parse(ts.URL)
	h.Set("Origin", "http://"+tsURL.Host)

	c, _, err := dialWS(t, wsURL(t, ts.URL, "/api/plugins/kanban/bridge/ws"), h)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()

	writeEnvelope(t, c, bridge.Envelope{V: 2, ID: "1", NS: "echo", Method: "echo"})
	env := readEnvelope(t, c)
	if env.Error == nil || env.Error.Code != "EINVAL" {
		t.Fatalf("want EINVAL; got %+v", env.Error)
	}
}

// TestBridgeWS_DispatchesToRegisteredNamespace — echo namespace returns args.
func TestBridgeWS_DispatchesToRegisteredNamespace(t *testing.T) {
	opts := defaultOpts()
	ts, srv, _, _ := newTestBridgeServer(t, opts)
	ns := &echoNS{}
	srv.RegisterNamespace("echo", ns)

	h := http.Header{}
	tsURL, _ := url.Parse(ts.URL)
	h.Set("Origin", "http://"+tsURL.Host)

	c, _, err := dialWS(t, wsURL(t, ts.URL, "/api/plugins/kanban/bridge/ws"), h)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()

	args := json.RawMessage(`{"x":42}`)
	writeEnvelope(t, c, bridge.Envelope{V: bridge.ProtocolVersion, ID: "42", NS: "echo", Method: "echo", Args: args})
	env := readEnvelope(t, c)
	if env.ID != "42" {
		t.Fatalf("id mismatch; got %q", env.ID)
	}
	if env.Error != nil {
		t.Fatalf("unexpected error: %+v", env.Error)
	}
	var result map[string]any
	if err := json.Unmarshal(env.Result, &result); err != nil {
		t.Fatalf("unmarshal result: %v; raw=%s", err, env.Result)
	}
	if result["method"] != "echo" {
		t.Errorf("method: want echo, got %v", result["method"])
	}
	if ns.calls.Load() != 1 {
		t.Errorf("dispatch count: want 1, got %d", ns.calls.Load())
	}
}

// TestBridgeWS_Concurrent100CallsSerialiseOK — 100 goroutines, all respond.
func TestBridgeWS_Concurrent100CallsSerialiseOK(t *testing.T) {
	opts := defaultOpts()
	opts.ratePerMinute = 10_000 // don't rate-limit this test
	ts, srv, _, _ := newTestBridgeServer(t, opts)
	srv.RegisterNamespace("echo", &echoNS{})

	h := http.Header{}
	tsURL, _ := url.Parse(ts.URL)
	h.Set("Origin", "http://"+tsURL.Host)

	c, _, err := dialWS(t, wsURL(t, ts.URL, "/api/plugins/kanban/bridge/ws"), h)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()

	const n = 100
	var wg sync.WaitGroup
	// Writes need to be serialised; but for the test we use a single mutex —
	// the handler's read side has to demux responses by correlation id.
	var writeMu sync.Mutex
	responses := make(chan bridge.Envelope, n)

	// Reader goroutine — fan-in all responses.
	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		for i := 0; i < n; i++ {
			c.SetReadDeadline(time.Now().Add(5 * time.Second))
			_, data, err := c.ReadMessage()
			if err != nil {
				t.Errorf("reader: %v", err)
				return
			}
			var env bridge.Envelope
			if err := json.Unmarshal(data, &env); err != nil {
				t.Errorf("reader unmarshal: %v", err)
				return
			}
			responses <- env
		}
	}()

	// Writer goroutines.
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			env := bridge.Envelope{V: bridge.ProtocolVersion, ID: fmt.Sprintf("r-%d", id), NS: "echo", Method: "echo"}
			b, _ := json.Marshal(env)
			writeMu.Lock()
			_ = c.WriteMessage(websocket.TextMessage, b)
			writeMu.Unlock()
		}(i)
	}
	wg.Wait()
	<-readerDone
	close(responses)

	seen := make(map[string]bool)
	for env := range responses {
		if env.Error != nil {
			t.Errorf("unexpected error on %s: %+v", env.ID, env.Error)
			continue
		}
		seen[env.ID] = true
	}
	if len(seen) != n {
		t.Errorf("want %d unique responses, got %d", n, len(seen))
	}
}

// TestBridgeWS_RateLimit — exceeding the bucket yields ETIMEOUT + retryAfterMs.
func TestBridgeWS_RateLimit(t *testing.T) {
	opts := defaultOpts()
	opts.ratePerMinute = 5
	ts, srv, _, _ := newTestBridgeServer(t, opts)
	srv.RegisterNamespace("echo", &echoNS{})

	h := http.Header{}
	tsURL, _ := url.Parse(ts.URL)
	h.Set("Origin", "http://"+tsURL.Host)

	c, _, err := dialWS(t, wsURL(t, ts.URL, "/api/plugins/kanban/bridge/ws"), h)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()

	// Fire 20 envelopes, one after another. The first 5 should succeed; the
	// rest should return ETIMEOUT.
	const total = 20
	var writeMu sync.Mutex
	for i := 0; i < total; i++ {
		env := bridge.Envelope{V: bridge.ProtocolVersion, ID: fmt.Sprintf("r-%d", i), NS: "echo", Method: "echo"}
		b, _ := json.Marshal(env)
		writeMu.Lock()
		_ = c.WriteMessage(websocket.TextMessage, b)
		writeMu.Unlock()
	}

	var okCount, limitCount int
	var seenRetryAfter bool
	for i := 0; i < total; i++ {
		env := readEnvelope(t, c)
		if env.Error == nil {
			okCount++
			continue
		}
		if env.Error.Code == "ETIMEOUT" {
			limitCount++
			if len(env.Error.Data) > 0 {
				var data map[string]any
				if err := json.Unmarshal(env.Error.Data, &data); err == nil {
					if ra, ok := data["retryAfterMs"].(float64); ok && ra > 0 {
						seenRetryAfter = true
					}
				}
			}
		}
	}
	if okCount == 0 {
		t.Errorf("expected some OK responses before rate-limit kicked in")
	}
	if limitCount == 0 {
		t.Errorf("expected at least one ETIMEOUT response")
	}
	if !seenRetryAfter {
		t.Errorf("expected retryAfterMs in WireError.Data")
	}
}

// TestBridgeWS_ClosesOnReadDeadline — idle connection is closed by the handler.
func TestBridgeWS_ClosesOnReadDeadline(t *testing.T) {
	opts := defaultOpts()
	opts.readTimeout = 150 * time.Millisecond
	ts, _, _, _ := newTestBridgeServer(t, opts)

	h := http.Header{}
	tsURL, _ := url.Parse(ts.URL)
	h.Set("Origin", "http://"+tsURL.Host)

	c, _, err := dialWS(t, wsURL(t, ts.URL, "/api/plugins/kanban/bridge/ws"), h)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()

	// Try to read — the server should close us out quickly.
	c.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err = c.ReadMessage()
	if err == nil {
		t.Fatalf("expected read error after idle close")
	}
}

// TestBridgeWS_PanicInNamespaceRecoversAndLogs — panicking namespace returns EINTERNAL.
func TestBridgeWS_PanicInNamespaceRecoversAndLogs(t *testing.T) {
	opts := defaultOpts()
	ts, srv, _, _ := newTestBridgeServer(t, opts)
	srv.RegisterNamespace("panic", &panicNS{})
	srv.RegisterNamespace("echo", &echoNS{})

	h := http.Header{}
	tsURL, _ := url.Parse(ts.URL)
	h.Set("Origin", "http://"+tsURL.Host)

	c, _, err := dialWS(t, wsURL(t, ts.URL, "/api/plugins/kanban/bridge/ws"), h)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()

	// First request panics.
	writeEnvelope(t, c, bridge.Envelope{V: bridge.ProtocolVersion, ID: "1", NS: "panic", Method: "x"})
	env := readEnvelope(t, c)
	if env.Error == nil || env.Error.Code != "EINTERNAL" {
		t.Fatalf("want EINTERNAL on panic; got %+v", env)
	}

	// Follow-up request to echo should still succeed — conn stayed alive.
	writeEnvelope(t, c, bridge.Envelope{V: bridge.ProtocolVersion, ID: "2", NS: "echo", Method: "echo"})
	env2 := readEnvelope(t, c)
	if env2.ID != "2" || env2.Error != nil {
		t.Fatalf("second request should succeed; got %+v", env2)
	}
}

// TestBridgeWS_DisconnectUnregistersFromManager — mgr.ActiveConns == 0 after close.
func TestBridgeWS_DisconnectUnregistersFromManager(t *testing.T) {
	opts := defaultOpts()
	ts, srv, _, _ := newTestBridgeServer(t, opts)

	h := http.Header{}
	tsURL, _ := url.Parse(ts.URL)
	h.Set("Origin", "http://"+tsURL.Host)

	c, _, err := dialWS(t, wsURL(t, ts.URL, "/api/plugins/kanban/bridge/ws"), h)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	// Wait a tick so the handler finishes Register.
	waitFor(t, 500*time.Millisecond, func() bool {
		return srv.bridgeMgr.ActiveConns("kanban") == 1
	}, "ActiveConns should be 1 after dial")

	// Close from client side — handler should Unregister on read error.
	c.Close()

	waitFor(t, 500*time.Millisecond, func() bool {
		return srv.bridgeMgr.ActiveConns("kanban") == 0
	}, "ActiveConns should be 0 after disconnect")
}

// waitFor polls until cond returns true or deadline hits. Fails with msg.
func waitFor(t *testing.T, within time.Duration, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting: %s", msg)
}

// ─── Auxiliary checks ────────────────────────────────────────────────────────

// TestBridgeWS_AcceptsAppOpendrayOrigin — Origin app://opendray (mobile WebView).
func TestBridgeWS_AcceptsAppOpendrayOrigin(t *testing.T) {
	opts := defaultOpts()
	ts, _, _, _ := newTestBridgeServer(t, opts)

	h := http.Header{}
	h.Set("Origin", "app://opendray")

	c, _, err := dialWS(t, wsURL(t, ts.URL, "/api/plugins/kanban/bridge/ws"), h)
	if err != nil {
		t.Fatalf("dial with app://opendray should succeed: %v", err)
	}
	defer c.Close()
}

// TestBridgeWS_AcceptsConfiguredFrontendOrigin — cfg.FrontendOrigin accepted.
func TestBridgeWS_AcceptsConfiguredFrontendOrigin(t *testing.T) {
	opts := defaultOpts()
	opts.frontendOrigin = "https://opendray.example.com"
	ts, _, _, _ := newTestBridgeServer(t, opts)

	h := http.Header{}
	h.Set("Origin", opts.frontendOrigin)

	c, _, err := dialWS(t, wsURL(t, ts.URL, "/api/plugins/kanban/bridge/ws"), h)
	if err != nil {
		t.Fatalf("dial with configured origin should succeed: %v", err)
	}
	defer c.Close()
}

// permDenyNS returns a *bridge.PermError on every Dispatch — used to verify
// that the handler maps it to an EPERM envelope.
type permDenyNS struct{}

func (p *permDenyNS) Dispatch(_ context.Context, _ string, _ string, _ json.RawMessage, _ *bridge.Conn) (any, error) {
	return nil, &bridge.PermError{Code: "EPERM", Msg: "capability denied"}
}

// TestBridgeWS_PermErrorReturnsEPERM — Namespace returning *bridge.PermError → EPERM envelope.
func TestBridgeWS_PermErrorReturnsEPERM(t *testing.T) {
	opts := defaultOpts()
	ts, srv, _, _ := newTestBridgeServer(t, opts)
	srv.RegisterNamespace("perm", &permDenyNS{})

	h := http.Header{}
	tsURL, _ := url.Parse(ts.URL)
	h.Set("Origin", "http://"+tsURL.Host)

	c, _, err := dialWS(t, wsURL(t, ts.URL, "/api/plugins/kanban/bridge/ws"), h)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()

	writeEnvelope(t, c, bridge.Envelope{V: bridge.ProtocolVersion, ID: "1", NS: "perm", Method: "x"})
	env := readEnvelope(t, c)
	if env.Error == nil || env.Error.Code != "EPERM" {
		t.Fatalf("want EPERM; got %+v", env.Error)
	}
}

// errorNS returns a plain error — should map to EINTERNAL.
type errorNS struct{}

func (e *errorNS) Dispatch(_ context.Context, _ string, _ string, _ json.RawMessage, _ *bridge.Conn) (any, error) {
	return nil, fmt.Errorf("something went wrong")
}

// TestBridgeWS_PlainErrorReturnsEINTERNAL — non-PermError maps to EINTERNAL.
func TestBridgeWS_PlainErrorReturnsEINTERNAL(t *testing.T) {
	opts := defaultOpts()
	ts, srv, _, _ := newTestBridgeServer(t, opts)
	srv.RegisterNamespace("err", &errorNS{})

	h := http.Header{}
	tsURL, _ := url.Parse(ts.URL)
	h.Set("Origin", "http://"+tsURL.Host)

	c, _, err := dialWS(t, wsURL(t, ts.URL, "/api/plugins/kanban/bridge/ws"), h)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()

	writeEnvelope(t, c, bridge.Envelope{V: bridge.ProtocolVersion, ID: "1", NS: "err", Method: "x"})
	env := readEnvelope(t, c)
	if env.Error == nil || env.Error.Code != "EINTERNAL" {
		t.Fatalf("want EINTERNAL; got %+v", env.Error)
	}
}

// TestBridgeWS_RealRuntimePluginMissing — no override, real runtime returns
// 404 for unknown plugin. Exercises the production s.plugins.Get branch.
func TestBridgeWS_RealRuntimePluginMissing(t *testing.T) {
	opts := defaultOpts()
	ts, srv, _, _ := newTestBridgeServer(t, opts)
	srv.bridgePluginsOverride = nil // fall back to s.plugins.Get

	h := http.Header{}
	tsURL, _ := url.Parse(ts.URL)
	h.Set("Origin", "http://"+tsURL.Host)

	_, resp, err := dialWS(t, wsURL(t, ts.URL, "/api/plugins/nope/bridge/ws"), h)
	if err == nil {
		t.Fatalf("expected dial to fail")
	}
	if resp == nil || resp.StatusCode != http.StatusNotFound {
		var got int
		if resp != nil {
			got = resp.StatusCode
		}
		t.Fatalf("want 404, got %d; err=%v", got, err)
	}
}

// TestBridgeWS_BridgeManagerMissing — cfg.BridgeManager==nil → 503 EBRIDGE.
func TestBridgeWS_BridgeManagerMissing(t *testing.T) {
	// Hand-build a Server that bypasses the manager wiring; we can't use
	// newTestBridgeServer because that always sets BridgeManager.
	hooks := plugin.NewHookBus(nil)
	rt := plugin.NewRuntime(nil, hooks, "", nil)
	srv := New(Config{
		Plugins: rt,
		// BridgeManager intentionally nil.
	})
	srv.bridgePluginsOverride = func(name string) (plugin.Provider, bool) {
		if name == "kanban" {
			return plugin.Provider{Name: "kanban", Version: "1.0.0"}, true
		}
		return plugin.Provider{}, false
	}

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	h := http.Header{}
	tsURL, _ := url.Parse(ts.URL)
	h.Set("Origin", "http://"+tsURL.Host)

	_, resp, err := dialWS(t, wsURL(t, ts.URL, "/api/plugins/kanban/bridge/ws"), h)
	if err == nil {
		t.Fatalf("expected dial to fail")
	}
	if resp == nil || resp.StatusCode != http.StatusServiceUnavailable {
		var got int
		if resp != nil {
			got = resp.StatusCode
		}
		t.Fatalf("want 503 EBRIDGE, got %d; err=%v", got, err)
	}
}

// unmarshalableNS returns a result that cannot be JSON-marshalled (a channel).
type unmarshalableNS struct{}

func (u *unmarshalableNS) Dispatch(_ context.Context, _ string, _ string, _ json.RawMessage, _ *bridge.Conn) (any, error) {
	return make(chan int), nil // channels don't marshal → EINTERNAL path
}

// TestBridgeWS_MarshalFailureReturnsEINTERNAL — result that can't be marshalled → EINTERNAL.
func TestBridgeWS_MarshalFailureReturnsEINTERNAL(t *testing.T) {
	opts := defaultOpts()
	ts, srv, _, _ := newTestBridgeServer(t, opts)
	srv.RegisterNamespace("bad", &unmarshalableNS{})

	h := http.Header{}
	tsURL, _ := url.Parse(ts.URL)
	h.Set("Origin", "http://"+tsURL.Host)

	c, _, err := dialWS(t, wsURL(t, ts.URL, "/api/plugins/kanban/bridge/ws"), h)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()

	writeEnvelope(t, c, bridge.Envelope{V: bridge.ProtocolVersion, ID: "1", NS: "bad", Method: "x"})
	env := readEnvelope(t, c)
	if env.Error == nil || env.Error.Code != "EINTERNAL" {
		t.Fatalf("want EINTERNAL; got %+v", env.Error)
	}
}

// TestBridgeWS_StreamEnvelopeIgnored — inbound stream envelope is discarded.
func TestBridgeWS_StreamEnvelopeIgnored(t *testing.T) {
	opts := defaultOpts()
	ts, srv, _, _ := newTestBridgeServer(t, opts)
	srv.RegisterNamespace("echo", &echoNS{})

	h := http.Header{}
	tsURL, _ := url.Parse(ts.URL)
	h.Set("Origin", "http://"+tsURL.Host)

	c, _, err := dialWS(t, wsURL(t, ts.URL, "/api/plugins/kanban/bridge/ws"), h)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()

	// Send a stream chunk envelope — handler should ignore it (no response).
	writeEnvelope(t, c, bridge.Envelope{V: bridge.ProtocolVersion, ID: "s1", Stream: "chunk"})
	// Follow with a real request — we should get a response with id="2", not "s1".
	writeEnvelope(t, c, bridge.Envelope{V: bridge.ProtocolVersion, ID: "2", NS: "echo", Method: "echo"})
	env := readEnvelope(t, c)
	if env.ID != "2" {
		t.Fatalf("expected response for id=2, got %s", env.ID)
	}
}

// TestBridgeWS_MalformedJSONReturnsEINVAL — bad JSON → EINVAL with empty id.
func TestBridgeWS_MalformedJSONReturnsEINVAL(t *testing.T) {
	opts := defaultOpts()
	ts, _, _, _ := newTestBridgeServer(t, opts)

	h := http.Header{}
	tsURL, _ := url.Parse(ts.URL)
	h.Set("Origin", "http://"+tsURL.Host)

	c, _, err := dialWS(t, wsURL(t, ts.URL, "/api/plugins/kanban/bridge/ws"), h)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()

	if err := c.WriteMessage(websocket.TextMessage, []byte("not-json")); err != nil {
		t.Fatalf("write: %v", err)
	}
	env := readEnvelope(t, c)
	if env.Error == nil || env.Error.Code != "EINVAL" {
		t.Fatalf("want EINVAL; got %+v", env.Error)
	}
}

// TestBridgeWS_EmptyMethodYieldsEINVAL — missing method → EINVAL.
func TestBridgeWS_EmptyMethodYieldsEINVAL(t *testing.T) {
	opts := defaultOpts()
	ts, srv, _, _ := newTestBridgeServer(t, opts)
	srv.RegisterNamespace("echo", &echoNS{})

	h := http.Header{}
	tsURL, _ := url.Parse(ts.URL)
	h.Set("Origin", "http://"+tsURL.Host)

	c, _, err := dialWS(t, wsURL(t, ts.URL, "/api/plugins/kanban/bridge/ws"), h)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()

	writeEnvelope(t, c, bridge.Envelope{V: bridge.ProtocolVersion, ID: "1", NS: "echo"})
	env := readEnvelope(t, c)
	if env.Error == nil || env.Error.Code != "EINVAL" {
		t.Fatalf("want EINVAL for empty method; got %+v", env.Error)
	}
}


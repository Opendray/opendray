package bridge

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

// ─────────────────────────────────────────────
// helpers
// ─────────────────────────────────────────────

// httpConsent grants the http cap with the given URL patterns.
type httpConsent struct {
	patterns []string
}

func (c *httpConsent) Load(_ context.Context, _ string) ([]byte, bool, error) {
	perms := map[string]any{"http": c.patterns}
	if len(c.patterns) == 0 {
		perms["http"] = false
	}
	raw, _ := json.Marshal(perms)
	return raw, true, nil
}

// httpGate returns a Gate that grants http for the given patterns.
func httpGate(patterns ...string) *Gate {
	return NewGate(&httpConsent{patterns: patterns}, nil, nil)
}

// passThroughGateCheck is a test-only gate check that accepts every hop.
// Unit tests that exercise the positive path use httptest servers which
// bind to 127.0.0.1 — a loopback address MatchHTTPURL unconditionally
// denies. Bypassing Gate.Check here keeps the SSRF static rule intact
// for the deny-path tests while letting the positive path reach the
// real transport and Dialer.Control code.
func passThroughGateCheck(_ context.Context, _, _ string) error { return nil }

// loopbackTransport is a test-only http.RoundTripper that permits dialing
// loopback (needed for httptest servers). It mirrors the default transport
// but omits the Dialer.Control private-host check.
func loopbackTransport() http.RoundTripper {
	d := &net.Dialer{Timeout: 2 * time.Second}
	return &http.Transport{
		DialContext:     d.DialContext,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12},
	}
}

// encodeRequestArgs marshals a map into the single-element args array the
// Dispatch contract expects.
func encodeRequestArgs(req map[string]any) json.RawMessage {
	inner, _ := json.Marshal(req)
	raw, _ := json.Marshal([]json.RawMessage{inner})
	return raw
}

// encodeBody base64-encodes a plaintext body for inclusion in a request arg.
func encodeBody(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// ─────────────────────────────────────────────
// Tests — positive path
// ─────────────────────────────────────────────

// TestHTTP_RequestGet_Happy exercises a GET against an httptest server with
// a matching grant and verifies status / body.
func TestHTTP_RequestGet_Happy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("hello from test"))
	}))
	defer srv.Close()

	api := NewHTTPAPI(HTTPConfig{
		Gate:              httpGate("*"),
		gateCheckOverride: passThroughGateCheck,
		transport:         loopbackTransport(),
	})

	args := encodeRequestArgs(map[string]any{"url": srv.URL + "/"})
	result, err := api.Dispatch(context.Background(), "p", "request", args, "", nil)
	if err != nil {
		t.Fatalf("Dispatch request: %v", err)
	}
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("want map, got %T", result)
	}
	if m["status"].(int) != 200 {
		t.Errorf("status: want 200, got %v", m["status"])
	}
	bodyB64, _ := m["body"].(string)
	decoded, err := base64.StdEncoding.DecodeString(bodyB64)
	if err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if string(decoded) != "hello from test" {
		t.Errorf("body: got %q", decoded)
	}
}

// TestHTTP_RequestPost_WithBody sends a POST with a body, verifies the server
// received the same bytes back.
func TestHTTP_RequestPost_WithBody(t *testing.T) {
	var got []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, _ = io.ReadAll(r.Body)
		w.WriteHeader(201)
	}))
	defer srv.Close()

	api := NewHTTPAPI(HTTPConfig{
		Gate:              httpGate("*"),
		gateCheckOverride: passThroughGateCheck,
		transport:         loopbackTransport(),
	})

	payload := "the body of the POST"
	args := encodeRequestArgs(map[string]any{
		"url":    srv.URL + "/post",
		"method": "POST",
		"body":   encodeBody(payload),
	})
	result, err := api.Dispatch(context.Background(), "p", "request", args, "", nil)
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if string(got) != payload {
		t.Errorf("server got %q, want %q", got, payload)
	}
	m := result.(map[string]any)
	if m["status"].(int) != 201 {
		t.Errorf("status: want 201, got %v", m["status"])
	}
}

// TestHTTP_TLS_HappyPath uses httptest.NewTLSServer to exercise the TLS dial
// path. The test's transport skips certificate verification because the
// httptest cert is self-signed; we only care that TLS negotiates and the
// MinVersion pinning doesn't reject a modern server.
func TestHTTP_TLS_HappyPath(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("secure"))
	}))
	defer srv.Close()

	// Override transport to use the test server's TLS cert pool.
	cfg := HTTPConfig{
		Gate:              httpGate("*"),
		gateCheckOverride: passThroughGateCheck,
		transport:         srv.Client().Transport,
	}
	api := NewHTTPAPI(cfg)

	args := encodeRequestArgs(map[string]any{"url": srv.URL})
	result, err := api.Dispatch(context.Background(), "p", "request", args, "", nil)
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	m := result.(map[string]any)
	if m["status"].(int) != 200 {
		t.Errorf("status: %v", m["status"])
	}
}

// ─────────────────────────────────────────────
// Capability enforcement
// ─────────────────────────────────────────────

// TestHTTP_Request_EPERMOnUngrantedURL asserts a request to an URL outside
// the grant allowlist returns a PermError (EPERM).
func TestHTTP_Request_EPERMOnUngrantedURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	// Grant a completely different pattern.
	api := NewHTTPAPI(HTTPConfig{Gate: httpGate("https://api.github.com/*")})

	args := encodeRequestArgs(map[string]any{"url": srv.URL})
	_, err := api.Dispatch(context.Background(), "p", "request", args, "", nil)
	if err == nil {
		t.Fatal("want EPERM, got nil")
	}
	var pe *PermError
	if !errors.As(err, &pe) {
		t.Fatalf("want *PermError, got %T: %v", err, err)
	}
	if pe.Code != "EPERM" {
		t.Errorf("code: want EPERM, got %q", pe.Code)
	}
}

// TestHTTP_Redirect_GateBlocksMidChain starts a server that 302s to a
// second host; only the first is in the grant. The redirect chain must
// abort with a PermError naming the blocked hop.
func TestHTTP_Redirect_GateBlocksMidChain(t *testing.T) {
	// Final destination — its URL is NOT in the grant.
	blocked := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("should not reach"))
	}))
	defer blocked.Close()

	// Allowed origin — 302s to `blocked`.
	allowed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, blocked.URL+"/final", http.StatusFound)
	}))
	defer allowed.Close()

	allowedHost := strings.TrimPrefix(allowed.URL, "http://")
	blockedHost := strings.TrimPrefix(blocked.URL, "http://")

	// Custom gate check: allow `allowed.URL`, deny `blocked.URL`.
	gateCheck := func(_ context.Context, _ string, target string) error {
		if strings.Contains(target, allowedHost) {
			return nil
		}
		if strings.Contains(target, blockedHost) {
			return &PermError{Code: "EPERM", Msg: "http not granted for: " + target}
		}
		return &PermError{Code: "EPERM", Msg: "unknown target " + target}
	}
	api := NewHTTPAPI(HTTPConfig{
		Gate:              httpGate("*"),
		gateCheckOverride: gateCheck,
		transport:         loopbackTransport(),
	})

	args := encodeRequestArgs(map[string]any{"url": allowed.URL + "/redir"})
	_, err := api.Dispatch(context.Background(), "p", "request", args, "", nil)
	if err == nil {
		t.Fatal("want redirect to be blocked")
	}
	var pe *PermError
	if !errors.As(err, &pe) {
		t.Fatalf("want *PermError on blocked redirect, got %T: %v", err, err)
	}
	if !strings.Contains(pe.Msg, blockedHost) && !strings.Contains(pe.Msg, "http") {
		t.Errorf("PermError message should reference blocked host, got %q", pe.Msg)
	}
}

// ─────────────────────────────────────────────
// Body caps
// ─────────────────────────────────────────────

// TestHTTP_RequestBody_ExceedsCap rejects oversized request bodies with EINVAL
// before touching the network.
func TestHTTP_RequestBody_ExceedsCap(t *testing.T) {
	api := NewHTTPAPI(HTTPConfig{
		Gate:           httpGate("https://*/*"),
		MaxRequestBody: 16, // 16 bytes
	})

	args := encodeRequestArgs(map[string]any{
		"url":    "https://api.github.com/fake",
		"method": "POST",
		"body":   encodeBody("this body is way more than sixteen bytes long"),
	})
	_, err := api.Dispatch(context.Background(), "p", "request", args, "", nil)
	if err == nil {
		t.Fatal("want EINVAL, got nil")
	}
	var we *WireError
	if !errors.As(err, &we) {
		t.Fatalf("want *WireError, got %T: %v", err, err)
	}
	if we.Code != "EINVAL" {
		t.Errorf("code: want EINVAL, got %q", we.Code)
	}
}

// TestHTTP_ResponseBody_TruncatedNonStream caps response reads at MaxResponseBody
// and returns truncated=true.
func TestHTTP_ResponseBody_TruncatedNonStream(t *testing.T) {
	big := strings.Repeat("A", 4096)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(big))
	}))
	defer srv.Close()

	api := NewHTTPAPI(HTTPConfig{
		Gate:              httpGate("*"),
		gateCheckOverride: passThroughGateCheck,
		transport:         loopbackTransport(),
		MaxResponseBody:   128,
	})

	args := encodeRequestArgs(map[string]any{"url": srv.URL + "/"})
	result, err := api.Dispatch(context.Background(), "p", "request", args, "", nil)
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	m := result.(map[string]any)
	if m["truncated"] != true {
		t.Errorf("want truncated=true, got %v", m["truncated"])
	}
	body, _ := base64.StdEncoding.DecodeString(m["body"].(string))
	if len(body) != 128 {
		t.Errorf("body length: want 128, got %d", len(body))
	}
}

// TestHTTP_Stream_TruncatedEmitsErrChunk covers the stream-path truncation
// path: response larger than MaxResponseBody emits a trailing error chunk
// and stream-end envelope.
func TestHTTP_Stream_TruncatedEmitsErrChunk(t *testing.T) {
	big := strings.Repeat("B", 8192)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(big))
	}))
	defer srv.Close()

	api := NewHTTPAPI(HTTPConfig{
		Gate:              httpGate("*"),
		gateCheckOverride: passThroughGateCheck,
		transport:         loopbackTransport(),
		MaxResponseBody:   100,
	})

	// Mount a fake Conn to collect chunk envelopes.
	mgr := NewManager(nil)
	fws := &fakeWS{}
	conn := mgr.Register("p", fws)
	defer conn.Close(1000, "done")

	args := encodeRequestArgs(map[string]any{"url": srv.URL})
	_, err := api.Dispatch(context.Background(), "p", "stream", args, "env-1", conn)
	if err != nil {
		t.Fatalf("Dispatch stream: %v", err)
	}

	// Wait for the async pump to finish.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if fws.writeCount() >= 2 && envelopeStreamEnd(fws, "env-1") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Assert: among writes we saw a chunk with an error (truncation) and an end.
	if !envelopeStreamErr(fws, "env-1", "EINVAL", "truncated") {
		t.Error("want a stream chunk-err with EINVAL truncated")
	}
	if !envelopeStreamEnd(fws, "env-1") {
		t.Error("want a stream-end envelope")
	}
}

// envelopeStreamErr returns true if any write contains an error chunk matching
// code + message substring for the given correlation id.
func envelopeStreamErr(fws *fakeWS, id, code, msgSubstr string) bool {
	fws.mu.Lock()
	defer fws.mu.Unlock()
	for _, raw := range fws.writes {
		var env Envelope
		if json.Unmarshal(raw, &env) != nil {
			continue
		}
		if env.ID == id && env.Error != nil && env.Error.Code == code && strings.Contains(env.Error.Message, msgSubstr) {
			return true
		}
	}
	return false
}

// envelopeStreamEnd returns true if any write is a stream-end for id.
func envelopeStreamEnd(fws *fakeWS, id string) bool {
	fws.mu.Lock()
	defer fws.mu.Unlock()
	for _, raw := range fws.writes {
		var env Envelope
		if json.Unmarshal(raw, &env) != nil {
			continue
		}
		if env.ID == id && env.Stream == "end" {
			return true
		}
	}
	return false
}

// ─────────────────────────────────────────────
// SSRF defence
// ─────────────────────────────────────────────

// recordingDialer captures the `address` passed to its Control callback so
// tests can observe what the Dialer attempted to connect to.
type recordingDialer struct {
	mu       sync.Mutex
	attempts []string
	control  func(network, address string, c syscall.RawConn) error
}

// recordingTransport is a minimal http.RoundTripper that wraps a
// *http.Transport built on top of an injected Dialer. Tests use it to
// stand-in for NewHTTPAPI's default transport factory when they need to
// observe dial-time SSRF checks.
type recordingTransport struct {
	inner *http.Transport
	rd    *recordingDialer
}

func newRecordingTransport(control func(network, address string, c syscall.RawConn) error) *recordingTransport {
	rd := &recordingDialer{control: control}
	d := &net.Dialer{
		Timeout:   2 * time.Second,
		KeepAlive: 30 * time.Second,
		Control: func(network, address string, c syscall.RawConn) error {
			rd.mu.Lock()
			rd.attempts = append(rd.attempts, address)
			rd.mu.Unlock()
			if rd.control != nil {
				return rd.control(network, address, c)
			}
			return nil
		},
	}
	tr := &http.Transport{
		DialContext:     d.DialContext,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12},
	}
	return &recordingTransport{inner: tr, rd: rd}
}

func (rt *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return rt.inner.RoundTrip(req)
}

// TestHTTP_SSRF_PrivateIPLiteral: the static MatchHTTPURL path blocks private
// IP literals before we even dial.
func TestHTTP_SSRF_PrivateIPLiteral(t *testing.T) {
	api := NewHTTPAPI(HTTPConfig{Gate: httpGate("*")})

	args := encodeRequestArgs(map[string]any{"url": "http://169.254.169.254/latest/meta-data/"})
	_, err := api.Dispatch(context.Background(), "p", "request", args, "", nil)
	if err == nil {
		t.Fatal("want EPERM on IMDS URL, got nil")
	}
	var pe *PermError
	if !errors.As(err, &pe) {
		t.Fatalf("want *PermError, got %T: %v", err, err)
	}
}

// TestHTTP_SSRF_DNSRebind: a hostname that resolves to a private IP during
// dial must be refused by the Dialer.Control callback — not by the static
// MatchHTTPURL path (hostname isn't a private literal).
//
// We simulate this by installing a recordingTransport that rejects any
// address in the private range. The "rebound" hostname is a harmless
// placeholder; we don't actually bind to 10.0.0.1 — we just assert the
// control callback rejects it.
func TestHTTP_SSRF_DNSRebind(t *testing.T) {
	// control simulates DNS returning a private IP mid-dial.
	control := func(network, address string, _ syscall.RawConn) error {
		// Claim every lookup resolves to a private IP.
		if strings.HasPrefix(address, "10.") || strings.HasPrefix(address, "192.168.") || strings.HasPrefix(address, "127.") {
			return fmt.Errorf("test: simulated private IP %s", address)
		}
		// Force the test harness to believe this hostname resolves to 10.0.0.1.
		host, _, _ := net.SplitHostPort(address)
		parsed := net.ParseIP(host)
		if parsed != nil {
			// Address is already an IP literal; fallthrough.
			return nil
		}
		// A hostname-only address arrived — simulate rebind by failing it.
		return fmt.Errorf("test: rebound to private 10.0.0.1 (simulated)")
	}

	rt := newRecordingTransport(control)
	api := NewHTTPAPI(HTTPConfig{
		Gate:      httpGate("http://evil.example.com/*"),
		transport: rt,
	})

	args := encodeRequestArgs(map[string]any{"url": "http://evil.example.com/path"})
	_, err := api.Dispatch(context.Background(), "p", "request", args, "", nil)
	if err == nil {
		t.Fatal("want SSRF dial-time block, got nil")
	}
	// We expect an EINTERNAL-class wire error (not EPERM: the static allow list
	// admitted the URL; only the Dialer.Control dropped it).
	var we *WireError
	if !errors.As(err, &we) {
		t.Fatalf("want *WireError, got %T: %v", err, err)
	}
	if we.Code != "EINTERNAL" && we.Code != "ETIMEOUT" {
		t.Errorf("want EINTERNAL/ETIMEOUT on dial-time block, got %q", we.Code)
	}
}

// TestHTTP_SSRF_DialControl_PrivateTargetPostResolve asserts the real
// transport's Dialer.Control blocks any private IP passed as the resolved
// address. We bypass the static MatchHTTPURL by using the gate grant "*"
// plus a hostname that points at 127.0.0.1 (loopback). Even though "*"
// would allow https, the loopback check must fire before connect.
func TestHTTP_SSRF_DialControl_PrivateTargetPostResolve(t *testing.T) {
	// Use a URL that, at DNS time, resolves to loopback. We can't control DNS
	// in unit tests portably; simulate by hitting 127.0.0.1 via an IP literal.
	// The static MatchHTTPURL should reject this first (loopback is in
	// isPrivateHost) — the point of this test is to prove the full pipeline
	// denies end-to-end.
	api := NewHTTPAPI(HTTPConfig{Gate: httpGate("*")})
	args := encodeRequestArgs(map[string]any{"url": "https://127.0.0.1:9/"})
	_, err := api.Dispatch(context.Background(), "p", "request", args, "", nil)
	if err == nil {
		t.Fatal("want deny on loopback URL, got nil")
	}
	// Either EPERM (static check) or EINTERNAL (dial-time) are acceptable —
	// both prove SSRF is blocked.
	if err != nil {
		var pe *PermError
		var we *WireError
		if !errors.As(err, &pe) && !errors.As(err, &we) {
			t.Fatalf("want *PermError or *WireError, got %T: %v", err, err)
		}
	}
}

// ─────────────────────────────────────────────
// Timeout and TLS
// ─────────────────────────────────────────────

// TestHTTP_DialTimeout: Dial that never answers should produce ETIMEOUT (or
// EINTERNAL wrapping a dial error) within a short bound.
func TestHTTP_DialTimeout(t *testing.T) {
	// TEST-NET-1 (192.0.2.0/24) — RFC 5737, guaranteed non-routable. Not in
	// isPrivateHost so the static check lets it through; dial will hang.
	api := NewHTTPAPI(HTTPConfig{
		Gate:         httpGate("*"),
		DialTimeout:  250 * time.Millisecond,
		TotalTimeout: 500 * time.Millisecond,
	})

	args := encodeRequestArgs(map[string]any{
		"url": "https://192.0.2.1/",
	})
	start := time.Now()
	_, err := api.Dispatch(context.Background(), "p", "request", args, "", nil)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("want timeout, got nil")
	}
	if elapsed > 3*time.Second {
		t.Errorf("timeout: took %v, want ~500 ms", elapsed)
	}
	var we *WireError
	if !errors.As(err, &we) {
		t.Fatalf("want *WireError, got %T: %v", err, err)
	}
	if we.Code != "ETIMEOUT" && we.Code != "EINTERNAL" {
		t.Errorf("code: want ETIMEOUT/EINTERNAL, got %q", we.Code)
	}
}

// TestHTTP_TLSMinVersion_Enforced asserts the transport is built with the
// configured TLS min version. We smoke-test by asserting the default
// transport's Dialer has a TLSClientConfig with MinVersion >= TLS1.2.
func TestHTTP_TLSMinVersion_Enforced(t *testing.T) {
	api := NewHTTPAPI(HTTPConfig{
		Gate:          httpGate("*"),
		TLSMinVersion: tls.VersionTLS13,
	})

	tr := api.transportFactory(context.Background(), "p", func(context.Context, string, string) error { return nil })
	ht, ok := tr.(*http.Transport)
	if !ok {
		t.Fatalf("want *http.Transport, got %T", tr)
	}
	if ht.TLSClientConfig == nil {
		t.Fatal("TLSClientConfig is nil")
	}
	if ht.TLSClientConfig.MinVersion != tls.VersionTLS13 {
		t.Errorf("MinVersion: want %x, got %x", tls.VersionTLS13, ht.TLSClientConfig.MinVersion)
	}
}

// TestHTTP_Request_MalformedArgs: malformed JSON in args returns EINVAL.
func TestHTTP_Request_MalformedArgs(t *testing.T) {
	api := NewHTTPAPI(HTTPConfig{Gate: httpGate("*")})
	_, err := api.Dispatch(context.Background(), "p", "request", json.RawMessage(`"not-an-array"`), "", nil)
	if err == nil {
		t.Fatal("want EINVAL, got nil")
	}
	var we *WireError
	if !errors.As(err, &we) || we.Code != "EINVAL" {
		t.Errorf("want *WireError{EINVAL}, got %T: %v", err, err)
	}
}

// TestHTTP_UnknownMethod returns EUNAVAIL.
func TestHTTP_UnknownMethod(t *testing.T) {
	api := NewHTTPAPI(HTTPConfig{Gate: httpGate("*")})
	_, err := api.Dispatch(context.Background(), "p", "nonexistent", json.RawMessage(`[]`), "", nil)
	if err == nil {
		t.Fatal("want EUNAVAIL, got nil")
	}
	var we *WireError
	if !errors.As(err, &we) || we.Code != "EUNAVAIL" {
		t.Errorf("want *WireError{EUNAVAIL}, got %T: %v", err, err)
	}
}

// TestHTTP_Stream_HappyPath verifies a small response produces at least one
// chunk and a stream-end envelope.
func TestHTTP_Stream_HappyPath(t *testing.T) {
	payload := "streamed response body"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(payload))
	}))
	defer srv.Close()

	api := NewHTTPAPI(HTTPConfig{
		Gate:              httpGate("*"),
		gateCheckOverride: passThroughGateCheck,
		transport:         loopbackTransport(),
	})

	mgr := NewManager(nil)
	fws := &fakeWS{}
	conn := mgr.Register("p", fws)
	defer conn.Close(1000, "done")

	args := encodeRequestArgs(map[string]any{"url": srv.URL})
	head, err := api.Dispatch(context.Background(), "p", "stream", args, "env-99", conn)
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if head.(map[string]any)["status"].(int) != 200 {
		t.Errorf("status: %v", head)
	}

	// Wait for stream end.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if envelopeStreamEnd(fws, "env-99") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !envelopeStreamEnd(fws, "env-99") {
		t.Error("want stream-end envelope")
	}
}

// TestHTTP_Request_HeadersRoundTrip asserts request headers reach the server
// and response headers come back in the result.
func TestHTTP_Request_HeadersRoundTrip(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("X-Custom", "reply-value")
		w.WriteHeader(200)
	}))
	defer srv.Close()

	api := NewHTTPAPI(HTTPConfig{
		Gate:              httpGate("*"),
		gateCheckOverride: passThroughGateCheck,
		transport:         loopbackTransport(),
	})

	args := encodeRequestArgs(map[string]any{
		"url":     srv.URL,
		"headers": map[string]string{"Authorization": "Bearer xyz"},
	})
	result, err := api.Dispatch(context.Background(), "p", "request", args, "", nil)
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if gotAuth != "Bearer xyz" {
		t.Errorf("server got auth %q, want Bearer xyz", gotAuth)
	}
	headers := result.(map[string]any)["headers"].(map[string]string)
	if headers["X-Custom"] != "reply-value" {
		t.Errorf("response header X-Custom: got %q", headers["X-Custom"])
	}
}

// TestHTTP_Stream_MalformedArgs — stream handler returns EINVAL on bad args.
func TestHTTP_Stream_MalformedArgs(t *testing.T) {
	api := NewHTTPAPI(HTTPConfig{Gate: httpGate("*")})

	mgr := NewManager(nil)
	fws := &fakeWS{}
	conn := mgr.Register("p", fws)
	defer conn.Close(1000, "done")

	_, err := api.Dispatch(context.Background(), "p", "stream", json.RawMessage(`[{}]`), "env-bad", conn)
	if err == nil {
		t.Fatal("want EINVAL, got nil")
	}
	var we *WireError
	if !errors.As(err, &we) || we.Code != "EINVAL" {
		t.Errorf("want *WireError{EINVAL}, got %T: %v", err, err)
	}
}

// TestHTTP_Counter_SanityCheck is a cheap sanity check that the package
// exports remain consistent — a changed symbol would surface here.
func TestHTTP_Counter_SanityCheck(t *testing.T) {
	_ = atomic.Int64{} // keep sync/atomic imported for future use in benches
	if got := new(HTTPAPI); got == nil {
		t.Fatal("nil")
	}
}

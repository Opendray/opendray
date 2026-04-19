// Package bridge — HTTP API namespace (M3 T12).
//
// HTTPAPI implements opendray.http.* over the bridge. Every hop (initial
// URL + every redirect target) passes through Gate.Check with
// Need{Cap:"http", Target: hopURL}. SSRF defence is layered:
//
//  1. Static URL check in MatchHTTPURL (RFC1918 / loopback / link-local
//     denied regardless of grants — already done by capabilities.go).
//  2. Dial-time check: a custom net.Dialer.Control closure re-inspects
//     the resolved IP just before connect(2) and rejects private addresses.
//     This defeats DNS rebinding — the static check saw the hostname; the
//     dial-time check sees the actual resolved IP.
//
// Body caps:
//
//   - Request body > MaxRequestBody            → EINVAL before dialing.
//   - Response body > MaxResponseBody (non-stream) → result.truncated=true.
//   - Response body > MaxResponseBody (stream)     → trailing
//     NewStreamChunkErr{"EINVAL","response body truncated"} + NewStreamEnd.
package bridge

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"
)

// ─────────────────────────────────────────────
// Public config
// ─────────────────────────────────────────────

// HTTPConfig wires the HTTPAPI dependencies. All fields are optional;
// NewHTTPAPI fills in sane defaults.
type HTTPConfig struct {
	Gate *Gate
	Log  *slog.Logger

	// MaxRequestBody caps the request body size in bytes. Default 4 MiB.
	MaxRequestBody int64

	// MaxResponseBody caps the response body read in bytes. Bytes beyond
	// the cap are dropped; the caller sees truncated=true (non-stream)
	// or a trailing error chunk (stream). Default 16 MiB.
	MaxResponseBody int64

	// MaxRedirects is the maximum number of 3xx hops followed. Default 5.
	MaxRedirects int

	// DialTimeout bounds the TCP connect phase. Default 10 s.
	DialTimeout time.Duration

	// TotalTimeout bounds the entire request lifecycle (connect + TLS +
	// send + receive). Default 60 s.
	TotalTimeout time.Duration

	// TLSMinVersion pins the minimum TLS protocol version negotiated
	// during handshake. Default tls.VersionTLS12.
	TLSMinVersion uint16

	// transport, if non-nil, replaces the default http.RoundTripper.
	// Used by tests to inject a faking transport with a fake dialer.
	// Not a public knob — production code never sets this.
	transport http.RoundTripper

	// gateCheckOverride, if non-nil, replaces the Gate.Check call used
	// on every hop. Test-only: httptest servers bind to 127.0.0.1 which
	// MatchHTTPURL unconditionally denies via isPrivateHost; tests that
	// want to exercise the positive path supply a pass-through check.
	// Production code never sets this.
	gateCheckOverride func(ctx context.Context, plugin, target string) error
}

// ─────────────────────────────────────────────
// HTTPAPI
// ─────────────────────────────────────────────

// HTTPAPI implements opendray.http.* over the bridge. Zero value is not
// valid — construct via NewHTTPAPI.
type HTTPAPI struct {
	gate *Gate
	log  *slog.Logger

	maxReqBody   int64
	maxRespBody  int64
	maxRedirects int
	dialTimeout  time.Duration
	totalTimeout time.Duration
	tlsMinVer    uint16

	// gateCheck is the per-hop capability check. Defaults to
	// Gate.Check(..., Need{Cap:"http", Target: target}); tests can
	// override via HTTPConfig.gateCheckOverride.
	gateCheck func(ctx context.Context, plugin, target string) error

	// transportFactory builds a per-request *http.Transport. Tests inject
	// a different factory to observe dial-time SSRF behaviour.
	transportFactory func(ctx context.Context, plugin string, checkGate func(ctx context.Context, plugin, target string) error) http.RoundTripper
}

// NewHTTPAPI constructs an HTTPAPI. Defaults are per spec.
func NewHTTPAPI(cfg HTTPConfig) *HTTPAPI {
	if cfg.Log == nil {
		cfg.Log = slog.Default()
	}
	if cfg.MaxRequestBody <= 0 {
		cfg.MaxRequestBody = 4 << 20 // 4 MiB
	}
	if cfg.MaxResponseBody <= 0 {
		cfg.MaxResponseBody = 16 << 20 // 16 MiB
	}
	if cfg.MaxRedirects <= 0 {
		cfg.MaxRedirects = 5
	}
	if cfg.DialTimeout <= 0 {
		cfg.DialTimeout = 10 * time.Second
	}
	if cfg.TotalTimeout <= 0 {
		cfg.TotalTimeout = 60 * time.Second
	}
	if cfg.TLSMinVersion == 0 {
		cfg.TLSMinVersion = tls.VersionTLS12
	}

	api := &HTTPAPI{
		gate:         cfg.Gate,
		log:          cfg.Log,
		maxReqBody:   cfg.MaxRequestBody,
		maxRespBody:  cfg.MaxResponseBody,
		maxRedirects: cfg.MaxRedirects,
		dialTimeout:  cfg.DialTimeout,
		totalTimeout: cfg.TotalTimeout,
		tlsMinVer:    cfg.TLSMinVersion,
	}
	// Production gate check.
	api.gateCheck = func(ctx context.Context, plugin, target string) error {
		return api.gate.Check(ctx, plugin, Need{Cap: "http", Target: target})
	}
	if cfg.gateCheckOverride != nil {
		api.gateCheck = cfg.gateCheckOverride
	}
	// Production transport factory: builds a fresh *http.Transport per
	// request so the Dialer.Control closure can capture the current
	// plugin name for dial-time SSRF checks.
	api.transportFactory = api.defaultTransportFactory
	if cfg.transport != nil {
		// Tests can override the whole transport (ignores dial-time gate
		// since the test-provided transport is responsible for its own
		// wiring).
		api.transportFactory = func(_ context.Context, _ string, _ func(context.Context, string, string) error) http.RoundTripper {
			return cfg.transport
		}
	}
	return api
}

// Dispatch implements gateway.Namespace.
func (a *HTTPAPI) Dispatch(ctx context.Context, plugin, method string, args json.RawMessage, envID string, conn *Conn) (any, error) {
	switch method {
	case "request":
		return a.handleRequest(ctx, plugin, args)
	case "stream":
		return a.handleStream(ctx, plugin, args, envID, conn)
	default:
		we := &WireError{Code: "EUNAVAIL", Message: fmt.Sprintf("http.%s: method not available", method)}
		return nil, fmt.Errorf("http %s: %w", method, we)
	}
}

// ─────────────────────────────────────────────
// Request parsing
// ─────────────────────────────────────────────

// httpReqWire is the on-wire shape of the [req] arg:
//
//	{url, method?, headers?, body?(base64), opts?{timeoutMs}}
type httpReqWire struct {
	URL     string            `json:"url"`
	Method  string            `json:"method,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"` // base64-encoded
	Opts    *httpReqOpts      `json:"opts,omitempty"`
}

type httpReqOpts struct {
	TimeoutMs int64 `json:"timeoutMs,omitempty"`
}

// parseRequestArgs decodes the [req] arg array into a validated struct.
// Returns a wire error with code EINVAL on malformed input.
func (a *HTTPAPI) parseRequestArgs(args json.RawMessage) (*httpReqWire, []byte, error) {
	var argList []json.RawMessage
	if err := json.Unmarshal(args, &argList); err != nil || len(argList) < 1 {
		we := &WireError{Code: "EINVAL", Message: "http: args must be [req]"}
		return nil, nil, fmt.Errorf("http: %w", we)
	}
	var req httpReqWire
	if err := json.Unmarshal(argList[0], &req); err != nil {
		we := &WireError{Code: "EINVAL", Message: "http: req must be an object"}
		return nil, nil, fmt.Errorf("http: %w", we)
	}
	if strings.TrimSpace(req.URL) == "" {
		we := &WireError{Code: "EINVAL", Message: "http: req.url is required"}
		return nil, nil, fmt.Errorf("http: %w", we)
	}
	if req.Method == "" {
		req.Method = http.MethodGet
	}

	// Decode base64 body if present.
	var body []byte
	if req.Body != "" {
		decoded, err := base64.StdEncoding.DecodeString(req.Body)
		if err != nil {
			we := &WireError{Code: "EINVAL", Message: "http: req.body must be base64-encoded"}
			return nil, nil, fmt.Errorf("http: %w", we)
		}
		body = decoded
	}

	// Request body cap — checked before anything touches the network.
	if int64(len(body)) > a.maxReqBody {
		we := &WireError{Code: "EINVAL", Message: fmt.Sprintf("http: request body exceeds %d bytes", a.maxReqBody)}
		return nil, nil, fmt.Errorf("http: %w", we)
	}
	return &req, body, nil
}

// ─────────────────────────────────────────────
// request (non-stream)
// ─────────────────────────────────────────────

// handleRequest implements http.request(req) → {status, headers, body(base64), truncated?}.
func (a *HTTPAPI) handleRequest(ctx context.Context, plugin string, args json.RawMessage) (any, error) {
	req, body, err := a.parseRequestArgs(args)
	if err != nil {
		return nil, err
	}

	// Gate check on the initial URL (before we even build a request).
	if gErr := a.gateCheck(ctx, plugin, req.URL); gErr != nil {
		return nil, gErr
	}

	resp, hopErr := a.do(ctx, plugin, req, body)
	if hopErr != nil {
		return nil, hopErr
	}
	defer resp.Body.Close()

	// Read body up to MaxResponseBody+1 so we can detect truncation.
	reader := io.LimitReader(resp.Body, a.maxRespBody+1)
	raw, err := io.ReadAll(reader)
	if err != nil {
		we := &WireError{Code: "EINTERNAL", Message: fmt.Sprintf("http: read response body: %v", err)}
		return nil, fmt.Errorf("http request: %w", we)
	}
	truncated := false
	if int64(len(raw)) > a.maxRespBody {
		raw = raw[:a.maxRespBody]
		truncated = true
	}

	out := map[string]any{
		"status":  resp.StatusCode,
		"headers": flattenHeaders(resp.Header),
		"body":    base64.StdEncoding.EncodeToString(raw),
	}
	if truncated {
		out["truncated"] = true
	}
	return out, nil
}

// ─────────────────────────────────────────────
// stream
// ─────────────────────────────────────────────

// handleStream implements http.stream(req) — returns {status, headers} and
// emits NewStreamChunk envelopes with base64-encoded body slices. On cap
// overflow emits a trailing NewStreamChunkErr + NewStreamEnd.
func (a *HTTPAPI) handleStream(ctx context.Context, plugin string, args json.RawMessage, envID string, conn *Conn) (any, error) {
	if envID == "" {
		we := &WireError{Code: "EINVAL", Message: "http.stream: envelope id is required for stream correlation"}
		return nil, fmt.Errorf("http stream: %w", we)
	}
	if conn == nil {
		we := &WireError{Code: "EUNAVAIL", Message: "http.stream: no bridge connection"}
		return nil, fmt.Errorf("http stream: %w", we)
	}

	req, body, err := a.parseRequestArgs(args)
	if err != nil {
		return nil, err
	}

	// Gate check on the initial URL.
	if gErr := a.gateCheck(ctx, plugin, req.URL); gErr != nil {
		return nil, gErr
	}

	resp, hopErr := a.do(ctx, plugin, req, body)
	if hopErr != nil {
		return nil, hopErr
	}

	// Head returned to caller synchronously.
	head := map[string]any{
		"status":  resp.StatusCode,
		"headers": flattenHeaders(resp.Header),
	}

	// Pump the body asynchronously. The returned result resolves the
	// original call; chunks use the same envID so the SDK's completer
	// and chunk handler share a correlation key (consistent with
	// EventsAPI.subscribe).
	go a.pumpResponseBody(resp, envID, conn)

	return head, nil
}

// pumpResponseBody reads resp.Body in chunks, emits NewStreamChunk for each
// slice, and closes with NewStreamEnd. If cumulative bytes exceed the
// response body cap it emits a trailing NewStreamChunkErr first.
func (a *HTTPAPI) pumpResponseBody(resp *http.Response, envID string, conn *Conn) {
	defer resp.Body.Close()

	const chunkSize = 32 * 1024 // 32 KiB per chunk — bridge WS frame budget is 1 MiB
	buf := make([]byte, chunkSize)
	var total int64

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			// Enforce cap: if this chunk would push us past the cap, trim it
			// and emit a terminal error chunk.
			if total+int64(n) > a.maxRespBody {
				allowed := a.maxRespBody - total
				if allowed > 0 {
					chunk := map[string]string{"body": base64.StdEncoding.EncodeToString(buf[:allowed])}
					if env, mErr := NewStreamChunk(envID, chunk); mErr == nil {
						_ = conn.WriteEnvelope(env)
					}
					total += allowed
				}
				errEnv := NewStreamChunkErr(envID, &WireError{Code: "EINVAL", Message: "response body truncated"})
				_ = conn.WriteEnvelope(errEnv)
				_ = conn.WriteEnvelope(NewStreamEnd(envID))
				return
			}

			chunk := map[string]string{"body": base64.StdEncoding.EncodeToString(buf[:n])}
			if env, mErr := NewStreamChunk(envID, chunk); mErr == nil {
				_ = conn.WriteEnvelope(env)
			}
			total += int64(n)
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				errEnv := NewStreamChunkErr(envID, &WireError{Code: "EINTERNAL", Message: fmt.Sprintf("http stream: read: %v", err)})
				_ = conn.WriteEnvelope(errEnv)
			}
			_ = conn.WriteEnvelope(NewStreamEnd(envID))
			return
		}
	}
}

// ─────────────────────────────────────────────
// do — shared GET/POST plumbing, redirect gate
// ─────────────────────────────────────────────

// do builds the HTTP request, gates every redirect hop, and returns the
// final *http.Response. It does NOT read the body — callers (request /
// stream) own body consumption.
//
// Errors returned from do are either:
//   - *PermError (capability denied on initial URL or redirect hop)
//   - wrapped *WireError (EINTERNAL / ETIMEOUT / EINVAL for transport /
//     timeout / SSRF-dial-block / DNS failure)
func (a *HTTPAPI) do(ctx context.Context, plugin string, req *httpReqWire, body []byte) (*http.Response, error) {
	// Per-request deadline. opts.timeoutMs overrides up to TotalTimeout.
	total := a.totalTimeout
	if req.Opts != nil && req.Opts.TimeoutMs > 0 {
		d := time.Duration(req.Opts.TimeoutMs) * time.Millisecond
		if d < total {
			total = d
		}
	}
	ctx, cancel := context.WithTimeout(ctx, total)
	// cancel is tied to the caller via context — do not call it here; the
	// response body reader needs the deadline to stay live. Register cancel
	// so the HTTP client's body-read path releases resources on error paths.
	// We transfer ownership of cancel to the caller via a goroutine that
	// watches the returned response.
	//
	// Simple approach: rely on context.WithTimeout's internal timer — the
	// cancel function is only needed to avoid a goroutine leak when the
	// request completes well before the deadline. We attach cancel via a
	// defer on the caller's control flow: the caller is expected to close
	// the body, which reaches this goroutine; we spawn a tiny watcher.
	go func() {
		<-ctx.Done()
		cancel()
	}()

	transport := a.transportFactory(ctx, plugin, a.gateCheck)

	// Custom CheckRedirect: runs gateCheck on each hop; denies break the chain.
	client := &http.Client{
		Transport: transport,
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			if len(via) >= a.maxRedirects {
				return fmt.Errorf("http: too many redirects (> %d)", a.maxRedirects)
			}
			target := r.URL.String()
			if gErr := a.gateCheck(ctx, plugin, target); gErr != nil {
				return gErr
			}
			return nil
		},
		Timeout: 0, // we use ctx deadlines; setting Client.Timeout would layer a second timer
	}

	// Build the request.
	var bodyReader io.Reader
	if len(body) > 0 {
		bodyReader = strings.NewReader(string(body))
	}
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.URL, bodyReader)
	if err != nil {
		we := &WireError{Code: "EINVAL", Message: fmt.Sprintf("http: invalid request: %v", err)}
		return nil, fmt.Errorf("http: %w", we)
	}
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		// Classify:
		//   - url.Error wrapping a *PermError → bubble the PermError (redirect gate)
		//   - context.DeadlineExceeded → ETIMEOUT
		//   - any other → EINTERNAL
		var permErr *PermError
		if errors.As(err, &permErr) {
			return nil, permErr
		}
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			we := &WireError{Code: "ETIMEOUT", Message: fmt.Sprintf("http: request timed out: %v", err)}
			return nil, fmt.Errorf("http: %w", we)
		}
		we := &WireError{Code: "EINTERNAL", Message: fmt.Sprintf("http: transport error: %v", err)}
		return nil, fmt.Errorf("http: %w", we)
	}
	return resp, nil
}

// ─────────────────────────────────────────────
// Dialer with per-hop SSRF defence
// ─────────────────────────────────────────────

// defaultTransportFactory builds a fresh *http.Transport whose Dialer.Control
// callback re-checks the resolved remote IP against isPrivateHost and
// re-runs the Gate on the hop URL. Called once per request so the closure
// can capture the current plugin/gate/ctx.
func (a *HTTPAPI) defaultTransportFactory(ctx context.Context, plugin string, checkGate func(context.Context, string, string) error) http.RoundTripper {
	dialer := &net.Dialer{
		Timeout:   a.dialTimeout,
		KeepAlive: 30 * time.Second,
		// Control runs AFTER DNS resolution but BEFORE connect(2). `address`
		// is the resolved "host:port" (host is an IP literal after DNS).
		// We parse it, check isPrivateHost, and abort if private.
		Control: func(network, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				// Shouldn't happen — Go always passes host:port here.
				return fmt.Errorf("http: invalid dial address %q: %w", address, err)
			}
			if isPrivateHost(host) {
				a.log.Warn("http: SSRF block — resolved IP is private",
					"plugin", plugin, "network", network, "address", address)
				return fmt.Errorf("http: resolved address %s is in a private range (SSRF block)", address)
			}
			return nil
		},
	}

	tlsConfig := &tls.Config{
		MinVersion: a.tlsMinVer,
	}

	tr := &http.Transport{
		DialContext:           dialer.DialContext,
		TLSClientConfig:       tlsConfig,
		TLSHandshakeTimeout:   a.dialTimeout,
		ResponseHeaderTimeout: a.totalTimeout,
		MaxIdleConns:          4,
		IdleConnTimeout:       30 * time.Second,
		DisableKeepAlives:     true, // per-request; plugins are low-rate
		ForceAttemptHTTP2:     false,
	}
	return tr
}

// ─────────────────────────────────────────────
// helpers
// ─────────────────────────────────────────────

// flattenHeaders turns http.Header (map[string][]string) into a flat
// map[string]string by joining multi-valued headers with ", " (RFC 7230
// §3.2.2). Preserves the last-wins semantics for Set-Cookie which we
// merge rather than preserve because we don't expose cookie state to
// plugins.
func flattenHeaders(h http.Header) map[string]string {
	out := make(map[string]string, len(h))
	for k, vs := range h {
		out[k] = strings.Join(vs, ", ")
	}
	return out
}

// urlHostOnly returns the hostname component of a URL literal. Used
// only for diagnostic messages — do NOT use for security decisions
// (those must go through url.Parse + isPrivateHost).
func urlHostOnly(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	return u.Host
}

// Compile-time assertion that urlHostOnly stays used even if a
// refactor drops the current caller.
var _ = urlHostOnly

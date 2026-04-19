package gateway

// T11 — Command invoke HTTP endpoint tests.
//
// All tests are unit-level — no DB, no real dispatcher. A fakeInvoker
// satisfies commandInvoker so the handler can be exercised in isolation via
// httptest. Tests follow the TDD Red→Green→Refactor cycle.
//
// Test cases:
//  1. TestCommandInvoke_HappyPath             — 200, body matches, args forwarded
//  2. TestCommandInvoke_EmptyBody             — empty body treated as {}, 200 (not 400)
//  3. TestCommandInvoke_MalformedJSON         — 400 EINVAL
//  4. TestCommandInvoke_CommandNotFound       — 404 ENOTFOUND
//  5. TestCommandInvoke_PermDenied            — 403 EPERM
//  6. TestCommandInvoke_RunKindNotImpl        — 501 ENOTIMPL
//  7. TestCommandInvoke_OtherError            — 500 EINTERNAL
//  8. TestCommandInvoke_NilInvoker            — 503 EINVOKER
//  9. TestCommandInvoke_URLParamsRoutedCorrectly — chi params resolved
// 10. TestCommandInvoke_ContentType           — always application/json; charset=utf-8

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// ─── Fake invoker ────────────────────────────────────────────────────────────

type fakeInvoker struct {
	ret        any
	err        error
	lastPlugin string
	lastCmd    string
	lastArgs   map[string]any
}

func (f *fakeInvoker) Invoke(ctx context.Context, plugin, cmd string, args map[string]any) (any, error) {
	f.lastPlugin = plugin
	f.lastCmd = cmd
	f.lastArgs = args
	return f.ret, f.err
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// newCommandInvokeServer returns a chi.Mux with the command invoke route
// wired to the provided Server. This allows chi URL params ({name}, {id})
// to resolve correctly.
func newCommandInvokeServer(s *Server) http.Handler {
	r := chi.NewMux()
	r.Post("/api/plugins/{name}/commands/{id}/invoke", s.commandInvoke)
	return r
}

// postInvoke issues a POST to the given handler at the given path with body.
func postInvoke(t *testing.T, h http.Handler, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	} else {
		bodyReader = strings.NewReader("")
	}
	req := httptest.NewRequest(http.MethodPost, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

// decodeResponse unmarshals the recorder body into a map for assertion.
func decodeResponse(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.NewDecoder(w.Body).Decode(&m); err != nil {
		t.Fatalf("failed to decode response body %q: %v", w.Body.String(), err)
	}
	return m
}

// ─── Tests ────────────────────────────────────────────────────────────────────

// 1. Happy path: invoker returns a result map; handler marshals it directly.
func TestCommandInvoke_HappyPath(t *testing.T) {
	fake := &fakeInvoker{
		ret: map[string]any{"kind": "notify", "message": "hi"},
	}
	s := &Server{cmdInvoker: fake}
	h := newCommandInvokeServer(s)

	w := postInvoke(t, h, "/api/plugins/myplugin/commands/mycommand/invoke",
		`{"args":{"x":1}}`)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	m := decodeResponse(t, w)
	if m["kind"] != "notify" {
		t.Errorf("expected kind=notify, got %v", m["kind"])
	}
	if m["message"] != "hi" {
		t.Errorf("expected message=hi, got %v", m["message"])
	}

	// Verify args were forwarded.
	if fake.lastArgs == nil {
		t.Fatal("expected lastArgs to be non-nil")
	}
	if v, ok := fake.lastArgs["x"]; !ok || v != float64(1) {
		t.Errorf("expected lastArgs[x]=1 (float64), got %v", fake.lastArgs["x"])
	}
}

// 2. Empty body: treated as {} — args nil is OK, handler must not 400.
func TestCommandInvoke_EmptyBody(t *testing.T) {
	fake := &fakeInvoker{ret: map[string]any{"kind": "notify"}}
	s := &Server{cmdInvoker: fake}
	h := newCommandInvokeServer(s)

	w := postInvoke(t, h, "/api/plugins/p/commands/c/invoke", "")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for empty body, got %d; body: %s", w.Code, w.Body.String())
	}
}

// 3. Malformed JSON body → 400 EINVAL.
func TestCommandInvoke_MalformedJSON(t *testing.T) {
	fake := &fakeInvoker{}
	s := &Server{cmdInvoker: fake}
	h := newCommandInvokeServer(s)

	w := postInvoke(t, h, "/api/plugins/p/commands/c/invoke", `{not json`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d; body: %s", w.Code, w.Body.String())
	}

	m := decodeResponse(t, w)
	if m["code"] != "EINVAL" {
		t.Errorf("expected code=EINVAL, got %v", m["code"])
	}
}

// 4. Invoker returns "command not found" → 404 ENOTFOUND.
func TestCommandInvoke_CommandNotFound(t *testing.T) {
	fake := &fakeInvoker{err: errors.New("command not found: foo")}
	s := &Server{cmdInvoker: fake}
	h := newCommandInvokeServer(s)

	w := postInvoke(t, h, "/api/plugins/p/commands/c/invoke", `{}`)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d; body: %s", w.Code, w.Body.String())
	}

	m := decodeResponse(t, w)
	if m["code"] != "ENOTFOUND" {
		t.Errorf("expected code=ENOTFOUND, got %v", m["code"])
	}
}

// 5. Invoker returns "EPERM denied" → 403 EPERM.
func TestCommandInvoke_PermDenied(t *testing.T) {
	fake := &fakeInvoker{err: errors.New("EPERM denied: exec not granted")}
	s := &Server{cmdInvoker: fake}
	h := newCommandInvokeServer(s)

	w := postInvoke(t, h, "/api/plugins/p/commands/c/invoke", `{}`)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d; body: %s", w.Code, w.Body.String())
	}

	m := decodeResponse(t, w)
	if m["code"] != "EPERM" {
		t.Errorf("expected code=EPERM, got %v", m["code"])
	}
}

// 6. Invoker returns "run kind requires M2" → 501 ENOTIMPL.
func TestCommandInvoke_RunKindNotImpl(t *testing.T) {
	fake := &fakeInvoker{err: errors.New("run kind requires M2: host")}
	s := &Server{cmdInvoker: fake}
	h := newCommandInvokeServer(s)

	w := postInvoke(t, h, "/api/plugins/p/commands/c/invoke", `{}`)

	if w.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d; body: %s", w.Code, w.Body.String())
	}

	m := decodeResponse(t, w)
	if m["code"] != "ENOTIMPL" {
		t.Errorf("expected code=ENOTIMPL, got %v", m["code"])
	}
}

// 7. Invoker returns an unclassified error → 500 EINTERNAL.
func TestCommandInvoke_OtherError(t *testing.T) {
	fake := &fakeInvoker{err: errors.New("boom")}
	s := &Server{cmdInvoker: fake}
	h := newCommandInvokeServer(s)

	w := postInvoke(t, h, "/api/plugins/p/commands/c/invoke", `{}`)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d; body: %s", w.Code, w.Body.String())
	}

	m := decodeResponse(t, w)
	if m["code"] != "EINTERNAL" {
		t.Errorf("expected code=EINTERNAL, got %v", m["code"])
	}
}

// 8. cmdInvoker is nil (not wired) → 503 EINVOKER.
func TestCommandInvoke_NilInvoker(t *testing.T) {
	s := &Server{cmdInvoker: nil}
	h := newCommandInvokeServer(s)

	w := postInvoke(t, h, "/api/plugins/p/commands/c/invoke", `{}`)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d; body: %s", w.Code, w.Body.String())
	}

	m := decodeResponse(t, w)
	if m["code"] != "EINVOKER" {
		t.Errorf("expected code=EINVOKER, got %v", m["code"])
	}
}

// 9. URL params routed correctly via chi: {name} and {id} resolve.
func TestCommandInvoke_URLParamsRoutedCorrectly(t *testing.T) {
	fake := &fakeInvoker{ret: map[string]any{"ok": true}}
	s := &Server{cmdInvoker: fake}
	h := newCommandInvokeServer(s)

	w := postInvoke(t, h, "/api/plugins/time-ninja/commands/time.start/invoke", `{}`)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	if fake.lastPlugin != "time-ninja" {
		t.Errorf("expected lastPlugin=time-ninja, got %q", fake.lastPlugin)
	}
	if fake.lastCmd != "time.start" {
		t.Errorf("expected lastCmd=time.start, got %q", fake.lastCmd)
	}
}

// 10. Content-Type is always application/json; charset=utf-8.
func TestCommandInvoke_ContentType(t *testing.T) {
	cases := []struct {
		name string
		fake *fakeInvoker
		body string
	}{
		{
			name: "success",
			fake: &fakeInvoker{ret: map[string]any{"kind": "notify"}},
			body: `{}`,
		},
		{
			name: "error ENOTFOUND",
			fake: &fakeInvoker{err: errors.New("command not found")},
			body: `{}`,
		},
		{
			name: "nil invoker",
			fake: nil, // will create Server with nil cmdInvoker
			body: `{}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var s *Server
			if tc.fake != nil {
				s = &Server{cmdInvoker: tc.fake}
			} else {
				s = &Server{cmdInvoker: nil}
			}
			h := newCommandInvokeServer(s)

			w := postInvoke(t, h, "/api/plugins/p/commands/c/invoke", tc.body)

			ct := w.Header().Get("Content-Type")
			if !strings.HasPrefix(ct, "application/json") {
				t.Errorf("[%s] expected Content-Type application/json..., got %q", tc.name, ct)
			}
			if !strings.Contains(ct, "charset=utf-8") {
				t.Errorf("[%s] expected charset=utf-8 in Content-Type, got %q", tc.name, ct)
			}
		})
	}
}

// ─── invokeError implementations for interface-path tests ────────────────────

// statusCodeErr is a test-only error type that implements the invokeError
// interface (defined in plugins_command.go). The test file is in package
// gateway (not gateway_test) so it can access the package-private interface.
type statusCodeErr struct {
	httpStatus int
	message    string
}

func (e *statusCodeErr) Error() string      { return e.message }
func (e *statusCodeErr) InvokeStatus() int  { return e.httpStatus }

// wrappedStatusCodeErr wraps a statusCodeErr to test the Unwrap path in
// asInvokeError.
type wrappedStatusCodeErr struct {
	cause *statusCodeErr
}

func (e *wrappedStatusCodeErr) Error() string { return "wrapped: " + e.cause.Error() }
func (e *wrappedStatusCodeErr) Unwrap() error { return e.cause }

// deadEndErr is a wrapped error whose Unwrap() returns nil — exercises the
// stop condition in asInvokeError when no invokeError is found in chain.
type deadEndErr struct{ msg string }

func (e *deadEndErr) Error() string  { return e.msg }
func (e *deadEndErr) Unwrap() error  { return nil }

// 11. InvokeStatus interface: error implementing InvokeStatus() int takes
// priority over message-prefix sniffing.
func TestCommandInvoke_InvokeStatusInterface(t *testing.T) {
	cases := []struct {
		name           string
		err            error
		wantStatus     int
		wantCode       string
	}{
		{
			name:       "404 via InvokeStatus",
			err:        &statusCodeErr{httpStatus: http.StatusNotFound, message: "boom no prefix"},
			wantStatus: http.StatusNotFound,
			wantCode:   "ENOTFOUND",
		},
		{
			name:       "403 via InvokeStatus",
			err:        &statusCodeErr{httpStatus: http.StatusForbidden, message: "boom no prefix"},
			wantStatus: http.StatusForbidden,
			wantCode:   "EPERM",
		},
		{
			name:       "501 via InvokeStatus",
			err:        &statusCodeErr{httpStatus: http.StatusNotImplemented, message: "boom no prefix"},
			wantStatus: http.StatusNotImplemented,
			wantCode:   "ENOTIMPL",
		},
		{
			name:       "other status via InvokeStatus falls back to EINTERNAL code",
			err:        &statusCodeErr{httpStatus: http.StatusConflict, message: "boom no prefix"},
			wantStatus: http.StatusConflict,
			wantCode:   "EINTERNAL",
		},
		{
			name:       "wrapped error satisfies InvokeStatus via Unwrap",
			err:        &wrappedStatusCodeErr{cause: &statusCodeErr{httpStatus: http.StatusNotFound, message: "inner"}},
			wantStatus: http.StatusNotFound,
			wantCode:   "ENOTFOUND",
		},
		{
			name:       "wrapped error with nil Unwrap stops chain and falls to message sniff",
			err:        &deadEndErr{msg: "command not found: xyz"},
			wantStatus: http.StatusNotFound,
			wantCode:   "ENOTFOUND",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeInvoker{err: tc.err}
			s := &Server{cmdInvoker: fake}
			h := newCommandInvokeServer(s)

			w := postInvoke(t, h, "/api/plugins/p/commands/c/invoke", `{}`)

			if w.Code != tc.wantStatus {
				t.Errorf("expected status %d, got %d; body: %s", tc.wantStatus, w.Code, w.Body.String())
			}

			m := decodeResponse(t, w)
			if m["code"] != tc.wantCode {
				t.Errorf("expected code=%s, got %v", tc.wantCode, m["code"])
			}
		})
	}
}

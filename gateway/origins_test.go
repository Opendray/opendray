package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOriginPolicy_AllowCORS(t *testing.T) {
	cases := []struct {
		name    string
		allowed []string
		origin  string
		want    string
	}{
		{"empty policy same-origin", nil, "https://foo.example", ""},
		{"empty policy empty origin", nil, "", ""},
		{"wildcard arbitrary origin", []string{"*"}, "https://foo.example", "*"},
		{"wildcard empty origin", []string{"*"}, "", ""}, // native client, no header
		{"exact match", []string{"https://foo.example"}, "https://foo.example", "https://foo.example"},
		{"trailing slash in allowlist", []string{"https://foo.example/"}, "https://foo.example", "https://foo.example"},
		{"trailing slash in origin", []string{"https://foo.example"}, "https://foo.example/", "https://foo.example/"},
		{"miss returns empty", []string{"https://foo.example"}, "https://bar.example", ""},
		{"blank entries ignored", []string{"", "  ", "https://foo.example"}, "https://foo.example", "https://foo.example"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := newOriginPolicy(tc.allowed)
			if got := p.allowCORS(tc.origin); got != tc.want {
				t.Fatalf("allowCORS(%q) = %q, want %q", tc.origin, got, tc.want)
			}
		})
	}
}

func TestOriginPolicy_AllowWS(t *testing.T) {
	cases := []struct {
		name    string
		allowed []string
		origin  string
		host    string
		want    bool
	}{
		{"no origin header allowed (native client)", nil, "", "server.example", true},
		{"empty policy same host passes", nil, "https://server.example", "server.example", true},
		{"empty policy cross-host blocked", nil, "https://evil.example", "server.example", false},
		{"wildcard allows cross-host", []string{"*"}, "https://evil.example", "server.example", true},
		{"allowlist exact match passes", []string{"https://good.example"}, "https://good.example", "server.example", true},
		{"allowlist miss falls back to host check (pass)", []string{"https://good.example"}, "https://server.example", "server.example", true},
		{"allowlist miss falls back to host check (fail)", []string{"https://good.example"}, "https://evil.example", "server.example", false},
		{"malformed origin blocked", nil, "://not-a-url", "server.example", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := newOriginPolicy(tc.allowed)
			r := httptest.NewRequest(http.MethodGet, "http://"+tc.host+"/", nil)
			r.Host = tc.host
			if tc.origin != "" {
				r.Header.Set("Origin", tc.origin)
			}
			if got := p.allowWS(r); got != tc.want {
				t.Fatalf("allowWS(origin=%q,host=%q) = %v, want %v", tc.origin, tc.host, got, tc.want)
			}
		})
	}
}

func TestOriginPolicy_CorsMiddleware_StampsHeaders(t *testing.T) {
	p := newOriginPolicy([]string{"https://good.example"})
	called := false
	h := p.corsMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	// Allowed cross-origin: headers stamped, handler runs.
	r := httptest.NewRequest(http.MethodGet, "http://server.example/", nil)
	r.Header.Set("Origin", "https://good.example")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://good.example" {
		t.Fatalf("allowed: ACAO header = %q, want %q", got, "https://good.example")
	}
	if !called {
		t.Fatal("handler was not invoked on allowed request")
	}

	// Disallowed cross-origin: no Allow-Origin header (browser blocks).
	called = false
	r = httptest.NewRequest(http.MethodGet, "http://server.example/", nil)
	r.Header.Set("Origin", "https://evil.example")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("disallowed: ACAO header = %q, want empty", got)
	}
	// The request still reaches the handler — the browser (not the server)
	// is responsible for blocking cross-origin responses. This keeps us
	// behaviourally compatible with Bearer-token APIs called from native
	// clients.
	if !called {
		t.Fatal("handler should still be invoked; browser enforces CORS, not the server")
	}
}

func TestOriginPolicy_CorsMiddleware_OPTIONSShortCircuit(t *testing.T) {
	p := newOriginPolicy(nil)
	called := false
	h := p.corsMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	r := httptest.NewRequest(http.MethodOptions, "http://server.example/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if called {
		t.Fatal("downstream handler should not run for OPTIONS preflight")
	}
	if w.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS status = %d, want 204", w.Code)
	}
}

func TestOriginPolicy_WildcardWins(t *testing.T) {
	p := newOriginPolicy([]string{"https://explicit.example", "*"})
	if !p.wildcard {
		t.Fatal("wildcard entry anywhere in the list should enable wildcard mode")
	}
	if got := p.allowCORS("https://arbitrary.example"); got != "*" {
		t.Fatalf("wildcard allowCORS = %q, want %q", got, "*")
	}
}

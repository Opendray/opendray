package memory

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/opendray/opendray-v2/internal/integration"
)

// The guards don't touch other Handlers fields, so a zero value is fine.
func guardNext() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
}

func reqWithPrincipal(p *integration.Principal) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/memory/list", nil)
	if p != nil {
		r = r.WithContext(integration.WithPrincipal(r.Context(), *p))
	}
	return r
}

func TestRequireScope(t *testing.T) {
	h := &Handlers{}
	mw := h.requireScope(ScopeMemoryRead)
	cases := []struct {
		name string
		p    *integration.Principal
		want int
	}{
		{"admin allowed", &integration.Principal{Kind: integration.KindAdmin}, http.StatusOK},
		{"integration with scope", &integration.Principal{Kind: integration.KindIntegration, Scopes: []string{ScopeMemoryRead}}, http.StatusOK},
		{"integration wrong scope", &integration.Principal{Kind: integration.KindIntegration, Scopes: []string{ScopeMemoryWrite}}, http.StatusForbidden},
		{"integration no scope", &integration.Principal{Kind: integration.KindIntegration}, http.StatusForbidden},
		{"no principal", nil, http.StatusUnauthorized},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			mw(guardNext()).ServeHTTP(rec, reqWithPrincipal(c.p))
			if rec.Code != c.want {
				t.Errorf("got %d, want %d", rec.Code, c.want)
			}
		})
	}
}

func TestGlobalWriteAllowed(t *testing.T) {
	cases := []struct {
		name  string
		scope Scope
		p     integration.Principal
		want  bool
	}{
		{"admin writes global", ScopeGlobal, integration.Principal{Kind: integration.KindAdmin}, true},
		{"integration writes global denied", ScopeGlobal, integration.Principal{Kind: integration.KindIntegration, Scopes: []string{ScopeMemoryWrite}}, false},
		{"unauthenticated writes global denied", ScopeGlobal, integration.Principal{}, false},
		{"integration writes project", ScopeProject, integration.Principal{Kind: integration.KindIntegration}, true},
		{"legacy session literal still non-global", legacyScopeSession, integration.Principal{Kind: integration.KindIntegration}, true},
		{"empty scope allowed (defaults to project)", Scope(""), integration.Principal{Kind: integration.KindIntegration}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := globalWriteAllowed(c.scope, c.p); got != c.want {
				t.Errorf("globalWriteAllowed(%q, %+v) = %v, want %v", c.scope, c.p, got, c.want)
			}
		})
	}
}

func TestRequireAdmin(t *testing.T) {
	h := &Handlers{}
	cases := []struct {
		name string
		p    *integration.Principal
		want int
	}{
		{"admin allowed", &integration.Principal{Kind: integration.KindAdmin}, http.StatusOK},
		// A scoped integration key must NOT reach destructive endpoints.
		{"integration rejected", &integration.Principal{Kind: integration.KindIntegration, Scopes: []string{ScopeMemoryRead, ScopeMemoryWrite}}, http.StatusForbidden},
		{"no principal", nil, http.StatusForbidden},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.requireAdmin(guardNext()).ServeHTTP(rec, reqWithPrincipal(c.p))
			if rec.Code != c.want {
				t.Errorf("got %d, want %d", rec.Code, c.want)
			}
		})
	}
}

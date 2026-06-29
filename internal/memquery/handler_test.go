package memquery

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/opendray/opendray-v2/internal/integration"
	"github.com/opendray/opendray-v2/internal/memory"
)

// TestRequireRead_Gate verifies the project-search read gate: it accepts an
// admin principal or an integration key carrying memory:read, mirroring the
// /memory/search bar — so the project_search MCP tool (driven by the
// scoped-key memory subprocess) is actually reachable, not admin-only 401.
func TestRequireRead_Gate(t *testing.T) {
	h := NewHandlers(nil, nil) // the gate never touches svc

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	gate := h.requireRead(next)

	cases := []struct {
		name  string
		princ *integration.Principal // nil → no principal in context
		want  int
	}{
		{"no principal", nil, http.StatusUnauthorized},
		{"admin", &integration.Principal{Kind: integration.KindAdmin}, http.StatusOK},
		{
			"integration with memory:read",
			&integration.Principal{Kind: integration.KindIntegration, Scopes: []string{memory.ScopeMemoryRead}},
			http.StatusOK,
		},
		{
			"integration without the scope",
			&integration.Principal{Kind: integration.KindIntegration, Scopes: []string{"session:read"}},
			http.StatusForbidden,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/project-search?cwd=/x&q=y", nil)
			if c.princ != nil {
				req = req.WithContext(integration.WithPrincipal(req.Context(), *c.princ))
			}
			rec := httptest.NewRecorder()
			gate.ServeHTTP(rec, req)
			if rec.Code != c.want {
				t.Errorf("status = %d, want %d", rec.Code, c.want)
			}
		})
	}
}

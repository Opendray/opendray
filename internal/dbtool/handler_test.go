package dbtool

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/opendray/opendray-v2/internal/integration"
)

// The scope matrix: which principal passes which gate. The middlewares
// run before any service code, so a nil Service is fine here.
func TestScopeGates(t *testing.T) {
	h := NewHandlers(nil, nil)
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	admin := integration.Principal{Kind: integration.KindAdmin, ID: "admin"}
	reader := integration.Principal{Kind: integration.KindIntegration, ID: "int_r", Scopes: []string{ScopeDBRead}}
	writer := integration.Principal{Kind: integration.KindIntegration, ID: "int_w", Scopes: []string{ScopeDBRead, ScopeDBWrite}}
	stranger := integration.Principal{Kind: integration.KindIntegration, ID: "int_x", Scopes: []string{"memory:read"}}

	tests := []struct {
		name string
		gate http.Handler
		p    *integration.Principal
		want int
	}{
		{"unauthenticated read gate", h.requireScope(ScopeDBRead)(ok), nil, http.StatusUnauthorized},
		{"admin read gate", h.requireScope(ScopeDBRead)(ok), &admin, http.StatusOK},
		{"admin write gate", h.requireScope(ScopeDBWrite)(ok), &admin, http.StatusOK},
		{"admin admin gate", h.requireAdmin(ok), &admin, http.StatusOK},
		{"reader read gate", h.requireScope(ScopeDBRead)(ok), &reader, http.StatusOK},
		{"reader write gate", h.requireScope(ScopeDBWrite)(ok), &reader, http.StatusForbidden},
		{"reader admin gate", h.requireAdmin(ok), &reader, http.StatusForbidden},
		{"writer write gate", h.requireScope(ScopeDBWrite)(ok), &writer, http.StatusOK},
		{"writer admin gate", h.requireAdmin(ok), &writer, http.StatusForbidden},
		{"unrelated scopes read gate", h.requireScope(ScopeDBRead)(ok), &stranger, http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/dbtool/connections", nil)
			if tt.p != nil {
				req = req.WithContext(integration.WithPrincipal(req.Context(), *tt.p))
			}
			rec := httptest.NewRecorder()
			tt.gate.ServeHTTP(rec, req)
			if rec.Code != tt.want {
				t.Fatalf("%s: status = %d, want %d", tt.name, rec.Code, tt.want)
			}
		})
	}
}

// requireConnCwd binds a non-admin caller to its own project. These
// cases resolve before the service is touched, so a nil svc is safe:
// admin bypasses, and a non-admin without a cwd param is rejected up
// front (project-enumeration prevention).
func TestRequireConnCwd(t *testing.T) {
	h := NewHandlers(nil, nil)

	tests := []struct {
		name    string
		p       *integration.Principal
		query   string
		allowed bool
	}{
		{
			"admin bypasses cwd binding",
			&integration.Principal{Kind: integration.KindAdmin},
			"", true,
		},
		{
			"non-admin without cwd rejected",
			&integration.Principal{Kind: integration.KindIntegration, Scopes: []string{ScopeDBRead}},
			"", false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/x"+tt.query, nil)
			if tt.p != nil {
				req = req.WithContext(integration.WithPrincipal(req.Context(), *tt.p))
			}
			rec := httptest.NewRecorder()
			got := h.requireConnCwd(rec, req)
			if got != tt.allowed {
				t.Fatalf("requireConnCwd = %v, want %v (status %d)", got, tt.allowed, rec.Code)
			}
		})
	}
}

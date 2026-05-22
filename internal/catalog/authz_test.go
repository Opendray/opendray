package catalog

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/opendray/opendray-v2/internal/integration"
)

func TestRequirePrivileged(t *testing.T) {
	h := &Handlers{log: slog.Default()}

	cases := []struct {
		name      string
		principal *integration.Principal
		scope     string
		wantOK    bool
		wantCode  int
	}{
		{"admin allowed", &integration.Principal{Kind: integration.KindAdmin, ID: "navid"}, scopeProvidersUpdate, true, 0},
		{"integration with scope", &integration.Principal{Kind: integration.KindIntegration, ID: "i1", Scopes: []string{scopeProvidersUpdate}}, scopeProvidersUpdate, true, 0},
		{"integration wrong scope", &integration.Principal{Kind: integration.KindIntegration, ID: "i2", Scopes: []string{"providers:write"}}, scopeProvidersUpdate, false, http.StatusForbidden},
		{"integration no scope", &integration.Principal{Kind: integration.KindIntegration, ID: "i3", Scopes: nil}, scopeProvidersUpdate, false, http.StatusForbidden},
		{"no principal", nil, scopeProvidersUpdate, false, http.StatusUnauthorized},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := context.Background()
			if c.principal != nil {
				ctx = integration.WithPrincipal(ctx, *c.principal)
			}
			r := httptest.NewRequest(http.MethodPost, "/x", nil).WithContext(ctx)
			w := httptest.NewRecorder()
			got := h.requirePrivileged(w, r, c.scope)
			if got != c.wantOK {
				t.Fatalf("requirePrivileged=%v want %v", got, c.wantOK)
			}
			if !c.wantOK && w.Code != c.wantCode {
				t.Errorf("status=%d want %d", w.Code, c.wantCode)
			}
		})
	}
}

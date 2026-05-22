package integration

import (
	"context"
	"net/http"
	"strings"

	"github.com/opendray/opendray-v2/internal/auth"
)

// Principal kinds carried in the request context.
const (
	KindAdmin       = "admin"
	KindIntegration = "integration"
)

type principalCtxKey struct{}

// Principal is the authenticated identity attached to a request.
// Admin principals carry no scopes (admin bypasses scope checks);
// integration principals carry the scopes granted at registration.
type Principal struct {
	Kind   string
	ID     string
	Scopes []string
}

// CurrentPrincipal extracts the request's principal. Returns ok=false
// if the route was not behind one of this package's middlewares.
func CurrentPrincipal(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalCtxKey{}).(Principal)
	return p, ok
}

// WithPrincipal attaches a principal to ctx. The middlewares use this on
// the request path; it's also the supported way for other packages and
// tests to set up a principal-bearing context.
func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalCtxKey{}, p)
}

// CombinedMiddleware accepts an admin bearer OR an integration API key
// on the same route. Used to wrap business endpoints (sessions /
// providers / channels / catalog / auth/me / auth/logout) so that
// either principal type works.
func CombinedMiddleware(adminSvc *auth.Service, intgrSvc *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tok := bearerFromRequest(r)
			if tok == "" {
				writeUnauth(w)
				return
			}

			// Admin first: in-memory map lookup, no bcrypt.
			if newCtx, ok := adminSvc.AttachContext(r.Context(), tok); ok {
				ctx := context.WithValue(newCtx, principalCtxKey{},
					Principal{Kind: KindAdmin, ID: auth.Username(newCtx)})
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Integration fall-back: bcrypt + per-row scan, cached.
			i, scopes, err := intgrSvc.Verify(r.Context(), tok)
			if err == nil {
				ctx := context.WithValue(r.Context(), principalCtxKey{},
					Principal{Kind: KindIntegration, ID: i.ID, Scopes: scopes})
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			writeUnauth(w)
		})
	}
}

// IntegrationOnlyMiddleware rejects admin tokens. Used by
// /integrations/_events so admins do not accidentally exhaust an
// integration's event quota.
func IntegrationOnlyMiddleware(intgrSvc *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tok := bearerFromRequest(r)
			if tok == "" {
				writeUnauth(w)
				return
			}
			i, scopes, err := intgrSvc.Verify(r.Context(), tok)
			if err != nil {
				writeUnauth(w)
				return
			}
			ctx := context.WithValue(r.Context(), principalCtxKey{},
				Principal{Kind: KindIntegration, ID: i.ID, Scopes: scopes})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func bearerFromRequest(r *http.Request) string {
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return r.URL.Query().Get("token")
}

func writeUnauth(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("WWW-Authenticate", `Bearer realm="opendray"`)
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
}

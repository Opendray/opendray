package checkpoint

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// TestMountCoexistsWithSessionSubtree proves the checkpoint routes can be
// mounted on the same router that already owns the /sessions/{id} subtree
// (the real session handler) without a chi registration panic, and that
// both the session-scoped and checkpoint-scoped routes still resolve. This
// guards the app wiring where session.Handlers and checkpoint.Handlers
// share one admin router.
func TestMountCoexistsWithSessionSubtree(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("router assembly panicked: %v", r)
		}
	}()

	r := chi.NewRouter()
	// Stand in for session.Handlers.Mount: owns /sessions/{id}.
	r.Route("/sessions", func(r chi.Router) {
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(299) })
			r.Post("/input", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(298) })
		})
	})
	// The real checkpoint handler on the same router (nil svc: routes must
	// still register; we only assert routing resolves to the handler).
	NewHandlers(NewService(nil, nil, "", nil), nil).Mount(r)

	cases := []struct {
		method, path string
		wantCode     int
	}{
		{http.MethodGet, "/sessions/abc", 299},                                        // session subtree still works
		{http.MethodPost, "/sessions/abc/input", 298},                                 // session subtree still works
		{http.MethodGet, "/sessions/abc/checkpoints", http.StatusInternalServerError}, // reaches checkpoint list (nil pool -> 500)
	}
	for _, c := range cases {
		req := httptest.NewRequest(c.method, c.path, nil)
		rw := httptest.NewRecorder()
		// The checkpoint list handler dereferences the DB pool; guard the
		// nil-pool panic so we're asserting *routing*, not handler internals.
		func() {
			defer func() { _ = recover() }()
			r.ServeHTTP(rw, req)
		}()
		// A 404 would mean the route didn't resolve at all — that's the
		// failure we care about here.
		if rw.Code == http.StatusNotFound {
			t.Errorf("%s %s resolved to 404 (route not registered)", c.method, c.path)
		}
	}
}

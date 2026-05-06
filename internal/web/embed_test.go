package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestHandler_Smoke covers the two operating modes: the dist tree was
// built (production) and the dist tree is empty (fresh checkout, dev
// mode). Both paths must return predictable, non-panicking responses.
func TestHandler_Smoke(t *testing.T) {
	h := Handler()

	if h == nil {
		t.Fatal("Handler() returned nil")
	}

	t.Run("root path responds without panic", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		h.ServeHTTP(rr, req)

		// In a fresh checkout (no pnpm build) the handler returns 503;
		// in a built repo it returns 200 with the SPA HTML. Either is
		// a successful "not panicked" smoke result.
		switch rr.Code {
		case http.StatusOK:
			ct := rr.Header().Get("Content-Type")
			if !strings.HasPrefix(ct, "text/html") {
				t.Errorf("expected text/html, got %q", ct)
			}
			if rr.Body.Len() == 0 {
				t.Error("empty body for root path with built dist")
			}
		case http.StatusServiceUnavailable:
			body := rr.Body.String()
			if !strings.Contains(body, "pnpm build") {
				t.Errorf("503 body should mention build instructions, got: %q", body)
			}
		default:
			t.Errorf("unexpected status %d", rr.Code)
		}
	})

	t.Run("unknown deep link falls back to index (or 503)", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/sessions/ses_doesnotexist", nil)
		h.ServeHTTP(rr, req)

		// SPA fallback should never 404 — the client router handles
		// unknown paths. Accept 200 (built) or 503 (unbuilt).
		if rr.Code != http.StatusOK && rr.Code != http.StatusServiceUnavailable {
			t.Errorf("SPA fallback returned %d, want 200 or 503", rr.Code)
		}
	})
}

// TestDistExists is a non-fatal probe: it just verifies the function
// is callable and returns a bool. The actual value depends on whether
// the dist tree was checked in / built before `go test` ran.
func TestDistExists(t *testing.T) {
	got := DistExists()
	t.Logf("DistExists() = %v (test environment: dist %s)", got,
		map[bool]string{true: "present", false: "missing"}[got])
}

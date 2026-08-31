package session

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// fakeGrokChecker is a minimal GrokAccountChecker for handler tests.
type fakeGrokChecker struct {
	known map[string]bool
}

func (f *fakeGrokChecker) CheckEnabled(_ context.Context, id string) error {
	if f.known[id] {
		return nil
	}
	return ErrNotFound
}

func newRouterWithGrokChecker(svc Service, c GrokAccountChecker) http.Handler {
	r := chi.NewRouter()
	NewHandlers(svc, nil, WithGrokAccountChecker(c)).Mount(r)
	return r
}

func TestSwitchGrokAccount_OK(t *testing.T) {
	svc := newFakeSvc()
	svc.sessions["s1"] = Session{ID: "s1", ProviderID: "grok", State: StateRunning}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/sessions/s1/grok-account",
		bytes.NewBufferString(`{"account_id":"grok_new"}`))
	newRouter(svc).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body)
	}
	var s Session
	if err := json.Unmarshal(rr.Body.Bytes(), &s); err != nil {
		t.Fatal(err)
	}
	if s.GrokAccountID != "grok_new" {
		t.Errorf("account_id=%q, want grok_new", s.GrokAccountID)
	}
}

func TestSwitchGrokAccount_NotGrok(t *testing.T) {
	svc := newFakeSvc()
	svc.sessions["s1"] = Session{ID: "s1", ProviderID: "shell", State: StateRunning}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/sessions/s1/grok-account",
		bytes.NewBufferString(`{"account_id":"grok_x"}`))
	newRouter(svc).ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d; want 400 for non-grok session", rr.Code)
	}
}

func TestSwitchGrokAccount_InvalidID(t *testing.T) {
	svc := newFakeSvc()
	svc.sessions["ses_live"] = Session{ID: "ses_live", ProviderID: "grok", State: StateRunning}
	checker := &fakeGrokChecker{known: map[string]bool{"grok_real": true}}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/sessions/ses_live/grok-account",
		bytes.NewBufferString(`{"account_id":"grok_does_not_exist"}`))
	newRouterWithGrokChecker(svc, checker).ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s; want 400", rr.Code, rr.Body)
	}
	// The rejected switch must not have stopped or mutated the session.
	if s := svc.sessions["ses_live"]; s.State != StateRunning {
		t.Errorf("session state changed despite rejected switch: %s", s.State)
	}
}

func TestSwitchGrokAccount_BadJSON(t *testing.T) {
	svc := newFakeSvc()
	svc.sessions["s1"] = Session{ID: "s1", ProviderID: "grok", State: StateRunning}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/sessions/s1/grok-account",
		bytes.NewBufferString(`not json`))
	newRouter(svc).ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d; want 400 for bad JSON", rr.Code)
	}
}

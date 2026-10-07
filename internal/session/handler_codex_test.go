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

type fakeCodexChecker struct{ usable map[string]bool }

func (f *fakeCodexChecker) CheckUsable(_ context.Context, id string) error {
	if f.usable[id] {
		return nil
	}
	return ErrNotFound
}

func newRouterWithCodexChecker(svc Service, c CodexAccountChecker) http.Handler {
	r := chi.NewRouter()
	NewHandlers(svc, nil, WithCodexAccountChecker(c)).Mount(r)
	return r
}

func TestSwitchCodexAccount_OKCarriesContext(t *testing.T) {
	svc := newFakeSvc()
	svc.sessions["s1"] = Session{ID: "s1", ProviderID: "codex", State: StateRunning}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/sessions/s1/codex-account",
		bytes.NewBufferString(`{"account_id":"cdx_new","carry_context":true}`))
	newRouterWithCodexChecker(svc, &fakeCodexChecker{usable: map[string]bool{"cdx_new": true}}).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body)
	}
	var s Session
	if err := json.Unmarshal(rr.Body.Bytes(), &s); err != nil {
		t.Fatal(err)
	}
	if s.CodexAccountID != "cdx_new" {
		t.Errorf("account_id=%q, want cdx_new", s.CodexAccountID)
	}
	if !svc.lastCarryContext {
		t.Error("carry_context should flow to the manager")
	}
}

func TestSwitchCodexAccount_NotCodex(t *testing.T) {
	svc := newFakeSvc()
	svc.sessions["s1"] = Session{ID: "s1", ProviderID: "claude", State: StateRunning}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/sessions/s1/codex-account",
		bytes.NewBufferString(`{"account_id":"cdx_new"}`))
	newRouter(svc).ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", rr.Code)
	}
}

func TestSwitchCodexAccount_RejectsLoggedOutBeforeStopping(t *testing.T) {
	svc := newFakeSvc()
	svc.sessions["s1"] = Session{ID: "s1", ProviderID: "codex", State: StateRunning, CodexAccountID: "cdx_old"}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/sessions/s1/codex-account",
		bytes.NewBufferString(`{"account_id":"cdx_loggedout"}`))
	newRouterWithCodexChecker(svc, &fakeCodexChecker{usable: map[string]bool{}}).ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", rr.Code)
	}
	if s := svc.sessions["s1"]; s.CodexAccountID != "cdx_old" || s.State != StateRunning {
		t.Errorf("rejected switch must not touch the session: %+v", s)
	}
}

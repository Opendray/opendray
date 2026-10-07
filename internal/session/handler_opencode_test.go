package session

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

type fakeOCChecker struct{ usable map[string]bool }

func (f fakeOCChecker) CheckUsable(_ context.Context, id string) error {
	if !f.usable[id] {
		return errors.New("opencode account has no usable credentials")
	}
	return nil
}

func newRouterWithOC(svc Service, c OpenCodeAccountChecker) http.Handler {
	r := chi.NewRouter()
	NewHandlers(svc, nil, WithOpenCodeAccountChecker(c)).Mount(r)
	return r
}

func TestSwitchOpenCodeAccount_OKCarriesContext(t *testing.T) {
	svc := newFakeSvc()
	svc.sessions["ses_oc"] = Session{ID: "ses_oc", ProviderID: "opencode", State: StateRunning}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/sessions/ses_oc/opencode-account",
		bytes.NewBufferString(`{"account_id":"oc_b","carry_context":true}`))
	newRouterWithOC(svc, fakeOCChecker{usable: map[string]bool{"oc_b": true}}).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body)
	}
	if got := svc.sessions["ses_oc"].OpenCodeAccountID; got != "oc_b" {
		t.Errorf("account = %q, want oc_b", got)
	}
	if !svc.lastCarryContext {
		t.Error("carry_context should reach the manager")
	}
}

func TestSwitchOpenCodeAccount_UnusableRejectedBeforeStop(t *testing.T) {
	svc := newFakeSvc()
	svc.sessions["ses_oc"] = Session{ID: "ses_oc", ProviderID: "opencode", State: StateRunning, OpenCodeAccountID: "oc_a"}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/sessions/ses_oc/opencode-account",
		bytes.NewBufferString(`{"account_id":"oc_nope"}`))
	newRouterWithOC(svc, fakeOCChecker{usable: map[string]bool{}}).ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", rr.Code)
	}
	if s := svc.sessions["ses_oc"]; s.OpenCodeAccountID != "oc_a" || s.State != StateRunning {
		t.Errorf("session mutated by rejected switch: %+v", s)
	}
}

func TestSwitchOpenCodeAccount_WrongProvider(t *testing.T) {
	svc := newFakeSvc()
	svc.sessions["ses_cl"] = Session{ID: "ses_cl", ProviderID: "claude", State: StateRunning}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/sessions/ses_cl/opencode-account",
		bytes.NewBufferString(`{"account_id":""}`))
	newRouterWithOC(svc, nil).ServeHTTP(rr, req)
	if rr.Code == http.StatusOK {
		t.Fatalf("switching a claude session via opencode-account must fail, got 200")
	}
}

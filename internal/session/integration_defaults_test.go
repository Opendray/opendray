package session

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/opendray/opendray-v2/internal/integration"
)

// fakeDefaults is a stand-in for the IntegrationDefaults resolver. It
// returns the configured spawn defaults for a single integration id; any
// other id yields err so the handler's "no defaults on error" path is
// exercised too.
type fakeDefaults struct {
	id       string
	provider string
	model    string
	account  string
	err      error
}

func (f *fakeDefaults) DefaultsFor(_ context.Context, id string) (string, string, string, error) {
	if f.err != nil {
		return "", "", "", f.err
	}
	if id != f.id {
		return "", "", "", errors.New("not found")
	}
	return f.provider, f.model, f.account, nil
}

// newRouterWithDefaults mounts the session handlers with both the
// account checker (so a default account passes validation) and the
// integration-defaults resolver.
func newRouterWithDefaults(svc Service, c ClaudeAccountChecker, d IntegrationDefaults) http.Handler {
	r := chi.NewRouter()
	NewHandlers(svc, nil,
		WithClaudeAccountChecker(c),
		WithIntegrationDefaults(d),
	).Mount(r)
	return r
}

// asIntegration wraps a request's context with an integration principal,
// mirroring what the auth middleware does in production.
func asIntegration(req *http.Request, id string) *http.Request {
	ctx := integration.WithPrincipal(req.Context(),
		integration.Principal{Kind: integration.KindIntegration, ID: id})
	return req.WithContext(ctx)
}

func TestCreate_IntegrationDefaultsFillEmptyFields(t *testing.T) {
	svc := newFakeSvc()
	checker := &fakeAcctChecker{known: map[string]bool{"cla_def": true}}
	defs := &fakeDefaults{
		id: "int_A", provider: "claude", model: "opus", account: "cla_def",
	}
	// Body omits provider/model/account entirely — only cwd is supplied.
	req := httptest.NewRequest(http.MethodPost, "/sessions",
		bytes.NewBufferString(`{"cwd":"/tmp"}`))
	req = asIntegration(req, "int_A")
	rr := httptest.NewRecorder()
	newRouterWithDefaults(svc, checker, defs).ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s; want 201", rr.Code, rr.Body)
	}
	var s Session
	if err := json.Unmarshal(rr.Body.Bytes(), &s); err != nil {
		t.Fatal(err)
	}
	if s.ProviderID != "claude" {
		t.Errorf("provider: got %q, want defaulted 'claude'", s.ProviderID)
	}
	if s.Model != "opus" {
		t.Errorf("model: got %q, want defaulted 'opus'", s.Model)
	}
	if s.ClaudeAccountID != "cla_def" {
		t.Errorf("account: got %q, want defaulted 'cla_def'", s.ClaudeAccountID)
	}
}

func TestCreate_RequestOverridesIntegrationDefaults(t *testing.T) {
	svc := newFakeSvc()
	checker := &fakeAcctChecker{known: map[string]bool{"cla_req": true, "cla_def": true}}
	defs := &fakeDefaults{
		id: "int_A", provider: "claude", model: "opus", account: "cla_def",
	}
	// Request supplies every field; defaults must NOT clobber them.
	body := `{"provider_id":"codex","model":"sonnet","claude_account_id":"cla_req","cwd":"/tmp"}`
	req := httptest.NewRequest(http.MethodPost, "/sessions", bytes.NewBufferString(body))
	req = asIntegration(req, "int_A")
	rr := httptest.NewRecorder()
	newRouterWithDefaults(svc, checker, defs).ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s; want 201", rr.Code, rr.Body)
	}
	var s Session
	if err := json.Unmarshal(rr.Body.Bytes(), &s); err != nil {
		t.Fatal(err)
	}
	if s.ProviderID != "codex" {
		t.Errorf("provider: got %q, want request value 'codex'", s.ProviderID)
	}
	if s.Model != "sonnet" {
		t.Errorf("model: got %q, want request value 'sonnet'", s.Model)
	}
	if s.ClaudeAccountID != "cla_req" {
		t.Errorf("account: got %q, want request value 'cla_req'", s.ClaudeAccountID)
	}
}

func TestCreate_DefaultModelNotInheritedAcrossProviders(t *testing.T) {
	// The integration default model is provider-specific (an antigravity
	// model name). When the request explicitly picks a DIFFERENT provider
	// but omits the model, opendray must NOT carry the mismatched model
	// over — `claude --model "Gemini 3.5 Flash (Medium)"` would fail to
	// spawn. The session falls back to claude's own default (empty model).
	svc := newFakeSvc()
	checker := &fakeAcctChecker{}
	defs := &fakeDefaults{
		id: "int_A", provider: "antigravity", model: "Gemini 3.5 Flash (Medium)",
	}
	body := `{"provider_id":"claude","cwd":"/tmp"}`
	req := httptest.NewRequest(http.MethodPost, "/sessions", bytes.NewBufferString(body))
	req = asIntegration(req, "int_A")
	rr := httptest.NewRecorder()
	newRouterWithDefaults(svc, checker, defs).ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s; want 201", rr.Code, rr.Body)
	}
	var s Session
	if err := json.Unmarshal(rr.Body.Bytes(), &s); err != nil {
		t.Fatal(err)
	}
	if s.ProviderID != "claude" {
		t.Errorf("provider: got %q, want request value 'claude'", s.ProviderID)
	}
	if s.Model != "" {
		t.Errorf("model: got %q, want empty (antigravity model must not cross to claude)", s.Model)
	}
}

func TestCreate_OperatorOriginIgnoresIntegrationDefaults(t *testing.T) {
	// An admin/operator principal (or no integration principal) must not
	// pick up any integration's defaults.
	svc := newFakeSvc()
	defs := &fakeDefaults{id: "int_A", provider: "claude", model: "opus"}
	req := httptest.NewRequest(http.MethodPost, "/sessions",
		bytes.NewBufferString(`{"provider_id":"shell","cwd":"/tmp"}`))
	req = req.WithContext(integration.WithPrincipal(req.Context(),
		integration.Principal{Kind: integration.KindAdmin, ID: "admin"}))
	rr := httptest.NewRecorder()
	newRouterWithDefaults(svc, &fakeAcctChecker{}, defs).ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s; want 201", rr.Code, rr.Body)
	}
	var s Session
	if err := json.Unmarshal(rr.Body.Bytes(), &s); err != nil {
		t.Fatal(err)
	}
	if s.ProviderID != "shell" {
		t.Errorf("provider: got %q, want request value 'shell'", s.ProviderID)
	}
	if s.Model != "" {
		t.Errorf("model: got %q, want empty (no integration default applied)", s.Model)
	}
}

func TestCreate_IntegrationDefaultsLookupErrorIsNonFatal(t *testing.T) {
	// A resolver error must not fail the spawn — the session is created
	// with whatever the request carried.
	svc := newFakeSvc()
	defs := &fakeDefaults{err: errors.New("db down")}
	req := httptest.NewRequest(http.MethodPost, "/sessions",
		bytes.NewBufferString(`{"provider_id":"shell","cwd":"/tmp"}`))
	req = asIntegration(req, "int_A")
	rr := httptest.NewRecorder()
	newRouterWithDefaults(svc, &fakeAcctChecker{}, defs).ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s; want 201", rr.Code, rr.Body)
	}
	var s Session
	if err := json.Unmarshal(rr.Body.Bytes(), &s); err != nil {
		t.Fatal(err)
	}
	if s.ProviderID != "shell" {
		t.Errorf("provider: got %q, want request value 'shell'", s.ProviderID)
	}
}

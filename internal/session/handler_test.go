package session

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

type fakeSvc struct {
	sessions  map[string]Session
	createErr error
	stopErr   error
	startErr  error
	switchErr error
	subCh     chan []byte
	// lastCarryContext records the carry_context flag the handler
	// forwarded on the most recent SwitchClaudeAccount call.
	lastCarryContext bool
}

func newFakeSvc() *fakeSvc { return &fakeSvc{sessions: map[string]Session{}} }

func (f *fakeSvc) Create(_ context.Context, req CreateRequest) (Session, error) {
	if f.createErr != nil {
		return Session{}, f.createErr
	}
	s := Session{
		ID: "ses_test", ProviderID: req.ProviderID, Cwd: req.Cwd,
		Args:            req.Args,
		State:           StateRunning,
		ClaudeAccountID: req.ClaudeAccountID, // surface the field so auto-assign tests can assert it
	}
	f.sessions[s.ID] = s
	return s, nil
}

func (f *fakeSvc) Get(_ context.Context, id string) (Session, error) {
	s, ok := f.sessions[id]
	if !ok {
		return Session{}, ErrNotFound
	}
	return s, nil
}

func (f *fakeSvc) List(_ context.Context) ([]Session, error) {
	out := make([]Session, 0, len(f.sessions))
	for _, s := range f.sessions {
		out = append(out, s)
	}
	return out, nil
}

func (f *fakeSvc) Remove(_ context.Context, id string) error {
	if f.stopErr != nil {
		return f.stopErr
	}
	if _, ok := f.sessions[id]; !ok {
		return ErrNotFound
	}
	delete(f.sessions, id)
	return nil
}

func (f *fakeSvc) Stop(_ context.Context, id string) error {
	if f.stopErr != nil {
		return f.stopErr
	}
	if _, ok := f.sessions[id]; !ok {
		return ErrNotFound
	}
	s := f.sessions[id]
	s.State = StateStopped
	f.sessions[id] = s
	return nil
}

func (f *fakeSvc) Start(_ context.Context, id string) (Session, error) {
	if f.startErr != nil {
		return Session{}, f.startErr
	}
	s, ok := f.sessions[id]
	if !ok {
		return Session{}, ErrNotFound
	}
	s.State = StateRunning
	f.sessions[id] = s
	return s, nil
}

func (f *fakeSvc) Input(_ context.Context, id string, _ []byte) error {
	if _, ok := f.sessions[id]; !ok {
		return ErrNotFound
	}
	return nil
}

func (f *fakeSvc) Resize(_ context.Context, id string, _, _ uint16) error {
	if _, ok := f.sessions[id]; !ok {
		return ErrNotFound
	}
	return nil
}

func (f *fakeSvc) Subscribe(_ context.Context, id string) (<-chan []byte, func(), error) {
	if _, ok := f.sessions[id]; !ok {
		return nil, nil, ErrNotFound
	}
	if f.subCh == nil {
		f.subCh = make(chan []byte)
	}
	return f.subCh, func() {}, nil
}

func (f *fakeSvc) SwitchClaudeAccount(_ context.Context, id, accountID string, carryContext bool) (Session, error) {
	f.lastCarryContext = carryContext
	if f.switchErr != nil {
		return Session{}, f.switchErr
	}
	s, ok := f.sessions[id]
	if !ok {
		return Session{}, ErrNotFound
	}
	if s.ProviderID != "claude" {
		return Session{}, ErrAccountSwitchUnsupported
	}
	s.ClaudeAccountID = accountID
	s.State = StateRunning
	f.sessions[id] = s
	return s, nil
}

func (f *fakeSvc) Buffer(_ context.Context, id string, since int64) (Replay, error) {
	if _, ok := f.sessions[id]; !ok {
		return Replay{}, ErrNotFound
	}
	full := []byte("buffered")
	written := int64(len(full))
	start := since
	if start < 0 {
		start = 0
	}
	if start >= written {
		return Replay{Start: start, Written: written}, nil
	}
	return Replay{Bytes: full[start:], Start: start, Written: written}, nil
}

func (f *fakeSvc) History(_ context.Context, id string, _ int) (HistoryResponse, error) {
	if _, ok := f.sessions[id]; !ok {
		return HistoryResponse{}, ErrNotFound
	}
	return HistoryResponse{Entries: []ProjectInput{}}, nil
}

func newRouter(svc Service) http.Handler {
	r := chi.NewRouter()
	NewHandlers(svc, nil).Mount(r)
	return r
}

// fakeAcctChecker is a stand-in for the cliacct.Service surface used
// by the handler's claude_account_id validation. The known map lets a
// test enumerate which ids the checker considers valid; disabled holds
// the ids that should fail with ErrDisabled-equivalent. Anything not
// in either map → ErrNotFound-equivalent.
type fakeAcctChecker struct {
	known        map[string]bool
	disabled     map[string]bool
	autoAssignTo string // if set, PickAutoAssign returns this id; empty otherwise
}

func (f *fakeAcctChecker) CheckClaudeAccountEnabled(_ context.Context, id string) error {
	if f.disabled[id] {
		return fmt.Errorf("account %q disabled", id)
	}
	if !f.known[id] {
		return fmt.Errorf("account %q not found", id)
	}
	return nil
}

func (f *fakeAcctChecker) PickAutoAssignClaudeAccount(_ context.Context) (string, error) {
	return f.autoAssignTo, nil
}

func newRouterWithChecker(svc Service, c ClaudeAccountChecker) http.Handler {
	r := chi.NewRouter()
	NewHandlers(svc, nil, WithClaudeAccountChecker(c)).Mount(r)
	return r
}

func TestCreate_Created(t *testing.T) {
	svc := newFakeSvc()
	body := bytes.NewBufferString(`{"provider_id":"shell","cwd":"/tmp"}`)
	req := httptest.NewRequest(http.MethodPost, "/sessions", body)
	rr := httptest.NewRecorder()
	newRouter(svc).ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body)
	}
	var s Session
	if err := json.Unmarshal(rr.Body.Bytes(), &s); err != nil {
		t.Fatal(err)
	}
	if s.State != StateRunning {
		t.Errorf("state=%s", s.State)
	}
}

func TestCreate_BadJSON(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/sessions",
		bytes.NewBufferString(`not json`))
	newRouter(newFakeSvc()).ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestCreate_InvalidClaudeAccountID(t *testing.T) {
	// Bogus id is rejected with 400 BEFORE Service.Create() is called.
	svc := newFakeSvc()
	checker := &fakeAcctChecker{known: map[string]bool{"cla_real": true}}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/sessions",
		bytes.NewBufferString(`{"provider_id":"claude","cwd":"/tmp","claude_account_id":"cla_bogus"}`))
	newRouterWithChecker(svc, checker).ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s; want 400", rr.Code, rr.Body)
	}
	if len(svc.sessions) != 0 {
		t.Errorf("session was created despite invalid account id: %d rows", len(svc.sessions))
	}
}

func TestCreate_DisabledClaudeAccountID(t *testing.T) {
	svc := newFakeSvc()
	checker := &fakeAcctChecker{
		known:    map[string]bool{"cla_off": true},
		disabled: map[string]bool{"cla_off": true},
	}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/sessions",
		bytes.NewBufferString(`{"provider_id":"claude","cwd":"/tmp","claude_account_id":"cla_off"}`))
	newRouterWithChecker(svc, checker).ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s; want 400", rr.Code, rr.Body)
	}
}

func TestCreate_EmptyClaudeAccountID_Skipped(t *testing.T) {
	// Empty id means "use the CLI's keychain default" — checker must
	// NOT be invoked. We pin this by setting an empty known map and a
	// fakeSvc with a happy create; an over-eager validator would 400.
	svc := newFakeSvc()
	checker := &fakeAcctChecker{}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/sessions",
		bytes.NewBufferString(`{"provider_id":"claude","cwd":"/tmp"}`))
	newRouterWithChecker(svc, checker).ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("empty claude_account_id should pass validation; status=%d body=%s",
			rr.Code, rr.Body)
	}
}

func TestCreate_AutoAssignPicksAccountForClaudeWithEmptyID(t *testing.T) {
	// When the caller omits claude_account_id on a Claude session and
	// the checker returns a pick, the handler MUST inject it into the
	// request so the spawned PTY uses that account.
	svc := newFakeSvc()
	checker := &fakeAcctChecker{
		known:        map[string]bool{"cla_picked": true},
		autoAssignTo: "cla_picked",
	}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/sessions",
		bytes.NewBufferString(`{"provider_id":"claude","cwd":"/tmp"}`))
	newRouterWithChecker(svc, checker).ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s; want 201", rr.Code, rr.Body)
	}
	var s Session
	if err := json.Unmarshal(rr.Body.Bytes(), &s); err != nil {
		t.Fatal(err)
	}
	// The fakeSvc.Create echoes back req.ClaudeAccountID — so this
	// pins the contract that auto-assign actually wrote into req
	// before svc.Create ran.
	if s.ClaudeAccountID != "cla_picked" {
		t.Errorf("expected handler to auto-assign 'cla_picked', got %q", s.ClaudeAccountID)
	}
}

func TestCreate_AutoAssignSkippedForNonClaudeProviders(t *testing.T) {
	// Auto-assign is Claude-specific (other providers don't even have
	// the column). A shell session must NOT have an auto-assigned
	// claude_account_id even when the checker would offer one.
	svc := newFakeSvc()
	checker := &fakeAcctChecker{autoAssignTo: "cla_wrong"}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/sessions",
		bytes.NewBufferString(`{"provider_id":"shell","cwd":"/tmp"}`))
	newRouterWithChecker(svc, checker).ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d", rr.Code)
	}
	var s Session
	if err := json.Unmarshal(rr.Body.Bytes(), &s); err != nil {
		t.Fatal(err)
	}
	if s.ClaudeAccountID != "" {
		t.Errorf("shell session should not have claude_account_id; got %q", s.ClaudeAccountID)
	}
}

func TestCreate_AutoAssignSkippedWhenCallerPinnedAnAccount(t *testing.T) {
	// Explicit user choice trumps auto-assign. The pinned id has to
	// pass validation; the checker's auto-assign hint is ignored.
	svc := newFakeSvc()
	checker := &fakeAcctChecker{
		known:        map[string]bool{"cla_pinned": true, "cla_picked": true},
		autoAssignTo: "cla_picked",
	}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/sessions",
		bytes.NewBufferString(`{"provider_id":"claude","cwd":"/tmp","claude_account_id":"cla_pinned"}`))
	newRouterWithChecker(svc, checker).ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body)
	}
	var s Session
	if err := json.Unmarshal(rr.Body.Bytes(), &s); err != nil {
		t.Fatal(err)
	}
	if s.ClaudeAccountID != "cla_pinned" {
		t.Errorf("operator's pinned id must win; got %q", s.ClaudeAccountID)
	}
}

func TestSwitchClaudeAccount_InvalidID(t *testing.T) {
	svc := newFakeSvc()
	svc.sessions["ses_live"] = Session{ID: "ses_live", ProviderID: "claude", State: StateRunning}
	checker := &fakeAcctChecker{known: map[string]bool{"cla_real": true}}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/sessions/ses_live/claude-account",
		bytes.NewBufferString(`{"account_id":"cla_does_not_exist"}`))
	newRouterWithChecker(svc, checker).ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s; want 400", rr.Code, rr.Body)
	}
	// Session must NOT have been stopped or mutated by the rejected switch.
	if s := svc.sessions["ses_live"]; s.State != StateRunning {
		t.Errorf("session state changed despite rejected switch: %s", s.State)
	}
}

func TestCreate_UnknownProvider(t *testing.T) {
	svc := newFakeSvc()
	svc.createErr = fmt.Errorf("%w: foo", ErrUnknownProvider)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/sessions",
		bytes.NewBufferString(`{"provider_id":"foo","cwd":"/tmp"}`))
	newRouter(svc).ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestList_EmptyArray(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/sessions", nil)
	newRouter(newFakeSvc()).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d", rr.Code)
	}
	var resp struct {
		Sessions []Session `json:"sessions"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Sessions) != 0 {
		t.Errorf("sessions=%d", len(resp.Sessions))
	}
}

func TestGet_NotFound(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/sessions/missing", nil)
	newRouter(newFakeSvc()).ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestTerminate_NoContent(t *testing.T) {
	svc := newFakeSvc()
	svc.sessions["s1"] = Session{ID: "s1", State: StateRunning}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/sessions/s1", nil)
	newRouter(svc).ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestRemove_AlreadyTerminal(t *testing.T) {
	svc := newFakeSvc()
	svc.sessions["s1"] = Session{ID: "s1"}
	svc.stopErr = ErrAlreadyEnded
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/sessions/s1", nil)
	newRouter(svc).ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestStart_OK(t *testing.T) {
	svc := newFakeSvc()
	svc.sessions["s1"] = Session{ID: "s1", State: StateStopped}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/sessions/s1/start", nil)
	newRouter(svc).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestStop_OK(t *testing.T) {
	svc := newFakeSvc()
	svc.sessions["s1"] = Session{ID: "s1", State: StateRunning}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/sessions/s1/stop", nil)
	newRouter(svc).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestInput_NoContent(t *testing.T) {
	svc := newFakeSvc()
	svc.sessions["s1"] = Session{ID: "s1"}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/sessions/s1/input",
		bytes.NewBufferString(`{"data":"hi\n"}`))
	newRouter(svc).ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestResize_BadInput(t *testing.T) {
	svc := newFakeSvc()
	svc.sessions["s1"] = Session{ID: "s1"}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/sessions/s1/resize",
		bytes.NewBufferString(`{"cols":0,"rows":24}`))
	newRouter(svc).ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestBuffer_OctetStream(t *testing.T) {
	svc := newFakeSvc()
	svc.sessions["s1"] = Session{ID: "s1"}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/sessions/s1/buffer", nil)
	newRouter(svc).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d", rr.Code)
	}
	if got := rr.Header().Get("Content-Type"); got != "application/octet-stream" {
		t.Errorf("content-type=%s", got)
	}
	if !bytes.Equal(rr.Body.Bytes(), []byte("buffered")) {
		t.Errorf("body=%q", rr.Body.String())
	}
	if got := rr.Header().Get("X-OpenDray-Buffer-Cursor"); got != "8" {
		t.Errorf("cursor header=%q", got)
	}
}

func TestBuffer_SinceQuery(t *testing.T) {
	svc := newFakeSvc()
	svc.sessions["s1"] = Session{ID: "s1"}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/sessions/s1/buffer?since=4", nil)
	newRouter(svc).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d", rr.Code)
	}
	if !bytes.Equal(rr.Body.Bytes(), []byte("ered")) {
		t.Errorf("body=%q", rr.Body.String())
	}
	if got := rr.Header().Get("X-OpenDray-Buffer-Start"); got != "4" {
		t.Errorf("start header=%q", got)
	}
}

func TestBuffer_InvalidSince(t *testing.T) {
	svc := newFakeSvc()
	svc.sessions["s1"] = Session{ID: "s1"}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/sessions/s1/buffer?since=-3", nil)
	newRouter(svc).ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestSwitchClaudeAccount_OK(t *testing.T) {
	svc := newFakeSvc()
	svc.sessions["s1"] = Session{ID: "s1", ProviderID: "claude", State: StateRunning}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/sessions/s1/claude-account",
		bytes.NewBufferString(`{"account_id":"cla_new"}`))
	newRouter(svc).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body)
	}
	var s Session
	if err := json.Unmarshal(rr.Body.Bytes(), &s); err != nil {
		t.Fatal(err)
	}
	if s.ClaudeAccountID != "cla_new" {
		t.Errorf("account_id=%q", s.ClaudeAccountID)
	}
}

func TestSwitchClaudeAccount_CarryContextFlows(t *testing.T) {
	t.Run("carry_context true is forwarded", func(t *testing.T) {
		svc := newFakeSvc()
		svc.sessions["s1"] = Session{ID: "s1", ProviderID: "claude", State: StateRunning}
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, "/sessions/s1/claude-account",
			bytes.NewBufferString(`{"account_id":"cla_new","carry_context":true}`))
		newRouter(svc).ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body)
		}
		if !svc.lastCarryContext {
			t.Error("expected carry_context=true forwarded to the service")
		}
	})

	t.Run("omitted carry_context defaults to false", func(t *testing.T) {
		svc := newFakeSvc()
		svc.sessions["s1"] = Session{ID: "s1", ProviderID: "claude", State: StateRunning}
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, "/sessions/s1/claude-account",
			bytes.NewBufferString(`{"account_id":"cla_new"}`))
		newRouter(svc).ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d", rr.Code)
		}
		if svc.lastCarryContext {
			t.Error("expected carry_context to default to false when omitted")
		}
	})
}

func TestSwitchClaudeAccount_ClearBinding(t *testing.T) {
	svc := newFakeSvc()
	svc.sessions["s1"] = Session{
		ID: "s1", ProviderID: "claude", ClaudeAccountID: "cla_old", State: StateRunning,
	}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/sessions/s1/claude-account",
		bytes.NewBufferString(`{"account_id":""}`))
	newRouter(svc).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestSwitchClaudeAccount_NotClaude(t *testing.T) {
	svc := newFakeSvc()
	svc.sessions["s1"] = Session{ID: "s1", ProviderID: "shell", State: StateRunning}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/sessions/s1/claude-account",
		bytes.NewBufferString(`{"account_id":"cla_x"}`))
	newRouter(svc).ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestSwitchClaudeAccount_NotFound(t *testing.T) {
	svc := newFakeSvc()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/sessions/missing/claude-account",
		bytes.NewBufferString(`{"account_id":"cla_x"}`))
	newRouter(svc).ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestSwitchClaudeAccount_BadJSON(t *testing.T) {
	svc := newFakeSvc()
	svc.sessions["s1"] = Session{ID: "s1", ProviderID: "claude"}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/sessions/s1/claude-account",
		bytes.NewBufferString(`not json`))
	newRouter(svc).ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}

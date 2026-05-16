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
}

func newFakeSvc() *fakeSvc { return &fakeSvc{sessions: map[string]Session{}} }

func (f *fakeSvc) Create(_ context.Context, req CreateRequest) (Session, error) {
	if f.createErr != nil {
		return Session{}, f.createErr
	}
	s := Session{
		ID: "ses_test", ProviderID: req.ProviderID, Cwd: req.Cwd,
		Args: req.Args, State: StateRunning,
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

func (f *fakeSvc) Resize(_ context.Context, id string, _ ClientKind, _, _ uint16) error {
	if _, ok := f.sessions[id]; !ok {
		return ErrNotFound
	}
	return nil
}

func (f *fakeSvc) Subscribe(_ context.Context, id string, _ ClientKind) (<-chan []byte, func(), error) {
	if _, ok := f.sessions[id]; !ok {
		return nil, nil, ErrNotFound
	}
	if f.subCh == nil {
		f.subCh = make(chan []byte)
	}
	return f.subCh, func() {}, nil
}

func (f *fakeSvc) SwitchClaudeAccount(_ context.Context, id, accountID string) (Session, error) {
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

package autoloop

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/opendray/opendray-v2/internal/integration"
)

type fakeSvc struct {
	created   CreateRequest
	createErr error
	getErr    error
	pauseErr  error
	loop      Loop
	runs      []Run
}

func (f *fakeSvc) Create(_ context.Context, req CreateRequest) (Loop, error) {
	f.created = req
	if f.createErr != nil {
		return Loop{}, f.createErr
	}
	return Loop{ID: "lp_1", SessionID: req.SessionID, Kind: req.Kind, Origin: req.Origin, Status: StatusRunning}, nil
}
func (f *fakeSvc) Get(context.Context, string) (Loop, error)   { return f.loop, f.getErr }
func (f *fakeSvc) List(context.Context) ([]Loop, error)        { return []Loop{f.loop}, nil }
func (f *fakeSvc) Runs(context.Context, string) ([]Run, error) { return f.runs, nil }
func (f *fakeSvc) Pause(context.Context, string) error         { return f.pauseErr }
func (f *fakeSvc) Resume(context.Context, string) error        { return f.pauseErr }
func (f *fakeSvc) Stop(context.Context, string) error          { return f.pauseErr }

func testRouter(svc LoopService, p *integration.Principal) http.Handler {
	r := chi.NewRouter()
	if p != nil {
		pr := *p
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				next.ServeHTTP(w, req.WithContext(integration.WithPrincipal(req.Context(), pr)))
			})
		})
	}
	NewHandlers(svc, nil).Mount(r)
	return r
}

func do(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestCreateOperatorOrigin(t *testing.T) {
	svc := &fakeSvc{}
	h := testRouter(svc, nil) // no principal → operator
	dl := time.Now().Add(time.Hour).Format(time.RFC3339)
	body := `{"session_id":"s1","kind":"goal","goal":"g","prompt":"p","deadline_at":"` + dl + `"}`
	rec := do(t, h, http.MethodPost, "/loops", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("code = %d, want 201 (%s)", rec.Code, rec.Body)
	}
	if svc.created.Origin != OriginOperator {
		t.Errorf("origin = %q, want operator", svc.created.Origin)
	}
	if svc.created.SessionID != "s1" || svc.created.Kind != KindGoal {
		t.Errorf("create req mismatch: %+v", svc.created)
	}
}

func TestCreateIntegrationOrigin(t *testing.T) {
	svc := &fakeSvc{}
	p := &integration.Principal{
		Kind:   integration.KindIntegration,
		ID:     "intg_42",
		Scopes: []string{ScopeLoopCreate},
	}
	h := testRouter(svc, p)
	dl := time.Now().Add(time.Hour).Format(time.RFC3339)
	body := `{"session_id":"s1","kind":"interval","prompt":"p","interval_seconds":30,"deadline_at":"` + dl + `"}`
	rec := do(t, h, http.MethodPost, "/loops", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("code = %d, want 201 (%s)", rec.Code, rec.Body)
	}
	if svc.created.Origin != OriginIntegration || svc.created.IntegrationID != "intg_42" {
		t.Errorf("origin/integration = %q/%q, want integration/intg_42", svc.created.Origin, svc.created.IntegrationID)
	}
}

func TestCreateBadJSON(t *testing.T) {
	rec := do(t, testRouter(&fakeSvc{}, nil), http.MethodPost, "/loops", "{not json")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code = %d, want 400", rec.Code)
	}
}

func TestCreateValidationErrorMapsTo400(t *testing.T) {
	svc := &fakeSvc{createErr: ErrNoDeadline}
	rec := do(t, testRouter(svc, nil), http.MethodPost, "/loops", `{"session_id":"s1","kind":"goal","prompt":"p"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code = %d, want 400", rec.Code)
	}
}

func TestGetNotFound(t *testing.T) {
	svc := &fakeSvc{getErr: ErrNotFound}
	rec := do(t, testRouter(svc, nil), http.MethodGet, "/loops/nope", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("code = %d, want 404", rec.Code)
	}
}

func TestListReturnsArray(t *testing.T) {
	svc := &fakeSvc{loop: Loop{ID: "lp_1"}}
	rec := do(t, testRouter(svc, nil), http.MethodGet, "/loops", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200", rec.Code)
	}
	var out []Loop
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil || len(out) != 1 {
		t.Fatalf("list body = %s (err %v)", rec.Body, err)
	}
}

func TestStopConflictWhenTerminal(t *testing.T) {
	svc := &fakeSvc{pauseErr: ErrNotRunnable}
	rec := do(t, testRouter(svc, nil), http.MethodPost, "/loops/lp_1/stop", "")
	if rec.Code != http.StatusConflict {
		t.Fatalf("code = %d, want 409", rec.Code)
	}
}

func TestPauseReturnsLoop(t *testing.T) {
	svc := &fakeSvc{loop: Loop{ID: "lp_1", Status: StatusPaused}}
	rec := do(t, testRouter(svc, nil), http.MethodPost, "/loops/lp_1/pause", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200 (%s)", rec.Code, rec.Body)
	}
	var l Loop
	if err := json.Unmarshal(rec.Body.Bytes(), &l); err != nil || l.Status != StatusPaused {
		t.Fatalf("pause body = %s (err %v)", rec.Body, err)
	}
}

func TestRunsEndpoint(t *testing.T) {
	svc := &fakeSvc{runs: []Run{{ID: 1, Iteration: 1, Prompt: "p"}}}
	rec := do(t, testRouter(svc, nil), http.MethodGet, "/loops/lp_1/runs", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200", rec.Code)
	}
	var rs []Run
	if err := json.Unmarshal(rec.Body.Bytes(), &rs); err != nil || len(rs) != 1 {
		t.Fatalf("runs body = %s (err %v)", rec.Body, err)
	}
}

func TestIntegrationForbiddenWithoutScope(t *testing.T) {
	// An integration principal missing the required scope is rejected; an
	// admin / no principal (operator UI, tests) is not.
	noScope := &integration.Principal{Kind: integration.KindIntegration, ID: "i1"}
	dl := time.Now().Add(time.Hour).Format(time.RFC3339)
	body := `{"session_id":"s1","kind":"goal","goal":"g","prompt":"p","deadline_at":"` + dl + `"}`

	rec := do(t, testRouter(&fakeSvc{}, noScope), http.MethodPost, "/loops", body)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("create without loop:create = %d, want 403", rec.Code)
	}
	rec = do(t, testRouter(&fakeSvc{}, noScope), http.MethodGet, "/loops", "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("list without loop:read = %d, want 403", rec.Code)
	}
	rec = do(t, testRouter(&fakeSvc{}, noScope), http.MethodPost, "/loops/lp_1/stop", "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("stop without loop:write = %d, want 403", rec.Code)
	}
}

func TestIntegrationAllowedWithScope(t *testing.T) {
	reader := &integration.Principal{
		Kind:   integration.KindIntegration,
		ID:     "i1",
		Scopes: []string{ScopeLoopRead},
	}
	svc := &fakeSvc{loop: Loop{ID: "lp_1"}}
	rec := do(t, testRouter(svc, reader), http.MethodGet, "/loops", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list with loop:read = %d, want 200", rec.Code)
	}
}

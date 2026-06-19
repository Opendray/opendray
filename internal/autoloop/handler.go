package autoloop

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/opendray/opendray-v2/internal/integration"
)

// LoopService is the engine surface the HTTP handlers need. *Engine satisfies
// it; defined here so handlers are testable without a real engine.
type LoopService interface {
	Create(ctx context.Context, req CreateRequest) (Loop, error)
	Get(ctx context.Context, id string) (Loop, error)
	List(ctx context.Context) ([]Loop, error)
	Runs(ctx context.Context, id string) ([]Run, error)
	Pause(ctx context.Context, id string) error
	Resume(ctx context.Context, id string) error
	Stop(ctx context.Context, id string) error
}

// Handlers serves the loop REST API.
type Handlers struct {
	svc LoopService
	log *slog.Logger
}

// NewHandlers wires the loop HTTP handlers.
func NewHandlers(svc LoopService, log *slog.Logger) *Handlers {
	if log == nil {
		log = slog.Default()
	}
	return &Handlers{svc: svc, log: log.With("component", "autoloop.http")}
}

// Mount adds the loop routes. Caller mounts under /api/v1 in the combined-auth
// group, so both an operator (admin bearer) and an integration (API key) can
// drive loops; origin is derived from the authenticated principal, never the
// request body.
func (h *Handlers) Mount(r chi.Router) {
	r.Route("/loops", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.create)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.get)
			r.Get("/runs", h.runs)
			r.Post("/pause", h.pause)
			r.Post("/resume", h.resume)
			r.Post("/stop", h.stop)
		})
	})
}

// createBody is the JSON shape for POST /loops. origin / integration_id are
// intentionally absent — they come from the principal.
type createBody struct {
	SessionID       string     `json:"session_id"`
	Kind            string     `json:"kind"`
	Goal            string     `json:"goal"`
	Prompt          string     `json:"prompt"`
	IntervalSeconds int        `json:"interval_seconds"`
	MaxIterations   int        `json:"max_iterations"`
	DeadlineAt      *time.Time `json:"deadline_at"`
	FailureCap      int        `json:"failure_cap"`
	JudgeTask       string     `json:"judge_task"`
}

func (h *Handlers) create(w http.ResponseWriter, r *http.Request) {
	var body createBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	req := CreateRequest{
		SessionID:       body.SessionID,
		Origin:          OriginOperator,
		Kind:            Kind(body.Kind),
		Goal:            body.Goal,
		Prompt:          body.Prompt,
		IntervalSeconds: body.IntervalSeconds,
		MaxIterations:   body.MaxIterations,
		DeadlineAt:      body.DeadlineAt,
		FailureCap:      body.FailureCap,
		JudgeTask:       body.JudgeTask,
	}
	// Provenance from the authenticated principal — never the body.
	if p, ok := integration.CurrentPrincipal(r.Context()); ok && p.Kind == integration.KindIntegration {
		req.Origin = OriginIntegration
		req.IntegrationID = p.ID
	}
	l, err := h.svc.Create(r.Context(), req)
	if err != nil {
		writeErr(w, statusForErr(err), err)
		return
	}
	writeJSON(w, http.StatusCreated, l)
}

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	loops, err := h.svc.List(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if loops == nil {
		loops = []Loop{}
	}
	writeJSON(w, http.StatusOK, loops)
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	l, err := h.svc.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, statusForErr(err), err)
		return
	}
	writeJSON(w, http.StatusOK, l)
}

func (h *Handlers) runs(w http.ResponseWriter, r *http.Request) {
	rs, err := h.svc.Runs(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, statusForErr(err), err)
		return
	}
	if rs == nil {
		rs = []Run{}
	}
	writeJSON(w, http.StatusOK, rs)
}

func (h *Handlers) pause(w http.ResponseWriter, r *http.Request)  { h.transition(w, r, h.svc.Pause) }
func (h *Handlers) resume(w http.ResponseWriter, r *http.Request) { h.transition(w, r, h.svc.Resume) }
func (h *Handlers) stop(w http.ResponseWriter, r *http.Request)   { h.transition(w, r, h.svc.Stop) }

func (h *Handlers) transition(w http.ResponseWriter, r *http.Request, fn func(context.Context, string) error) {
	id := chi.URLParam(r, "id")
	if err := fn(r.Context(), id); err != nil {
		writeErr(w, statusForErr(err), err)
		return
	}
	l, err := h.svc.Get(r.Context(), id)
	if err != nil {
		writeErr(w, statusForErr(err), err)
		return
	}
	writeJSON(w, http.StatusOK, l)
}

// statusForErr maps domain errors to HTTP codes.
func statusForErr(err error) int {
	switch {
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrNotRunnable):
		return http.StatusConflict
	case errors.Is(err, ErrEmptySession), errors.Is(err, ErrEmptyPrompt),
		errors.Is(err, ErrBadKind), errors.Is(err, ErrBadOrigin),
		errors.Is(err, ErrNoDeadline), errors.Is(err, ErrPastDeadline),
		errors.Is(err, ErrBadInterval):
		return http.StatusBadRequest
	case errors.Is(err, ErrClosed):
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, map[string]string{"error": err.Error()})
}

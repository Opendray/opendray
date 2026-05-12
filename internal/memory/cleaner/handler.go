package cleaner

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/opendray/opendray-v2/internal/memory"
)

// Handlers exposes cleaner over HTTP under /memory/cleanup/*.
// Mount under admin auth — these endpoints can delete memories,
// so they must not be reachable by integration tokens.
type Handlers struct {
	svc *Service
	log *slog.Logger
}

// NewHandlers wires the HTTP layer. svc may be nil when the
// subsystem is disabled at startup; in that case all routes
// return 503 so the UI can surface "cleaner is off".
func NewHandlers(svc *Service, log *slog.Logger) *Handlers {
	if log == nil {
		log = slog.Default()
	}
	return &Handlers{svc: svc, log: log.With("component", "memory.cleaner.http")}
}

// Mount registers routes on r. r should already have admin auth
// applied.
func (h *Handlers) Mount(r chi.Router) {
	r.Route("/memory/cleanup", func(r chi.Router) {
		r.Post("/run", h.run)
		r.Get("/decisions", h.list)
		r.Get("/decisions/{id}", h.get)
		r.Post("/decisions/{id}/approve", h.approve)
		r.Post("/decisions/{id}/reject", h.reject)
	})
}

// run kicks off a one-shot cleanup pass for one (scope, scope_key).
// Body shape:
//
//	{ "scope": "project", "scope_key": "/path/to/cwd" }
//
// Returns the RunResult including the run_id so the UI can
// immediately poll decisions filtered by it.
func (h *Handlers) run(w http.ResponseWriter, r *http.Request) {
	if !h.ensure(w) {
		return
	}
	var req struct {
		Scope    string `json:"scope"`
		ScopeKey string `json:"scope_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Scope == "" {
		writeError(w, http.StatusBadRequest, errors.New("scope is required"))
		return
	}
	out, err := h.svc.Run(r.Context(), memory.Scope(req.Scope), req.ScopeKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// list returns existing decisions. Query params:
//
//	?status=pending|approved|rejected|executed|expired   (optional)
//	?scope=project|global|session                        (optional)
//	?scope_key=…                                         (optional)
//	?n=50                                                (cap 200)
func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	if !h.ensure(w) {
		return
	}
	q := r.URL.Query()
	limit := 50
	if v := q.Get("n"); v != "" {
		if x, err := strconv.Atoi(v); err == nil && x > 0 {
			limit = x
		}
	}
	rows, err := h.svc.List(r.Context(),
		q.Get("status"), q.Get("scope"), q.Get("scope_key"), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if rows == nil {
		rows = []Decision{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"decisions": rows})
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	if !h.ensure(w) {
		return
	}
	id := chi.URLParam(r, "id")
	d, err := h.svc.Get(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

// approve marks the decision approved AND executes it. Returns the
// updated decision so the UI sees status=executed (or status=expired
// when the executor couldn't apply).
func (h *Handlers) approve(w http.ResponseWriter, r *http.Request) {
	if !h.ensure(w) {
		return
	}
	id := chi.URLParam(r, "id")
	if err := h.svc.Approve(r.Context(), id); err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			writeError(w, http.StatusNotFound, err)
		case errors.Is(err, ErrAlreadyClosed):
			writeError(w, http.StatusConflict, err)
		default:
			writeError(w, http.StatusInternalServerError, err)
		}
		return
	}
	d, err := h.svc.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (h *Handlers) reject(w http.ResponseWriter, r *http.Request) {
	if !h.ensure(w) {
		return
	}
	id := chi.URLParam(r, "id")
	if err := h.svc.Reject(r.Context(), id); err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			writeError(w, http.StatusNotFound, err)
		case errors.Is(err, ErrAlreadyClosed):
			writeError(w, http.StatusConflict, err)
		default:
			writeError(w, http.StatusInternalServerError, err)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) ensure(w http.ResponseWriter) bool {
	if h.svc == nil {
		writeError(w, http.StatusServiceUnavailable,
			errors.New("memory cleaner disabled (no summarizer configured)"))
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, code int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

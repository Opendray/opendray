package checkpoint

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Handlers exposes the admin-facing checkpoint API. Capture is a mutating
// action (snapshots the working tree to disk); list/get/diff are read-only.
type Handlers struct {
	svc *Service
	log *slog.Logger
}

func NewHandlers(svc *Service, log *slog.Logger) *Handlers {
	if log == nil {
		log = slog.Default()
	}
	return &Handlers{svc: svc, log: log.With("component", "checkpoint.http")}
}

// Mount wires the routes. Session-scoped list/capture live under the
// session id; per-checkpoint reads live under the checkpoint id.
func (h *Handlers) Mount(r chi.Router) {
	r.Route("/sessions/{id}/checkpoints", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.capture)
	})
	r.Route("/checkpoints/{cid}", func(r chi.Router) {
		r.Get("/", h.get)
		r.Get("/diff", h.diff)
		r.Post("/restore", h.restore)
		r.Delete("/", h.delete)
	})
}

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	cps, err := h.svc.List(r.Context(), sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if cps == nil {
		cps = []Checkpoint{}
	}
	writeJSON(w, http.StatusOK, cps)
}

// captureRequest is the optional JSON body for POST .../checkpoints.
type captureRequest struct {
	Note string `json:"note,omitempty"`
}

func (h *Handlers) capture(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	var req captureRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req) // body is optional
	}
	cp, err := h.svc.CaptureManual(r.Context(), sessionID, req.Note)
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, errors.New("session not found or not live"))
		return
	case errors.Is(err, ErrNoStorageDir):
		writeError(w, http.StatusServiceUnavailable, errors.New("checkpoint storage not configured"))
		return
	case err != nil:
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, cp)
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	cp, err := h.svc.Get(r.Context(), chi.URLParam(r, "cid"))
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, cp)
}

// diff returns the raw uncommitted diff as text/plain (empty body when the
// working tree had no tracked changes).
func (h *Handlers) diff(w http.ResponseWriter, r *http.Request) {
	data, err := h.svc.ReadDiff(r.Context(), chi.URLParam(r, "cid"))
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// restore re-applies a checkpoint onto its cwd under the strict guards in
// Service.Restore. A guard failure is a 409 Conflict (the operator must
// resolve the working-tree state first), a bad/absent checkpoint is 404.
func (h *Handlers) restore(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.Restore(r.Context(), chi.URLParam(r, "cid"))
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, err)
		return
	case errors.Is(err, ErrNotGitCheckpoint),
		errors.Is(err, ErrNotWorkTree),
		errors.Is(err, ErrHeadMismatch),
		errors.Is(err, ErrDirtyWorktree),
		errors.Is(err, ErrApplyCheckFailed):
		writeError(w, http.StatusConflict, err)
		return
	case err != nil:
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *Handlers) delete(w http.ResponseWriter, r *http.Request) {
	err := h.svc.Delete(r.Context(), chi.URLParam(r, "cid"))
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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

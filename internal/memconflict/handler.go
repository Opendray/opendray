package memconflict

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// Handlers serves the conflict inbox over HTTP:
//
//	GET  /api/v1/memory/conflicts?cwd=&status=&n=        → {conflicts:[…]}
//	POST /api/v1/memory/conflicts/{id}/{action}           → {conflict}
//	POST /api/v1/memory/conflicts/detect?cwd=             → {detected: N}
//
// Mount under admin auth — conflicts surface raw plan/fact content.
type Handlers struct {
	svc *Service
	log *slog.Logger
}

func NewHandlers(svc *Service, log *slog.Logger) *Handlers {
	if log == nil {
		log = slog.Default()
	}
	return &Handlers{svc: svc, log: log.With("component", "memconflict.http")}
}

func (h *Handlers) Mount(r chi.Router) {
	r.Route("/memory/conflicts", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/detect", h.detect)
		r.Post("/{id}/{action}", h.decide)
	})
}

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	cwd := r.URL.Query().Get("cwd")
	if cwd == "" {
		writeError(w, http.StatusBadRequest, "cwd query param is required")
		return
	}
	status := r.URL.Query().Get("status")
	limit := 50
	if v := r.URL.Query().Get("n"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	conflicts, err := h.svc.List(r.Context(), cwd, status, limit)
	if err != nil {
		h.log.Warn("memconflict: list failed", "err", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if conflicts == nil {
		conflicts = []Conflict{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"conflicts": conflicts})
}

func (h *Handlers) decide(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	action := chi.URLParam(r, "action")
	if err := h.svc.Decide(r.Context(), id, action, "operator"); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handlers) detect(w http.ResponseWriter, r *http.Request) {
	cwd := r.URL.Query().Get("cwd")
	if cwd == "" {
		writeError(w, http.StatusBadRequest, "cwd query param is required")
		return
	}
	n, err := h.svc.DetectForCwd(r.Context(), cwd)
	if err != nil {
		h.log.Warn("memconflict: detect failed", "cwd", cwd, "err", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"detected": n})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": msg})
}

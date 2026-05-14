package memhealth

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Handlers exposes ComputeForCwd over HTTP.
//
// Routes (under the gateway's /api/v1 prefix):
//
//	GET /memory/health?cwd=<cwd>   → Snapshot
//
// Auth: mount behind admin auth — the snapshot reveals project
// activity volumes which can leak operator behaviour.
type Handlers struct {
	svc *Service
	log *slog.Logger
}

// NewHandlers wires an HTTP surface around an existing Service.
func NewHandlers(svc *Service, log *slog.Logger) *Handlers {
	if log == nil {
		log = slog.Default()
	}
	return &Handlers{svc: svc, log: log.With("component", "memhealth.http")}
}

// Mount registers routes on r. r should already have admin auth
// applied by the caller; the handlers themselves do not.
func (h *Handlers) Mount(r chi.Router) {
	r.Get("/memory/health", h.get)
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	cwd := r.URL.Query().Get("cwd")
	if cwd == "" {
		writeError(w, http.StatusBadRequest, "cwd query param is required")
		return
	}
	snap, err := h.svc.ComputeForCwd(r.Context(), cwd)
	if err != nil {
		h.log.Warn("memhealth: compute failed", "cwd", cwd, "err", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, snap)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": msg})
}

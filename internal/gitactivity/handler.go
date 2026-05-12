package gitactivity

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Handlers exposes manual run + status over HTTP under
// /git-activity/*. Admin-auth only (same shape as projectscan).
type Handlers struct {
	svc *Service
	log *slog.Logger
}

func NewHandlers(svc *Service, log *slog.Logger) *Handlers {
	if log == nil {
		log = slog.Default()
	}
	return &Handlers{svc: svc, log: log.With("component", "gitactivity.http")}
}

func (h *Handlers) Mount(r chi.Router) {
	r.Route("/git-activity", func(r chi.Router) {
		r.Post("/run", h.run)
	})
}

func (h *Handlers) run(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("git activity scanner disabled"))
		return
	}
	var req struct {
		Cwd string `json:"cwd"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Cwd == "" {
		writeError(w, http.StatusBadRequest, errors.New("cwd is required"))
		return
	}
	doc, err := h.svc.Run(r.Context(), req.Cwd)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, doc)
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

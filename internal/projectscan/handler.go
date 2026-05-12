package projectscan

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Handlers exposes manual scan triggers over HTTP. Spawn-time
// scans run automatically inside the catalog adapter; this surface
// exists so operators can force a refresh from mobile / web
// without spinning up a new session.
//
// Mounted under admin auth — a scan reads project files and writes
// project_docs, both privileged operations.
type Handlers struct {
	svc *Service
	log *slog.Logger
}

func NewHandlers(svc *Service, log *slog.Logger) *Handlers {
	if log == nil {
		log = slog.Default()
	}
	return &Handlers{svc: svc, log: log.With("component", "projectscan.http")}
}

// Mount registers routes on r. r should already have admin auth.
func (h *Handlers) Mount(r chi.Router) {
	r.Route("/project-scan", func(r chi.Router) {
		r.Post("/run", h.run)
	})
}

// run executes a one-shot scan for the cwd in the request body.
//
// Body: { "cwd": "/path/to/project" }
// Response: the resulting tech_stack project_doc row.
func (h *Handlers) run(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("project scanner disabled"))
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
	doc, err := h.svc.RunAndReturn(r.Context(), req.Cwd)
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

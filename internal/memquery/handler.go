package memquery

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// Handlers exposes cross-layer search over HTTP.
//
//	GET /api/v1/project-search?cwd=<cwd>&q=<query>&top_k=<N>
//
// Mount under admin auth — results expose every layer including
// goal/plan content which can include private project context.
type Handlers struct {
	svc *Service
	log *slog.Logger
}

func NewHandlers(svc *Service, log *slog.Logger) *Handlers {
	if log == nil {
		log = slog.Default()
	}
	return &Handlers{svc: svc, log: log.With("component", "memquery.http")}
}

func (h *Handlers) Mount(r chi.Router) {
	r.Get("/project-search", h.search)
}

func (h *Handlers) search(w http.ResponseWriter, r *http.Request) {
	cwd := r.URL.Query().Get("cwd")
	if cwd == "" {
		writeError(w, http.StatusBadRequest, "cwd query param is required")
		return
	}
	query := r.URL.Query().Get("q")
	if query == "" {
		writeError(w, http.StatusBadRequest, "q query param is required")
		return
	}
	topK := 10
	if v := r.URL.Query().Get("top_k"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			topK = n
		}
	}
	hits, err := h.svc.Search(r.Context(), SearchRequest{
		Cwd:   cwd,
		Query: query,
		TopK:  topK,
	})
	if err != nil {
		h.log.Warn("memquery: search failed", "cwd", cwd, "err", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"hits": hits})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": msg})
}

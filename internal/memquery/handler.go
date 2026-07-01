package memquery

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/opendray/opendray-v2/internal/integration"
	"github.com/opendray/opendray-v2/internal/memory"
)

// Handlers exposes cross-layer search over HTTP.
//
//	GET /api/v1/project-search?cwd=<cwd>&q=<query>&top_k=<N>
//
// Mount under the dual-auth group (admin OR integration key) and gate
// each route with the memory:read scope — the same bar as
// /memory/search. project_search is an MCP tool driven by the
// scoped-key `opendray mcp-memory` subprocess, so it must accept the
// same credential path the other memory tools do; mounting it admin-only
// made the advertised tool 401 for every agent. Results expose goal/plan
// content, but no more than doc_read (also dual-auth + memory:read), and
// scope_key already confines results to the caller's project.
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
	r.With(h.requireRead).Get("/project-search", h.search)
}

// requireRead admits an admin principal or an integration key carrying
// memory:read, matching memory.Handlers' read gate. Mirrored here (rather
// than reused) because the route lives in a different package; the scope
// constant is shared so the two surfaces stay in lockstep.
func (h *Handlers) requireRead(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, ok := integration.CurrentPrincipal(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if p.Kind == integration.KindAdmin || integration.HasScope(p.Scopes, memory.ScopeMemoryRead) {
			next.ServeHTTP(w, r)
			return
		}
		writeError(w, http.StatusForbidden, "requires admin or the memory:read scope")
	})
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

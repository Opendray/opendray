package knowledge

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// Handlers exposes the knowledge graph over HTTP. Mounted under the gateway's
// dual-auth group (admin OR integration token) so both operators and the
// auto-attached opendray-memory MCP can reach it once later phases wire the
// agent surface. Phase 0 ships CRUD only.
type Handlers struct {
	svc *Service
	log *slog.Logger
}

// NewHandlers wraps a Service for HTTP.
func NewHandlers(svc *Service, log *slog.Logger) *Handlers {
	if log == nil {
		log = slog.Default()
	}
	return &Handlers{svc: svc, log: log.With("component", "knowledge.http")}
}

// Mount registers the /knowledge routes.
func (h *Handlers) Mount(r chi.Router) {
	r.Route("/knowledge", func(r chi.Router) {
		r.Get("/nodes", h.listNodes)
		r.Post("/nodes", h.createNode)
		r.Get("/nodes/{id}", h.getNode)
		r.Delete("/nodes/{id}", h.deleteNode)
		r.Get("/nodes/{id}/edges", h.listEdges)
		r.Get("/nodes/{id}/graph", h.neighborhood)
		r.Post("/nodes/{id}/promote", h.promote)
		r.Post("/nodes/{id}/skillify", h.skillify)
		r.Post("/edges", h.createEdge)
		r.Get("/brain", h.projectBrain)
		r.Get("/search", h.search)
		r.Post("/reset", h.reset)
		r.Post("/kb/draft", h.draftKB)
	})
}

func (h *Handlers) listNodes(w http.ResponseWriter, r *http.Request) {
	f := NodeFilter{
		Kind:     NodeKind(r.URL.Query().Get("kind")),
		Scope:    Scope(r.URL.Query().Get("scope")),
		ScopeKey: r.URL.Query().Get("scope_key"),
	}
	nodes, err := h.svc.ListNodes(r.Context(), f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"nodes": nodes})
}

func (h *Handlers) createNode(w http.ResponseWriter, r *http.Request) {
	var n Node
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&n); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	created, err := h.svc.CreateNode(r.Context(), n)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, created)
}

func (h *Handlers) getNode(w http.ResponseWriter, r *http.Request) {
	n, err := h.svc.GetNode(r.Context(), chi.URLParam(r, "id"))
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, n)
}

func (h *Handlers) listEdges(w http.ResponseWriter, r *http.Request) {
	edges, err := h.svc.ListEdges(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"edges": edges})
}

func (h *Handlers) createEdge(w http.ResponseWriter, r *http.Request) {
	var e Edge
	if err := json.NewDecoder(io.LimitReader(r.Body, 64<<10)).Decode(&e); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := h.svc.CreateEdge(r.Context(), e); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) neighborhood(w http.ResponseWriter, r *http.Request) {
	center, neighbors, err := h.svc.Neighborhood(r.Context(), chi.URLParam(r, "id"))
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"node": center, "neighbors": neighbors})
}

func (h *Handlers) projectBrain(w http.ResponseWriter, r *http.Request) {
	cwd := r.URL.Query().Get("cwd")
	if cwd == "" {
		writeError(w, http.StatusBadRequest, errors.New("cwd is required"))
		return
	}
	view, err := h.svc.ProjectBrain(r.Context(), cwd)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (h *Handlers) promote(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Scope    string `json:"scope"`
		ScopeKey string `json:"scope_key"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 8<<10)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	scope := Scope(req.Scope)
	if !scope.Valid() {
		writeError(w, http.StatusBadRequest, errors.New("invalid scope (project|domain|global)"))
		return
	}
	if err := h.svc.PromoteNode(r.Context(), chi.URLParam(r, "id"), scope, req.ScopeKey); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) skillify(w http.ResponseWriter, r *http.Request) {
	skill, err := h.svc.Skillify(r.Context(), chi.URLParam(r, "id"))
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, skill)
}

func (h *Handlers) search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeError(w, http.StatusBadRequest, errors.New("q is required"))
		return
	}
	topK, _ := strconv.Atoi(r.URL.Query().Get("top_k"))
	hits, err := h.svc.SearchNodes(r.Context(), q, r.URL.Query().Get("cwd"), topK)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"hits": hits})
}

func (h *Handlers) deleteNode(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteNode(r.Context(), chi.URLParam(r, "id")); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) draftKB(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.DraftKB(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": res})
}

func (h *Handlers) reset(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Reset(r.Context()); err != nil {
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

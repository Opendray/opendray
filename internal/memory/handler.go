package memory

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// Handlers exposes the memory subsystem over HTTP under /memory/*.
// Mount under the dual-auth route group (admin OR integration) so
// the auto-attached opendray-memory MCP subprocess can reach it
// with an integration bearer rather than admin credentials.
//
// Endpoints (admin OR integration token):
//
//	POST   /memory/store        body: StoreRequest         → {id}
//	POST   /memory/search       body: SearchRequest        → {hits}
//	GET    /memory/list         ?scope=…&scope_key=…&n=…   → {memories}
//	DELETE /memory/{id}                                    → 204
//	GET    /memory/status                                  → {embedder, store, dim}
//	POST   /memory/test         body: {text}               → {dim, vector_preview}
type Handlers struct {
	svc *Service
	log *slog.Logger
}

// NewHandlers builds the HTTP wrapper around svc. svc may be nil
// when memory is disabled at startup; in that case all routes
// return 503 service-unavailable so the UI can surface that
// without a separate "is memory configured?" probe.
func NewHandlers(svc *Service, log *slog.Logger) *Handlers {
	if log == nil {
		log = slog.Default()
	}
	return &Handlers{svc: svc, log: log.With("component", "memory.http")}
}

// Mount registers all /memory/* routes on r. r should already have
// dual-auth middleware (admin OR integration) applied.
func (h *Handlers) Mount(r chi.Router) {
	r.Route("/memory", func(r chi.Router) {
		r.Get("/status", h.status)
		r.Post("/store", h.store)
		r.Post("/search", h.search)
		r.Get("/list", h.list)
		r.Delete("/{id}", h.delete)
		r.Post("/test", h.test)
	})
}

func (h *Handlers) ensure(w http.ResponseWriter) bool {
	if h.svc == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("memory subsystem disabled in this build"))
		return false
	}
	return true
}

func (h *Handlers) status(w http.ResponseWriter, r *http.Request) {
	if !h.ensure(w) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"embedder":   h.svc.EmbedderName(),
		"dimensions": h.svc.Dimensions(),
		"enabled":    true,
	})
}

func (h *Handlers) store(w http.ResponseWriter, r *http.Request) {
	if !h.ensure(w) {
		return
	}
	var req StoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	id, err := h.svc.Store(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (h *Handlers) search(w http.ResponseWriter, r *http.Request) {
	if !h.ensure(w) {
		return
	}
	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	hits, err := h.svc.Search(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if hits == nil {
		hits = []SearchHit{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"hits": hits})
}

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	if !h.ensure(w) {
		return
	}
	scope := Scope(r.URL.Query().Get("scope"))
	if scope == "" {
		scope = ScopeProject
	}
	scopeKey := r.URL.Query().Get("scope_key")
	limit := 100
	if v := r.URL.Query().Get("n"); v != "" {
		if x, err := strconv.Atoi(v); err == nil && x > 0 {
			limit = x
		}
	}
	out, err := h.svc.List(r.Context(), scope, scopeKey, limit)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if out == nil {
		out = []Memory{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"memories": out})
}

func (h *Handlers) delete(w http.ResponseWriter, r *http.Request) {
	if !h.ensure(w) {
		return
	}
	id := chi.URLParam(r, "id")
	if err := h.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// test exercises the embedder roundtrip without persisting. UI's
// "Test" button calls this to confirm the configured backend is
// alive + reports its dimensionality.
func (h *Handlers) test(w http.ResponseWriter, r *http.Request) {
	if !h.ensure(w) {
		return
	}
	var req struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Text == "" {
		req.Text = "opendray memory subsystem test"
	}
	vecs, err := h.svc.emb.Embed(r.Context(), []string{req.Text})
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	preview := vecs[0]
	if len(preview) > 8 {
		preview = preview[:8]
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"dim":            len(vecs[0]),
		"embedder":       h.svc.EmbedderName(),
		"vector_preview": preview,
	})
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

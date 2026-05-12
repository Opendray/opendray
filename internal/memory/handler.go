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
		r.Get("/scope-keys", h.scopeKeys)
		r.Get("/{id}", h.getOne)
		r.Patch("/{id}", h.update)
		r.Delete("/{id}", h.delete)
		r.Post("/delete-by-scope", h.deleteByScope)
		r.Post("/test", h.test)
		r.Post("/probe", h.probe)
		r.Get("/embedder-stats", h.embedderStats)
		r.Post("/reembed", h.reembed)
		r.Post("/mirror", h.mirror)
	})
}

// mirror runs an on-demand sync of Claude's local <cwd>/.claude/
// memory/*.md files into the pgvector store. Body shape:
//
//	{ "cwd": "/path/to/project" }
//
// Idempotent — files whose path+mtime were already ingested are
// skipped. Returns 503 when the mirror isn't wired (BM25-only
// builds or memory disabled).
func (h *Handlers) mirror(w http.ResponseWriter, r *http.Request) {
	if !h.ensure(w) {
		return
	}
	var req struct {
		Cwd string `json:"cwd"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	n, err := h.svc.MirrorCwd(r.Context(), req.Cwd)
	if err != nil {
		if errors.Is(err, ErrMirrorUnavailable) {
			writeError(w, http.StatusServiceUnavailable, err)
			return
		}
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ingested": n, "cwd": req.Cwd})
}

// embedderStats reports how many memories live under each embedder
// name (across every scope). The Settings → Memory → Migrate panel
// uses this to warn "you have N memories on bm25 that the current
// embedder will never match".
func (h *Handlers) embedderStats(w http.ResponseWriter, r *http.Request) {
	if !h.ensure(w) {
		return
	}
	stats, err := h.svc.EmbedderStats(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if stats.Counts == nil {
		stats.Counts = map[string]int{}
	}
	writeJSON(w, http.StatusOK, stats)
}

// reembed kicks off the migration: walk every memory whose embedder
// column doesn't match the current one, recompute its vector, and
// write it back. Synchronous (we expect kilos, not megas) — the
// HTTP request blocks until done. Body is empty; optional
// `?batch=NN` overrides the default batch size.
func (h *Handlers) reembed(w http.ResponseWriter, r *http.Request) {
	if !h.ensure(w) {
		return
	}
	batch := 32
	if v := r.URL.Query().Get("batch"); v != "" {
		if x, err := strconv.Atoi(v); err == nil && x > 0 {
			batch = x
		}
	}
	report, err := h.svc.Reembed(r.Context(), batch)
	if err != nil {
		// Even on error we return what we accomplished so the UI can
		// show partial progress; status code reflects the failure.
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":  err.Error(),
			"report": report,
		})
		return
	}
	writeJSON(w, http.StatusOK, report)
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
		"embedder":      h.svc.EmbedderName(),
		"dimensions":    h.svc.Dimensions(),
		"enabled":       true,
		"auto_detected": h.svc.AutoDetected(),
	})
}

// probe checks whether a given OpenAI-compatible base_url responds.
// The UI's "Test connection" button hits this. Body shape:
//
//	{ "base_url": "http://localhost:11434/v1", "api_key": "" }
//
// Always returns 200 with a ProbeResult — the result tells the UI
// whether the upstream is alive.
func (h *Handlers) probe(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BaseURL string `json:"base_url"`
		APIKey  string `json:"api_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	res := ProbeEndpoint(r.Context(), req.BaseURL, req.APIKey)
	writeJSON(w, http.StatusOK, res)
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
		// Gatekeeper rejections get their own status so MCP / UI
		// can distinguish "the model judged this noise" from
		// validation errors. 422 = Unprocessable Entity.
		if errors.Is(err, ErrNotDurable) {
			writeError(w, http.StatusUnprocessableEntity, err)
			return
		}
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

// scopeKeys returns distinct scope_key values stored under the given
// scope. Powers the UI's "Recent" picker so operators don't have to
// remember exact cwds.
func (h *Handlers) scopeKeys(w http.ResponseWriter, r *http.Request) {
	if !h.ensure(w) {
		return
	}
	scope := Scope(r.URL.Query().Get("scope"))
	if scope == "" {
		scope = ScopeProject
	}
	keys, err := h.svc.ListScopeKeys(r.Context(), scope)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if keys == nil {
		keys = []string{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"scope_keys": keys})
}

// update edits a memory in place. Body shape:
//
//	{ "text": "new content", "metadata": {...} }
//
// Re-embeds before persisting so the new vector matches the new text.
func (h *Handlers) update(w http.ResponseWriter, r *http.Request) {
	if !h.ensure(w) {
		return
	}
	id := chi.URLParam(r, "id")
	var body struct {
		Text     string                 `json:"text"`
		Metadata map[string]interface{} `json:"metadata,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := h.svc.Update(r.Context(), EditRequest{
		ID:       id,
		Text:     body.Text,
		Metadata: body.Metadata,
	}); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusBadRequest, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// deleteByScope wipes every memory under the given (scope, scope_key)
// in one SQL operation. Body shape:
//
//	{ "scope": "project", "scope_key": "/Users/you/projects/foo" }
//
// Returns {"deleted": N}. Refuses non-global scopes with empty
// scope_key — that would have wiped the whole table.
func (h *Handlers) deleteByScope(w http.ResponseWriter, r *http.Request) {
	if !h.ensure(w) {
		return
	}
	var body struct {
		Scope    Scope  `json:"scope"`
		ScopeKey string `json:"scope_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	n, err := h.svc.DeleteByScope(r.Context(), body.Scope, body.ScopeKey)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": n})
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

// getOne returns a single memory by id. Used by the new
// memory_get_provenance MCP tool which renders the row's
// source_kind / confidence / etc. for the agent.
func (h *Handlers) getOne(w http.ResponseWriter, r *http.Request) {
	if !h.ensure(w) {
		return
	}
	id := chi.URLParam(r, "id")
	mem, err := h.svc.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, mem)
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

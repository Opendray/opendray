package worker

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

// Handlers exposes per-task worker config + metrics over HTTP.
//
// Routes (under the gateway's /api/v1 prefix):
//
//	GET    /memory/workers              → {workers: [Config…]}
//	GET    /memory/workers/{task}       → Config
//	PUT    /memory/workers/{task}       → Config (after upsert)
//	POST   /memory/workers/{task}/test  → {ok, duration_ms, error?}
//	GET    /memory/workers/calls        → {calls: [CallSummary…]}
type Handlers struct {
	reg *Registry
	log *slog.Logger
}

func NewHandlers(reg *Registry, log *slog.Logger) *Handlers {
	if log == nil {
		log = slog.Default()
	}
	return &Handlers{reg: reg, log: log.With("component", "memory.worker.http")}
}

func (h *Handlers) Mount(r chi.Router) {
	r.Route("/memory/workers", func(r chi.Router) {
		r.Get("/", h.list)
		r.Get("/calls", h.listCalls)
		r.Get("/{task}", h.get)
		r.Put("/{task}", h.upsert)
		r.Post("/{task}/test", h.test)
	})
}

// list returns the config rows for all four tasks. Used by the
// settings UI to render the worker grid.
func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	configs, err := h.reg.Store().List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if configs == nil {
		configs = []Config{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"workers": configs})
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	task := TaskKind(chi.URLParam(r, "task"))
	cfg, err := h.reg.Store().Get(r.Context(), task)
	if err != nil {
		if errors.Is(err, ErrNoWorkerConfigured) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

func (h *Handlers) upsert(w http.ResponseWriter, r *http.Request) {
	task := TaskKind(chi.URLParam(r, "task"))
	var body struct {
		Kind         WorkerKind `json:"kind"`
		SummarizerID string     `json:"summarizer_id"`
		ProviderID   string     `json:"provider_id"`
		AccountID    string     `json:"account_id"`
		Enabled      *bool      `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	enabled := true
	if body.Enabled != nil {
		enabled = *body.Enabled
	}
	cfg := Config{
		Task:         task,
		Kind:         body.Kind,
		SummarizerID: body.SummarizerID,
		ProviderID:   body.ProviderID,
		AccountID:    body.AccountID,
		Enabled:      enabled,
	}
	if err := h.reg.Store().Upsert(r.Context(), cfg); err != nil {
		// Worker config validation errors surface as 400s so the
		// UI can show a meaningful message inline.
		if errors.Is(err, ErrAgentUnsupported) ||
			cfg.Valid() != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	// Re-fetch to return the canonical row (with server-set
	// updated_at).
	fresh, err := h.reg.Store().Get(r.Context(), task)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, fresh)
}

// test runs a minimal synthetic prompt through the worker to
// confirm it's reachable / authed. The UI's "Test connection"
// button calls this.
func (h *Handlers) test(w http.ResponseWriter, r *http.Request) {
	task := TaskKind(chi.URLParam(r, "task"))
	t0 := time.Now()
	resp, err := h.reg.Run(r.Context(), Request{
		Task:         task,
		SystemPrompt: "You are a connectivity test. Reply with the single word OK and nothing else.",
		UserInput:    "ping",
		MaxTokens:    16,
		Timeout:      60 * time.Second,
	})
	out := map[string]any{
		"task":        string(task),
		"ok":          err == nil,
		"duration_ms": time.Since(t0).Milliseconds(),
	}
	if err != nil {
		out["error"] = err.Error()
	} else {
		out["worker_kind"] = string(resp.WorkerKind)
		out["provider_id"] = resp.ProviderID
		out["preview"] = resp.Content
		if len(resp.Content) > 200 {
			out["preview"] = resp.Content[:200] + "…"
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handlers) listCalls(w http.ResponseWriter, r *http.Request) {
	task := TaskKind(r.URL.Query().Get("task")) // empty = all
	limit := 100
	if v := r.URL.Query().Get("n"); v != "" {
		if x, err := strconv.Atoi(v); err == nil && x > 0 {
			limit = x
		}
	}
	calls, err := h.reg.Store().ListCalls(r.Context(), task, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if calls == nil {
		calls = []CallSummary{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"calls": calls})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]any{"error": err.Error()})
}

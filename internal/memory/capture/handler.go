package capture

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Handlers wires capture admin endpoints (rule CRUD + run-now) onto chi.
type Handlers struct {
	store  *RuleStore
	engine *Engine
	log    *slog.Logger
}

func NewHandlers(store *RuleStore, engine *Engine, log *slog.Logger) *Handlers {
	if log == nil {
		log = slog.Default()
	}
	return &Handlers{store: store, engine: engine, log: log}
}

func (h *Handlers) Mount(r chi.Router) {
	r.Route("/memory-capture-rules", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.create)
		r.Get("/{id}", h.get)
		r.Patch("/{id}", h.update)
		r.Delete("/{id}", h.delete)
		r.Post("/{id}/run-now", h.runNow)
	})
}

// runNow forces an immediate fire of the rule, bypassing trigger
// evaluation + pause state. Useful for manual triggers and Phase
// C UI buttons. Synchronous: returns after every matching session
// has been processed (sessions usually < 5; provider call ~1-2s
// each).
func (h *Handlers) runNow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if h.engine == nil {
		writeError(w, http.StatusServiceUnavailable, fmt.Errorf("capture engine not wired"))
		return
	}
	count, err := h.engine.RunRuleNow(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrRuleNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"rule_id":          id,
		"sessions_invoked": count,
	})
}

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	rules, err := h.store.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if rules == nil {
		rules = []Rule{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"rules": rules})
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rule, err := h.store.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrRuleNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, rule)
}

func (h *Handlers) create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionID            string         `json:"session_id"`
		Name                 string         `json:"name"`
		Enabled              *bool          `json:"enabled,omitempty"`
		TriggerKind          string         `json:"trigger_kind"`
		TriggerConfig        map[string]any `json:"trigger_config"`
		SummarizerProviderID string         `json:"summarizer_provider_id"`
		DedupThreshold       float32        `json:"dedup_threshold"`
		TargetScope          string         `json:"target_scope"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid body: %w", err))
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	rule, err := h.store.Insert(r.Context(), Rule{
		SessionID:            req.SessionID,
		Name:                 req.Name,
		Enabled:              enabled,
		TriggerKind:          req.TriggerKind,
		TriggerConfig:        req.TriggerConfig,
		SummarizerProviderID: req.SummarizerProviderID,
		DedupThreshold:       req.DedupThreshold,
		TargetScope:          req.TargetScope,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, rule)
}

func (h *Handlers) update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Name                 *string        `json:"name,omitempty"`
		Enabled              *bool          `json:"enabled,omitempty"`
		TriggerKind          *string        `json:"trigger_kind,omitempty"`
		TriggerConfig        map[string]any `json:"trigger_config,omitempty"`
		SummarizerProviderID *string        `json:"summarizer_provider_id,omitempty"`
		DedupThreshold       *float32       `json:"dedup_threshold,omitempty"`
		TargetScope          *string        `json:"target_scope,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid body: %w", err))
		return
	}
	rule, err := h.store.Update(r.Context(), id, RulePatch{
		Name:                 req.Name,
		Enabled:              req.Enabled,
		TriggerKind:          req.TriggerKind,
		TriggerConfig:        req.TriggerConfig,
		SummarizerProviderID: req.SummarizerProviderID,
		DedupThreshold:       req.DedupThreshold,
		TargetScope:          req.TargetScope,
	})
	if err != nil {
		if errors.Is(err, ErrRuleNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, rule)
}

func (h *Handlers) delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.store.Delete(r.Context(), id); err != nil {
		if errors.Is(err, ErrRuleNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
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
	writeJSON(w, code, map[string]string{"error": err.Error()})
}

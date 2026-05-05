package injector

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Handlers struct {
	store *ProfileStore
	log   *slog.Logger
}

func NewHandlers(store *ProfileStore, log *slog.Logger) *Handlers {
	if log == nil {
		log = slog.Default()
	}
	return &Handlers{store: store, log: log}
}

func (h *Handlers) Mount(r chi.Router) {
	r.Route("/memory-injection-profiles", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.create)
		r.Get("/{id}", h.get)
		r.Patch("/{id}", h.update)
		r.Delete("/{id}", h.delete)
	})
}

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	profs, err := h.store.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if profs == nil {
		profs = []Profile{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"profiles": profs})
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p, err := h.store.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrProfileNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *Handlers) create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionID    string         `json:"session_id"`
		StrategyKind string         `json:"strategy_kind"`
		Config       map[string]any `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid body: %w", err))
		return
	}
	p, err := h.store.Insert(r.Context(), Profile{
		SessionID:    req.SessionID,
		StrategyKind: req.StrategyKind,
		Config:       req.Config,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (h *Handlers) update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		StrategyKind *string        `json:"strategy_kind,omitempty"`
		Config       map[string]any `json:"config,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid body: %w", err))
		return
	}
	p, err := h.store.Update(r.Context(), id, ProfilePatch{
		StrategyKind: req.StrategyKind,
		Config:       req.Config,
	})
	if err != nil {
		if errors.Is(err, ErrProfileNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *Handlers) delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.store.Delete(r.Context(), id); err != nil {
		if errors.Is(err, ErrProfileNotFound) {
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

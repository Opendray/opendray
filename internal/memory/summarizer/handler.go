package summarizer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

// Handlers wires summarizer admin endpoints onto the chi router.
// Caller mounts under the admin-only group. Sensitive fields
// (api_key) are write-only in/out: ProviderRow.APIKeyPlaintext is
// taken from request bodies on POST/PATCH and never returned in
// responses (we only echo the fingerprint).
type Handlers struct {
	registry *Registry
	store    *Store
	log      *slog.Logger
}

func NewHandlers(reg *Registry, store *Store, log *slog.Logger) *Handlers {
	if log == nil {
		log = slog.Default()
	}
	return &Handlers{registry: reg, store: store, log: log}
}

// Mount registers /memory-summarizer-providers under r.
func (h *Handlers) Mount(r chi.Router) {
	r.Route("/memory-summarizer-providers", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.create)
		r.Get("/{id}", h.get)
		r.Patch("/{id}", h.update)
		r.Delete("/{id}", h.delete)
		r.Post("/{id}/test", h.test)
		r.Get("/{id}/cost", h.cost)
	})
}

// providerView is what we send out — strips ciphertext + plaintext
// (only the fingerprint goes back to the UI).
type providerView struct {
	ID                string         `json:"id"`
	Name              string         `json:"name"`
	Kind              string         `json:"kind"`
	Model             string         `json:"model"`
	BaseURL           string         `json:"base_url,omitempty"`
	APIKeyFingerprint string         `json:"api_key_fingerprint,omitempty"`
	APIKeySet         bool           `json:"api_key_set"`
	ExtraConfig       map[string]any `json:"extra_config,omitempty"`
	Enabled           bool           `json:"enabled"`
	IsDefault         bool           `json:"is_default"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

func toView(r ProviderRow) providerView {
	return providerView{
		ID: r.ID, Name: r.Name, Kind: r.Kind, Model: r.Model,
		BaseURL: r.BaseURL, APIKeyFingerprint: r.APIKeyFingerprint,
		APIKeySet:   r.APIKeyCiphertext != "",
		ExtraConfig: r.ExtraConfig, Enabled: r.Enabled, IsDefault: r.IsDefault,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	rows, err := h.store.ListProviders(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	views := make([]providerView, 0, len(rows))
	for _, row := range rows {
		views = append(views, toView(row))
	}
	writeJSON(w, http.StatusOK, map[string]any{"providers": views})
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	row, err := h.store.GetProvider(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrProviderNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, toView(row))
}

func (h *Handlers) create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string         `json:"name"`
		Kind        string         `json:"kind"`
		Model       string         `json:"model"`
		BaseURL     string         `json:"base_url"`
		APIKey      string         `json:"api_key"`
		ExtraConfig map[string]any `json:"extra_config"`
		Enabled     *bool          `json:"enabled,omitempty"`
		IsDefault   bool           `json:"is_default"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid body: %w", err))
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	row, err := h.store.InsertProvider(r.Context(), ProviderRow{
		Name:            req.Name,
		Kind:            req.Kind,
		Model:           req.Model,
		BaseURL:         req.BaseURL,
		APIKeyPlaintext: req.APIKey,
		ExtraConfig:     req.ExtraConfig,
		Enabled:         enabled,
		IsDefault:       req.IsDefault,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrCipherRequired), errors.Is(err, ErrDuplicateName):
			writeError(w, http.StatusBadRequest, err)
		default:
			writeError(w, http.StatusBadRequest, err)
		}
		return
	}
	writeJSON(w, http.StatusCreated, toView(row))
}

func (h *Handlers) update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Name        *string        `json:"name,omitempty"`
		Model       *string        `json:"model,omitempty"`
		BaseURL     *string        `json:"base_url,omitempty"`
		APIKey      *string        `json:"api_key,omitempty"`
		ExtraConfig map[string]any `json:"extra_config,omitempty"`
		Enabled     *bool          `json:"enabled,omitempty"`
		IsDefault   *bool          `json:"is_default,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid body: %w", err))
		return
	}
	patch := ProviderPatch{
		Name:            req.Name,
		Model:           req.Model,
		BaseURL:         req.BaseURL,
		APIKeyPlaintext: req.APIKey,
		ExtraConfig:     req.ExtraConfig,
		Enabled:         req.Enabled,
		IsDefault:       req.IsDefault,
	}
	row, err := h.store.UpdateProvider(r.Context(), id, patch)
	if err != nil {
		if errors.Is(err, ErrProviderNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, toView(row))
}

func (h *Handlers) delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.store.DeleteProvider(r.Context(), id); err != nil {
		if errors.Is(err, ErrProviderNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) test(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	prov, err := h.registry.Build(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	tctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	if err := prov.Available(tctx); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handlers) cost(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	since := time.Time{} // all-time by default
	if v := r.URL.Query().Get("since"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			since = t
		} else if n, err2 := strconv.ParseInt(v, 10, 64); err2 == nil {
			since = time.Unix(n, 0)
		}
	}
	cs, err := h.store.ProviderCostSince(r.Context(), id, since)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, cs)
}

// ── helpers ──────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, map[string]string{"error": err.Error()})
}

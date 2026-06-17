package integration

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Handlers serves /api/v1/integrations admin REST.
type Handlers struct {
	svc *Service
	log *slog.Logger
}

func NewHandlers(svc *Service, log *slog.Logger) *Handlers {
	if log == nil {
		log = slog.Default()
	}
	return &Handlers{svc: svc, log: log.With("component", "integration.http")}
}

// MountAdmin mounts the admin-only CRUD endpoints. Caller wraps with
// admin middleware before calling.
//
// Uses direct paths instead of r.Route so the integration-only events
// route in a sibling chi.Group can also live under /integrations
// without panicking on duplicate Mount.
func (h *Handlers) MountAdmin(r chi.Router) {
	r.Get("/integrations", h.list)
	r.Post("/integrations", h.register)
	r.Get("/integrations/{id}", h.get)
	r.Patch("/integrations/{id}", h.update)
	r.Delete("/integrations/{id}", h.delete)
	r.Post("/integrations/{id}/rotate-key", h.rotateKey)
}

func (h *Handlers) register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// is_system is internal-only; the Register struct's `json:"-"`
	// tag already drops it from the wire, but be defensive — never
	// trust the client to bootstrap a system row.
	req.IsSystem = false
	res, err := h.svc.Register(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrPrefixTaken),
			errors.Is(err, ErrNameTaken),
			errors.Is(err, ErrReservedPrefix):
			writeError(w, http.StatusConflict, err)
		default:
			writeError(w, http.StatusBadRequest, err)
		}
		return
	}
	writeJSON(w, http.StatusCreated, res)
}

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if list == nil {
		list = []Integration{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"integrations": list})
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	i, err := h.svc.Get(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, i)
}

func (h *Handlers) update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		BaseURL                *string          `json:"base_url,omitempty"`
		Scopes                 *[]string        `json:"scopes,omitempty"`
		Version                *string          `json:"version,omitempty"`
		Enabled                *bool            `json:"enabled,omitempty"`
		MemoryPolicy           *MemoryPolicy    `json:"memory_policy,omitempty"`
		DefaultProviderID      *string          `json:"default_provider_id,omitempty"`
		DefaultModel           *string          `json:"default_model,omitempty"`
		DefaultClaudeAccountID *string          `json:"default_claude_account_id,omitempty"`
		MCPServers             *json.RawMessage `json:"mcp_servers,omitempty"`
		SystemPrompt           *string          `json:"system_prompt,omitempty"`
		PermissionMode         *PermissionMode  `json:"permission_mode,omitempty"`
		AgentID                *string          `json:"agent_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.MemoryPolicy != nil && !ValidMemoryPolicy(*req.MemoryPolicy) {
		writeError(w, http.StatusBadRequest,
			fmt.Errorf("memory_policy must be none|quarantine|full, got %q", *req.MemoryPolicy))
		return
	}
	if req.PermissionMode != nil && *req.PermissionMode != "" && !ValidPermissionMode(*req.PermissionMode) {
		writeError(w, http.StatusBadRequest,
			fmt.Errorf("permission_mode must be default|bypass, got %q", *req.PermissionMode))
		return
	}
	if req.MCPServers != nil {
		if err := validateMCPServers(*req.MCPServers); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
	}
	patch := UpdatePatch{
		BaseURL:                req.BaseURL,
		Scopes:                 req.Scopes,
		Version:                req.Version,
		Enabled:                req.Enabled,
		MemoryPolicy:           req.MemoryPolicy,
		DefaultProviderID:      req.DefaultProviderID,
		DefaultModel:           req.DefaultModel,
		DefaultClaudeAccountID: req.DefaultClaudeAccountID,
		MCPServers:             req.MCPServers,
		SystemPrompt:           req.SystemPrompt,
		PermissionMode:         req.PermissionMode,
		AgentID:                req.AgentID,
	}
	i, err := h.svc.Update(r.Context(), id, patch)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, i)
}

func (h *Handlers) delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.guardSystemMutation(r, id); err != nil {
		writeError(w, http.StatusForbidden, err)
		return
	}
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

func (h *Handlers) rotateKey(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.guardSystemMutation(r, id); err != nil {
		writeError(w, http.StatusForbidden, err)
		return
	}
	res, err := h.svc.RotateKey(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// guardSystemMutation rejects delete / rotate-key on rows opendray
// manages itself. Returns nil when the row is operator-owned (or
// missing — let the downstream handler emit the canonical 404).
func (h *Handlers) guardSystemMutation(r *http.Request, id string) error {
	i, err := h.svc.Get(r.Context(), id)
	if err != nil {
		return nil
	}
	if i.IsSystem {
		return ErrSystemIntegration
	}
	return nil
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

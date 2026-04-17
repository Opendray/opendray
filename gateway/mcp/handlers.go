package mcp

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/linivek/ntc/kernel/store"
)

// Handlers returns HTTP handlers for MCP server CRUD.
type Handlers struct {
	rt *Runtime
}

// NewHandlers wraps a Runtime with HTTP handlers.
func NewHandlers(rt *Runtime) *Handlers { return &Handlers{rt: rt} }

// List returns all MCP servers.
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	servers, err := h.rt.db.ListMCPServers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if servers == nil {
		servers = []store.MCPServer{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"servers": servers})
}

// Create inserts a new MCP server.
func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	var body store.MCPServer
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if !validTransport(body.Transport) {
		body.Transport = "stdio"
	}
	if body.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	created, err := h.rt.db.CreateMCPServer(r.Context(), body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// Get returns one MCP server.
func (h *Handlers) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	server, err := h.rt.db.GetMCPServer(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, server)
}

// Update replaces an MCP server.
func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body store.MCPServer
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if !validTransport(body.Transport) {
		body.Transport = "stdio"
	}
	if body.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	updated, err := h.rt.db.UpdateMCPServer(r.Context(), id, body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// Toggle flips the enabled flag.
func (h *Handlers) Toggle(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := h.rt.db.SetMCPServerEnabled(r.Context(), id, body.Enabled); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"enabled": body.Enabled})
}

// Delete removes an MCP server.
func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.rt.db.DeleteMCPServer(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Agents reports which agents we can inject into — the frontend uses
// this to populate the applies_to multi-select.
func (h *Handlers) Agents(w http.ResponseWriter, r *http.Request) {
	agents := make([]string, 0, len(renderers))
	for name := range renderers {
		agents = append(agents, name)
	}
	writeJSON(w, http.StatusOK, map[string]any{"agents": agents})
}

func validTransport(t string) bool {
	return t == "stdio" || t == "sse" || t == "http"
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

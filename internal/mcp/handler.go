package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
)

// Handlers expose the MCP registry + secrets file as a REST surface
// for the web Plugins page. Mirrors skills.Handlers — vault is full
// CRUD, no built-ins to disable.
type Handlers struct {
	loader      *Loader
	secretsPath string
	log         *slog.Logger
}

func NewHandlers(loader *Loader, secretsPath string, log *slog.Logger) *Handlers {
	if log == nil {
		log = slog.Default()
	}
	return &Handlers{
		loader:      loader,
		secretsPath: secretsPath,
		log:         log.With("component", "mcp.http"),
	}
}

func (h *Handlers) Mount(r chi.Router) {
	r.Route("/mcps", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.create)
		// Secrets endpoints sit under /mcps/_secrets so the prefix
		// stays single — slash-routes for everything MCP-related.
		// Underscore prefix avoids id collisions (we already reserve
		// ValidID to lowercase-alphanumeric/dash/underscore-in-middle).
		r.Get("/_secrets", h.secretsGet)
		r.Put("/_secrets/{key}", h.secretsSet)
		r.Delete("/_secrets/{key}", h.secretsDelete)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.get)
			r.Put("/", h.update)
			r.Delete("/", h.delete)
		})
	})
}

type writeReq struct {
	ID     string  `json:"id"`
	Server *Server `json:"server"`
}

func (h *Handlers) list(w http.ResponseWriter, _ *http.Request) {
	all, err := h.loader.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	// Strip SourcePath from the API response — it's a server-side
	// implementation detail.
	out := make([]Server, len(all))
	for i, s := range all {
		s.SourcePath = ""
		out[i] = s
	}
	writeJSON(w, http.StatusOK, map[string]any{"servers": out})
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !ValidID(id) {
		writeError(w, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	s, err := h.loader.Get(id)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	s.SourcePath = ""
	writeJSON(w, http.StatusOK, s)
}

func (h *Handlers) create(w http.ResponseWriter, r *http.Request) {
	var req writeReq
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	id := strings.TrimSpace(req.ID)
	if !ValidID(id) {
		writeError(w, http.StatusBadRequest,
			errors.New("id must be lowercase alphanumeric / dash / underscore"))
		return
	}
	if h.loader.VaultRoot() == "" {
		writeError(w, http.StatusConflict,
			errors.New("vault root is not configured; cannot create MCP servers"))
		return
	}
	if req.Server == nil {
		writeError(w, http.StatusBadRequest, errors.New("server payload is required"))
		return
	}
	dest := filepath.Join(h.loader.VaultRoot(), id)
	if _, err := os.Stat(dest); err == nil {
		writeError(w, http.StatusConflict,
			fmt.Errorf("MCP server %s already exists", id))
		return
	}
	if err := os.MkdirAll(dest, 0o700); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	srv := *req.Server
	srv.ID = id
	if strings.TrimSpace(srv.Name) == "" {
		srv.Name = id
	}
	if err := writeServer(dest, srv); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	loaded, err := h.loader.Get(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	loaded.SourcePath = ""
	writeJSON(w, http.StatusCreated, loaded)
}

func (h *Handlers) update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !ValidID(id) {
		writeError(w, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	if h.loader.VaultRoot() == "" {
		writeError(w, http.StatusConflict, errors.New("vault root not configured"))
		return
	}
	var req writeReq
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Server == nil {
		writeError(w, http.StatusBadRequest, errors.New("server payload is required"))
		return
	}
	dest := filepath.Join(h.loader.VaultRoot(), id)
	if err := os.MkdirAll(dest, 0o700); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	srv := *req.Server
	srv.ID = id
	if strings.TrimSpace(srv.Name) == "" {
		srv.Name = id
	}
	if err := writeServer(dest, srv); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	loaded, err := h.loader.Get(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	loaded.SourcePath = ""
	writeJSON(w, http.StatusOK, loaded)
}

func (h *Handlers) delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !ValidID(id) {
		writeError(w, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	if h.loader.VaultRoot() == "" {
		writeError(w, http.StatusConflict, errors.New("vault root not configured"))
		return
	}
	dest := filepath.Join(h.loader.VaultRoot(), id)
	info, err := os.Stat(dest)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeError(w, http.StatusNotFound,
				fmt.Errorf("MCP server %s not found", id))
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !info.IsDir() {
		writeError(w, http.StatusInternalServerError,
			errors.New("vault MCP path is not a directory"))
		return
	}
	if err := os.RemoveAll(dest); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// secretsState is the shape returned by GET /mcps/_secrets and echoed
// after a successful PUT. Values are NEVER included — only the list
// of key names is exposed to the API surface, mirroring how OS-level
// keychains behave.
type secretsState struct {
	Path      string   `json:"path"`
	Present   bool     `json:"present"`
	Encrypted bool     `json:"encrypted"`
	Keys      []string `json:"keys"`
}

// secretsGet returns the vault metadata + the loaded key names (no
// values). The `encrypted` field tells the UI whether the on-disk
// file is AES-GCM encrypted (key in OS keychain) or fell back to
// plaintext.
func (h *Handlers) secretsGet(w http.ResponseWriter, _ *http.Request) {
	state, err := h.loadState()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, state)
}

type secretsSetReq struct {
	Value string `json:"value"`
}

// secretsSet stores or updates a single key. URL form
// `PUT /mcps/_secrets/{key}` keeps the value out of any URL logging.
// Body is `{"value": "..."}`.
func (h *Handlers) secretsSet(w http.ResponseWriter, r *http.Request) {
	if h.secretsPath == "" {
		writeError(w, http.StatusConflict,
			errors.New("secrets file path not configured"))
		return
	}
	key := chi.URLParam(r, "key")
	if !validSecretKey(key) {
		writeError(w, http.StatusBadRequest,
			errors.New("invalid key (must match [A-Za-z_][A-Za-z0-9_]*)"))
		return
	}
	var req secretsSetReq
	if err := json.NewDecoder(io.LimitReader(r.Body, 64<<10)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Value == "" {
		writeError(w, http.StatusBadRequest, errors.New("value is required (use DELETE to remove)"))
		return
	}
	secrets, err := LoadSecretsWithLogger(h.secretsPath, h.log)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if err := secrets.Set(key, req.Value); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	state, err := h.loadState()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, state)
}

// secretsDelete removes one key. 404 when missing.
func (h *Handlers) secretsDelete(w http.ResponseWriter, r *http.Request) {
	if h.secretsPath == "" {
		writeError(w, http.StatusConflict,
			errors.New("secrets file path not configured"))
		return
	}
	key := chi.URLParam(r, "key")
	if !validSecretKey(key) {
		writeError(w, http.StatusBadRequest, errors.New("invalid key"))
		return
	}
	secrets, err := LoadSecretsWithLogger(h.secretsPath, h.log)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if err := secrets.Delete(key); err != nil {
		if errors.Is(err, ErrSecretNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// loadState centralises the GET response — used by every mutator that
// echoes the post-write state back to the client.
func (h *Handlers) loadState() (secretsState, error) {
	state := secretsState{Path: h.secretsPath}
	if h.secretsPath == "" {
		return state, nil
	}
	if _, err := os.Stat(h.secretsPath); err == nil {
		state.Present = true
	}
	secrets, err := LoadSecretsWithLogger(h.secretsPath, h.log)
	if err != nil {
		return state, err
	}
	state.Encrypted = secrets.Encrypted()
	state.Keys = secrets.Keys()
	return state, nil
}

func writeServer(dir string, s Server) error {
	body, err := Marshal(s)
	if err != nil {
		return fmt.Errorf("marshal mcp server: %w", err)
	}
	path := filepath.Join(dir, "mcp.json")
	if err := os.WriteFile(path, body, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
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

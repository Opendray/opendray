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

	// builtins are gateway-provided servers (e.g. opendray-memory)
	// surfaced in the list for visibility. Read-only: create with a
	// colliding id, update, and delete are all rejected.
	builtins []Server
}

// SetBuiltins installs the gateway-provided server descriptors shown at
// the top of the registry list. Called from app startup once the memory
// auto-attach decision is made; safe to leave unset (no built-ins shown).
func (h *Handlers) SetBuiltins(servers []Server) {
	for i := range servers {
		servers[i].Builtin = true
	}
	h.builtins = servers
}

// AddBuiltins appends gateway-provided server descriptors without
// disturbing ones installed earlier (memory and dbtool auto-attach are
// decided independently at startup).
func (h *Handlers) AddBuiltins(servers ...Server) {
	for i := range servers {
		servers[i].Builtin = true
	}
	h.builtins = append(h.builtins, servers...)
}

// isBuiltin reports whether id belongs to a gateway-provided server.
func (h *Handlers) isBuiltin(id string) bool {
	for _, b := range h.builtins {
		if b.ID == id {
			return true
		}
	}
	return false
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
			// Validate the server from the daemon: stdio → live MCP
			// handshake; sse/http → config-sanity + reachability.
			r.Post("/test", h.test)
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
	// Built-ins first so the auto-attached gateway servers are always
	// visible, then the vault entries. Strip SourcePath from the API
	// response — it's a server-side implementation detail.
	out := make([]Server, 0, len(h.builtins)+len(all))
	out = append(out, h.builtins...)
	for _, s := range all {
		s.SourcePath = ""
		out = append(out, s)
	}
	writeJSON(w, http.StatusOK, map[string]any{"servers": out})
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !ValidID(id) {
		writeError(w, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	for _, b := range h.builtins {
		if b.ID == id {
			writeJSON(w, http.StatusOK, b)
			return
		}
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

// test validates one MCP server from the daemon. ${SECRET} placeholders
// are resolved (best-effort) so the handshake uses real credentials.
// Admin-only via the route group it's mounted under.
func (h *Handlers) test(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !ValidID(id) {
		writeError(w, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	if h.isBuiltin(id) {
		// Built-ins authenticate with a gateway-minted key injected at
		// spawn time, which the registry test path doesn't carry.
		writeError(w, http.StatusConflict,
			fmt.Errorf("%s is a built-in opendray server — it is attached and authenticated automatically at session spawn", id))
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
	var missing []string
	if sec, serr := LoadSecrets(h.secretsPath); serr == nil {
		s, missing = sec.Resolve(s)
	}
	writeJSON(w, http.StatusOK, Validate(r.Context(), s, missing))
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
	if h.isBuiltin(id) {
		writeError(w, http.StatusConflict,
			fmt.Errorf("%s is a built-in opendray server; pick another id", id))
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
	srv := *req.Server
	srv.ID = id
	if strings.TrimSpace(srv.Name) == "" {
		srv.Name = id
	}
	if err := prepareServerForWrite(&srv); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := os.MkdirAll(dest, 0o700); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
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
	if h.isBuiltin(id) {
		writeError(w, http.StatusConflict,
			fmt.Errorf("%s is a built-in opendray server and cannot be edited", id))
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
	srv := *req.Server
	srv.ID = id
	if strings.TrimSpace(srv.Name) == "" {
		srv.Name = id
	}
	if err := prepareServerForWrite(&srv); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := os.MkdirAll(dest, 0o700); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
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
	if h.isBuiltin(id) {
		writeError(w, http.StatusConflict,
			fmt.Errorf("%s is a built-in opendray server and cannot be deleted", id))
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

func prepareServerForWrite(s *Server) error {
	normalizeServer(s)
	if (s.Transport == "sse" || s.Transport == "http") && strings.TrimSpace(s.URL) == "" {
		return errors.New("remote MCP servers require url for sse/http transport")
	}
	return nil
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

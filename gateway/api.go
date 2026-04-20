package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/opendray/opendray/gateway/docs"
	"github.com/opendray/opendray/gateway/files"
	"github.com/opendray/opendray/kernel/store"
	"github.com/opendray/opendray/plugin"
)

// ── Session handlers ────────────────────────────────────────────

func (s *Server) listSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := s.hub.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if sessions == nil {
		sessions = []store.Session{}
	}
	respondJSON(w, http.StatusOK, sessions)
}

func (s *Server) getSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sess, _, err := s.hub.Get(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, sess)
}

func (s *Server) createSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name            string            `json:"name"`
		SessionType     string            `json:"sessionType"`
		CWD             string            `json:"cwd"`
		Model           string            `json:"model"`
		ExtraArgs       []string          `json:"extraArgs"`
		EnvOverrides    map[string]string `json:"envOverrides"`
		ClaudeAccountID string            `json:"claudeAccountId"`
		LLMProviderID   string            `json:"llmProviderId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.CWD == "" {
		respondError(w, http.StatusBadRequest, "cwd is required")
		return
	}

	sess, err := s.hub.Create(r.Context(), store.Session{
		Name: req.Name, SessionType: req.SessionType, CWD: req.CWD,
		Model: req.Model, ExtraArgs: req.ExtraArgs, EnvOverrides: req.EnvOverrides,
		ClaudeAccountID: req.ClaudeAccountID,
		LLMProviderID:   req.LLMProviderID,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, sess)
}

func (s *Server) deleteSession(w http.ResponseWriter, r *http.Request) {
	if err := s.hub.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) startSession(w http.ResponseWriter, r *http.Request) {
	if err := s.hub.Start(r.Context(), chi.URLParam(r, "id")); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "started"})
}

func (s *Server) stopSession(w http.ResponseWriter, r *http.Request) {
	if err := s.hub.Stop(r.Context(), chi.URLParam(r, "id")); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
}

func (s *Server) sendInput(w http.ResponseWriter, r *http.Request) {
	var req struct{ Input string `json:"input"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	ts, ok := s.hub.GetTerminalSession(chi.URLParam(r, "id"))
	if !ok {
		respondError(w, http.StatusNotFound, "session not running")
		return
	}
	if err := ts.WriteInput([]byte(req.Input)); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

func (s *Server) resizeSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Rows uint16 `json:"rows"`
		Cols uint16 `json:"cols"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	ts, ok := s.hub.GetTerminalSession(chi.URLParam(r, "id"))
	if !ok {
		respondError(w, http.StatusNotFound, "session not running")
		return
	}
	if err := ts.Resize(req.Rows, req.Cols); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "resized"})
}

// ── Provider handlers ───────────────────────────────────────────

func (s *Server) listProviders(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, s.plugins.ListInfo())
}

func (s *Server) getProvider(w http.ResponseWriter, r *http.Request) {
	p, ok := s.plugins.Get(chi.URLParam(r, "name"))
	if !ok {
		respondError(w, http.StatusNotFound, "provider not found")
		return
	}
	respondJSON(w, http.StatusOK, p)
}

func (s *Server) registerProvider(w http.ResponseWriter, r *http.Request) {
	var p plugin.Provider
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if p.Name == "" || p.Type == "" {
		respondError(w, http.StatusBadRequest, "name and type are required")
		return
	}
	if p.DisplayName == "" {
		p.DisplayName = p.Name
	}
	if p.Version == "" {
		p.Version = "1.0.0"
	}
	if err := s.plugins.Register(r.Context(), p); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, map[string]string{"status": "registered", "name": p.Name})
}

func (s *Server) toggleProvider(w http.ResponseWriter, r *http.Request) {
	var req struct{ Enabled bool `json:"enabled"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if err := s.plugins.SetEnabled(r.Context(), chi.URLParam(r, "name"), req.Enabled); err != nil {
		if errors.Is(err, plugin.ErrRequiredPlugin) {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"name": chi.URLParam(r, "name"), "enabled": req.Enabled})
}

func (s *Server) updateProviderConfig(w http.ResponseWriter, r *http.Request) {
	var cfg plugin.ProviderConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		respondError(w, http.StatusBadRequest, "invalid config")
		return
	}
	if err := s.plugins.UpdateConfig(r.Context(), chi.URLParam(r, "name"), cfg); err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) deleteProvider(w http.ResponseWriter, r *http.Request) {
	if err := s.plugins.Remove(r.Context(), chi.URLParam(r, "name")); err != nil {
		if errors.Is(err, plugin.ErrRequiredPlugin) {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) detectModels(w http.ResponseWriter, r *http.Request) {
	models, err := s.plugins.DetectModels(chi.URLParam(r, "name"))
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, models)
}

// ── Docs handlers (panel plugins) ───────────────────────────────

func (s *Server) getDocsConfig(ctx context.Context, pluginName string) (docs.ForgeConfig, error) {
	info := s.plugins.ListInfo()
	for _, pi := range info {
		if pi.Provider.Name == pluginName && pi.Provider.Type == plugin.ProviderTypePanel && pi.Enabled {
			cfg := s.effectiveConfig(ctx, pluginName, pi.Config)
			// NOTE: token should migrate to s.configSecrets().PlatformGet
			// to match the git-forge pattern (secrets never in plugin_kv).
			// For now the effectiveConfig overlay skips secret fields so
			// the inline token here still works but reads from pi.Config
			// only. Left as-is to preserve the existing Configure UX
			// until obsidian-reader's next iteration.
			return docs.ForgeConfig{
				ForgeType:      stringVal(cfg, "forgeType", "gitea"),
				BaseURL:        stringVal(cfg, "baseUrl", ""),
				Repo:           stringVal(cfg, "repo", ""),
				Token:          stringVal(cfg, "token", ""),
				Branch:         stringVal(cfg, "branch", "main"),
				BasePath:       stringVal(cfg, "basePath", ""),
				FileExtensions: stringVal(cfg, "fileExtensions", ".md"),
			}, nil
		}
	}
	return docs.ForgeConfig{}, fmt.Errorf("docs plugin %q not found or not enabled", pluginName)
}

func (s *Server) docsTree(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.getDocsConfig(r.Context(), chi.URLParam(r, "plugin"))
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	if cfg.BaseURL == "" || cfg.Repo == "" {
		respondError(w, http.StatusBadRequest, "plugin not configured: set baseUrl and repo in Providers page")
		return
	}
	path := r.URL.Query().Get("path")
	entries, err := docs.ListDir(cfg, path)
	if err != nil {
		respondError(w, http.StatusBadGateway, err.Error())
		return
	}
	if entries == nil {
		entries = []docs.FileEntry{}
	}
	respondJSON(w, http.StatusOK, entries)
}

func (s *Server) docsFile(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.getDocsConfig(r.Context(), chi.URLParam(r, "plugin"))
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	path := r.URL.Query().Get("path")
	if path == "" {
		respondError(w, http.StatusBadRequest, "path is required")
		return
	}
	file, err := docs.ReadFile(cfg, path)
	if err != nil {
		respondError(w, http.StatusBadGateway, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, file)
}

func (s *Server) docsSearch(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.getDocsConfig(r.Context(), chi.URLParam(r, "plugin"))
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	query := r.URL.Query().Get("q")
	if query == "" {
		respondError(w, http.StatusBadRequest, "q is required")
		return
	}
	results, err := docs.Search(cfg, query)
	if err != nil {
		respondError(w, http.StatusBadGateway, err.Error())
		return
	}
	if results == nil {
		results = []docs.FileEntry{}
	}
	respondJSON(w, http.StatusOK, results)
}

// ── File browser handlers (panel plugins) ───────────────────────

func (s *Server) getFilesConfig(ctx context.Context, pluginName string) (files.BrowserConfig, error) {
	info := s.plugins.ListInfo()
	for _, pi := range info {
		if pi.Provider.Name == pluginName && pi.Provider.Type == plugin.ProviderTypePanel && pi.Enabled {
			cfg := s.effectiveConfig(ctx, pluginName, pi.Config)
			roots := strings.Split(stringVal(cfg, "allowedRoots", ""), ",")
			var cleanRoots []string
			for _, r := range roots {
				r = strings.TrimSpace(r)
				if r != "" {
					cleanRoots = append(cleanRoots, r)
				}
			}
			// maxFileSize is expressed in KB; intVal handles both the
			// legacy float64/int shapes and the JSON-string shape that
			// effectiveConfig overlays from plugin_kv.
			maxSize := int64(intVal(cfg, "maxFileSize", 512)) * 1024
			return files.BrowserConfig{
				AllowedRoots: cleanRoots,
				ShowHidden:   boolVal(cfg, "showHidden", false),
				MaxFileSize:  maxSize,
				DefaultPath:  stringVal(cfg, "defaultPath", ""),
			}, nil
		}
	}
	return files.BrowserConfig{}, fmt.Errorf("file browser plugin %q not found or not enabled", pluginName)
}

func (s *Server) filesTree(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.getFilesConfig(r.Context(), chi.URLParam(r, "plugin"))
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	if len(cfg.AllowedRoots) == 0 {
		respondError(w, http.StatusBadRequest, "plugin not configured: set allowedRoots in Providers page")
		return
	}
	path := r.URL.Query().Get("path")
	entries, err := files.ListDir(cfg, path)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if entries == nil {
		entries = []files.FileEntry{}
	}
	respondJSON(w, http.StatusOK, entries)
}

// filesMkdir creates a new directory under the given parent, inside the
// plugin's allowed roots. Body: { "parent": "<path>", "name": "<new folder>" }.
func (s *Server) filesMkdir(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.getFilesConfig(r.Context(), chi.URLParam(r, "plugin"))
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	var req struct {
		Parent string `json:"parent"`
		Name   string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	abs, err := files.MakeDir(cfg, req.Parent, req.Name)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"path": abs})
}

func (s *Server) filesFile(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.getFilesConfig(r.Context(), chi.URLParam(r, "plugin"))
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	path := r.URL.Query().Get("path")
	if path == "" {
		respondError(w, http.StatusBadRequest, "path is required")
		return
	}
	file, err := files.ReadFile(cfg, path)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, file)
}

func (s *Server) filesSearch(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.getFilesConfig(r.Context(), chi.URLParam(r, "plugin"))
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	query := r.URL.Query().Get("q")
	basePath := r.URL.Query().Get("path")
	if query == "" {
		respondError(w, http.StatusBadRequest, "q is required")
		return
	}
	results, err := files.Search(cfg, basePath, query)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if results == nil {
		results = []files.FileEntry{}
	}
	respondJSON(w, http.StatusOK, results)
}

func stringVal(m map[string]any, key, fallback string) string {
	if v, ok := m[key].(string); ok && v != "" {
		return v
	}
	return fallback
}

func intVal(m map[string]any, key string, fallback int) int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		case string:
			if n == "" {
				return fallback
			}
			// tolerate strings like "500"
			var parsed int
			_, err := fmt.Sscanf(n, "%d", &parsed)
			if err == nil {
				return parsed
			}
		}
	}
	return fallback
}

// Legacy database handlers removed — replaced by pg-browser v1
// marketplace plugin (plugins/marketplace/packages/pg-browser/). The
// plugin runs in its own sidecar and exposes list/query commands
// through the standard plugin command pipeline, so nothing here needs
// a per-plugin HTTP surface anymore.

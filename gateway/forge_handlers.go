package gateway

// HTTP handlers for the git-forge plugin. See gateway/forge/ for the
// per-forge adapters the dispatch layer calls.
//
//   GET /api/git-forge/{plugin}/pulls?state={open|closed|all}&limit=N
//   GET /api/git-forge/{plugin}/pulls/{n}
//   GET /api/git-forge/{plugin}/pulls/{n}/diff
//   GET /api/git-forge/{plugin}/pulls/{n}/comments
//
// All routes are read-only. The panel is observer-only by design —
// writes (PR creation, merge, approve, comment) happen through the
// Claude session, not here.

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/opendray/opendray/gateway/forge"
	"github.com/opendray/opendray/plugin"
)

// getForgeConfig resolves the git-forge plugin's saved config into
// forge.Config. Non-secret keys come from the effective config
// (plugin_kv.__config.* overlaid on pi.Config) so values written
// through the v1 Configure form are visible; the API token is read
// from the platform secret store on every call so rotations pick up
// without a sidecar restart.
func (s *Server) getForgeConfig(r *http.Request, pluginName string) (forge.Config, error) {
	info := s.plugins.ListInfo()
	var pi *plugin.ProviderInfo
	for i := range info {
		if info[i].Provider.Name == pluginName {
			pi = &info[i]
			break
		}
	}
	if pi == nil {
		return forge.Config{}, fmt.Errorf("forge plugin %q not found", pluginName)
	}
	if pi.Provider.Type != plugin.ProviderTypePanel || !pi.Enabled {
		return forge.Config{}, fmt.Errorf("forge plugin %q not enabled", pluginName)
	}
	cfg := s.effectiveConfig(r.Context(), pluginName, pi.Config)

	token := ""
	if secrets := s.configSecrets(); secrets != nil {
		// The configure form writes to __config.token — same prefix
		// pluginsConfigGet uses. A missing value is normal for new
		// installs (public-repo case); the adapter handles it.
		if v, found, err := secrets.PlatformGet(r.Context(), pluginName, configKeyPrefix+"token"); err == nil && found {
			token = v
		}
	}

	return forge.Config{
		ForgeType: stringVal(cfg, "forgeType", ""),
		BaseURL:   stringVal(cfg, "baseUrl", ""),
		Repo:      stringVal(cfg, "repo", ""),
		Token:     token,
		Timeout:   time.Duration(intVal(cfg, "commandTimeoutSec", 20)) * time.Second,
	}, nil
}

// parseForgeNumber reads the {n} URL segment as a positive integer.
// Non-numeric or <=0 returns a 400 — the handler bails before
// touching the adapter layer.
func parseForgeNumber(r *http.Request) (int, error) {
	raw := chi.URLParam(r, "number")
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("pull number must be a positive integer (got %q)", raw)
	}
	return n, nil
}

func (s *Server) forgePullsList(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.getForgeConfig(r, chi.URLParam(r, "plugin"))
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	q := r.URL.Query()
	state := forge.State(q.Get("state"))
	limit := 0
	if v := q.Get("limit"); v != "" {
		_, _ = fmt.Sscanf(v, "%d", &limit)
	}
	prs, err := forge.List(r.Context(), cfg, state, limit)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, prs)
}

func (s *Server) forgePullDetail(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.getForgeConfig(r, chi.URLParam(r, "plugin"))
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	n, err := parseForgeNumber(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	pr, err := forge.Detail(r.Context(), cfg, n)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, pr)
}

func (s *Server) forgePullDiff(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.getForgeConfig(r, chi.URLParam(r, "plugin"))
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	n, err := parseForgeNumber(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	files, err := forge.Diff(r.Context(), cfg, n)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, files)
}

func (s *Server) forgePullComments(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.getForgeConfig(r, chi.URLParam(r, "plugin"))
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	n, err := parseForgeNumber(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	cs, err := forge.Comments(r.Context(), cfg, n)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, cs)
}

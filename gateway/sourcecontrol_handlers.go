package gateway

// HTTP handlers for the "source-control" panel plugin. Routes are
// mounted in server.go under /api/source-control/{plugin}/*. The
// plugin consolidates the former git-viewer + git-forge surfaces;
// Phase 1 ships the local-git half (repo discovery, bookmarks,
// status, multi-file diff, log, branches, DB-backed baselines). Forge
// / markdown preview lands in Phase 2.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	gitpkg "github.com/opendray/opendray/gateway/git"
	sc "github.com/opendray/opendray/gateway/sourcecontrol"
	"github.com/opendray/opendray/plugin"
)

// sourceControlPluginKVKey — bookmarks persist through the plugin_kv
// namespace the Configure form already reserves, so they survive
// gateway restarts without requiring a dedicated migration.
const sourceControlBookmarksKey = "bookmarks"

// ── Config resolver ─────────────────────────────────────────────

func (s *Server) getSourceControlConfig(ctx context.Context, pluginName string) (sc.Config, error) {
	info := s.plugins.ListInfo()
	for _, pi := range info {
		if pi.Provider.Name != pluginName {
			continue
		}
		if pi.Provider.Type != plugin.ProviderTypePanel || !pi.Enabled {
			return sc.Config{}, fmt.Errorf("source-control plugin %q not enabled", pluginName)
		}
		cfg := s.effectiveConfig(ctx, pluginName, pi.Config)
		// Reuse the manifest-default + $HOME expansion helper so users
		// without an explicit allowedRoots value still land on a
		// sensible $HOME root right after install.
		roots := resolveRoots(cfg, pi.Provider.ConfigSchema, "allowedRoots")
		return sc.Config{
			AllowedRoots:    roots,
			GitBinary:       stringVal(cfg, "gitBinary", "git"),
			LogLimit:        intVal(cfg, "logLimit", 50),
			DiffContext:     intVal(cfg, "diffContext", 3),
			CommandTimeout:  time.Duration(intVal(cfg, "commandTimeoutSec", 20)) * time.Second,
			MarkdownPreview: boolVal(cfg, "markdownPreview", true),
		}, nil
	}
	return sc.Config{}, fmt.Errorf("source-control plugin %q not found", pluginName)
}

// bookmarksFor returns the stored bookmarks for this plugin. Empty
// slice when none have been added; errors are soft — we'd rather
// show an empty repo list than a 500.
func (s *Server) sourceControlBookmarks(ctx context.Context, pluginName string) []string {
	kv := s.configKVStore()
	if kv == nil {
		return nil
	}
	raw, ok, err := kv.KVGet(ctx, pluginName, sourceControlBookmarksKey)
	if err != nil || !ok {
		return nil
	}
	var list []string
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil
	}
	return list
}

func (s *Server) setSourceControlBookmarks(ctx context.Context, pluginName string, list []string) error {
	kv := s.configKVStore()
	if kv == nil {
		return fmt.Errorf("kv store unavailable")
	}
	raw, err := json.Marshal(list)
	if err != nil {
		return err
	}
	return kv.KVSet(ctx, pluginName, sourceControlBookmarksKey, raw)
}

// ── Repo discovery + bookmarks ─────────────────────────────────

func (s *Server) scRepos(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "plugin")
	cfg, err := s.getSourceControlConfig(r.Context(), name)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	bms := s.sourceControlBookmarks(r.Context(), name)
	repos, err := sc.DiscoverRepos(r.Context(), cfg, bms)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if repos == nil {
		repos = []sc.Repo{}
	}
	respondJSON(w, http.StatusOK, repos)
}

func (s *Server) scBookmarksAdd(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "plugin")
	cfg, err := s.getSourceControlConfig(r.Context(), name)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Path) == "" {
		respondError(w, http.StatusBadRequest, "path is required")
		return
	}
	// Validate the path lives inside allowedRoots before persisting.
	abs, err := gitpkg.SecurePath(gitConfigFromSC(cfg), req.Path)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	bms := s.sourceControlBookmarks(r.Context(), name)
	for _, b := range bms {
		if b == abs {
			respondJSON(w, http.StatusOK, map[string]any{"path": abs, "added": false})
			return
		}
	}
	bms = append(bms, abs)
	if err := s.setSourceControlBookmarks(r.Context(), name, bms); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"path": abs, "added": true})
}

func (s *Server) scBookmarksRemove(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "plugin")
	if _, err := s.getSourceControlConfig(r.Context(), name); err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Path) == "" {
		respondError(w, http.StatusBadRequest, "path is required")
		return
	}
	bms := s.sourceControlBookmarks(r.Context(), name)
	filtered := make([]string, 0, len(bms))
	removed := false
	for _, b := range bms {
		if b == req.Path {
			removed = true
			continue
		}
		filtered = append(filtered, b)
	}
	if removed {
		if err := s.setSourceControlBookmarks(r.Context(), name, filtered); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	respondJSON(w, http.StatusOK, map[string]any{"removed": removed})
}

// ── Status / Log / Branches (delegated to gateway/git) ─────────

func (s *Server) scStatus(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.getSourceControlConfig(r.Context(), chi.URLParam(r, "plugin"))
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	result, err := gitpkg.Status(r.Context(), gitConfigFromSC(cfg), r.URL.Query().Get("repo"))
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, result)
}

func (s *Server) scLog(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.getSourceControlConfig(r.Context(), chi.URLParam(r, "plugin"))
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	limit := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		_, _ = fmt.Sscanf(v, "%d", &limit)
	}
	commits, err := gitpkg.Log(r.Context(), gitConfigFromSC(cfg), r.URL.Query().Get("repo"), limit)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if commits == nil {
		commits = []gitpkg.Commit{}
	}
	respondJSON(w, http.StatusOK, commits)
}

func (s *Server) scBranches(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.getSourceControlConfig(r.Context(), chi.URLParam(r, "plugin"))
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	branches, err := gitpkg.Branches(r.Context(), gitConfigFromSC(cfg), r.URL.Query().Get("repo"))
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if branches == nil {
		branches = []gitpkg.Branch{}
	}
	respondJSON(w, http.StatusOK, branches)
}

// ── Multi-file diff ─────────────────────────────────────────────

func (s *Server) scDiff(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.getSourceControlConfig(r.Context(), chi.URLParam(r, "plugin"))
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	q := r.URL.Query()
	mode := sc.MultiDiffMode(q.Get("mode"))
	if mode == "" {
		mode = sc.ModeUnstaged
	}
	opts := sc.MultiDiffOptions{
		Mode:   mode,
		Since:  q.Get("since"),
		Commit: q.Get("commit"),
		Full:   q.Get("full") == "1" || q.Get("full") == "true",
	}
	// Baseline mode can resolve Since from DB when caller supplies
	// sessionId instead of an explicit SHA — matches the UI flow
	// (user clicks "Show changes since session start" once, later
	// requests just include sessionId).
	if mode == sc.ModeBaseline && opts.Since == "" {
		sessionID := q.Get("sessionId")
		if sessionID == "" {
			respondError(w, http.StatusBadRequest, "baseline mode requires since or sessionId")
			return
		}
		// Resolve the repo path through SecurePath so the lookup
		// matches what SCBaselineUpsert stored.
		abs, err := gitpkg.SecurePath(gitConfigFromSC(cfg), q.Get("repo"))
		if err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		b, ok, err := s.hub.DB().SCBaselineGet(r.Context(), sessionID, abs)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			respondError(w, http.StatusNotFound, "no baseline for this session/repo")
			return
		}
		opts.Since = b.HeadSHA
	}
	result, err := sc.MultiDiff(r.Context(), cfg, q.Get("repo"), opts)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if result.Files == nil {
		result.Files = []sc.FileDiff{}
	}
	respondJSON(w, http.StatusOK, result)
}

// ── Baselines (DB-backed) ──────────────────────────────────────

func (s *Server) scBaselinePut(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.getSourceControlConfig(r.Context(), chi.URLParam(r, "plugin"))
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	var req struct {
		SessionID string `json:"sessionId"`
		Repo      string `json:"repo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.SessionID == "" || req.Repo == "" {
		respondError(w, http.StatusBadRequest, "sessionId and repo are required")
		return
	}
	abs, err := gitpkg.SecurePath(gitConfigFromSC(cfg), req.Repo)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	status, err := gitpkg.Status(r.Context(), gitConfigFromSC(cfg), abs)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if status.Head == "" {
		respondError(w, http.StatusBadRequest, "repo has no HEAD (empty repo?)")
		return
	}
	b, err := s.hub.DB().SCBaselineUpsert(r.Context(), req.SessionID, abs, status.Head)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, b)
}

func (s *Server) scBaselineGet(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.getSourceControlConfig(r.Context(), chi.URLParam(r, "plugin"))
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	q := r.URL.Query()
	sessionID := q.Get("sessionId")
	if sessionID == "" {
		respondError(w, http.StatusBadRequest, "sessionId is required")
		return
	}
	if repo := q.Get("repo"); repo != "" {
		abs, err := gitpkg.SecurePath(gitConfigFromSC(cfg), repo)
		if err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		b, ok, err := s.hub.DB().SCBaselineGet(r.Context(), sessionID, abs)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			respondJSON(w, http.StatusOK, nil)
			return
		}
		respondJSON(w, http.StatusOK, b)
		return
	}
	// No repo → list all baselines for the session.
	list, err := s.hub.DB().SCBaselineListSession(r.Context(), sessionID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, list)
}

func (s *Server) scBaselineDelete(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.getSourceControlConfig(r.Context(), chi.URLParam(r, "plugin"))
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	q := r.URL.Query()
	sessionID := q.Get("sessionId")
	repo := q.Get("repo")
	if sessionID == "" || repo == "" {
		respondError(w, http.StatusBadRequest, "sessionId and repo are required")
		return
	}
	abs, err := gitpkg.SecurePath(gitConfigFromSC(cfg), repo)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	removed, err := s.hub.DB().SCBaselineDelete(r.Context(), sessionID, abs)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"removed": removed})
}

// gitConfigFromSC is a tiny adapter so the handler layer doesn't have
// to reach into the unexported gitConfig method on sc.Config. Copies
// the slice so a later caller mutating one doesn't trip the other.
func gitConfigFromSC(c sc.Config) gitpkg.Config {
	return gitpkg.Config{
		AllowedRoots: append([]string{}, c.AllowedRoots...),
		GitBinary:    c.GitBinary,
		LogLimit:     c.LogLimit,
		DiffContext:  c.DiffContext,
		Timeout:      c.CommandTimeout,
	}
}

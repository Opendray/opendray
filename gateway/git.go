package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	gitpkg "github.com/linivek/ntc/gateway/git"
	"github.com/linivek/ntc/plugin"
)

// getGitConfig resolves the "git" panel plugin's saved config into the
// typed gitpkg.Config the backend uses.
func (s *Server) getGitConfig(pluginName string) (gitpkg.Config, error) {
	info := s.plugins.ListInfo()
	for _, pi := range info {
		if pi.Provider.Name != pluginName {
			continue
		}
		if pi.Provider.Type != plugin.ProviderTypePanel || !pi.Enabled {
			return gitpkg.Config{}, fmt.Errorf("git plugin %q not enabled", pluginName)
		}
		cfg := pi.Config

		var roots []string
		for _, r := range strings.Split(stringVal(cfg, "allowedRoots", ""), ",") {
			if r = strings.TrimSpace(r); r != "" {
				roots = append(roots, r)
			}
		}
		allowCommit := true
		if v, ok := cfg["allowCommit"].(bool); ok {
			allowCommit = v
		}
		return gitpkg.Config{
			AllowedRoots: roots,
			DefaultPath:  stringVal(cfg, "defaultPath", ""),
			GitBinary:    stringVal(cfg, "gitBinary", "git"),
			LogLimit:     intVal(cfg, "logLimit", 50),
			DiffContext:  intVal(cfg, "diffContext", 3),
			Timeout:      time.Duration(intVal(cfg, "commandTimeoutSec", 20)) * time.Second,
			AllowCommit:  allowCommit,
		}, nil
	}
	return gitpkg.Config{}, fmt.Errorf("git plugin %q not found", pluginName)
}

func (s *Server) gitStatus(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.getGitConfig(chi.URLParam(r, "plugin"))
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	result, err := gitpkg.Status(r.Context(), cfg, r.URL.Query().Get("path"))
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, result)
}

func (s *Server) gitDiff(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.getGitConfig(chi.URLParam(r, "plugin"))
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	q := r.URL.Query()
	opts := gitpkg.DiffOptions{
		Staged: q.Get("staged") == "true" || q.Get("staged") == "1",
		Since:  q.Get("since"),
		Path:   q.Get("file"),
	}
	result, err := gitpkg.Diff(r.Context(), cfg, q.Get("path"), opts)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, result)
}

func (s *Server) gitLog(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.getGitConfig(chi.URLParam(r, "plugin"))
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	limit := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		_, _ = fmt.Sscanf(v, "%d", &limit)
	}
	commits, err := gitpkg.Log(r.Context(), cfg, r.URL.Query().Get("path"), limit)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if commits == nil {
		commits = []gitpkg.Commit{}
	}
	respondJSON(w, http.StatusOK, commits)
}

func (s *Server) gitBranches(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.getGitConfig(chi.URLParam(r, "plugin"))
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	branches, err := gitpkg.Branches(r.Context(), cfg, r.URL.Query().Get("path"))
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if branches == nil {
		branches = []gitpkg.Branch{}
	}
	respondJSON(w, http.StatusOK, branches)
}

type gitPathMutation struct {
	Path  string   `json:"path"`
	Files []string `json:"files"`
}

func decodeMutation(r *http.Request) (gitPathMutation, error) {
	var req gitPathMutation
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, fmt.Errorf("invalid request body: %w", err)
	}
	return req, nil
}

func (s *Server) gitStage(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.getGitConfig(chi.URLParam(r, "plugin"))
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	req, err := decodeMutation(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := gitpkg.Stage(r.Context(), cfg, req.Path, req.Files); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) gitUnstage(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.getGitConfig(chi.URLParam(r, "plugin"))
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	req, err := decodeMutation(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := gitpkg.Unstage(r.Context(), cfg, req.Path, req.Files); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) gitDiscard(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.getGitConfig(chi.URLParam(r, "plugin"))
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	req, err := decodeMutation(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := gitpkg.Discard(r.Context(), cfg, req.Path, req.Files); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) gitCommit(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.getGitConfig(chi.URLParam(r, "plugin"))
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	var req struct {
		Path    string `json:"path"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	c, err := gitpkg.CreateCommit(r.Context(), cfg, req.Path, req.Message)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, c)
}

// ── Session baselines ───────────────────────────────────────────

func (s *Server) gitSessionSnapshot(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.getGitConfig(chi.URLParam(r, "plugin"))
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	var req struct {
		SessionID string `json:"sessionId"`
		Path      string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Path == "" {
		if sess, _, err := s.hub.Get(r.Context(), req.SessionID); err == nil {
			req.Path = sess.CWD
		}
	}
	b, err := s.git.Snapshot(r.Context(), cfg, req.SessionID, req.Path)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, b)
}

func (s *Server) gitSessionDiff(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.getGitConfig(chi.URLParam(r, "plugin"))
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	sessionID := r.URL.Query().Get("sessionId")
	result, err := s.git.SessionDiff(r.Context(), cfg, sessionID)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, result)
}

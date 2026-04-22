package gateway

// Forge instance management for the source-control plugin.
//
// One source-control install can track many forges (multiple Gitea
// servers, personal GitHub + company GitLab, etc.). Each instance is
// { id, name, type, baseUrl } stored as a JSON array in plugin_kv, and
// the access token goes through plugin_secret under a derived key so
// it never lands in the kv column.
//
// Endpoints:
//
//	GET    /forges          → list (tokens redacted, tokenSet:bool per row)
//	POST   /forges          → create (body includes token once)
//	PUT    /forges/{id}     → update (token optional; omit = keep existing)
//	DELETE /forges/{id}     → delete + clear secret
//
// PR routes (Phase 2.C) take ?repo=owner/name so one forge instance
// can answer for many repositories — solving the git-forge "one repo
// locked in config" complaint.

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/opendray/opendray/gateway/forge"
)

// forgesKVKey is the plugin_kv row that holds the instance list.
// Shares the namespace with bookmarks — keep the keys distinct so a
// single JSON decode doesn't trip the other.
const forgesKVKey = "forges"

// forgeTokenKeyPrefix — plugin_secret key layout. Per-instance so
// deleting one forge can clear its token without touching the others.
const forgeTokenKeyPrefix = "forge-token-"

// ForgeInstance is the client-facing shape. Token is never returned
// in GETs — TokenSet is true iff a secret exists for this id.
type ForgeInstance struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"` // "gitea" | "github" | "gitlab"
	BaseURL  string `json:"baseUrl"`
	TokenSet bool   `json:"tokenSet"`
}

// forgeStored is the kv-persisted shape. Excludes tokens; they live
// in plugin_secret.
type forgeStored struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	BaseURL string `json:"baseUrl"`
}

// ── Storage ─────────────────────────────────────────────────────

func (s *Server) loadForges(ctx context.Context, pluginName string) ([]forgeStored, error) {
	kv := s.configKVStore()
	if kv == nil {
		return nil, nil
	}
	raw, ok, err := kv.KVGet(ctx, pluginName, forgesKVKey)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	var list []forgeStored
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, fmt.Errorf("forges decode: %w", err)
	}
	return list, nil
}

func (s *Server) saveForges(ctx context.Context, pluginName string, list []forgeStored) error {
	kv := s.configKVStore()
	if kv == nil {
		return errors.New("kv store unavailable")
	}
	raw, err := json.Marshal(list)
	if err != nil {
		return err
	}
	return kv.KVSet(ctx, pluginName, forgesKVKey, raw)
}

// tokenExists returns true when plugin_secret holds a value for this
// forge id — used to build the TokenSet boolean in list responses.
func (s *Server) forgeTokenExists(ctx context.Context, pluginName, forgeID string) bool {
	sec := s.configSecrets()
	if sec == nil {
		return false
	}
	_, ok, err := sec.PlatformGet(ctx, pluginName, forgeTokenKeyPrefix+forgeID)
	return err == nil && ok
}

// readForgeToken returns the plaintext token for a forge id, or empty
// string when none is set. Errors are soft — forge adapters already
// tolerate an empty token on public repos.
func (s *Server) readForgeToken(ctx context.Context, pluginName, forgeID string) string {
	sec := s.configSecrets()
	if sec == nil {
		return ""
	}
	val, ok, err := sec.PlatformGet(ctx, pluginName, forgeTokenKeyPrefix+forgeID)
	if err != nil || !ok {
		return ""
	}
	return val
}

func (s *Server) writeForgeToken(ctx context.Context, pluginName, forgeID, token string) error {
	sec := s.configSecrets()
	if sec == nil {
		return errors.New("secret store unavailable")
	}
	return sec.PlatformSet(ctx, pluginName, forgeTokenKeyPrefix+forgeID, token)
}

func (s *Server) deleteForgeToken(ctx context.Context, pluginName, forgeID string) {
	sec := s.configSecrets()
	if sec == nil {
		return
	}
	// Best-effort — a missing row is fine.
	_ = sec.PlatformDelete(ctx, pluginName, forgeTokenKeyPrefix+forgeID)
}

// ── Validation ──────────────────────────────────────────────────

var forgeTypes = map[string]bool{"gitea": true, "github": true, "gitlab": true}

// forgeDefaultBaseURL returns the canonical public-instance URL for
// hosted forges (GitHub, GitLab SaaS). Gitea returns empty because
// it's typically self-hosted — there is no universal default.
func forgeDefaultBaseURL(forgeType string) string {
	switch forgeType {
	case "github":
		return "https://api.github.com"
	case "gitlab":
		return "https://gitlab.com"
	default:
		return ""
	}
}

func validateForgeInput(f forgeStored) error {
	if strings.TrimSpace(f.Name) == "" {
		return errors.New("name is required")
	}
	if !forgeTypes[f.Type] {
		return fmt.Errorf("type must be one of gitea|github|gitlab (got %q)", f.Type)
	}
	if strings.TrimSpace(f.BaseURL) == "" {
		return errors.New("baseUrl is required")
	}
	return nil
}

// newForgeID generates a short URL-safe id. 8 bytes = 16 hex chars —
// enough for no-collision in a plugin's forge list (bounded at ~dozens)
// without creating UUID-length noise in URLs.
func newForgeID() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("forge id: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

// ── Handlers ────────────────────────────────────────────────────

func (s *Server) scForgesList(w http.ResponseWriter, r *http.Request) {
	pluginName := chi.URLParam(r, "plugin")
	if _, err := s.getSourceControlConfig(r.Context(), pluginName); err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	list, err := s.loadForges(r.Context(), pluginName)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]ForgeInstance, 0, len(list))
	for _, f := range list {
		out = append(out, ForgeInstance{
			ID:       f.ID,
			Name:     f.Name,
			Type:     f.Type,
			BaseURL:  f.BaseURL,
			TokenSet: s.forgeTokenExists(r.Context(), pluginName, f.ID),
		})
	}
	respondJSON(w, http.StatusOK, out)
}

type forgeUpsertReq struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	BaseURL string `json:"baseUrl"`
	Token   string `json:"token,omitempty"` // only sent when the user (re-)enters it
}

func (s *Server) scForgesCreate(w http.ResponseWriter, r *http.Request) {
	pluginName := chi.URLParam(r, "plugin")
	if _, err := s.getSourceControlConfig(r.Context(), pluginName); err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	var req forgeUpsertReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	stored := forgeStored{
		Name:    strings.TrimSpace(req.Name),
		Type:    req.Type,
		BaseURL: strings.TrimRight(strings.TrimSpace(req.BaseURL), "/"),
	}
	// Substitute a sane default for hosted forges so the user isn't
	// forced to paste "https://api.github.com" every time. Self-
	// hosted Gitea still requires an explicit URL because there is
	// no universal default.
	if stored.BaseURL == "" {
		stored.BaseURL = forgeDefaultBaseURL(stored.Type)
	}
	if err := validateForgeInput(stored); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	id, err := newForgeID()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	stored.ID = id

	list, err := s.loadForges(r.Context(), pluginName)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	list = append(list, stored)
	if err := s.saveForges(r.Context(), pluginName, list); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if strings.TrimSpace(req.Token) != "" {
		if err := s.writeForgeToken(r.Context(), pluginName, id, req.Token); err != nil {
			// Roll back the kv write — otherwise the UI sees a forge
			// with tokenSet:false even though the user submitted one.
			rollback := make([]forgeStored, 0, len(list)-1)
			for _, f := range list {
				if f.ID != id {
					rollback = append(rollback, f)
				}
			}
			_ = s.saveForges(r.Context(), pluginName, rollback)
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	respondJSON(w, http.StatusOK, ForgeInstance{
		ID:       id,
		Name:     stored.Name,
		Type:     stored.Type,
		BaseURL:  stored.BaseURL,
		TokenSet: req.Token != "",
	})
}

func (s *Server) scForgesUpdate(w http.ResponseWriter, r *http.Request) {
	pluginName := chi.URLParam(r, "plugin")
	id := chi.URLParam(r, "id")
	if _, err := s.getSourceControlConfig(r.Context(), pluginName); err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	var req forgeUpsertReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	list, err := s.loadForges(r.Context(), pluginName)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var found bool
	for i := range list {
		if list[i].ID != id {
			continue
		}
		found = true
		updated := list[i]
		if strings.TrimSpace(req.Name) != "" {
			updated.Name = strings.TrimSpace(req.Name)
		}
		if req.Type != "" {
			updated.Type = req.Type
		}
		if strings.TrimSpace(req.BaseURL) != "" {
			updated.BaseURL = strings.TrimRight(strings.TrimSpace(req.BaseURL), "/")
		}
		if err := validateForgeInput(updated); err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		list[i] = updated
		break
	}
	if !found {
		respondError(w, http.StatusNotFound, "forge not found")
		return
	}
	if err := s.saveForges(r.Context(), pluginName, list); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Token policy: empty / missing → keep existing; non-empty →
	// overwrite. Mirrors the v1 Configure form's secret-field
	// contract so callers reuse the mental model.
	if strings.TrimSpace(req.Token) != "" {
		if err := s.writeForgeToken(r.Context(), pluginName, id, req.Token); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	respondJSON(w, http.StatusOK, map[string]any{"updated": true})
}

func (s *Server) scForgesDelete(w http.ResponseWriter, r *http.Request) {
	pluginName := chi.URLParam(r, "plugin")
	id := chi.URLParam(r, "id")
	if _, err := s.getSourceControlConfig(r.Context(), pluginName); err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	list, err := s.loadForges(r.Context(), pluginName)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	filtered := make([]forgeStored, 0, len(list))
	removed := false
	for _, f := range list {
		if f.ID == id {
			removed = true
			continue
		}
		filtered = append(filtered, f)
	}
	if !removed {
		respondError(w, http.StatusNotFound, "forge not found")
		return
	}
	if err := s.saveForges(r.Context(), pluginName, filtered); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.deleteForgeToken(r.Context(), pluginName, id)
	// Cascade: drop the per-forge saved-repos list too so orphan
	// rows don't linger when the user deletes the forge.
	s.deleteSavedReposFor(r.Context(), pluginName, id)
	respondJSON(w, http.StatusOK, map[string]any{"removed": true})
}

// forgeInstanceByID is a helper used by the PR routes (Phase 2.C) to
// resolve an id to a ready-to-dispatch forge.Config. Public within
// package gateway so adjacent files can reuse the lookup.
func (s *Server) forgeInstanceByID(ctx context.Context, pluginName, forgeID string, timeout time.Duration) (forgeStored, string, error) {
	_ = timeout // reserved for a per-instance override later; kept on the signature
	list, err := s.loadForges(ctx, pluginName)
	if err != nil {
		return forgeStored{}, "", err
	}
	for _, f := range list {
		if f.ID == forgeID {
			return f, s.readForgeToken(ctx, pluginName, forgeID), nil
		}
	}
	return forgeStored{}, "", fmt.Errorf("forge %q not found", forgeID)
}

// buildForgeConfig derives a forge.Config for one instance + optional
// repo. The Source Control plugin's commandTimeoutSec controls the
// HTTP timeout so the adapter is sandboxed consistently with the git
// invocations in the same panel.
func (s *Server) buildForgeConfig(ctx context.Context, pluginName, forgeID, repo string) (forge.Config, error) {
	cfg, err := s.getSourceControlConfig(ctx, pluginName)
	if err != nil {
		return forge.Config{}, err
	}
	f, token, err := s.forgeInstanceByID(ctx, pluginName, forgeID, cfg.CommandTimeout)
	if err != nil {
		return forge.Config{}, err
	}
	return forge.Config{
		ForgeType: f.Type,
		BaseURL:   f.BaseURL,
		Repo:      repo,
		Token:     token,
		Timeout:   cfg.CommandTimeout,
	}, nil
}

// ── Repo picker under a forge instance ─────────────────────────

func (s *Server) scForgesRepos(w http.ResponseWriter, r *http.Request) {
	pluginName := chi.URLParam(r, "plugin")
	forgeID := chi.URLParam(r, "id")
	fcfg, err := s.buildForgeConfig(r.Context(), pluginName, forgeID, "")
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		_, _ = fmt.Sscanf(v, "%d", &limit)
	}
	repos, err := forge.ListRepos(r.Context(), fcfg, limit)
	if err != nil {
		respondError(w, http.StatusBadGateway, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, repos)
}

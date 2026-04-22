package gateway

// Saved repositories per forge instance. Solves the "one repo at a
// time" complaint in a second dimension: the user curates a list of
// {owner/name, description, addedAt, lastUsedAt} under each forge so
// the Flutter picker can offer a stable, searchable list without
// hammering the upstream /repos endpoint every time the panel opens.
//
// Storage: plugin_kv key "forge-saved-repos-<forgeId>", JSON array of
// savedRepo. Deleting a forge cascades to this list via the sibling
// handler in sourcecontrol_forges.go.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

// savedReposKeyPrefix — one list per forge id.
const savedReposKeyPrefix = "forge-saved-repos-"

// SavedRepo is the stored shape. FullName is "owner/name" (or
// "group/subgroup/name" on GitLab) — same thing the PR routes take in
// ?repo=. Description is captured at add-time from upstream so the
// picker can show context without re-hitting the forge.
type SavedRepo struct {
	FullName    string    `json:"fullName"`
	Description string    `json:"description,omitempty"`
	AddedAt     time.Time `json:"addedAt"`
	LastUsedAt  time.Time `json:"lastUsedAt,omitempty"`
}

func savedReposKey(forgeID string) string { return savedReposKeyPrefix + forgeID }

// loadSavedRepos returns the stored list (empty slice when never set
// or on decode error — we prefer to surface "no saved repos yet" over
// a 500 that blocks the panel from opening).
func (s *Server) loadSavedRepos(ctx context.Context, pluginName, forgeID string) []SavedRepo {
	kv := s.configKVStore()
	if kv == nil {
		return nil
	}
	raw, ok, err := kv.KVGet(ctx, pluginName, savedReposKey(forgeID))
	if err != nil || !ok {
		return nil
	}
	var list []SavedRepo
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil
	}
	return list
}

func (s *Server) saveSavedRepos(ctx context.Context, pluginName, forgeID string, list []SavedRepo) error {
	kv := s.configKVStore()
	if kv == nil {
		return errors.New("kv store unavailable")
	}
	raw, err := json.Marshal(list)
	if err != nil {
		return err
	}
	return kv.KVSet(ctx, pluginName, savedReposKey(forgeID), raw)
}

// deleteSavedReposFor clears the list for a forge — called from
// scForgesDelete so cascading cleanup stays in one place.
func (s *Server) deleteSavedReposFor(ctx context.Context, pluginName, forgeID string) {
	kv := s.configKVStore()
	if kv == nil {
		return
	}
	_ = kv.KVDelete(ctx, pluginName, savedReposKey(forgeID))
}

// bumpSavedRepoLastUsed updates lastUsedAt for (forgeId, fullName) if
// that repo is in the saved list. No-op otherwise — we deliberately
// do NOT auto-insert because a user might be probing random repos
// and we don't want the list polluted.
func (s *Server) bumpSavedRepoLastUsed(ctx context.Context, pluginName, forgeID, fullName string) {
	fullName = strings.TrimSpace(fullName)
	if fullName == "" {
		return
	}
	list := s.loadSavedRepos(ctx, pluginName, forgeID)
	touched := false
	for i := range list {
		if list[i].FullName == fullName {
			list[i].LastUsedAt = time.Now().UTC()
			touched = true
			break
		}
	}
	if !touched {
		return
	}
	_ = s.saveSavedRepos(ctx, pluginName, forgeID, list)
}

// ── Handlers ────────────────────────────────────────────────────

func (s *Server) scSavedReposList(w http.ResponseWriter, r *http.Request) {
	pluginName := chi.URLParam(r, "plugin")
	forgeID := chi.URLParam(r, "id")
	if _, err := s.getSourceControlConfig(r.Context(), pluginName); err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	if _, _, err := s.forgeInstanceByID(r.Context(), pluginName, forgeID, 0); err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	list := s.loadSavedRepos(r.Context(), pluginName, forgeID)
	// Sort: lastUsedAt DESC (recent first), then addedAt DESC for
	// never-used rows. Matches the "recents on top" affordance the
	// UI will render at 10-20 repos.
	sort.SliceStable(list, func(i, j int) bool {
		li, lj := list[i].LastUsedAt, list[j].LastUsedAt
		if !li.IsZero() || !lj.IsZero() {
			return li.After(lj)
		}
		return list[i].AddedAt.After(list[j].AddedAt)
	})
	if list == nil {
		list = []SavedRepo{}
	}
	respondJSON(w, http.StatusOK, list)
}

type savedRepoAddReq struct {
	FullName    string `json:"fullName"`
	Description string `json:"description,omitempty"`
}

func (s *Server) scSavedReposAdd(w http.ResponseWriter, r *http.Request) {
	pluginName := chi.URLParam(r, "plugin")
	forgeID := chi.URLParam(r, "id")
	if _, err := s.getSourceControlConfig(r.Context(), pluginName); err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	if _, _, err := s.forgeInstanceByID(r.Context(), pluginName, forgeID, 0); err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	var req savedRepoAddReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	full := strings.TrimSpace(req.FullName)
	if full == "" {
		respondError(w, http.StatusBadRequest, "fullName is required")
		return
	}
	// Minimal shape check — `owner/name` or deeper (GitLab groups).
	// Reject shell metacharacters and backslashes; adapters expect
	// safe path-like strings. We leave the rest of the validation to
	// the forge itself on first PR request.
	if strings.ContainsAny(full, "\\`$|;<>") {
		respondError(w, http.StatusBadRequest, "fullName contains disallowed characters")
		return
	}
	if !strings.Contains(full, "/") {
		respondError(w, http.StatusBadRequest, "fullName must be owner/name")
		return
	}
	list := s.loadSavedRepos(r.Context(), pluginName, forgeID)
	for _, existing := range list {
		if existing.FullName == full {
			// Idempotent add — refresh description if caller sent one,
			// leave timestamps alone. Simpler for the UI than 409.
			if strings.TrimSpace(req.Description) != "" {
				for i := range list {
					if list[i].FullName == full {
						list[i].Description = req.Description
					}
				}
				_ = s.saveSavedRepos(r.Context(), pluginName, forgeID, list)
			}
			respondJSON(w, http.StatusOK, map[string]any{"added": false, "fullName": full})
			return
		}
	}
	list = append(list, SavedRepo{
		FullName:    full,
		Description: strings.TrimSpace(req.Description),
		AddedAt:     time.Now().UTC(),
	})
	if err := s.saveSavedRepos(r.Context(), pluginName, forgeID, list); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"added": true, "fullName": full})
}

func (s *Server) scSavedReposRemove(w http.ResponseWriter, r *http.Request) {
	pluginName := chi.URLParam(r, "plugin")
	forgeID := chi.URLParam(r, "id")
	if _, err := s.getSourceControlConfig(r.Context(), pluginName); err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	if _, _, err := s.forgeInstanceByID(r.Context(), pluginName, forgeID, 0); err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	var req struct {
		FullName string `json:"fullName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.FullName) == "" {
		respondError(w, http.StatusBadRequest, "fullName is required")
		return
	}
	list := s.loadSavedRepos(r.Context(), pluginName, forgeID)
	filtered := make([]SavedRepo, 0, len(list))
	removed := false
	for _, sr := range list {
		if sr.FullName == req.FullName {
			removed = true
			continue
		}
		filtered = append(filtered, sr)
	}
	if !removed {
		respondJSON(w, http.StatusOK, map[string]any{"removed": false})
		return
	}
	if err := s.saveSavedRepos(r.Context(), pluginName, forgeID, filtered); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"removed": true})
}

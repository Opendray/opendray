package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/opendray/opendray/kernel/store"
)

// claudeAccountView is the public shape of a ClaudeAccount. It never
// carries the actual token — just a "filled" bit that mirrors whether
// the token file on disk is non-empty. The raw token stays on the host.
type claudeAccountView struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	ConfigDir   string `json:"configDir"`
	TokenPath   string `json:"tokenPath"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	TokenFilled bool   `json:"tokenFilled"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

func viewOf(a store.ClaudeAccount) claudeAccountView {
	return claudeAccountView{
		ID:          a.ID,
		Name:        a.Name,
		DisplayName: a.DisplayName,
		ConfigDir:   a.ConfigDir,
		TokenPath:   a.TokenPath,
		Description: a.Description,
		Enabled:     a.Enabled,
		TokenFilled: tokenFileFilled(a.TokenPath),
		CreatedAt:   a.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   a.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// tokenFileFilled reports whether the token file exists and is non-empty.
// Returns false silently on any error — the handler surfaces that to the UI
// as "token missing, please (re-)set" rather than an HTTP error.
func tokenFileFilled(path string) bool {
	if path == "" {
		return false
	}
	st, err := os.Stat(path)
	if err != nil {
		return false
	}
	return st.Size() > 0
}

// nameRE only allows a-z, 0-9, dashes, underscores. This matches the
// claude-acc host tool's convention (claude-<name>) so the two stay in sync.
var nameRE = regexp.MustCompile(`^[a-z0-9_-]{1,32}$`)

func (s *Server) listClaudeAccounts(w http.ResponseWriter, r *http.Request) {
	accounts, err := s.hub.DB().ListClaudeAccounts(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]claudeAccountView, 0, len(accounts))
	for _, a := range accounts {
		out = append(out, viewOf(a))
	}
	respondJSON(w, http.StatusOK, map[string]any{"accounts": out})
}

func (s *Server) getClaudeAccount(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	a, err := s.hub.DB().GetClaudeAccount(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, viewOf(a))
}

// createClaudeAccount inserts a new account. Token (if supplied) is written
// to TokenPath with chmod 600; if TokenPath is empty we default to
// ~/.claude-accounts/tokens/<name>.token to stay compatible with the claude-acc
// host tool.
func (s *Server) createClaudeAccount(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
		ConfigDir   string `json:"configDir"`
		TokenPath   string `json:"tokenPath"`
		Token       string `json:"token"`
		Description string `json:"description"`
		Enabled     *bool  `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	if !nameRE.MatchString(body.Name) {
		respondError(w, http.StatusBadRequest, "name must be 1-32 chars of [a-z0-9_-]")
		return
	}

	home, _ := os.UserHomeDir()
	if body.ConfigDir == "" {
		body.ConfigDir = filepath.Join(home, ".claude-accounts", body.Name)
	}
	if body.TokenPath == "" {
		body.TokenPath = filepath.Join(home, ".claude-accounts", "tokens", body.Name+".token")
	}

	enabled := true
	if body.Enabled != nil {
		enabled = *body.Enabled
	}

	if body.Token != "" {
		if err := writeToken(body.TokenPath, body.Token); err != nil {
			respondError(w, http.StatusInternalServerError, "write token: "+err.Error())
			return
		}
	}

	created, err := s.hub.DB().CreateClaudeAccount(r.Context(), store.ClaudeAccount{
		Name:        body.Name,
		DisplayName: body.DisplayName,
		ConfigDir:   body.ConfigDir,
		TokenPath:   body.TokenPath,
		Description: body.Description,
		Enabled:     enabled,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, viewOf(created))
}

func (s *Server) updateClaudeAccount(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	current, err := s.hub.DB().GetClaudeAccount(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	var body struct {
		DisplayName *string `json:"displayName"`
		ConfigDir   *string `json:"configDir"`
		TokenPath   *string `json:"tokenPath"`
		Description *string `json:"description"`
		Enabled     *bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if body.DisplayName != nil {
		current.DisplayName = *body.DisplayName
	}
	if body.ConfigDir != nil {
		current.ConfigDir = *body.ConfigDir
	}
	if body.TokenPath != nil {
		current.TokenPath = *body.TokenPath
	}
	if body.Description != nil {
		current.Description = *body.Description
	}
	if body.Enabled != nil {
		current.Enabled = *body.Enabled
	}
	updated, err := s.hub.DB().UpdateClaudeAccount(r.Context(), id, current)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, viewOf(updated))
}

func (s *Server) toggleClaudeAccount(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := s.hub.DB().SetClaudeAccountEnabled(r.Context(), id, body.Enabled); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"enabled": body.Enabled})
}

func (s *Server) deleteClaudeAccount(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.hub.DB().DeleteClaudeAccount(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// setClaudeAccountToken writes/overwrites the token file at TokenPath.
// Body: { "token": "sk-ant-oat01-..." }
func (s *Server) setClaudeAccountToken(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	a, err := s.hub.DB().GetClaudeAccount(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(body.Token) == "" {
		respondError(w, http.StatusBadRequest, "token is required")
		return
	}
	if err := writeToken(a.TokenPath, body.Token); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"tokenFilled": true})
}

// importLocalClaudeAccounts scans ~/.claude-accounts/tokens/ on the server
// host and upserts a claude_accounts row for each *.token file found. It
// does not read the token contents — just records metadata so the UI can
// show "importable accounts" even when the gateway was installed after
// `claude-acc` was already set up.
func (s *Server) importLocalClaudeAccounts(w http.ResponseWriter, r *http.Request) {
	home, err := os.UserHomeDir()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "homedir: "+err.Error())
		return
	}
	tokensDir := filepath.Join(home, ".claude-accounts", "tokens")
	entries, err := os.ReadDir(tokensDir)
	if err != nil {
		if os.IsNotExist(err) {
			respondJSON(w, http.StatusOK, map[string]any{
				"imported": []claudeAccountView{},
				"skipped":  []string{},
				"hint":     "no ~/.claude-accounts/tokens — run `claude-acc init` on the host first",
			})
			return
		}
		respondError(w, http.StatusInternalServerError, "read tokens dir: "+err.Error())
		return
	}

	imported := make([]claudeAccountView, 0)
	skipped := make([]string, 0)

	// Sort deterministically so repeated imports are stable in the UI.
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".token")
		if name == e.Name() || !nameRE.MatchString(name) {
			skipped = append(skipped, e.Name())
			continue
		}
		if _, err := s.hub.DB().GetClaudeAccountByName(r.Context(), name); err == nil {
			// Already imported; skip.
			skipped = append(skipped, name+" (already imported)")
			continue
		}

		configDir := filepath.Join(home, ".claude-accounts", name)
		tokenPath := filepath.Join(tokensDir, e.Name())
		created, err := s.hub.DB().CreateClaudeAccount(r.Context(), store.ClaudeAccount{
			Name:        name,
			DisplayName: name,
			ConfigDir:   configDir,
			TokenPath:   tokenPath,
			Description: "imported from ~/.claude-accounts/tokens",
			Enabled:     true,
		})
		if err != nil {
			skipped = append(skipped, fmt.Sprintf("%s (create failed: %v)", name, err))
			continue
		}
		imported = append(imported, viewOf(created))
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"imported": imported,
		"skipped":  skipped,
	})
}

// switchSessionAccount re-binds a running or stopped Claude session to a
// different account. Body: { "accountId": "<uuid>" | "" }.
func (s *Server) switchSessionAccount(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		AccountID string `json:"accountId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := s.hub.SwitchAccount(r.Context(), id, body.AccountID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "switched"})
}

// writeToken writes a token to disk with chmod 600, creating the parent
// directory as needed. We trim a trailing newline so a copy-paste that
// includes one doesn't break the Claude CLI's header parsing.
func writeToken(path, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("token is empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(token+"\n"), 0o600)
}

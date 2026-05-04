package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/opendray/opendray/kernel/store"
)

// Host introspection — replaces hardcoded "where do CLIs live, what's
// already on disk, do we have credentials" assumptions with a runtime
// probe. The Flutter first-run wizard (and a future re-probe button in
// Settings) consume the result to drive its 4-step flow:
//
//   1. show what was detected
//   2. install missing CLIs (admin-gated, server-side npm install -g)
//   3. import existing ~/.claude as a claude_account row
//   4. smoke-test a session

type hostFacts struct {
	GeneratedAt        time.Time         `json:"generatedAt"`
	OS                 string            `json:"os"`
	Arch               string            `json:"arch"`
	Home               string            `json:"home"`
	DefaultProjectsDir string            `json:"defaultProjectsDir"`
	NpmGlobalBin       string            `json:"npmGlobalBin,omitempty"`
	CLIs               map[string]cliFact `json:"clis"`
	Credentials        []credentialFact   `json:"credentials"`
}

type cliFact struct {
	Name        string `json:"name"`
	Found       bool   `json:"found"`
	Path        string `json:"path,omitempty"`
	Version     string `json:"version,omitempty"`
	InstallHint string `json:"installHint,omitempty"`
}

type credentialFact struct {
	Provider    string `json:"provider"`
	Path        string `json:"path"`
	Valid       bool   `json:"valid"`
	Email       string `json:"email,omitempty"`
	Subscription string `json:"subscription,omitempty"`
	AlreadyImported bool `json:"alreadyImported,omitempty"`
}

// hostFactsHandler returns a fresh probe each call (no cache — the user
// fixes things and clicks "Re-detect"; cache would just confuse them).
func (s *Server) hostFactsHandler(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, s.probeHostFacts(r.Context()))
}

func (s *Server) probeHostFacts(ctx context.Context) hostFacts {
	home, _ := os.UserHomeDir()
	facts := hostFacts{
		GeneratedAt:        time.Now().UTC(),
		OS:                 runtime.GOOS,
		Arch:               runtime.GOARCH,
		Home:               home,
		DefaultProjectsDir: filepath.Join(home, "projects"),
		CLIs:               map[string]cliFact{},
	}

	if home != "" {
		// Common npm-global location; surface if present so the wizard
		// can show "your CLIs live at $HOME/.npm-global/bin".
		nb := filepath.Join(home, ".npm-global", "bin")
		if st, err := os.Stat(nb); err == nil && st.IsDir() {
			facts.NpmGlobalBin = nb
		}
	}

	// CLIs we want to know about. Each entry: (name, install hint).
	type probe struct{ name, hint string }
	probes := []probe{
		{"claude", "npm install -g @anthropic-ai/claude-code"},
		{"codex", "npm install -g @openai/codex"},
		{"gemini", "pipx install google-gemini-cli  # or: npm install -g @google/gemini-cli"},
		{"opencode", "curl -fsSL https://opencode.ai/install | sh"},
		{"npm", "install Node.js: https://nodejs.org"},
		{"git", "apt install git  # or: brew install git"},
	}
	for _, p := range probes {
		facts.CLIs[p.name] = probeCLI(ctx, p.name, p.hint)
	}

	// Pre-existing Claude credentials worth offering to import.
	if home != "" {
		for _, candidate := range []string{
			filepath.Join(home, ".claude", ".credentials.json"),
			filepath.Join(home, ".claude-accounts"),
		} {
			facts.Credentials = append(facts.Credentials,
				probeClaudeCreds(ctx, s, candidate)...)
		}
	}

	return facts
}

func probeCLI(ctx context.Context, name, hint string) cliFact {
	p, err := exec.LookPath(name)
	if err != nil {
		return cliFact{Name: name, Found: false, InstallHint: hint}
	}
	out := cliFact{Name: name, Found: true, Path: p}
	// Best-effort version probe (2 s timeout).
	vctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if v, err := exec.CommandContext(vctx, p, "--version").Output(); err == nil {
		out.Version = strings.TrimSpace(string(v))
	}
	return out
}

func probeClaudeCreds(ctx context.Context, s *Server, path string) []credentialFact {
	st, err := os.Stat(path)
	if err != nil {
		return nil
	}

	out := []credentialFact{}
	if st.IsDir() {
		// Treat each subdirectory holding .credentials.json as one account.
		entries, _ := os.ReadDir(path)
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			credPath := filepath.Join(path, e.Name(), ".credentials.json")
			if _, err := os.Stat(credPath); err == nil {
				out = append(out, parseClaudeCreds(ctx, s, credPath))
			}
		}
		return out
	}

	out = append(out, parseClaudeCreds(ctx, s, path))
	return out
}

func parseClaudeCreds(ctx context.Context, s *Server, path string) credentialFact {
	cf := credentialFact{Provider: "claude", Path: path}
	data, err := os.ReadFile(path)
	if err != nil {
		return cf
	}
	var creds struct {
		ClaudeAiOauth struct {
			AccessToken      string   `json:"accessToken"`
			RefreshToken     string   `json:"refreshToken"`
			SubscriptionType string   `json:"subscriptionType"`
			Scopes           []string `json:"scopes"`
		} `json:"claudeAiOauth"`
	}
	if err := json.Unmarshal(data, &creds); err != nil {
		return cf
	}
	if creds.ClaudeAiOauth.AccessToken == "" {
		return cf
	}
	cf.Valid = true
	cf.Subscription = creds.ClaudeAiOauth.SubscriptionType

	// Detect whether this credential file path is already registered as
	// a claude_accounts row, so the wizard can hide "Import" for it.
	if s != nil && s.hub != nil && s.hub.DB() != nil {
		dir := filepath.Dir(path)
		accs, err := s.hub.DB().ListClaudeAccounts(ctx)
		if err == nil {
			for _, a := range accs {
				if a.ConfigDir == dir || a.TokenPath == path {
					cf.AlreadyImported = true
					break
				}
			}
		}
	}
	return cf
}

// hostImportClaudeCredsHandler turns a detected pre-existing
// .credentials.json into a claude_accounts row so the user doesn't
// have to re-OAuth. Body: {"path":"/abs/path/to/.credentials.json","name":"optional"}.
func (s *Server) hostImportClaudeCredsHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Path == "" {
		respondError(w, http.StatusBadRequest, "missing path")
		return
	}
	if _, err := os.Stat(req.Path); err != nil {
		respondError(w, http.StatusBadRequest, "credentials not found at path: "+err.Error())
		return
	}
	if !strings.HasSuffix(req.Path, ".credentials.json") {
		respondError(w, http.StatusBadRequest, "path must end with .credentials.json")
		return
	}

	cf := parseClaudeCreds(r.Context(), s, req.Path)
	if !cf.Valid {
		respondError(w, http.StatusBadRequest, "credentials file is not a valid Claude OAuth file")
		return
	}
	if cf.AlreadyImported {
		respondError(w, http.StatusConflict, "this credential file is already registered as a Claude account")
		return
	}

	configDir := filepath.Dir(req.Path)
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "claude-imported-" + time.Now().Format("20060102-150405")
	}
	name = sanitizeFsName(name)
	if name == "" {
		name = "claude-imported-" + time.Now().Format("20060102-150405")
	}

	acc, err := s.hub.DB().CreateClaudeAccount(r.Context(), claudeAccountFromImport(name, configDir, req.Path, cf))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "create account: "+err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, map[string]any{
		"accountId":     acc.ID,
		"name":          acc.Name,
		"profile":       map[string]any{"subscription": cf.Subscription},
	})
}

// hostInstallClaudeCLIHandler runs `npm install -g @anthropic-ai/claude-code`
// server-side so a non-technical user can fix a missing CLI from the
// wizard without SSH. Streams the install output back as a single
// response (good enough for the typical 30 s install; a future PR can
// upgrade to WS streaming if installs get long).
func (s *Server) hostInstallClaudeCLIHandler(w http.ResponseWriter, r *http.Request) {
	npm, err := exec.LookPath("npm")
	if err != nil {
		respondError(w, http.StatusServiceUnavailable,
			"npm not found on PATH; install Node.js first: https://nodejs.org")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, npm, "install", "-g", "@anthropic-ai/claude-code")
	out, err := cmd.CombinedOutput()
	if err != nil {
		respondJSON(w, http.StatusOK, map[string]any{
			"installed": false,
			"error":     err.Error(),
			"output":    string(out),
		})
		return
	}
	// Re-probe to surface the resolved path/version after install.
	resetClaudeCLICache()
	cli := probeCLI(r.Context(), "claude", "")
	respondJSON(w, http.StatusOK, map[string]any{
		"installed": cli.Found,
		"path":      cli.Path,
		"version":   cli.Version,
		"output":    string(out),
	})
}

func claudeAccountFromImport(name, configDir, tokenPath string, cf credentialFact) store.ClaudeAccount {
	desc := "imported via first-run wizard"
	if cf.Subscription != "" {
		desc = fmt.Sprintf("imported via first-run wizard (Claude %s)", cf.Subscription)
	}
	return store.ClaudeAccount{
		Name:        name,
		ConfigDir:   configDir,
		TokenPath:   tokenPath,
		Description: desc,
		Enabled:     true,
	}
}

package gateway

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/go-chi/chi/v5"

	"github.com/opendray/opendray/kernel/store"
)

// In-app Claude OAuth flow.
//
// Wraps the official `claude auth login --claudeai` subprocess so the user
// never touches a terminal or a filesystem path. Architecture summary:
//
//   POST /api/claude-accounts/oauth/preflight   → check claude CLI presence
//   POST /api/claude-accounts/oauth/start       → spawn CLI, return auth URL
//   POST /api/claude-accounts/oauth/complete    → submit user-pasted code
//   POST /api/claude-accounts/oauth/cancel      → kill PTY + cleanup
//
// State is held in an in-memory map keyed by flowID. A background sweeper
// kills any flow older than oauthFlowTimeout. Server restart wipes all
// in-flight flows (which is fine — the user just retries from the UI).
//
// Per-flow CLAUDE_CONFIG_DIR isolation is non-negotiable: two concurrent
// users initiating OAuth at the same time must not see each other's
// credentials, and a cancelled flow must leave no residue.

const (
	// How long the user has to complete the browser flow before we kill
	// the PTY and clean up. Anthropic's auth code itself is shorter-
	// lived; this is just the upper bound on our wrapper holding state.
	oauthFlowTimeout = 10 * time.Minute

	// How long we wait for the CLI's first output (the URL line) after
	// spawn. If we hit this without seeing the URL, something is wrong
	// — usually claude CLI not on PATH, missing version, or env issue.
	oauthURLReadTimeout = 15 * time.Second

	// How long we wait after submitting the code for the CLI to either
	// succeed (write credentials.json) or print an error. 60 s is
	// generous; Anthropic's token exchange usually completes < 2 s.
	oauthCompleteTimeout = 60 * time.Second

	oauthMaxConcurrent = 5
	oauthTempDirPrefix = "od-claude-oauth-"
)

// oauthURLPattern matches the line the CLI prints with the authorization URL.
// Example line:
//
//	If the browser didn't open, visit: https://claude.com/cai/oauth/authorize?code=true&client_id=...
//
// The capture group is the URL.
var oauthURLPattern = regexp.MustCompile(
	`(?:^|\s)(https://claude\.com/cai/oauth/authorize\?\S+)(?:\s|$)`)

// oauthFlow holds the state for a single in-flight OAuth attempt.
type oauthFlow struct {
	id          string
	configDir   string // CLAUDE_CONFIG_DIR for this flow's CLI subprocess
	name        string // user-supplied or derived account name
	displayName string
	authURL     string
	cmd         *exec.Cmd
	ptmx        *os.File // PTY master — read CLI output, write user input
	startedAt   time.Time
	cancel      context.CancelFunc

	// Buffered stdout output — used for completion detection and debug.
	mu  sync.Mutex
	buf strings.Builder
}

// oauthFlows tracks active flows by ID.
var (
	oauthFlowsMu sync.Mutex
	oauthFlows   = make(map[string]*oauthFlow)
)

// claudeCLIBinary returns the resolved path to the claude CLI, or "" if
// not on PATH. Cached because LookPath does an actual file stat per call.
var (
	cachedClaudeCLI   string
	cachedClaudeCLIMu sync.Mutex
	cachedClaudeCLIOK bool
)

func claudeCLIBinary() (string, bool) {
	cachedClaudeCLIMu.Lock()
	defer cachedClaudeCLIMu.Unlock()
	if cachedClaudeCLIOK {
		return cachedClaudeCLI, cachedClaudeCLI != ""
	}
	p, err := exec.LookPath("claude")
	cachedClaudeCLIOK = true
	if err != nil {
		cachedClaudeCLI = ""
		return "", false
	}
	cachedClaudeCLI = p
	return p, true
}

// resetClaudeCLICache forces the next claudeCLIBinary call to re-probe
// PATH. Used by tests; production has no reason to invalidate.
func resetClaudeCLICache() {
	cachedClaudeCLIMu.Lock()
	cachedClaudeCLIOK = false
	cachedClaudeCLI = ""
	cachedClaudeCLIMu.Unlock()
}

// newOAuthFlowID returns a 16-byte hex ID. Crypto-grade random; collision
// probability is negligible at our concurrency cap.
func newOAuthFlowID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand on Linux only fails when the kernel's getrandom
		// syscall is missing — falling back to something unique-ish keeps
		// the API alive even on weird platforms. The trailing nano time
		// ensures uniqueness across calls.
		return fmt.Sprintf("fl-fallback-%d", time.Now().UnixNano())
	}
	return "fl-" + hex.EncodeToString(b)
}

// ── Preflight ───────────────────────────────────────────────────────

// preflightClaudeOAuth reports whether the in-app OAuth flow is usable
// on this host. The Flutter UI calls this before showing the "Sign in
// with Claude" button so it can surface a friendly install hint instead
// of a hard 5xx error after the user clicks.
func (s *Server) preflightClaudeOAuth(w http.ResponseWriter, r *http.Request) {
	cli, ok := claudeCLIBinary()
	if !ok {
		respondJSON(w, http.StatusOK, map[string]any{
			"available": false,
			"installHint": "OpenDray needs the official Claude Code CLI on this host. " +
				"Install it as the user that runs OpenDray:\n" +
				"  npm install -g @anthropic-ai/claude-code",
		})
		return
	}
	// Best-effort version probe — not fatal if it fails.
	version := ""
	if out, err := exec.Command(cli, "--version").Output(); err == nil {
		version = strings.TrimSpace(string(out))
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"available": true,
		"path":      cli,
		"version":   version,
	})
}

// ── Start ───────────────────────────────────────────────────────────

func (s *Server) startClaudeOAuth(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
	}
	// Body is optional — both fields are auto-derived from the OAuth
	// profile on completion if the user doesn't supply them.
	_ = json.NewDecoder(r.Body).Decode(&req)

	cli, ok := claudeCLIBinary()
	if !ok {
		respondError(w, http.StatusServiceUnavailable,
			"claude CLI not found on PATH; run `npm install -g @anthropic-ai/claude-code`")
		return
	}

	// Concurrency cap.
	oauthFlowsMu.Lock()
	if len(oauthFlows) >= oauthMaxConcurrent {
		oauthFlowsMu.Unlock()
		respondError(w, http.StatusTooManyRequests,
			fmt.Sprintf("too many concurrent OAuth flows (max %d) — cancel an existing one first", oauthMaxConcurrent))
		return
	}
	oauthFlowsMu.Unlock()

	flowID := newOAuthFlowID()
	configDir, err := os.MkdirTemp("", oauthTempDirPrefix+flowID+"-")
	if err != nil {
		respondError(w, http.StatusInternalServerError, "create temp config dir: "+err.Error())
		return
	}
	// Tighten perms — config dir holds in-progress credentials.
	_ = os.Chmod(configDir, 0o700)

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, cli, "auth", "login", "--claudeai")
	cmd.Env = append(os.Environ(),
		"CLAUDE_CONFIG_DIR="+configDir,
		// Disable any interactive features the CLI might add later
		// that could surprise our scraper.
		"NO_COLOR=1",
		"FORCE_COLOR=0",
		"TERM=dumb",
	)

	ptmx, err := pty.Start(cmd)
	if err != nil {
		cancel()
		_ = os.RemoveAll(configDir)
		respondError(w, http.StatusInternalServerError, "spawn claude CLI: "+err.Error())
		return
	}

	flow := &oauthFlow{
		id:          flowID,
		configDir:   configDir,
		name:        strings.TrimSpace(req.Name),
		displayName: strings.TrimSpace(req.DisplayName),
		cmd:         cmd,
		ptmx:        ptmx,
		startedAt:   time.Now(),
		cancel:      cancel,
	}

	// Read until we see the URL line OR hit the URL-read timeout. The
	// CLI typically prints the URL within ~500ms of spawn.
	urlCh := make(chan string, 1)
	errCh := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(ptmx)
		// Lines can be long (the URL is ~500 chars). Bump the buffer.
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			flow.mu.Lock()
			flow.buf.WriteString(line + "\n")
			flow.mu.Unlock()

			if sm := oauthURLPattern.FindStringSubmatch(line); len(sm) >= 2 {
				urlCh <- sm[1]
				// Keep draining stdout in this goroutine so the PTY
				// doesn't fill its buffer and stall the CLI; once URL
				// is sent we just accumulate output.
				for scanner.Scan() {
					flow.mu.Lock()
					flow.buf.WriteString(scanner.Text() + "\n")
					flow.mu.Unlock()
				}
				return
			}
		}
		if err := scanner.Err(); err != nil {
			errCh <- err
		} else {
			errCh <- io.EOF
		}
	}()

	select {
	case authURL := <-urlCh:
		flow.authURL = authURL
		oauthFlowsMu.Lock()
		oauthFlows[flowID] = flow
		oauthFlowsMu.Unlock()

		// Schedule a hard timeout — the CLI shouldn't hang forever
		// if the user walks away mid-flow.
		go func() {
			time.Sleep(oauthFlowTimeout)
			s.killOAuthFlow(flowID, "timed out — user did not complete the browser flow within 10 minutes")
		}()

		respondJSON(w, http.StatusOK, map[string]any{
			"flowId":           flowID,
			"authorizationUrl": authURL,
			"expiresInSec":     int(oauthFlowTimeout.Seconds()),
		})
		return

	case <-time.After(oauthURLReadTimeout):
		s.cleanupFlow(flow, "no URL in CLI output within "+oauthURLReadTimeout.String())
		respondError(w, http.StatusBadGateway,
			"claude CLI did not print a sign-in URL within "+oauthURLReadTimeout.String()+" — check that `claude --version` works as the OpenDray user")
		return

	case e := <-errCh:
		flow.mu.Lock()
		out := flow.buf.String()
		flow.mu.Unlock()
		s.cleanupFlow(flow, "CLI exited before printing URL: "+e.Error())
		respondError(w, http.StatusBadGateway,
			fmt.Sprintf("claude CLI exited unexpectedly: %v\n--- output ---\n%s", e, out))
		return
	}
}

// ── Complete ────────────────────────────────────────────────────────

func (s *Server) completeClaudeOAuth(w http.ResponseWriter, r *http.Request) {
	flowID := chi.URLParam(r, "flowId")
	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	code := strings.TrimSpace(req.Code)
	if code == "" {
		respondError(w, http.StatusBadRequest, "code is required")
		return
	}

	oauthFlowsMu.Lock()
	flow, ok := oauthFlows[flowID]
	oauthFlowsMu.Unlock()
	if !ok {
		respondError(w, http.StatusNotFound, "flow not found (may have expired or been cancelled)")
		return
	}

	// Submit code to CLI.
	if _, err := flow.ptmx.Write([]byte(code + "\n")); err != nil {
		s.cleanupFlow(flow, "write code to CLI failed")
		respondError(w, http.StatusInternalServerError, "write code to CLI: "+err.Error())
		return
	}

	// Wait for either: credentials file to appear (success), CLI to exit,
	// or the complete-timeout to fire.
	credPath := filepath.Join(flow.configDir, ".credentials.json")
	deadline := time.Now().Add(oauthCompleteTimeout)
	tick := time.NewTicker(250 * time.Millisecond)
	defer tick.Stop()

	for {
		// Check creds file.
		if info, err := os.Stat(credPath); err == nil && info.Size() > 0 {
			break
		}
		// Check if CLI exited.
		if flow.cmd.ProcessState != nil && flow.cmd.ProcessState.Exited() {
			// Either successful exit (creds should exist by now) or
			// failure. Read accumulated output to surface to user.
			flow.mu.Lock()
			out := flow.buf.String()
			flow.mu.Unlock()
			if _, err := os.Stat(credPath); err != nil {
				s.cleanupFlow(flow, "CLI exited without writing credentials")
				respondError(w, http.StatusBadRequest,
					fmt.Sprintf("sign-in failed: %s",
						lastErrorLine(out)))
				return
			}
			break
		}
		if time.Now().After(deadline) {
			s.cleanupFlow(flow, "timed out waiting for token exchange")
			respondError(w, http.StatusGatewayTimeout,
				"claude CLI did not write credentials within "+oauthCompleteTimeout.String())
			return
		}
		<-tick.C
	}

	// Credentials present — query auth status to pull profile.
	cli, _ := claudeCLIBinary()
	statusCmd := exec.Command(cli, "auth", "status")
	statusCmd.Env = append(os.Environ(), "CLAUDE_CONFIG_DIR="+flow.configDir, "TERM=dumb")
	statusOut, statusErr := statusCmd.Output()
	if statusErr != nil {
		// Non-fatal — credentials file exists, we just don't know the
		// user's email. Proceed with placeholder profile.
		statusOut = []byte(`{}`)
	}
	var profile struct {
		LoggedIn   bool   `json:"loggedIn"`
		AuthMethod string `json:"authMethod"`
		User       struct {
			Email string `json:"email"`
		} `json:"user"`
	}
	_ = json.Unmarshal(statusOut, &profile)

	// Move credentials to a permanent home and create the DB row.
	name := flow.name
	if name == "" {
		name = deriveAccountName(profile.User.Email, flow.configDir)
	}
	displayName := flow.displayName
	if displayName == "" {
		displayName = profile.User.Email
	}

	home, err := os.UserHomeDir()
	if err != nil {
		s.cleanupFlow(flow, "no home dir for permanent install: "+err.Error())
		respondError(w, http.StatusInternalServerError, "no home dir: "+err.Error())
		return
	}
	permDir := filepath.Join(home, ".opendray", "claude-accounts", name)
	if err := os.MkdirAll(filepath.Dir(permDir), 0o700); err != nil {
		s.cleanupFlow(flow, "create permanent dir failed")
		respondError(w, http.StatusInternalServerError, "create permanent dir: "+err.Error())
		return
	}
	// If a previous account with the same name was registered here,
	// move it aside rather than overwriting (user can clean up).
	if _, err := os.Stat(permDir); err == nil {
		_ = os.Rename(permDir, permDir+".old."+time.Now().Format("20060102-150405"))
	}
	if err := os.Rename(flow.configDir, permDir); err != nil {
		// Cross-device rename (EXDEV) can happen if /tmp and $HOME are on
		// different filesystems (e.g. tmpfs /tmp). Fall back to copy+remove.
		if cpErr := copyDir(flow.configDir, permDir); cpErr != nil {
			s.cleanupFlow(flow, "rename config dir failed: "+err.Error()+"; copy fallback failed: "+cpErr.Error())
			respondError(w, http.StatusInternalServerError, "rename config dir: "+err.Error()+"; copy fallback: "+cpErr.Error())
			return
		}
		_ = os.RemoveAll(flow.configDir)
	}
	// configDir is now permDir; clear flow.configDir so cleanup doesn't
	// nuke the credentials we just registered.
	flow.configDir = ""
	credPathPerm := filepath.Join(permDir, ".credentials.json")

	// Pre-seed Claude CLI onboarding state so the user lands on the
	// trust-folder prompt instead of the welcome / theme picker / login
	// method picker chain. Without this, every new account hits the
	// "Select login method" screen on first interactive launch even
	// though valid OAuth credentials are already in place.
	if err := seedClaudeOnboardingState(permDir); err != nil {
		s.logger.Warn("seed claude onboarding state failed",
			"err", err, "configDir", permDir)
	}

	// Insert claude_accounts row.
	acc := store.ClaudeAccount{
		Name:        name,
		DisplayName: displayName,
		ConfigDir:   permDir,
		TokenPath:   credPathPerm,
		Description: "added via in-app OAuth on " + time.Now().Format("2006-01-02"),
		Enabled:     true,
	}
	created, err := s.hub.DB().CreateClaudeAccount(r.Context(), acc)
	if err != nil {
		// DB row failed but credentials are on disk — leave them so
		// the user can register manually if they want. Rare edge.
		s.cleanupFlowOAuthOnly(flow)
		respondError(w, http.StatusInternalServerError,
			"credentials saved to "+permDir+" but DB row creation failed: "+err.Error())
		return
	}

	s.cleanupFlowOAuthOnly(flow) // flow object only; permDir stays
	respondJSON(w, http.StatusOK, map[string]any{
		"accountId": created.ID,
		"profile": map[string]any{
			"email":      profile.User.Email,
			"authMethod": profile.AuthMethod,
		},
		"name":        name,
		"displayName": displayName,
	})
}

// ── Cancel ──────────────────────────────────────────────────────────

func (s *Server) cancelClaudeOAuth(w http.ResponseWriter, r *http.Request) {
	flowID := chi.URLParam(r, "flowId")
	if !s.killOAuthFlow(flowID, "cancelled by user") {
		respondError(w, http.StatusNotFound, "flow not found")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

// ── Internal helpers ────────────────────────────────────────────────

// killOAuthFlow terminates a flow by ID and removes it from the map.
// Returns true if the flow existed.
func (s *Server) killOAuthFlow(flowID, reason string) bool {
	oauthFlowsMu.Lock()
	flow, ok := oauthFlows[flowID]
	if ok {
		delete(oauthFlows, flowID)
	}
	oauthFlowsMu.Unlock()
	if !ok {
		return false
	}
	s.cleanupFlow(flow, reason)
	return true
}

// cleanupFlow kills the PTY child + removes the temp config dir. Safe
// to call multiple times (idempotent).
func (s *Server) cleanupFlow(flow *oauthFlow, reason string) {
	if flow == nil {
		return
	}
	if flow.cancel != nil {
		flow.cancel()
	}
	if flow.ptmx != nil {
		_ = flow.ptmx.Close()
	}
	// Process may already be dead from the cancel; ignore errors.
	if flow.cmd != nil && flow.cmd.Process != nil {
		_ = flow.cmd.Process.Kill()
		_, _ = flow.cmd.Process.Wait()
	}
	if flow.configDir != "" {
		_ = os.RemoveAll(flow.configDir)
	}
	if s != nil && s.logger != nil {
		s.logger.Info("claude oauth flow ended", "flow", flow.id, "reason", reason)
	}
}

// cleanupFlowOAuthOnly is the post-success cleanup — remove the flow
// from the in-memory map + close the PTY but leave configDir alone
// (it's been moved to its permanent home and we don't want to delete
// the just-saved credentials).
func (s *Server) cleanupFlowOAuthOnly(flow *oauthFlow) {
	oauthFlowsMu.Lock()
	delete(oauthFlows, flow.id)
	oauthFlowsMu.Unlock()
	if flow.cancel != nil {
		flow.cancel()
	}
	if flow.ptmx != nil {
		_ = flow.ptmx.Close()
	}
	if flow.cmd != nil && flow.cmd.Process != nil {
		_, _ = flow.cmd.Process.Wait()
	}
}

// seedClaudeOnboardingState writes the keys Claude CLI checks before
// showing its welcome wizard / theme picker / "Select login method"
// picker on first interactive launch. Without this, every newly-
// registered Claude account drops the user into a 3-screen onboarding
// chain even though their OAuth credentials are already valid.
//
// What it writes:
//   - settings.json: {"theme":"dark","forceLoginMethod":"claudeai"}
//     (theme is harmless even when correct; forceLoginMethod is what
//     skips the login method picker per the decompiled CLI source).
//   - .claude.json: hasCompletedOnboarding=true + lastOnboardingVersion
//     (the resolved Claude CLI version, so the wizard's "we updated,
//     please re-onboard" trigger doesn't fire either).
//
// Both files are merged with whatever the OAuth flow has already
// written (the CLI itself populates .claude.json with userID,
// oauthAccount, etc. during `claude auth login --claudeai`). Failure
// to write is non-fatal — the user just sees the wizard once.
func seedClaudeOnboardingState(configDir string) error {
	// settings.json — small, easy to write whole.
	settings := map[string]any{
		"theme":             "dark",
		"forceLoginMethod":  "claudeai",
	}
	settingsBytes, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "settings.json"),
		settingsBytes, 0o600); err != nil {
		return fmt.Errorf("write settings.json: %w", err)
	}

	// .claude.json — read-modify-write so we don't clobber the CLI's
	// own state (userID, oauthAccount, migrationVersion, cached*).
	clauPath := filepath.Join(configDir, ".claude.json")
	var clau map[string]any
	if data, err := os.ReadFile(clauPath); err == nil {
		_ = json.Unmarshal(data, &clau)
	}
	if clau == nil {
		clau = map[string]any{}
	}
	clau["hasCompletedOnboarding"] = true
	clau["lastOnboardingVersion"] = claudeCLIVersion()
	out, err := json.MarshalIndent(clau, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal .claude.json: %w", err)
	}
	if err := os.WriteFile(clauPath, out, 0o600); err != nil {
		return fmt.Errorf("write .claude.json: %w", err)
	}
	return nil
}

// claudeCLIVersion returns the version string from `claude --version`
// (e.g. "2.1.126 (Claude Code)"), parsed down to the SemVer prefix.
// Falls back to "0.0.0" so the field is always present — the value
// only matters as a "we've onboarded for at least this version" mark.
func claudeCLIVersion() string {
	cli, ok := claudeCLIBinary()
	if !ok {
		return "0.0.0"
	}
	out, err := exec.Command(cli, "--version").Output()
	if err != nil {
		return "0.0.0"
	}
	s := strings.TrimSpace(string(out))
	// "2.1.126 (Claude Code)" → "2.1.126"
	if i := strings.Index(s, " "); i > 0 {
		s = s[:i]
	}
	return s
}

// copyDir copies src to dst preserving file modes. Used as a fallback
// when os.Rename fails with EXDEV (cross-filesystem move — common when
// /tmp is tmpfs and the destination is on a real disk).
func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat src: %w", err)
	}
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return fmt.Errorf("mkdir dst: %w", err)
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("read src: %w", err)
	}
	for _, e := range entries {
		sp := filepath.Join(src, e.Name())
		dp := filepath.Join(dst, e.Name())
		if e.IsDir() {
			if err := copyDir(sp, dp); err != nil {
				return err
			}
			continue
		}
		sf, err := os.Open(sp)
		if err != nil {
			return fmt.Errorf("open %s: %w", sp, err)
		}
		info, err := e.Info()
		if err != nil {
			sf.Close()
			return fmt.Errorf("stat %s: %w", sp, err)
		}
		df, err := os.OpenFile(dp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
		if err != nil {
			sf.Close()
			return fmt.Errorf("create %s: %w", dp, err)
		}
		if _, err := io.Copy(df, sf); err != nil {
			sf.Close()
			df.Close()
			return fmt.Errorf("copy %s -> %s: %w", sp, dp, err)
		}
		sf.Close()
		if err := df.Close(); err != nil {
			return fmt.Errorf("close %s: %w", dp, err)
		}
	}
	return nil
}

// deriveAccountName picks a sensible default account name from the
// OAuth profile's email (or falls back to a timestamp). The output is
// safe for use in filesystem paths.
func deriveAccountName(email, fallback string) string {
	if email != "" {
		// e.g. "navid@example.com" → "navid-example-com"
		s := strings.ReplaceAll(email, "@", "-")
		s = strings.ReplaceAll(s, ".", "-")
		s = sanitizeFsName(s)
		if s != "" {
			return s
		}
	}
	return "claude-" + time.Now().Format("20060102-150405")
}

// sanitizeFsName keeps only [a-zA-Z0-9._-], lowercases, and collapses
// runs of separators so e.g. "emoji-🚀-strip" → "emoji-strip" rather
// than the ugly "emoji---strip". Mirrors the validation rule used by
// the existing claude_accounts.go createClaudeAccount handler.
var (
	fsSafeName     = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)
	fsRepeatedSep  = regexp.MustCompile(`[-_.]{2,}`)
)

func sanitizeFsName(s string) string {
	s = fsSafeName.ReplaceAllString(s, "-")
	s = fsRepeatedSep.ReplaceAllString(s, "-")
	s = strings.Trim(s, ".-_")
	return strings.ToLower(s)
}

// lastErrorLine extracts the most recent error-flavoured line from the
// CLI's accumulated output, used to give the user a helpful message
// when the OAuth flow fails. Falls back to a generic message.
var errorLinePattern = regexp.MustCompile(`(?i)(invalid|error|failed|expired)`)

func lastErrorLine(out string) string {
	lines := strings.Split(out, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		l := strings.TrimSpace(lines[i])
		if l == "" {
			continue
		}
		if errorLinePattern.MatchString(l) {
			return l
		}
	}
	return "claude CLI rejected the code (check your paste — no spaces, full string)"
}

// Compile-time use of errors / io to keep linters quiet if we trim
// imports during refactor. Both are referenced in real branches above.
var _ = errors.New
var _ = io.EOF

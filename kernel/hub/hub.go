// Package hub manages multiple terminal sessions.
package hub

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/opendray/opendray/kernel/store"
	"github.com/opendray/opendray/kernel/terminal"
)

const maxConcurrent = 20

// ProviderResolver resolves provider names to CLI launch specs.
type ProviderResolver interface {
	ResolveCLI(name string) (ResolvedCLI, bool)
}

// EventEmitter is the minimal contract the hub uses to fire session
// lifecycle events. plugin.HookBus.DispatchSessionEvent satisfies it,
// but we keep this as an interface so the hub doesn't import plugin.
type EventEmitter interface {
	DispatchSessionEvent(hookType, sessionID string)
}

// SessionInjector is an optional hook for rendering per-session config
// files (e.g. MCP configs) and appending the resulting args / env to
// the provider CLI spawn. Cleanup is invoked when the session exits.
type SessionInjector interface {
	RenderFor(ctx context.Context, sessionID, agent string) (Injection, error)
	Cleanup(sessionID string)
}

// Injection is the extra args and env contributed by a SessionInjector.
type Injection struct {
	Args []string
	Env  map[string]string
}

// Hook event names that the hub emits. Kept in sync with plugin.HookOn*.
const (
	hookOnIdle        = "onIdle"
	hookOnSessionStop = "onSessionStop"
)

// ResolvedCLI is the CLI specification for launching a provider.
type ResolvedCLI struct {
	Command string
	Args    []string
	Env     map[string]string
}

// Hub coordinates all terminal sessions.
type Hub struct {
	db       *store.DB
	logger   *slog.Logger
	resolver ProviderResolver
	events   EventEmitter
	injector SessionInjector

	mu            sync.RWMutex
	sessions      map[string]*terminal.Session
	stopRequested map[string]bool // session IDs the user has explicitly stopped

	idleThreshold time.Duration
}

// Config holds hub configuration.
type Config struct {
	DB            *store.DB
	IdleThreshold time.Duration
	Logger        *slog.Logger
	Resolver      ProviderResolver
	Events        EventEmitter    // optional — when set, idle/stop events are dispatched
	Injector      SessionInjector // optional — per-session config injection (MCP, etc.)
}

// New creates a session hub.
func New(cfg Config) *Hub {
	if cfg.IdleThreshold <= 0 {
		cfg.IdleThreshold = 8 * time.Second
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	return &Hub{
		db:            cfg.DB,
		logger:        cfg.Logger,
		resolver:      cfg.Resolver,
		events:        cfg.Events,
		injector:      cfg.Injector,
		sessions:      make(map[string]*terminal.Session),
		stopRequested: make(map[string]bool),
		idleThreshold: cfg.IdleThreshold,
	}
}

func (h *Hub) emit(hookType, sessionID string) {
	if h.events != nil {
		h.events.DispatchSessionEvent(hookType, sessionID)
	}
}

// watchIdle blocks on the session's idle channel; each time it closes
// (terminal went quiet for IdleThreshold) it fires the onIdle hook, then
// waits for the session to produce new output (state → Active) before
// listening for the next idle. Without this gate, a closed IdleCh is
// immediately re-read and we'd spam idle events every 500 ms for the
// entire time the session is waiting for input.
func (h *Hub) watchIdle(sessionID string, ts *terminal.Session) {
	for {
		// Phase 1: wait for idle.
		select {
		case <-ts.Done():
			return
		case <-ts.IdleCh():
			h.emit(hookOnIdle, sessionID)
		}

		// Phase 2: wait until the session produces new output (state
		// flips back to Active). We CANNOT use IdleCh() here — when the
		// session is still idle, IdleCh() returns the same closed channel
		// which reads instantly and causes an infinite emit loop. Poll
		// the state every 2 s instead.
		for {
			select {
			case <-ts.Done():
				return
			case <-time.After(2 * time.Second):
				if ts.State() == terminal.StateActive {
					goto nextCycle
				}
			}
		}
	nextCycle:
	}
}

// DB exposes the underlying store. Handlers that manage resources adjacent
// to session lifecycle (claude accounts, MCP servers, …) go through it so
// we don't have to pass yet another dependency through the gateway config.
func (h *Hub) DB() *store.DB { return h.db }

// Create persists a new session to the database.
func (h *Hub) Create(ctx context.Context, s store.Session) (store.Session, error) {
	if s.SessionType == "" {
		s.SessionType = "claude"
	}
	return h.db.CreateSession(ctx, s)
}

// Start spawns a PTY process for the given session.
func (h *Hub) Start(ctx context.Context, id string) error {
	h.mu.Lock()
	if _, exists := h.sessions[id]; exists {
		h.mu.Unlock()
		return fmt.Errorf("hub: session %s already running", id)
	}
	if len(h.sessions) >= maxConcurrent {
		h.mu.Unlock()
		return fmt.Errorf("hub: max concurrent sessions (%d) reached", maxConcurrent)
	}
	h.mu.Unlock()

	sess, err := h.db.GetSession(ctx, id)
	if err != nil {
		return fmt.Errorf("hub: get session: %w", err)
	}

	// Resolve provider CLI spec
	resolved, ok := h.resolver.ResolveCLI(sess.SessionType)
	if !ok {
		return fmt.Errorf("hub: provider %q not available", sess.SessionType)
	}

	// Build final args: provider defaults + session-specific (model, resume, extra)
	args := resolved.Args
	if sess.SessionType == "claude" {
		// Resume an existing conversation when we already have a Claude
		// session id captured (set on first launch below — see
		// --session-id branch). Otherwise mint a deterministic UUID,
		// pass it via --session-id, and persist BEFORE spawn so a crash
		// during launch still leaves a resumable id in the DB.
		if sess.ClaudeSessionID != "" {
			args = append(args, "--resume", sess.ClaudeSessionID)
		} else {
			newID := newUUIDv4()
			args = append(args, "--session-id", newID)
			if err := h.db.UpdateClaudeSessionID(ctx, sess.ID, newID); err != nil {
				h.logger.Warn("hub: persist new claude session id failed",
					"session", sess.ID, "err", err)
			} else {
				sess.ClaudeSessionID = newID
			}
		}
	}
	if sess.Model != "" && sess.SessionType != "terminal" {
		args = append(args, "--model", sess.Model)
	}
	args = append(args, sess.ExtraArgs...)

	// Merge env: provider env + session overrides
	env := make(map[string]string)
	for k, v := range resolved.Env {
		env[k] = v
	}
	for k, v := range sess.EnvOverrides {
		env[k] = v
	}

	// Claude multi-account: if bound, inject OAuth token + config dir.
	// We do this AFTER provider env / session overrides so explicit user
	// overrides still win, and BEFORE the MCP injector so MCP env can't
	// shadow the token. Failure to read the token file is hard-failing —
	// launching with a stale keychain account silently would confuse the user.
	if sess.ClaudeAccountID != "" && sess.SessionType == "claude" {
		acc, err := h.db.GetClaudeAccount(ctx, sess.ClaudeAccountID)
		if err != nil {
			return fmt.Errorf("hub: claude account %s: %w", sess.ClaudeAccountID, err)
		}
		if !acc.Enabled {
			return fmt.Errorf("hub: claude account %q is disabled", acc.Name)
		}
		token, err := readClaudeToken(acc.TokenPath)
		if err != nil {
			return fmt.Errorf("hub: read token for %q: %w", acc.Name, err)
		}
		env["CLAUDE_CODE_OAUTH_TOKEN"] = token
		if acc.ConfigDir != "" {
			env["CLAUDE_CONFIG_DIR"] = acc.ConfigDir
		}
	}

	// LLM provider binding: agents that aren't Claude read their model
	// endpoint from a separate address book. The injection shape depends
	// on the agent:
	//
	//   - OpenCode doesn't consume OPENAI_* env vars; it reads its own
	//     opencode.json listing named providers. We generate a per-
	//     session config that routes a single provider name at the
	//     user's chosen BaseURL, set XDG_CONFIG_HOME so OpenCode finds
	//     it, and rewrite --model to "<providerName>/<modelId>".
	//   - Anything else (future OpenAI-native CLIs) gets the plain
	//     OPENAI_BASE_URL / OPENAI_API_KEY / OPENAI_MODEL passthrough.
	//
	// Placed after the claude_account block so a session with both
	// bindings (which the UI prevents) won't silently cross-wire.
	if sess.LLMProviderID != "" {
		p, err := h.db.GetLLMProvider(ctx, sess.LLMProviderID)
		if err != nil {
			return fmt.Errorf("hub: llm provider %s: %w", sess.LLMProviderID, err)
		}
		if !p.Enabled {
			return fmt.Errorf("hub: llm provider %q is disabled", p.Name)
		}
		if strings.TrimSpace(p.BaseURL) == "" {
			return fmt.Errorf("hub: llm provider %q has empty base_url", p.Name)
		}

		apiKey := ""
		if p.APIKeyEnv != "" {
			apiKey = os.Getenv(p.APIKeyEnv)
		}

		switch sess.SessionType {
		case "opencode":
			inj, err := buildOpenCodeConfig(id, p, apiKey, sess.Model)
			if err != nil {
				return fmt.Errorf("hub: %w", err)
			}
			// Three injection paths, belt-and-braces — OpenCode's
			// config-discovery rules have changed between versions
			// and not every release honours OPENCODE_CONFIG. The
			// official Ollama integration uses OPENCODE_CONFIG_CONTENT
			// (inline JSON), so we set that too. A top-level `model`
			// inside the JSON is the third safety net for versions
			// that ignore both env vars.
			env["OPENCODE_CONFIG"] = inj.ConfigPath
			env["OPENCODE_CONFIG_CONTENT"] = inj.ConfigContent
			args = rewriteModelArg(args, p.Name+"/"+sess.Model)
			h.logger.Info("hub: opencode provider wired",
				"session", id,
				"provider", p.Name,
				"base_url", p.BaseURL,
				"model", sess.Model,
				"config", inj.ConfigPath)
		default:
			if apiKey == "" {
				apiKey = "opendray-local-" + p.ID
			}
			env["OPENAI_BASE_URL"] = p.BaseURL
			env["OPENAI_API_KEY"] = apiKey
			if sess.Model != "" {
				env["OPENAI_MODEL"] = sess.Model
			}
		}
	}

	// Per-session injection (MCP config files, etc). Best-effort:
	// failure to render MCP should not block session launch.
	if h.injector != nil {
		inj, err := h.injector.RenderFor(ctx, id, sess.SessionType)
		if err != nil {
			h.logger.Warn("hub: session injector failed", "session", id, "error", err)
		} else {
			args = append(args, inj.Args...)
			for k, v := range inj.Env {
				env[k] = v
			}
		}
	}

	engine, err := terminal.Spawn(terminal.SpawnConfig{
		Command: resolved.Command,
		Args:    args,
		CWD:     sess.CWD,
		Env:     env,
	})
	if err != nil {
		return fmt.Errorf("hub: spawn: %w", err)
	}

	ts := terminal.NewSession(terminal.SessionConfig{
		Engine:        engine,
		IdleThreshold: h.idleThreshold,
		Logger:        h.logger.With("session", id),
		OnExit: func(exitErr error) {
			h.mu.Lock()
			requested := h.stopRequested[id]
			delete(h.sessions, id)
			delete(h.stopRequested, id)
			h.mu.Unlock()

			if h.injector != nil {
				h.injector.Cleanup(id)
			}
			// Best-effort wipe of any per-session OpenCode config we
			// may have written. No-op if the session wasn't bound to
			// an LLM provider.
			cleanupOpenCodeConfig(id)

			// User-requested stops always report "stopped" even if process died via SIGKILL
			status := "stopped"
			if exitErr != nil && !requested {
				status = "error"
			}
			if err := h.db.UpdateSessionStatus(context.Background(), id, status, 0); err != nil {
				h.logger.Error("hub: failed to update exit status", "session", id, "error", err)
			}
			h.emit(hookOnSessionStop, id)
		},
	})

	// Idle-watcher: when the terminal goes quiet for IdleThreshold, fire
	// the onIdle hook so subscribers (e.g. the Telegram bridge) can notify.
	// Re-arms after each idle event by calling IdleCh again.
	go h.watchIdle(id, ts)

	h.mu.Lock()
	h.sessions[id] = ts
	h.mu.Unlock()

	if err := h.db.UpdateSessionStatus(ctx, id, "running", engine.PID()); err != nil {
		h.logger.Error("hub: failed to update running status", "session", id, "error", err)
	}
	return nil
}

// SwitchAccount re-binds a running or stopped Claude session to a different
// claude_account_id. If the session is running, it is stopped, the binding
// updated, and restarted (which, when ClaudeSessionID is set, naturally
// resumes via `--resume` on the new account). Pass empty accountID to
// unbind (fall back to system keychain / env).
func (h *Hub) SwitchAccount(ctx context.Context, sessionID, accountID string) error {
	sess, err := h.db.GetSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("hub: get session: %w", err)
	}
	if sess.SessionType != "claude" {
		return fmt.Errorf("hub: cannot switch account on non-claude session")
	}

	h.mu.RLock()
	_, running := h.sessions[sessionID]
	h.mu.RUnlock()

	if running {
		if err := h.Stop(ctx, sessionID); err != nil {
			return fmt.Errorf("hub: stop before switch: %w", err)
		}
		// Wait for the exit callback to clear the map. We poll the
		// hub's own map rather than the terminal's Done() channel, since
		// the OnExit path is what removes the session from h.sessions.
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			h.mu.RLock()
			_, still := h.sessions[sessionID]
			h.mu.RUnlock()
			if !still {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	}

	if err := h.db.UpdateSessionClaudeAccount(ctx, sessionID, accountID); err != nil {
		return err
	}

	if running {
		return h.Start(ctx, sessionID)
	}
	return nil
}

// Stop terminates a running session.
func (h *Hub) Stop(ctx context.Context, id string) error {
	h.mu.Lock()
	ts, ok := h.sessions[id]
	if ok {
		h.stopRequested[id] = true
	}
	h.mu.Unlock()
	if !ok {
		return fmt.Errorf("hub: session %s not running", id)
	}
	return ts.Stop()
}

// Delete removes a session. Stops it first if running.
func (h *Hub) Delete(ctx context.Context, id string) error {
	h.mu.Lock()
	ts, running := h.sessions[id]
	if running {
		h.stopRequested[id] = true
	}
	h.mu.Unlock()
	if running {
		_ = ts.Stop()
		select {
		case <-ts.Done():
		case <-time.After(5 * time.Second):
		}
	}
	return h.db.DeleteSession(ctx, id)
}

// Get returns the database session record.
func (h *Hub) Get(ctx context.Context, id string) (store.Session, bool, error) {
	sess, err := h.db.GetSession(ctx, id)
	if err != nil {
		return store.Session{}, false, err
	}
	h.mu.RLock()
	_, running := h.sessions[id]
	h.mu.RUnlock()
	return sess, running, nil
}

// List returns all sessions.
func (h *Hub) List(ctx context.Context) ([]store.Session, error) {
	return h.db.ListSessions(ctx)
}

// GetTerminalSession returns the live terminal session for WebSocket streaming.
func (h *Hub) GetTerminalSession(id string) (*terminal.Session, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ts, ok := h.sessions[id]
	return ts, ok
}

// RunningCount returns the number of active sessions.
func (h *Hub) RunningCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.sessions)
}

// RecoverOnStartup handles sessions left in running state.
func (h *Hub) RecoverOnStartup(ctx context.Context, autoResume bool) {
	sessions, err := h.db.ListSessions(ctx)
	if err != nil {
		h.logger.Error("hub: recovery failed", "error", err)
		return
	}
	for _, s := range sessions {
		if s.Status != "running" {
			continue
		}
		alive := s.PID > 0 && isProcessAlive(s.PID)
		if !alive {
			if autoResume && s.SessionType == "claude" && s.ClaudeSessionID != "" {
				h.logger.Info("hub: auto-resuming", "session", s.ID)
				if err := h.Start(ctx, s.ID); err != nil {
					_ = h.db.UpdateSessionStatus(ctx, s.ID, "stopped", 0)
				}
			} else {
				h.logger.Info("hub: marking stale session stopped", "session", s.ID)
				_ = h.db.UpdateSessionStatus(ctx, s.ID, "stopped", 0)
			}
		}
	}
}

// StartHealthCheck periodically verifies running sessions are alive.
func (h *Hub) StartHealthCheck(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				h.mu.RLock()
				for id, ts := range h.sessions {
					if !ts.Engine().IsAlive() {
						h.logger.Warn("hub: session process died", "session", id)
					}
				}
				h.mu.RUnlock()
			}
		}
	}()
}

// StopAll terminates all running sessions.
func (h *Hub) StopAll() {
	h.mu.RLock()
	ids := make([]string, 0, len(h.sessions))
	for id := range h.sessions {
		ids = append(ids, id)
	}
	h.mu.RUnlock()
	for _, id := range ids {
		_ = h.Stop(context.Background(), id)
	}
}

func isProcessAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

// readClaudeToken reads an OAuth token from disk, trimming whitespace.
// Tokens are stored chmod 600 by the claude-acc tool; we don't enforce
// the permission bit here (the host FS is trusted), but we do reject
// empty files so a half-configured account surfaces clearly.
func readClaudeToken(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("token path is empty")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	tok := strings.TrimSpace(string(data))
	if tok == "" {
		return "", fmt.Errorf("token file is empty: %s", path)
	}
	// In-app OAuth writes the Claude CLI's structured .credentials.json,
	// not a bare token string. Detect that shape and extract accessToken.
	// Fall back to the raw value for legacy manual-setup files that hold
	// a plain `sk-ant-oat01-...` string.
	if strings.HasPrefix(tok, "{") {
		var creds struct {
			ClaudeAiOauth struct {
				AccessToken string `json:"accessToken"`
			} `json:"claudeAiOauth"`
		}
		if err := json.Unmarshal([]byte(tok), &creds); err == nil && creds.ClaudeAiOauth.AccessToken != "" {
			return creds.ClaudeAiOauth.AccessToken, nil
		}
	}
	return tok, nil
}

// newUUIDv4 generates an RFC 4122 v4 UUID using crypto/rand. Used to
// pre-mint a Claude session id we hand to `claude --session-id <uuid>`
// at spawn so the conversation can be resumed across restarts via
// `--resume <uuid>`. Avoids adding a UUID dependency for one call site.
func newUUIDv4() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand on Linux only fails when the kernel's getrandom
		// syscall is missing — fall back to a time-based fingerprint
		// so Start doesn't refuse the spawn over an entropy hiccup.
		t := time.Now().UnixNano()
		for i := 0; i < 8; i++ {
			b[i] = byte(t >> (8 * i))
		}
	}
	// RFC 4122 §4.4: set version (4) and variant (10) bits.
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	hexs := hex.EncodeToString(b[:])
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hexs[0:8], hexs[8:12], hexs[12:16], hexs[16:20], hexs[20:32])
}

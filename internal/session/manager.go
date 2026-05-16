package session

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/hinshun/vt10x"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/opendray/opendray-v2/internal/eventbus"
)

const (
	// DefaultRingSize is the per-session stdout ring buffer capacity.
	DefaultRingSize = 1 << 20 // 1 MiB
	fanoutBuffer    = 64
	pumpBufSize     = 4096
	terminateGrace  = 3 * time.Second

	defaultIdleThreshold = 30 * time.Second
	defaultIdleInterval  = 5 * time.Second

	// defaultVTCols / defaultVTRows seed the virtual terminal we keep
	// for screen snapshots. Most modern CLIs query the real PTY size on
	// startup and re-render to fit, so the actual width arrives via the
	// first /resize call — these defaults just give a sane initial
	// canvas. Cards rendering wider than this get clipped.
	defaultVTCols = 120
	defaultVTRows = 40
)

// ClientKind tags a subscriber so the manager can gate resize
// requests by source. The mobile-vs-web distinction is the only
// one that matters today: web clients have flexible, draggable
// viewports that should NOT disrupt the operator's phone session
// when both are attached at once. The PTY itself still has one
// canvas size; the gating just decides whose viewport "wins".
//
// Unknown is the safe default for legacy clients that don't pass
// a ?client= query parameter — treated as web (the more common
// caller, and the one whose resize we're willing to suppress).
type ClientKind int

const (
	ClientUnknown ClientKind = iota
	ClientWeb
	ClientMobile
)

// ParseClientKind maps the ?client= query parameter to a kind.
// Empty / unrecognised values become ClientUnknown (treated as
// web at the gating layer). Casing is normalised.
func ParseClientKind(s string) ClientKind {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "mobile":
		return ClientMobile
	case "web":
		return ClientWeb
	default:
		return ClientUnknown
	}
}

// ManagerOption mutates Manager defaults; pass to NewManager.
type ManagerOption func(*Manager)

// WithIdleThreshold sets how long a session must be silent before
// session.idle fires. Pass 0 to disable idle detection.
func WithIdleThreshold(d time.Duration) ManagerOption {
	return func(m *Manager) { m.idleThreshold = d }
}

// WithIdleInterval sets the idle-detector poll cadence. Lower values
// improve latency; higher values reduce wakeups.
func WithIdleInterval(d time.Duration) ManagerOption {
	return func(m *Manager) { m.idleInterval = d }
}

// WithClaudeHistoryConfig overrides the Claude transcript discovery
// paths used by Manager.History. Empty config = built-in HOME defaults.
func WithClaudeHistoryConfig(cfg ClaudeHistoryConfig) ManagerOption {
	return func(m *Manager) { m.claudeHistoryCfg = cfg }
}

// WithCodexHistoryConfig overrides the Codex sessions root used by
// Manager.History. Empty config = built-in ~/.codex/sessions default.
func WithCodexHistoryConfig(cfg CodexHistoryConfig) ManagerOption {
	return func(m *Manager) { m.codexHistoryCfg = cfg }
}

// WithGeminiHistoryConfig overrides the Gemini tmp + projects-file
// paths used by Manager.History. Empty config = ~/.gemini defaults.
func WithGeminiHistoryConfig(cfg GeminiHistoryConfig) ManagerOption {
	return func(m *Manager) { m.geminiHistoryCfg = cfg }
}

// Manager owns the lifecycle of all live sessions in this process.
// Sessions are persisted in postgres for visibility / audit, but the
// authoritative state for a running session is the in-memory map here.
type Manager struct {
	log       *slog.Logger
	bus       *eventbus.Hub
	store     *sessionStore
	providers ProviderResolver

	idleThreshold time.Duration
	idleInterval  time.Duration

	claudeHistoryCfg ClaudeHistoryConfig
	codexHistoryCfg  CodexHistoryConfig
	geminiHistoryCfg GeminiHistoryConfig

	mu       sync.RWMutex
	closed   bool
	sessions map[string]*runningSession
	wg       sync.WaitGroup

	// stopRequested tracks session ids the user has explicitly asked
	// to stop. waitExit consumes this to decide between StateStopped
	// (user) vs StateEnded (process exited on its own). Mirrors v1
	// hub.go's stopRequested map.
	stopMu        sync.Mutex
	stopRequested map[string]bool
}

// markStopRequested records that the user asked for a stop. Idempotent.
func (m *Manager) markStopRequested(id string) {
	m.stopMu.Lock()
	if m.stopRequested == nil {
		m.stopRequested = make(map[string]bool)
	}
	m.stopRequested[id] = true
	m.stopMu.Unlock()
}

// consumeStopRequest returns true (and clears) if the session was
// stopped by an explicit user request.
func (m *Manager) consumeStopRequest(id string) bool {
	m.stopMu.Lock()
	defer m.stopMu.Unlock()
	if m.stopRequested == nil {
		return false
	}
	v := m.stopRequested[id]
	delete(m.stopRequested, id)
	return v
}

// runningSession holds the runtime state for one active PTY-backed
// session. The exported view (Manager.Get returns Session) snapshots
// `sess` under sessMu.
type runningSession struct {
	sessMu sync.RWMutex
	sess   Session

	cmd  *exec.Cmd
	pty  *os.File
	ring *RingBuffer
	// vt is a virtual-terminal emulator fed in lockstep with `ring`.
	// ring keeps the byte-stream history for client replay; vt keeps
	// the *current screen* (post-redraw) for snapshots used by
	// notifications / previews. Without vt, snapshotting the ring
	// just yields raw TUI redraw frames.
	vt vt10x.Terminal

	tempDir string // per-session scratch dir, removed on session.ended

	subsMu sync.Mutex
	subs   map[chan []byte]ClientKind

	activityMu   sync.Mutex
	lastActivity time.Time
	isIdle       bool

	endOnce sync.Once
	endedCh chan struct{}
}

// markActive records new activity and reports whether the session was
// previously idle (so the caller can flip state back to running).
func (rs *runningSession) markActive(t time.Time) bool {
	rs.activityMu.Lock()
	defer rs.activityMu.Unlock()
	rs.lastActivity = t
	wasIdle := rs.isIdle
	rs.isIdle = false
	return wasIdle
}

// checkIdle returns true if the session has just transitioned from
// active to idle (silent for >= threshold). Returns false if already
// idle (so callers fire session.idle once per idle window) or still
// active.
func (rs *runningSession) checkIdle(now time.Time, threshold time.Duration) bool {
	rs.activityMu.Lock()
	defer rs.activityMu.Unlock()
	if rs.isIdle {
		return false
	}
	if now.Sub(rs.lastActivity) >= threshold {
		rs.isIdle = true
		return true
	}
	return false
}

func NewManager(pool *pgxpool.Pool, bus *eventbus.Hub, providers ProviderResolver, log *slog.Logger, opts ...ManagerOption) *Manager {
	if log == nil {
		log = slog.Default()
	}
	m := &Manager{
		log:           log.With("component", "session"),
		bus:           bus,
		store:         newStore(pool),
		providers:     providers,
		sessions:      make(map[string]*runningSession),
		idleThreshold: defaultIdleThreshold,
		idleInterval:  defaultIdleInterval,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// ReconcileStartup marks any DB rows in non-terminal states as
// 'ended' (exit_code=-1). Call once after NewManager and before
// serving traffic — otherwise WS clients reconnect forever to
// sessions whose PTYs died with the old gateway process.
func (m *Manager) ReconcileStartup(ctx context.Context) error {
	n, err := m.store.MarkAllRunningAsEnded(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		m.log.Info("reconciled stale sessions on startup",
			"count", n,
			"reason", "previous gateway process exited; PTYs gone")
	}
	return nil
}

// Create resolves the provider, spawns a PTY, persists the row, and
// starts the stdout pump + exit detector goroutines. Returns the
// persisted Session view.
func (m *Manager) Create(ctx context.Context, req CreateRequest) (Session, error) {
	if err := req.Validate(); err != nil {
		return Session{}, err
	}

	m.mu.RLock()
	if m.closed {
		m.mu.RUnlock()
		return Session{}, errors.New("session manager closed")
	}
	m.mu.RUnlock()

	sessID := newID()
	sess := Session{
		ID:              sessID,
		Name:            req.Name,
		ProviderID:      req.ProviderID,
		Cwd:             req.Cwd,
		Args:            req.Args,
		State:           StateRunning,
		ClaudeAccountID: req.ClaudeAccountID,
		ParentSessionID: req.ParentSessionID,
		StartedAt:       time.Now().UTC(),
	}
	if sess.Args == nil {
		sess.Args = []string{}
	}

	rs, err := m.spawn(ctx, sess, false)
	if err != nil {
		return Session{}, err
	}

	m.bus.Publish(eventbus.Event{
		Topic: "session.started",
		Data: map[string]any{
			"session_id":  rs.sess.ID,
			"provider_id": rs.sess.ProviderID,
			"name":        rs.sess.Name,
		},
	})
	return rs.sess, nil
}

// Start re-spawns a previously-stopped/ended session row. The row
// must exist and be in a terminal state. The new process inherits
// the original provider/cwd/args/claude_account_id; only the PID
// and started_at change in the DB.
func (m *Manager) Start(ctx context.Context, id string) (Session, error) {
	m.mu.RLock()
	if m.closed {
		m.mu.RUnlock()
		return Session{}, errors.New("session manager closed")
	}
	m.mu.RUnlock()

	if rs := m.lookup(id); rs != nil {
		rs.sessMu.RLock()
		state := rs.sess.State
		out := rs.sess
		rs.sessMu.RUnlock()
		if !state.IsTerminal() {
			return out, fmt.Errorf("session %s is %s: %w", id, state, ErrAlreadyRunning)
		}
	}

	sess, err := m.store.Get(ctx, id)
	if err != nil {
		return Session{}, err
	}
	// If sess.State is non-terminal at this point, the row says
	// running but it's not in our in-memory map — likely a stale
	// row surviving a gateway restart. Fall through and respawn
	// regardless of state.
	sess.State = StateRunning
	sess.EndedAt = nil
	sess.ExitCode = nil
	sess.StartedAt = time.Now().UTC()

	rs, err := m.spawn(ctx, sess, true)
	if err != nil {
		return Session{}, err
	}

	m.bus.Publish(eventbus.Event{
		Topic: "session.restarted",
		Data: map[string]any{
			"session_id":  rs.sess.ID,
			"provider_id": rs.sess.ProviderID,
		},
	})
	return rs.sess, nil
}

// spawn does the shared "PTY launch + bookkeeping" work for both
// Create (insert row) and Start (reactivate row). When reactivate is
// true, the session row is expected to already exist and is updated
// via Reactivate; otherwise a fresh row is inserted.
func (m *Manager) spawn(ctx context.Context, sess Session, reactivate bool) (*runningSession, error) {
	if info, err := os.Stat(sess.Cwd); err != nil {
		return nil, fmt.Errorf("cwd: %w", err)
	} else if !info.IsDir() {
		return nil, fmt.Errorf("cwd is not a directory: %s", sess.Cwd)
	}

	resolveCtx := WithAccountID(ctx, sess.ClaudeAccountID)
	p, err := m.providers.Resolve(resolveCtx, sess.ProviderID)
	if err != nil {
		return nil, err
	}

	tempDir := filepath.Join(os.TempDir(), "opendray-sess-"+sess.ID)
	if err := os.MkdirAll(tempDir, 0o700); err != nil {
		return nil, fmt.Errorf("session tempdir: %w", err)
	}

	var (
		extraArgs []string
		extraEnv  map[string]string
	)
	var preparedClaudeSessionID string
	if p.Prepare != nil {
		out, err := p.Prepare(WithSessionID(WithCwd(ctx, sess.Cwd), sess.ID), sess.ID, tempDir)
		if err != nil {
			_ = os.RemoveAll(tempDir)
			return nil, fmt.Errorf("provider prepare: %w", err)
		}
		extraArgs = out.Args
		extraEnv = out.Env
		// Capture the agent-side session UUID so the M18 transcript
		// reader can anchor the right *.jsonl file. For fresh spawns
		// this lands in the Insert below via sess.ClaudeSessionID;
		// for Reactivate we issue a follow-up UPDATE since that path
		// preserves the original row's columns.
		if out.ClaudeSessionID != "" {
			sess.ClaudeSessionID = out.ClaudeSessionID
			preparedClaudeSessionID = out.ClaudeSessionID
		}
	}

	// User-supplied spawn args take precedence over provider-config-derived
	// args: drop any flag from p.Args that the user is re-specifying, plus
	// any provider-side flag that the catalog declares mutually exclusive
	// with a user-supplied flag. Without this, CLIs that reject duplicate
	// flags (codex's clap rejects a second --ask-for-approval) or
	// ArgGroup conflicts (codex's --dangerously-bypass-approvals-and-sandbox
	// vs --ask-for-approval) fail to spawn.
	providerArgs := dropOverriddenFlags(p.Args, sess.Args)
	providerArgs = dropConflictingFlags(providerArgs, sess.Args, p.Conflicts)
	args := append([]string(nil), providerArgs...)
	args = append(args, extraArgs...)
	args = append(args, sess.Args...)

	cmd := exec.Command(p.Executable, args...)
	cmd.Dir = sess.Cwd
	cmd.Env = mergeEnv(os.Environ(), extraEnv)

	ptmx, err := pty.Start(cmd)
	if err != nil {
		_ = os.RemoveAll(tempDir)
		return nil, fmt.Errorf("pty.Start: %w", err)
	}

	sess.PID = cmd.Process.Pid
	sess.State = StateRunning

	if reactivate {
		if err := m.store.Reactivate(ctx, sess.ID, sess.PID); err != nil {
			_ = cmd.Process.Kill()
			_ = ptmx.Close()
			_ = os.RemoveAll(tempDir)
			return nil, err
		}
		// Reactivate preserves the original row's columns, so we
		// have to issue a follow-up UPDATE when the provider picked
		// a fresh agent-side UUID for this respawn.
		if preparedClaudeSessionID != "" {
			if err := m.store.SetClaudeSessionID(ctx, sess.ID, preparedClaudeSessionID); err != nil {
				m.log.Warn("persist claude_session_id failed; M18 transcript matching may fall back to mtime",
					"session_id", sess.ID, "err", err)
			}
		}
	} else {
		if err := m.store.Insert(ctx, sess); err != nil {
			_ = cmd.Process.Kill()
			_ = ptmx.Close()
			_ = os.RemoveAll(tempDir)
			return nil, err
		}
	}

	rs := &runningSession{
		sess:         sess,
		cmd:          cmd,
		pty:          ptmx,
		ring:         NewRing(DefaultRingSize),
		vt:           vt10x.New(vt10x.WithSize(defaultVTCols, defaultVTRows)),
		tempDir:      tempDir,
		subs:         make(map[chan []byte]ClientKind),
		lastActivity: sess.StartedAt,
		endedCh:      make(chan struct{}),
	}

	m.mu.Lock()
	m.sessions[sess.ID] = rs
	m.mu.Unlock()

	m.wg.Add(2)
	go m.pumpStdout(rs)
	go m.waitExit(rs)
	if m.idleThreshold > 0 {
		m.wg.Add(1)
		go m.idleWatcher(rs)
	}

	return rs, nil
}

func (m *Manager) lookup(id string) *runningSession {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions[id]
}

// RecentScreen returns the current visible screen of the session's
// virtual terminal, with blank trailing rows trimmed. This is the
// preferred preview source for notifications and inbox cards — it
// reflects what the user sees in the live web terminal *right now*,
// not the raw byte-stream history of the PTY (which is full of TUI
// redraw frames).
//
// Returns "" when the session is not currently running.
func (m *Manager) RecentScreen(id string) string {
	rs := m.lookup(id)
	if rs == nil || rs.vt == nil {
		return ""
	}
	return ScreenSnapshot(rs.vt)
}

// mergeEnv overlays `overrides` onto a base "K=V" slice. Keys present
// in both win for `overrides`. Used so PrepareFunc can inject env vars
// like CODEX_HOME without losing the inherited environment.
func mergeEnv(base []string, overrides map[string]string) []string {
	if len(overrides) == 0 {
		return base
	}
	seen := make(map[string]bool, len(overrides))
	out := make([]string, 0, len(base)+len(overrides))
	for _, kv := range base {
		if eq := strings.IndexByte(kv, '='); eq > 0 {
			key := kv[:eq]
			if v, ok := overrides[key]; ok {
				out = append(out, key+"="+v)
				seen[key] = true
				continue
			}
		}
		out = append(out, kv)
	}
	for k, v := range overrides {
		if !seen[k] {
			out = append(out, k+"="+v)
		}
	}
	return out
}

func (m *Manager) Get(ctx context.Context, id string) (Session, error) {
	if rs := m.lookup(id); rs != nil {
		rs.sessMu.RLock()
		defer rs.sessMu.RUnlock()
		return rs.sess, nil
	}
	return m.store.Get(ctx, id)
}

// List returns persisted sessions overlaid with in-memory state for
// any session still managed in this process. Useful so /sessions
// reports `idle` even though we don't write that to the DB on every
// transition.
func (m *Manager) List(ctx context.Context) ([]Session, error) {
	list, err := m.store.List(ctx)
	if err != nil {
		return nil, err
	}
	m.mu.RLock()
	inflight := make(map[string]State, len(m.sessions))
	for id, rs := range m.sessions {
		rs.sessMu.RLock()
		inflight[id] = rs.sess.State
		rs.sessMu.RUnlock()
	}
	m.mu.RUnlock()
	for i, s := range list {
		if state, ok := inflight[s.ID]; ok {
			list[i].State = state
		}
	}
	return list, nil
}

// Stop terminates the running process for a session but preserves
// the DB row. The user can subsequently call Start to re-spawn.
// For an already-terminal session it succeeds as a no-op.
func (m *Manager) Stop(ctx context.Context, id string) error {
	rs := m.lookup(id)
	if rs == nil {
		sess, err := m.store.Get(ctx, id)
		if err != nil {
			return err
		}
		if sess.State.IsTerminal() {
			return nil
		}
		// Row says running but not in our map — likely a stale row
		// surviving a gateway restart. Mark it stopped directly.
		return m.store.MarkTerminal(ctx, id, StateStopped, 0)
	}

	rs.sessMu.RLock()
	state := rs.sess.State
	pid := rs.sess.PID
	rs.sessMu.RUnlock()
	if state.IsTerminal() {
		return nil
	}

	m.markStopRequested(id)
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil && !errors.Is(err, syscall.ESRCH) {
		return fmt.Errorf("sigterm: %w", err)
	}

	select {
	case <-rs.endedCh:
		return nil
	case <-time.After(terminateGrace):
		_ = syscall.Kill(pid, syscall.SIGKILL)
	case <-ctx.Done():
		return ctx.Err()
	}

	select {
	case <-rs.endedCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// SwitchClaudeAccount terminates the running CLI process and respawns
// it under a different Claude account binding, reusing the same row id
// (so the UI tab and history stay intact). The CLI's in-memory
// conversation state is lost — the underlying child process is
// replaced. newAccountID == "" clears the binding (CLI uses its
// system-keychain default).
//
// Rollback: if the respawn fails the row is left in 'stopped' state
// with the *original* account_id preserved, so the user can manually
// Restart with the previous credential.
func (m *Manager) SwitchClaudeAccount(ctx context.Context, id, newAccountID string) (Session, error) {
	m.mu.RLock()
	if m.closed {
		m.mu.RUnlock()
		return Session{}, errors.New("session manager closed")
	}
	m.mu.RUnlock()

	current, err := m.Get(ctx, id)
	if err != nil {
		return Session{}, err
	}
	if current.ProviderID != "claude" {
		return Session{}, ErrAccountSwitchUnsupported
	}
	if current.ClaudeAccountID == newAccountID {
		// No-op: caller picked the binding already in place. Return
		// the current view so the UI can refresh idempotently.
		return current, nil
	}

	if err := m.Stop(ctx, id); err != nil {
		return Session{}, fmt.Errorf("stop before switch: %w", err)
	}

	sess, err := m.store.Get(ctx, id)
	if err != nil {
		return Session{}, err
	}
	sess.ClaudeAccountID = newAccountID
	sess.State = StateRunning
	sess.EndedAt = nil
	sess.ExitCode = nil
	sess.StartedAt = time.Now().UTC()

	rs, err := m.spawn(ctx, sess, true)
	if err != nil {
		// spawn failed; row is still 'stopped' with the original
		// claude_account_id (we never persisted the new value), so
		// the user can Restart back to the previous credential.
		return Session{}, fmt.Errorf("respawn under new account: %w", err)
	}

	if err := m.store.UpdateClaudeAccount(ctx, id, newAccountID); err != nil {
		// In-memory state is correct but the DB row still has the old
		// account_id. Log and continue rather than killing the freshly
		// spawned process — gateway restarts are rare and the user can
		// re-issue the switch if necessary.
		m.log.Error("persist new claude account failed",
			"session", id, "account", newAccountID, "err", err)
	}

	m.bus.Publish(eventbus.Event{
		Topic: "session.account_switched",
		Data: map[string]any{
			"session_id":  rs.sess.ID,
			"provider_id": rs.sess.ProviderID,
			"account_id":  newAccountID,
		},
	})
	return rs.sess, nil
}

// Remove tears down a session permanently — running processes are
// stopped first, then the DB row is deleted. This is the destructive
// counterpart to Stop (which leaves the row behind for restart).
func (m *Manager) Remove(ctx context.Context, id string) error {
	if err := m.Stop(ctx, id); err != nil {
		return err
	}
	return m.store.Delete(ctx, id)
}

func (m *Manager) Input(_ context.Context, id string, data []byte) error {
	rs := m.lookup(id)
	if rs == nil {
		return ErrNotFound
	}
	rs.sessMu.RLock()
	terminal := rs.sess.State.IsTerminal()
	rs.sessMu.RUnlock()
	if terminal {
		return ErrAlreadyEnded
	}
	// Strip terminal-emulator capability answers (Primary DA, CPR,
	// Status Report) before they reach the CLI's stdin. These are
	// auto-emitted by xterm.js and our Dart xterm fork when the CLI
	// queries terminal state — they're protocol-level back-channel
	// responses, not user input. Most TUIs absorb them as escape
	// sequences and discard them silently, but Gemini's input
	// parser leaks the trailing `1;2c` into the visible prompt and
	// enters a broken state that swallows the next Enter. Filtering
	// here is harmless for Claude/Codex (they fall back to defaults
	// when no DA response arrives) and fixes Gemini cleanly.
	data = stripTerminalCapabilityResponses(data)
	if len(data) == 0 {
		return nil
	}
	if _, err := rs.pty.Write(data); err != nil {
		return fmt.Errorf("pty write: %w", err)
	}
	if rs.markActive(time.Now()) {
		m.flipBackToRunning(rs)
	}
	return nil
}

// Resize updates the PTY size, with one gating rule: when at
// least one MOBILE client is subscribed, resize requests from
// WEB clients are silently dropped. The intent is to stop the
// operator's draggable browser window from disrupting the
// layout of their phone session — mobile is the primary
// surface, web is the optional "peek" view.
//
// Mobile resize requests always go through (mobile's natural
// viewport is what we want the PTY to match). When no mobile
// is connected, web's resize works as before. Unknown / legacy
// callers are treated as web for the purposes of gating.
//
// Returns ErrNotFound for missing sessions; nil for both a
// successful resize and a gated no-op (the caller can't usefully
// distinguish, and surfacing "your resize was ignored" as an
// error would just spam logs).
func (m *Manager) Resize(_ context.Context, id string, kind ClientKind, cols, rows uint16) error {
	rs := m.lookup(id)
	if rs == nil {
		return ErrNotFound
	}
	// Gate: web (or unknown) resize is suppressed when any mobile
	// client is attached. We check under subsMu so the count is
	// consistent with the actual subscriber map.
	if kind != ClientMobile {
		rs.subsMu.Lock()
		hasMobile := false
		for _, k := range rs.subs {
			if k == ClientMobile {
				hasMobile = true
				break
			}
		}
		rs.subsMu.Unlock()
		if hasMobile {
			return nil
		}
	}
	if rs.vt != nil && cols > 0 && rows > 0 {
		rs.vt.Resize(int(cols), int(rows))
	}
	return pty.Setsize(rs.pty, &pty.Winsize{Cols: cols, Rows: rows})
}

// Subscribe registers a channel that receives every chunk of stdout
// written after registration. The unsub function is idempotent.
// `kind` tags the subscriber so Resize can decide whose resize
// requests to honour when multiple client types are attached.
//
// Returns ErrAlreadyEnded if the session has already exited — the
// pump goroutine is gone, so a fresh subscriber would never receive
// data. Callers should fall back to Buffer() to read the ring
// snapshot instead of opening a stream.
func (m *Manager) Subscribe(_ context.Context, id string, kind ClientKind) (<-chan []byte, func(), error) {
	rs := m.lookup(id)
	if rs == nil {
		return nil, nil, ErrNotFound
	}
	select {
	case <-rs.endedCh:
		return nil, nil, ErrAlreadyEnded
	default:
	}
	rs.sessMu.RLock()
	if rs.sess.State.IsTerminal() {
		rs.sessMu.RUnlock()
		return nil, nil, ErrAlreadyEnded
	}
	rs.sessMu.RUnlock()

	ch := make(chan []byte, fanoutBuffer)
	rs.subsMu.Lock()
	rs.subs[ch] = kind
	rs.subsMu.Unlock()
	unsub := func() {
		rs.subsMu.Lock()
		if _, ok := rs.subs[ch]; ok {
			delete(rs.subs, ch)
			close(ch)
		}
		rs.subsMu.Unlock()
	}
	return ch, unsub, nil
}

// Buffer returns ring-buffer bytes since the caller's cursor. Pass
// since=0 to receive whatever is currently in the ring.
func (m *Manager) Buffer(_ context.Context, id string, since int64) (Replay, error) {
	rs := m.lookup(id)
	if rs == nil {
		return Replay{}, ErrNotFound
	}
	return rs.ring.SnapshotSince(since), nil
}

// History returns the user prompts found in the agent's on-disk
// transcripts under this session's project (cwd). Each provider
// has its own storage shape:
//
//   - claude → ~/.claude/projects/<encoded-cwd>/*.jsonl
//   - codex  → ~/.codex/sessions/.../rollout-*.jsonl filtered by session_meta.cwd
//   - gemini → ~/.gemini/tmp/<sha256(cwd)>/logs.json
//
// Providers without a transcript on disk (shell, etc.) return
// UnsupportedProvider=true with empty entries so the UI can render
// a friendly empty state.
//
// Reads from the persisted Session row so an ended session still
// returns its history.
func (m *Manager) History(ctx context.Context, id string, limit int) (HistoryResponse, error) {
	sess, err := m.Get(ctx, id)
	if err != nil {
		return HistoryResponse{}, err
	}
	var entries []ProjectInput
	switch sess.ProviderID {
	case "claude":
		entries = ProjectInputHistory(m.claudeHistoryCfg, sess.Cwd, limit)
	case "codex":
		entries = CodexInputHistory(m.codexHistoryCfg, sess.Cwd, limit)
	case "gemini":
		entries = GeminiInputHistory(m.geminiHistoryCfg, sess.Cwd, limit)
	default:
		return HistoryResponse{Entries: []ProjectInput{}, UnsupportedProvider: true}, nil
	}
	if entries == nil {
		entries = []ProjectInput{}
	}
	return HistoryResponse{Entries: entries}, nil
}

// Shutdown signals SIGTERM to all live sessions, waits up to 5s, then
// SIGKILL stragglers. Idempotent.
func (m *Manager) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil
	}
	m.closed = true
	rss := make([]*runningSession, 0, len(m.sessions))
	for _, rs := range m.sessions {
		rss = append(rss, rs)
	}
	m.mu.Unlock()

	for _, rs := range rss {
		rs.sessMu.RLock()
		pid := rs.sess.PID
		terminal := rs.sess.State.IsTerminal()
		rs.sessMu.RUnlock()
		if !terminal {
			_ = syscall.Kill(pid, syscall.SIGTERM)
		}
	}

	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(5 * time.Second):
		m.log.Warn("session shutdown timed out, sending SIGKILL")
		for _, rs := range rss {
			rs.sessMu.RLock()
			pid := rs.sess.PID
			ended := rs.sess.State == StateEnded
			rs.sessMu.RUnlock()
			if !ended {
				_ = syscall.Kill(pid, syscall.SIGKILL)
			}
		}
		select {
		case <-done:
		case <-ctx.Done():
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// dropOverriddenFlags returns providerArgs with any flag (and its value)
// removed when the same flag is also present in userArgs. Lets per-session
// spawn args override saved provider config without producing duplicates
// that CLI parsers reject (e.g. codex's clap rejects repeated
// --ask-for-approval).
//
// Value-flag detection is a peek heuristic: a flag is treated as taking a
// value when the following token does not itself start with "-". This
// matches every flag opendray's bundled providers actually emit (codex,
// claude, gemini). It does NOT support flag values that start with "-"
// (e.g. negative numbers); none of our providers use such values.
func dropOverriddenFlags(providerArgs, userArgs []string) []string {
	if len(providerArgs) == 0 || len(userArgs) == 0 {
		return providerArgs
	}
	override := map[string]struct{}{}
	for _, a := range userArgs {
		if name, ok := flagName(a); ok {
			override[name] = struct{}{}
		}
	}
	if len(override) == 0 {
		return providerArgs
	}
	out := make([]string, 0, len(providerArgs))
	for i := 0; i < len(providerArgs); i++ {
		tok := providerArgs[i]
		name, isFlag := flagName(tok)
		if !isFlag {
			out = append(out, tok)
			continue
		}
		if _, drop := override[name]; !drop {
			out = append(out, tok)
			continue
		}
		// Drop this flag. If it's the "--key=value" form the value is
		// already attached; otherwise peek the next token and drop it
		// too when it looks like a value (not another flag).
		if strings.Contains(tok, "=") {
			continue
		}
		if i+1 < len(providerArgs) {
			next := providerArgs[i+1]
			if _, nextIsFlag := flagName(next); !nextIsFlag {
				i++
			}
		}
	}
	return out
}

// dropConflictingFlags strips from providerArgs every flag in the
// conflict set triggered by any user spawn arg. Used for CLI parsers
// where two distinct flags can't appear together (clap ArgGroup); the
// catalog declares the rules per provider in ProviderInfo.Conflicts.
//
// Example for codex: when userArgs contains
// --dangerously-bypass-approvals-and-sandbox, every occurrence of
// --ask-for-approval, -a, --sandbox, -s (plus their values) is removed
// from providerArgs.
func dropConflictingFlags(providerArgs, userArgs []string, conflicts map[string][]string) []string {
	if len(providerArgs) == 0 || len(userArgs) == 0 || len(conflicts) == 0 {
		return providerArgs
	}
	drop := map[string]struct{}{}
	for _, a := range userArgs {
		name, ok := flagName(a)
		if !ok {
			continue
		}
		for _, victim := range conflicts[name] {
			drop[victim] = struct{}{}
		}
	}
	if len(drop) == 0 {
		return providerArgs
	}
	out := make([]string, 0, len(providerArgs))
	for i := 0; i < len(providerArgs); i++ {
		tok := providerArgs[i]
		name, isFlag := flagName(tok)
		if !isFlag {
			out = append(out, tok)
			continue
		}
		if _, victim := drop[name]; !victim {
			out = append(out, tok)
			continue
		}
		if strings.Contains(tok, "=") {
			continue
		}
		if i+1 < len(providerArgs) {
			next := providerArgs[i+1]
			if _, nextIsFlag := flagName(next); !nextIsFlag {
				i++
			}
		}
	}
	return out
}

// flagName returns the canonical name of a CLI flag token ("--ask-for-approval"
// for "--ask-for-approval=never" or "--ask-for-approval"), with ok=false for
// non-flag tokens (positional args, values).
func flagName(tok string) (string, bool) {
	if len(tok) < 2 || tok[0] != '-' {
		return "", false
	}
	if eq := strings.IndexByte(tok, '='); eq > 0 {
		return tok[:eq], true
	}
	return tok, true
}

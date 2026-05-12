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
	subs   map[chan []byte]struct{}

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
			return out, fmt.Errorf("session %s is %s — already running", id, state)
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

	args := append([]string(nil), p.Args...)
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
		subs:         make(map[chan []byte]struct{}),
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
	if _, err := rs.pty.Write(data); err != nil {
		return fmt.Errorf("pty write: %w", err)
	}
	if rs.markActive(time.Now()) {
		m.flipBackToRunning(rs)
	}
	return nil
}

func (m *Manager) Resize(_ context.Context, id string, cols, rows uint16) error {
	rs := m.lookup(id)
	if rs == nil {
		return ErrNotFound
	}
	if rs.vt != nil && cols > 0 && rows > 0 {
		rs.vt.Resize(int(cols), int(rows))
	}
	return pty.Setsize(rs.pty, &pty.Winsize{Cols: cols, Rows: rows})
}

// Subscribe registers a channel that receives every chunk of stdout
// written after registration. The unsub function is idempotent.
//
// Returns ErrAlreadyEnded if the session has already exited — the
// pump goroutine is gone, so a fresh subscriber would never receive
// data. Callers should fall back to Buffer() to read the ring
// snapshot instead of opening a stream.
func (m *Manager) Subscribe(_ context.Context, id string) (<-chan []byte, func(), error) {
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
	rs.subs[ch] = struct{}{}
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

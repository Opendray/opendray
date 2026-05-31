package session

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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

	// defaultIdleThreshold is deliberately generous: "idle" is only a
	// soft notification/label — the session process keeps running and
	// flips back to running on the next output — so a short value just
	// makes healthy sessions look idle during normal think/tool pauses.
	// Override per-deploy via [session] idle_threshold or the
	// OPENDRAY_SESSION_IDLE_THRESHOLD env var.
	defaultIdleThreshold = 5 * time.Minute
	defaultIdleInterval  = 5 * time.Second

	// defaultTurnThreshold / defaultTurnInterval drive turn-complete
	// detection: a short quiescence after observed output that means
	// "the agent likely finished replying". Distinct from idle (which
	// is a much longer "nobody's home" window) — a turn fires in
	// seconds so a chat channel can stop its "typing…" indicator and
	// deliver the reply promptly. Only armed sessions (ExpectTurn) are
	// watched, so the bus stays quiet during ordinary work.
	defaultTurnThreshold = 3 * time.Second
	defaultTurnInterval  = 1 * time.Second

	// autoResumeConcurrency bounds how many interrupted sessions are
	// re-spawned in parallel at startup. Each resume runs a provider
	// Prepare (token reads, MCP config) + pty.Start, so we throttle to
	// avoid a thundering herd on a box that just (re)booted. The total
	// count is unbounded by default — the throttle, not a cap, keeps the
	// spike bounded; OPENDRAY_AUTO_RESUME_MAX adds an optional hard cap.
	autoResumeConcurrency = 4

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
// ClaudeAccountResolver is the minimum cliacct surface SwitchClaudeAccount
// needs to migrate the conversation transcript when an operator switches
// a live session to a different account. Wiring it as a small interface
// keeps session free of the cliacct ↔ session cyclic-dep risk and lets
// tests inject a fake. Nil → no migration is attempted, behavior matches
// the pre-2026-05-31 "resume starts a fresh conversation" semantics.
type ClaudeAccountResolver interface {
	// ResolveClaudeConfigDir returns the CLAUDE_CONFIG_DIR that would
	// be injected for the given account id, or "" + nil when the
	// account is the synthetic empty-id default (CLI's own ~/.claude).
	ResolveClaudeConfigDir(ctx context.Context, accountID string) (string, error)
	// MarkClaudeAccountThrottled records that an account is rate-
	// limited until the given time. Called by the rate-limit scanner
	// hooked into pumpStdout when a session's PTY surfaces the
	// "session limit · resets HH:MM" banner.
	MarkClaudeAccountThrottled(accountID string, until time.Time)
	// IsClaudeAccountThrottled reports whether an account is in the
	// throttle map right now. Used to short-circuit repeat scans on
	// the same banner so failover only fires once per rate-limit hit.
	IsClaudeAccountThrottled(accountID string) bool
	// PickFailoverClaudeAccount picks the next account to switch a
	// throttled session to. Returns "" + nil when no non-throttled
	// enabled account is available.
	PickFailoverClaudeAccount(ctx context.Context, currentAccountID string) (string, error)
}

// WithClaudeAccountResolver injects the cliacct resolver SwitchClaudeAccount
// uses to migrate the transcript JSONL across accounts so --resume
// finds the conversation under the new CLAUDE_CONFIG_DIR. Defaults to
// nil (no migration) so existing callers keep working unchanged.
func WithClaudeAccountResolver(r ClaudeAccountResolver) ManagerOption {
	return func(m *Manager) { m.claudeAccounts = r }
}

// WithAutoFailoverEnabled flips on the rate-limit-aware auto-failover
// behavior: pumpStdout scans each Claude session's PTY output for the
// "session limit · resets HH:MM" banner and, on a match, marks the
// current account throttled and switches the session to the next
// non-throttled enabled account (via SwitchClaudeAccount with full
// transcript migration). Requires WithClaudeAccountResolver to also
// be wired. Default false so existing operators aren't surprised by
// silent account switches.
func WithAutoFailoverEnabled(enabled bool) ManagerOption {
	return func(m *Manager) { m.autoFailoverEnabled = enabled }
}

func WithIdleThreshold(d time.Duration) ManagerOption {
	return func(m *Manager) { m.idleThreshold = d }
}

// WithIdleInterval sets the idle-detector poll cadence. Lower values
// improve latency; higher values reduce wakeups.
func WithIdleInterval(d time.Duration) ManagerOption {
	return func(m *Manager) { m.idleInterval = d }
}

// WithTurnThreshold sets how long a session must be silent (after
// producing output) before session.turn_completed fires for an armed
// (ExpectTurn) session. Pass 0 to disable turn detection. Keep this
// well below the idle threshold — it's a "reply settled" signal, not
// an "abandoned" one.
func WithTurnThreshold(d time.Duration) ManagerOption {
	return func(m *Manager) { m.turnThreshold = d }
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
	log                 *slog.Logger
	bus                 *eventbus.Hub
	store               *sessionStore
	providers           ProviderResolver
	claudeAccounts      ClaudeAccountResolver // optional; nil disables transcript migration + failover
	autoFailoverEnabled bool                  // when true + claudeAccounts != nil, rate-limit scanner is hot

	idleThreshold time.Duration
	idleInterval  time.Duration
	turnThreshold time.Duration
	turnInterval  time.Duration

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

// isClosing reports whether Shutdown has begun. waitExit uses it to
// record a daemon-driven exit as 'interrupted' (resume on next start)
// rather than 'ended' (a real, agent-initiated exit).
func (m *Manager) isClosing() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.closed
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
	// expectTurn arms turn-complete detection: set by ExpectTurn when a
	// channel forwards a message into this session and wants to know
	// when the agent's reply settles. expectAt is the arming instant —
	// a turn only fires once we've seen output at or after it, so a
	// no-op submit can't trip an instant false "turn done".
	expectTurn bool
	expectAt   time.Time

	// rlMu protects rlWindow + rlLastScan. The window is a small
	// rolling buffer (≤rateLimitWindowBytes) populated from the same
	// chunk stream pumpStdout writes to the ring + fanout. Used by
	// the rate-limit scanner to detect the Claude "session limit"
	// banner without re-reading the full 1 MiB ring.
	rlMu       sync.Mutex
	rlWindow   []byte
	rlLastScan time.Time

	endOnce sync.Once
	endedCh chan struct{}
}

const (
	// rateLimitWindowBytes is the rolling buffer size the rate-limit
	// scanner sees. The banner is ≤80 bytes; 4 KiB gives plenty of
	// room for it to remain visible across several ANSI redraws.
	rateLimitWindowBytes = 4 * 1024
	// rateLimitScanCooldown bounds how often a single session re-runs
	// the regex. Without this, a banner that persists in PTY output
	// would drive the scanner on every chunk; with it, we scan at
	// most once per 5 seconds per session.
	rateLimitScanCooldown = 5 * time.Second
)

// appendRateLimitWindow appends chunk to the rolling rate-limit
// window, truncating from the front when the size exceeds the cap.
// Returns true if enough time has elapsed since the last scan that
// the caller should now run ScanForRateLimitBanner against rlWindow.
func (rs *runningSession) appendRateLimitWindow(chunk []byte, now time.Time) bool {
	rs.rlMu.Lock()
	defer rs.rlMu.Unlock()
	rs.rlWindow = append(rs.rlWindow, chunk...)
	if len(rs.rlWindow) > rateLimitWindowBytes {
		// Slide: keep the most recent rateLimitWindowBytes.
		rs.rlWindow = rs.rlWindow[len(rs.rlWindow)-rateLimitWindowBytes:]
	}
	if now.Sub(rs.rlLastScan) < rateLimitScanCooldown {
		return false
	}
	rs.rlLastScan = now
	return true
}

// rateLimitWindow returns a copy of the current rolling window so the
// scanner can run without holding rlMu during regex evaluation.
func (rs *runningSession) rateLimitWindow() []byte {
	rs.rlMu.Lock()
	defer rs.rlMu.Unlock()
	cp := make([]byte, len(rs.rlWindow))
	copy(cp, rs.rlWindow)
	return cp
}

// clearRateLimitWindow drops the rolling buffer — called after a
// successful failover so the same banner can't be matched twice from
// the bytes that were already in the window before the switch.
func (rs *runningSession) clearRateLimitWindow() {
	rs.rlMu.Lock()
	defer rs.rlMu.Unlock()
	rs.rlWindow = nil
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

// arm marks the session as awaiting a reply turn as of t. The next
// active→quiet transition (see checkTurnComplete) fires exactly one
// session.turn_completed. Idempotent re-arming just moves the marker.
func (rs *runningSession) arm(t time.Time) {
	rs.activityMu.Lock()
	defer rs.activityMu.Unlock()
	rs.expectTurn = true
	rs.expectAt = t
}

// checkTurnComplete reports whether an armed session has just settled
// into a completed reply turn: it's armed, has produced output at or
// after the arming instant, and has now been quiet for >= threshold.
// Returns true exactly once per arming (it disarms on fire) so the
// caller emits a single session.turn_completed.
func (rs *runningSession) checkTurnComplete(now time.Time, threshold time.Duration) bool {
	rs.activityMu.Lock()
	defer rs.activityMu.Unlock()
	if !rs.expectTurn {
		return false
	}
	// No output seen since we armed → the agent hasn't started
	// replying yet; keep waiting (the channel layer's own cap stops a
	// never-answering session from showing "typing…" forever).
	if rs.lastActivity.Before(rs.expectAt) {
		return false
	}
	if now.Sub(rs.lastActivity) >= threshold {
		rs.expectTurn = false
		return true
	}
	return false
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
		turnThreshold: defaultTurnThreshold,
		turnInterval:  defaultTurnInterval,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// ReconcileStartup reconciles DB rows left non-terminal by a prior
// gateway process (their PTYs died with it). Each such row is marked
// 'interrupted', then — unless OPENDRAY_NO_AUTO_RESUME is set — the
// session is re-spawned via Start, which for claude resumes the
// original transcript (--resume <claude_session_id>) so a daemon
// restart (e.g. a self-update) no longer destroys live work. Call once
// after NewManager and before serving traffic; failures to resume a
// single session are logged and skipped, never fatal.
func (m *Manager) ReconcileStartup(ctx context.Context) error {
	// Crash path: a daemon that was SIGKILLed (or died hard) never ran
	// waitExit, so its sessions are still 'running'/'idle'/'pending'.
	// Flip them to 'interrupted'. A clean shutdown already marked its
	// sessions 'interrupted' from waitExit, so nothing to flip there.
	if _, err := m.store.MarkRunningAsInterrupted(ctx); err != nil {
		return err
	}
	// Resume everything left in 'interrupted' — both the rows just
	// flipped above (crash) and those recorded by waitExit during a
	// graceful restart. This is the set that must come back live.
	ids, err := m.store.ListInterrupted(ctx)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	if os.Getenv("OPENDRAY_NO_AUTO_RESUME") != "" {
		m.log.Info("interrupted sessions present on startup; auto-resume disabled",
			"count", len(ids), "reason", "OPENDRAY_NO_AUTO_RESUME set")
		return nil
	}

	// Optional hard cap. ids are newest-first, so a cap keeps the most
	// recent and leaves the rest 'interrupted' (recoverable via an
	// explicit Start). 0 / unset = no cap.
	var skipped int
	if max := autoResumeMaxFromEnv(); max > 0 && len(ids) > max {
		skipped = len(ids) - max
		ids = ids[:max]
	}

	// Resume in the background with bounded concurrency: startup must
	// not block on N PTY spawns, and we must not fan out an unbounded
	// burst at boot. Per-session failures are logged and leave the row
	// 'interrupted' for a later manual / next-boot resume.
	m.log.Info("auto-resuming interrupted sessions in background",
		"count", len(ids), "skipped_over_cap", skipped,
		"concurrency", autoResumeConcurrency)
	go m.resumeInterrupted(ctx, ids)
	return nil
}

// resumeInterrupted re-spawns the given sessions, at most
// autoResumeConcurrency at a time, stopping early if ctx is cancelled
// (gateway shutting down). Runs in its own goroutine off ReconcileStartup.
func (m *Manager) resumeInterrupted(ctx context.Context, ids []string) {
	sem := make(chan struct{}, autoResumeConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var resumed, failed int
	for _, id := range ids {
		if ctx.Err() != nil {
			break // shutting down — stop launching more resumes
		}
		sem <- struct{}{}
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			defer func() { <-sem }()
			if _, err := m.Start(ctx, id); err != nil {
				m.log.Warn("startup auto-resume failed; session left interrupted",
					"session_id", id, "err", err)
				mu.Lock()
				failed++
				mu.Unlock()
				return
			}
			mu.Lock()
			resumed++
			mu.Unlock()
		}(id)
	}
	wg.Wait()
	m.log.Info("startup auto-resume complete", "resumed", resumed, "failed", failed)
}

// autoResumeMaxFromEnv reads OPENDRAY_AUTO_RESUME_MAX; 0/unset/invalid
// means no cap.
func autoResumeMaxFromEnv() int {
	n, err := strconv.Atoi(strings.TrimSpace(os.Getenv("OPENDRAY_AUTO_RESUME_MAX")))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// defaultSessionName derives a friendly label for sessions created
// without an explicit name, so channel surfaces (/list, idle cards)
// show something operators recognise instead of a bare nano id. The
// working-directory basename is the most meaningful default; we fall
// back to the provider id when the cwd has no usable basename (root,
// empty, ".").
func defaultSessionName(providerID, cwd string) string {
	base := filepath.Base(strings.TrimRight(cwd, string(filepath.Separator)))
	switch base {
	case "", ".", string(filepath.Separator):
		return providerID
	}
	return base
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
	name := req.Name
	if name == "" {
		name = defaultSessionName(req.ProviderID, req.Cwd)
	}
	sess := Session{
		ID:              sessID,
		Name:            name,
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
		// On reactivation (Start/resume) carry the existing agent-side
		// UUID into Prepare so the provider emits `--resume <id>` and
		// the prior transcript continues, instead of minting a fresh
		// session and orphaning history.
		prepareCtx := WithSessionID(WithCwd(ctx, sess.Cwd), sess.ID)
		if reactivate {
			prepareCtx = WithResumeClaudeSessionID(prepareCtx, sess.ClaudeSessionID)
		}
		out, err := p.Prepare(prepareCtx, sess.ID, tempDir)
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
	cmd.Env = mergeEnv(ensureColorTerm(os.Environ()), extraEnv)

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
	if m.turnThreshold > 0 {
		m.wg.Add(1)
		go m.turnWatcher(rs)
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
// ensureColorTerm guarantees child CLIs see a color-capable terminal.
// opendray always allocates a real PTY (pty.Start), so the CLIs'
// isatty() check passes — but systemd starts the daemon with no TERM,
// and Node/ink-based CLIs (claude, codex, gemini) fall back to
// monochrome output when TERM is unset. We inject xterm-256color +
// truecolor as defaults only; an explicit TERM/COLORTERM already in
// the environment (or set later by provider config, which mergeEnv
// applies as an override) still wins, and we never touch NO_COLOR so
// an operator who opted out stays opted out.
func ensureColorTerm(env []string) []string {
	var hasTERM, hasCOLORTERM bool
	for _, kv := range env {
		switch {
		case strings.HasPrefix(kv, "TERM="):
			hasTERM = true
		case strings.HasPrefix(kv, "COLORTERM="):
			hasCOLORTERM = true
		}
	}
	if !hasTERM {
		env = append(env, "TERM=xterm-256color")
	}
	if !hasCOLORTERM {
		env = append(env, "COLORTERM=truecolor")
	}
	return env
}

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

// ActiveCountByProvider returns the number of currently non-terminal
// sessions backed by providerID. Iterates the in-memory live map only,
// so it's O(live sessions) and lock-cheap — designed for the catalog
// update-check path that surfaces "N session(s) running on claude" so
// the operator can confirm before swapping the CLI binary underneath
// them.
func (m *Manager) ActiveCountByProvider(providerID string) int {
	if providerID == "" {
		return 0
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	n := 0
	for _, rs := range m.sessions {
		rs.sessMu.RLock()
		if rs.sess.ProviderID == providerID && !rs.sess.State.IsTerminal() {
			n++
		}
		rs.sessMu.RUnlock()
	}
	return n
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

	// Transcript migration: Claude stores per-config-dir transcripts at
	// <CLAUDE_CONFIG_DIR>/projects/<workspace>/<session_id>.jsonl. The
	// respawn below renders a different CLAUDE_CONFIG_DIR than the one
	// the conversation was authored under, so --resume would find
	// nothing and the conversation would start fresh — exactly the
	// "in-progress state lost" UX the dialog warns about.
	//
	// We hard-link the JSONL into the new config dir's projects tree
	// before respawn. Hard-link keeps both views pointing at one inode
	// so future writes by the new account are visible if the operator
	// switches back later. Cross-fs hard-link or missing source falls
	// back to a one-shot copy. Errors are non-fatal: the worst case is
	// what we had before — fresh conversation — and we still bring the
	// session up under the new account so rate-limit-driven switches
	// keep working.
	if m.claudeAccounts != nil && sess.ClaudeSessionID != "" {
		oldCfg, errOld := m.claudeAccounts.ResolveClaudeConfigDir(ctx, current.ClaudeAccountID)
		newCfg, errNew := m.claudeAccounts.ResolveClaudeConfigDir(ctx, newAccountID)
		if errOld == nil && errNew == nil && oldCfg != newCfg {
			if err := migrateClaudeTranscript(oldCfg, newCfg, sess.ClaudeSessionID); err != nil {
				m.log.Warn("claude transcript migration failed; switch will start fresh conversation",
					"session", id, "old_cfg", oldCfg, "new_cfg", newCfg, "err", err)
			}
		}
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

// ExpectTurn arms turn-complete detection for a live session: after
// the caller has submitted a message into the session's stdin, the
// next time the agent produces output and then falls quiet for
// turnThreshold, the manager publishes session.turn_completed. This is
// the seam the channel hub uses to drive a chat "typing…" indicator
// and deliver the reply promptly (rather than waiting for the long
// idle window). No-op on an unknown / terminal session, or when turn
// detection is disabled.
func (m *Manager) ExpectTurn(id string) {
	if m.turnThreshold <= 0 {
		return
	}
	rs := m.lookup(id)
	if rs == nil {
		return
	}
	rs.sessMu.RLock()
	terminal := rs.sess.State.IsTerminal()
	rs.sessMu.RUnlock()
	if terminal {
		return
	}
	rs.arm(time.Now())
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

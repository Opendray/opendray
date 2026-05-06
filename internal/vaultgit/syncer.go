package vaultgit

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/opendray/opendray-v2/internal/eventbus"
)

const (
	minTickInterval = 30 * time.Second
	defaultPoll     = 1 * time.Minute
)

// SyncConfig is the persistent shape of the auto-sync settings. The
// table is single-row (id=1, enforced by CHECK constraint) — updates
// are always UPDATE … WHERE id=1.
type SyncConfig struct {
	Enabled        bool       `json:"enabled"`
	CommitInterval string     `json:"commit_interval"` // human form, e.g. "10m"
	PushEnabled    bool       `json:"push_enabled"`
	PullEnabled    bool       `json:"pull_enabled"`
	PullInterval   string     `json:"pull_interval"`
	CommitMessage  string     `json:"commit_message,omitempty"`
	LastCommitAt   *time.Time `json:"last_commit_at,omitempty"`
	LastCommitHash string     `json:"last_commit_hash,omitempty"`
	LastPushAt     *time.Time `json:"last_push_at,omitempty"`
	LastPullAt     *time.Time `json:"last_pull_at,omitempty"`
	LastError      string     `json:"last_error,omitempty"`
	LastErrorAt    *time.Time `json:"last_error_at,omitempty"`
}

// SyncConfigUpdate is the JSON body for PUT — only fields the client
// can set (timestamps + last_error are server-managed).
type SyncConfigUpdate struct {
	Enabled        *bool   `json:"enabled,omitempty"`
	CommitInterval *string `json:"commit_interval,omitempty"`
	PushEnabled    *bool   `json:"push_enabled,omitempty"`
	PullEnabled    *bool   `json:"pull_enabled,omitempty"`
	PullInterval   *string `json:"pull_interval,omitempty"`
	CommitMessage  *string `json:"commit_message,omitempty"`
}

// Syncer runs the auto-sync loop. Started once per process from
// app.New. The loop wakes on a poll cadence (1 min default) and
// decides whether to commit/push/pull based on the persisted config
// + last-run timestamps. State changes from the UI take effect on
// the next tick — no need to restart.
type Syncer struct {
	pool    *pgxpool.Pool
	bus     *eventbus.Hub
	gitexec *Handlers // reuses run/argsWithAuth/etc — single source of truth
	log     *slog.Logger
	// wakeup is single-buffered so a non-blocking send always succeeds
	// without us needing extra synchronisation.
	wakeup chan struct{}
}

func NewSyncer(pool *pgxpool.Pool, bus *eventbus.Hub, h *Handlers, log *slog.Logger) *Syncer {
	if log == nil {
		log = slog.Default()
	}
	return &Syncer{
		pool:    pool,
		bus:     bus,
		gitexec: h,
		log:     log.With("component", "vaultgit.sync"),
		wakeup:  make(chan struct{}, 1),
	}
}

// Run blocks until ctx is cancelled. Spawned by app.New into a
// goroutine; cleanly exits on graceful shutdown.
func (s *Syncer) Run(ctx context.Context) {
	s.log.Info("vault auto-sync started")
	defer s.log.Info("vault auto-sync stopped")
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		s.tick(ctx)
		select {
		case <-ctx.Done():
			return
		case <-time.After(defaultPoll):
		case <-s.wakeup:
		}
	}
}

// Wake nudges the loop so it doesn't have to wait for the next poll
// tick. Called from the "Run now" endpoint and after config saves.
func (s *Syncer) Wake() {
	select {
	case s.wakeup <- struct{}{}:
	default:
	}
}

func (s *Syncer) tick(ctx context.Context) {
	cfg, err := s.read(ctx)
	if err != nil {
		s.log.Warn("read sync config", "err", err)
		return
	}
	if !cfg.Enabled {
		return
	}
	commitInt, _ := time.ParseDuration(cfg.CommitInterval)
	pullInt, _ := time.ParseDuration(cfg.PullInterval)
	if commitInt < minTickInterval {
		commitInt = minTickInterval
	}
	if pullInt < minTickInterval {
		pullInt = minTickInterval
	}

	now := time.Now()
	// Pull first (so a subsequent commit lands on top of upstream).
	if cfg.PullEnabled {
		if cfg.LastPullAt == nil || now.Sub(*cfg.LastPullAt) >= pullInt {
			s.runPull(ctx)
		}
	}
	// Commit + push when the interval has elapsed AND there's something
	// to commit. We let the commit-no-changes case be a no-op via the
	// DB write so the next tick sees a fresh last_commit_at.
	if cfg.LastCommitAt == nil || now.Sub(*cfg.LastCommitAt) >= commitInt {
		s.runCommit(ctx, cfg.CommitMessage)
		if cfg.PushEnabled {
			s.runPush(ctx)
		}
	}
}

func (s *Syncer) runCommit(ctx context.Context, msg string) {
	cctx, cancel := context.WithTimeout(ctx, gitTimeout)
	defer cancel()
	if !s.gitexec.isRepo(cctx) {
		s.recordErr(ctx, errors.New("vault is not a git repo"))
		return
	}
	// Skip when working tree is clean — avoids "nothing to commit"
	// noise in the error column.
	out, _ := s.gitexec.run(cctx, "status", "--porcelain")
	if len(out) == 0 {
		s.markCommitTimestamp(ctx, "")
		return
	}
	if _, err := s.gitexec.run(cctx, "add", "."); err != nil {
		s.recordErr(ctx, fmt.Errorf("git add: %w", err))
		return
	}
	if msg == "" {
		msg = fmt.Sprintf("Auto-sync: %s", time.Now().Format("2006-01-02 15:04"))
	}
	if _, err := s.gitexec.run(cctx, "commit", "-m", msg); err != nil {
		s.recordErr(ctx, fmt.Errorf("git commit: %w", err))
		return
	}
	hashOut, _ := s.gitexec.run(cctx, "rev-parse", "--short", "HEAD")
	hash := trimSpace(string(hashOut))
	s.markCommitTimestamp(ctx, hash)
	s.bus.Publish(eventbus.Event{
		Topic: "vault.auto_sync.committed",
		Data:  map[string]any{"hash": hash, "message": msg},
	})
}

func (s *Syncer) runPush(ctx context.Context) {
	cctx, cancel := context.WithTimeout(ctx, gitTimeout)
	defer cancel()
	args, err := s.gitexec.argsWithAuth(cctx, "push", "-u", "origin", "HEAD")
	if err != nil {
		s.recordErr(ctx, err)
		return
	}
	if out, err := s.gitexec.run(cctx, args...); err != nil {
		s.recordErr(ctx, fmt.Errorf("git push: %w (%s)", err, trimSpace(string(out))))
		return
	}
	s.markPushTimestamp(ctx)
	s.bus.Publish(eventbus.Event{Topic: "vault.auto_sync.pushed"})
}

func (s *Syncer) runPull(ctx context.Context) {
	cctx, cancel := context.WithTimeout(ctx, gitTimeout)
	defer cancel()
	pullArgs := []string{"pull", "--rebase", "--autostash"}
	if !s.gitexec.hasUpstream(cctx) {
		// Pull origin <branch> explicitly when no tracking is set up.
		branch := trimSpace(string(must(s.gitexec.run(cctx, "rev-parse", "--abbrev-ref", "HEAD"))))
		if branch == "" || branch == "HEAD" {
			branch = "main"
		}
		pullArgs = append(pullArgs, "origin", branch)
	}
	args, err := s.gitexec.argsWithAuth(cctx, pullArgs...)
	if err != nil {
		s.recordErr(ctx, err)
		return
	}
	if out, err := s.gitexec.run(cctx, args...); err != nil {
		s.recordErr(ctx, fmt.Errorf("git pull: %w (%s)", err, trimSpace(string(out))))
		return
	}
	s.markPullTimestamp(ctx)
	s.bus.Publish(eventbus.Event{Topic: "vault.auto_sync.pulled"})
}

// ── Persistence ────────────────────────────────────────────────

func (s *Syncer) read(ctx context.Context) (SyncConfig, error) {
	row := s.pool.QueryRow(ctx, `
        SELECT enabled, commit_interval_ms, push_enabled,
               pull_enabled, pull_interval_ms, commit_message,
               last_commit_at, last_commit_hash,
               last_push_at, last_pull_at, last_error, last_error_at
        FROM vault_sync_config WHERE id = 1`)
	var (
		c                SyncConfig
		commitMs, pullMs int64
		commitAt, pushAt sql.NullTime
		pullAt, errAt    sql.NullTime
		hashStr, errStr  sql.NullString
		msgStr           sql.NullString
	)
	if err := row.Scan(
		&c.Enabled, &commitMs, &c.PushEnabled,
		&c.PullEnabled, &pullMs, &msgStr,
		&commitAt, &hashStr,
		&pushAt, &pullAt, &errStr, &errAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SyncConfig{}, fmt.Errorf("vault_sync_config row missing — migration not applied?")
		}
		return SyncConfig{}, err
	}
	c.CommitInterval = time.Duration(commitMs * int64(time.Millisecond)).String()
	c.PullInterval = time.Duration(pullMs * int64(time.Millisecond)).String()
	if msgStr.Valid {
		c.CommitMessage = msgStr.String
	}
	if commitAt.Valid {
		t := commitAt.Time
		c.LastCommitAt = &t
	}
	if hashStr.Valid {
		c.LastCommitHash = hashStr.String
	}
	if pushAt.Valid {
		t := pushAt.Time
		c.LastPushAt = &t
	}
	if pullAt.Valid {
		t := pullAt.Time
		c.LastPullAt = &t
	}
	if errStr.Valid {
		c.LastError = errStr.String
	}
	if errAt.Valid {
		t := errAt.Time
		c.LastErrorAt = &t
	}
	return c, nil
}

func (s *Syncer) write(ctx context.Context, u SyncConfigUpdate) (SyncConfig, error) {
	cur, err := s.read(ctx)
	if err != nil {
		return SyncConfig{}, err
	}
	if u.Enabled != nil {
		cur.Enabled = *u.Enabled
	}
	if u.PushEnabled != nil {
		cur.PushEnabled = *u.PushEnabled
	}
	if u.PullEnabled != nil {
		cur.PullEnabled = *u.PullEnabled
	}
	if u.CommitMessage != nil {
		cur.CommitMessage = *u.CommitMessage
	}
	commitMs := durationMs(cur.CommitInterval, 600_000)
	if u.CommitInterval != nil {
		commitMs = parseDurationMs(*u.CommitInterval, commitMs)
	}
	pullMs := durationMs(cur.PullInterval, 3_600_000)
	if u.PullInterval != nil {
		pullMs = parseDurationMs(*u.PullInterval, pullMs)
	}

	if _, err := s.pool.Exec(ctx, `
        UPDATE vault_sync_config
        SET enabled=$1, commit_interval_ms=$2, push_enabled=$3,
            pull_enabled=$4, pull_interval_ms=$5, commit_message=$6,
            updated_at=NOW()
        WHERE id=1`,
		cur.Enabled, commitMs, cur.PushEnabled,
		cur.PullEnabled, pullMs, cur.CommitMessage,
	); err != nil {
		return SyncConfig{}, err
	}
	s.Wake()
	return s.read(ctx)
}

func (s *Syncer) markCommitTimestamp(ctx context.Context, hash string) {
	_, err := s.pool.Exec(ctx, `
        UPDATE vault_sync_config
        SET last_commit_at=NOW(),
            last_commit_hash=NULLIF($1, ''),
            last_error=NULL, last_error_at=NULL
        WHERE id=1`, hash)
	if err != nil {
		s.log.Warn("update last_commit_at", "err", err)
	}
}

func (s *Syncer) markPushTimestamp(ctx context.Context) {
	_, err := s.pool.Exec(ctx, `
        UPDATE vault_sync_config
        SET last_push_at=NOW(), last_error=NULL, last_error_at=NULL
        WHERE id=1`)
	if err != nil {
		s.log.Warn("update last_push_at", "err", err)
	}
}

func (s *Syncer) markPullTimestamp(ctx context.Context) {
	_, err := s.pool.Exec(ctx, `
        UPDATE vault_sync_config
        SET last_pull_at=NOW(), last_error=NULL, last_error_at=NULL
        WHERE id=1`)
	if err != nil {
		s.log.Warn("update last_pull_at", "err", err)
	}
}

func (s *Syncer) recordErr(ctx context.Context, srcErr error) {
	s.log.Warn("auto-sync step failed", "err", srcErr)
	_, err := s.pool.Exec(ctx, `
        UPDATE vault_sync_config
        SET last_error=$1, last_error_at=NOW()
        WHERE id=1`, srcErr.Error())
	if err != nil {
		s.log.Warn("update last_error", "err", err)
	}
}

// ── HTTP wiring (mounted by Handlers.Mount) ─────────────────────

// MountSync attaches the sync endpoints under the same /vault/git
// chi route group used by the rest of the package. Called by app.New
// after constructing the Syncer.
func (s *Syncer) Mount(r chi.Router) {
	r.Route("/vault/git/sync", func(r chi.Router) {
		r.Get("/config", s.handleGetConfig)
		r.Put("/config", s.handlePutConfig)
		r.Post("/run", s.handleRun)
	})
}

func (s *Syncer) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.read(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

func (s *Syncer) handlePutConfig(w http.ResponseWriter, r *http.Request) {
	var u SyncConfigUpdate
	if err := json.NewDecoder(io.LimitReader(r.Body, 16<<10)).Decode(&u); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	cfg, err := s.write(r.Context(), u)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

func (s *Syncer) handleRun(w http.ResponseWriter, _ *http.Request) {
	s.Wake()
	writeJSON(w, http.StatusOK, map[string]string{"status": "scheduled"})
}

// ── helpers ────────────────────────────────────────────────────

func parseDurationMs(s string, fallback int64) int64 {
	d, err := time.ParseDuration(s)
	if err != nil {
		return fallback
	}
	if d < minTickInterval {
		d = minTickInterval
	}
	return d.Milliseconds()
}

func durationMs(s string, fallback int64) int64 {
	d, err := time.ParseDuration(s)
	if err != nil {
		return fallback
	}
	return d.Milliseconds()
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\n' || s[0] == '\t' || s[0] == '\r') {
		s = s[1:]
	}
	for len(s) > 0 {
		l := len(s) - 1
		if c := s[l]; c == ' ' || c == '\n' || c == '\t' || c == '\r' {
			s = s[:l]
		} else {
			break
		}
	}
	return s
}

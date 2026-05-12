package gitactivity

import (
	"context"
	"log/slog"
	"time"

	"github.com/opendray/opendray-v2/internal/memory"
)

// ScopeKeyLister is the dependency the scheduler needs to know
// which cwds deserve a periodic refresh. Backed by
// memory.Service.ListScopeKeys at runtime; defined as an interface
// so tests can stub it.
type ScopeKeyLister interface {
	ListScopeKeys(ctx context.Context, scope memory.Scope) ([]string, error)
}

// Scheduler periodically fires Service.Run across every project
// scope that has at least one stored memory or doc. Same loop
// shape as the cleaner / capture engine schedulers — gives the
// recent_activity doc a refresh cadence operators don't have to
// think about.
type Scheduler struct {
	svc    *Service
	scopes ScopeKeyLister
	cfg    SchedulerConfig
	log    *slog.Logger
}

// SchedulerConfig drives the tick cadence.
type SchedulerConfig struct {
	// Interval between full sweeps. Default 24h. Set to 0 to
	// disable auto-runs (operator triggers POST /git-activity/run
	// manually).
	Interval time.Duration

	// InitialDelay before the first sweep. Default 10min — gives
	// the boot sequence time to settle and the cipher to arm
	// before any LLM call.
	InitialDelay time.Duration

	// MaxAge — a per-cwd staleness threshold. If the existing
	// recent_activity doc is younger than this, the sweep skips
	// it. Default 12h — give the cron a chance to run twice a
	// day without re-running each cwd more than once between
	// commits.
	MaxAge time.Duration
}

func (c SchedulerConfig) applyDefaults() SchedulerConfig {
	if c.Interval <= 0 {
		c.Interval = 24 * time.Hour
	}
	if c.InitialDelay <= 0 {
		c.InitialDelay = 10 * time.Minute
	}
	if c.MaxAge <= 0 {
		c.MaxAge = 12 * time.Hour
	}
	return c
}

// NewScheduler wires a scheduler. svc may be nil — the run loop
// just no-ops in that case so callers can pass an unconditional
// Run() to a goroutine.
func NewScheduler(svc *Service, scopes ScopeKeyLister, cfg SchedulerConfig, log *slog.Logger) *Scheduler {
	if log == nil {
		log = slog.Default()
	}
	return &Scheduler{
		svc:    svc,
		scopes: scopes,
		cfg:    cfg.applyDefaults(),
		log:    log.With("component", "gitactivity.scheduler"),
	}
}

// Run blocks until ctx is cancelled. Per tick: list project
// scope_keys, skip cwds with a fresh-enough doc, call Service.Run
// on the rest. LLM failures inside Service.Run are logged there
// and degrade to raw stats — they never bubble up to abort the
// sweep.
func (s *Scheduler) Run(ctx context.Context) {
	if s.svc == nil {
		s.log.Info("gitactivity scheduler: no service wired; idle")
		<-ctx.Done()
		return
	}
	if s.cfg.Interval <= 0 {
		s.log.Info("gitactivity scheduler: interval=0; auto-runs disabled")
		<-ctx.Done()
		return
	}
	s.log.Info("gitactivity scheduler running",
		"interval", s.cfg.Interval,
		"initial_delay", s.cfg.InitialDelay,
		"max_age", s.cfg.MaxAge)

	select {
	case <-ctx.Done():
		return
	case <-time.After(s.cfg.InitialDelay):
	}

	t := time.NewTicker(s.cfg.Interval)
	defer t.Stop()
	s.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			s.log.Info("gitactivity scheduler stopping")
			return
		case <-t.C:
			s.tick(ctx)
		}
	}
}

func (s *Scheduler) tick(ctx context.Context) {
	keys, err := s.scopes.ListScopeKeys(ctx, memory.ScopeProject)
	if err != nil {
		s.log.Warn("scheduler.list_scope_keys_failed", "err", err)
		return
	}
	s.log.Info("scheduler.tick", "project_keys", len(keys))
	for _, k := range keys {
		if k == "" {
			continue
		}
		if !s.svc.IsStale(ctx, k, s.cfg.MaxAge) {
			s.log.Debug("scheduler.skip_fresh", "scope_key", k)
			continue
		}
		if _, err := s.svc.Run(ctx, k); err != nil {
			s.log.Warn("scheduler.run_failed", "scope_key", k, "err", err)
		}
	}
}

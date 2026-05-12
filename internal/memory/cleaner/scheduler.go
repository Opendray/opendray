package cleaner

import (
	"context"
	"log/slog"
	"time"

	"github.com/opendray/opendray-v2/internal/memory"
)

// ScopeKeyLister is the dependency the scheduler needs to know
// which (scope, scope_key) pairs deserve a periodic sweep. Backed
// by memory.Service.ListScopeKeys at runtime; defined as an
// interface so tests can stub the iteration.
type ScopeKeyLister interface {
	ListScopeKeys(ctx context.Context, scope memory.Scope) ([]string, error)
}

// Scheduler periodically fires Service.Run across every project
// scope that has memories. Global scope sweeps once per interval
// with an empty scope_key.
//
// Design notes:
//
//   - Only project scope is auto-swept. Session-scope memories are
//     short-lived (one session) and the auto-cleanup signal is
//     unreliable for them; operators trim those via the inspector.
//   - One LLM call per scope_key per tick. We don't fan out
//     concurrently because the LLM is the bottleneck and concurrent
//     calls would just queue at the provider.
//   - Failures per scope are logged and the loop continues — never
//     block the next project on this one's outage.
type Scheduler struct {
	svc    *Service
	scopes ScopeKeyLister
	cfg    SchedulerConfig
	log    *slog.Logger
}

// SchedulerConfig controls when + how often the scheduler fires.
type SchedulerConfig struct {
	// Interval between full sweeps. Default 24h. Set to 0 to
	// disable auto-runs (operator must POST /memory/cleanup/run).
	Interval time.Duration

	// InitialDelay before the first sweep. Default 5min so the
	// process has time to warm up (mainly: cipher armed, LLM
	// provider reachable) before we start judging anything.
	InitialDelay time.Duration

	// IncludeGlobalScope sweeps the global scope alongside every
	// project. Default false — global memories are typically
	// user-curated and operators don't want a librarian touching
	// them. Operators flip this on once they trust the cleaner.
	IncludeGlobalScope bool
}

func (c SchedulerConfig) applyDefaults() SchedulerConfig {
	if c.Interval <= 0 {
		c.Interval = 24 * time.Hour
	}
	if c.InitialDelay <= 0 {
		c.InitialDelay = 5 * time.Minute
	}
	return c
}

// NewScheduler wires a scheduler. Returns nil + the same nil result
// when svc is nil so the caller can pass an unconditional Run() and
// the scheduler just becomes a no-op.
func NewScheduler(svc *Service, scopes ScopeKeyLister, cfg SchedulerConfig, log *slog.Logger) *Scheduler {
	if log == nil {
		log = slog.Default()
	}
	return &Scheduler{
		svc:    svc,
		scopes: scopes,
		cfg:    cfg.applyDefaults(),
		log:    log.With("component", "memory.cleaner.scheduler"),
	}
}

// Run blocks until ctx is cancelled. Each tick:
//
//  1. Wait for the next interval (or initial delay on first tick).
//  2. List project scope_keys (and global if enabled).
//  3. For each scope_key, call svc.Run sequentially. Errors are
//     logged and the loop continues.
//
// Returns nil; failures don't propagate up to crash the goroutine.
func (s *Scheduler) Run(ctx context.Context) {
	if s.svc == nil {
		s.log.Info("scheduler: no cleaner service wired; sleeping forever")
		<-ctx.Done()
		return
	}
	if s.cfg.Interval <= 0 {
		s.log.Info("scheduler: interval = 0; auto-runs disabled")
		<-ctx.Done()
		return
	}
	s.log.Info("scheduler running",
		"interval", s.cfg.Interval,
		"initial_delay", s.cfg.InitialDelay,
		"include_global", s.cfg.IncludeGlobalScope)

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
			s.log.Info("scheduler stopping")
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
		if res, err := s.svc.Run(ctx, memory.ScopeProject, k); err != nil {
			s.log.Warn("scheduler.run_failed", "scope_key", k, "err", err)
		} else {
			s.log.Debug("scheduler.run_ok",
				"scope_key", k,
				"memories_in", res.MemoriesIn,
				"decisions_out", res.DecisionsOut)
		}
	}
	if s.cfg.IncludeGlobalScope {
		if res, err := s.svc.Run(ctx, memory.ScopeGlobal, ""); err != nil {
			s.log.Warn("scheduler.global_run_failed", "err", err)
		} else {
			s.log.Debug("scheduler.global_run_ok",
				"memories_in", res.MemoriesIn,
				"decisions_out", res.DecisionsOut)
		}
	}
}

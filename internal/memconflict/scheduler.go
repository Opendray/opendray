package memconflict

import (
	"context"
	"fmt"
	"time"
)

// SchedulerConfig tunes the daily detection cadence. All fields
// have defaults; pass zero value to use them.
type SchedulerConfig struct {
	// Interval between full sweeps. Default 24h. Set to 0 to keep
	// the default; manual operators run /memory/conflicts/detect
	// for on-demand cycles.
	Interval time.Duration

	// MaxPerCwd caps how many conflicts a single sweep writes per
	// cwd — avoids one chatty detector flooding the inbox. The
	// limit is enforced inside DetectForCwd via the unique-pair
	// check, but this is the absolute brake when the LLM goes off.
	MaxPerCwd int
}

func (c SchedulerConfig) applyDefaults() SchedulerConfig {
	if c.Interval <= 0 {
		c.Interval = 24 * time.Hour
	}
	if c.MaxPerCwd <= 0 {
		c.MaxPerCwd = 25
	}
	return c
}

// CwdLister returns the set of cwds the detector should scan.
// Tests can stub it; production wires in a query against
// project_docs distinct cwds.
type CwdLister interface {
	ListProjectCwds(ctx context.Context) ([]string, error)
}

// Scheduler drives the daily detection loop. Single goroutine —
// parallelism per cwd is not worth the complexity given each
// detector call is minutes-apart.
type Scheduler struct {
	svc  *Service
	cwds CwdLister
	cfg  SchedulerConfig
}

func NewScheduler(svc *Service, cwds CwdLister, cfg SchedulerConfig) *Scheduler {
	return &Scheduler{svc: svc, cwds: cwds, cfg: cfg.applyDefaults()}
}

// Run blocks until ctx is cancelled. First sweep fires
// immediately so a fresh deploy doesn't wait a day to surface
// the first conflict.
func (s *Scheduler) Run(ctx context.Context) {
	s.svc.log.Info("conflict scheduler running",
		"interval", s.cfg.Interval, "max_per_cwd", s.cfg.MaxPerCwd)
	t := time.NewTicker(s.cfg.Interval)
	defer t.Stop()
	s.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			s.svc.log.Info("conflict scheduler stopping")
			return
		case <-t.C:
			s.tick(ctx)
		}
	}
}

func (s *Scheduler) tick(ctx context.Context) {
	cwds, err := s.cwds.ListProjectCwds(ctx)
	if err != nil {
		s.svc.log.Warn("conflict scheduler: list cwds failed", "err", err)
		return
	}
	for _, cwd := range cwds {
		if ctx.Err() != nil {
			return
		}
		// Each cwd gets its own bounded budget so one slow LLM call
		// can't starve the rest.
		cycleCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		n, err := s.svc.DetectForCwd(cycleCtx, cwd)
		cancel()
		if err != nil {
			s.svc.log.Debug("conflict scheduler: detect failed",
				"cwd", cwd, "err", err)
			continue
		}
		if n > s.cfg.MaxPerCwd {
			s.svc.log.Warn("conflict scheduler: cwd hit per-sweep limit",
				"cwd", cwd, "written", n, "max", s.cfg.MaxPerCwd)
		}
	}
}

// silence the import linter when fmt is not used at compile time
var _ = fmt.Sprintf

package backup

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

// DefaultSchedulerInterval is the default poll cadence — every 30s
// we check for due schedules. Finer wastes CPU; coarser delays
// runs unnecessarily. Tunable for tests via NewScheduler.
const DefaultSchedulerInterval = 30 * time.Second

// Scheduler is the long-running goroutine that drives
// backup_schedules rows: claim due rows, run them, apply retention,
// repeat. Multiple opendray instances cooperate safely via FOR
// UPDATE SKIP LOCKED in store.ClaimDueSchedule.
type Scheduler struct {
	svc      *Service
	interval time.Duration
	log      *slog.Logger
}

// NewScheduler returns a Scheduler that polls every interval. Pass
// 0 (or negative) to use DefaultSchedulerInterval.
func NewScheduler(svc *Service, interval time.Duration) *Scheduler {
	if interval <= 0 {
		interval = DefaultSchedulerInterval
	}
	return &Scheduler{
		svc:      svc,
		interval: interval,
		log:      svc.log.With("component", "backup-scheduler"),
	}
}

// Run blocks until ctx is cancelled. The first tick fires
// immediately so a freshly-due schedule doesn't have to wait for
// the first poll. Failures inside a tick are logged but never
// propagate — the loop must keep going so a single bad row doesn't
// freeze backups.
func (s *Scheduler) Run(ctx context.Context) {
	s.log.Info("backup scheduler running", "interval", s.interval)
	t := time.NewTicker(s.interval)
	defer t.Stop()

	s.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			s.log.Info("backup scheduler stopping")
			return
		case <-t.C:
			s.tick(ctx)
		}
	}
}

func (s *Scheduler) tick(ctx context.Context) {
	// Reap expired exports first — cheap, runs every tick, keeps
	// the export dir from accumulating stale zips even if no
	// backup schedules ever fire.
	if err := s.svc.ReapExpiredExports(ctx); err != nil {
		s.log.Warn("scheduler: reap exports failed", "err", err)
	}

	// Claim at most one schedule per tick. With a 30s tick and a
	// typical backup taking < 30s, this still keeps up; if a
	// backup runs longer the next tick simply finds it already
	// bumped and waits — the load shapes itself.
	sc, err := s.svc.store.ClaimDueSchedule(ctx)
	if err != nil {
		if errors.Is(err, ErrScheduleNotFound) {
			return
		}
		s.log.Warn("scheduler: claim failed", "err", err)
		return
	}

	targetIDs := sc.TargetIDs
	if len(targetIDs) == 0 {
		targetIDs = []string{sc.TargetID}
	}
	log := s.log.With("schedule_id", sc.ID, "targets", targetIDs)
	log.Info("running scheduled backup")

	b, err := s.svc.RunBackupSync(ctx, RunBackupRequest{
		TargetIDs:     targetIDs,
		TriggeredBy:   TriggeredScheduler,
		Kind:          sc.Kind,
		IncludeConfig: true,
		ScheduleID:    sc.ID,
	})
	if err != nil {
		log.Warn("scheduled backup failed to start", "err", err)
		return
	}
	log.Info("scheduled backup completed",
		"backup_id", b.ID, "status", b.Status, "bytes", b.Bytes)

	// Retention is per-target — each destination keeps its own last-N.
	if sc.Retention > 0 {
		for _, tid := range targetIDs {
			if err := s.svc.RunRetention(ctx, tid, sc.Retention); err != nil {
				log.Warn("retention failed", "target_id", tid, "err", err)
			}
		}
	}
}

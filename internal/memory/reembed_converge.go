package memory

import (
	"context"
	"time"
)

// ReembedConvergeConfig tunes the automatic re-embed convergence loop
// (M-U Phase 6). Zero values fall back to the defaults in withDefaults.
type ReembedConvergeConfig struct {
	// BatchSize is the row count per internal Reembed batch.
	BatchSize int
	// Interval is the pause after a pass that made progress — there may
	// be more rows, so we cycle again soon but not instantly.
	Interval time.Duration
	// IdleInterval is the pause when already converged or when a pass
	// made no progress (e.g. the embedder is unreachable), so the loop
	// doesn't hammer a broken endpoint.
	IdleInterval time.Duration
}

func (c ReembedConvergeConfig) withDefaults() ReembedConvergeConfig {
	if c.BatchSize <= 0 {
		c.BatchSize = 64
	}
	if c.Interval <= 0 {
		c.Interval = 2 * time.Second
	}
	if c.IdleInterval <= 0 {
		c.IdleInterval = 5 * time.Minute
	}
	return c
}

// RunReembedConverge is the M-U Phase 6 background loop that makes an
// embedder change converge automatically.
//
// pgvector similarity is partitioned by (embedder, dim): a memory is
// only comparable to the query when its `embedder` column matches the
// active embedder (see PgvectorStore.Search's `WHERE embedder = $2`).
// So the moment an operator switches the configured embedder, every
// pre-existing row silently drops out of recall until it is
// re-embedded. Before Phase 6 that required clicking a "Migrate"
// button (POST /reembed). This loop removes the manual step: it
// detects the drift and re-embeds in the background until every row
// carries the current embedder, restoring recall on its own. The
// manual endpoint stays as an on-demand trigger.
//
// Resumable + rate-limited: Reembed batches internally and advances a
// cursor across passes, and we sleep between passes so the loop never
// hogs the embedder. Self-skips when there's nothing to converge
// (the common steady state). Soft-fails: a pass that makes no progress
// (e.g. embedder outage) backs off to IdleInterval rather than
// spinning. Launched once from the composition root; blocks until ctx
// is cancelled, like the embed backfills.
func (s *Service) RunReembedConverge(ctx context.Context, cfg ReembedConvergeConfig) {
	cfg = cfg.withDefaults()
	s.log.Info("memory.reembed_converge running",
		"batch", cfg.BatchSize, "interval", cfg.Interval, "idle_interval", cfg.IdleInterval)

	for {
		progressed := false
		drift, err := s.driftCount(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			s.log.Debug("memory.reembed_converge drift check failed", "err", err)
		} else if drift > 0 {
			s.log.Info("memory.reembed_converge drift detected; converging",
				"rows", drift, "to", s.emb.Name())
			report, rerr := s.Reembed(ctx, cfg.BatchSize)
			if rerr != nil && ctx.Err() == nil {
				s.log.Warn("memory.reembed_converge pass failed", "err", rerr)
			}
			progressed = report.Reembed > 0
		}

		delay := cfg.IdleInterval
		if progressed {
			delay = cfg.Interval
		}
		select {
		case <-ctx.Done():
			s.log.Info("memory.reembed_converge stopping")
			return
		case <-time.After(delay):
		}
	}
}

// driftCount returns how many stored memories carry an embedder name
// other than the active one — rows invisible to search until they are
// re-embedded. Cheap: one COUNT(*) GROUP BY via CountByEmbedder.
func (s *Service) driftCount(ctx context.Context) (int, error) {
	counts, err := s.store.CountByEmbedder(ctx)
	if err != nil {
		return 0, err
	}
	current := s.emb.Name()
	drift := 0
	for name, c := range counts {
		if name != current {
			drift += c
		}
	}
	return drift, nil
}

package projectdoc

import (
	"context"
	"fmt"
	"time"
)

// LogEmbedBackfillConfig tunes the M-PB backfill worker. All
// fields have sensible defaults — pass a zero value to use them.
type LogEmbedBackfillConfig struct {
	// BatchSize is how many rows the worker reads + embeds per
	// cycle. 50 balances embed latency (one HTTP call per cycle if
	// the embedder is HTTP-backed) against memory pressure (each
	// row carries title + content of arbitrary length).
	BatchSize int

	// Interval is the delay between cycles when the previous cycle
	// found rows to embed. We keep moving while there's work.
	Interval time.Duration

	// IdleInterval is the delay between cycles when the previous
	// cycle found nothing. Longer to avoid spinning when the
	// journal is caught up.
	IdleInterval time.Duration
}

func (c LogEmbedBackfillConfig) applyDefaults() LogEmbedBackfillConfig {
	if c.BatchSize <= 0 {
		c.BatchSize = 50
	}
	if c.Interval <= 0 {
		c.Interval = 5 * time.Second
	}
	if c.IdleInterval <= 0 {
		c.IdleInterval = 5 * time.Minute
	}
	return c
}

// RunLogEmbedBackfill drives the M-PB journal embedding catch-up
// loop. Designed to run as a single goroutine launched by the
// composition root; blocks until ctx is cancelled.
//
// Each cycle:
//  1. SELECT up to BatchSize rows where embedding IS NULL,
//     prioritising newest first so live searches benefit before
//     ancient backlog.
//  2. Call svc.embedder.Embed on the batch.
//  3. UPDATE each row with its vector + embedder + timestamp,
//     individually so one malformed row doesn't sink the batch.
//
// Soft-fail at every step — backfill is a non-critical
// optimisation; an embedder outage logs and tries again next tick.
func (s *Service) RunLogEmbedBackfill(ctx context.Context, cfg LogEmbedBackfillConfig) {
	cfg = cfg.applyDefaults()
	if s.embedder == nil {
		s.log.Info("projectdoc: no embedder wired; skipping log embed backfill")
		return
	}
	s.log.Info("projectdoc: log embed backfill running",
		"batch", cfg.BatchSize, "interval", cfg.Interval, "idle_interval", cfg.IdleInterval)

	for {
		processed, err := s.backfillOneCycle(ctx, cfg.BatchSize)
		if err != nil {
			s.log.Warn("projectdoc: backfill cycle failed", "err", err)
		}
		delay := cfg.Interval
		if processed == 0 {
			delay = cfg.IdleInterval
		}
		select {
		case <-ctx.Done():
			s.log.Info("projectdoc: log embed backfill stopping")
			return
		case <-time.After(delay):
		}
	}
}

// backfillOneCycle reads one batch, embeds it, writes the vectors
// back. Returns the number of rows successfully embedded so the
// caller can pick the next sleep duration.
func (s *Service) backfillOneCycle(ctx context.Context, batch int) (int, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, title, content
		  FROM session_logs
		 WHERE embedding IS NULL
		 ORDER BY created_at DESC
		 LIMIT $1`, batch)
	if err != nil {
		return 0, fmt.Errorf("scan needs_embed: %w", err)
	}
	type pending struct {
		id   string
		text string
	}
	var todo []pending
	for rows.Next() {
		var p pending
		var title, content string
		if err := rows.Scan(&p.id, &title, &content); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan row: %w", err)
		}
		entry := LogEntry{Title: title, Content: content}
		p.text = embedTextForLog(entry)
		todo = append(todo, p)
	}
	rows.Close()
	if len(todo) == 0 {
		return 0, nil
	}

	texts := make([]string, len(todo))
	for i, p := range todo {
		texts[i] = p.text
	}
	vecs, err := s.embedder.Embed(ctx, texts)
	if err != nil {
		return 0, fmt.Errorf("embed batch: %w", err)
	}
	if len(vecs) != len(todo) {
		return 0, fmt.Errorf("embedder returned %d vectors for %d inputs", len(vecs), len(todo))
	}

	name := s.embedder.Name()
	written := 0
	for i, p := range todo {
		if len(vecs[i]) == 0 {
			// Empty vector — could be the embedder declining (e.g.
			// the text was empty after trimming). Mark with the
			// embedder name + NULL embedding so we don't keep
			// retrying; the predicate `embedding IS NULL` excludes
			// it from future cycles. Use a length-1 zero vector so
			// the predicate flips without consuming index space.
			if _, err := s.pool.Exec(ctx, `
				UPDATE session_logs
				   SET embedding = $1,
				       embedder = $2,
				       embedding_at = NOW()
				 WHERE id = $3`, "[0]", name, p.id); err != nil {
				s.log.Debug("projectdoc: backfill mark-empty failed",
					"log_id", p.id, "err", err)
			}
			continue
		}
		if _, err := s.pool.Exec(ctx, `
			UPDATE session_logs
			   SET embedding = $1,
			       embedder = $2,
			       embedding_at = NOW()
			 WHERE id = $3`, pgvecString(vecs[i]), name, p.id); err != nil {
			s.log.Debug("projectdoc: backfill write failed",
				"log_id", p.id, "err", err)
			continue
		}
		written++
	}
	if written > 0 {
		s.log.Info("projectdoc: log embed backfill cycle done",
			"written", written, "batch", len(todo))
	}
	return written, nil
}

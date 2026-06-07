package projectdoc

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// RunDocEmbedBackfill catches up goal / plan docs that lack an
// embedding — pre-Phase-2 rows, or rows whose write-time embed failed
// (the upsert clears the vector to NULL on every content change, so a
// failed synchronous embed lands here). Mirrors RunLogEmbedBackfill:
// single goroutine launched by the composition root, blocks until ctx
// is cancelled, soft-fails at every step.
//
// project_docs is tiny (a handful of rows per cwd), so this loop is
// almost always idle; it exists to guarantee eventual consistency, not
// throughput. Reuses LogEmbedBackfillConfig for the same knobs.
func (s *Service) RunDocEmbedBackfill(ctx context.Context, cfg LogEmbedBackfillConfig) {
	cfg = cfg.applyDefaults()
	if s.embedder == nil {
		s.log.Info("projectdoc: no embedder wired; skipping doc embed backfill")
		return
	}
	s.log.Info("projectdoc: doc embed backfill running",
		"batch", cfg.BatchSize, "interval", cfg.Interval, "idle_interval", cfg.IdleInterval)

	for {
		processed, err := s.backfillDocsOneCycle(ctx, cfg.BatchSize)
		if err != nil {
			s.log.Warn("projectdoc: doc backfill cycle failed", "err", err)
		}
		delay := cfg.Interval
		if processed == 0 {
			delay = cfg.IdleInterval
		}
		select {
		case <-ctx.Done():
			s.log.Info("projectdoc: doc embed backfill stopping")
			return
		case <-time.After(delay):
		}
	}
}

// backfillDocsOneCycle reads one batch of un-embedded goal/plan docs,
// embeds them, writes the vectors back individually. Returns the number
// of rows successfully embedded so the caller picks the next sleep.
func (s *Service) backfillDocsOneCycle(ctx context.Context, batch int) (int, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, content
		  FROM project_docs
		 WHERE embedding IS NULL
		   AND kind IN ('goal', 'plan')
		 ORDER BY updated_at DESC
		 LIMIT $1`, batch)
	if err != nil {
		return 0, fmt.Errorf("scan needs_embed docs: %w", err)
	}
	type pending struct {
		id   string
		text string
	}
	var todo []pending
	for rows.Next() {
		var p pending
		var content string
		if err := rows.Scan(&p.id, &content); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan doc row: %w", err)
		}
		p.text = strings.TrimSpace(content)
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
		return 0, fmt.Errorf("embed doc batch: %w", err)
	}
	if len(vecs) != len(todo) {
		return 0, fmt.Errorf("embedder returned %d vectors for %d inputs", len(vecs), len(todo))
	}

	name := s.embedder.Name()
	written := 0
	for i, p := range todo {
		vec := "[0]" // empty/declined -> length-1 zero vec flips the NULL predicate so we stop retrying
		if len(vecs[i]) > 0 {
			vec = pgvecString(vecs[i])
		}
		if _, err := s.pool.Exec(ctx, `
			UPDATE project_docs
			   SET embedding = $1,
			       embedder = $2,
			       embedding_at = NOW()
			 WHERE id = $3`, vec, name, p.id); err != nil {
			s.log.Debug("projectdoc: doc backfill write failed", "doc_id", p.id, "err", err)
			continue
		}
		if len(vecs[i]) > 0 {
			written++
		}
	}
	if written > 0 {
		s.log.Info("projectdoc: doc embed backfill cycle done", "written", written, "batch", len(todo))
	}
	return written, nil
}

// Package memhealth aggregates "is the memory system actually
// working?" metrics across both memory subsystems (internal/memory
// and internal/projectdoc) and exposes them through one HTTP
// endpoint backing the /memory landing-page dashboard.
//
// Why a separate package: the data spans two domains (layer-5 facts
// + layer-2-4 docs/journal/proposals) and adding the aggregation
// queries to either would create unwanted cross-imports. This
// package only depends on pgxpool — the same shape every existing
// store wrapper uses — and is consumed by an HTTP handler in
// internal/app.
package memhealth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LookbackDays is the moving window we summarise activity over. 7
// days is short enough to feel current and long enough that a
// slowly-iterating project still sees non-zero numbers.
const LookbackDays = 7

// Snapshot is one read of every panel on the memory health card.
// JSON-tagged so the HTTP handler can ship it without translation.
type Snapshot struct {
	Cwd          string    `json:"cwd"`
	GeneratedAt  time.Time `json:"generated_at"`
	LookbackDays int       `json:"lookback_days"`

	// Layer 5 — discrete facts.
	NewFactsCount     int    `json:"new_facts_count"`
	TotalFactsCount   int    `json:"total_facts_count"`
	ZeroHitFactsCount int    `json:"zero_hit_facts_count"`
	TopHitFactText    string `json:"top_hit_fact_text"`
	TopHitFactHits    int    `json:"top_hit_fact_hits"`

	// Capture engine — how often the layer-5 writer is firing.
	CaptureFires          int `json:"capture_fires"`
	CaptureFactsExtracted int `json:"capture_facts_extracted"`
	CaptureFactsStored    int `json:"capture_facts_stored"`
	CaptureFactsDeduped   int `json:"capture_facts_deduped"`
	CaptureFailedFires    int `json:"capture_failed_fires"`

	// Layer 4 — session journal.
	NewJournalCount   int `json:"new_journal_count"`
	TotalJournalCount int `json:"total_journal_count"`

	// Layers 2-3 — operator-owned plan/goal.
	PlanLastUpdatedAt  *time.Time `json:"plan_last_updated_at,omitempty"`
	GoalLastUpdatedAt  *time.Time `json:"goal_last_updated_at,omitempty"`
	PendingProposals   int        `json:"pending_proposals"`
	OldestPendingDays  int        `json:"oldest_pending_days"`
	PlanDriftProposals int        `json:"plan_drift_proposals"`
}

// Service computes Snapshots against a pgxpool. Stateless beyond
// the pool reference; safe to construct once per process.
type Service struct {
	pool *pgxpool.Pool
}

// New wires a Service against the given pool. A nil pool is a
// programmer error — return an error instead of panicking so the
// composition root can surface it cleanly.
func New(pool *pgxpool.Pool) (*Service, error) {
	if pool == nil {
		return nil, errors.New("memhealth: pool is nil")
	}
	return &Service{pool: pool}, nil
}

// ComputeForCwd returns a fresh snapshot for the given project
// cwd. Empty cwd returns an error — every panel is project-scoped.
// Queries run sequentially; the dashboard fetches this once on
// page load so latency is not a hot path.
func (s *Service) ComputeForCwd(ctx context.Context, cwd string) (Snapshot, error) {
	if cwd == "" {
		return Snapshot{}, errors.New("memhealth: cwd is required")
	}
	since := time.Now().UTC().Add(-time.Duration(LookbackDays) * 24 * time.Hour)
	snap := Snapshot{
		Cwd:          cwd,
		GeneratedAt:  time.Now().UTC(),
		LookbackDays: LookbackDays,
	}

	if err := s.layer5(ctx, &snap, cwd, since); err != nil {
		return Snapshot{}, fmt.Errorf("memhealth: layer5: %w", err)
	}
	if err := s.captureMetrics(ctx, &snap, cwd, since); err != nil {
		return Snapshot{}, fmt.Errorf("memhealth: capture: %w", err)
	}
	if err := s.layer24(ctx, &snap, cwd, since); err != nil {
		return Snapshot{}, fmt.Errorf("memhealth: layer24: %w", err)
	}
	return snap, nil
}

// layer5 fills the discrete-fact panels: counts plus the top-hit
// memory and how many memories have never been retrieved.
func (s *Service) layer5(ctx context.Context, snap *Snapshot, cwd string, since time.Time) error {
	row := s.pool.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN created_at >= $2 THEN 1 ELSE 0 END), 0) AS new_count,
			COUNT(*)                                                       AS total_count,
			COALESCE(SUM(CASE WHEN hit_count = 0 AND created_at < $2
			                  THEN 1 ELSE 0 END), 0)                       AS zero_hit_count
		  FROM memories
		 WHERE scope = 'project' AND scope_key = $1`, cwd, since)
	if err := row.Scan(&snap.NewFactsCount, &snap.TotalFactsCount, &snap.ZeroHitFactsCount); err != nil {
		return fmt.Errorf("layer5 counts: %w", err)
	}

	topRow := s.pool.QueryRow(ctx, `
		SELECT text, hit_count
		  FROM memories
		 WHERE scope = 'project' AND scope_key = $1 AND hit_count > 0
		 ORDER BY hit_count DESC, created_at DESC
		 LIMIT 1`, cwd)
	var (
		text string
		hits int
	)
	if err := topRow.Scan(&text, &hits); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("layer5 top hit: %w", err)
		}
		// no hits yet → leave fields zero/empty.
	} else {
		snap.TopHitFactText = text
		snap.TopHitFactHits = hits
	}
	return nil
}

// captureMetrics joins memory_summarizer_calls to sessions so we
// can filter by cwd; capture-engine fires are per-session and have
// no direct cwd column.
func (s *Service) captureMetrics(ctx context.Context, snap *Snapshot, cwd string, since time.Time) error {
	row := s.pool.QueryRow(ctx, `
		SELECT
			COUNT(*)                                            AS fires,
			COALESCE(SUM(c.facts_extracted), 0)                 AS extracted,
			COALESCE(SUM(c.facts_stored), 0)                    AS stored,
			COALESCE(SUM(c.facts_skipped_dedup), 0)             AS deduped,
			COALESCE(SUM(CASE WHEN c.status <> 'succeeded'
			                  THEN 1 ELSE 0 END), 0)            AS failed
		  FROM memory_summarizer_calls c
		  JOIN sessions s ON s.id = c.session_id
		 WHERE s.cwd = $1 AND c.started_at >= $2`, cwd, since)
	return row.Scan(
		&snap.CaptureFires,
		&snap.CaptureFactsExtracted,
		&snap.CaptureFactsStored,
		&snap.CaptureFactsDeduped,
		&snap.CaptureFailedFires,
	)
}

// layer24 fills the projectdoc-side panels: journal counts, plan /
// goal last-touched timestamps, pending proposal queue depth.
func (s *Service) layer24(ctx context.Context, snap *Snapshot, cwd string, since time.Time) error {
	jrow := s.pool.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN created_at >= $2 THEN 1 ELSE 0 END), 0) AS new_count,
			COUNT(*)                                                       AS total_count
		  FROM session_logs
		 WHERE cwd = $1`, cwd, since)
	if err := jrow.Scan(&snap.NewJournalCount, &snap.TotalJournalCount); err != nil {
		return fmt.Errorf("layer24 journal: %w", err)
	}

	if err := s.scanDocTimestamp(ctx, cwd, "plan", &snap.PlanLastUpdatedAt); err != nil {
		return err
	}
	if err := s.scanDocTimestamp(ctx, cwd, "goal", &snap.GoalLastUpdatedAt); err != nil {
		return err
	}

	prow := s.pool.QueryRow(ctx, `
		SELECT
			COUNT(*)                                                       AS pending,
			COALESCE(EXTRACT(EPOCH FROM (NOW() - MIN(created_at)))::int / 86400, 0) AS oldest_days,
			COALESCE(SUM(CASE WHEN kind = 'plan' AND created_at >= $2
			                  THEN 1 ELSE 0 END), 0)                       AS plan_drift_count
		  FROM project_doc_proposals
		 WHERE cwd = $1 AND decided_at IS NULL`, cwd, since)
	return prow.Scan(&snap.PendingProposals, &snap.OldestPendingDays, &snap.PlanDriftProposals)
}

// scanDocTimestamp loads project_docs.updated_at for one (cwd,
// kind). Missing row → leave dest nil so JSON serialisation emits
// "no plan/goal yet".
func (s *Service) scanDocTimestamp(ctx context.Context, cwd, kind string, dest **time.Time) error {
	var ts time.Time
	err := s.pool.QueryRow(ctx,
		`SELECT updated_at FROM project_docs WHERE cwd = $1 AND kind = $2`,
		cwd, kind).Scan(&ts)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("layer24 doc %s: %w", kind, err)
	}
	t := ts
	*dest = &t
	return nil
}

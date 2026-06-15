package cleaner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/opendray/opendray-v2/internal/memory"
	"github.com/opendray/opendray-v2/internal/memory/worker"
)

// Config controls scanning + LLM batching behaviour. Provider routing
// is NOT here — the worker registry (memory_workers.cleaner) picks the
// summarizer / agent at call time.
type Config struct {
	// BatchSize caps how many memories are reviewed in one LLM call.
	// Default 30. Larger batches mean fewer round-trips but exceed
	// context windows quickly — qwen3.5-9b can handle ~50 reliably.
	BatchSize int

	// MinAge skips memories younger than this duration so the
	// cleaner never reviews something the user just wrote.
	// Default 24h.
	MinAge time.Duration

	// SkipIfDecidedWithin avoids re-proposing decisions about the
	// same memory within this window. Prevents the scheduler from
	// flooding the inbox after the operator approves/rejects a
	// batch. Default 7 days.
	SkipIfDecidedWithin time.Duration

	// CallTimeout caps the per-Run LLM call. Reasoning models on
	// LM Studio can take 10-30s for a 30-row batch, so allow plenty.
	// Default 60s.
	CallTimeout time.Duration

	// GracePeriod is how long a soft-archived memory stays restorable
	// before PurgeExpired hard-deletes it. Default 30 days (decision
	// §8.2). The window the operator has to undo an auto-archive from
	// the Archived view.
	GracePeriod time.Duration

	// LifecycleDormantDays is the project-lifecycle signal: when a
	// project's memory has had no new write or retrieval for this many
	// days, the project is treated as finished and its never-hit, aged
	// facts are auto-archived (reversible). Unset (0) defaults to 90; a
	// NEGATIVE value disables the lifecycle pass entirely.
	LifecycleDormantDays int
}

// applyDefaults fills zero values with the documented defaults.
func (c Config) applyDefaults() Config {
	if c.BatchSize <= 0 {
		c.BatchSize = 30
	}
	if c.MinAge <= 0 {
		c.MinAge = 24 * time.Hour
	}
	if c.SkipIfDecidedWithin <= 0 {
		c.SkipIfDecidedWithin = 7 * 24 * time.Hour
	}
	if c.CallTimeout <= 0 {
		c.CallTimeout = 60 * time.Second
	}
	if c.GracePeriod <= 0 {
		c.GracePeriod = 30 * 24 * time.Hour
	}
	if c.LifecycleDormantDays == 0 {
		c.LifecycleDormantDays = 90
	}
	return c
}

// MemoryAdapter is the subset of memory.Service the cleaner uses.
// Defined here so tests don't need the full *memory.Service graph.
type MemoryAdapter interface {
	List(ctx context.Context, scope memory.Scope, scopeKey string, limit int) ([]memory.Memory, error)
	Get(ctx context.Context, id string) (memory.Memory, error)
	Delete(ctx context.Context, id string) error
	// Archive soft-deletes a memory (reversible) — the cleaner's
	// auto-apply path uses this instead of Delete so a wrong call is
	// undoable within the grace window.
	Archive(ctx context.Context, id, reason string) error
	// PurgeArchived hard-deletes archived rows past the grace cutoff.
	PurgeArchived(ctx context.Context, cutoff time.Time) (int64, error)
	// PurgeExpiredQuarantine hard-deletes quarantined rows past their
	// TTL (Cortex Phase 2 — un-reviewed third-party capture ages out).
	PurgeExpiredQuarantine(ctx context.Context, now time.Time) (int64, error)
	// ArchiveDormantStale soft-archives never-hit aged facts of a dormant
	// project (the lifecycle signal). Returns the count archived.
	ArchiveDormantStale(ctx context.Context, scope memory.Scope, scopeKey string, agedBefore, dormantBefore time.Time, reason string) (int64, error)
}

// (ProviderFetcher was removed in M25 — provider selection now
// lives behind the worker.Registry, which handles decryption +
// fallback internally per memory_workers.cleaner row.)

// Service is the cleaner's public surface.
type Service struct {
	pool   *pgxpool.Pool
	store  *store
	mem    MemoryAdapter
	worker *worker.Registry
	cfg    Config
	log    *slog.Logger
}

// NewService wires a cleaner. mem + worker registry must be
// non-nil; Run will fail with a clear error if the worker registry
// is missing rather than silently no-op'ing.
//
// M25 — replaced direct summarizer.Registry / ProviderFetcher
// deps with worker.Registry. Provider selection now happens per-
// call via memory_workers.cleaner config so operators flip
// between local-LLM and agent without restart.
func NewService(
	pool *pgxpool.Pool,
	mem MemoryAdapter,
	wr *worker.Registry,
	cfg Config,
	log *slog.Logger,
) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{
		pool:   pool,
		store:  newStore(pool),
		mem:    mem,
		worker: wr,
		cfg:    cfg.applyDefaults(),
		log:    log.With("component", "memory.cleaner"),
	}
}

// RunResult summarises one cleanup pass.
type RunResult struct {
	RunID        string `json:"run_id"`
	Scope        string `json:"scope"`
	ScopeKey     string `json:"scope_key"`
	MemoriesIn   int    `json:"memories_in"`
	DecisionsOut int    `json:"decisions_out"`
}

// Run scans up to BatchSize aged-eligible memories under (scope,
// scope_key), packages them as one LLM call, persists each returned
// decision as 'pending', and returns the run summary.
//
// Errors are surfaced verbatim — the caller (HTTP handler or the
// scheduler goroutine) decides whether to swallow + log or
// propagate. Partial success is possible: if the LLM returns 30
// decisions but the DB rejects 2, we still report 28 written.
func (s *Service) Run(ctx context.Context, scope memory.Scope, scopeKey string) (RunResult, error) {
	if s.worker == nil {
		return RunResult{}, errors.New("cleaner: no worker registry wired")
	}
	if scope == "" {
		return RunResult{}, errors.New("cleaner: scope required")
	}

	runID := newID("mcr_")

	// 1. Fetch a window of memories. We pull 3× BatchSize because
	// some will be filtered out (too young, recently decided).
	rawWindow := s.cfg.BatchSize * 3
	if rawWindow < 60 {
		rawWindow = 60
	}
	all, err := s.mem.List(ctx, scope, scopeKey, rawWindow)
	if err != nil {
		return RunResult{}, fmt.Errorf("cleaner: list memories: %w", err)
	}

	// 2. Filter by min_age and recent-decisions.
	candidate, err := s.filterEligible(ctx, all, scope, scopeKey)
	if err != nil {
		return RunResult{}, err
	}
	if len(candidate) > s.cfg.BatchSize {
		candidate = candidate[:s.cfg.BatchSize]
	}
	if len(candidate) == 0 {
		s.log.Info("cleaner.no_work", "scope", scope, "scope_key", scopeKey, "total_in_window", len(all))
		return RunResult{RunID: runID, Scope: string(scope), ScopeKey: scopeKey, MemoriesIn: 0, DecisionsOut: 0}, nil
	}

	// 3. Build LLM client + run judgement. M25 — dispatch goes
	// through the worker registry; provider selection happens per
	// memory_workers.cleaner row.
	cli := NewClient(s.worker)
	if cli == nil {
		return RunResult{}, errors.New("cleaner: worker registry not wired")
	}
	providerID := ""

	items := make([]BatchItem, 0, len(candidate))
	for _, m := range candidate {
		items = append(items, BatchItem{
			ID:        m.ID,
			Text:      m.Text,
			CreatedAt: m.CreatedAt,
			HitCount:  m.HitCount,
		})
	}
	decisions, err := cli.Judge(ctx, items, s.cfg.CallTimeout)
	if err != nil {
		return RunResult{}, fmt.Errorf("cleaner: judge: %w", err)
	}

	// 4. Persist decisions. Skip duplicates we already have for
	// this memory_id (defense in depth — filterEligible already
	// dropped these, but a parallel Run could race).
	written := 0
	scopeStr := string(scope)
	for _, d := range decisions {
		// Find the original memory to snapshot text + scope.
		var orig *memory.Memory
		for i := range candidate {
			if candidate[i].ID == d.MemoryID {
				orig = &candidate[i]
				break
			}
		}
		if orig == nil {
			// LLM hallucinated an id not in the batch. Skip.
			s.log.Warn("cleaner.hallucinated_id", "id", d.MemoryID, "run_id", runID)
			continue
		}
		mergeInto := ""
		if d.MergeInto != nil {
			mergeInto = *d.MergeInto
		}
		dec := Decision{
			MemoryID:             orig.ID,
			MemoryScope:          scopeStr,
			MemoryScopeKey:       scopeKey,
			MemoryTextSnapshot:   orig.Text,
			Verdict:              d.Verdict,
			Reason:               d.Reason,
			MergeInto:            mergeInto,
			RunID:                runID,
			SummarizerProviderID: providerID,
		}
		inserted, err := s.store.Insert(ctx, dec)
		if err != nil {
			s.log.Warn("cleaner.insert_decision_failed", "memory_id", orig.ID, "err", err)
			continue
		}
		written++

		// Auto-apply (M-U Phase 4): no manual approval queue. The verdict
		// is executed immediately as a reversible soft-archive; the
		// decision row is the audit trail. Operators review only
		// conflicts now — routine staleness/dupes are handled here and
		// undoable from the Archived view within the grace window.
		if execErr := s.execute(ctx, inserted); execErr != nil {
			s.log.Warn("cleaner.auto_execute_failed",
				"decision_id", inserted.ID, "verdict", inserted.Verdict, "err", execErr)
			_ = s.store.SetStatus(ctx, inserted.ID, StatusExpired, StatusPending)
			continue
		}
		_ = s.store.SetStatus(ctx, inserted.ID, StatusExecuted, StatusPending)
	}
	s.log.Info("cleaner.run_complete",
		"run_id", runID, "scope", scope, "scope_key", scopeKey,
		"memories_in", len(items), "decisions_out", written,
	)
	return RunResult{
		RunID:        runID,
		Scope:        scopeStr,
		ScopeKey:     scopeKey,
		MemoriesIn:   len(items),
		DecisionsOut: written,
	}, nil
}

// List returns existing decisions filtered by status / scope.
func (s *Service) List(ctx context.Context, status, scope, scopeKey string, limit int) ([]Decision, error) {
	return s.store.List(ctx, status, scope, scopeKey, limit)
}

// Get returns one decision by id.
func (s *Service) Get(ctx context.Context, id string) (Decision, error) {
	return s.store.Get(ctx, id)
}

// execute applies one verdict by SOFT-ARCHIVING (not hard-deleting), so
// every action is reversible within the grace window. A "duplicate" is
// archived with the survivor noted in archived_reason — because the row
// is preserved (not destroyed), there's no need to mutate the survivor's
// metadata, which retires the old merge-metadata gap.
func (s *Service) execute(ctx context.Context, d Decision) error {
	switch d.Verdict {
	case VerdictKeep:
		// Nothing to do — the LLM judged this worth keeping.
		return nil
	case VerdictStale:
		if err := s.mem.Archive(ctx, d.MemoryID, archiveReason("stale", d.Reason)); err != nil {
			return fmt.Errorf("cleaner: archive stale: %w", err)
		}
		s.log.Info("cleaner.archived_stale", "memory_id", d.MemoryID, "decision_id", d.ID)
		return nil
	case VerdictDuplicate:
		if d.MergeInto == "" {
			return errors.New("cleaner: duplicate with no merge_into")
		}
		// Survivor must still be active — otherwise skip rather than
		// archive the only remaining copy.
		if _, err := s.mem.Get(ctx, d.MergeInto); err != nil {
			return fmt.Errorf("cleaner: merge_into %s missing: %w", d.MergeInto, err)
		}
		if err := s.mem.Archive(ctx, d.MemoryID, "duplicate of "+d.MergeInto); err != nil {
			return fmt.Errorf("cleaner: archive duplicate: %w", err)
		}
		s.log.Info("cleaner.archived_duplicate",
			"memory_id", d.MemoryID, "merged_into", d.MergeInto, "decision_id", d.ID)
		return nil
	default:
		return fmt.Errorf("cleaner: unknown verdict %q", d.Verdict)
	}
}

// ArchiveDormant applies the project-lifecycle signal for one project:
// if its memory has gone quiet for LifecycleDormantDays, its never-hit
// aged facts are soft-archived (reversible). No-op when disabled
// (negative days) or the project is still active. Returns count archived.
func (s *Service) ArchiveDormant(ctx context.Context, scopeKey string) (int64, error) {
	if s.cfg.LifecycleDormantDays < 0 || strings.TrimSpace(scopeKey) == "" {
		return 0, nil
	}
	now := time.Now()
	agedBefore := now.Add(-s.cfg.MinAge)
	dormantBefore := now.AddDate(0, 0, -s.cfg.LifecycleDormantDays)
	reason := fmt.Sprintf("lifecycle: project dormant >%dd", s.cfg.LifecycleDormantDays)
	n, err := s.mem.ArchiveDormantStale(ctx, memory.ScopeProject, scopeKey, agedBefore, dormantBefore, reason)
	if err != nil {
		return 0, fmt.Errorf("cleaner: archive dormant: %w", err)
	}
	if n > 0 {
		s.log.Info("cleaner.archived_dormant", "scope_key", scopeKey, "count", n, "dormant_days", s.cfg.LifecycleDormantDays)
	}
	return n, nil
}

// PurgeExpired hard-deletes memories whose soft-archive grace window has
// elapsed, plus quarantined rows past their TTL (Cortex Phase 2).
// Called once per scheduler tick. Returns the count purged.
func (s *Service) PurgeExpired(ctx context.Context) (int64, error) {
	cutoff := time.Now().Add(-s.cfg.GracePeriod)
	n, err := s.mem.PurgeArchived(ctx, cutoff)
	if err != nil {
		return 0, fmt.Errorf("cleaner: purge expired: %w", err)
	}
	if n > 0 {
		s.log.Info("cleaner.purged_expired", "count", n, "grace", s.cfg.GracePeriod)
	}
	q, err := s.mem.PurgeExpiredQuarantine(ctx, time.Now())
	if err != nil {
		return n, fmt.Errorf("cleaner: purge expired quarantine: %w", err)
	}
	if q > 0 {
		s.log.Info("cleaner.purged_expired_quarantine", "count", q)
	}
	return n + q, nil
}

// archiveReason builds a short archived_reason from a verdict label and
// the LLM's justification, bounded so the column stays compact.
func archiveReason(label, reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "cleaner: " + label
	}
	if len(reason) > 160 {
		reason = reason[:160]
	}
	return "cleaner: " + label + " — " + reason
}

// filterEligible drops memories younger than MinAge and memories
// that already have a recent decision (any non-expired status).
func (s *Service) filterEligible(ctx context.Context, in []memory.Memory, scope memory.Scope, scopeKey string) ([]memory.Memory, error) {
	if len(in) == 0 {
		return nil, nil
	}
	cutoff := time.Now().Add(-s.cfg.MinAge)

	// Aged-out first.
	aged := make([]memory.Memory, 0, len(in))
	ids := make([]string, 0, len(in))
	for _, m := range in {
		if m.CreatedAt.After(cutoff) {
			continue
		}
		aged = append(aged, m)
		ids = append(ids, m.ID)
	}
	if len(aged) == 0 {
		return nil, nil
	}

	// Dedup against existing decisions.
	since := time.Now().Add(-s.cfg.SkipIfDecidedWithin)
	rows, err := s.pool.Query(ctx, `
		SELECT memory_id
		  FROM memory_cleanup_decisions
		 WHERE memory_id = ANY($1)
		   AND created_at >= $2
		   AND status <> 'expired'`, ids, since)
	if err != nil {
		return nil, fmt.Errorf("cleaner: dedup decisions: %w", err)
	}
	defer rows.Close()
	skip := map[string]struct{}{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		skip[id] = struct{}{}
	}

	out := make([]memory.Memory, 0, len(aged))
	for _, m := range aged {
		if _, dup := skip[m.ID]; dup {
			continue
		}
		out = append(out, m)
	}
	return out, nil
}

// (resolveProvider and peekAPIKey were removed in M25 — provider
// selection now happens inside the worker registry per call,
// driven by the memory_workers.cleaner row.)

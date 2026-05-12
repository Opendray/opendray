package cleaner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/opendray/opendray-v2/internal/memory"
	"github.com/opendray/opendray-v2/internal/memory/summarizer"
)

// Config controls scanning + LLM batching behaviour.
type Config struct {
	// SummarizerID picks the LLM provider. Empty → registry default.
	SummarizerID string

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
	return c
}

// MemoryAdapter is the subset of memory.Service the cleaner uses.
// Defined here so tests don't need the full *memory.Service graph.
type MemoryAdapter interface {
	List(ctx context.Context, scope memory.Scope, scopeKey string, limit int) ([]memory.Memory, error)
	Get(ctx context.Context, id string) (memory.Memory, error)
	Delete(ctx context.Context, id string) error
}

// ProviderFetcher fetches a single summarizer provider row WITH the
// decrypted api_key in APIKeyPlaintext (when the row is anthropic
// + the cipher is armed). summarizer.Store satisfies this directly
// via GetProvider. Defined as an interface here so cleaner tests
// can stub it without dragging in the cipher chain.
type ProviderFetcher interface {
	GetProvider(ctx context.Context, id string) (summarizer.ProviderRow, error)
}

// Service is the cleaner's public surface.
type Service struct {
	pool      *pgxpool.Pool
	store     *store
	mem       MemoryAdapter
	reg       *summarizer.Registry
	providers ProviderFetcher
	cfg       Config
	log       *slog.Logger
}

// NewService wires a cleaner. mem + reg + providers must non-nil;
// Run will fail with a clear error if reg is missing rather than
// silently no-op'ing.
func NewService(
	pool *pgxpool.Pool,
	mem MemoryAdapter,
	reg *summarizer.Registry,
	providers ProviderFetcher,
	cfg Config,
	log *slog.Logger,
) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{
		pool:      pool,
		store:     newStore(pool),
		mem:       mem,
		reg:       reg,
		providers: providers,
		cfg:       cfg.applyDefaults(),
		log:       log.With("component", "memory.cleaner"),
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
	if s.reg == nil {
		return RunResult{}, errors.New("cleaner: no summarizer registry wired")
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

	// 3. Build LLM client + run judgement.
	provider, providerID, err := s.resolveProvider(ctx)
	if err != nil {
		return RunResult{}, err
	}
	apiKey, _ := s.peekAPIKey(ctx, providerID)
	cli, err := NewClient(provider, apiKey)
	if err != nil {
		return RunResult{}, err
	}

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
		if _, err := s.store.Insert(ctx, dec); err != nil {
			s.log.Warn("cleaner.insert_decision_failed", "memory_id", orig.ID, "err", err)
			continue
		}
		written++
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

// Approve marks the decision approved AND executes it in the same
// call. We don't separate the two states because the operator's
// "Approve" click implies "do it now"; deferred execution would
// just add a state to babysit.
func (s *Service) Approve(ctx context.Context, id string) error {
	if err := s.store.SetStatus(ctx, id, StatusApproved, StatusPending); err != nil {
		return err
	}
	d, err := s.store.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.execute(ctx, d); err != nil {
		// Mark expired so the inbox stops showing this as actionable.
		// The original approval still has decided_at set; expired
		// tells the UI "we tried but couldn't apply".
		_ = s.store.SetStatus(ctx, id, StatusExpired, StatusApproved)
		return err
	}
	return s.store.SetStatus(ctx, id, StatusExecuted, StatusApproved)
}

// Reject closes the decision without touching the memory store.
func (s *Service) Reject(ctx context.Context, id string) error {
	return s.store.SetStatus(ctx, id, StatusRejected, StatusPending)
}

// execute performs the actual memory mutation per verdict.
func (s *Service) execute(ctx context.Context, d Decision) error {
	switch d.Verdict {
	case VerdictKeep:
		// Nothing to do. Approving a keep is a "noted, no-op" gesture.
		return nil
	case VerdictStale:
		if err := s.mem.Delete(ctx, d.MemoryID); err != nil {
			return fmt.Errorf("cleaner: delete stale: %w", err)
		}
		s.log.Info("cleaner.executed_stale", "memory_id", d.MemoryID, "decision_id", d.ID)
		return nil
	case VerdictDuplicate:
		if d.MergeInto == "" {
			return errors.New("cleaner: duplicate with no merge_into")
		}
		// Verify merge_into still exists — otherwise we'd delete the
		// duplicate without preserving the canonical row.
		if _, err := s.mem.Get(ctx, d.MergeInto); err != nil {
			return fmt.Errorf("cleaner: merge_into %s missing: %w", d.MergeInto, err)
		}
		if err := s.mem.Delete(ctx, d.MemoryID); err != nil {
			return fmt.Errorf("cleaner: delete duplicate: %w", err)
		}
		s.log.Info("cleaner.executed_duplicate",
			"memory_id", d.MemoryID, "merged_into", d.MergeInto, "decision_id", d.ID)
		return nil
	default:
		return fmt.Errorf("cleaner: unknown verdict %q", d.Verdict)
	}
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

// resolveProvider picks the configured summarizer row + builds a
// Provider (just to validate it's wireable). We return both the
// raw row (for HTTP client construction) and the row id for
// audit logging.
func (s *Service) resolveProvider(ctx context.Context) (summarizer.ProviderRow, string, error) {
	rows, err := s.reg.ListEnabledRows(ctx)
	if err != nil {
		return summarizer.ProviderRow{}, "", fmt.Errorf("cleaner: list summarizer rows: %w", err)
	}
	if len(rows) == 0 {
		return summarizer.ProviderRow{}, "", errors.New("cleaner: no enabled summarizer provider")
	}
	if s.cfg.SummarizerID != "" {
		for _, r := range rows {
			if r.ID == s.cfg.SummarizerID {
				return r, r.ID, nil
			}
		}
		return summarizer.ProviderRow{}, "", fmt.Errorf("cleaner: summarizer %q not found / not enabled", s.cfg.SummarizerID)
	}
	for _, r := range rows {
		if r.IsDefault {
			return r, r.ID, nil
		}
	}
	// No default flagged: pick the first enabled one for
	// determinism. Operators can pin via SummarizerID.
	return rows[0], rows[0].ID, nil
}

// peekAPIKey fetches the plaintext API key for one provider row.
// Goes through ProviderFetcher.GetProvider — that's the path that
// decrypts the ciphered key when the backup cipher is armed. LM
// Studio rows have no api_key and return "" + nil error, which is
// what we want (HTTP client just skips the Authorization header).
func (s *Service) peekAPIKey(ctx context.Context, providerID string) (string, error) {
	if providerID == "" || s.providers == nil {
		return "", nil
	}
	row, err := s.providers.GetProvider(ctx, providerID)
	if err != nil {
		return "", err
	}
	return row.APIKeyPlaintext, nil
}

package knowledge

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

// --- P-C: unified consolidation engine ---------------------------------------
//
// Before P-C, three independent background loops (anchor / reflect / KB) each
// ran on their own timer. Nothing ordered them, so a KB draft could fire on
// facts the anchor sweep hadn't produced yet, and a reflect pass could miss the
// freshest facts — wasting an LLM call that the next cycle redid.
//
// The ConsolidationEngine folds them into ONE loop that runs the stages in
// dependency order per cycle:
//
//	anchor   — lift new Memory facts into entities (substrate)
//	compile  — mine cross-project episodes for recurring, repeatedly
//	           successful procedures (the experience compiler)
//	KB draft — render the human-readable KB pages from the fresh facts/playbooks
//
// Every stage is already dirty-checked + lock-aware internally, so steady-state
// cost stays ~0 LLM calls; the engine only guarantees ordering + a single
// schedule. Any stage may be nil (feature-flagged off) and is skipped.

// ConsolidationEngine sequences the knowledge-side consolidation stages.
type ConsolidationEngine struct {
	anchorer *Anchorer
	compiler *ExperienceCompiler
	kb       *KBDrafter
	overview *OverviewDrafter
	// curator runs the skill-lifecycle sweep (active→stale→auto-
	// disabled). Optional, like every other stage.
	curator interface {
		CurateSkills(ctx context.Context) (int, int, error)
	}
	log *slog.Logger
}

// WithCurator wires the skill-lifecycle curator (the knowledge
// Service implements it).
func (e *ConsolidationEngine) WithCurator(c interface {
	CurateSkills(ctx context.Context) (int, int, error)
}) *ConsolidationEngine {
	e.curator = c
	return e
}

// NewConsolidationEngine wires the stages. Any of them may be nil.
func NewConsolidationEngine(a *Anchorer, c *ExperienceCompiler, kb *KBDrafter, ov *OverviewDrafter, log *slog.Logger) *ConsolidationEngine {
	if log == nil {
		log = slog.Default()
	}
	return &ConsolidationEngine{anchorer: a, compiler: c, kb: kb, overview: ov, log: log.With("component", "knowledge.consolidate")}
}

// Enabled reports whether at least one stage is wired (otherwise the loop is a
// no-op and the caller can skip launching it).
func (e *ConsolidationEngine) Enabled() bool {
	return e.anchorer != nil || e.compiler != nil || e.kb != nil || e.overview != nil
}

// ConsolidateConfig tunes the unified loop. One interval drives the whole
// pipeline; the per-stage knobs map to the old per-sweep configs.
type ConsolidateConfig struct {
	Interval     time.Duration // between cycles (default 15m)
	InitialDelay time.Duration // before the first cycle (default 1m)
	PerProject   int           // anchor: max memories pulled per project (default 500)
}

func (c ConsolidateConfig) withDefaults() ConsolidateConfig {
	if c.Interval <= 0 {
		c.Interval = 15 * time.Minute
	}
	if c.InitialDelay <= 0 {
		c.InitialDelay = time.Minute
	}
	if c.PerProject <= 0 {
		c.PerProject = 500
	}
	return c
}

// Run blocks until ctx is cancelled, running one ordered consolidation cycle
// per interval. Replaces the separate RunAnchorSweep / RunReflectSweep /
// RunKBSweep goroutines.
func (e *ConsolidationEngine) Run(ctx context.Context, cfg ConsolidateConfig) {
	if !e.Enabled() {
		return
	}
	cfg = cfg.withDefaults()
	e.log.Info("knowledge consolidation engine running",
		"interval", cfg.Interval,
		"anchor", e.anchorer != nil, "compile", e.compiler != nil, "kb", e.kb != nil)
	timer := time.NewTimer(cfg.InitialDelay)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		e.RunOnce(ctx, cfg)
		timer.Reset(cfg.Interval)
	}
}

// RunOnce executes a single ordered cycle: anchor → compile → KB. Each stage
// soft-fails (logged, not propagated) so one bad stage never starves the next
// cycle. A cancelled context short-circuits the remaining stages.
func (e *ConsolidationEngine) RunOnce(ctx context.Context, cfg ConsolidateConfig) {
	cfg = cfg.withDefaults()

	if e.anchorer != nil {
		if err := e.anchorer.AnchorAll(ctx, cfg.PerProject); err != nil && !errors.Is(err, context.Canceled) {
			e.log.Warn("consolidation: anchor stage failed", "err", err)
		}
	}
	if ctx.Err() != nil {
		return
	}
	if e.compiler != nil {
		if _, err := e.compiler.CompileAll(ctx); err != nil && !errors.Is(err, context.Canceled) {
			e.log.Warn("consolidation: experience-compile stage failed", "err", err)
		}
	}
	if ctx.Err() != nil {
		return
	}
	if e.kb != nil {
		if _, err := e.kb.DraftAll(ctx); err != nil && !errors.Is(err, context.Canceled) {
			e.log.Warn("consolidation: kb stage failed", "err", err)
		}
	}
	if ctx.Err() != nil {
		return
	}
	// Overview last — it synthesises the project's own goal/plan/tech + journal
	// + memory into the per-project official document (Notes).
	if e.overview != nil {
		if _, err := e.overview.DraftAll(ctx); err != nil && !errors.Is(err, context.Canceled) {
			e.log.Warn("consolidation: overview stage failed", "err", err)
		}
	}
	if ctx.Err() != nil {
		return
	}
	// Curator — the Hermes-style lifecycle sweep over the skill
	// library (active → stale 30d → auto-disabled 90d). Cheap SQL.
	if e.curator != nil {
		if stale, disabled, err := e.curator.CurateSkills(ctx); err != nil && !errors.Is(err, context.Canceled) {
			e.log.Warn("consolidation: curator stage failed", "err", err)
		} else if stale+disabled > 0 {
			e.log.Info("consolidation: curator swept skills", "stale", stale, "auto_disabled", disabled)
		}
	}
}

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
//	reflect  — distil facts + journal into playbooks
//	KB draft — render the human-readable KB pages from the fresh facts/playbooks
//
// Every stage is already dirty-checked + lock-aware internally, so steady-state
// cost stays ~0 LLM calls; the engine only guarantees ordering + a single
// schedule. Any stage may be nil (feature-flagged off) and is skipped.

// ConsolidationEngine sequences the knowledge-side consolidation stages.
type ConsolidationEngine struct {
	anchorer  *Anchorer
	reflector *Reflector
	kb        *KBDrafter
	log       *slog.Logger
}

// NewConsolidationEngine wires the stages. Any of them may be nil.
func NewConsolidationEngine(a *Anchorer, r *Reflector, kb *KBDrafter, log *slog.Logger) *ConsolidationEngine {
	if log == nil {
		log = slog.Default()
	}
	return &ConsolidationEngine{anchorer: a, reflector: r, kb: kb, log: log.With("component", "knowledge.consolidate")}
}

// Enabled reports whether at least one stage is wired (otherwise the loop is a
// no-op and the caller can skip launching it).
func (e *ConsolidationEngine) Enabled() bool {
	return e.anchorer != nil || e.reflector != nil || e.kb != nil
}

// ConsolidateConfig tunes the unified loop. One interval drives the whole
// pipeline; the per-stage knobs map to the old per-sweep configs.
type ConsolidateConfig struct {
	Interval     time.Duration // between cycles (default 15m)
	InitialDelay time.Duration // before the first cycle (default 1m)
	PerProject   int           // anchor: max memories pulled per project (default 500)
	MinFacts     int           // reflect: skip projects with fewer facts (default 5)
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
	if c.MinFacts <= 0 {
		c.MinFacts = 5
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
		"anchor", e.anchorer != nil, "reflect", e.reflector != nil, "kb", e.kb != nil)
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

// RunOnce executes a single ordered cycle: anchor → reflect → KB. Each stage
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
	if e.reflector != nil {
		if _, err := e.reflector.ReflectAll(ctx, cfg.MinFacts); err != nil && !errors.Is(err, context.Canceled) {
			e.log.Warn("consolidation: reflect stage failed", "err", err)
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
}

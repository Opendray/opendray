package capture

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/opendray/opendray-v2/internal/memory/summarizer"
)

// DefaultTickInterval is the polling cadence. 10s is the planned
// trade-off between latency-to-fire and CPU/network cost.
const DefaultTickInterval = 10 * time.Second

// DefaultHistoryLimit caps how many transcript entries each tick
// reads per session — a guard against an enormous conversation
// freezing the loop. Phase A 200 should comfortably exceed any
// practical N for after_messages triggers.
const DefaultHistoryLimit = 200

// Deps groups the dependencies the engine wires together. Rules /
// Registry / Memory / Sessions / History / CallLog are required;
// the rest are optional with sensible defaults.
type Deps struct {
	Rules    *RuleStore
	Registry *summarizer.Registry
	Memory   MemoryWriter
	Sessions SessionLister
	History  HistoryReader
	CallLog  SummarizerCallLogger
	Log      *slog.Logger
	// TickInterval — 0 falls back to DefaultTickInterval.
	TickInterval time.Duration
	// HistoryLimit — 0 falls back to DefaultHistoryLimit.
	HistoryLimit int
	// WorkerProvider is the M-PE worker-fabric-backed default
	// provider. Nil keeps the pre-M-PE behaviour
	// (summarizer.Registry.Default() for rules with no pinned
	// SummarizerProviderID). When non-nil, no-pinning rules
	// dispatch through the worker fabric so operators can choose
	// Agent (CLI --print) for capture from the Memory → Workers UI.
	WorkerProvider summarizer.Provider
	// Policy resolves integration memory policies (Cortex Phase 2).
	// Nil quarantines every integration-created session's facts.
	Policy PolicyResolver
	// QuarantineTTL — 0 falls back to DefaultQuarantineTTL.
	QuarantineTTL time.Duration
}

// Engine drives the ambient capture loop: every TickInterval, list
// live sessions, resolve each session's rule, evaluate the trigger,
// and call the runner when the trigger fires.
//
// The engine is a single goroutine — sequential per-tick processing
// is fine because (a) summarizer calls are minutes-apart in steady
// state, (b) sequential keeps DB load predictable, and (c) the
// 30-60s provider call budget per tick won't starve any other
// session for catastrophic durations.
type Engine struct {
	deps   Deps
	runner *runner
	state  *stateMap
	log    *slog.Logger
}

func NewEngine(deps Deps) (*Engine, error) {
	if deps.Rules == nil {
		return nil, errors.New("capture: Deps.Rules required")
	}
	if deps.Registry == nil {
		return nil, errors.New("capture: Deps.Registry required")
	}
	if deps.Memory == nil {
		return nil, errors.New("capture: Deps.Memory required")
	}
	if deps.Sessions == nil {
		return nil, errors.New("capture: Deps.Sessions required")
	}
	if deps.History == nil {
		return nil, errors.New("capture: Deps.History required")
	}
	if deps.CallLog == nil {
		return nil, errors.New("capture: Deps.CallLog required")
	}
	if deps.Log == nil {
		deps.Log = slog.Default()
	}
	if deps.TickInterval == 0 {
		deps.TickInterval = DefaultTickInterval
	}
	if deps.HistoryLimit == 0 {
		deps.HistoryLimit = DefaultHistoryLimit
	}
	state := newStateMap()
	r := &runner{
		rules:          deps.Rules,
		registry:       deps.Registry,
		workerProvider: deps.WorkerProvider,
		memory:         deps.Memory,
		history:        deps.History,
		callLog:        deps.CallLog,
		state:          state,
		historyLimit:   deps.HistoryLimit,
		log:            deps.Log.With("component", "capture-runner"),
		policy:         deps.Policy,
		quarantineTTL:  deps.QuarantineTTL,
	}
	return &Engine{
		deps:   deps,
		runner: r,
		state:  state,
		log:    deps.Log.With("component", "capture-engine"),
	}, nil
}

// Run blocks until ctx is cancelled. Each tick:
//  1. List live sessions.
//  2. For each session, resolve its rule (per-session override or
//     global default).
//  3. Hand to runner.runForSession.
//
// Errors at any step are logged and the loop continues — capture
// is non-critical, never block the rest of opendray.
func (e *Engine) Run(ctx context.Context) {
	e.log.Info("capture engine running",
		"interval", e.deps.TickInterval,
		"history_limit", e.deps.HistoryLimit)
	t := time.NewTicker(e.deps.TickInterval)
	defer t.Stop()
	// Run one tick immediately so the engine reacts fast to the
	// first session that comes online instead of waiting for the
	// initial timer.
	e.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			e.log.Info("capture engine stopping")
			return
		case <-t.C:
			e.tick(ctx)
		}
	}
}

func (e *Engine) tick(ctx context.Context) {
	sessions, err := e.deps.Sessions.List(ctx)
	if err != nil {
		e.log.Warn("tick: list sessions failed", "err", err)
		return
	}
	for _, sess := range sessions {
		// Only summarise providers whose transcripts we know how
		// to read. session.Manager.History returns
		// UnsupportedProvider=true for others; capture's wrapper
		// returns an empty slice so the rule simply never fires
		// for that session — but we save the call by skipping
		// here.
		if !providerSupported(sess.ProviderID) {
			continue
		}
		rule, err := e.deps.Rules.Resolve(ctx, sess.ID)
		if err != nil {
			if errors.Is(err, ErrRuleNotFound) {
				continue
			}
			e.log.Warn("tick: resolve rule failed", "session_id", sess.ID, "err", err)
			continue
		}
		if !rule.Enabled {
			continue
		}
		e.runner.runForSession(ctx, rule, sess)
	}
}

// RunRuleNow invokes the runner for the given rule across every
// matching live session, bypassing trigger evaluation. Used by:
//   - the /run-now admin endpoint to fire a manual rule on demand,
//   - future Phase C UI buttons in the session toolbar.
//
// session_id behaviour:
//   - rule.SessionID == "" (global default): runs for every live
//     session whose provider transcripts capture can read.
//   - rule.SessionID != "": runs only for that one session.
//
// Each invocation goes through the same dedup + call-log path as
// auto-fired ticks, so manual runs are auditable identically.
//
// Returns the count of sessions touched. Per-session errors are
// logged via the runner's own log surface and don't abort siblings.
func (e *Engine) RunRuleNow(ctx context.Context, ruleID string) (int, error) {
	rule, err := e.deps.Rules.Get(ctx, ruleID)
	if err != nil {
		return 0, err
	}
	if !rule.Enabled {
		return 0, fmt.Errorf("capture: rule %s is disabled", ruleID)
	}
	sessions, err := e.deps.Sessions.List(ctx)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, sess := range sessions {
		if !providerSupported(sess.ProviderID) {
			continue
		}
		if rule.SessionID != "" && rule.SessionID != sess.ID {
			continue
		}
		// Force-fire by setting the cursor to "everything new" then
		// calling runForSession; runForSession's trigger eval will
		// pass because we just stuffed CharsSinceLastFire / message
		// counts to satisfy any ManualTrigger or otherwise.
		// Easiest: call runForSession with force=true via a helper.
		e.runner.runForceForSession(ctx, rule, sess)
		count++
	}
	return count, nil
}

// providerSupported is the allowlist of provider ids whose
// transcripts capture can read via session.Manager.History. Only
// claude + codex have an input-history reader; antigravity exposes a
// conversation transcript (Manager.Transcript) but no History reader,
// so it stays out of this list until one exists.
func providerSupported(providerID string) bool {
	switch providerID {
	case "claude", "codex":
		return true
	}
	return false
}

package autoloop

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/opendray/opendray-v2/internal/eventbus"
)

// Store is the persistence the engine depends on. *PgStore satisfies it; tests
// use a fake. Defined here (at the point of use) per Go convention.
type Store interface {
	Create(ctx context.Context, l Loop) error
	Get(ctx context.Context, id string) (Loop, error)
	List(ctx context.Context) ([]Loop, error)
	ListLive(ctx context.Context) ([]Loop, error)
	ListRuns(ctx context.Context, loopID string) ([]Run, error)
	MarkRunning(ctx context.Context, id string) error
	SetStatus(ctx context.Context, id string, st Status, reason string) error
	SaveProgress(ctx context.Context, id string, iteration int, verdict, reason string) error
	AppendRun(ctx context.Context, loopID string, iteration int, prompt string) (int64, error)
	FinishRun(ctx context.Context, runID int64, verdict, reason string) error
}

// SessionDriver is the slice of session.Manager the engine drives. PTY-level
// and provider-agnostic: the same three calls drive any CLI opendray supports.
type SessionDriver interface {
	// ExpectTurn arms the session's turn watcher so the next settle fires
	// session.turn_completed.
	ExpectTurn(id string)
	// Input writes bytes to the session's stdin.
	Input(ctx context.Context, id string, data []byte) error
	// RecentSnippet returns the session's recent visible output (fallback
	// source for the judge when the turn_completed event carries none).
	RecentSnippet(id string) string
}

// Judger verifies a goal-mode turn. *WorkerJudge satisfies it.
type Judger interface {
	Judge(ctx context.Context, loop Loop, turnOutput string) (Verdict, error)
}

// ErrClosed is returned when starting a loop on a shut-down engine.
var ErrClosed = errors.New("autoloop: engine is shut down")

// Engine orchestrates persistent loops: one goroutine per running loop, with
// a registry so loops can be paused/stopped and re-armed after a restart.
type Engine struct {
	store  Store
	driver SessionDriver
	judge  Judger
	bus    *eventbus.Hub
	log    *slog.Logger

	// Injected for deterministic tests; production uses the real defaults.
	now          func() time.Time
	intervalUnit time.Duration // IntervalSeconds is multiplied by this (real: 1s)
	perRuneDelay time.Duration // inter-keystroke pause when submitting a prompt
	submitDelay  time.Duration // settle pause before the Enter byte
	turnTimeout  time.Duration // max wait for session.turn_completed per turn

	mu     sync.Mutex
	active map[string]context.CancelFunc
	wg     sync.WaitGroup
	closed bool
}

// Option configures an Engine.
type Option func(*Engine)

// WithClock overrides the time source (deadline checks).
func WithClock(f func() time.Time) Option { return func(e *Engine) { e.now = f } }

// WithIntervalUnit overrides the multiplier applied to IntervalSeconds (tests
// set this to time.Millisecond to run interval loops fast).
func WithIntervalUnit(d time.Duration) Option { return func(e *Engine) { e.intervalUnit = d } }

// WithInputDelays overrides the per-rune + submit pauses (tests set 0).
func WithInputDelays(perRune, submit time.Duration) Option {
	return func(e *Engine) { e.perRuneDelay, e.submitDelay = perRune, submit }
}

// WithTurnTimeout overrides the per-turn wait for turn_completed.
func WithTurnTimeout(d time.Duration) Option { return func(e *Engine) { e.turnTimeout = d } }

// New wires an engine with production defaults.
func New(store Store, driver SessionDriver, judge Judger, bus *eventbus.Hub, log *slog.Logger, opts ...Option) *Engine {
	if log == nil {
		log = slog.Default()
	}
	e := &Engine{
		store:        store,
		driver:       driver,
		judge:        judge,
		bus:          bus,
		log:          log.With("component", "autoloop"),
		now:          time.Now,
		intervalUnit: time.Second,
		perRuneDelay: 5 * time.Millisecond,
		submitDelay:  40 * time.Millisecond,
		turnTimeout:  5 * time.Minute,
		active:       make(map[string]context.CancelFunc),
	}
	for _, o := range opts {
		o(e)
	}
	return e
}

// Create validates, persists, and starts a loop.
func (e *Engine) Create(ctx context.Context, req CreateRequest) (Loop, error) {
	req = req.normalize()
	if err := req.validate(e.now()); err != nil {
		return Loop{}, err
	}
	l := newLoop(req, e.now())
	if err := e.store.Create(ctx, l); err != nil {
		return Loop{}, err
	}
	if err := e.start(ctx, l); err != nil {
		return l, err
	}
	l.Status = StatusRunning
	return l, nil
}

// Get / List / Runs are read-throughs for the API.
func (e *Engine) Get(ctx context.Context, id string) (Loop, error) { return e.store.Get(ctx, id) }
func (e *Engine) List(ctx context.Context) ([]Loop, error)         { return e.store.List(ctx) }
func (e *Engine) Runs(ctx context.Context, id string) ([]Run, error) {
	return e.store.ListRuns(ctx, id)
}

// Pause cancels the loop's goroutine and parks it as paused (resumable).
func (e *Engine) Pause(ctx context.Context, id string) error {
	l, err := e.store.Get(ctx, id)
	if err != nil {
		return err
	}
	if l.Status.IsTerminal() {
		return ErrNotRunnable
	}
	e.cancel(id)
	if err := e.store.SetStatus(ctx, id, StatusPaused, "paused by operator"); err != nil {
		return err
	}
	l.Status = StatusPaused
	e.publish("autoloop.paused", l, "paused by operator")
	return nil
}

// Resume re-arms a paused loop from its persisted iteration.
func (e *Engine) Resume(ctx context.Context, id string) error {
	l, err := e.store.Get(ctx, id)
	if err != nil {
		return err
	}
	if l.Status != StatusPaused {
		return ErrNotRunnable
	}
	if err := e.start(ctx, l); err != nil {
		return err
	}
	e.publish("autoloop.resumed", l, "")
	return nil
}

// Stop terminates a loop. The driven session is left running so an operator
// can inspect it or take over.
func (e *Engine) Stop(ctx context.Context, id string) error {
	l, err := e.store.Get(ctx, id)
	if err != nil {
		return err
	}
	if l.Status.IsTerminal() {
		return ErrNotRunnable
	}
	e.cancel(id)
	if err := e.store.SetStatus(ctx, id, StatusStopped, "stopped by operator"); err != nil {
		return err
	}
	l.Status = StatusStopped
	e.publish("autoloop.stopped", l, "stopped by operator")
	return nil
}

// ReconcileStartup re-arms loops that were running when the gateway last
// stopped. Paused loops are left for an explicit Resume. Mirrors
// session.Manager.ReconcileStartup.
func (e *Engine) ReconcileStartup(ctx context.Context) error {
	live, err := e.store.ListLive(ctx)
	if err != nil {
		return err
	}
	resumed := 0
	for _, l := range live {
		if l.Status != StatusRunning {
			continue
		}
		if err := e.start(ctx, l); err != nil {
			e.log.Warn("reconcile: failed to re-arm loop", "loop_id", l.ID, "err", err)
			continue
		}
		resumed++
	}
	if resumed > 0 {
		e.log.Info("reconcile: re-armed running loops", "count", resumed)
	}
	return nil
}

// Shutdown cancels every running goroutine WITHOUT changing DB status, so the
// loops stay 'running' and ReconcileStartup resumes them on the next boot.
func (e *Engine) Shutdown(ctx context.Context) error {
	e.mu.Lock()
	e.closed = true
	for id, cancel := range e.active {
		cancel()
		delete(e.active, id)
	}
	e.mu.Unlock()
	done := make(chan struct{})
	go func() { e.wg.Wait(); close(done) }()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// start marks the loop running and launches its goroutine. Idempotent: a loop
// already in the active set is left alone.
func (e *Engine) start(ctx context.Context, l Loop) error {
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return ErrClosed
	}
	if _, ok := e.active[l.ID]; ok {
		e.mu.Unlock()
		return nil
	}
	if err := e.store.MarkRunning(ctx, l.ID); err != nil {
		e.mu.Unlock()
		return err
	}
	// The loop must outlive the request ctx (an HTTP create returns
	// immediately) — drive it from a background ctx we cancel ourselves.
	runCtx, cancel := context.WithCancel(context.Background())
	e.active[l.ID] = cancel
	e.wg.Add(1)
	e.mu.Unlock()

	l.Status = StatusRunning
	e.publish("autoloop.started", l, "")
	go func() {
		defer e.wg.Done()
		e.run(runCtx, l)
		e.mu.Lock()
		delete(e.active, l.ID)
		e.mu.Unlock()
	}()
	return nil
}

func (e *Engine) cancel(id string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if c, ok := e.active[id]; ok {
		c()
		delete(e.active, id)
	}
}

func (e *Engine) run(ctx context.Context, l Loop) {
	switch l.Kind {
	case KindInterval:
		e.runInterval(ctx, l)
	case KindGoal:
		e.runGoal(ctx, l)
	}
}

// runInterval fires Prompt immediately, then once per interval, until a cap is
// hit. Interval loops don't verify; reaching the budget is a clean 'done'.
func (e *Engine) runInterval(ctx context.Context, l Loop) {
	interval := time.Duration(l.IntervalSeconds) * e.intervalUnit
	if interval <= 0 {
		interval = time.Duration(MinIntervalSeconds) * e.intervalUnit
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if done, reason := l.budgetExhausted(e.now()); done {
			e.finish(ctx, &l, StatusDone, reason)
			return
		}
		l.Iteration++
		runID, _ := e.store.AppendRun(ctx, l.ID, l.Iteration, l.Prompt)
		_ = e.store.SaveProgress(ctx, l.ID, l.Iteration, "", "")
		e.publish("autoloop.iteration", l, "")
		if err := e.submit(ctx, l.SessionID, l.Prompt, false); err != nil {
			_ = e.store.FinishRun(ctx, runID, "", "input error: "+err.Error())
			if ctx.Err() != nil {
				return
			}
			e.finish(ctx, &l, StatusEscalated, "session input failed: "+err.Error())
			return
		}
		_ = e.store.FinishRun(ctx, runID, "", "")
		if done, reason := l.budgetExhausted(e.now()); done {
			e.finish(ctx, &l, StatusDone, reason)
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// runGoal feeds the seed prompt, waits for the turn to settle, judges it, and
// continues with the judge's next prompt until done / escalate / a cap.
func (e *Engine) runGoal(ctx context.Context, l Loop) {
	sub, unsub := e.bus.Subscribe("session.turn_completed", 8)
	defer unsub()
	prompt := l.Prompt
	failures := 0
	for {
		if done, reason := l.budgetExhausted(e.now()); done {
			// budget gone without the goal being met: a stop, not a success.
			e.finish(ctx, &l, StatusStopped, reason)
			return
		}
		l.Iteration++
		runID, _ := e.store.AppendRun(ctx, l.ID, l.Iteration, prompt)
		e.publish("autoloop.iteration", l, "")
		if err := e.submit(ctx, l.SessionID, prompt, true); err != nil {
			_ = e.store.FinishRun(ctx, runID, "", "input error: "+err.Error())
			if ctx.Err() != nil {
				return
			}
			e.finish(ctx, &l, StatusEscalated, "session input failed: "+err.Error())
			return
		}

		output, ok := e.waitTurn(ctx, sub, l.SessionID)
		if !ok {
			if ctx.Err() != nil {
				_ = e.store.FinishRun(ctx, runID, "", "interrupted")
				return
			}
			// turn timed out — treat as a failed iteration.
			failures++
			l.LastVerdict, l.LastReason = DecisionFail, "turn timed out"
			_ = e.store.SaveProgress(ctx, l.ID, l.Iteration, l.LastVerdict, l.LastReason)
			_ = e.store.FinishRun(ctx, runID, DecisionFail, "turn timed out")
			if failures >= l.FailureCap {
				e.finish(ctx, &l, StatusEscalated, "failure cap reached (turn timeouts)")
				return
			}
			continue
		}

		verdict, _ := e.judge.Judge(ctx, l, output)
		if ctx.Err() != nil {
			_ = e.store.FinishRun(ctx, runID, "", "interrupted")
			return
		}
		l.LastVerdict, l.LastReason = verdict.Decision, verdict.Reason
		_ = e.store.SaveProgress(ctx, l.ID, l.Iteration, verdict.Decision, verdict.Reason)
		_ = e.store.FinishRun(ctx, runID, verdict.Decision, verdict.Reason)
		e.publish("autoloop.verdict", l, verdict.Reason)

		switch verdict.Decision {
		case DecisionDone:
			e.finish(ctx, &l, StatusDone, verdict.Reason)
			return
		case DecisionEscalate:
			e.finish(ctx, &l, StatusEscalated, verdict.Reason)
			return
		case DecisionFail:
			failures++
			if failures >= l.FailureCap {
				e.finish(ctx, &l, StatusEscalated, "failure cap reached: "+verdict.Reason)
				return
			}
			// retry the same prompt
		case DecisionContinue:
			failures = 0
			if next := strings.TrimSpace(verdict.NextPrompt); next != "" {
				prompt = next
			}
		}
	}
}

// waitTurn blocks until session.turn_completed fires for sessionID, the turn
// times out, or the loop is cancelled. Returns (output, true) on a real turn.
func (e *Engine) waitTurn(ctx context.Context, sub <-chan eventbus.Event, sessionID string) (string, bool) {
	timer := time.NewTimer(e.turnTimeout)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return "", false
		case <-timer.C:
			return "", false
		case ev, ok := <-sub:
			if !ok {
				return "", false
			}
			data, _ := ev.Data.(map[string]any)
			if data == nil {
				continue
			}
			if sid, _ := data["session_id"].(string); sid != sessionID {
				continue
			}
			out, _ := data["recent_output"].(string)
			if out == "" {
				out = e.driver.RecentSnippet(sessionID)
			}
			return out, true
		}
	}
}

// submit types text into the session rune-by-rune then sends Enter on its own
// — the cross-CLI input pattern (a single text+\r write is misread as a paste
// burst by Gemini's Ink input, swallowing the Enter). arm=true first arms the
// turn watcher so the settle fires turn_completed (goal mode consumes it).
func (e *Engine) submit(ctx context.Context, sid, text string, arm bool) error {
	if arm {
		e.driver.ExpectTurn(sid)
	}
	for _, r := range text {
		if err := e.driver.Input(ctx, sid, []byte(string(r))); err != nil {
			return err
		}
		if e.perRuneDelay > 0 {
			select {
			case <-time.After(e.perRuneDelay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	if e.submitDelay > 0 {
		select {
		case <-time.After(e.submitDelay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return e.driver.Input(ctx, sid, []byte{'\r'})
}

// finish persists a terminal status + publishes the matching event. Called
// only on natural termination; cancellation (pause/stop/shutdown) returns
// without touching status (the caller already set it, or shutdown leaves it
// running for reconcile).
func (e *Engine) finish(ctx context.Context, l *Loop, st Status, reason string) {
	l.Status = st
	l.LastReason = reason
	wctx := ctx
	if wctx.Err() != nil {
		wctx = context.Background()
	}
	if err := e.store.SetStatus(wctx, l.ID, st, reason); err != nil {
		e.log.Warn("finish: set status failed", "loop_id", l.ID, "status", st, "err", err)
	}
	e.publish("autoloop."+string(st), *l, reason)
}

func (e *Engine) publish(topic string, l Loop, reason string) {
	e.bus.Publish(eventbus.Event{
		Topic: topic,
		Data: map[string]any{
			"loop_id":    l.ID,
			"session_id": l.SessionID,
			"kind":       string(l.Kind),
			"status":     string(l.Status),
			"iteration":  l.Iteration,
			"verdict":    l.LastVerdict,
			"reason":     reason,
		},
	})
}

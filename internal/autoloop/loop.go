// Package autoloop is the Loop Engine: a gateway-level orchestration layer
// that drives an agent session repeatedly — on a fixed interval, or until a
// judged goal is met — turning opendray from a "tool you prompt" into a
// "system that prompts the agent" (the 2026 Loop Engineering shift).
//
// A Loop is 1:1 with a session.Manager session. The engine drives that session
// at the PTY layer (ExpectTurn + Input + the session.turn_completed event), so
// a loop works identically over every provider opendray supports —
// claude / codex / gemini / antigravity / opencode. Loop holds no provider
// field; the bound session determines the CLI.
package autoloop

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"
)

// Kind selects the loop's trigger + decision model. Both share one lifecycle.
type Kind string

const (
	// KindInterval re-feeds Prompt every IntervalSeconds; it has no goal
	// verification and ends when a cap (max iterations / deadline) is hit.
	KindInterval Kind = "interval"
	// KindGoal feeds Prompt as a seed, waits for the turn to settle, then
	// runs the judge worker to decide continue|done|escalate|fail.
	KindGoal Kind = "goal"
)

// ValidKind reports whether k is a recognised kind.
func ValidKind(k Kind) bool { return k == KindInterval || k == KindGoal }

// Status is the loop's lifecycle state, persisted in session_loops.status.
type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusPaused    Status = "paused"
	StatusDone      Status = "done"      // ran its course / goal met
	StatusStopped   Status = "stopped"   // operator stop, or budget exhausted without goal
	StatusFailed    Status = "failed"    // unrecoverable
	StatusEscalated Status = "escalated" // handed to a human (failure cap / judge escalate)
)

// IsTerminal reports whether the loop has finished and will not run again
// without an explicit resume.
func (s Status) IsTerminal() bool {
	switch s {
	case StatusDone, StatusStopped, StatusFailed, StatusEscalated:
		return true
	}
	return false
}

// Origin mirrors session.Origin: who created the loop.
type Origin string

const (
	OriginOperator    Origin = "operator"
	OriginIntegration Origin = "integration"
)

// Verdict decisions returned by the goal-mode judge.
const (
	DecisionContinue = "continue"
	DecisionDone     = "done"
	DecisionEscalate = "escalate"
	DecisionFail     = "fail"
)

// Guardrail defaults. The interval floor and a mandatory deadline are the
// core "don't burn tokens forever" protections (verification + stopping
// conditions are the hard part of loop engineering).
const (
	// MinIntervalSeconds floors interval loops so a misconfigured loop can't
	// hammer a CLI dozens of times a second.
	MinIntervalSeconds   = 30
	DefaultMaxIterations = 20
	DefaultFailureCap    = 3
	// DefaultJudgeTask is the worker touchpoint a goal loop uses to verify a
	// turn when the caller doesn't name one. Provider-agnostic summarizer.
	DefaultJudgeTask = "loop_judge"
)

// Loop is one persistent autonomous loop.
type Loop struct {
	ID              string     `json:"id"`
	SessionID       string     `json:"session_id"`
	Origin          Origin     `json:"origin"`
	IntegrationID   string     `json:"integration_id,omitempty"`
	Kind            Kind       `json:"kind"`
	Status          Status     `json:"status"`
	Goal            string     `json:"goal,omitempty"`
	Prompt          string     `json:"prompt"`
	IntervalSeconds int        `json:"interval_seconds,omitempty"`
	MaxIterations   int        `json:"max_iterations"`
	DeadlineAt      *time.Time `json:"deadline_at,omitempty"`
	FailureCap      int        `json:"failure_cap"`
	JudgeTask       string     `json:"judge_task,omitempty"`
	Iteration       int        `json:"iteration"`
	LastVerdict     string     `json:"last_verdict,omitempty"`
	LastReason      string     `json:"last_reason,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	EndedAt         *time.Time `json:"ended_at,omitempty"`
}

// CreateRequest is the validated input for creating a loop.
type CreateRequest struct {
	SessionID       string
	Origin          Origin
	IntegrationID   string
	Kind            Kind
	Goal            string
	Prompt          string
	IntervalSeconds int
	MaxIterations   int
	DeadlineAt      *time.Time
	FailureCap      int
	JudgeTask       string
}

// Validation errors. Stable so handlers can map them to 4xx codes.
var (
	ErrEmptySession = errors.New("autoloop: session_id required")
	ErrEmptyPrompt  = errors.New("autoloop: prompt required")
	ErrBadKind      = errors.New("autoloop: kind must be 'interval' or 'goal'")
	ErrBadOrigin    = errors.New("autoloop: origin must be 'operator' or 'integration'")
	ErrNoDeadline   = errors.New("autoloop: deadline_at is required")
	ErrPastDeadline = errors.New("autoloop: deadline_at must be in the future")
	ErrBadInterval  = errors.New("autoloop: interval_seconds must be >= 30 for an interval loop")
	ErrNotFound     = errors.New("autoloop: loop not found")
	ErrNotRunnable  = errors.New("autoloop: loop is in a terminal state")
)

// normalize fills defaults that don't depend on validation. Pure: returns a
// new request, never mutates the input (immutability).
func (r CreateRequest) normalize() CreateRequest {
	out := r
	if out.Origin == "" {
		out.Origin = OriginOperator
	}
	if out.MaxIterations <= 0 {
		out.MaxIterations = DefaultMaxIterations
	}
	if out.FailureCap <= 0 {
		out.FailureCap = DefaultFailureCap
	}
	if out.Kind == KindGoal && out.JudgeTask == "" {
		out.JudgeTask = DefaultJudgeTask
	}
	if out.Kind == KindInterval {
		// interval loops don't verify; clear any stray judge task.
		out.JudgeTask = ""
	}
	return out
}

// validate checks the normalized request against the guardrails. now is
// injected so the deadline check is deterministic in tests.
func (r CreateRequest) validate(now time.Time) error {
	if r.SessionID == "" {
		return ErrEmptySession
	}
	if r.Prompt == "" {
		return ErrEmptyPrompt
	}
	if !ValidKind(r.Kind) {
		return ErrBadKind
	}
	if r.Origin != OriginOperator && r.Origin != OriginIntegration {
		return ErrBadOrigin
	}
	// deadline_at is mandatory: guardrail-first. A loop with no deadline and
	// only an iteration cap can still run for an unbounded wall-clock time.
	if r.DeadlineAt == nil {
		return ErrNoDeadline
	}
	if !r.DeadlineAt.After(now) {
		return ErrPastDeadline
	}
	if r.Kind == KindInterval && r.IntervalSeconds < MinIntervalSeconds {
		return ErrBadInterval
	}
	return nil
}

// newLoop builds a pending Loop from a normalized+validated request.
func newLoop(r CreateRequest, now time.Time) Loop {
	return Loop{
		ID:              newID(),
		SessionID:       r.SessionID,
		Origin:          r.Origin,
		IntegrationID:   r.IntegrationID,
		Kind:            r.Kind,
		Status:          StatusPending,
		Goal:            r.Goal,
		Prompt:          r.Prompt,
		IntervalSeconds: r.IntervalSeconds,
		MaxIterations:   r.MaxIterations,
		DeadlineAt:      r.DeadlineAt,
		FailureCap:      r.FailureCap,
		JudgeTask:       r.JudgeTask,
		CreatedAt:       now,
	}
}

// budgetExhausted reports whether a loop has hit a stopping cap: it has run
// at least MaxIterations turns, or passed its deadline. Returns the reason.
func (l Loop) budgetExhausted(now time.Time) (bool, string) {
	if l.MaxIterations > 0 && l.Iteration >= l.MaxIterations {
		return true, "max_iterations reached"
	}
	if l.DeadlineAt != nil && !now.Before(*l.DeadlineAt) {
		return true, "deadline reached"
	}
	return false, ""
}

func newID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		// rand.Read never fails on supported platforms; fall back to a
		// time-seeded value so we still return a usable id.
		return "lp_" + hex.EncodeToString([]byte(time.Now().Format("150405.000000")))
	}
	return "lp_" + hex.EncodeToString(b[:])
}

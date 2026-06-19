package autoloop

import (
	"errors"
	"testing"
	"time"
)

func TestCreateRequestValidate(t *testing.T) {
	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	future := now.Add(time.Hour)
	past := now.Add(-time.Hour)

	tests := []struct {
		name string
		req  CreateRequest
		want error
	}{
		{
			name: "valid interval",
			req:  CreateRequest{SessionID: "s1", Kind: KindInterval, Prompt: "p", IntervalSeconds: 30, DeadlineAt: &future},
			want: nil,
		},
		{
			name: "valid goal",
			req:  CreateRequest{SessionID: "s1", Kind: KindGoal, Prompt: "p", Goal: "g", DeadlineAt: &future},
			want: nil,
		},
		{
			name: "missing session",
			req:  CreateRequest{Kind: KindGoal, Prompt: "p", DeadlineAt: &future},
			want: ErrEmptySession,
		},
		{
			name: "missing prompt",
			req:  CreateRequest{SessionID: "s1", Kind: KindGoal, DeadlineAt: &future},
			want: ErrEmptyPrompt,
		},
		{
			name: "bad kind",
			req:  CreateRequest{SessionID: "s1", Kind: "weird", Prompt: "p", DeadlineAt: &future},
			want: ErrBadKind,
		},
		{
			name: "missing deadline",
			req:  CreateRequest{SessionID: "s1", Kind: KindGoal, Prompt: "p"},
			want: ErrNoDeadline,
		},
		{
			name: "past deadline",
			req:  CreateRequest{SessionID: "s1", Kind: KindGoal, Prompt: "p", DeadlineAt: &past},
			want: ErrPastDeadline,
		},
		{
			name: "interval below floor",
			req:  CreateRequest{SessionID: "s1", Kind: KindInterval, Prompt: "p", IntervalSeconds: 5, DeadlineAt: &future},
			want: ErrBadInterval,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.req.normalize().validate(now)
			if !errors.Is(got, tc.want) {
				t.Fatalf("validate() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNormalizeDefaults(t *testing.T) {
	got := CreateRequest{SessionID: "s1", Kind: KindGoal, Prompt: "p"}.normalize()
	if got.Origin != OriginOperator {
		t.Errorf("origin = %q, want operator", got.Origin)
	}
	if got.MaxIterations != DefaultMaxIterations {
		t.Errorf("max iterations = %d, want %d", got.MaxIterations, DefaultMaxIterations)
	}
	if got.FailureCap != DefaultFailureCap {
		t.Errorf("failure cap = %d, want %d", got.FailureCap, DefaultFailureCap)
	}
	if got.JudgeTask != DefaultJudgeTask {
		t.Errorf("judge task = %q, want %q", got.JudgeTask, DefaultJudgeTask)
	}
}

func TestNormalizeIntervalClearsJudge(t *testing.T) {
	got := CreateRequest{SessionID: "s1", Kind: KindInterval, Prompt: "p", JudgeTask: "loop_judge"}.normalize()
	if got.JudgeTask != "" {
		t.Errorf("interval judge task = %q, want empty", got.JudgeTask)
	}
}

func TestBudgetExhausted(t *testing.T) {
	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	future := now.Add(time.Hour)
	past := now.Add(-time.Minute)

	if done, _ := (Loop{MaxIterations: 3, Iteration: 2, DeadlineAt: &future}).budgetExhausted(now); done {
		t.Error("iteration 2 of 3 should not be exhausted")
	}
	if done, reason := (Loop{MaxIterations: 3, Iteration: 3, DeadlineAt: &future}).budgetExhausted(now); !done || reason == "" {
		t.Errorf("iteration 3 of 3 should be exhausted, got done=%v reason=%q", done, reason)
	}
	if done, reason := (Loop{MaxIterations: 99, Iteration: 1, DeadlineAt: &past}).budgetExhausted(now); !done || reason == "" {
		t.Errorf("past deadline should be exhausted, got done=%v reason=%q", done, reason)
	}
}

func TestStatusIsTerminal(t *testing.T) {
	terminal := []Status{StatusDone, StatusStopped, StatusFailed, StatusEscalated}
	for _, s := range terminal {
		if !s.IsTerminal() {
			t.Errorf("%q should be terminal", s)
		}
	}
	for _, s := range []Status{StatusPending, StatusRunning, StatusPaused} {
		if s.IsTerminal() {
			t.Errorf("%q should not be terminal", s)
		}
	}
}

func TestVerdictNormalise(t *testing.T) {
	if got := (Verdict{Decision: "bogus"}).normalise(); got.Decision != DecisionEscalate {
		t.Errorf("unknown decision should normalise to escalate, got %q", got.Decision)
	}
	if got := (Verdict{Decision: DecisionDone}).normalise(); got.Decision != DecisionDone {
		t.Errorf("valid decision should pass through, got %q", got.Decision)
	}
}

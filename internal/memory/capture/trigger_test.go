package capture

import (
	"testing"
	"time"
)

func TestAfterMessagesTrigger_Evaluate(t *testing.T) {
	cases := []struct {
		name     string
		n        int
		lastSeen int
		current  int
		wantFire bool
	}{
		{"first_n_msgs_fires_at_n", 3, -1, 3, true},
		{"first_n_minus_one_does_not_fire", 3, -1, 2, false},
		{"need_n_new_after_last_fire", 5, 4, 9, false},
		{"exact_n_new", 5, 4, 10, true},
		{"more_than_n_new", 5, 4, 100, true},
		{"default_n_when_zero", 0, -1, 6, true},
		{"default_n_when_zero_short", 0, -1, 5, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := AfterMessagesTrigger{N: c.n}.Evaluate(EvaluationInputs{
				LastSeenIndex:       c.lastSeen,
				CurrentMessageCount: c.current,
			})
			if got != c.wantFire {
				t.Errorf("got fire=%v want %v", got, c.wantFire)
			}
		})
	}
}

func TestOnIdleTrigger_Evaluate(t *testing.T) {
	now := time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC)
	cases := []struct {
		name     string
		seconds  int
		lastMsg  time.Time
		lastSeen int
		current  int
		wantFire bool
	}{
		{"default_60s_not_idle", 0, now.Add(-30 * time.Second), 0, 5, false},
		{"60s_idle_with_new_msgs", 0, now.Add(-90 * time.Second), 0, 5, true},
		{"60s_idle_no_new_msgs", 0, now.Add(-90 * time.Second), 4, 5, false},
		{"no_msgs_yet", 0, time.Time{}, -1, 0, false},
		{"30s_threshold_just_under", 30, now.Add(-29 * time.Second), -1, 5, false},
		{"30s_threshold_at", 30, now.Add(-30 * time.Second), -1, 5, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := OnIdleTrigger{Seconds: c.seconds}.Evaluate(EvaluationInputs{
				LastSeenIndex:       c.lastSeen,
				CurrentMessageCount: c.current,
				LastMessageAt:       c.lastMsg,
				Now:                 now,
			})
			if got != c.wantFire {
				t.Errorf("got fire=%v want %v", got, c.wantFire)
			}
		})
	}
}

func TestKCharsTrigger_Evaluate(t *testing.T) {
	cases := []struct {
		k        int
		chars    int
		wantFire bool
	}{
		{0, 4000, true},
		{0, 3999, false},
		{100, 99, false},
		{100, 100, true},
		{100, 101, true},
	}
	for _, c := range cases {
		got := KCharsTrigger{K: c.k}.Evaluate(EvaluationInputs{CharsSinceLastFire: c.chars})
		if got != c.wantFire {
			t.Errorf("k=%d chars=%d got %v want %v", c.k, c.chars, got, c.wantFire)
		}
	}
}

func TestManualTrigger_NeverAutoFires(t *testing.T) {
	if (ManualTrigger{}).Evaluate(EvaluationInputs{LastSeenIndex: 0, CurrentMessageCount: 1000}) {
		t.Error("manual trigger should never fire from auto-evaluation")
	}
}

func TestTriggerFromRule_AllKinds(t *testing.T) {
	cases := []struct {
		kind     string
		expectKind string
	}{
		{"after_messages", "AfterMessagesTrigger"},
		{"on_idle", "OnIdleTrigger"},
		{"k_chars", "KCharsTrigger"},
		{"manual", "ManualTrigger"},
	}
	for _, c := range cases {
		t.Run(c.kind, func(t *testing.T) {
			r := Rule{TriggerKind: c.kind, TriggerConfig: map[string]any{}}
			tg, err := triggerFromRule(r)
			if err != nil {
				t.Fatal(err)
			}
			gotKind := simpleTypeName(tg)
			if gotKind != c.expectKind {
				t.Errorf("got %s, want %s", gotKind, c.expectKind)
			}
		})
	}
}

func TestTriggerFromRule_UnsupportedKind(t *testing.T) {
	_, err := triggerFromRule(Rule{TriggerKind: "made_up_kind"})
	if err == nil {
		t.Error("expected error for unsupported kind")
	}
}

func simpleTypeName(t any) string {
	switch t.(type) {
	case AfterMessagesTrigger:
		return "AfterMessagesTrigger"
	case OnIdleTrigger:
		return "OnIdleTrigger"
	case KCharsTrigger:
		return "KCharsTrigger"
	case ManualTrigger:
		return "ManualTrigger"
	}
	return "unknown"
}

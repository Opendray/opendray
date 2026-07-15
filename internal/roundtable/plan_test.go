package roundtable

import (
	"testing"
)

func TestParsePlan(t *testing.T) {
	seats := []Seat{
		{Provider: "claude", Model: "opus"},
		{Provider: "antigravity"},
		{Provider: "codex", Model: "gpt-5.4-mini"},
	}

	t.Run("parses fenced JSON, inherits seat model, keeps order", func(t *testing.T) {
		reply := "Sure, here's the plan:\n```json\n" +
			`[{"assignee":"claude","task":"write the API"},` +
			`{"assignee":"antigravity","task":"design the UI"},` +
			`{"assignee":"codex","task":"review the code"}]` +
			"\n```\n"
		steps := parsePlan(reply, seats)
		if len(steps) != 3 {
			t.Fatalf("want 3 steps, got %d", len(steps))
		}
		if steps[0].Assignee != "claude" || steps[0].Model != "opus" {
			t.Errorf("step 0 should be claude/opus, got %s/%s", steps[0].Assignee, steps[0].Model)
		}
		if steps[2].Assignee != "codex" || steps[2].Status != StepPending {
			t.Errorf("step 2 should be codex/pending, got %s/%s", steps[2].Assignee, steps[2].Status)
		}
	})

	t.Run("unknown assignee falls back to first seat", func(t *testing.T) {
		steps := parsePlan(`[{"assignee":"gemini","task":"do a thing"}]`, seats)
		if len(steps) != 1 || steps[0].Assignee != "claude" {
			t.Errorf("unknown assignee should fall back to first seat, got %v", steps)
		}
	})

	t.Run("drops empty tasks; nil on no JSON", func(t *testing.T) {
		if got := parsePlan(`[{"assignee":"claude","task":"  "}]`, seats); len(got) != 0 {
			t.Errorf("empty task should be dropped, got %v", got)
		}
		if got := parsePlan("no json here", seats); got != nil {
			t.Errorf("no JSON should yield nil, got %v", got)
		}
	})
}

func TestNormalizePlan(t *testing.T) {
	in := []PlanStep{
		{Assignee: "claude", Task: " build ", Status: "bogus"},
		{Assignee: "", Task: "no assignee"}, // dropped
		{Assignee: "codex", Task: ""},       // dropped
	}
	out, err := normalizePlan(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("want 1 valid step, got %d", len(out))
	}
	if out[0].Task != "build" || out[0].Status != StepPending {
		t.Errorf("step should be trimmed + defaulted to pending, got %q/%q", out[0].Task, out[0].Status)
	}
}

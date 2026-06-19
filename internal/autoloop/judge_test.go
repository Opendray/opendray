package autoloop

import (
	"strings"
	"testing"
)

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"clean", `{"decision":"done"}`, `{"decision":"done"}`},
		{"prose wrapped", "Here is my verdict: {\"decision\":\"done\"} hope that helps", `{"decision":"done"}`},
		{"fenced", "```json\n{\"decision\":\"continue\"}\n```", `{"decision":"continue"}`},
		{"no json", "nothing here", "nothing here"},
		{"nested", `prefix {"a":{"b":1}} suffix`, `{"a":{"b":1}}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractJSON(tc.in); got != tc.want {
				t.Errorf("extractJSON(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestBuildJudgeInput(t *testing.T) {
	l := Loop{Goal: "ship the feature", Iteration: 2, MaxIterations: 10,
		LastVerdict: DecisionContinue, LastReason: "made progress"}
	got := buildJudgeInput(l, "the agent did X")
	for _, want := range []string{"ship the feature", "2 of 10", "made progress", "the agent did X"} {
		if !strings.Contains(got, want) {
			t.Errorf("judge input missing %q\n--- got ---\n%s", want, got)
		}
	}
}

func TestBuildJudgeInputEmptyOutput(t *testing.T) {
	got := buildJudgeInput(Loop{Goal: "g", MaxIterations: 5}, "   ")
	if !strings.Contains(got, "no visible output") {
		t.Errorf("empty output should be noted, got:\n%s", got)
	}
}

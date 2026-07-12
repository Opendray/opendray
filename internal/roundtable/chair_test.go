package roundtable

import (
	"testing"
)

// synthesize is the deterministic heart of the chair. These tests pin its
// ranking rules so the Verdict stays fully reproducible.

func props(m map[string]proposal) map[string]proposal { return m }

func TestSynthesize_FewestBlockersWins(t *testing.T) {
	proposals := props(map[string]proposal{
		"claude": {Summary: "A", Plan: "plan-A", Confidence: 0.9},
		"codex":  {Summary: "B", Plan: "plan-B", Confidence: 0.5},
	})
	// claude's proposal takes 2 blockers; codex takes none.
	critiques := []critique{
		{TargetProvider: "claude", Severity: "blocker", Point: "x"},
		{TargetProvider: "claude", Severity: "blocker", Point: "y"},
	}
	v := synthesize(proposals, []string{"claude", "codex"}, critiques)
	if v.RecommendedBy != "codex" {
		t.Fatalf("want codex recommended (0 blockers beats 2), got %q", v.RecommendedBy)
	}
	if v.Recommended != "plan-B" {
		t.Errorf("recommended plan = %q, want plan-B", v.Recommended)
	}
	if len(v.Alternatives) != 1 || v.Alternatives[0] != "claude: A" {
		t.Errorf("alternatives = %v, want [claude: A]", v.Alternatives)
	}
}

func TestSynthesize_ConfidenceBreaksTie(t *testing.T) {
	proposals := props(map[string]proposal{
		"claude":      {Summary: "A", Plan: "pa", Confidence: 0.6},
		"codex":       {Summary: "B", Plan: "pb", Confidence: 0.8},
		"antigravity": {Summary: "C", Plan: "pc", Confidence: 0.7},
	})
	// No critiques → tie on blockers+concerns → highest confidence wins.
	v := synthesize(proposals, []string{"claude", "codex", "antigravity"}, nil)
	if v.RecommendedBy != "codex" {
		t.Fatalf("want codex (highest confidence), got %q", v.RecommendedBy)
	}
}

func TestSynthesize_ConcernsBreakBlockerTie(t *testing.T) {
	proposals := props(map[string]proposal{
		"claude": {Summary: "A", Plan: "pa", Confidence: 0.9},
		"codex":  {Summary: "B", Plan: "pb", Confidence: 0.9},
	})
	// Equal blockers (0), claude has a concern → codex wins.
	critiques := []critique{
		{TargetProvider: "claude", Severity: "concern", Point: "z"},
	}
	v := synthesize(proposals, []string{"claude", "codex"}, critiques)
	if v.RecommendedBy != "codex" {
		t.Fatalf("want codex (fewer concerns), got %q", v.RecommendedBy)
	}
}

func TestSynthesize_SeatOrderIsFinalTiebreak(t *testing.T) {
	proposals := props(map[string]proposal{
		"claude": {Summary: "A", Plan: "pa", Confidence: 0.5},
		"codex":  {Summary: "B", Plan: "pb", Confidence: 0.5},
	})
	// Everything equal → first seat in order wins, deterministically.
	v := synthesize(proposals, []string{"codex", "claude"}, nil)
	if v.RecommendedBy != "codex" {
		t.Fatalf("want codex (first in seat order), got %q", v.RecommendedBy)
	}
}

func TestSynthesize_OpenQuestionsIncludeBlockersAndConcernsNotNits(t *testing.T) {
	proposals := props(map[string]proposal{
		"claude": {Summary: "A", Plan: "pa", Confidence: 0.9},
		"codex":  {Summary: "B", Plan: "pb", Confidence: 0.5},
	})
	critiques := []critique{
		{TargetProvider: "codex", Severity: "blocker", Point: "big"},
		{TargetProvider: "codex", Severity: "concern", Point: "med"},
		{TargetProvider: "codex", Severity: "nit", Point: "small"},
	}
	v := synthesize(proposals, []string{"claude", "codex"}, critiques)
	if len(v.OpenQuestions) != 2 {
		t.Fatalf("want 2 open questions (blocker+concern, not nit), got %d: %v", len(v.OpenQuestions), v.OpenQuestions)
	}
}

func TestSynthesize_TradeoffsUnionDedup(t *testing.T) {
	proposals := props(map[string]proposal{
		"claude": {Summary: "A", Plan: "pa", Tradeoffs: []string{"cost", "risk"}},
		"codex":  {Summary: "B", Plan: "pb", Tradeoffs: []string{"risk", "time"}},
	})
	v := synthesize(proposals, []string{"claude", "codex"}, nil)
	// Union of {cost,risk} and {risk,time} deduped = 3.
	if len(v.Tradeoffs) != 3 {
		t.Fatalf("want 3 deduped tradeoffs, got %d: %v", len(v.Tradeoffs), v.Tradeoffs)
	}
}

func TestSynthesize_EmptyProposalsNoPanic(t *testing.T) {
	v := synthesize(map[string]proposal{}, nil, nil)
	if v.RecommendedBy != "" || v.Recommended != "" || len(v.Ranking) != 0 {
		t.Errorf("empty proposals should yield empty verdict, got %+v", v)
	}
}

func TestSynthesize_WinnerTaskBreakdownCarried(t *testing.T) {
	proposals := props(map[string]proposal{
		"claude": {Summary: "A", Plan: "pa", Tasks: []string{"t1", "t2"}, Confidence: 0.9},
		"codex":  {Summary: "B", Plan: "pb", Tasks: []string{"x"}, Confidence: 0.1},
	})
	v := synthesize(proposals, []string{"claude", "codex"}, nil)
	if len(v.TaskBreakdown) != 2 || v.TaskBreakdown[0] != "t1" {
		t.Errorf("want winner (claude) tasks [t1 t2], got %v", v.TaskBreakdown)
	}
}

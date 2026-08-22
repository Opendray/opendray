package session

import (
	"strings"
	"testing"
)

// Grok accepts --rules at most once per invocation. Several independent
// spawn injections each emit a fragment, and a caller may pass one more
// in the request args; CoalesceRulesArgs merges them into the single
// slot grok allows so the spawn doesn't fail with "--rules cannot be
// used multiple times".

func TestCoalesceRulesArgs_MergesMultiple(t *testing.T) {
	in := []string{"--model", "grok-build", "--rules", "SKILLS", "--foo", "--rules", "GUIDANCE"}
	got := CoalesceRulesArgs(in)
	want := []string{"--model", "grok-build", "--rules", "SKILLS\n\n---\n\nGUIDANCE", "--foo"}
	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Errorf("merge wrong:\n got  %q\n want %q", got, want)
	}
}

func TestCoalesceRulesArgs_SinglePreserved(t *testing.T) {
	in := []string{"--rules", "ONLY"}
	got := CoalesceRulesArgs(in)
	if len(got) != 2 || got[0] != "--rules" || got[1] != "ONLY" {
		t.Errorf("single --rules should be untouched; got %q", got)
	}
}

func TestCoalesceRulesArgs_NoneNoop(t *testing.T) {
	in := []string{"--model", "grok-build", "--foo"}
	got := CoalesceRulesArgs(in)
	if strings.Join(got, "\x00") != strings.Join(in, "\x00") {
		t.Errorf("no --rules should be a no-op; got %q", got)
	}
}

func TestCoalesceRulesArgs_TrailingFlagNoValueDropped(t *testing.T) {
	in := []string{"--rules", "A", "--rules"} // malformed trailing flag, no value
	got := CoalesceRulesArgs(in)
	if len(got) != 2 || got[0] != "--rules" || got[1] != "A" {
		t.Errorf("dangling --rules should be dropped, value kept; got %q", got)
	}
}

func TestCoalesceRulesArgs_DoesNotEatFollowingFlag(t *testing.T) {
	// A bare --rules adjacent to another flag must not consume that flag
	// as its value.
	in := []string{"--rules", "--model", "grok-build", "--rules", "REAL"}
	got := CoalesceRulesArgs(in)
	want := []string{"--rules", "REAL", "--model", "grok-build"}
	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Errorf("adjacent flag eaten:\n got  %q\n want %q", got, want)
	}
}

func TestFinalizeSpawnArgs_GrokCallerRulesMergedIntoOne(t *testing.T) {
	// The Prepare-level coalesce cannot see userArgs; the finalizer must
	// still guarantee exactly one --rules on the assembled command line.
	provider := []string{"--model", "grok-build"}
	extra := []string{"--rules", "INJECTED"} // already coalesced by Prepare
	user := []string{"--rules", "CALLER"}
	got := finalizeSpawnArgs("grok", provider, extra, user)

	n := 0
	for _, a := range got {
		if a == "--rules" {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("grok must get exactly one --rules; got %d in %q", n, got)
	}
	merged := ""
	for i, a := range got {
		if a == "--rules" {
			merged = got[i+1]
		}
	}
	if !strings.Contains(merged, "INJECTED") || !strings.Contains(merged, "CALLER") {
		t.Errorf("merged --rules must carry both fragments; got %q", merged)
	}
}

func TestFinalizeSpawnArgs_NonGrokUntouched(t *testing.T) {
	provider := []string{"--model", "opus"}
	extra := []string{"--append-system-prompt", "A", "--append-system-prompt", "B"}
	user := []string{"--resume", "abc"}
	got := finalizeSpawnArgs("claude", provider, extra, user)
	want := []string{"--model", "opus", "--append-system-prompt", "A", "--append-system-prompt", "B", "--resume", "abc"}
	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Errorf("non-grok args must pass through verbatim:\n got  %q\n want %q", got, want)
	}
}

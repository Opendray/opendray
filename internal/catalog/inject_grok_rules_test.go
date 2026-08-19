package catalog

import (
	"strings"
	"testing"

	"github.com/opendray/opendray-v2/internal/session"
	"github.com/opendray/opendray-v2/internal/skills"
)

// Grok accepts --rules at most once per invocation. opendray has several
// independent system-text injectors (skills index, global instruction, …)
// that each emit a --rules for grok; coalesceRulesArgs merges them into
// the single slot grok allows, so grok gets every fragment without the
// "--rules cannot be used multiple times" spawn failure.

func TestCoalesceRulesArgs_MergesMultiple(t *testing.T) {
	in := []string{"--model", "grok-4", "--rules", "SKILLS", "--foo", "--rules", "GLOBAL"}
	got := coalesceRulesArgs(in)
	want := []string{"--model", "grok-4", "--rules", "SKILLS\n\n---\n\nGLOBAL", "--foo"}
	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Errorf("merge wrong:\n got  %q\n want %q", got, want)
	}
}

func TestCoalesceRulesArgs_SinglePreserved(t *testing.T) {
	in := []string{"--rules", "ONLY"}
	got := coalesceRulesArgs(in)
	if len(got) != 2 || got[0] != "--rules" || got[1] != "ONLY" {
		t.Errorf("single --rules should be untouched; got %q", got)
	}
}

func TestCoalesceRulesArgs_NoneNoop(t *testing.T) {
	in := []string{"--model", "grok-4", "--foo"}
	got := coalesceRulesArgs(in)
	if strings.Join(got, "\x00") != strings.Join(in, "\x00") {
		t.Errorf("no --rules should be a no-op; got %q", got)
	}
}

func TestCoalesceRulesArgs_TrailingFlagNoValueDropped(t *testing.T) {
	in := []string{"--rules", "A", "--rules"} // malformed trailing flag, no value
	got := coalesceRulesArgs(in)
	if len(got) != 2 || got[0] != "--rules" || got[1] != "A" {
		t.Errorf("dangling --rules should be dropped, value kept; got %q", got)
	}
}

func TestInjectSkillsFor_Grok(t *testing.T) {
	out := &session.PrepareOutput{}
	err := injectSkillsFor("grok", t.TempDir(), []skills.Skill{
		{ID: "brainstorming", Name: "brainstorming", Description: "explore intent"},
	}, out)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Args) != 2 || out.Args[0] != "--rules" {
		t.Fatalf("grok: want [--rules <index>], got %q", out.Args)
	}
	if !strings.Contains(out.Args[1], "brainstorming") {
		t.Errorf("grok: --rules value missing skill; got %q", out.Args[1])
	}
}

func TestProviderSupportsSkills_Grok(t *testing.T) {
	if !providerSupportsSkills("grok") {
		t.Error("grok should support skills so the index auto-injects at spawn")
	}
}

// The real spawn scenario: both the skill index and the global instruction
// emit a grok --rules; the finalizer must leave exactly one --rules that
// carries both, or grok fails to start.
func TestGrokSkillsAndGlobalInstruction_CoalesceToOne(t *testing.T) {
	out := &session.PrepareOutput{}
	if err := injectSkillsFor("grok", t.TempDir(), []skills.Skill{
		{ID: "brainstorming", Name: "brainstorming", Description: "explore intent"},
	}, out); err != nil {
		t.Fatal(err)
	}
	if err := injectGlobalInstructionFor("grok", t.TempDir(), "HOUSE_STYLE", out); err != nil {
		t.Fatal(err)
	}
	out.Args = coalesceRulesArgs(out.Args)

	n := 0
	for _, a := range out.Args {
		if a == "--rules" {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("grok must get exactly one --rules; got %d in %q", n, out.Args)
	}
	merged := out.Args[len(out.Args)-1]
	if !strings.Contains(merged, "brainstorming") || !strings.Contains(merged, "HOUSE_STYLE") {
		t.Errorf("merged --rules must carry both fragments; got %q", merged)
	}
}

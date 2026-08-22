package catalog

import (
	"strings"
	"testing"

	"github.com/opendray/opendray-v2/internal/session"
	"github.com/opendray/opendray-v2/internal/skills"
)

// Grok is the one provider with no file-based system-prompt surface: it
// appends extra system-prompt text via --rules, which it permits at most
// once per invocation. Every existing spawn injection (memory guidance,
// ambient memory / project docs / knowledge banners, carryover,
// integration boot prompt, skill index) therefore emits its own --rules
// fragment, and session.CoalesceRulesArgs merges them into the single
// slot grok allows — once at the end of Prepare, and once more in the
// manager after caller args are appended. This is what makes grok a
// first-class Cortex citizen — no separate content source needed.

func TestInjectAmbientMemory_Grok(t *testing.T) {
	out := &session.PrepareOutput{}
	if err := injectAmbientMemoryFor("grok", t.TempDir(), "AMBIENT_BANNER", out); err != nil {
		t.Fatal(err)
	}
	if len(out.Args) != 2 || out.Args[0] != "--rules" || out.Args[1] != "AMBIENT_BANNER" {
		t.Errorf("grok: want [--rules AMBIENT_BANNER], got %+v", out.Args)
	}
}

func TestInjectMemoryGuidance_Grok(t *testing.T) {
	out := &session.PrepareOutput{}
	if err := injectMemoryGuidanceFor("grok", t.TempDir(), out); err != nil {
		t.Fatal(err)
	}
	if len(out.Args) != 2 || out.Args[0] != "--rules" || out.Args[1] != memoryGuidanceText {
		t.Errorf("grok: want [--rules <guidance>], got %d args", len(out.Args))
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

// The real spawn scenario: several independent Cortex injections each
// emit a grok --rules fragment; the finalizer must leave exactly one
// --rules carrying all of them, or grok fails to start with "--rules
// cannot be used multiple times".
func TestGrokCortexInjections_CoalesceToOne(t *testing.T) {
	out := &session.PrepareOutput{}
	base := t.TempDir()
	if err := injectSkillsFor("grok", base, []skills.Skill{
		{ID: "brainstorming", Name: "brainstorming", Description: "explore intent"},
	}, out); err != nil {
		t.Fatal(err)
	}
	if err := injectMemoryGuidanceFor("grok", base, out); err != nil {
		t.Fatal(err)
	}
	if err := injectAmbientMemoryFor("grok", base, "KNOWLEDGE_BANNER", out); err != nil {
		t.Fatal(err)
	}
	out.Args = session.CoalesceRulesArgs(out.Args)

	n := 0
	for _, a := range out.Args {
		if a == "--rules" {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("grok must get exactly one --rules; got %d in %d args", n, len(out.Args))
	}
	merged := out.Args[len(out.Args)-1]
	for _, frag := range []string{"brainstorming", memoryGuidanceText, "KNOWLEDGE_BANNER"} {
		if !strings.Contains(merged, frag) {
			t.Errorf("merged --rules missing fragment %q", frag[:min(len(frag), 30)])
		}
	}
}

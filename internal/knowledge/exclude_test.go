package knowledge

import (
	"context"
	"strings"
	"testing"
)

func TestExcluderMatching(t *testing.T) {
	ex := newExcluder([]string{"rcc", "ntc", "远程网关"})
	tests := []struct {
		name string
		line string
		want bool
	}{
		{"exact lowercase", "rcc and the gateway", true},
		{"uppercase", "RCC service location", true},
		{"mixed case", "Rcc plugin system", true},
		{"slash separated", "rcc/ntc are retired", true},
		{"backticked", "predecessors `rcc` / `ntc`", true},
		{"end of line", "unified into rcc", true},
		{"whole line", "ntc", true},
		// The reason for word boundaries: these are different words that
		// merely contain the pattern, and firing on them would quietly eat
		// unrelated content off the page.
		{"substring inside word", "the accrual account", false},
		{"substring prefix", "rccx is a different thing", false},
		{"substring suffix", "xrcc is a different thing", false},
		{"unrelated", "opendray is the gateway", false},
		// CJK has no word breaks, so those patterns match as substrings.
		{"cjk substring", "这个远程网关已经退役", true},
		{"cjk absent", "这个网关正在运行", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ex.hits(tt.line); got != tt.want {
				t.Errorf("hits(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestExcluderEmptyIsNoOp(t *testing.T) {
	for _, pats := range [][]string{nil, {}, {"", "   "}} {
		ex := newExcluder(pats)
		if !ex.empty() {
			t.Fatalf("newExcluder(%q) should be empty", pats)
		}
		const body = "- rcc and ntc are deprecated\n- keep me"
		if got, n := ex.scrub(body); got != body || n != 0 {
			t.Fatalf("empty excluder scrubbed: %q (%d)", got, n)
		}
	}
}

func TestExcluderScrubDropsOnlyOffendingLines(t *testing.T) {
	ex := newExcluder([]string{"rcc"})
	in := "## Gateway\n- opendray is the gateway\n- rcc is deprecated\n- pnpm only\n"
	got, n := ex.scrub(in)
	if n != 1 {
		t.Fatalf("dropped %d lines, want 1", n)
	}
	if strings.Contains(got, "rcc") {
		t.Errorf("excluded subject survived: %q", got)
	}
	for _, keep := range []string{"## Gateway", "opendray is the gateway", "pnpm only"} {
		if !strings.Contains(got, keep) {
			t.Errorf("scrub removed neighbouring content %q from %q", keep, got)
		}
	}
}

func TestExcluderScrubCollapsesBlankRuns(t *testing.T) {
	ex := newExcluder([]string{"rcc"})
	got, _ := ex.scrub("a\n\nrcc line\n\nb\n")
	if strings.Contains(got, "\n\n\n") {
		t.Errorf("scrub left a ragged gap: %q", got)
	}
	// Trailing shape preserved — the signature hashes this string.
	if got != "a\n\nb\n" {
		t.Errorf("got %q, want %q", got, "a\n\nb\n")
	}
}

// Dropping the LAST line must not change the trailing shape either, or a
// sweep that removed only excluded evidence would look like a real change.
func TestExcluderScrubKeepsTrailingShape(t *testing.T) {
	ex := newExcluder([]string{"rcc"})
	const base = "FACTS:\n- opendray is the gateway\n"
	got, n := ex.scrub(base + "- rcc is deprecated\n")
	if n != 1 {
		t.Fatalf("dropped %d lines, want 1", n)
	}
	if got != base {
		t.Errorf("got %q, want it identical to the un-excluded feedstock %q", got, base)
	}
}

func TestSigPartChangesWithList(t *testing.T) {
	if sigPart(nil) != "" || sigPart([]string{" "}) != "" {
		t.Fatal("empty exclusion list must contribute nothing to the signature")
	}
	if sigPart([]string{"rcc"}) == sigPart([]string{"rcc", "ntc"}) {
		t.Fatal("adding an exclusion must change the signature")
	}
}

func TestExcludeInstructionNamesEachSubject(t *testing.T) {
	if excludeInstruction(nil) != "" {
		t.Fatal("no exclusions must add no instruction")
	}
	got := excludeInstruction([]string{"rcc", "ntc"})
	for _, want := range []string{"rcc", "ntc", "EXCLUDED SUBJECTS"} {
		if !strings.Contains(got, want) {
			t.Errorf("instruction missing %q: %q", want, got)
		}
	}
}

// The regression this whole feature exists for: a page that carries an
// excluded subject, and evidence that keeps re-asserting it, must come back
// clean — and the subject must never reach the model in the first place.
func TestDraftScrubsExcludedSubjectFromBothEnds(t *testing.T) {
	llm := &fakeLLM{body: "# Infra\n- opendray is the gateway\n- rcc is deprecated\n"}
	docs := &fakeDocSink{doc: KBDoc{
		Exists:     true,
		Content:    "# Infra\n- opendray is the gateway\n- do not mention rcc\n",
		Exclusions: []string{"rcc"},
	}}
	feedstock := "FACTS:\n- opendray is the gateway\n- rcc and the gateway are unified\n"

	res := draftOrPropose(context.Background(), llm, docs, nil, discardLog(),
		GlobalKBCwd, KBKindInfrastructure, "sys", feedstock,
		draftOpts{honorMaintainerMode: true, preserveCurrent: true, applyPromptHint: true})

	if res.Status != "written" {
		t.Fatalf("status = %q (%s), want written", res.Status, res.Err)
	}
	if strings.Contains(llm.lastUser, "rcc") {
		t.Errorf("excluded subject reached the model as evidence:\n%s", llm.lastUser)
	}
	if strings.Contains(docs.putContent, "rcc") {
		t.Errorf("excluded subject survived into the page:\n%s", docs.putContent)
	}
	if !strings.Contains(docs.putContent, "opendray is the gateway") {
		t.Errorf("scrub ate legitimate content:\n%s", docs.putContent)
	}
}

// New evidence about an excluded subject must not even count as a change:
// it vanishes before the signature is computed, so the page stays skipped
// instead of burning an LLM call and a proposal every sweep.
func TestExcludedFeedstockDoesNotDirtyThePage(t *testing.T) {
	const base = "FACTS:\n- opendray is the gateway\n"
	docs := &fakeDocSink{doc: KBDoc{Exists: true, Exclusions: []string{"rcc"}}}
	llm := &fakeLLM{body: "# Infra\n- opendray is the gateway\n"}

	// First pass writes the page and stamps it with a signature.
	first := draftOrPropose(context.Background(), llm, docs, nil, discardLog(),
		GlobalKBCwd, KBKindInfrastructure, "sys", base,
		draftOpts{honorMaintainerMode: true, preserveCurrent: true, applyPromptHint: true})
	if first.Status != "written" {
		t.Fatalf("first status = %q (%s), want written", first.Status, first.Err)
	}
	docs.doc.Content = docs.putContent

	// Second pass: same evidence plus one new excluded fact.
	callsBefore := llm.calls
	second := draftOrPropose(context.Background(), llm, docs, nil, discardLog(),
		GlobalKBCwd, KBKindInfrastructure, "sys", base+"- rcc is deprecated\n",
		draftOpts{honorMaintainerMode: true, preserveCurrent: true, applyPromptHint: true})
	if second.Status != "skipped-unchanged" {
		t.Fatalf("status = %q, want skipped-unchanged", second.Status)
	}
	if llm.calls != callsBefore {
		t.Errorf("excluded evidence still triggered %d LLM call(s)", llm.calls-callsBefore)
	}
}

// Editing the list has to take effect immediately, or the operator's change
// silently waits for unrelated evidence to move.
func TestChangingExclusionsInvalidatesTheCachedDraft(t *testing.T) {
	const feedstock = "FACTS:\n- opendray is the gateway\n"
	docs := &fakeDocSink{doc: KBDoc{Exists: true}}
	llm := &fakeLLM{body: "# Infra\n- opendray is the gateway\n"}
	opts := draftOpts{honorMaintainerMode: true, preserveCurrent: true, applyPromptHint: true}

	draftOrPropose(context.Background(), llm, docs, nil, discardLog(),
		GlobalKBCwd, KBKindInfrastructure, "sys", feedstock, opts)
	docs.doc.Content = docs.putContent

	if res := draftOrPropose(context.Background(), llm, docs, nil, discardLog(),
		GlobalKBCwd, KBKindInfrastructure, "sys", feedstock, opts); res.Status != "skipped-unchanged" {
		t.Fatalf("unchanged inputs should skip, got %q", res.Status)
	}

	docs.doc.Exclusions = []string{"rcc"}
	if res := draftOrPropose(context.Background(), llm, docs, nil, discardLog(),
		GlobalKBCwd, KBKindInfrastructure, "sys", feedstock, opts); res.Status != "written" {
		t.Fatalf("new exclusion should redraft, got %q", res.Status)
	}
}

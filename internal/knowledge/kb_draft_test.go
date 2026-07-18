package knowledge

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
)

// --- fakes ---------------------------------------------------------------

type fakeLLM struct {
	body       string
	calls      int
	lastSystem string
	lastUser   string
}

func (f *fakeLLM) Complete(_ context.Context, system, user string) (string, error) {
	f.calls++
	f.lastSystem = system
	f.lastUser = user
	return f.body, nil
}

type fakeDocSink struct {
	doc        KBDoc
	putCalls   int
	putContent string
}

func (f *fakeDocSink) GetKBDoc(_ context.Context, _, _ string) (KBDoc, error) { return f.doc, nil }
func (f *fakeDocSink) PutKBDoc(_ context.Context, _, _, content string) error {
	f.putCalls++
	f.putContent = content
	return nil
}

type fakeProposalSink struct {
	pending      bool
	rejected     []string
	proposeCalls int
	proposed     string
}

func (f *fakeProposalSink) HasPendingKBProposal(_ context.Context, _, _ string) (bool, error) {
	return f.pending, nil
}
func (f *fakeProposalSink) ProposeKBDoc(_ context.Context, _, _, content, _ string) error {
	f.proposeCalls++
	f.proposed = content
	return nil
}
func (f *fakeProposalSink) RejectedSigs(_ context.Context, _, _ string) ([]string, error) {
	return f.rejected, nil
}

func discardLog() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

const (
	testCwd  = "__global__"
	testKind = "kb_infrastructure"
	testFeed = "ENTITIES:\nhost: kv01\n\nFACTS:\n- dev db at 192.168.3.88\n"
)

// kbOpts mirrors what the KB drafter passes (operator-owned-form on).
var kbOpts = draftOpts{honorMaintainerMode: true, preserveCurrent: true, applyPromptHint: true}

func run(llm *fakeLLM, docs *fakeDocSink, props *fakeProposalSink, opts draftOpts) KBDraftResult {
	return draftOrPropose(context.Background(), llm, docs, props, discardLog(), testCwd, testKind, "SYS", testFeed, opts)
}

// --- tests ---------------------------------------------------------------

// A locked page whose feedstock has diverged files a proposal (does not
// overwrite) when the draft has not been rejected before.
func TestDraftOrPropose_LockedProposes(t *testing.T) {
	docs := &fakeDocSink{doc: KBDoc{Content: "old\n<!-- kb-sig:0000000000000000 -->\n", HumanLocked: true, Exists: true}}
	props := &fakeProposalSink{}
	res := run(&fakeLLM{body: "fresh body"}, docs, props, kbOpts)

	if res.Status != "proposed" {
		t.Fatalf("status = %q, want proposed", res.Status)
	}
	if props.proposeCalls != 1 {
		t.Fatalf("propose calls = %d, want 1", props.proposeCalls)
	}
	if docs.putCalls != 0 {
		t.Fatalf("locked page must not be overwritten (put calls = %d)", docs.putCalls)
	}
}

// Bug 2 regression: a draft the operator already rejected (same feedstock
// signature) must NOT be re-proposed on the next sweep.
func TestDraftOrPropose_LockedSkipsRejectedSig(t *testing.T) {
	docs := &fakeDocSink{doc: KBDoc{Content: "old\n<!-- kb-sig:0000000000000000 -->\n", HumanLocked: true, Exists: true}}
	props := &fakeProposalSink{rejected: []string{kbSig(testFeed)}}
	res := run(&fakeLLM{body: "fresh body"}, docs, props, kbOpts)

	if res.Status != "skipped-rejected" {
		t.Fatalf("status = %q, want skipped-rejected", res.Status)
	}
	if props.proposeCalls != 0 {
		t.Fatalf("a rejected draft must not be re-proposed (propose calls = %d)", props.proposeCalls)
	}
}

// A rejection for a DIFFERENT feedstock signature does not suppress a genuinely
// new divergence.
func TestDraftOrPropose_LockedProposesWhenRejectionIsStale(t *testing.T) {
	docs := &fakeDocSink{doc: KBDoc{Content: "old\n<!-- kb-sig:0000000000000000 -->\n", HumanLocked: true, Exists: true}}
	props := &fakeProposalSink{rejected: []string{"deadbeefdeadbeef"}}
	res := run(&fakeLLM{body: "fresh body"}, docs, props, kbOpts)

	if res.Status != "proposed" {
		t.Fatalf("status = %q, want proposed", res.Status)
	}
	if props.proposeCalls != 1 {
		t.Fatalf("propose calls = %d, want 1", props.proposeCalls)
	}
}

// When the page already carries the current feedstock signature, the sweep is a
// no-op — no LLM call, no proposal, no overwrite.
func TestDraftOrPropose_SkipsUnchanged(t *testing.T) {
	content := "body\n<!-- kb-sig:" + kbSig(testFeed) + " -->\n"
	docs := &fakeDocSink{doc: KBDoc{Content: content, HumanLocked: true, Exists: true}}
	props := &fakeProposalSink{}
	llm := &fakeLLM{body: "should not be used"}
	res := run(llm, docs, props, kbOpts)

	if res.Status != "skipped-unchanged" {
		t.Fatalf("status = %q, want skipped-unchanged", res.Status)
	}
	if llm.calls != 0 || props.proposeCalls != 0 || docs.putCalls != 0 {
		t.Fatalf("unchanged page must be untouched (llm=%d propose=%d put=%d)", llm.calls, props.proposeCalls, docs.putCalls)
	}
}

// An unlocked (agent-owned) page whose feedstock diverged is rewritten in place.
func TestDraftOrPropose_UnlockedWrites(t *testing.T) {
	docs := &fakeDocSink{doc: KBDoc{Content: "old\n<!-- kb-sig:0000000000000000 -->\n", HumanLocked: false, Exists: true}}
	props := &fakeProposalSink{}
	res := run(&fakeLLM{body: "fresh body"}, docs, props, kbOpts)

	if res.Status != "written" {
		t.Fatalf("status = %q, want written", res.Status)
	}
	if docs.putCalls != 1 {
		t.Fatalf("put calls = %d, want 1", docs.putCalls)
	}
	if props.proposeCalls != 0 {
		t.Fatalf("unlocked page must not propose (propose calls = %d)", props.proposeCalls)
	}
}

// maintainer_mode=human hands the page entirely to the operator: the drafter
// never calls the LLM, proposes, or writes.
func TestDraftOrPropose_HumanModeSkips(t *testing.T) {
	docs := &fakeDocSink{doc: KBDoc{Content: "operator's page", HumanLocked: true, Exists: true, MaintainerMode: "human"}}
	props := &fakeProposalSink{}
	llm := &fakeLLM{body: "should not be used"}
	res := run(llm, docs, props, kbOpts)

	if res.Status != "skipped-human" {
		t.Fatalf("status = %q, want skipped-human", res.Status)
	}
	if llm.calls != 0 || props.proposeCalls != 0 || docs.putCalls != 0 {
		t.Fatalf("human page must be untouched (llm=%d propose=%d put=%d)", llm.calls, props.proposeCalls, docs.putCalls)
	}
}

// honorMaintainerMode is opt-in: with the zero opts (Overview), maintainer_mode
// is ignored and the page is drafted as before.
func TestDraftOrPropose_HumanModeIgnoredWithoutOpt(t *testing.T) {
	docs := &fakeDocSink{doc: KBDoc{Content: "x\n<!-- kb-sig:0 -->\n", HumanLocked: false, Exists: true, MaintainerMode: "human"}}
	res := run(&fakeLLM{body: "fresh"}, docs, &fakeProposalSink{}, draftOpts{})

	if res.Status != "written" {
		t.Fatalf("status = %q, want written (zero opts must ignore maintainer_mode)", res.Status)
	}
}

// The operator's prompt_hint is appended to the system prompt so it can steer
// the page's form without a code change.
func TestDraftOrPropose_PromptHintSteersSystem(t *testing.T) {
	docs := &fakeDocSink{doc: KBDoc{Content: "x\n<!-- kb-sig:0 -->\n", HumanLocked: false, Exists: true, PromptHint: "KEEP-IT-A-ONE-LINER"}}
	llm := &fakeLLM{body: "fresh"}
	run(llm, docs, &fakeProposalSink{}, kbOpts)

	if !strings.Contains(llm.lastSystem, "KEEP-IT-A-ONE-LINER") {
		t.Fatalf("system prompt did not include the operator hint; got:\n%s", llm.lastSystem)
	}
}

// Changing the prompt_hint alone (feedstock unchanged) must still trigger a
// redraft: the hint is folded into the dirty-check signature, so a page that
// carries the plain-feedstock sig is no longer "unchanged" once a hint is set.
func TestDraftOrPropose_PromptHintChangeTriggersRedraft(t *testing.T) {
	// Content carries the OLD signature (feedstock only), but a hint now exists.
	content := "body\n<!-- kb-sig:" + kbSig(testFeed) + " -->\n"
	docs := &fakeDocSink{doc: KBDoc{Content: content, HumanLocked: false, Exists: true, PromptHint: "NEW-HINT"}}
	llm := &fakeLLM{body: "fresh"}
	res := run(llm, docs, &fakeProposalSink{}, kbOpts)

	if res.Status != "written" {
		t.Fatalf("status = %q, want written (a hint change must trigger a redraft)", res.Status)
	}
	if llm.calls != 1 || !strings.Contains(llm.lastSystem, "NEW-HINT") {
		t.Fatalf("hint not applied on redraft (llm calls=%d, system=%q)", llm.calls, llm.lastSystem)
	}
}

// Once a hinted page has been redrafted, its signature includes the hint, so an
// unchanged feedstock + unchanged hint is a no-op (converges, no re-draft loop).
func TestDraftOrPropose_HintedPageConverges(t *testing.T) {
	const hint = "KEEP-IT-A-ONE-LINER"
	settledSig := kbSig(testFeed + "\x00prompt_hint:" + hint)
	content := "body\n<!-- kb-sig:" + settledSig + " -->\n"
	docs := &fakeDocSink{doc: KBDoc{Content: content, HumanLocked: false, Exists: true, PromptHint: hint}}
	llm := &fakeLLM{body: "should not be used"}
	res := run(llm, docs, &fakeProposalSink{}, kbOpts)

	if res.Status != "skipped-unchanged" {
		t.Fatalf("status = %q, want skipped-unchanged (hinted page must converge)", res.Status)
	}
	if llm.calls != 0 {
		t.Fatalf("settled hinted page must not call the LLM (calls=%d)", llm.calls)
	}
}

// preserveCurrent feeds the current page into the LLM so it edits the operator's
// structure rather than regenerating from scratch.
func TestDraftOrPropose_PreservesCurrentContent(t *testing.T) {
	docs := &fakeDocSink{doc: KBDoc{Content: "## OPERATORS-OWN-HEADING\n- keep me\n<!-- kb-sig:0 -->\n", HumanLocked: false, Exists: true}}
	llm := &fakeLLM{body: "fresh"}
	run(llm, docs, &fakeProposalSink{}, kbOpts)

	if !strings.Contains(llm.lastUser, "OPERATORS-OWN-HEADING") {
		t.Fatalf("current content not fed to the LLM for preservation; user turn:\n%s", llm.lastUser)
	}
	// The kb-sig marker must be stripped from the fed content, not echoed.
	if strings.Contains(llm.lastUser, "kb-sig:") {
		t.Fatalf("kb-sig marker leaked into the LLM user turn:\n%s", llm.lastUser)
	}
}

// With no opts (Overview), the user turn is exactly the feedstock — current
// content is NOT injected, preserving historical behaviour.
func TestDraftOrPropose_OverviewUserTurnUnchanged(t *testing.T) {
	docs := &fakeDocSink{doc: KBDoc{Content: "## SHOULD-NOT-APPEAR\n<!-- kb-sig:0 -->\n", HumanLocked: false, Exists: true}}
	llm := &fakeLLM{body: "fresh"}
	run(llm, docs, &fakeProposalSink{}, draftOpts{})

	if llm.lastUser != testFeed {
		t.Fatalf("zero-opts user turn must equal feedstock verbatim; got:\n%s", llm.lastUser)
	}
}

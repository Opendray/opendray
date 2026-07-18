package knowledge

import (
	"context"
	"io"
	"log/slog"
	"testing"
)

// --- fakes ---------------------------------------------------------------

type fakeLLM struct{ body string }

func (f fakeLLM) Complete(_ context.Context, _, _ string) (string, error) {
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

// --- tests ---------------------------------------------------------------

// A locked page whose feedstock has diverged files a proposal (does not
// overwrite) when the draft has not been rejected before.
func TestDraftOrPropose_LockedProposes(t *testing.T) {
	docs := &fakeDocSink{doc: KBDoc{Content: "old\n<!-- kb-sig:0000000000000000 -->\n", HumanLocked: true, Exists: true}}
	props := &fakeProposalSink{}

	res := draftOrPropose(context.Background(), fakeLLM{body: "fresh body"}, docs, props, discardLog(), testCwd, testKind, "sys", testFeed)

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

	res := draftOrPropose(context.Background(), fakeLLM{body: "fresh body"}, docs, props, discardLog(), testCwd, testKind, "sys", testFeed)

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

	res := draftOrPropose(context.Background(), fakeLLM{body: "fresh body"}, docs, props, discardLog(), testCwd, testKind, "sys", testFeed)

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

	res := draftOrPropose(context.Background(), fakeLLM{body: "should not be used"}, docs, props, discardLog(), testCwd, testKind, "sys", testFeed)

	if res.Status != "skipped-unchanged" {
		t.Fatalf("status = %q, want skipped-unchanged", res.Status)
	}
	if props.proposeCalls != 0 || docs.putCalls != 0 {
		t.Fatalf("unchanged page must be untouched (propose=%d put=%d)", props.proposeCalls, docs.putCalls)
	}
}

// An unlocked (agent-owned) page whose feedstock diverged is rewritten in place.
func TestDraftOrPropose_UnlockedWrites(t *testing.T) {
	docs := &fakeDocSink{doc: KBDoc{Content: "old\n<!-- kb-sig:0000000000000000 -->\n", HumanLocked: false, Exists: true}}
	props := &fakeProposalSink{}

	res := draftOrPropose(context.Background(), fakeLLM{body: "fresh body"}, docs, props, discardLog(), testCwd, testKind, "sys", testFeed)

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

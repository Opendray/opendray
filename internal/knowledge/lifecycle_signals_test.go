package knowledge

import (
	"context"
	"strings"
	"testing"
	"time"
)

// --- NormalizeLine -------------------------------------------------------

func TestNormalizeLine(t *testing.T) {
	// All of these are THE SAME LINE for deletion-as-signal purposes.
	same := []string{
		"- Foo is deprecated",
		"* **Foo** is deprecated",
		"  + foo   is Deprecated  ",
		"3. `foo` is deprecated",
		"Foo is deprecated",
	}
	want := NormalizeLine(same[0])
	for _, s := range same[1:] {
		if got := NormalizeLine(s); got != want {
			t.Errorf("NormalizeLine(%q) = %q, want %q", s, got, want)
		}
	}
	// And these are different lines.
	if NormalizeLine("foo is deprecated") == NormalizeLine("foo is not deprecated") {
		t.Error("distinct content must normalize differently")
	}
	if NormalizeLine("") != "" || NormalizeLine("   ") != "" {
		t.Error("blank lines normalize to empty")
	}
}

// --- banned-line scrub ---------------------------------------------------

func TestBannedScrub(t *testing.T) {
	banned := []string{NormalizeLine("- foo is deprecated")}
	body := "# Infra\n- opendray is the gateway\n* **Foo** is deprecated\n- keep me\n"
	got, n := bannedScrub(body, banned)
	if n != 1 {
		t.Fatalf("dropped %d, want 1", n)
	}
	if strings.Contains(strings.ToLower(got), "foo is deprecated") {
		t.Errorf("banned line survived reformatting: %q", got)
	}
	for _, keep := range []string{"# Infra", "opendray is the gateway", "keep me"} {
		if !strings.Contains(got, keep) {
			t.Errorf("scrub ate neighbouring content %q", keep)
		}
	}
	// Empty ban list is a strict no-op.
	if out, n := bannedScrub(body, nil); out != body || n != 0 {
		t.Error("empty ban list must not touch the body")
	}
}

// --- meta filtering + rule tagging --------------------------------------

func TestDropMeta(t *testing.T) {
	rows := []MemoryRow{
		{ID: "1", Text: "the db is at host X", Polarity: PolarityFact},
		{ID: "2", Text: "docs must not mention foo", Polarity: PolarityMeta},
		{ID: "3", Text: "never use npm", Polarity: PolarityRule},
		{ID: "4", Text: "unclassified thing"}, // empty polarity = fact
	}
	got := dropMeta(rows)
	if len(got) != 3 {
		t.Fatalf("kept %d rows, want 3", len(got))
	}
	for _, r := range got {
		if r.Polarity == PolarityMeta {
			t.Errorf("meta row %s survived", r.ID)
		}
	}
}

func TestWriteFactTitlesTagsRules(t *testing.T) {
	var b strings.Builder
	writeFactTitles(&b, []MemoryRow{
		{Text: "the db is at host X", Polarity: PolarityFact},
		{Text: "never use npm", Polarity: PolarityRule},
		{Text: "plain unclassified"},
	})
	out := b.String()
	if !strings.Contains(out, "- [rule] never use npm") {
		t.Errorf("rule row not tagged:\n%s", out)
	}
	if strings.Contains(out, "[rule] the db") || strings.Contains(out, "[rule] plain") {
		t.Errorf("non-rule rows tagged:\n%s", out)
	}
}

// --- removed-lines prompt + draft integration ----------------------------

func TestDraftHonoursRemovedAndBannedLines(t *testing.T) {
	llm := &fakeLLM{body: "# Infra\n- opendray is the gateway\n- foo is deprecated\n"}
	docs := &fakeDocSink{doc: KBDoc{
		Exists:       true,
		Content:      "# Infra\n- opendray is the gateway\n",
		RemovedLines: []string{"- foo is deprecated"},
		BannedLines:  []string{NormalizeLine("- foo is deprecated")},
	}}
	res := draftOrPropose(context.Background(), llm, docs, nil, discardLog(),
		GlobalKBCwd, KBKindInfrastructure, "sys", "FACTS:\n- some new fact\n",
		draftOpts{honorMaintainerMode: true, preserveCurrent: true, applyPromptHint: true})
	if res.Status != "written" {
		t.Fatalf("status = %q (%s)", res.Status, res.Err)
	}
	if !strings.Contains(llm.lastSystem, "OPERATOR-REMOVED LINES") ||
		!strings.Contains(llm.lastSystem, "- foo is deprecated") {
		t.Errorf("removed-lines instruction missing from system prompt")
	}
	// The model reintroduced the line anyway — the ban scrub must catch it.
	if strings.Contains(docs.putContent, "foo is deprecated") {
		t.Errorf("banned line survived into the page:\n%s", docs.putContent)
	}
	if !strings.Contains(docs.putContent, "opendray is the gateway") {
		t.Errorf("scrub ate legitimate content:\n%s", docs.putContent)
	}
}

// An operator deletion must QUIET the page, not dirty it: removal signals
// stay out of the signature, so recording one does not trigger a redraft.
func TestRemovalSignalsDoNotDirtyThePage(t *testing.T) {
	const feedstock = "FACTS:\n- some fact\n"
	docs := &fakeDocSink{doc: KBDoc{Exists: true}}
	llm := &fakeLLM{body: "# Infra\n- some fact\n"}
	opts := draftOpts{honorMaintainerMode: true, preserveCurrent: true, applyPromptHint: true}

	draftOrPropose(context.Background(), llm, docs, nil, discardLog(),
		GlobalKBCwd, KBKindInfrastructure, "sys", feedstock, opts)
	docs.doc.Content = docs.putContent

	docs.doc.RemovedLines = []string{"- something the operator deleted"}
	docs.doc.BannedLines = []string{NormalizeLine("- something the operator deleted")}
	if res := draftOrPropose(context.Background(), llm, docs, nil, discardLog(),
		GlobalKBCwd, KBKindInfrastructure, "sys", feedstock, opts); res.Status != "skipped-unchanged" {
		t.Fatalf("a recorded deletion must not force a redraft, got %q", res.Status)
	}
}

// --- per-page topical feedstock ------------------------------------------

type fakeTopical struct {
	hits []MemoryRow
	err  error
	last string
}

func (f *fakeTopical) SearchMemories(_ context.Context, q string, _ int) ([]MemoryRow, error) {
	f.last = q
	return f.hits, f.err
}

func TestPageFactsTopicalAndFallback(t *testing.T) {
	shared := []MemoryRow{
		{ID: "s1", Text: "recent shared", CreatedAt: time.Unix(300, 0)},
		{ID: "s2", Text: "older shared", CreatedAt: time.Unix(100, 0)},
	}
	docs := &fakeDocSink{doc: KBDoc{Exists: true, Title: "Infrastructure", Description: "hosts and rules"}}
	d := &KBDrafter{docs: docs, log: discardLog()}

	// No topical source → shared list unchanged (the historical path).
	if got := d.pageFacts(context.Background(), GlobalKBCwd, KBKindInfrastructure, shared); len(got) != 2 {
		t.Fatalf("without topical: got %d rows", len(got))
	}

	// Topical error → fallback to shared.
	d.topical = &fakeTopical{err: context.DeadlineExceeded}
	if got := d.pageFacts(context.Background(), GlobalKBCwd, KBKindInfrastructure, shared); len(got) != 2 {
		t.Fatalf("topical error must fall back, got %d rows", len(got))
	}

	// Topical hits → union with recent, deduped, meta dropped,
	// deterministically ordered (recency desc, then id).
	ft := &fakeTopical{hits: []MemoryRow{
		{ID: "t1", Text: "topical hit", CreatedAt: time.Unix(200, 0)},
		{ID: "s1", Text: "recent shared", CreatedAt: time.Unix(300, 0)}, // dup with shared
		{ID: "t2", Text: "meta directive", CreatedAt: time.Unix(250, 0), Polarity: PolarityMeta},
	}}
	d.topical = ft
	got := d.pageFacts(context.Background(), GlobalKBCwd, KBKindInfrastructure, shared)
	if ft.last == "" || !strings.Contains(ft.last, "Infrastructure") {
		t.Errorf("topic query should carry the section title, got %q", ft.last)
	}
	var ids []string
	for _, r := range got {
		ids = append(ids, r.ID)
	}
	want := []string{"s1", "t1", "s2"} // 300, 200, 100 — meta dropped, dup collapsed
	if strings.Join(ids, ",") != strings.Join(want, ",") {
		t.Errorf("ids = %v, want %v", ids, want)
	}
}

// --- classifier ----------------------------------------------------------

type fakePolaritySource struct {
	rows []MemoryRow
	set  map[string]string
}

func (f *fakePolaritySource) ListUnclassified(context.Context, int) ([]MemoryRow, error) {
	return f.rows, nil
}
func (f *fakePolaritySource) SetPolarity(_ context.Context, id, p string) error {
	if f.set == nil {
		f.set = map[string]string{}
	}
	f.set[id] = p
	return nil
}

func TestClassifierRunOnce(t *testing.T) {
	src := &fakePolaritySource{rows: []MemoryRow{
		{ID: "m1", Text: "the db is at host X"},
		{ID: "m2", Text: "docs must not mention foo"},
	}}
	llm := &fakeLLM{body: `[
		{"id": "m1", "polarity": "fact"},
		{"id": "m2", "polarity": "meta"},
		{"id": "hallucinated", "polarity": "fact"},
		{"id": "m1", "polarity": "bogus-value"}
	]`}
	c := NewClassifier(src, llm, discardLog())
	n, err := c.RunOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("classified %d, want 2 (hallucinated id + bogus value skipped)", n)
	}
	if src.set["m1"] != "fact" || src.set["m2"] != "meta" {
		t.Errorf("verdicts wrong: %v", src.set)
	}
	if _, ok := src.set["hallucinated"]; ok {
		t.Error("hallucinated id must not be written")
	}
}

func TestClassifierBadJSONIsContained(t *testing.T) {
	src := &fakePolaritySource{rows: []MemoryRow{{ID: "m1", Text: "x"}}}
	c := NewClassifier(src, &fakeLLM{body: "sorry, I cannot"}, discardLog())
	n, err := c.RunOnce(context.Background())
	if err != nil || n != 0 {
		t.Fatalf("bad JSON must skip the batch quietly: n=%d err=%v", n, err)
	}
	if len(src.set) != 0 {
		t.Error("nothing may be written on a parse failure")
	}
}

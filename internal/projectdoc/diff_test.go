package projectdoc

import (
	"strings"
	"testing"
)

// renderDiff flattens a DocDiff into a readable form for assertions.
func renderDiff(d DocDiff) string {
	var b strings.Builder
	for _, h := range d.Hunks {
		if h.SkippedBefore > 0 {
			b.WriteString("⋯\n")
		}
		for _, ln := range h.Lines {
			switch ln.Kind {
			case DiffAdd:
				b.WriteString("+" + ln.Text + "\n")
			case DiffRemove:
				b.WriteString("-" + ln.Text + "\n")
			default:
				b.WriteString(" " + ln.Text + "\n")
			}
		}
	}
	return b.String()
}

// A pending proposal's PriorContent is empty — that column is only
// written on APPROVAL, to record what the approval replaced. Diffing
// against it would make every pending proposal read as "all lines
// added", which is the full-document view this feature exists to
// replace. The baseline must be the live document.
func TestDiffBaseline_UsesLiveDocNotPriorContent(t *testing.T) {
	live := "alpha\nbravo\ncharlie\n"
	p := Proposal{
		ProposedContent: "alpha\nbravo\nNEW\ncharlie\n",
		PriorContent:    "", // what a pending proposal actually carries
	}

	d := DiffBaseline(live, p)
	if d.Added != 1 || d.Removed != 0 {
		t.Errorf("Added/Removed = %d/%d, want 1/0 — the diff is not against the live doc", d.Added, d.Removed)
	}
	if got := renderDiff(d); strings.Count(got, "+") != 1 {
		t.Errorf("expected a single added line, got:\n%s", got)
	}
}

// Even when PriorContent happens to be populated (an approved row), the
// live document still wins — it is what the operator is deciding about.
func TestDiffBaseline_IgnoresStalePriorContent(t *testing.T) {
	p := Proposal{
		ProposedContent: "one\ntwo\n",
		PriorContent:    "something\nentirely\ndifferent\n",
	}
	if d := DiffBaseline("one\ntwo\n", p); !d.Unchanged {
		t.Errorf("proposal matching the live doc should show no changes; got +%d/-%d", d.Added, d.Removed)
	}
}

func TestDiffLines_AddedLine(t *testing.T) {
	before := "alpha\nbravo\ncharlie\n"
	after := "alpha\nbravo\nnew line\ncharlie\n"

	d := DiffLines(before, after, 3)
	if d.Unchanged {
		t.Fatal("Unchanged = true for a real change")
	}
	if d.Added != 1 || d.Removed != 0 {
		t.Errorf("Added/Removed = %d/%d, want 1/0", d.Added, d.Removed)
	}
	got := renderDiff(d)
	if !strings.Contains(got, "+new line") {
		t.Errorf("added line not marked:\n%s", got)
	}
	// Surrounding lines come through as context, not as changes.
	if !strings.Contains(got, " alpha") || !strings.Contains(got, " charlie") {
		t.Errorf("context lines missing:\n%s", got)
	}
}

func TestDiffLines_ReplacedLine(t *testing.T) {
	d := DiffLines("keep\nold value\ntail\n", "keep\nnew value\ntail\n", 3)
	got := renderDiff(d)
	if !strings.Contains(got, "-old value") || !strings.Contains(got, "+new value") {
		t.Errorf("replacement should show both sides:\n%s", got)
	}
	if d.Added != 1 || d.Removed != 1 {
		t.Errorf("Added/Removed = %d/%d, want 1/1", d.Added, d.Removed)
	}
}

// The whole point of the feature: a long document with one edit must not
// come back as the whole document.
func TestDiffLines_LongDocumentCollapsesUnchangedRegions(t *testing.T) {
	var before, after []string
	for i := range 200 {
		before = append(before, "line "+string(rune('a'+i%26))+string(rune('0'+i%10)))
	}
	after = append([]string{}, before...)
	after[100] = "CHANGED"

	d := DiffLines(strings.Join(before, "\n"), strings.Join(after, "\n"), 3)

	total := 0
	for _, h := range d.Hunks {
		total += len(h.Lines)
	}
	// One change + 3 lines of context either side ≈ 8 lines, nowhere near 200.
	if total > 20 {
		t.Errorf("emitted %d lines for a one-line edit in a 200-line doc — unchanged regions are not collapsed", total)
	}
	if len(d.Hunks) != 1 {
		t.Fatalf("hunks = %d, want 1", len(d.Hunks))
	}
	if d.Hunks[0].SkippedBefore == 0 {
		t.Error("first hunk should report the lines skipped before it so the UI can say how many collapsed")
	}
}

// Two distant edits stay two separate hunks rather than merging into one
// giant block that drags the untouched middle along.
func TestDiffLines_SeparateEditsStaySeparateHunks(t *testing.T) {
	var lines []string
	for i := range 100 {
		lines = append(lines, "row "+string(rune('0'+i%10)))
	}
	before := strings.Join(lines, "\n")
	lines[5] = "FIRST EDIT"
	lines[80] = "SECOND EDIT"
	after := strings.Join(lines, "\n")

	d := DiffLines(before, after, 3)
	if len(d.Hunks) != 2 {
		t.Fatalf("hunks = %d, want 2 (one per edit)", len(d.Hunks))
	}
	if d.Hunks[1].SkippedBefore == 0 {
		t.Error("the gap between two hunks must be reported as skipped")
	}
}

func TestDiffLines_IdenticalContent(t *testing.T) {
	d := DiffLines("same\ntext\n", "same\ntext\n", 3)
	if !d.Unchanged {
		t.Error("Unchanged = false for identical content")
	}
	if len(d.Hunks) != 0 {
		t.Errorf("hunks = %d, want 0 for identical content", len(d.Hunks))
	}
	if d.Added != 0 || d.Removed != 0 {
		t.Errorf("Added/Removed = %d/%d, want 0/0", d.Added, d.Removed)
	}
}

// A brand-new page (no prior content) is all additions, not a broken diff.
func TestDiffLines_EmptyBefore(t *testing.T) {
	d := DiffLines("", "first\nsecond\n", 3)
	if d.Unchanged {
		t.Error("Unchanged = true when content was added to an empty doc")
	}
	if d.Added != 2 {
		t.Errorf("Added = %d, want 2", d.Added)
	}
	if d.Removed != 0 {
		t.Errorf("Removed = %d, want 0", d.Removed)
	}
}

// Trailing-newline differences are a formatting detail, not a content
// change the operator needs to review.
func TestDiffLines_TrailingNewlineIgnored(t *testing.T) {
	if d := DiffLines("a\nb", "a\nb\n", 3); !d.Unchanged {
		t.Errorf("a missing trailing newline should not read as a change: %+v", d.Hunks)
	}
}

// CRLF input must not make every line look changed.
func TestDiffLines_CRLFNormalised(t *testing.T) {
	if d := DiffLines("a\r\nb\r\n", "a\nb\n", 3); !d.Unchanged {
		t.Errorf("line-ending style alone should not read as a change: %+v", d.Hunks)
	}
}

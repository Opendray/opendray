package projectdoc

import (
	"strings"
	"testing"
)

func TestRenderJournalSection_NoBudget(t *testing.T) {
	logs := []LogEntry{
		{Title: "Newest", Content: "C"},
		{Title: "Older", Content: "B"},
		{Title: "Oldest", Content: "A"},
	}
	out, truncated := renderJournalSection(logs, 0)
	if truncated {
		t.Error("expected no truncation with budget=0")
	}
	if !strings.Contains(out, "**Oldest**") || !strings.Contains(out, "**Newest**") {
		t.Errorf("missing entries:\n%s", out)
	}
	// Oldest first: index of Oldest should precede Newest.
	if strings.Index(out, "Oldest") > strings.Index(out, "Newest") {
		t.Errorf("chronology wrong:\n%s", out)
	}
}

func TestRenderJournalSection_BudgetTruncates(t *testing.T) {
	logs := []LogEntry{
		{Title: "A", Content: strings.Repeat("xxxxxx", 50)},
		{Title: "B", Content: strings.Repeat("yyyyyy", 50)},
		{Title: "C", Content: strings.Repeat("zzzzzz", 50)},
	}
	out, truncated := renderJournalSection(logs, 200)
	if !truncated {
		t.Error("expected truncation at 200 bytes")
	}
	if len(out) > 250 {
		// the header + at most one entry should fit; allow a little slack
		t.Errorf("output unexpectedly large: %d bytes\n%s", len(out), out)
	}
}

func TestRenderJournalSection_EmptyIfNothingFits(t *testing.T) {
	logs := []LogEntry{
		{Title: "huge", Content: strings.Repeat("a", 5000)},
	}
	// 20 bytes can't even fit one line — section should collapse to empty.
	out, truncated := renderJournalSection(logs, 20)
	if !truncated {
		t.Error("expected truncation")
	}
	if out != "" {
		t.Errorf("expected empty when nothing fits; got:\n%s", out)
	}
}

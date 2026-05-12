package projectdoc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// renderJournalFile + writeIfDifferent only — no DB needed.
// The Service.Mirror path is exercised end-to-end via the smoke
// test (gated by `smoke` build tag).

func TestRenderJournalFile_Empty(t *testing.T) {
	out := renderJournalFile(nil)
	if !strings.Contains(out, "No entries yet") {
		t.Errorf("empty journal missing placeholder: %q", out)
	}
}

func TestRenderJournalFile_WithEntries(t *testing.T) {
	now := time.Date(2026, 5, 12, 7, 30, 0, 0, time.UTC)
	logs := []LogEntry{
		{
			Title:     "M5 landed",
			Kind:      LogKindManual,
			UpdatedBy: AuthorOperator,
			CreatedAt: now,
			Content:   "Schema and handlers shipped.",
		},
		{
			Title:     "",
			Kind:      LogKindSessionSummary,
			UpdatedBy: AuthorSummarizer,
			CreatedAt: now.Add(-time.Hour),
			Content:   "  ",
		},
	}
	out := renderJournalFile(logs)
	for _, want := range []string{
		"# Project journal",
		"## M5 landed",
		"manual · operator ·",
		"Schema and handlers shipped.",
		"## (untitled)",
		"session_summary · summarizer ·",
		"_(empty)_",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestWriteIfDifferent_NoOpWhenIdentical(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(p, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	st1, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	if err := writeIfDifferent(p, "hello"); err != nil {
		t.Fatal(err)
	}
	st2, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if !st1.ModTime().Equal(st2.ModTime()) {
		t.Errorf("mtime changed on no-op write: %v → %v", st1.ModTime(), st2.ModTime())
	}
}

func TestWriteIfDifferent_RewritesWhenChanged(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.txt")
	if err := writeIfDifferent(p, "alpha"); err != nil {
		t.Fatal(err)
	}
	if err := writeIfDifferent(p, "beta"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "beta" {
		t.Errorf("got %q want %q", got, "beta")
	}
}

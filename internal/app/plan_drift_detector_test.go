package app

import (
	"context"
	"strings"
	"testing"

	"github.com/opendray/opendray-v2/internal/projectdoc"
)

func TestPlanDriftDetector_NoRegistryIsNoop(t *testing.T) {
	d := newPlanDriftDetector(nil)
	out, err := d.DetectDrift(context.Background(), projectdoc.DriftInput{
		Cwd:               "/p",
		CurrentPlan:       "## Plan\n- x",
		TranscriptSummary: "did the work",
	})
	if err != nil {
		t.Fatalf("nil registry should not error: %v", err)
	}
	if out.ShouldPropose {
		t.Errorf("nil registry should not propose")
	}
}

func TestPlanDriftDetector_EmptyPlanShortCircuits(t *testing.T) {
	// Build a detector that would normally call the registry; the
	// short-circuit must hit before any registry call, so a nil
	// registry being safe here means the guard ran.
	d := newPlanDriftDetector(nil)
	out, err := d.DetectDrift(context.Background(), projectdoc.DriftInput{
		Cwd:               "/p",
		CurrentPlan:       "   \n  ",
		TranscriptSummary: "did the work",
	})
	if err != nil {
		t.Fatalf("empty plan should not error: %v", err)
	}
	if out.ShouldPropose {
		t.Errorf("empty plan should not propose")
	}
}

func TestPlanDriftDetector_EmptySummaryShortCircuits(t *testing.T) {
	d := newPlanDriftDetector(nil)
	out, err := d.DetectDrift(context.Background(), projectdoc.DriftInput{
		Cwd:               "/p",
		CurrentPlan:       "## Plan\n- x",
		TranscriptSummary: "",
	})
	if err != nil {
		t.Fatalf("empty summary should not error: %v", err)
	}
	if out.ShouldPropose {
		t.Errorf("empty summary should not propose")
	}
}

func TestBuildDriftUserInput_Shape(t *testing.T) {
	in := projectdoc.DriftInput{
		Cwd:               "/projects/foo",
		CurrentPlan:       "## Plan\n- M1 deploy",
		TranscriptSummary: "Agent landed M1 deploy script",
		RecentJournal: []projectdoc.LogEntry{
			// newest first (as docs.ListLogs returns)
			{Title: "M1 progress", Content: "Wired up deploy.sh"},
			{Title: "M1 kickoff", Content: "Discussed deploy strategy"},
		},
	}
	got := buildDriftUserInput(in)
	for _, want := range []string{
		"`/projects/foo`",
		"## Current plan",
		"M1 deploy",
		"## Latest session summary",
		"Agent landed M1 deploy script",
		"## Recent journal entries",
		"**M1 kickoff**", // oldest rendered first
		"**M1 progress**",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	// Chronology: kickoff (oldest) must appear before progress (newest).
	if strings.Index(got, "M1 kickoff") > strings.Index(got, "M1 progress") {
		t.Errorf("journal entries rendered in wrong chronological order:\n%s", got)
	}
}

func TestBuildDriftUserInput_NoJournalSection(t *testing.T) {
	got := buildDriftUserInput(projectdoc.DriftInput{
		Cwd:               "/p",
		CurrentPlan:       "## Plan",
		TranscriptSummary: "did work",
	})
	if strings.Contains(got, "Recent journal entries") {
		t.Errorf("journal section should be omitted when empty:\n%s", got)
	}
}

func TestBuildDriftUserInput_LongEntryTruncated(t *testing.T) {
	long := strings.Repeat("a", 1000)
	in := projectdoc.DriftInput{
		Cwd:               "/p",
		CurrentPlan:       "## Plan",
		TranscriptSummary: "x",
		RecentJournal: []projectdoc.LogEntry{
			{Title: "long", Content: long},
		},
	}
	got := buildDriftUserInput(in)
	if !strings.Contains(got, "…") {
		t.Errorf("expected ellipsis for truncated entry; got:\n%s", got)
	}
	if strings.Contains(got, long) {
		t.Errorf("entry should not be rendered in full (1000 chars)")
	}
}

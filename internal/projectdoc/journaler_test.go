package projectdoc

import (
	"strings"
	"testing"
	"time"
)

func TestBuildJournalBody_FullSession(t *testing.T) {
	start := time.Date(2026, 5, 12, 10, 0, 0, 0, time.UTC)
	end := start.Add(7*time.Minute + 30*time.Second)
	exit := 0
	sess := SessionInfo{
		ID:         "sess_abcdef0123456789",
		ProviderID: "claude",
		Cwd:        "/projects/foo",
		StartedAt:  start,
		EndedAt:    &end,
		ExitCode:   &exit,
	}
	inputs := []HistoryEntry{
		{Ts: end.Add(-time.Minute), Text: "Add the bottom-tab nav"},
		{Ts: end.Add(-3 * time.Minute), Text: "Fix\n\nlogin\tflow"},
	}

	title, body := buildJournalBody(sess, "ended", inputs)

	if !strings.Contains(title, "Claude") {
		t.Errorf("title missing provider label: %q", title)
	}
	if !strings.Contains(title, "ended") {
		t.Errorf("title missing state: %q", title)
	}
	if !strings.HasSuffix(title, "ended") && !strings.Contains(title, "ended") {
		t.Errorf("title shape wrong: %q", title)
	}
	for _, want := range []string{
		"sess_abcdef0123456789",
		"`/projects/foo`",
		"duration: 7m30s",
		"exit_code: 0",
		"Recent operator inputs",
		"Add the bottom-tab nav",
		"Fix login flow", // whitespace collapsed
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q\n--- body ---\n%s", want, body)
		}
	}
}

func TestBuildJournalBody_NoHistory(t *testing.T) {
	start := time.Now().UTC()
	sess := SessionInfo{
		ID:         "sess_42",
		ProviderID: "shell",
		Cwd:        "/tmp/x",
		StartedAt:  start,
	}
	_, body := buildJournalBody(sess, "stopped", nil)
	if strings.Contains(body, "Recent operator inputs") {
		t.Errorf("body should omit empty inputs block, got:\n%s", body)
	}
	if !strings.Contains(body, "Session metadata") {
		t.Errorf("body missing metadata block:\n%s", body)
	}
}

func TestNewSessionSummaryEntry_StampsOutcome(t *testing.T) {
	exit := 0
	started := time.Date(2026, 6, 20, 9, 18, 37, 0, time.UTC)
	ended := started.Add(5 * time.Minute)
	sess := SessionInfo{ID: "ses_x", Cwd: "/proj", ExitCode: &exit, StartedAt: started, EndedAt: &ended}

	e := newSessionSummaryEntry(sess, "stopped", "the title", "the body")

	if e.Kind != LogKindSessionSummary || e.UpdatedBy != AuthorSummarizer {
		t.Fatalf("kind/author = %q/%q", e.Kind, e.UpdatedBy)
	}
	if e.OutcomeState != "stopped" {
		t.Errorf("OutcomeState = %q, want stopped", e.OutcomeState)
	}
	if e.ExitCode == nil || *e.ExitCode != 0 {
		t.Errorf("ExitCode = %v, want 0", e.ExitCode)
	}
	if e.StartedAt == nil || !e.StartedAt.Equal(started) {
		t.Errorf("StartedAt = %v, want %v", e.StartedAt, started)
	}
	if e.EndedAt == nil || !e.EndedAt.Equal(ended) {
		t.Errorf("EndedAt = %v, want %v", e.EndedAt, ended)
	}
}

func TestNewSessionSummaryEntry_NilEndAndExit(t *testing.T) {
	sess := SessionInfo{ID: "ses_y", Cwd: "/p", StartedAt: time.Now().UTC()}
	e := newSessionSummaryEntry(sess, "ended", "t", "b")
	if e.ExitCode != nil || e.EndedAt != nil {
		t.Errorf("expected nil exit/end, got %v/%v", e.ExitCode, e.EndedAt)
	}
	if e.StartedAt == nil {
		t.Error("StartedAt should always be set")
	}
}

func TestCompactOneLine(t *testing.T) {
	for _, tc := range []struct {
		in, want string
		max      int
	}{
		{"hello", "hello", 100},
		{"  hello\nworld   ", "hello world", 100},
		{"line1\tline2", "line1 line2", 100},
		{"abcdef", "abc…", 3},
		{"", "", 100},
	} {
		got := compactOneLine(tc.in, tc.max)
		if got != tc.want {
			t.Errorf("compactOneLine(%q,%d) = %q want %q", tc.in, tc.max, got, tc.want)
		}
	}
}

func TestSessionSucceeded(t *testing.T) {
	zero, fail := 0, 1
	cases := []struct {
		name  string
		exit  *int
		state string
		want  bool
	}{
		{"clean exit", &zero, "ended", true},
		{"no exit code recorded", nil, "ended", true},
		{"non-zero exit", &fail, "ended", false},
		{"operator stop is not a failure", &fail, "stopped", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := SessionSucceeded(SessionInfo{ExitCode: tc.exit}, tc.state); got != tc.want {
				t.Errorf("SessionSucceeded(exit=%v, state=%s) = %v, want %v", tc.exit, tc.state, got, tc.want)
			}
		})
	}
}

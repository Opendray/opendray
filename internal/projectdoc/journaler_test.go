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

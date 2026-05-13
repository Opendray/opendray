package cleaner

import (
	"strings"
	"testing"
	"time"
)

func TestRenderBatch_Shape(t *testing.T) {
	items := []BatchItem{
		{
			ID:        "mem_abc",
			Text:      "User prefers pnpm",
			CreatedAt: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
			HitCount:  5,
		},
		{
			ID:        "mem_def",
			Text:      "uses pnpm not npm",
			CreatedAt: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
			HitCount:  0,
		},
	}
	out := renderBatch(items)
	for _, want := range []string{
		"[1] mem_abc | created 2026-01-15 | hit_count=5",
		"User prefers pnpm",
		"[2] mem_def | created 2026-02-01 | hit_count=0",
		"uses pnpm not npm",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered batch missing %q\n--- batch ---\n%s", want, out)
		}
	}
}

func TestRenderBatch_CollapsesMultilineText(t *testing.T) {
	items := []BatchItem{
		{
			ID:        "mem_x",
			Text:      "line one\nline two\nline three",
			CreatedAt: time.Now().UTC(),
		},
	}
	out := renderBatch(items)
	if strings.Contains(out, "\n    line two") {
		t.Errorf("multi-line memory text should be collapsed, got:\n%s", out)
	}
	if !strings.Contains(out, "line one line two line three") {
		t.Errorf("text should be flattened onto one line, got:\n%s", out)
	}
}

// (TestResponseFormatFor_* removed in M25 — response_format
// enforcement moved from cleaner.responseFormatFor into the
// worker package's SummarizerWorker.Run, which always emits
// json_schema when Request.ResponseFormatJSONSchema is set.
// cleaner.Client just passes DecisionsJSONSchema through.)

func TestValidVerdict(t *testing.T) {
	for _, tc := range []struct {
		v    Verdict
		want bool
	}{
		{VerdictKeep, true},
		{VerdictStale, true},
		{VerdictDuplicate, true},
		{Verdict("delete"), false},
		{Verdict(""), false},
	} {
		if got := ValidVerdict(tc.v); got != tc.want {
			t.Errorf("ValidVerdict(%q) = %v want %v", tc.v, got, tc.want)
		}
	}
}

func TestConfigApplyDefaults(t *testing.T) {
	c := Config{}.applyDefaults()
	if c.BatchSize != 30 {
		t.Errorf("default BatchSize = %d, want 30", c.BatchSize)
	}
	if c.MinAge != 24*time.Hour {
		t.Errorf("default MinAge = %s, want 24h", c.MinAge)
	}
	if c.SkipIfDecidedWithin != 7*24*time.Hour {
		t.Errorf("default SkipIfDecidedWithin = %s, want 168h", c.SkipIfDecidedWithin)
	}
	if c.CallTimeout != 60*time.Second {
		t.Errorf("default CallTimeout = %s, want 60s", c.CallTimeout)
	}
}

func TestTruncate(t *testing.T) {
	for _, tc := range []struct {
		in   string
		max  int
		want string
	}{
		{"short", 100, "short"},
		{"abcdefghij", 5, "abcde…"},
		{"", 10, ""},
	} {
		if got := truncate(tc.in, tc.max); got != tc.want {
			t.Errorf("truncate(%q, %d) = %q want %q", tc.in, tc.max, got, tc.want)
		}
	}
}

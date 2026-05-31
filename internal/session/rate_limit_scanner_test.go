package session

import (
	"testing"
	"time"
)

func TestScanForRateLimitBanner_StandardForm(t *testing.T) {
	// The exact form observed in the operator's screenshot.
	buf := []byte("You've hit your session limit · resets 10:20am (UTC)\n")
	now := mustTime(t, "2006-01-02T08:00:00Z")
	reset, ok := ScanForRateLimitBanner(buf, now)
	if !ok {
		t.Fatal("expected banner to be detected")
	}
	want := mustTime(t, "2006-01-02T10:20:00Z")
	if !reset.Equal(want) {
		t.Errorf("reset = %v, want %v", reset, want)
	}
}

func TestScanForRateLimitBanner_24HourForm(t *testing.T) {
	buf := []byte("You've hit your session limit · resets 22:30 (UTC)\n")
	now := mustTime(t, "2006-01-02T08:00:00Z")
	reset, ok := ScanForRateLimitBanner(buf, now)
	if !ok {
		t.Fatal("expected banner to be detected")
	}
	want := mustTime(t, "2006-01-02T22:30:00Z")
	if !reset.Equal(want) {
		t.Errorf("reset = %v, want %v", reset, want)
	}
}

func TestScanForRateLimitBanner_RollsToTomorrowWhenPast(t *testing.T) {
	// Reset time HH:MM is already past for today → reset is tomorrow.
	buf := []byte("You've hit your session limit · resets 09:00 (UTC)\n")
	now := mustTime(t, "2006-01-02T15:00:00Z")
	reset, ok := ScanForRateLimitBanner(buf, now)
	if !ok {
		t.Fatal("expected banner to be detected")
	}
	want := mustTime(t, "2006-01-03T09:00:00Z")
	if !reset.Equal(want) {
		t.Errorf("reset = %v, want %v", reset, want)
	}
}

func TestScanForRateLimitBanner_HyphenSeparatorFallback(t *testing.T) {
	// Future-proofing: if the CLI swaps the · for a -, we still match.
	buf := []byte("You've hit your session limit - resets 11:00am (UTC)\n")
	now := mustTime(t, "2006-01-02T08:00:00Z")
	_, ok := ScanForRateLimitBanner(buf, now)
	if !ok {
		t.Error("expected hyphen-separator banner to match")
	}
}

func TestScanForRateLimitBanner_NoMatchOnUnrelatedOutput(t *testing.T) {
	tests := []string{
		"Some output that mentions session and limit but not the banner",
		"You hit your session limit yesterday but this is a recap",
		"You've hit your session limit · resets soon",          // no time
		"You've hit your session limit · resets 10:20am",       // no (UTC)
		"You've hit your session limit · resets 10:20am (EST)", // wrong tz
	}
	now := mustTime(t, "2006-01-02T08:00:00Z")
	for _, s := range tests {
		t.Run(s[:min(40, len(s))], func(t *testing.T) {
			if _, ok := ScanForRateLimitBanner([]byte(s), now); ok {
				t.Errorf("expected NO match on: %q", s)
			}
		})
	}
}

func TestScanForRateLimitBanner_BannerEmbeddedInLargerOutput(t *testing.T) {
	// Realistic: the banner appears in the middle of a stream of TUI
	// redraws and ANSI escape codes.
	buf := []byte(
		"\x1b[2J\x1b[H> some prompt\n" +
			"Response chunk here...\n" +
			"\x1b[31mYou've hit your session limit · resets 14:00 (UTC)\x1b[0m\n" +
			"more output trailing...\n",
	)
	now := mustTime(t, "2006-01-02T08:00:00Z")
	reset, ok := ScanForRateLimitBanner(buf, now)
	if !ok {
		t.Fatal("expected banner to be detected even when surrounded by ANSI")
	}
	want := mustTime(t, "2006-01-02T14:00:00Z")
	if !reset.Equal(want) {
		t.Errorf("reset = %v, want %v", reset, want)
	}
}

func mustTime(t *testing.T, s string) time.Time {
	t.Helper()
	tm, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}
	return tm
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

package opencodeacct

import (
	"testing"
	"time"
)

func TestPickSession(t *testing.T) {
	dir := t.TempDir()
	other := t.TempDir()
	start := time.UnixMilli(2_000_000)
	// Shape of `opencode session list --format json`.
	raw := []byte(`[
	 {"id":"ses_old_here","directory":"` + dir + `","created":1000000,"updated":1500000},
	 {"id":"ses_new_elsewhere","directory":"` + other + `","created":1000000,"updated":9000000},
	 {"id":"ses_since_here","directory":"` + dir + `","created":1000000,"updated":2500000},
	 {"id":"ses_newest_here","directory":"` + dir + `","created":1000000,"updated":3000000}
	]`)

	if got := pickSession(raw, dir, time.Time{}); got != "ses_newest_here" {
		t.Errorf("no since: got %q, want newest in dir", got)
	}
	if got := pickSession(raw, dir, start); got != "ses_newest_here" {
		t.Errorf("since: got %q, want newest active since start", got)
	}
	// Nothing active since → fall back to newest in the directory.
	if got := pickSession(raw, dir, time.UnixMilli(5_000_000)); got != "ses_newest_here" {
		t.Errorf("fallback: got %q", got)
	}
	if got := pickSession(raw, t.TempDir(), time.Time{}); got != "" {
		t.Errorf("unrelated dir: got %q, want empty", got)
	}
	if got := pickSession([]byte("not json"), dir, time.Time{}); got != "" {
		t.Errorf("bad json: got %q, want empty", got)
	}
}

// The "since" preference must beat a newer-but-older-started session only
// when that one was not active in this run; here the run's own session is
// the only one updated after start.
func TestPickSession_PrefersRunsOwnConversation(t *testing.T) {
	dir := t.TempDir()
	raw := []byte(`[
	 {"id":"ses_mine","directory":"` + dir + `","created":2100000,"updated":2200000},
	 {"id":"ses_before","directory":"` + dir + `","created":1000000,"updated":1900000}
	]`)
	if got := pickSession(raw, dir, time.UnixMilli(2_000_000)); got != "ses_mine" {
		t.Errorf("got %q, want ses_mine", got)
	}
}

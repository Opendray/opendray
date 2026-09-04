package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writeGrokCarryoverFixture writes a minimal grok chat transcript at
// <root>/<percent-encoded-cwd>/<sessionID>/chat_history.jsonl, mirroring
// grok's real on-disk layout, and returns a config pointing at <root>.
func writeGrokCarryoverFixture(t *testing.T, cwd, sessionID string, lines []string) GrokHistoryConfig {
	t.Helper()
	root := t.TempDir()
	// grok encodes the cwd into the session dir name by percent-encoding
	// the path separators (e.g. /var/lib -> %2Fvar%2Flib).
	encoded := strings.ReplaceAll(cwd, "/", "%2F")
	dir := filepath.Join(root, encoded, sessionID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(dir, "chat_history.jsonl"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return GrokHistoryConfig{SessionsRoots: []string{root}}
}

func TestBuildGrokCarryover(t *testing.T) {
	cwd := "/var/lib/opendray/vocaler-grok"
	sid := "01a04fbd-435a-78c2-9a39-95a26d7192e5"

	t.Run("user_query + assistant text kept, synthetic + tool noise dropped", func(t *testing.T) {
		cfg := writeGrokCarryoverFixture(t, cwd, sid, []string{
			`{"type":"system","content":"you are grok"}`,
			`{"type":"user","content":[{"type":"text","text":"<user_info>\nOS Version: linux\n</user_info>"}]}`,
			`{"type":"user","content":[{"type":"text","text":"<user_query> add a login form </user_query>"}]}`,
			`{"type":"reasoning","content":null}`,
			`{"type":"assistant","content":"On it, creating the form.","tool_calls":[{"id":"c1","name":"Write"}]}`,
			`{"type":"tool_result","content":"file written ok"}`,
			`{"type":"user","content":[{"type":"text","text":"<system-reminder> background task done </system-reminder>"}]}`,
		})
		got := BuildGrokCarryover(cfg, cwd, sid, 0)
		if got == "" {
			t.Fatal("expected a recap, got empty")
		}
		if !strings.Contains(got, "Carried-over context from a previous account") {
			t.Error("missing header")
		}
		if !strings.Contains(got, "You: add a login form") {
			t.Errorf("missing/untrimmed user query turn:\n%s", got)
		}
		if !strings.Contains(got, "Assistant: On it, creating the form.") {
			t.Errorf("missing assistant text turn:\n%s", got)
		}
		for _, noise := range []string{
			"user_info", "user_query", "system-reminder", "reasoning",
			"file written ok", "you are grok",
		} {
			if strings.Contains(got, noise) {
				t.Errorf("noise %q leaked into recap:\n%s", noise, got)
			}
		}
		if !strings.Contains(got, "End of carried-over context") {
			t.Error("missing footer")
		}
	})

	t.Run("missing transcript returns empty (degrade to fresh)", func(t *testing.T) {
		cfg := writeGrokCarryoverFixture(t, cwd, sid, []string{
			`{"type":"user","content":[{"type":"text","text":"<user_query> hi </user_query>"}]}`,
		})
		got := BuildGrokCarryover(cfg, cwd, "01a00000-0000-0000-0000-000000000000", 0)
		if got != "" {
			t.Errorf("expected empty for missing transcript, got:\n%s", got)
		}
	})

	t.Run("empty inputs are no-ops", func(t *testing.T) {
		cfg := GrokHistoryConfig{SessionsRoots: []string{t.TempDir()}}
		if BuildGrokCarryover(cfg, "", sid, 0) != "" {
			t.Error("empty cwd should return empty")
		}
		if BuildGrokCarryover(cfg, cwd, "", 0) != "" {
			t.Error("empty sessionID should return empty")
		}
	})

	t.Run("only-synthetic transcript returns empty", func(t *testing.T) {
		cfg := writeGrokCarryoverFixture(t, cwd, sid, []string{
			`{"type":"user","content":[{"type":"text","text":"<user_info> OS: linux </user_info>"}]}`,
			`{"type":"reasoning","content":null}`,
			`{"type":"tool_result","content":"ok"}`,
		})
		if got := BuildGrokCarryover(cfg, cwd, sid, 0); got != "" {
			t.Errorf("expected empty when no genuine turns, got:\n%s", got)
		}
	})

	t.Run("tail-truncates to budget with elision marker", func(t *testing.T) {
		big := strings.Repeat("x", 2000)
		cfg := writeGrokCarryoverFixture(t, cwd, sid, []string{
			`{"type":"user","content":[{"type":"text","text":"<user_query> OLDEST ` + big + ` </user_query>"}]}`,
			`{"type":"user","content":[{"type":"text","text":"<user_query> MIDDLE ` + big + ` </user_query>"}]}`,
			`{"type":"user","content":[{"type":"text","text":"<user_query> NEWEST ` + big + ` </user_query>"}]}`,
		})
		got := BuildGrokCarryover(cfg, cwd, sid, 3200)
		if !strings.Contains(got, "NEWEST") {
			t.Errorf("most recent turn should survive truncation:\n%.200s", got)
		}
		if strings.Contains(got, "OLDEST") {
			t.Error("oldest turn should have been dropped by the budget")
		}
		if !strings.Contains(got, "earlier turns omitted") {
			t.Error("expected elision marker when turns are dropped")
		}
	})

	// R2: a crafted sessionID must never let the reader escape the
	// sessions root and pull an unrelated file.
	t.Run("path traversal in sessionID is contained", func(t *testing.T) {
		cfg := writeGrokCarryoverFixture(t, cwd, sid, []string{
			`{"type":"user","content":[{"type":"text","text":"<user_query> hi </user_query>"}]}`,
		})
		// Plant a decoy transcript one level up from the sessions root.
		root := cfg.SessionsRoots[0]
		decoyDir := filepath.Join(filepath.Dir(root), "decoy")
		if err := os.MkdirAll(decoyDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(decoyDir, "chat_history.jsonl"),
			[]byte(`{"type":"user","content":[{"type":"text","text":"<user_query> SECRET </user_query>"}]}`+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		for _, evil := range []string{
			"../decoy",
			"../../decoy",
			"foo/bar",
		} {
			if got := BuildGrokCarryover(cfg, cwd, evil, 0); got != "" {
				t.Errorf("traversal sessionID %q must return empty, got:\n%s", evil, got)
			}
		}
	})
}

func TestLatestGrokSessionID(t *testing.T) {
	cwd := "/var/lib/opendray/vocaler-grok"

	newFixtureWithSessions := func(t *testing.T, sessions map[string]time.Time) GrokHistoryConfig {
		t.Helper()
		root := t.TempDir()
		encoded := strings.ReplaceAll(cwd, "/", "%2F")
		for sid, mt := range sessions {
			dir := filepath.Join(root, encoded, sid)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			ch := filepath.Join(dir, "chat_history.jsonl")
			if err := os.WriteFile(ch, []byte("{}\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.Chtimes(ch, mt, mt); err != nil {
				t.Fatal(err)
			}
		}
		return GrokHistoryConfig{SessionsRoots: []string{root}}
	}

	t.Run("returns the session whose transcript was touched most recently", func(t *testing.T) {
		base := time.Now().Add(-time.Hour)
		cfg := newFixtureWithSessions(t, map[string]time.Time{
			"01a00000-0000-0000-0000-00000000aaaa": base,
			"01a00000-0000-0000-0000-00000000bbbb": base.Add(30 * time.Minute), // newest
			"01a00000-0000-0000-0000-00000000cccc": base.Add(10 * time.Minute),
		})
		if got := LatestGrokSessionID(cfg, cwd); got != "01a00000-0000-0000-0000-00000000bbbb" {
			t.Errorf("got %q, want the newest session bbbb", got)
		}
	})

	t.Run("empty cwd is a no-op", func(t *testing.T) {
		cfg := newFixtureWithSessions(t, map[string]time.Time{
			"01a00000-0000-0000-0000-00000000aaaa": time.Now(),
		})
		if got := LatestGrokSessionID(cfg, ""); got != "" {
			t.Errorf("empty cwd should return empty, got %q", got)
		}
	})

	t.Run("no matching cwd dir returns empty", func(t *testing.T) {
		cfg := GrokHistoryConfig{SessionsRoots: []string{t.TempDir()}}
		if got := LatestGrokSessionID(cfg, cwd); got != "" {
			t.Errorf("expected empty when no session dir exists, got %q", got)
		}
	})

	t.Run("dir without chat_history is ignored", func(t *testing.T) {
		root := t.TempDir()
		encoded := strings.ReplaceAll(cwd, "/", "%2F")
		// A session dir with no transcript yet must not be selected.
		if err := os.MkdirAll(filepath.Join(root, encoded, "01a00000-0000-0000-0000-0000000empty"), 0o755); err != nil {
			t.Fatal(err)
		}
		cfg := GrokHistoryConfig{SessionsRoots: []string{root}}
		if got := LatestGrokSessionID(cfg, cwd); got != "" {
			t.Errorf("dir without chat_history must be ignored, got %q", got)
		}
	})
}

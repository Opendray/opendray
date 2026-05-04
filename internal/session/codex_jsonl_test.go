package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodexInputHistory_FiltersByCwdAndStripsBootstrap(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	root := filepath.Join(tmpHome, ".codex", "sessions", "2026", "05", "03")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	otherRoot := filepath.Join(tmpHome, ".codex", "sessions", "2026", "05", "02")
	if err := os.MkdirAll(otherRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	// Two rollouts in our cwd: one with bootstrap+real, one with two reals.
	writeCodexRollout(t, filepath.Join(root, "rollout-2026-05-03T10-00-00-019aaaaa-bbbb-7000-8000-aaaaaaaaaaaa.jsonl"),
		"/tmp/proj/A",
		[]codexUserMsg{
			{ts: "2026-05-03T10:00:00Z", text: "<environment_context><cwd>/tmp/proj/A</cwd></environment_context>"},
			{ts: "2026-05-03T10:00:01Z", text: "describe the project"},
			{ts: "2026-05-03T10:05:00Z", text: "fix the bug"},
		})
	writeCodexRollout(t, filepath.Join(root, "rollout-2026-05-03T11-00-00-019bbbbb-bbbb-7000-8000-bbbbbbbbbbbb.jsonl"),
		"/tmp/proj/A",
		[]codexUserMsg{
			{ts: "2026-05-03T11:00:00Z", text: "AGENTS.md instructions for /tmp/proj/A"},
			{ts: "2026-05-03T11:00:01Z", text: "second session prompt"},
		})
	// One rollout in a DIFFERENT cwd — must be ignored.
	writeCodexRollout(t, filepath.Join(otherRoot, "rollout-2026-05-02T09-00-00-019cccc1-cccc-7000-8000-cccccccccccc.jsonl"),
		"/tmp/proj/B",
		[]codexUserMsg{
			{ts: "2026-05-02T09:00:00Z", text: "should not appear"},
		})

	got := CodexInputHistory(CodexHistoryConfig{}, "/tmp/proj/A", 10)

	wantTexts := []string{
		"second session prompt", // 11:00:01
		"fix the bug",           // 10:05:00
		"describe the project",  // 10:00:01
	}
	if len(got) != len(wantTexts) {
		t.Fatalf("got %d entries, want %d:\n%+v", len(got), len(wantTexts), got)
	}
	for i, want := range wantTexts {
		if got[i].Text != want {
			t.Errorf("entry %d = %q, want %q", i, got[i].Text, want)
		}
	}
	for _, e := range got {
		if strings.Contains(e.Text, "<environment_context>") {
			t.Errorf("bootstrap leaked: %q", e.Text)
		}
		if strings.Contains(e.Text, "AGENTS.md") {
			t.Errorf("AGENTS.md bootstrap leaked: %q", e.Text)
		}
	}
}

func TestCodexInputHistory_NoSessions(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	if got := CodexInputHistory(CodexHistoryConfig{}, "/tmp/missing", 10); got != nil {
		t.Errorf("want nil, got %v", got)
	}
}

func TestCodexInputHistory_LimitTrims(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	root := filepath.Join(tmpHome, ".codex", "sessions", "2026", "05", "03")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	msgs := make([]codexUserMsg, 0, 25)
	for i := 0; i < 25; i++ {
		ts := time2026("10:" + twoDigit(i) + ":00Z")
		msgs = append(msgs, codexUserMsg{ts: ts, text: "p" + twoDigit(i)})
	}
	writeCodexRollout(t, filepath.Join(root, "rollout-2026-05-03T10-00-00-019dddd1-dddd-7000-8000-dddddddddddd.jsonl"),
		"/tmp/proj/L", msgs)

	got := CodexInputHistory(CodexHistoryConfig{}, "/tmp/proj/L", 5)
	if len(got) != 5 {
		t.Fatalf("limit not honoured: got %d, want 5", len(got))
	}
	if got[0].Text != "p24" {
		t.Errorf("newest first failed: %q", got[0].Text)
	}
}

func TestCodexInputHistory_CustomSessionsRoot(t *testing.T) {
	t.Setenv("HOME", "") // default discovery would yield nothing
	tmp := t.TempDir()
	root := filepath.Join(tmp, "custom-codex", "2026", "05", "03")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	writeCodexRollout(t, filepath.Join(root, "rollout-2026-05-03T10-00-00-019eeee1-eeee-7000-8000-eeeeeeeeeeee.jsonl"),
		"/tmp/proj/Custom",
		[]codexUserMsg{{ts: "2026-05-03T10:00:00Z", text: "from custom codex root"}})

	got := CodexInputHistory(
		CodexHistoryConfig{SessionsRoot: filepath.Join(tmp, "custom-codex")},
		"/tmp/proj/Custom", 10,
	)
	if len(got) != 1 || got[0].Text != "from custom codex root" {
		t.Errorf("custom SessionsRoot not honoured: %+v", got)
	}
}

func TestCodexSessionIDFromName(t *testing.T) {
	got := codexSessionIDFromName("rollout-2026-05-03T23-00-35-019deded-27fd-7180-81a5-c6f2ce6ca0ee.jsonl")
	want := "019deded-27fd-7180-81a5-c6f2ce6ca0ee"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

// codexUserMsg is a tiny test struct: timestamp + text for one
// "response_item.message.user" entry.
type codexUserMsg struct {
	ts   string
	text string
}

// writeCodexRollout writes one .jsonl with a session_meta header
// (carrying cwd) followed by one response_item entry per msg.
func writeCodexRollout(t *testing.T, path, cwd string, msgs []codexUserMsg) {
	t.Helper()
	var b strings.Builder
	meta := map[string]any{
		"timestamp": msgsTs(msgs, 0),
		"type":      "session_meta",
		"payload": map[string]any{
			"cwd": cwd,
		},
	}
	mustJSONL(t, &b, meta)
	for _, m := range msgs {
		entry := map[string]any{
			"timestamp": m.ts,
			"type":      "response_item",
			"payload": map[string]any{
				"type": "message",
				"role": "user",
				"content": []any{
					map[string]any{"type": "input_text", "text": m.text},
				},
			},
		}
		mustJSONL(t, &b, entry)
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
}

func mustJSONL(t *testing.T, b *strings.Builder, v any) {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	b.Write(raw)
	b.WriteByte('\n')
}

func msgsTs(msgs []codexUserMsg, i int) string {
	if i < len(msgs) {
		return msgs[i].ts
	}
	return "2026-01-01T00:00:00Z"
}

func twoDigit(i int) string {
	if i < 10 {
		return "0" + string(rune('0'+i))
	}
	tens := i / 10
	ones := i % 10
	return string(rune('0'+tens)) + string(rune('0'+ones))
}

func time2026(suffix string) string { return "2026-05-03T" + suffix }

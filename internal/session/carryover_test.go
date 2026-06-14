package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeJSONL writes a minimal Claude transcript file under a project
// dir that findClaudeProjectDir will match for `cwd`.
func writeCarryoverFixture(t *testing.T, cwd, sessionID string, lines []string) ClaudeHistoryConfig {
	t.Helper()
	root := t.TempDir()
	// Claude encodes the cwd into the project dir name by replacing
	// path separators with dashes. findClaudeProjectDir matches fuzzily
	// (every cwd component must appear), so a dash-joined name works.
	encoded := strings.ReplaceAll(strings.TrimPrefix(cwd, "/"), "/", "-")
	dir := filepath.Join(root, "-"+encoded)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(dir, sessionID+".jsonl"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return ClaudeHistoryConfig{HistoryRoots: []string{root}}
}

func TestBuildClaudeCarryover(t *testing.T) {
	cwd := "/home/op/project"
	sid := "11111111-1111-1111-1111-111111111111"

	t.Run("text turns kept, tool noise dropped, labeled block", func(t *testing.T) {
		cfg := writeCarryoverFixture(t, cwd, sid, []string{
			`{"type":"user","message":{"role":"user","content":"add a login form"}}`,
			`{"type":"assistant","message":{"role":"assistant","content":[{"type":"thinking","text":"hmm"},{"type":"text","text":"On it — creating the form."},{"type":"tool_use","name":"Write","tool_use_id":"t1"}]}}`,
			`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"t1","content":"ok"}]}}`,
		})
		got := BuildClaudeCarryover(cfg, cwd, sid, 0)
		if got == "" {
			t.Fatal("expected a recap, got empty")
		}
		if !strings.Contains(got, "Carried-over context from a previous account") {
			t.Error("missing header")
		}
		if !strings.Contains(got, "You: add a login form") {
			t.Errorf("missing user turn:\n%s", got)
		}
		if !strings.Contains(got, "Assistant: On it — creating the form.") {
			t.Errorf("missing assistant text turn:\n%s", got)
		}
		if strings.Contains(got, "thinking") || strings.Contains(got, "tool_use") || strings.Contains(got, "tool_result") {
			t.Errorf("tool/thinking noise leaked into recap:\n%s", got)
		}
		if !strings.Contains(got, "End of carried-over context") {
			t.Error("missing footer")
		}
	})

	t.Run("missing transcript returns empty (degrade to fresh)", func(t *testing.T) {
		cfg := writeCarryoverFixture(t, cwd, sid, []string{
			`{"type":"user","message":{"role":"user","content":"hi"}}`,
		})
		// Ask for a different session id than the one on disk.
		got := BuildClaudeCarryover(cfg, cwd, "22222222-2222-2222-2222-222222222222", 0)
		if got != "" {
			t.Errorf("expected empty for missing transcript, got:\n%s", got)
		}
	})

	t.Run("empty inputs are no-ops", func(t *testing.T) {
		cfg := ClaudeHistoryConfig{HistoryRoots: []string{t.TempDir()}}
		if BuildClaudeCarryover(cfg, "", sid, 0) != "" {
			t.Error("empty cwd should return empty")
		}
		if BuildClaudeCarryover(cfg, cwd, "", 0) != "" {
			t.Error("empty sessionID should return empty")
		}
	})

	t.Run("tail-truncates to budget with elision marker", func(t *testing.T) {
		// Three large turns; a tight budget should keep only the most
		// recent and mark the rest elided.
		big := strings.Repeat("x", 2000)
		cfg := writeCarryoverFixture(t, cwd, sid, []string{
			`{"type":"user","message":{"role":"user","content":"OLDEST ` + big + `"}}`,
			`{"type":"user","message":{"role":"user","content":"MIDDLE ` + big + `"}}`,
			`{"type":"user","message":{"role":"user","content":"NEWEST ` + big + `"}}`,
		})
		// Budget big enough for header+footer+one turn (~2.5k) but not all.
		got := BuildClaudeCarryover(cfg, cwd, sid, 3200)
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
}

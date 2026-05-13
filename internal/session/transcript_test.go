package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// M19 — verify the transcript readers handle synthetic Claude /
// Codex / Gemini fixtures. Uses real on-disk files so the readers
// exercise their actual filesystem path lookup, not just the parse
// logic.

func TestClaudeTranscript_Synthetic(t *testing.T) {
	dir := t.TempDir()
	projectsRoot := filepath.Join(dir, ".claude", "projects")
	// Claude encodes the cwd as a flat dir name: leading slash
	// dropped, slashes → dashes. /tmp/test-project → -tmp-test-project
	cwd := "/tmp/test-project"
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}
	projDir := filepath.Join(projectsRoot, "-tmp-test-project")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Write a synthetic JSONL with three turns: user, assistant
	// (text + tool_use), user.
	jsonlPath := filepath.Join(projDir, "session-1.jsonl")
	lines := []string{
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"refactor the auth module"}]},"timestamp":"2026-05-12T10:00:00Z"}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"I'll start by extracting the bcrypt helper."},{"type":"tool_use","name":"Edit","input":{"file_path":"internal/auth/bcrypt.go"}},{"type":"tool_use","name":"Bash","input":{"command":"go test ./internal/auth/..."}}]},"timestamp":"2026-05-12T10:01:00Z"}`,
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"looks good, ship it"}]},"timestamp":"2026-05-12T10:02:00Z"}`,
	}
	if err := os.WriteFile(jsonlPath, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}

	turns := claudeTranscript(
		ClaudeHistoryConfig{HistoryRoots: []string{projectsRoot}},
		cwd, "", time.Time{}, time.Time{}, 16*1024,
	)
	if len(turns) != 3 {
		t.Fatalf("want 3 turns, got %d: %+v", len(turns), turns)
	}
	if turns[0].Role != "user" || !strings.Contains(turns[0].Text, "refactor") {
		t.Errorf("first turn wrong: %+v", turns[0])
	}
	if turns[1].Role != "assistant" {
		t.Errorf("second turn role: %s", turns[1].Role)
	}
	// Assistant text + tool-use summaries should be inline.
	for _, want := range []string{"bcrypt helper", "(Edit internal/auth/bcrypt.go)", "(Bash: go test"} {
		if !strings.Contains(turns[1].Text, want) {
			t.Errorf("assistant turn missing %q\nfull: %s", want, turns[1].Text)
		}
	}
	if turns[2].Role != "user" || !strings.Contains(turns[2].Text, "ship") {
		t.Errorf("third turn wrong: %+v", turns[2])
	}
}

func TestClaudeTranscript_DropsToolResults(t *testing.T) {
	dir := t.TempDir()
	cwd := "/tmp/cw"
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}
	projDir := filepath.Join(dir, "-tmp-cw")
	_ = os.MkdirAll(projDir, 0o755)
	lines := []string{
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"running tests"},{"type":"tool_use","name":"Bash","input":{"command":"go test ./..."}}]}}`,
		`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"x","content":"---FAIL: TestFoo (0.01s)\nmuch noise\nmuch noise\nmuch noise"}]}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"the test failed because of X"}]}}`,
	}
	jsonlPath := filepath.Join(projDir, "s.jsonl")
	_ = os.WriteFile(jsonlPath, []byte(strings.Join(lines, "\n")), 0o644)

	turns := claudeTranscript(
		ClaudeHistoryConfig{HistoryRoots: []string{dir}},
		cwd, "", time.Time{}, time.Time{}, 16*1024,
	)
	for _, tn := range turns {
		if strings.Contains(tn.Text, "much noise") || strings.Contains(tn.Text, "FAIL: TestFoo") {
			t.Errorf("tool_result content leaked into transcript: %s", tn.Text)
		}
	}
	// Should still have the assistant text turns.
	if len(turns) < 2 {
		t.Errorf("expected ≥2 assistant turns, got %d", len(turns))
	}
}

func TestCodexTranscript_Synthetic(t *testing.T) {
	dir := t.TempDir()
	sessionsRoot := filepath.Join(dir, "sessions")
	_ = os.MkdirAll(filepath.Join(sessionsRoot, "2026", "05", "12"), 0o755)
	cwd := "/tmp/codex-cw"

	// Codex rollout format: top-level `type` discriminates
	// session_meta vs response_item; per-message info nests under
	// payload. See internal/session/codex_jsonl.go for the
	// canonical shape.
	lines := []string{
		`{"timestamp":"2026-05-12T10:00:00Z","type":"session_meta","payload":{"cwd":"/tmp/codex-cw","model":"o4-mini"}}`,
		`{"timestamp":"2026-05-12T10:01:00Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"text":"refactor auth"}]}}`,
		`{"timestamp":"2026-05-12T10:02:00Z","type":"response_item","payload":{"type":"message","role":"assistant","content":[{"text":"extracted bcrypt helper"}]}}`,
		`{"timestamp":"2026-05-12T10:03:00Z","type":"response_item","payload":{"type":"function_call","name":"edit_file","arguments":"{\"path\":\"a.go\"}"}}`,
	}
	jsonlPath := filepath.Join(sessionsRoot, "2026", "05", "12", "rollout-abc.jsonl")
	_ = os.WriteFile(jsonlPath, []byte(strings.Join(lines, "\n")), 0o644)

	turns := codexTranscript(
		CodexHistoryConfig{SessionsRoot: sessionsRoot},
		cwd, time.Time{}, time.Time{}, 16*1024,
	)
	if len(turns) != 3 {
		t.Fatalf("want 3 turns, got %d: %+v", len(turns), turns)
	}
	if turns[0].Role != "user" || !strings.Contains(turns[0].Text, "refactor") {
		t.Errorf("first turn wrong: %+v", turns[0])
	}
	if turns[1].Role != "assistant" || !strings.Contains(turns[1].Text, "bcrypt") {
		t.Errorf("second turn wrong: %+v", turns[1])
	}
	if turns[2].Role != "assistant" || !strings.Contains(turns[2].Text, "edit_file") {
		t.Errorf("function_call turn wrong: %+v", turns[2])
	}
}

func TestCodexTranscript_FiltersByCwd(t *testing.T) {
	dir := t.TempDir()
	sessionsRoot := filepath.Join(dir, "sessions")
	_ = os.MkdirAll(sessionsRoot, 0o755)

	// Two rollout files, two different cwds. We should only see
	// the one matching our cwd.
	rolloutA := filepath.Join(sessionsRoot, "a.jsonl")
	_ = os.WriteFile(rolloutA, []byte(strings.Join([]string{
		`{"type":"session_meta","payload":{"cwd":"/tmp/other"}}`,
		`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"text":"ignore me"}]}}`,
	}, "\n")), 0o644)
	rolloutB := filepath.Join(sessionsRoot, "b.jsonl")
	_ = os.WriteFile(rolloutB, []byte(strings.Join([]string{
		`{"type":"session_meta","payload":{"cwd":"/tmp/want"}}`,
		`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"text":"pick me"}]}}`,
	}, "\n")), 0o644)

	turns := codexTranscript(CodexHistoryConfig{SessionsRoot: sessionsRoot}, "/tmp/want", time.Time{}, time.Time{}, 16*1024)
	if len(turns) != 1 || !strings.Contains(turns[0].Text, "pick me") {
		t.Errorf("wrong rollout matched: %+v", turns)
	}
}

func TestGeminiTranscript_Synthetic(t *testing.T) {
	dir := t.TempDir()
	chatsPath := filepath.Join(dir, "chats.json")

	doc := map[string]any{
		"sessions": []map[string]any{
			{
				"cwd":     "/tmp/gemini-cw",
				"updated": time.Now().UTC().Format(time.RFC3339),
				"messages": []map[string]any{
					{"role": "user", "content": "refactor please", "timestamp": "2026-05-12T10:00:00Z"},
					{"role": "model", "content": "I extracted the bcrypt helper", "timestamp": "2026-05-12T10:01:00Z"},
				},
			},
			{
				"cwd":     "/tmp/other",
				"updated": time.Now().UTC().Format(time.RFC3339),
				"messages": []map[string]any{
					{"role": "user", "content": "different project", "timestamp": "2026-05-12T11:00:00Z"},
				},
			},
		},
	}
	body, _ := json.Marshal(doc)
	_ = os.WriteFile(chatsPath, body, 0o644)

	turns := geminiTranscript(
		GeminiHistoryConfig{ProjectsFile: chatsPath},
		"/tmp/gemini-cw", time.Time{}, time.Time{}, 16*1024,
	)
	if len(turns) != 2 {
		t.Fatalf("want 2 turns, got %d: %+v", len(turns), turns)
	}
	if turns[0].Role != "user" || !strings.Contains(turns[0].Text, "refactor") {
		t.Errorf("user turn wrong: %+v", turns[0])
	}
	// "model" should normalise to "assistant" so the journaler /
	// summariser see a unified role enum.
	if turns[1].Role != "assistant" {
		t.Errorf("model role should normalise to assistant, got %q", turns[1].Role)
	}
	if !strings.Contains(turns[1].Text, "bcrypt") {
		t.Errorf("assistant text wrong: %+v", turns[1])
	}
	// Verify the "other" session is excluded.
	for _, tn := range turns {
		if strings.Contains(tn.Text, "different project") {
			t.Errorf("wrong session leaked: %+v", tn)
		}
	}
}

func TestFormatTranscript(t *testing.T) {
	out := FormatTranscript([]Turn{
		{Role: "user", Text: "hi"},
		{Role: "assistant", Text: "hello"},
		{Role: "user", Text: "do X"},
	})
	want := "USER: hi\nASSISTANT: hello\nUSER: do X"
	if out != want {
		t.Errorf("FormatTranscript mismatch:\n got:\n%s\nwant:\n%s", out, want)
	}
}

// M22 — three layers of isolation against transcript leakage.
// Each test isolates one defense so a regression in any single
// layer is caught directly.

func TestClaudeTranscript_FailClosedOnMissingUUID(t *testing.T) {
	// Caller asks for a specific session UUID. The file doesn't
	// exist. There ARE other jsonls in the dir (newer mtime). The
	// reader MUST NOT fall back to those — that's how unrelated
	// sessions leak in. Pre-M22 this returned the newest jsonl;
	// post-M22 it returns nil.
	dir := t.TempDir()
	cwd := "/tmp/proj"
	_ = os.MkdirAll(cwd, 0o755)
	projDir := filepath.Join(dir, "-tmp-proj")
	_ = os.MkdirAll(projDir, 0o755)
	// A jsonl from an UNRELATED session.
	_ = os.WriteFile(filepath.Join(projDir, "other-session.jsonl"),
		[]byte(`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"unrelated work"}]},"cwd":"/tmp/proj"}`),
		0o644)

	turns := claudeTranscript(
		ClaudeHistoryConfig{HistoryRoots: []string{dir}},
		cwd, "missing-uuid", time.Time{}, time.Time{}, 16*1024,
	)
	if len(turns) != 0 {
		t.Errorf("expected empty when requested UUID missing; got %d turns. Pre-M22 regression: reader fell back to latest mtime and leaked unrelated session content.", len(turns))
	}
}

func TestClaudeTranscript_CwdCanaryRejectsWrongProject(t *testing.T) {
	// Reader is asked for cwd /tmp/foo but the jsonl claims its
	// cwd is /tmp/bar (e.g. Claude Code mis-routed the file).
	// The whole file must be rejected — better empty than wrong.
	dir := t.TempDir()
	cwd := "/tmp/foo"
	_ = os.MkdirAll(cwd, 0o755)
	projDir := filepath.Join(dir, "-tmp-foo")
	_ = os.MkdirAll(projDir, 0o755)
	jsonlPath := filepath.Join(projDir, "s.jsonl")
	// First entry's cwd is /tmp/bar — mismatch.
	_ = os.WriteFile(jsonlPath, []byte(strings.Join([]string{
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"work from wrong project"}]},"cwd":"/tmp/bar"}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"wrong project response"}]}}`,
	}, "\n")), 0o644)

	turns := claudeTranscript(
		ClaudeHistoryConfig{HistoryRoots: []string{dir}},
		cwd, "", time.Time{}, time.Time{}, 16*1024,
	)
	if len(turns) != 0 {
		t.Errorf("cwd canary should have rejected wrong-cwd jsonl; got %d turns: %+v", len(turns), turns)
	}
}

func TestClaudeTranscript_TimeWindowFiltersAccumulatedFile(t *testing.T) {
	// Mimics the production case where Claude Code's jsonl
	// accumulates content across multiple opendray spawns in the
	// same cwd. Only turns within [startedAt-30s, endedAt+30s]
	// must survive.
	dir := t.TempDir()
	cwd := "/tmp/p"
	_ = os.MkdirAll(cwd, 0o755)
	projDir := filepath.Join(dir, "-tmp-p")
	_ = os.MkdirAll(projDir, 0o755)
	jsonlPath := filepath.Join(projDir, "long-session.jsonl")
	// 3 turns: one weeks ago, one inside the window, one days ago.
	weeksAgo := time.Now().Add(-30 * 24 * time.Hour).UTC().Format(time.RFC3339)
	insideWindow := time.Now().Add(-5 * time.Minute).UTC().Format(time.RFC3339)
	daysAgo := time.Now().Add(-2 * 24 * time.Hour).UTC().Format(time.RFC3339)
	_ = os.WriteFile(jsonlPath, []byte(strings.Join([]string{
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"ancient work"}]},"cwd":"/tmp/p","timestamp":"` + weeksAgo + `"}`,
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"current work"}]},"cwd":"/tmp/p","timestamp":"` + insideWindow + `"}`,
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"recent but pre-spawn"}]},"cwd":"/tmp/p","timestamp":"` + daysAgo + `"}`,
	}, "\n")), 0o644)

	startedAt := time.Now().Add(-10 * time.Minute)
	endedAt := time.Now().Add(-1 * time.Minute)
	turns := claudeTranscript(
		ClaudeHistoryConfig{HistoryRoots: []string{dir}},
		cwd, "", startedAt, endedAt, 16*1024,
	)
	if len(turns) != 1 {
		t.Fatalf("expected exactly the inside-window turn; got %d: %+v", len(turns), turns)
	}
	if !strings.Contains(turns[0].Text, "current") {
		t.Errorf("wrong turn survived; want 'current work', got %q", turns[0].Text)
	}
}

func TestTrimTurnsHead_BytesBudget(t *testing.T) {
	// Build 10 turns of 100 bytes each, budget 300 bytes.
	var turns []Turn
	for i := 0; i < 10; i++ {
		turns = append(turns, Turn{Role: "user", Text: strings.Repeat("a", 100)})
	}
	bytes := 10 * (100 + 4 + 4) // approx accounting per turn
	turns = trimTurnsHead(turns, &bytes, 300)
	if len(turns) > 4 {
		t.Errorf("trim left too many turns: %d", len(turns))
	}
}

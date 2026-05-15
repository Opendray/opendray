package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMatchesClaudeProjectDir(t *testing.T) {
	cases := []struct {
		name    string
		encoded string
		cwd     string
		want    bool
	}{
		{
			name:    "exact path components",
			encoded: "-Users-alice-projects-opendray-v2",
			cwd:     "/Users/alice/projects/opendray-v2",
			want:    true,
		},
		{
			name:    "underscore→dash normalisation",
			encoded: "-Users-alice-Documents-Claude-Workspace-opendray-v2",
			cwd:     "/Users/alice/Documents/Claude_Workspace/opendray-v2",
			want:    true,
		},
		{
			name:    "missing component",
			encoded: "-Users-alice-projects-different-app",
			cwd:     "/Users/alice/projects/opendray-v2",
			want:    false,
		},
		{
			name:    "out-of-order parts",
			encoded: "-Users-alice-v2-opendray-projects",
			cwd:     "/Users/alice/projects/opendray-v2",
			want:    false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			parts := splitPathParts(c.cwd)
			got := matchesClaudeProjectDir(c.encoded, parts)
			if got != c.want {
				t.Errorf("got %v want %v", got, c.want)
			}
		})
	}
}

func TestRenderAssistantBlocks_BulletText(t *testing.T) {
	blocks := []claudeContentBlock{
		{Type: "text", Text: "Here is your answer.\n\nIt has two paragraphs."},
	}
	var b strings.Builder
	renderAssistantBlocks(&b, blocks, map[string]string{})
	out := b.String()
	if !strings.HasPrefix(out, "● ") {
		t.Errorf("text bullet missing:\n%s", out)
	}
	if !strings.Contains(out, "Here is your answer.") || !strings.Contains(out, "two paragraphs") {
		t.Errorf("text body lost:\n%s", out)
	}
}

func TestRenderAssistantBlocks_DropsThinking(t *testing.T) {
	blocks := []claudeContentBlock{
		{Type: "thinking", Text: "<internal reasoning the user shouldn't see>"},
		{Type: "text", Text: "Visible reply."},
	}
	var b strings.Builder
	renderAssistantBlocks(&b, blocks, map[string]string{})
	out := b.String()
	if strings.Contains(out, "internal reasoning") {
		t.Errorf("thinking leaked: %s", out)
	}
	if !strings.Contains(out, "Visible reply") {
		t.Errorf("text dropped: %s", out)
	}
}

func TestFormatToolUse_FilePathTitle(t *testing.T) {
	b := claudeContentBlock{
		Type: "tool_use",
		Name: "Write",
		Input: newToolInput(t, map[string]string{
			"file_path": "docs/ANDROID_PORT_SPEC.md",
		}),
	}
	got := formatToolUse(b)
	if got != "Write(docs/ANDROID_PORT_SPEC.md)" {
		t.Errorf("format = %q", got)
	}
}

func TestFormatToolUse_BashCommandWithDesc(t *testing.T) {
	b := claudeContentBlock{
		Type: "tool_use",
		Name: "Bash",
		Input: newToolInput(t, map[string]string{
			"command":     "rg 'TODO' --type go",
			"description": "find outstanding TODOs",
		}),
	}
	got := formatToolUse(b)
	if !strings.Contains(got, "Bash(rg 'TODO' --type go)") || !strings.Contains(got, "find outstanding") {
		t.Errorf("format = %q", got)
	}
}

func TestRenderToolResults_AttachedAfterToolUse(t *testing.T) {
	useID := "tool_abc"
	summary := map[string]string{useID: "Write(README.md)"}
	resultBlocks := []claudeContentBlock{{
		Type:       "tool_result",
		ToolUseID:  useID,
		RawContent: mustRawString(t, "Wrote 734 lines to README.md\n\n  1 # Hello\n  2 World"),
	}}
	var b strings.Builder
	renderToolResults(&b, resultBlocks, summary)
	out := b.String()
	if !strings.HasPrefix(strings.TrimSpace(out), "└ Wrote 734 lines") {
		t.Errorf("result not rendered with └ prefix:\n%s", out)
	}
	// summary[useID] should be cleared after the result is consumed.
	if _, still := summary[useID]; still {
		t.Errorf("summary not cleared after tool_result attach")
	}
}

func TestRenderClaudeRecentTurn_FullFlow(t *testing.T) {
	tmp := t.TempDir()
	jsonl := filepath.Join(tmp, "session.jsonl")

	lines := []string{
		mustEntry(t, "user", "create the Android spec doc"),
		assistantWithTool(t, "我将创建 Android 移植规格文档到 docs/ANDROID_PORT_SPEC.md。",
			"tool_use_1", "Write", map[string]string{
				"file_path": "docs/ANDROID_PORT_SPEC.md",
			}),
		userToolResult(t, "tool_use_1", "Wrote 734 lines to docs/ANDROID_PORT_SPEC.md"),
		mustEntry(t, "assistant", "已创建 Android 移植规格文档：docs/ANDROID_PORT_SPEC.md"),
	}
	if err := os.WriteFile(jsonl, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := renderClaudeRecentTurn(jsonl, 30)
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"● 我将创建 Android 移植规格文档",
		"● Write(docs/ANDROID_PORT_SPEC.md)",
		"└ Wrote 734 lines",
		"● 已创建 Android 移植规格文档",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q in:\n%s", want, got)
		}
	}
}

func TestLastAssistantText_PicksMostRecent_Compat(t *testing.T) {
	tmp := t.TempDir()
	jsonl := filepath.Join(tmp, "session.jsonl")
	lines := []string{
		mustEntry(t, "user", "first user message"),
		mustEntry(t, "assistant", "old reply (should be replaced)"),
		mustEntry(t, "user", "follow-up question"),
		mustEntry(t, "assistant", "newest reply — this should win"),
	}
	if err := os.WriteFile(jsonl, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := renderClaudeRecentTurn(jsonl, 30)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "newest reply") {
		t.Errorf("did not pick most recent: %q", got)
	}
	if strings.Contains(got, "old reply") {
		t.Errorf("returned stale assistant entry: %q", got)
	}
}

// newToolInput materialises the anonymous Input struct used inside
// claudeContentBlock from a string-keyed map. Test helper.
func newToolInput(t *testing.T, kv map[string]string) *struct {
	Command     string `json:"command,omitempty"`
	Description string `json:"description,omitempty"`
	Content     string `json:"content,omitempty"`
	FilePath    string `json:"file_path,omitempty"`
	Path        string `json:"path,omitempty"`
	Pattern     string `json:"pattern,omitempty"`
	Query       string `json:"query,omitempty"`
	Question    string `json:"question,omitempty"`
	URL         string `json:"url,omitempty"`
	Prompt      string `json:"prompt,omitempty"`
} {
	t.Helper()
	in := &struct {
		Command     string `json:"command,omitempty"`
		Description string `json:"description,omitempty"`
		Content     string `json:"content,omitempty"`
		FilePath    string `json:"file_path,omitempty"`
		Path        string `json:"path,omitempty"`
		Pattern     string `json:"pattern,omitempty"`
		Query       string `json:"query,omitempty"`
		Question    string `json:"question,omitempty"`
		URL         string `json:"url,omitempty"`
		Prompt      string `json:"prompt,omitempty"`
	}{}
	in.Command = kv["command"]
	in.Description = kv["description"]
	in.FilePath = kv["file_path"]
	in.Path = kv["path"]
	in.Pattern = kv["pattern"]
	in.Query = kv["query"]
	in.Question = kv["question"]
	in.URL = kv["url"]
	in.Prompt = kv["prompt"]
	return in
}

func mustRawString(t *testing.T, s string) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// assistantWithTool builds a JSONL line for an assistant entry that
// contains one text block followed by a single tool_use block with
// the given id, name, and input map.
func assistantWithTool(t *testing.T, text, useID, toolName string, input map[string]string) string {
	t.Helper()
	textBlock := map[string]any{"type": "text", "text": text}
	inputObj := map[string]any{}
	for k, v := range input {
		inputObj[k] = v
	}
	toolBlock := map[string]any{
		"type":  "tool_use",
		"id":    useID,
		"name":  toolName,
		"input": inputObj,
	}
	// JSONL stores tool_use_id under the actual JSON key the
	// production decoder expects ("tool_use_id" on tool_result;
	// tool_use entries set "id"). For tool_use blocks we mirror the
	// id under both keys so the unmarshaler picks it up regardless
	// of which field the runtime ends up using.
	toolBlock["tool_use_id"] = useID
	content, _ := json.Marshal([]any{textBlock, toolBlock})
	entry := map[string]any{
		"type": "assistant",
		"message": map[string]any{
			"role":    "assistant",
			"content": json.RawMessage(content),
		},
	}
	raw, err := json.Marshal(entry)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// userToolResult builds a JSONL line for a user entry whose content
// is a single tool_result block addressing the given tool_use_id.
func userToolResult(t *testing.T, useID, body string) string {
	t.Helper()
	resultBlock := map[string]any{
		"type":        "tool_result",
		"tool_use_id": useID,
		"content":     body,
	}
	content, _ := json.Marshal([]any{resultBlock})
	entry := map[string]any{
		"type": "user",
		"message": map[string]any{
			"role":    "user",
			"content": json.RawMessage(content),
		},
	}
	raw, err := json.Marshal(entry)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestProjectInputHistory_MergesAcrossSessions(t *testing.T) {
	tmpHome := t.TempDir()
	encoded := "-tmp-fake-cwd"
	pdir := filepath.Join(tmpHome, ".claude", "projects", encoded)
	if err := os.MkdirAll(pdir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Two sessions, interleaved timestamps. Each session has a mix
	// of user prompts (string content + array content) and one
	// tool_result-only entry that must be skipped.
	sessionA := filepath.Join(pdir, "sess-A.jsonl")
	sessionB := filepath.Join(pdir, "sess-B.jsonl")

	writeJSONL(t, sessionA, []map[string]any{
		userEntryString(t, "first prompt in A", "2026-05-04T10:00:00Z"),
		assistantEntry(t, "ok", "2026-05-04T10:00:30Z"),
		userToolResultOnly(t, "tool_a", "2026-05-04T10:01:00Z"), // skipped
		userEntryString(t, "second prompt in A", "2026-05-04T10:02:00Z"),
	})
	writeJSONL(t, sessionB, []map[string]any{
		userEntryString(t, "first prompt in B", "2026-05-04T10:01:30Z"),
		userEntryArray(t, "second prompt in B", "2026-05-04T10:03:00Z"),
	})

	t.Setenv("HOME", tmpHome)
	got := ProjectInputHistory(ClaudeHistoryConfig{}, "/tmp/fake/cwd", 50)

	wantTexts := []string{
		"second prompt in B", // 10:03:00 — newest
		"second prompt in A", // 10:02:00
		"first prompt in B",  // 10:01:30
		"first prompt in A",  // 10:00:00 — oldest
	}
	if len(got) != len(wantTexts) {
		t.Fatalf("got %d entries, want %d:\n%+v", len(got), len(wantTexts), got)
	}
	for i, want := range wantTexts {
		if got[i].Text != want {
			t.Errorf("entry %d text = %q, want %q", i, got[i].Text, want)
		}
	}
	// Session id round-trip check.
	if got[0].SessionID != "sess-B" {
		t.Errorf("session id for newest = %q, want sess-B", got[0].SessionID)
	}
}

func TestProjectInputHistory_LimitTrims(t *testing.T) {
	tmpHome := t.TempDir()
	encoded := "-tmp-fake-cwd"
	pdir := filepath.Join(tmpHome, ".claude", "projects", encoded)
	if err := os.MkdirAll(pdir, 0o755); err != nil {
		t.Fatal(err)
	}

	entries := make([]map[string]any, 0, 30)
	for i := 0; i < 30; i++ {
		ts := fmt.Sprintf("2026-05-04T10:%02d:00Z", i)
		entries = append(entries, userEntryString(t, fmt.Sprintf("prompt %d", i), ts))
	}
	writeJSONL(t, filepath.Join(pdir, "x.jsonl"), entries)

	t.Setenv("HOME", tmpHome)
	got := ProjectInputHistory(ClaudeHistoryConfig{}, "/tmp/fake/cwd", 5)

	if len(got) != 5 {
		t.Fatalf("limit not honoured: got %d, want 5", len(got))
	}
	if got[0].Text != "prompt 29" {
		t.Errorf("newest first failed: %q", got[0].Text)
	}
}

func TestProjectInputHistory_CustomHistoryRoots(t *testing.T) {
	// HOME stays empty; default discovery would yield nothing.
	t.Setenv("HOME", "")

	tmp := t.TempDir()
	pdir := filepath.Join(tmp, "custom-root", "-tmp-fake-cwd")
	if err := os.MkdirAll(pdir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeJSONL(t, filepath.Join(pdir, "x.jsonl"), []map[string]any{
		userEntryString(t, "from custom root", "2026-05-04T12:00:00Z"),
	})

	got := ProjectInputHistory(
		ClaudeHistoryConfig{HistoryRoots: []string{filepath.Join(tmp, "custom-root")}},
		"/tmp/fake/cwd", 10,
	)
	if len(got) != 1 || got[0].Text != "from custom root" {
		t.Errorf("custom HistoryRoots not honoured: %+v", got)
	}
}

func TestProjectInputHistory_CustomAccountsDir(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Account root is somewhere unrelated to ~/.claude-accounts —
	// only reachable through cfg.AccountsDir.
	customAccounts := filepath.Join(tmpHome, "elsewhere", "accounts")
	pdir := filepath.Join(customAccounts, "alice", "projects", "-tmp-fake-cwd")
	if err := os.MkdirAll(pdir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeJSONL(t, filepath.Join(pdir, "x.jsonl"), []map[string]any{
		userEntryString(t, "from custom accounts dir", "2026-05-04T12:00:00Z"),
	})

	got := ProjectInputHistory(
		ClaudeHistoryConfig{AccountsDir: customAccounts},
		"/tmp/fake/cwd", 10,
	)
	if len(got) != 1 || got[0].Text != "from custom accounts dir" {
		t.Errorf("custom AccountsDir not honoured: %+v", got)
	}
}

func TestProjectInputHistory_NoHomeOrProject(t *testing.T) {
	t.Setenv("HOME", "")
	if got := ProjectInputHistory(ClaudeHistoryConfig{}, "/anywhere", 50); got != nil {
		t.Errorf("no HOME: want nil, got %v", got)
	}

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	if got := ProjectInputHistory(ClaudeHistoryConfig{}, "/tmp/nothing-matches", 50); got != nil {
		t.Errorf("no project dir: want nil, got %v", got)
	}
}

func TestExtractUserText_ShapeVariants(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{"bare string", `"hello"`, "hello"},
		{"trim whitespace", `"  hello  "`, "hello"},
		{"text block", `[{"type":"text","text":"hi there"}]`, "hi there"},
		{"text + tool_result mixed → only text", `[{"type":"text","text":"go"},{"type":"tool_result","tool_use_id":"x","content":"output"}]`, "go"},
		{"only tool_result → empty", `[{"type":"tool_result","tool_use_id":"x","content":"output"}]`, ""},
		{"empty array → empty", `[]`, ""},
		{"empty string → empty", `""`, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := extractUserText([]byte(c.raw))
			if got != c.want {
				t.Errorf("extractUserText(%q) = %q, want %q", c.raw, got, c.want)
			}
		})
	}
}

func TestResolveLatestClaudeJSONL_RealFile(t *testing.T) {
	// Build a fake ~/.claude/projects/<encoded>/<sid>.jsonl tree under TempDir.
	tmpHome := t.TempDir()
	encoded := "-tmp-fake-cwd"
	pdir := filepath.Join(tmpHome, ".claude", "projects", encoded)
	if err := os.MkdirAll(pdir, 0o755); err != nil {
		t.Fatal(err)
	}
	jsonl := filepath.Join(pdir, "session-abc.jsonl")
	body := mustEntry(t, "assistant", "hello from JSONL") + "\n"
	if err := os.WriteFile(jsonl, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", tmpHome)
	got := resolveLatestClaudeJSONL("/tmp/fake/cwd")
	if got != jsonl {
		t.Errorf("resolveLatestClaudeJSONL = %q, want %q", got, jsonl)
	}
	resp := claudeRecentResponse("/tmp/fake/cwd")
	if !strings.Contains(resp, "hello from JSONL") {
		t.Errorf("claudeRecentResponse = %q", resp)
	}
}

// mustEntry serialises one JSONL line for a user/assistant message.
// Keeps tests succinct (each line is one assistant turn with one
// text block).
func mustEntry(t *testing.T, role, text string) string {
	t.Helper()
	content, err := json.Marshal([]claudeContentBlock{{Type: "text", Text: text}})
	if err != nil {
		t.Fatal(err)
	}
	entry := map[string]any{
		"type": role,
		"message": map[string]any{
			"role":    role,
			"content": json.RawMessage(content),
		},
	}
	raw, err := json.Marshal(entry)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// writeJSONL writes the given entries one-per-line to path, each
// marshalled as a single JSON object. Overwrites any existing file.
func writeJSONL(t *testing.T, path string, entries []map[string]any) {
	t.Helper()
	var b strings.Builder
	for _, e := range entries {
		raw, err := json.Marshal(e)
		if err != nil {
			t.Fatal(err)
		}
		b.Write(raw)
		b.WriteByte('\n')
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
}

// userEntryString builds a user JSONL entry whose content is a bare
// JSON string (the legacy "user typed this" shape).
func userEntryString(t *testing.T, text, ts string) map[string]any {
	t.Helper()
	content, err := json.Marshal(text)
	if err != nil {
		t.Fatal(err)
	}
	return map[string]any{
		"type":      "user",
		"timestamp": ts,
		"message": map[string]any{
			"role":    "user",
			"content": json.RawMessage(content),
		},
	}
}

// userEntryArray builds a user JSONL entry whose content is an array
// containing a single text block.
func userEntryArray(t *testing.T, text, ts string) map[string]any {
	t.Helper()
	content, err := json.Marshal([]any{
		map[string]any{"type": "text", "text": text},
	})
	if err != nil {
		t.Fatal(err)
	}
	return map[string]any{
		"type":      "user",
		"timestamp": ts,
		"message": map[string]any{
			"role":    "user",
			"content": json.RawMessage(content),
		},
	}
}

// assistantEntry builds an assistant JSONL entry with a single text
// block.
func assistantEntry(t *testing.T, text, ts string) map[string]any {
	t.Helper()
	content, err := json.Marshal([]any{
		map[string]any{"type": "text", "text": text},
	})
	if err != nil {
		t.Fatal(err)
	}
	return map[string]any{
		"type":      "assistant",
		"timestamp": ts,
		"message": map[string]any{
			"role":    "assistant",
			"content": json.RawMessage(content),
		},
	}
}

// userToolResultOnly builds a user JSONL entry whose content array
// holds only a tool_result block. Claude writes these so the next
// assistant turn can see the tool's output — they are not human
// input and ProjectInputHistory must skip them.
func userToolResultOnly(t *testing.T, toolUseID, ts string) map[string]any {
	t.Helper()
	content, err := json.Marshal([]any{
		map[string]any{
			"type":        "tool_result",
			"tool_use_id": toolUseID,
			"content":     "tool output goes here",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return map[string]any{
		"type":      "user",
		"timestamp": ts,
		"message": map[string]any{
			"role":    "user",
			"content": json.RawMessage(content),
		},
	}
}

// Regression guard: with "Unlimited — split into multiple messages"
// chosen at the channel layer, the source-level snippet must NOT
// silently truncate. These tests pin the new behaviour (full
// tool_use args, full tool_result body, multi-line indentation,
// generous JSONL window). If you reintroduce a cap, do it at the
// channel layer (notify_snippet_max_chars) — never here.
func TestFormatToolUse_LongBashCommandNotTruncated(t *testing.T) {
	longCmd := strings.Repeat("very-long-flag --option=value ", 30) // ~900 chars
	b := claudeContentBlock{
		Type: "tool_use",
		Name: "Bash",
		Input: newToolInput(t, map[string]string{
			"command": longCmd,
		}),
	}
	got := formatToolUse(b)
	if !strings.Contains(got, longCmd) {
		t.Errorf("long bash command was truncated:\ninput:  %q\nresult: %q", longCmd, got)
	}
	if strings.Contains(got, "…") {
		t.Errorf("unexpected truncation ellipsis for normal-length args: %q", got)
	}
}

func TestFormatToolUse_HugeArgStillSoftCapped(t *testing.T) {
	huge := strings.Repeat("X", toolUseArgSoftCap+1000)
	b := claudeContentBlock{
		Type:  "tool_use",
		Name:  "Bash",
		Input: newToolInput(t, map[string]string{"command": huge}),
	}
	got := formatToolUse(b)
	// Pathological input still gets clamped; the cap is generous
	// (>3× a typical 80-col terminal line) so normal flows never
	// hit it.
	if !strings.HasSuffix(got, "…)") {
		t.Errorf("expected ellipsis on pathological input, got tail: %q", got[len(got)-50:])
	}
}

func TestRenderToolResultBody_PreservesAllLines(t *testing.T) {
	body := "Wrote 734 lines to file.dart\n  1 import 'package:flutter/material.dart';\n  2 import 'dart:async';\n  3 \n  4 void main() {\n  5   runApp(const MyApp());\n  6 }\n  7 \n  8 // ... 727 more lines ..."
	got := renderToolResultBody(body)
	// First line gets the bullet; rest gets continuation indent.
	if !strings.HasPrefix(got, "  └ Wrote 734 lines") {
		t.Errorf("first-line prefix missing:\n%s", got)
	}
	// Every original non-empty line must survive.
	for _, want := range []string{
		"import 'package:flutter/material.dart';",
		"runApp(const MyApp());",
		"// ... 727 more lines ...",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("line dropped %q in:\n%s", want, got)
		}
	}
	// No 200-char cap → length should match the input plus prefixes.
	if len(got) < len(body) {
		t.Errorf("output shorter than input — content was lost; got %d bytes for %d-byte input",
			len(got), len(body))
	}
}

func TestRenderToolResultBody_EmptyAndWhitespaceOnly(t *testing.T) {
	if renderToolResultBody("") != "" {
		t.Error("empty input should produce empty output")
	}
	if renderToolResultBody("   \n   \n") != "" {
		t.Error("whitespace-only input should produce empty output")
	}
}

// Long turn = many tool_use/tool_result entries in a row.
// Previously capped at 60 entries, which silently dropped the start
// of any turn longer than that. Now bumped to 5000; the test
// constructs a long turn and asserts the EARLIEST assistant text
// survives the window.
func TestRenderClaudeRecentTurn_LongTurnSurvivesWindow(t *testing.T) {
	tmp := t.TempDir()
	jsonl := filepath.Join(tmp, "session.jsonl")

	lines := []string{
		mustEntry(t, "user", "do many things"),
		mustEntry(t, "assistant", "FIRST_REPLY_SENTINEL — kicking off a long turn"),
	}
	// 78 tool_use + tool_result pairs follow → 158 entries total
	// in the turn. Old cap of 60 would have lost everything up to
	// entry 98.
	for i := 0; i < 78; i++ {
		useID := fmt.Sprintf("tool_use_%d", i)
		lines = append(lines, assistantWithTool(t, "",
			useID, "Bash", map[string]string{
				"command": fmt.Sprintf("echo step-%d", i),
			}))
		lines = append(lines, userToolResult(t, useID,
			fmt.Sprintf("step-%d done", i)))
	}
	lines = append(lines, mustEntry(t, "assistant",
		"LAST_REPLY_SENTINEL — wrapping up"))

	if err := os.WriteFile(jsonl, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := renderClaudeRecentTurn(jsonl, 5000)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "FIRST_REPLY_SENTINEL") {
		t.Errorf("start of long turn missing — entry window too small")
	}
	if !strings.Contains(got, "LAST_REPLY_SENTINEL") {
		t.Errorf("end of long turn missing")
	}
}

// Multi-line tool_result must render as "  └ first / 4-space rest",
// preserving every line. Previously each result was compressed to
// "  └ line1 · line2 · line3" with a 200-rune cap.
func TestRenderToolResults_MultilineRespectsIndent(t *testing.T) {
	useID := "tool_read"
	summary := map[string]string{useID: "Read(big.go)"}
	long := "Read 500 lines from big.go\n" + strings.Repeat("line content here\n", 60)
	blocks := []claudeContentBlock{{
		Type:       "tool_result",
		ToolUseID:  useID,
		RawContent: mustRawString(t, long),
	}}
	var b strings.Builder
	renderToolResults(&b, blocks, summary)
	out := b.String()
	if !strings.Contains(out, "  └ Read 500 lines from big.go") {
		t.Errorf("first-line bullet missing:\n%s", out)
	}
	if !strings.Contains(out, "\n    line content here") {
		t.Errorf("continuation indent missing — multi-line was flattened or dropped:\n%s", out)
	}
	// All 60 "line content here" lines should be present.
	if got := strings.Count(out, "line content here"); got != 60 {
		t.Errorf("lines dropped: got %d, want 60", got)
	}
}

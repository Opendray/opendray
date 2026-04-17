package jsonl

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDetectAskUserQuestion(t *testing.T) {
	tests := []struct {
		name     string
		blocks   []ContentBlock
		wantQ    string
		wantOK   bool
	}{
		{
			name: "AskUserQuestion with question text",
			blocks: []ContentBlock{
				{Type: "text", Text: "Let me help you."},
				{Type: "tool_use", Name: "AskUserQuestion", Input: &struct {
					Command     string `json:"command,omitempty"`
					Description string `json:"description,omitempty"`
					Content     string `json:"content,omitempty"`
					FilePath    string `json:"file_path,omitempty"`
					Question    string `json:"question,omitempty"`
				}{Question: "What color scheme do you prefer?"}},
			},
			wantQ:  "What color scheme do you prefer?",
			wantOK: true,
		},
		{
			name: "other tool_use — not AskUserQuestion",
			blocks: []ContentBlock{
				{Type: "tool_use", Name: "Bash", Input: &struct {
					Command     string `json:"command,omitempty"`
					Description string `json:"description,omitempty"`
					Content     string `json:"content,omitempty"`
					FilePath    string `json:"file_path,omitempty"`
					Question    string `json:"question,omitempty"`
				}{Command: "ls -la"}},
			},
			wantQ:  "",
			wantOK: false,
		},
		{
			name:   "empty blocks",
			blocks: nil,
			wantQ:  "",
			wantOK: false,
		},
		{
			name: "AskUserQuestion with empty question",
			blocks: []ContentBlock{
				{Type: "tool_use", Name: "AskUserQuestion", Input: &struct {
					Command     string `json:"command,omitempty"`
					Description string `json:"description,omitempty"`
					Content     string `json:"content,omitempty"`
					FilePath    string `json:"file_path,omitempty"`
					Question    string `json:"question,omitempty"`
				}{Question: ""}},
			},
			wantQ:  "",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, ok := detectAskUserQuestion(tt.blocks)
			if ok != tt.wantOK {
				t.Errorf("ok = %v, want %v", ok, tt.wantOK)
			}
			if q != tt.wantQ {
				t.Errorf("question = %q, want %q", q, tt.wantQ)
			}
		})
	}
}

func TestDetectNumberedOptions(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		wantLen int
		wantKey string // first option key
	}{
		{
			name: "clean numbered list after question",
			text: `Which approach would you like?
1. Refactor the existing code
2. Rewrite from scratch
3. Add a compatibility layer`,
			wantLen: 3,
			wantKey: "1",
		},
		{
			name: "numbered list with ) separator",
			text: `Choose one:
1) Option Alpha
2) Option Beta`,
			wantLen: 2,
			wantKey: "1",
		},
		{
			name:    "no question context — rejected",
			text:    "1. First item\n2. Second item\n3. Third item",
			wantLen: 0,
		},
		{
			name:    "single option — too few",
			text:    "Which one?\n1. Only option",
			wantLen: 0,
		},
		{
			name: "non-consecutive numbering — rejected",
			text: `Which?
1. First
3. Third`,
			wantLen: 0,
		},
		{
			name: "code output — not options (no question context)",
			text: `git log:
1. abc123 Initial commit
2. def456 Add feature
3. ghi789 Fix bug`,
			wantLen: 0,
		},
		{
			name: "following keyword — not question context (false positive guard)",
			text: `The following files were modified:
1. src/main.go
2. src/utils.go`,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := detectNumberedOptions(tt.text)
			if len(opts) != tt.wantLen {
				t.Errorf("got %d options, want %d", len(opts), tt.wantLen)
				return
			}
			if tt.wantLen > 0 && opts[0].Key != tt.wantKey {
				t.Errorf("first key = %q, want %q", opts[0].Key, tt.wantKey)
			}
		})
	}
}

func TestDetectLetteredOptions(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		wantLen int
	}{
		{
			name:    "inline (a) (b) (c) pattern",
			text:    "Would you like me to (a) continue, (b) modify the plan, or (c) stop?",
			wantLen: 3,
		},
		{
			name:    "two options",
			text:    "Should I (a) retry or (b) abort?",
			wantLen: 2,
		},
		{
			name:    "non-consecutive — rejected",
			text:    "Choose (a) first or (c) third.",
			wantLen: 0,
		},
		{
			name:    "single letter — too few",
			text:    "Do (a) the thing.",
			wantLen: 0,
		},
		{
			name:    "no separator between options",
			text:    "(a) first approach (b) second approach (c) third approach",
			wantLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := detectLetteredOptions(tt.text)
			if len(opts) != tt.wantLen {
				t.Errorf("got %d options, want %d", len(opts), tt.wantLen)
			}
		})
	}
}

func TestDetectBulletOptions(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		wantLen int
	}{
		{
			name: "bullet list after question",
			text: `Here are your options:
- Run the tests
- Skip and continue
- Abort`,
			wantLen: 3,
		},
		{
			name: "bullet with asterisk",
			text: `Which approach?
* Approach A
* Approach B`,
			wantLen: 2,
		},
		{
			name:    "bullets without question — rejected",
			text:    "- item 1\n- item 2\n- item 3",
			wantLen: 0,
		},
		{
			name: "single bullet — too few",
			text: `Choose:
- Only one`,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := detectBulletOptions(tt.text)
			if len(opts) != tt.wantLen {
				t.Errorf("got %d options, want %d", len(opts), tt.wantLen)
			}
		})
	}
}

func TestDetectPlanApproval(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{"shall I proceed?", "Here's the plan.\n\nShall I proceed?", true},
		{"should I start?", "Ready.\n\nShould I start?", true},
		{"do you want me to proceed", "Do you want me to proceed with this plan?", true},
		{"want me to go ahead", "Want me to go ahead?", true},
		{"ready to implement", "I have a plan. Ready to implement?", true},
		{"normal question", "What do you think?", false},
		{"no question at all", "Here is the output.", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectPlanApproval(tt.text)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectDefaultValue(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		wantDef string
		wantOK  bool
	}{
		{"bracket default", "Enter the directory [./src]:", "./src", true},
		{"file name default", "File name [index.ts]:", "index.ts", true},
		{"port default", "Port number [8080]:", "8080", true},
		{"no default", "Enter your name:", "", false},
		{"question mark — not default", "Are you sure?", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def, ok := detectDefaultValue(tt.text)
			if ok != tt.wantOK {
				t.Errorf("ok = %v, want %v", ok, tt.wantOK)
			}
			if def != tt.wantDef {
				t.Errorf("default = %q, want %q", def, tt.wantDef)
			}
		})
	}
}

func TestDetectEnterContinue(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{"press enter", "Press Enter to continue...", true},
		{"hit enter", "Hit Enter when ready.", true},
		{"enter to continue", "Ready. Enter to continue.", true},
		{"normal text", "This is just regular output.", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectEnterContinue(tt.text)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsOpenEndedQuestion(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{"which file", "Which file should I modify?", true},
		{"what color", "What color scheme do you prefer?", true},
		{"how many", "How many retries should I set?", true},
		{"where should", "Where should I put the config?", true},
		{"X or Y", "Should I use tabs or spaces?", true},
		{"do you want", "Do you want me to fix this?", false},
		{"is this correct", "Is this correct?", false},
		{"should I continue", "Should I continue?", false},
		{"can I proceed", "Can I proceed?", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isOpenEndedQuestion(tt.text)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAnalyzePrompt_Priority(t *testing.T) {
	tests := []struct {
		name     string
		blocks   []ContentBlock
		text     string
		wantKind PromptKind
	}{
		{
			name: "AskUserQuestion takes priority over text patterns",
			blocks: []ContentBlock{
				{Type: "tool_use", Name: "AskUserQuestion", Input: &struct {
					Command     string `json:"command,omitempty"`
					Description string `json:"description,omitempty"`
					Content     string `json:"content,omitempty"`
					FilePath    string `json:"file_path,omitempty"`
					Question    string `json:"question,omitempty"`
				}{Question: "Which file?"}},
			},
			text:     "Some text with [y/N]",
			wantKind: PromptAskUser,
		},
		{
			name:     "[y/N/a] takes priority over numbered list",
			blocks:   nil,
			text:     "1. Option\n2. Option\nAllow? [y/N/a]",
			wantKind: PromptYesNoAlways,
		},
		{
			name:     "[y/N] detected",
			blocks:   nil,
			text:     "Do you want to proceed? [y/N]",
			wantKind: PromptYesNo,
		},
		{
			name:     "no prompt detected",
			blocks:   nil,
			text:     "Just some regular output text.",
			wantKind: PromptNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := analyzePrompt(tt.blocks, tt.text)
			if got.Kind != tt.wantKind {
				t.Errorf("kind = %v, want %v", got.Kind, tt.wantKind)
			}
		})
	}
}

func TestAnalyzePrompt_MultiSelectUpgrade(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		wantKind PromptKind
		wantOpts int
	}{
		{
			name:     "select all that apply",
			text:     "Which features do you need? Select all that apply.\n1. Auth\n2. Billing\n3. Notifications",
			wantKind: PromptMultiSelect,
			wantOpts: 3,
		},
		{
			name:     "choose one or more",
			text:     "Choose one or more languages:\n1. Go\n2. Rust\n3. Python",
			wantKind: PromptMultiSelect,
			wantOpts: 3,
		},
		{
			name:     "CJK multi-select hint in english header",
			text:     "Which features (可多选)?\n1. A\n2. B\n3. C",
			wantKind: PromptMultiSelect,
			wantOpts: 3,
		},
		{
			name:     "plain numbered list stays single-pick",
			text:     "Which approach?\n1. Refactor\n2. Rewrite\n3. Layer",
			wantKind: PromptNumberedList,
			wantOpts: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := analyzePrompt(nil, tt.text)
			if got.Kind != tt.wantKind {
				t.Errorf("kind = %v, want %v", got.Kind, tt.wantKind)
			}
			if len(got.Options) != tt.wantOpts {
				t.Errorf("options = %d, want %d", len(got.Options), tt.wantOpts)
			}
		})
	}
}

func TestLastResponseDetail_JSONL(t *testing.T) {
	// Create a temp JSONL file with an AskUserQuestion.
	dir := t.TempDir()
	jsonlPath := filepath.Join(dir, "test-session.jsonl")

	entries := []map[string]interface{}{
		{
			"type": "assistant",
			"message": map[string]interface{}{
				"role": "assistant",
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": "I need to know your preference.",
					},
					{
						"type":  "tool_use",
						"name":  "AskUserQuestion",
						"input": map[string]string{"question": "Which theme do you want?"},
					},
				},
			},
		},
	}

	f, err := os.Create(jsonlPath)
	if err != nil {
		t.Fatal(err)
	}
	enc := json.NewEncoder(f)
	for _, e := range entries {
		if err := enc.Encode(e); err != nil {
			t.Fatal(err)
		}
	}
	f.Close()

	text, prompt, err := LastResponseDetail(jsonlPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text == "" {
		t.Fatal("expected non-empty text")
	}
	if prompt.Kind != PromptAskUser {
		t.Errorf("kind = %v, want PromptAskUser", prompt.Kind)
	}
	if prompt.Question != "Which theme do you want?" {
		t.Errorf("question = %q, want %q", prompt.Question, "Which theme do you want?")
	}
}

func TestLastResponseDetail_NumberedList(t *testing.T) {
	dir := t.TempDir()
	jsonlPath := filepath.Join(dir, "test-session.jsonl")

	entries := []map[string]interface{}{
		{
			"type": "assistant",
			"message": map[string]interface{}{
				"role": "assistant",
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": "Which approach would you like?\n1. Refactor existing code\n2. Rewrite from scratch\n3. Add compatibility layer",
					},
				},
			},
		},
	}

	f, err := os.Create(jsonlPath)
	if err != nil {
		t.Fatal(err)
	}
	enc := json.NewEncoder(f)
	for _, e := range entries {
		if err := enc.Encode(e); err != nil {
			t.Fatal(err)
		}
	}
	f.Close()

	text, prompt, err := LastResponseDetail(jsonlPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text == "" {
		t.Fatal("expected non-empty text")
	}
	if prompt.Kind != PromptNumberedList {
		t.Errorf("kind = %v, want PromptNumberedList", prompt.Kind)
	}
	if len(prompt.Options) != 3 {
		t.Errorf("options count = %d, want 3", len(prompt.Options))
	}
}

func TestLastResponse_BackwardCompat(t *testing.T) {
	dir := t.TempDir()
	jsonlPath := filepath.Join(dir, "test-session.jsonl")

	entries := []map[string]interface{}{
		{
			"type": "assistant",
			"message": map[string]interface{}{
				"role": "assistant",
				"content": []map[string]interface{}{
					{"type": "text", "text": "Do you want to continue? [y/N]"},
				},
			},
		},
	}

	f, err := os.Create(jsonlPath)
	if err != nil {
		t.Fatal(err)
	}
	enc := json.NewEncoder(f)
	for _, e := range entries {
		if err := enc.Encode(e); err != nil {
			t.Fatal(err)
		}
	}
	f.Close()

	text, isQuestion, err := LastResponse(jsonlPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text == "" {
		t.Fatal("expected non-empty text")
	}
	if !isQuestion {
		t.Error("expected isQuestion = true for [y/N] prompt")
	}
}

func TestResolveLatestJSONL_PicksNewestAcrossBases(t *testing.T) {
	// Simulate two Claude config roots (multi-account setup) each with a
	// project dir that matches the session's cwd.
	root := t.TempDir()
	baseA := filepath.Join(root, "accountA")
	baseB := filepath.Join(root, "accountB")
	cwd := "/work/proj"

	// Encoded project-dir name — matchesProjectDir only needs each cwd part
	// to appear in order; dash-separated mirrors Claude's real encoding.
	encoded := "-work-proj"

	for _, base := range []string{baseA, baseB} {
		dir := filepath.Join(base, "projects", encoded)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	olderPath := filepath.Join(baseA, "projects", encoded, "old.jsonl")
	newerPath := filepath.Join(baseB, "projects", encoded, "new.jsonl")
	mustWrite(t, olderPath, `{"type":"assistant"}`+"\n")
	mustWrite(t, newerPath, `{"type":"assistant"}`+"\n")

	// Explicitly backdate the older file so mtime comparison is deterministic.
	past := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(olderPath, past, past); err != nil {
		t.Fatal(err)
	}

	got := ResolveLatestJSONL([]string{baseA, baseB}, cwd)
	if got != newerPath {
		t.Errorf("ResolveLatestJSONL = %q, want %q", got, newerPath)
	}
}

func TestResolveLatestJSONL_NoMatch(t *testing.T) {
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, "projects"), 0o755); err != nil {
		t.Fatal(err)
	}
	got := ResolveLatestJSONL([]string{base}, "/nonexistent/cwd")
	if got != "" {
		t.Errorf("ResolveLatestJSONL = %q, want empty string", got)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()
}

func TestTailNonEmpty(t *testing.T) {
	text := "line1\n\nline2\n\n\nline3\n"
	got := tailNonEmpty(text, 5)
	if len(got) != 3 {
		t.Errorf("got %d lines, want 3", len(got))
	}
	if got[0] != "line1" || got[1] != "line2" || got[2] != "line3" {
		t.Errorf("unexpected lines: %v", got)
	}
}

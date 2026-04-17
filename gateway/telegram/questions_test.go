package telegram

import (
	"strings"
	"testing"

	"github.com/linivek/ntc/gateway/telegram/jsonl"
)

func TestDetectPrompt_Priority(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		wantKind jsonl.PromptKind
	}{
		// y/N/a highest textual priority
		{"yna", "Allow this? [y/N/a]", jsonl.PromptYesNoAlways},
		{"yn bracket", "Continue? [y/N]", jsonl.PromptYesNo},
		{"yn paren", "Continue? (y/n)", jsonl.PromptYesNo},

		// Numbered list
		{"numbered", "Which approach?\n1. Refactor\n2. Rewrite\n3. Layer", jsonl.PromptNumberedList},

		// Lettered
		{"lettered", "Would you like to (a) continue, (b) modify, or (c) stop?", jsonl.PromptLetteredList},

		// Bullet list
		{"bullets", "Choose one:\n- Alpha\n- Beta\n- Gamma", jsonl.PromptBulletList},

		// Allow pattern
		{"allow", "Allow this tool to execute?", jsonl.PromptYesNo},

		// Plan approval
		{"plan approval", "Here is the plan.\n\nShall I proceed?", jsonl.PromptPlanApproval},
		{"should I implement", "Ready.\n\nShould I implement this?", jsonl.PromptPlanApproval},

		// Default value
		{"default value", "Enter the directory [./src]:", jsonl.PromptDefaultValue},
		{"port default", "Port [8080]:", jsonl.PromptDefaultValue},

		// Enter to continue
		{"press enter", "Press Enter to continue...", jsonl.PromptEnterContinue},
		{"hit enter", "Hit Enter when ready.", jsonl.PromptEnterContinue},

		// Open-ended question (no Yes/No buttons)
		{"which file", "Which file should I modify?", jsonl.PromptGenericQuestion},
		{"what color", "What color do you prefer?", jsonl.PromptGenericQuestion},
		{"X or Y", "Should I use tabs or spaces?", jsonl.PromptGenericQuestion},

		// Closed question → PromptYesNo
		{"do you want", "Do you want me to fix this?", jsonl.PromptYesNo},
		{"is correct", "Is this correct?", jsonl.PromptYesNo},

		// No prompt
		{"no prompt", "Here is the output.\nAll done.", jsonl.PromptNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectPrompt(tt.text)
			if got.Kind != tt.wantKind {
				t.Errorf("DetectPrompt(%q) kind = %v, want %v", tt.text, got.Kind, tt.wantKind)
			}
		})
	}
}

func TestDetectPrompt_NumberedListDetails(t *testing.T) {
	text := "Which approach?\n1. Refactor the code\n2. Rewrite from scratch\n3. Add a layer"
	prompt := DetectPrompt(text)
	if prompt.Kind != jsonl.PromptNumberedList {
		t.Fatalf("kind = %v, want PromptNumberedList", prompt.Kind)
	}
	if len(prompt.Options) != 3 {
		t.Fatalf("options = %d, want 3", len(prompt.Options))
	}
	if prompt.Options[0].Key != "1" || prompt.Options[0].Label != "Refactor the code" {
		t.Errorf("opt[0] = %+v", prompt.Options[0])
	}
	if prompt.Options[2].Key != "3" || prompt.Options[2].Label != "Add a layer" {
		t.Errorf("opt[2] = %+v", prompt.Options[2])
	}
}

func TestDetectPrompt_DefaultValueContent(t *testing.T) {
	prompt := DetectPrompt("Enter name [myproject]:")
	if prompt.Kind != jsonl.PromptDefaultValue {
		t.Fatalf("kind = %v, want PromptDefaultValue", prompt.Kind)
	}
	if prompt.DefaultValue != "myproject" {
		t.Errorf("default = %q, want %q", prompt.DefaultValue, "myproject")
	}
}

func TestKeyboardFromPrompt_YesNo(t *testing.T) {
	kb := KeyboardFromPrompt(jsonl.PromptInfo{Kind: jsonl.PromptYesNo}, "sess-123")
	if kb == nil {
		t.Fatal("expected non-nil keyboard")
	}
	// First row: Yes, No. Second row: Screen.
	if len(kb.InlineKeyboard) != 2 {
		t.Fatalf("rows = %d, want 2", len(kb.InlineKeyboard))
	}
	if len(kb.InlineKeyboard[0]) != 2 {
		t.Errorf("row0 buttons = %d, want 2", len(kb.InlineKeyboard[0]))
	}
	if kb.InlineKeyboard[0][0].Text != "✅ Yes" {
		t.Errorf("button0 text = %q", kb.InlineKeyboard[0][0].Text)
	}
}

func TestKeyboardFromPrompt_YesNoAlways(t *testing.T) {
	kb := KeyboardFromPrompt(jsonl.PromptInfo{Kind: jsonl.PromptYesNoAlways}, "sess-123")
	if kb == nil {
		t.Fatal("expected non-nil keyboard")
	}
	if len(kb.InlineKeyboard[0]) != 3 {
		t.Errorf("row0 buttons = %d, want 3 (yes/no/always)", len(kb.InlineKeyboard[0]))
	}
}

func TestKeyboardFromPrompt_NumberedList(t *testing.T) {
	prompt := jsonl.PromptInfo{
		Kind: jsonl.PromptNumberedList,
		Options: []jsonl.PromptOption{
			{Key: "1", Label: "Refactor"},
			{Key: "2", Label: "Rewrite"},
			{Key: "3", Label: "Add layer"},
		},
	}
	kb := KeyboardFromPrompt(prompt, "sess-abc")
	if kb == nil {
		t.Fatal("expected non-nil keyboard")
	}
	// 3 option buttons in 1 row + Screen row = 2 rows
	if len(kb.InlineKeyboard) != 2 {
		t.Fatalf("rows = %d, want 2", len(kb.InlineKeyboard))
	}
	// First row should have the 3 options
	if len(kb.InlineKeyboard[0]) != 3 {
		t.Errorf("row0 buttons = %d, want 3", len(kb.InlineKeyboard[0]))
	}
	// Verify callback data format
	cb := kb.InlineKeyboard[0][0].CallbackData
	if cb != "send:sess-abc:1\r" {
		t.Errorf("callback = %q, want %q", cb, "send:sess-abc:1\r")
	}
}

func TestKeyboardFromPrompt_DefaultValue(t *testing.T) {
	prompt := jsonl.PromptInfo{Kind: jsonl.PromptDefaultValue, DefaultValue: "./src"}
	kb := KeyboardFromPrompt(prompt, "sess-123")
	if kb == nil {
		t.Fatal("expected non-nil keyboard")
	}
	// Should have "Accept: ./src" button
	found := false
	for _, row := range kb.InlineKeyboard {
		for _, btn := range row {
			if strings.Contains(btn.Text, "Accept") {
				found = true
				if btn.CallbackData != "send:sess-123:\r" {
					t.Errorf("accept callback = %q, want Enter", btn.CallbackData)
				}
			}
		}
	}
	if !found {
		t.Error("no Accept button found")
	}
}

func TestKeyboardFromPrompt_EnterContinue(t *testing.T) {
	prompt := jsonl.PromptInfo{Kind: jsonl.PromptEnterContinue}
	kb := KeyboardFromPrompt(prompt, "sess-123")
	if kb == nil {
		t.Fatal("expected non-nil keyboard")
	}
	if kb.InlineKeyboard[0][0].Text != "Enter ↵" {
		t.Errorf("button text = %q", kb.InlineKeyboard[0][0].Text)
	}
}

func TestKeyboardFromPrompt_AskUser_ReturnsNil(t *testing.T) {
	prompt := jsonl.PromptInfo{Kind: jsonl.PromptAskUser, Question: "What do you think?"}
	kb := KeyboardFromPrompt(prompt, "sess-123")
	if kb != nil {
		t.Error("expected nil keyboard for AskUser (free-text)")
	}
}

func TestKeyboardFromPrompt_GenericQuestion_ReturnsNil(t *testing.T) {
	prompt := jsonl.PromptInfo{Kind: jsonl.PromptGenericQuestion}
	kb := KeyboardFromPrompt(prompt, "sess-123")
	if kb != nil {
		t.Error("expected nil keyboard for GenericQuestion (open-ended)")
	}
}

func TestKeyboardFromPrompt_None_ReturnsNil(t *testing.T) {
	prompt := jsonl.PromptInfo{}
	kb := KeyboardFromPrompt(prompt, "sess-123")
	if kb != nil {
		t.Error("expected nil keyboard for PromptNone")
	}
}

func TestCallbackDataLength(t *testing.T) {
	// UUID session IDs are 36 chars. Verify no callback data exceeds 64 bytes.
	sessionID := "12345678-1234-1234-1234-123456789012" // 36 chars

	prompts := []jsonl.PromptInfo{
		{Kind: jsonl.PromptYesNo},
		{Kind: jsonl.PromptYesNoAlways},
		{Kind: jsonl.PromptPlanApproval},
		{Kind: jsonl.PromptDefaultValue, DefaultValue: "some-default"},
		{Kind: jsonl.PromptEnterContinue},
		{Kind: jsonl.PromptNumberedList, Options: []jsonl.PromptOption{
			{Key: "1", Label: "A"}, {Key: "2", Label: "B"}, {Key: "3", Label: "C"},
			{Key: "4", Label: "D"}, {Key: "5", Label: "E"}, {Key: "6", Label: "F"},
			{Key: "7", Label: "G"}, {Key: "8", Label: "H"}, {Key: "9", Label: "I"},
		}},
		{Kind: jsonl.PromptLetteredList, Options: []jsonl.PromptOption{
			{Key: "a", Label: "A"}, {Key: "b", Label: "B"}, {Key: "c", Label: "C"},
		}},
		{Kind: jsonl.PromptBulletList, Options: []jsonl.PromptOption{
			{Key: "1", Label: "X"}, {Key: "2", Label: "Y"},
		}},
	}

	for _, p := range prompts {
		kb := KeyboardFromPrompt(p, sessionID)
		if kb == nil {
			continue
		}
		for _, row := range kb.InlineKeyboard {
			for _, btn := range row {
				if len(btn.CallbackData) > 64 {
					t.Errorf("callback data too long (%d bytes): %q (prompt kind %v)",
						len(btn.CallbackData), btn.CallbackData, p.Kind)
				}
			}
		}
	}
}

func TestTruncateLabel(t *testing.T) {
	tests := []struct {
		input    string
		max      int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly ten", 11, "exactly ten"},
		{"this is a very long label that needs truncation", 20, "this is a very lo..."},
	}

	for _, tt := range tests {
		got := truncateLabel(tt.input, tt.max)
		if got != tt.expected {
			t.Errorf("truncateLabel(%q, %d) = %q, want %q", tt.input, tt.max, got, tt.expected)
		}
	}
}

func TestDetectQuestion_BackwardCompat(t *testing.T) {
	// Verify the old DetectQuestion function still works correctly
	// for all previously handled patterns.
	tests := []struct {
		name       string
		text       string
		wantNil    bool
		wantButton string // first button text (approximate check)
	}{
		{"yna", "Allow this? [y/N/a]", false, "✅ Yes"},
		{"yn", "Continue? [y/N]", false, "✅ Yes"},
		{"allow", "Allow tool to run?", false, "✅ Allow"},
		{"no question", "Just output.", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kb := DetectQuestion(tt.text, "test-session")
			if tt.wantNil {
				if kb != nil {
					t.Error("expected nil keyboard")
				}
				return
			}
			if kb == nil {
				t.Fatal("expected non-nil keyboard")
			}
			if len(kb.InlineKeyboard) == 0 || len(kb.InlineKeyboard[0]) == 0 {
				t.Fatal("expected at least one button")
			}
			if kb.InlineKeyboard[0][0].Text != tt.wantButton {
				t.Errorf("first button = %q, want %q",
					kb.InlineKeyboard[0][0].Text, tt.wantButton)
			}
		})
	}
}

func TestCallbackLabel(t *testing.T) {
	tests := []struct {
		payload string
		want    string
	}{
		{"y\r", "Yes"},
		{"n\r", "No"},
		{"a\r", "Always"},
		{"\x03", "Ctrl+C"},
		{"\r", "Enter"},
		{"1\r", "Option 1"},
		{"5\r", "Option 5"},
		{"a\r", "Always"}, // 'a' with \r is Always (special case)
		{"b\r", "Option (b)"},
		{"hello\r", "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := callbackLabel(tt.payload)
			if got != tt.want {
				t.Errorf("callbackLabel(%q) = %q, want %q", tt.payload, got, tt.want)
			}
		})
	}
}

func TestMakeOptionKeyboard_Layout(t *testing.T) {
	// 6 short options → should be 2 rows of 3
	opts := []jsonl.PromptOption{
		{Key: "1", Label: "Alpha"},
		{Key: "2", Label: "Beta"},
		{Key: "3", Label: "Gamma"},
		{Key: "4", Label: "Delta"},
		{Key: "5", Label: "Epsilon"},
		{Key: "6", Label: "Zeta"},
	}
	kb := makeOptionKeyboard("sess", opts)
	if kb == nil {
		t.Fatal("expected non-nil keyboard")
	}
	// 2 option rows + Screen = 3 rows
	if len(kb.InlineKeyboard) != 3 {
		t.Errorf("rows = %d, want 3", len(kb.InlineKeyboard))
	}
	if len(kb.InlineKeyboard[0]) != 3 {
		t.Errorf("row0 = %d buttons, want 3", len(kb.InlineKeyboard[0]))
	}
	if len(kb.InlineKeyboard[1]) != 3 {
		t.Errorf("row1 = %d buttons, want 3", len(kb.InlineKeyboard[1]))
	}
}

func TestMakeOptionKeyboard_LongLabels(t *testing.T) {
	// Long labels → should be 1 per row
	opts := []jsonl.PromptOption{
		{Key: "1", Label: "This is a very long option label that exceeds twenty five runes"},
		{Key: "2", Label: "Another equally long option label for testing purposes"},
	}
	kb := makeOptionKeyboard("sess", opts)
	if kb == nil {
		t.Fatal("expected non-nil keyboard")
	}
	// 2 option rows (1 each) + Screen = 3 rows
	if len(kb.InlineKeyboard) != 3 {
		t.Errorf("rows = %d, want 3", len(kb.InlineKeyboard))
	}
	if len(kb.InlineKeyboard[0]) != 1 {
		t.Errorf("row0 = %d buttons, want 1 (long label → 1 per row)", len(kb.InlineKeyboard[0]))
	}
}

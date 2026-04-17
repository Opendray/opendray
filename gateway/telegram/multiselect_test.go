package telegram

import (
	"testing"
	"time"

	"github.com/opendray/opendray/gateway/telegram/jsonl"
)

func testOptions() []jsonl.PromptOption {
	return []jsonl.PromptOption{
		{Key: "1", Label: "Alpha"},
		{Key: "2", Label: "Beta"},
		{Key: "3", Label: "Gamma"},
	}
}

func TestMultiSelectStore_CreateAndGet(t *testing.T) {
	s := NewMultiSelectStore(time.Hour)
	s.Create(100, 42, "sess-1", testOptions())

	st := s.Get(100, 42)
	if st == nil {
		t.Fatal("expected state, got nil")
	}
	if st.SessionID != "sess-1" {
		t.Errorf("sessionID = %q, want %q", st.SessionID, "sess-1")
	}
	if len(st.Options) != 3 {
		t.Errorf("options = %d, want 3", len(st.Options))
	}
	if len(st.Checked) != 0 {
		t.Errorf("checked = %d, want empty", len(st.Checked))
	}
}

func TestMultiSelectStore_Get_Missing(t *testing.T) {
	s := NewMultiSelectStore(time.Hour)
	if st := s.Get(1, 2); st != nil {
		t.Errorf("expected nil for missing key, got %+v", st)
	}
}

func TestMultiSelectStore_Toggle(t *testing.T) {
	s := NewMultiSelectStore(time.Hour)
	s.Create(100, 42, "sess-1", testOptions())

	// First toggle of "2" → checked
	st := s.Toggle(100, 42, "2")
	if st == nil {
		t.Fatal("toggle returned nil")
	}
	if !st.Checked["2"] {
		t.Errorf("checked[2] should be true")
	}

	// Second toggle of "2" → unchecked (removed from map)
	st = s.Toggle(100, 42, "2")
	if st == nil {
		t.Fatal("toggle returned nil on second call")
	}
	if st.Checked["2"] {
		t.Errorf("checked[2] should be false after re-toggle")
	}

	// Toggle an unknown key → nil
	if st := s.Toggle(100, 42, "zzz"); st != nil {
		t.Errorf("expected nil for unknown key, got %+v", st)
	}
}

func TestMultiSelectStore_Get_ReturnsSnapshot(t *testing.T) {
	s := NewMultiSelectStore(time.Hour)
	s.Create(100, 42, "sess-1", testOptions())
	s.Toggle(100, 42, "1")

	st := s.Get(100, 42)
	if st == nil {
		t.Fatal("expected state")
	}
	// Mutating the returned map must not affect the store.
	st.Checked["1"] = false
	st.Checked["2"] = true

	fresh := s.Get(100, 42)
	if !fresh.Checked["1"] {
		t.Errorf("store mutated through Get snapshot: checked[1] should still be true")
	}
	if fresh.Checked["2"] {
		t.Errorf("store mutated through Get snapshot: checked[2] should be false")
	}
}

func TestMultiSelectStore_Submit_DrainsState(t *testing.T) {
	s := NewMultiSelectStore(time.Hour)
	s.Create(100, 42, "sess-1", testOptions())
	s.Toggle(100, 42, "1")
	s.Toggle(100, 42, "3")

	st := s.Submit(100, 42)
	if st == nil {
		t.Fatal("submit returned nil")
	}
	if !st.Checked["1"] || !st.Checked["3"] {
		t.Errorf("lost picks: %+v", st.Checked)
	}

	// State is drained — re-submit returns nil.
	if again := s.Submit(100, 42); again != nil {
		t.Errorf("expected nil on second submit, got %+v", again)
	}
	// Toggle after submit is also nil.
	if st := s.Toggle(100, 42, "1"); st != nil {
		t.Errorf("expected nil toggle after submit, got %+v", st)
	}
}

func TestMultiSelectStore_TTLExpiry(t *testing.T) {
	s := NewMultiSelectStore(50 * time.Millisecond)
	s.Create(100, 42, "sess-1", testOptions())

	// Immediately accessible.
	if st := s.Get(100, 42); st == nil {
		t.Fatal("state should exist immediately")
	}

	time.Sleep(80 * time.Millisecond)

	if st := s.Get(100, 42); st != nil {
		t.Errorf("expected nil after TTL, got %+v", st)
	}
}

func TestMultiSelectStore_MultiRound(t *testing.T) {
	// Two concurrent prompts for the same session under different
	// messageIDs must have fully independent state.
	s := NewMultiSelectStore(time.Hour)
	s.Create(100, 42, "sess-1", testOptions())
	s.Create(100, 99, "sess-1", testOptions())

	s.Toggle(100, 42, "1")
	s.Toggle(100, 99, "3")

	a := s.Get(100, 42)
	b := s.Get(100, 99)
	if !a.Checked["1"] || a.Checked["3"] {
		t.Errorf("msg42 state bled: %+v", a.Checked)
	}
	if !b.Checked["3"] || b.Checked["1"] {
		t.Errorf("msg99 state bled: %+v", b.Checked)
	}

	// Submitting one leaves the other intact.
	s.Submit(100, 42)
	if s.Get(100, 42) != nil {
		t.Error("msg42 should be drained")
	}
	if s.Get(100, 99) == nil {
		t.Error("msg99 should still exist")
	}
}

func TestMakeMultiSelectKeyboard_RendersCheckboxes(t *testing.T) {
	opts := testOptions()
	checked := map[string]bool{"2": true}

	kb := makeMultiSelectKeyboard("sess-abc", opts, checked)
	if kb == nil {
		t.Fatal("expected non-nil keyboard")
	}

	var found1, found2, found3 bool
	for _, row := range kb.InlineKeyboard {
		for _, b := range row {
			switch b.CallbackData {
			case "multi_toggle:sess-abc:1":
				found1 = true
				if !containsRune(b.Text, '☐') {
					t.Errorf("key 1 should be unchecked (☐): %q", b.Text)
				}
			case "multi_toggle:sess-abc:2":
				found2 = true
				if !containsRune(b.Text, '☑') {
					t.Errorf("key 2 should be checked (☑): %q", b.Text)
				}
			case "multi_toggle:sess-abc:3":
				found3 = true
				if !containsRune(b.Text, '☐') {
					t.Errorf("key 3 should be unchecked (☐): %q", b.Text)
				}
			}
		}
	}
	if !found1 || !found2 || !found3 {
		t.Errorf("missing toggle buttons: 1=%v 2=%v 3=%v", found1, found2, found3)
	}

	// Last row should have Submit + Screen.
	last := kb.InlineKeyboard[len(kb.InlineKeyboard)-1]
	if len(last) != 2 {
		t.Fatalf("last row has %d buttons, want 2 (submit+screen)", len(last))
	}
	if last[0].CallbackData != "multi_submit:sess-abc" {
		t.Errorf("submit callback = %q", last[0].CallbackData)
	}
	if last[1].CallbackData != "screen:sess-abc" {
		t.Errorf("screen callback = %q", last[1].CallbackData)
	}
}

func TestMakeMultiSelectKeyboard_CallbackLength(t *testing.T) {
	// UUID-style sessionID (36 chars) must keep all callback data under 64 bytes.
	sessionID := "12345678-1234-1234-1234-123456789012"
	opts := []jsonl.PromptOption{
		{Key: "1", Label: "A"}, {Key: "2", Label: "B"}, {Key: "3", Label: "C"},
		{Key: "4", Label: "D"}, {Key: "5", Label: "E"}, {Key: "6", Label: "F"},
		{Key: "7", Label: "G"}, {Key: "8", Label: "H"}, {Key: "9", Label: "I"},
	}
	kb := makeMultiSelectKeyboard(sessionID, opts, nil)
	if kb == nil {
		t.Fatal("expected keyboard")
	}
	for _, row := range kb.InlineKeyboard {
		for _, b := range row {
			if len(b.CallbackData) > 64 {
				t.Errorf("callback too long (%d): %q", len(b.CallbackData), b.CallbackData)
			}
		}
	}
}

func TestMakeMultiSelectKeyboard_EmptyOptions(t *testing.T) {
	if kb := makeMultiSelectKeyboard("sess", nil, nil); kb != nil {
		t.Error("expected nil for empty options")
	}
}

func TestKeyboardFromPrompt_MultiSelect(t *testing.T) {
	prompt := jsonl.PromptInfo{
		Kind:    jsonl.PromptMultiSelect,
		Options: testOptions(),
	}
	kb := KeyboardFromPrompt(prompt, "sess-xyz")
	if kb == nil {
		t.Fatal("expected keyboard for PromptMultiSelect")
	}
	// Last row → submit + screen.
	last := kb.InlineKeyboard[len(kb.InlineKeyboard)-1]
	if last[0].CallbackData != "multi_submit:sess-xyz" {
		t.Errorf("submit callback = %q", last[0].CallbackData)
	}
}

func containsRune(s string, r rune) bool {
	for _, c := range s {
		if c == r {
			return true
		}
	}
	return false
}

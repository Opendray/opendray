package roundtable

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// truncate must slice on a rune boundary. A byte slice (s[:max]) can sever a
// multi-byte UTF-8 character (Chinese, emoji) in half; the resulting invalid
// UTF-8 makes codex reject its stdin prompt ("input is not valid UTF-8").
func TestTruncate_RuneBoundary(t *testing.T) {
	// 10 Chinese chars = 30 bytes. Cutting at byte 8 (mid-character with a
	// naive s[:max]) would produce invalid UTF-8.
	s := strings.Repeat("你", 10)
	got := truncate(s, 8)
	if !utf8.ValidString(got) {
		t.Fatalf("truncate produced invalid UTF-8: %q", got)
	}
	if r := []rune(strings.TrimSuffix(got, "…")); len(r) != 8 {
		t.Errorf("expected 8 runes kept, got %d (%q)", len(r), got)
	}

	// ASCII-only strings shorter than max pass through untouched.
	if truncate("hello", 100) != "hello" {
		t.Errorf("short string must be returned unchanged")
	}
}

func TestChatSystemPrompt_FramingAndPersona(t *testing.T) {
	rt := RoundTable{
		Topic:   "auth redesign",
		Framing: "Topic: session tokens. claude leads, codex hunts security holes.",
		Seats:   []Seat{{Provider: "claude"}, {Provider: "codex", Persona: "only hunt security holes"}},
	}

	// Framing appears in every member's prompt.
	claudePrompt := chatSystemPrompt(rt, rt.Seats[0], false)
	if !strings.Contains(claudePrompt, "session tokens") {
		t.Errorf("framing must be injected for all members; got:\n%s", claudePrompt)
	}

	// A seat's persona is injected as a directive lens for that seat.
	codexPrompt := chatSystemPrompt(rt, rt.Seats[1], false)
	if !strings.Contains(codexPrompt, "only hunt security holes") {
		t.Errorf("persona must be injected for the seat; got:\n%s", codexPrompt)
	}
	// claude (no persona) should NOT carry codex's persona text.
	if strings.Contains(claudePrompt, "only hunt security holes") {
		t.Errorf("a seat's persona must not leak into another seat's prompt")
	}

	// No framing → no framing header.
	noFraming := chatSystemPrompt(RoundTable{Seats: rt.Seats}, rt.Seats[0], false)
	if strings.Contains(noFraming, "DISCUSSION FRAMING") {
		t.Errorf("empty framing must not emit a framing header")
	}
}

// The tool-use instruction flips with hasMemoryTools: off tells members not to
// use tools; on tells them they have read-only opendray-memory to ground claims.
func TestChatSystemPrompt_MemoryToolsInstruction(t *testing.T) {
	rt := RoundTable{Topic: "kb", Seats: []Seat{{Provider: "claude"}}}

	off := chatSystemPrompt(rt, rt.Seats[0], false)
	if !strings.Contains(off, "Do NOT use tools") {
		t.Errorf("tool-less member must be told not to use tools; got:\n%s", off)
	}
	if strings.Contains(off, "memory_search") {
		t.Errorf("tool-less member prompt must not advertise memory tools")
	}

	on := chatSystemPrompt(rt, rt.Seats[0], true)
	if !strings.Contains(on, "memory_search") || !strings.Contains(on, "read-only") {
		t.Errorf("tool-enabled member must be told about the read-only memory tools; got:\n%s", on)
	}
	if strings.Contains(on, "Do NOT use tools") {
		t.Errorf("tool-enabled member must not carry the no-tools instruction")
	}
}

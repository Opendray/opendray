package summarizer

import (
	"strings"
	"testing"
	"time"
)

func TestSystemPrompt_HasKeyDirectives(t *testing.T) {
	p := SystemPrompt()
	required := []string{
		"DURABLE facts",
		"preference",
		"identifier",
		"decision",
		"task",
		"STRICT JSON only",
		"Empty facts array is valid",
	}
	for _, want := range required {
		if !strings.Contains(p, want) {
			t.Errorf("system prompt missing required directive: %q", want)
		}
	}
}

func TestFactsToolSchema_HasRequiredFields(t *testing.T) {
	s := FactsToolJSONSchema
	for _, want := range []string{`"facts"`, `"text"`, `"category"`, `"confidence"`} {
		if !strings.Contains(s, want) {
			t.Errorf("tool schema missing field: %s", want)
		}
	}
}

func TestMessagesToTranscriptText(t *testing.T) {
	now := time.Now()
	msgs := []Message{
		{Role: RoleUser, Text: "I prefer pnpm.", Timestamp: now},
		{Role: RoleAssistant, Text: "Noted.", Timestamp: now},
		{Role: RoleSystem, Text: "should be dropped", Timestamp: now},
		{Role: RoleUser, Text: "Also: dev DB is db.example.com.", Timestamp: now},
	}
	got := MessagesToTranscriptText(msgs)
	want := "USER: I prefer pnpm.\nASSISTANT: Noted.\nUSER: Also: dev DB is db.example.com."
	if got != want {
		t.Errorf("transcript flatten mismatch:\n got: %q\nwant: %q", got, want)
	}
}

func TestMessagesToTranscriptText_Empty(t *testing.T) {
	if got := MessagesToTranscriptText(nil); got != "" {
		t.Errorf("empty msgs should yield empty string, got %q", got)
	}
}

func TestMessagesToTranscriptText_OnlySystem(t *testing.T) {
	msgs := []Message{{Role: RoleSystem, Text: "ignored"}}
	if got := MessagesToTranscriptText(msgs); got != "" {
		t.Errorf("only-system msgs should yield empty string, got %q", got)
	}
}

func TestTruncateRaw(t *testing.T) {
	short := "short"
	if got := TruncateRaw(short); got != short {
		t.Errorf("short string should pass through, got len %d", len(got))
	}
	long := strings.Repeat("x", 10*1024)
	got := TruncateRaw(long)
	if len(got) != 4*1024 {
		t.Errorf("long string truncate len = %d, want %d", len(got), 4*1024)
	}
}

package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func mustParseGeminiTime(t *testing.T) time.Time { return time.Now().UTC() }

// TestGeminiChatRecentResponse verifies the newer chats/*.jsonl reader:
// it returns the latest type:"gemini" content, skips metadata ($set)
// and user lines, and prefers the most-recently-modified session file.
func TestGeminiChatRecentResponse(t *testing.T) {
	dir := t.TempDir()
	chats := filepath.Join(dir, "chats")
	if err := os.MkdirAll(chats, 0o755); err != nil {
		t.Fatal(err)
	}
	older := filepath.Join(chats, "session-2026-05-24T00-00-aaaa.jsonl")
	newer := filepath.Join(chats, "session-2026-05-24T07-50-bbbb.jsonl")
	if err := os.WriteFile(older, []byte(
		`{"type":"user","content":"old prompt"}`+"\n"+
			`{"type":"gemini","content":"STALE reply from an earlier session"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newer, []byte(
		`{"type":"user","content":"ping?"}`+"\n"+
			`{"type":"gemini","content":"pong! ready to help."}`+"\n"+
			`{"$set":{"lastUpdated":"2026-05-24T07:53:42.997Z"}}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Make `newer` the most-recently-modified file.
	now := mustParseGeminiTime(t)
	_ = os.Chtimes(older, now.Add(-time.Hour), now.Add(-time.Hour))
	_ = os.Chtimes(newer, now, now)

	got := geminiChatRecentResponse(dir)
	if got != "pong! ready to help." {
		t.Errorf("got %q, want the latest reply from the newest session file", got)
	}

	// No chats dir → empty (caller falls back to logs.json).
	if r := geminiChatRecentResponse(t.TempDir()); r != "" {
		t.Errorf("missing chats dir should yield empty, got %q", r)
	}
}

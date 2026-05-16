package session

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// writeGeminiLogs writes a logs.json under the SHA-256-hashed
// project dir for `cwd`. Returns the file path so individual tests
// can mutate it if needed.
func writeGeminiLogs(t *testing.T, cwd string, entries []map[string]any) string {
	t.Helper()
	home := os.Getenv("HOME")
	hash := sha256.Sum256([]byte(cwd))
	dir := filepath.Join(home, ".gemini", "tmp", hex.EncodeToString(hash[:]))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(entries)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "logs.json")
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestGeminiRecentResponse_PicksLatestModelReply(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cwd := "/tmp/proj/gemini-test"
	writeGeminiLogs(t, cwd, []map[string]any{
		{
			"sessionId": "sess-1",
			"messageId": 0,
			"type":      "user",
			"message":   "first prompt",
			"timestamp": "2026-05-04T10:00:00.000Z",
		},
		{
			"sessionId": "sess-1",
			"messageId": 1,
			"type":      "model",
			"message":   "earlier reply that should NOT win",
			"timestamp": "2026-05-04T10:00:01.000Z",
		},
		{
			"sessionId": "sess-1",
			"messageId": 2,
			"type":      "user",
			"message":   "follow-up",
			"timestamp": "2026-05-04T10:01:00.000Z",
		},
		{
			"sessionId": "sess-1",
			"messageId": 3,
			"type":      "model",
			"message":   "MOST_RECENT_REPLY_SENTINEL",
			"timestamp": "2026-05-04T10:01:30.000Z",
		},
	})

	got := geminiRecentResponse(cwd)
	if got != "MOST_RECENT_REPLY_SENTINEL" {
		t.Errorf("expected newest model reply, got %q", got)
	}
}

// File order isn't trusted — Gemini appends as messages arrive but
// the contract is timestamp-based. The resolver compares timestamps
// across all entries.
func TestGeminiRecentResponse_TimestampWinsOverFileOrder(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cwd := "/tmp/proj/gemini-out-of-order"
	writeGeminiLogs(t, cwd, []map[string]any{
		// File-order LATER but timestamp EARLIER:
		{
			"sessionId": "sess-1",
			"messageId": 10,
			"type":      "model",
			"message":   "EARLIER_BY_TIMESTAMP",
			"timestamp": "2026-05-04T09:00:00.000Z",
		},
		// File-order EARLIER but timestamp LATER:
		{
			"sessionId": "sess-1",
			"messageId": 1,
			"type":      "model",
			"message":   "LATER_BY_TIMESTAMP",
			"timestamp": "2026-05-04T12:00:00.000Z",
		},
	})

	got := geminiRecentResponse(cwd)
	if got != "LATER_BY_TIMESTAMP" {
		t.Errorf("timestamp ordering broken; got %q", got)
	}
}

func TestGeminiRecentResponse_IgnoresUserPrompts(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cwd := "/tmp/proj/gemini-no-model-yet"
	writeGeminiLogs(t, cwd, []map[string]any{
		{
			"sessionId": "sess-1",
			"messageId": 0,
			"type":      "user",
			"message":   "this should not appear",
			"timestamp": "2026-05-04T10:00:00.000Z",
		},
		{
			"sessionId": "sess-1",
			"messageId": 1,
			"type":      "user",
			"message":   "neither should this",
			"timestamp": "2026-05-04T10:00:01.000Z",
		},
	})

	if got := geminiRecentResponse(cwd); got != "" {
		t.Errorf("expected empty when no model reply, got %q", got)
	}
}

func TestGeminiRecentResponse_IgnoresEmptyMessages(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cwd := "/tmp/proj/gemini-empty-model"
	writeGeminiLogs(t, cwd, []map[string]any{
		{
			"sessionId": "sess-1",
			"messageId": 0,
			"type":      "model",
			"message":   "",
			"timestamp": "2026-05-04T10:00:00.000Z",
		},
		{
			"sessionId": "sess-1",
			"messageId": 1,
			"type":      "model",
			"message":   "   ",
			"timestamp": "2026-05-04T10:00:01.000Z",
		},
		{
			"sessionId": "sess-1",
			"messageId": 2,
			"type":      "model",
			"message":   "real content here",
			"timestamp": "2026-05-04T10:00:02.000Z",
		},
	})

	if got := geminiRecentResponse(cwd); got != "real content here" {
		t.Errorf("expected non-empty model message; got %q", got)
	}
}

func TestGeminiRecentResponse_NoTranscriptReturnsEmpty(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if got := geminiRecentResponse("/tmp/proj/never-touched"); got != "" {
		t.Errorf("expected empty for missing transcript, got %q", got)
	}
}

func TestGeminiRecentResponse_NoHomeReturnsEmpty(t *testing.T) {
	t.Setenv("HOME", "")
	if got := geminiRecentResponse("/anything"); got != "" {
		t.Errorf("expected empty when HOME unset, got %q", got)
	}
}

func TestGeminiRecentResponse_MalformedJSONReturnsEmpty(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	cwd := "/tmp/proj/malformed"
	hash := sha256.Sum256([]byte(cwd))
	dir := filepath.Join(tmpHome, ".gemini", "tmp", hex.EncodeToString(hash[:]))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "logs.json"),
		[]byte("not valid json {"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := geminiRecentResponse(cwd); got != "" {
		t.Errorf("expected empty on malformed json, got %q", got)
	}
}

func TestGeminiRecentResponse_TrimsWhitespace(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cwd := "/tmp/proj/whitespace"
	writeGeminiLogs(t, cwd, []map[string]any{
		{
			"sessionId": "sess-1",
			"messageId": 0,
			"type":      "model",
			"message":   "\n  the actual reply  \n\n",
			"timestamp": "2026-05-04T10:00:00.000Z",
		},
	})
	if got := geminiRecentResponse(cwd); got != "the actual reply" {
		t.Errorf("whitespace not trimmed; got %q", got)
	}
}

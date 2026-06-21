package agyacct

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeConv(t *testing.T, home, id string, withWAL bool) {
	t.Helper()
	dir := filepath.Join(home, agyConversationsRel)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, id+".db"), []byte("DBDATA-"+id), 0o600); err != nil {
		t.Fatal(err)
	}
	if withWAL {
		if err := os.WriteFile(filepath.Join(dir, id+".db-wal"), []byte("WAL"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func writeLastConv(t *testing.T, home string, m map[string]string) {
	t.Helper()
	p := filepath.Join(home, agyLastConvRel)
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(m)
	if err := os.WriteFile(p, body, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestConversationIDForCwd(t *testing.T) {
	home := t.TempDir()
	if got := conversationIDForCwd(home, "/work"); got != "" {
		t.Errorf("no map yet → want empty, got %q", got)
	}
	writeLastConv(t, home, map[string]string{"/work": "conv-123", "/other": "conv-999"})
	if got := conversationIDForCwd(home, "/work"); got != "conv-123" {
		t.Errorf("got %q, want conv-123", got)
	}
	if got := conversationIDForCwd(home, "/unmapped"); got != "" {
		t.Errorf("unmapped cwd → want empty, got %q", got)
	}
	if got := conversationIDForCwd("", "/work"); got != "" {
		t.Errorf("empty home → want empty, got %q", got)
	}
}

func TestCopyConversation(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	writeConv(t, src, "conv-abc", true) // .db + .db-wal

	if err := copyConversation(src, dst, "conv-abc", "/work"); err != nil {
		t.Fatalf("copy: %v", err)
	}
	// db + wal landed in dst
	for _, suf := range []string{".db", ".db-wal"} {
		if _, err := os.Stat(filepath.Join(dst, agyConversationsRel, "conv-abc"+suf)); err != nil {
			t.Errorf("expected %s in dst: %v", suf, err)
		}
	}
	// dst now resumes that conversation for the cwd
	if got := conversationIDForCwd(dst, "/work"); got != "conv-abc" {
		t.Errorf("dst last_conversations not set; got %q", got)
	}
	// content preserved
	b, _ := os.ReadFile(filepath.Join(dst, agyConversationsRel, "conv-abc.db"))
	if string(b) != "DBDATA-conv-abc" {
		t.Errorf("db content mismatch: %q", b)
	}
}

func TestCopyConversation_Edges(t *testing.T) {
	home := t.TempDir()
	// same src==dst → no-op, no error
	if err := copyConversation(home, home, "x", "/w"); err != nil {
		t.Errorf("same home should be no-op, got %v", err)
	}
	// missing conversation → error
	if err := copyConversation(t.TempDir(), t.TempDir(), "missing", "/w"); err == nil {
		t.Error("missing conversation should error")
	}
	// missing args → error
	if err := copyConversation("", "x", "y", "/w"); err == nil {
		t.Error("missing src should error")
	}
}

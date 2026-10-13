package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeTranscript(t *testing.T, cfg, workDir, id, body string, mod time.Time) {
	t.Helper()
	dir := filepath.Join(cfg, "projects", claudeWorkspaceName(workDir))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, id+".jsonl")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(p, mod, mod); err != nil {
		t.Fatal(err)
	}
}

func TestClaudeWorkspaceName(t *testing.T) {
	if got := claudeWorkspaceName("/var/lib/opendray/rcc/musica"); got != "-var-lib-opendray-rcc-musica" {
		t.Fatalf("got %q", got)
	}
	if got := claudeWorkspaceName("/a/b.c_d"); got != "-a-b-c-d" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveClaudeResumeID(t *testing.T) {
	const wd = "/work/app"
	t0 := time.Now().Add(-time.Hour)

	t.Run("stored transcript exists and is newest", func(t *testing.T) {
		cfg := t.TempDir()
		writeTranscript(t, cfg, wd, "aaa", `{"session_id":"aaa"}`+"\n", t0)
		if got := resolveClaudeResumeID(cfg, wd, "aaa"); got != "aaa" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("stored run id lives in another transcript (no own file)", func(t *testing.T) {
		// The vocaler.ai case: launched as cd63…, records appended to 4fd0….
		cfg := t.TempDir()
		writeTranscript(t, cfg, wd, "4fd0", `{"sessionId":"4fd0","session_id":"cd63"}`+"\n", t0)
		if got := resolveClaudeResumeID(cfg, wd, "cd63"); got != "4fd0" {
			t.Fatalf("got %q, want 4fd0", got)
		}
	})

	t.Run("after /clear the newer transcript wins over the stored file", func(t *testing.T) {
		cfg := t.TempDir()
		writeTranscript(t, cfg, wd, "old", `{"session_id":"old"}`+"\n", t0)
		writeTranscript(t, cfg, wd, "new", `{"sessionId":"new","session_id":"old"}`+"\n", t0.Add(time.Minute))
		if got := resolveClaudeResumeID(cfg, wd, "old"); got != "new" {
			t.Fatalf("got %q, want new", got)
		}
	})

	t.Run("another session's newer transcript is ignored", func(t *testing.T) {
		cfg := t.TempDir()
		writeTranscript(t, cfg, wd, "mine", `{"session_id":"mine"}`+"\n", t0)
		writeTranscript(t, cfg, wd, "theirs", `{"session_id":"theirs"}`+"\n", t0.Add(time.Minute))
		if got := resolveClaudeResumeID(cfg, wd, "mine"); got != "mine" {
			t.Fatalf("got %q, want mine", got)
		}
	})

	t.Run("stored transcript under a different workspace", func(t *testing.T) {
		cfg := t.TempDir()
		writeTranscript(t, cfg, "/elsewhere", "aaa", `{"session_id":"aaa"}`+"\n", t0)
		if got := resolveClaudeResumeID(cfg, wd, "aaa"); got != "aaa" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("no transcript anywhere → empty (fresh start)", func(t *testing.T) {
		cfg := t.TempDir()
		writeTranscript(t, cfg, wd, "other", `{"session_id":"other"}`+"\n", t0)
		if got := resolveClaudeResumeID(cfg, wd, "gone"); got != "" {
			t.Fatalf("got %q, want empty", got)
		}
	})

	t.Run("symlinked transcript is not trusted", func(t *testing.T) {
		cfg := t.TempDir()
		dir := filepath.Join(cfg, "projects", claudeWorkspaceName(wd))
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(t.TempDir(), "secret")
		if err := os.WriteFile(target, []byte(`"session_id":"lnk"`), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, filepath.Join(dir, "lnk.jsonl")); err != nil {
			t.Fatal(err)
		}
		if got := resolveClaudeResumeID(cfg, wd, "lnk"); got != "" {
			t.Fatalf("got %q, want empty", got)
		}
	})

	t.Run("nothing to check against → unchanged", func(t *testing.T) {
		if got := resolveClaudeResumeID("", wd, "aaa"); got != "aaa" {
			t.Fatalf("got %q", got)
		}
		if got := resolveClaudeResumeID(t.TempDir(), wd, ""); got != "" {
			t.Fatalf("got %q", got)
		}
	})
}

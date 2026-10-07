package session

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const testThread = "019deded-aaaa-7bbb-8ccc-0123456789ab"

// stageRollout writes <home>/sessions/2026/10/07/rollout-<ts>-<id>.jsonl.
func stageRollout(t *testing.T, home, id, body string) string {
	t.Helper()
	dir := filepath.Join(home, "sessions", "2026", "10", "07")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "rollout-2026-10-07T20-21-55-"+id+".jsonl")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestFindCodexRollout(t *testing.T) {
	home := t.TempDir()
	if got := FindCodexRollout(home, testThread); got != "" {
		t.Fatalf("empty home: got %q", got)
	}
	p := stageRollout(t, home, testThread, "x\n")
	if got := FindCodexRollout(home, testThread); got != p {
		t.Errorf("got %q, want %q", got, p)
	}
	if got := FindCodexRollout(home, "../etc"); got != "" {
		t.Errorf("traversal id must not match, got %q", got)
	}
	if got := FindCodexRollout(home, "other-id"); got != "" {
		t.Errorf("unrelated id matched %q", got)
	}
}

func TestMigrateCodexRollout_CopiesAtSameRelativePath(t *testing.T) {
	oldHome, newHome := t.TempDir(), t.TempDir()
	stageRollout(t, oldHome, testThread, "{\"type\":\"session_meta\"}\nHELLO\n")

	ok, err := MigrateCodexRollout(oldHome, newHome, testThread)
	if err != nil || !ok {
		t.Fatalf("migrate = (%v, %v), want (true, nil)", ok, err)
	}
	dst := filepath.Join(newHome, "sessions", "2026", "10", "07", "rollout-2026-10-07T20-21-55-"+testThread+".jsonl")
	body, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("destination missing: %v", err)
	}
	if !strings.Contains(string(body), "HELLO") {
		t.Errorf("content = %q", body)
	}
	if st, _ := os.Stat(dst); st.Mode().Perm() != 0o600 {
		t.Errorf("perm = %v, want 0600", st.Mode().Perm())
	}
}

func TestMigrateCodexRollout_MissingSourceNotResumable(t *testing.T) {
	ok, err := MigrateCodexRollout(t.TempDir(), t.TempDir(), testThread)
	if err != nil || ok {
		t.Errorf("missing source = (%v, %v), want (false, nil)", ok, err)
	}
	if ok, _ := MigrateCodexRollout(t.TempDir(), t.TempDir(), ""); ok {
		t.Error("empty thread id must not be resumable")
	}
}

func TestMigrateCodexRollout_SameHomeIsResumableNoOp(t *testing.T) {
	home := t.TempDir()
	stageRollout(t, home, testThread, "x\n")
	if ok, err := MigrateCodexRollout(home, home, testThread); err != nil || !ok {
		t.Errorf("same home = (%v, %v), want (true, nil)", ok, err)
	}
}

func TestMigrateCodexRollout_ReplacesStaleDest(t *testing.T) {
	oldHome, newHome := t.TempDir(), t.TempDir()
	stageRollout(t, oldHome, testThread, "LATEST\n")
	dst := stageRollout(t, newHome, testThread, "STALE\n")
	if ok, err := MigrateCodexRollout(oldHome, newHome, testThread); err != nil || !ok {
		t.Fatalf("migrate = (%v, %v)", ok, err)
	}
	if body, _ := os.ReadFile(dst); string(body) != "LATEST\n" {
		t.Errorf("stale destination should be replaced, got %q", body)
	}
}

func TestMigrateCodexRollout_RejectsSymlinkedSource(t *testing.T) {
	tmp := t.TempDir()
	oldHome, newHome := filepath.Join(tmp, "old"), filepath.Join(tmp, "new")
	victim := filepath.Join(tmp, "secret.txt")
	if err := os.WriteFile(victim, []byte("SECRET"), 0o600); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(oldHome, "sessions", "2026", "10", "07")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(victim, filepath.Join(dir, "rollout-x-"+testThread+".jsonl")); err != nil {
		if runtime.GOOS == "windows" {
			t.Skip(err.Error())
		}
		t.Fatal(err)
	}
	// A symlinked rollout is never even found (WalkDir sees a symlink, not
	// a regular file), so the switch falls back to a fresh conversation.
	ok, err := MigrateCodexRollout(oldHome, newHome, testThread)
	if ok || err != nil {
		t.Fatalf("symlinked source = (%v, %v), want (false, nil)", ok, err)
	}
	if _, err := os.Stat(filepath.Join(newHome, "sessions")); !os.IsNotExist(err) {
		t.Errorf("nothing should be written for a symlinked source; stat err=%v", err)
	}
	// And the copy primitive refuses one outright.
	if err := copyRegularFileAtomic(filepath.Join(dir, "rollout-x-"+testThread+".jsonl"), filepath.Join(tmp, "out")); err == nil {
		t.Error("copyRegularFileAtomic must refuse a symlink")
	}
}

func TestSyncCodexHome_CopiesRolloutsBackAndReportsThread(t *testing.T) {
	scratch, cred := t.TempDir(), t.TempDir()
	older := stageRollout(t, scratch, "019d0000-0000-7000-8000-000000000001", "old\n")
	past := time.Now().Add(-time.Hour)
	_ = os.Chtimes(older, past, past)
	stageRollout(t, scratch, testThread, "new\n")

	got := SyncCodexHome(scratch, cred)
	if got != testThread {
		t.Errorf("thread id = %q, want newest %q", got, testThread)
	}
	if FindCodexRollout(cred, testThread) == "" || FindCodexRollout(cred, "019d0000-0000-7000-8000-000000000001") == "" {
		t.Error("every scratch rollout should be synced into the credential home")
	}
}

func TestSyncCodexHome_AppendedRolloutOverwritesOlderCopy(t *testing.T) {
	scratch, cred := t.TempDir(), t.TempDir()
	dst := stageRollout(t, cred, testThread, "turn1\n")
	past := time.Now().Add(-time.Hour)
	_ = os.Chtimes(dst, past, past)
	stageRollout(t, scratch, testThread, "turn1\nturn2\n")

	SyncCodexHome(scratch, cred)
	if body, _ := os.ReadFile(dst); string(body) != "turn1\nturn2\n" {
		t.Errorf("credential copy = %q, want appended conversation", body)
	}
}

func TestSyncCodexHome_NoRolloutsNoThread(t *testing.T) {
	if got := SyncCodexHome(t.TempDir(), t.TempDir()); got != "" {
		t.Errorf("got %q, want empty", got)
	}
	home := t.TempDir()
	if got := SyncCodexHome(home, home); got != "" {
		t.Errorf("same dir must be a no-op, got %q", got)
	}
}

func TestCodexSessionsRoots_IncludesAccountHomes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".codex-accounts", "work"), 0o700); err != nil {
		t.Fatal(err)
	}
	got := codexSessionsRoots(CodexHistoryConfig{})
	want := []string{
		filepath.Join(home, ".codex", "sessions"),
		filepath.Join(home, ".codex-accounts", "work", "sessions"),
	}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("roots = %v, want %v", got, want)
	}
	if got := codexSessionsRoots(CodexHistoryConfig{SessionsRoot: "/x"}); len(got) != 1 || got[0] != "/x" {
		t.Errorf("explicit SessionsRoot must win, got %v", got)
	}
}

func TestSyncCodexHome_AuthWriteBack(t *testing.T) {
	scratch, cred := t.TempDir(), t.TempDir()
	credAuth := filepath.Join(cred, "auth.json")
	scratchAuth := filepath.Join(scratch, "auth.json")

	// Untouched login (identical bytes) is never rewritten.
	_ = os.WriteFile(credAuth, []byte(`{"v":1}`), 0o600)
	past := time.Now().Add(-time.Hour)
	_ = os.Chtimes(credAuth, past, past)
	_ = os.WriteFile(scratchAuth, []byte(`{"v":1}`), 0o600)
	SyncCodexHome(scratch, cred)
	if st, _ := os.Stat(credAuth); st.ModTime().After(past.Add(time.Second)) {
		t.Error("identical auth.json must not be rewritten")
	}

	// Refreshed during the run (differs, newer) → written back.
	_ = os.WriteFile(scratchAuth, []byte(`{"v":2}`), 0o600)
	SyncCodexHome(scratch, cred)
	if body, _ := os.ReadFile(credAuth); string(body) != `{"v":2}` {
		t.Errorf("refreshed auth.json should be written back, got %s", body)
	}

	// Credential home refreshed by another session after ours copied it
	// (credential newer) → left alone.
	_ = os.WriteFile(scratchAuth, []byte(`{"v":3}`), 0o600)
	_ = os.Chtimes(scratchAuth, past, past)
	_ = os.WriteFile(credAuth, []byte(`{"v":4}`), 0o600)
	SyncCodexHome(scratch, cred)
	if body, _ := os.ReadFile(credAuth); string(body) != `{"v":4}` {
		t.Errorf("newer credential login must win, got %s", body)
	}
}

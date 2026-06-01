package session

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// stage builds a fake claude config-dir layout with a transcript at
// <root>/projects/<workspace>/<id>.jsonl and returns the root path.
func stageTranscript(t *testing.T, root, workspace, id, body string) string {
	t.Helper()
	dir := filepath.Join(root, "projects", workspace)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, id+".jsonl")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestMigrateClaudeTranscript_HardLinksWhenPossible(t *testing.T) {
	tmp := t.TempDir()
	oldCfg := filepath.Join(tmp, "old")
	newCfg := filepath.Join(tmp, "new")
	id := "abc-123"
	stageTranscript(t, oldCfg, "-var-lib-test", id, `{"role":"user","content":"hi"}`+"\n")

	if err := migrateClaudeTranscript(oldCfg, newCfg, id); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(newCfg, "projects", "-var-lib-test", id+".jsonl")
	body, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("destination not created: %v", err)
	}
	if !strings.Contains(string(body), `"hi"`) {
		t.Errorf("destination contents wrong: %s", string(body))
	}
	// Both files should share inode (hard-link). On the same tempdir
	// fs this is always possible.
	src := filepath.Join(oldCfg, "projects", "-var-lib-test", id+".jsonl")
	stSrc, _ := os.Stat(src)
	stDst, _ := os.Stat(dst)
	if !os.SameFile(stSrc, stDst) {
		t.Error("expected hard-link (same inode), got distinct files")
	}
}

func TestMigrateClaudeTranscript_NoSourceIsNotError(t *testing.T) {
	tmp := t.TempDir()
	// No transcript ever staged.
	if err := migrateClaudeTranscript(
		filepath.Join(tmp, "old"),
		filepath.Join(tmp, "new"),
		"never-existed",
	); err != nil {
		t.Errorf("missing source must not error; got %v", err)
	}
}

func TestMigrateClaudeTranscript_SameDirIsNoOp(t *testing.T) {
	tmp := t.TempDir()
	stageTranscript(t, tmp, "ws", "id1", "x")
	if err := migrateClaudeTranscript(tmp, tmp, "id1"); err != nil {
		t.Errorf("same-dir migration must be a no-op; got %v", err)
	}
}

func TestMigrateClaudeTranscript_EmptySessionIDIsNoOp(t *testing.T) {
	tmp := t.TempDir()
	if err := migrateClaudeTranscript(
		filepath.Join(tmp, "old"),
		filepath.Join(tmp, "new"),
		"",
	); err != nil {
		t.Errorf("empty session id must be a no-op; got %v", err)
	}
}

func TestMigrateClaudeTranscript_LeavesExistingDestUntouched(t *testing.T) {
	// If the destination already holds a transcript for this session
	// id (e.g. the operator switched back-and-forth previously), we
	// must NOT overwrite it — the existing file might have updates
	// from the other account that aren't in the source.
	tmp := t.TempDir()
	oldCfg := filepath.Join(tmp, "old")
	newCfg := filepath.Join(tmp, "new")
	id := "id-x"
	stageTranscript(t, oldCfg, "ws", id, "OLD\n")
	stageTranscript(t, newCfg, "ws", id, "EXISTING\n")

	if err := migrateClaudeTranscript(oldCfg, newCfg, id); err != nil {
		t.Fatal(err)
	}
	body, _ := os.ReadFile(filepath.Join(newCfg, "projects", "ws", id+".jsonl"))
	if string(body) != "EXISTING\n" {
		t.Errorf("destination should be untouched, got %q", body)
	}
}

func TestMigrateClaudeTranscript_PicksNewestSource(t *testing.T) {
	// Pathological: same id exists under two workspaces (shouldn't
	// really happen, but glob can match multiple). We pick the
	// newest-mtime so the most recently-touched conversation wins.
	tmp := t.TempDir()
	oldCfg := filepath.Join(tmp, "old")
	newCfg := filepath.Join(tmp, "new")
	id := "id-y"
	older := stageTranscript(t, oldCfg, "ws-older", id, "OLD\n")
	newer := stageTranscript(t, oldCfg, "ws-newer", id, "NEW\n")
	// Force distinct mtimes.
	old := mustStat(t, older).ModTime().Add(-1)
	_ = os.Chtimes(older, old, old)
	now := mustStat(t, newer).ModTime()
	_ = os.Chtimes(newer, now, now)

	if err := migrateClaudeTranscript(oldCfg, newCfg, id); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(newCfg, "projects", "ws-newer", id+".jsonl")
	body, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("expected newer workspace to win, got %v", err)
	}
	if string(body) != "NEW\n" {
		t.Errorf("expected newer file content, got %q", body)
	}
}

func TestMigrateClaudeTranscript_RejectsSymlinkedSource(t *testing.T) {
	// A symlinked source must NOT be migrated — see the security
	// rationale comment in migrateClaudeTranscript. os.Link to a
	// symlink would create a same-symlink hardlink whose target the
	// new account would then read as conversation history.
	tmp := t.TempDir()
	oldCfg := filepath.Join(tmp, "old")
	newCfg := filepath.Join(tmp, "new")
	id := "sym-id"
	// Stage a real "victim" file outside the projects tree.
	victim := filepath.Join(tmp, "victim-secret.txt")
	if err := os.WriteFile(victim, []byte("SECRET"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Plant a symlink where the legitimate transcript would live.
	workspace := "-var-lib-test"
	dir := filepath.Join(oldCfg, "projects", workspace)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(dir, id+".jsonl")
	if err := os.Symlink(victim, src); err != nil {
		if runtime.GOOS == "windows" {
			t.Skip("symlink creation requires elevated privileges on Windows: " + err.Error())
		}
		t.Fatal(err)
	}

	err := migrateClaudeTranscript(oldCfg, newCfg, id)
	if err == nil {
		t.Fatal("expected migration to refuse a symlinked source")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Errorf("expected error to mention symlink; got %v", err)
	}
	// And the destination must NOT have been created — neither as a
	// link, a copy, nor an empty file.
	dst := filepath.Join(newCfg, "projects", workspace, id+".jsonl")
	if _, statErr := os.Lstat(dst); !os.IsNotExist(statErr) {
		t.Errorf("destination must not exist after symlink-source refusal; stat err=%v", statErr)
	}
}

func mustStat(t *testing.T, p string) os.FileInfo {
	t.Helper()
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	return fi
}

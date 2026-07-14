package checkpoint

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// captureRepo captures cwd and returns the resulting Checkpoint, so the
// DB-free applyCheckpoint core can be exercised directly.
func captureRepo(t *testing.T, cwd string) Checkpoint {
	t.Helper()
	root := t.TempDir()
	cp, err := Capture(context.Background(), root, CaptureRequest{
		CheckpointID: "chk_r",
		SessionID:    "ses_r",
		Cwd:          cwd,
		Trigger:      TriggerManual,
		Now:          time.Unix(1_700_000_000, 0),
	})
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	return cp
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, dir, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(b)
}

// TestRestoreHappyPath: capture a dirty repo, revert the working tree, then
// restore — the tracked diff is re-applied and untracked files come back.
func TestRestoreHappyPath(t *testing.T) {
	dir := gitRepo(t)
	writeFile(t, dir, "tracked.txt", "CHANGED\n")
	writeFile(t, dir, "new.txt", "fresh\n")
	cp := captureRepo(t, dir)

	// Revert the working tree to the committed state + remove the untracked.
	runGit(t, dir, "checkout", "--", "tracked.txt")
	if err := os.Remove(filepath.Join(dir, "new.txt")); err != nil {
		t.Fatal(err)
	}

	res, err := applyCheckpoint(context.Background(), cp)
	if err != nil {
		t.Fatalf("applyCheckpoint: %v", err)
	}
	if !res.DiffApplied {
		t.Error("expected DiffApplied=true")
	}
	if res.UntrackedRestored != 1 {
		t.Errorf("UntrackedRestored = %d, want 1", res.UntrackedRestored)
	}
	if got := readFile(t, dir, "tracked.txt"); got != "CHANGED\n" {
		t.Errorf("tracked.txt = %q, want CHANGED", got)
	}
	if got := readFile(t, dir, "new.txt"); got != "fresh\n" {
		t.Errorf("new.txt = %q, want fresh", got)
	}
}

// TestRestoreRefusesHeadMismatch: a commit after capture moves HEAD, so
// restore refuses before touching the tree.
func TestRestoreRefusesHeadMismatch(t *testing.T) {
	dir := gitRepo(t)
	writeFile(t, dir, "tracked.txt", "CHANGED\n")
	cp := captureRepo(t, dir)

	runGit(t, dir, "checkout", "--", "tracked.txt")
	writeFile(t, dir, "another.txt", "x\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-q", "-m", "second")

	if _, err := applyCheckpoint(context.Background(), cp); !errors.Is(err, ErrHeadMismatch) {
		t.Fatalf("want ErrHeadMismatch, got %v", err)
	}
}

// TestRestoreRefusesDirtyWorktree: uncommitted tracked changes block restore.
func TestRestoreRefusesDirtyWorktree(t *testing.T) {
	dir := gitRepo(t)
	writeFile(t, dir, "tracked.txt", "CHANGED\n")
	cp := captureRepo(t, dir)
	// Leave the tree dirty (tracked.txt still modified).

	if _, err := applyCheckpoint(context.Background(), cp); !errors.Is(err, ErrDirtyWorktree) {
		t.Fatalf("want ErrDirtyWorktree, got %v", err)
	}
}

// TestRestoreUntrackedNoOverwrite: an existing target untracked file is
// skipped, not clobbered.
func TestRestoreUntrackedNoOverwrite(t *testing.T) {
	dir := gitRepo(t)
	writeFile(t, dir, "new.txt", "fresh\n")
	cp := captureRepo(t, dir)

	// Clean tracked tree; pre-create new.txt with content that must survive.
	writeFile(t, dir, "new.txt", "LOCAL EDIT\n")

	res, err := applyCheckpoint(context.Background(), cp)
	if err != nil {
		t.Fatalf("applyCheckpoint: %v", err)
	}
	if res.UntrackedRestored != 0 {
		t.Errorf("UntrackedRestored = %d, want 0 (target existed)", res.UntrackedRestored)
	}
	if len(res.UntrackedSkipped) != 1 {
		t.Errorf("UntrackedSkipped = %v, want 1 entry", res.UntrackedSkipped)
	}
	if got := readFile(t, dir, "new.txt"); got != "LOCAL EDIT\n" {
		t.Errorf("new.txt was clobbered: %q", got)
	}
}

// TestRestoreNonGit: a metadata-only (non-git) checkpoint has nothing to
// restore.
func TestRestoreNonGit(t *testing.T) {
	dir := t.TempDir()
	cp := captureRepo(t, dir)
	if _, err := applyCheckpoint(context.Background(), cp); !errors.Is(err, ErrNotGitCheckpoint) {
		t.Fatalf("want ErrNotGitCheckpoint, got %v", err)
	}
}

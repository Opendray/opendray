package checkpoint

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// gitRepo initialises a temp git repo with one committed file and returns
// its path. Skips the test if git is unavailable.
func gitRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	runGit(t, dir, "init", "-q")
	runGit(t, dir, "config", "user.email", "t@t.test")
	runGit(t, dir, "config", "user.name", "t")
	runGit(t, dir, "config", "commit.gpgsign", "false")
	if err := os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("original\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-q", "-m", "init")
	return dir
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func capture(t *testing.T, cwd string, input []byte) (string, Checkpoint) {
	t.Helper()
	root := t.TempDir()
	cp, err := Capture(context.Background(), root, CaptureRequest{
		CheckpointID: "chk_test",
		SessionID:    "ses_test",
		Cwd:          cwd,
		Trigger:      TriggerManual,
		InputHistory: input,
		Now:          time.Unix(1_700_000_000, 0),
	})
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	return root, cp
}

// TestCaptureGitDiffAndUntracked is the core happy path: a modified tracked
// file shows up in the diff, an untracked non-ignored file is copied, and a
// .gitignore'd file is NOT copied (secrets/build output stay out).
func TestCaptureGitDiffAndUntracked(t *testing.T) {
	dir := gitRepo(t)
	// Modify a tracked file.
	if err := os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("CHANGED\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// One untracked file that should be captured...
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("brand new\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// ...and one ignored file that must NOT be captured.
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("secret.env\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "secret.env"), []byte("TOKEN=abc\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	root, cp := capture(t, dir, nil)

	if !cp.IsGit {
		t.Fatal("expected IsGit=true")
	}
	if !cp.GitDirty {
		t.Error("expected GitDirty=true")
	}
	if cp.DiffBytes == 0 {
		t.Error("expected a non-empty diff")
	}
	diff, err := os.ReadFile(filepath.Join(root, "chk_test", fileDiff))
	if err != nil {
		t.Fatalf("read diff: %v", err)
	}
	if !strings.Contains(string(diff), "CHANGED") {
		t.Errorf("diff should mention the changed content, got:\n%s", diff)
	}

	// Untracked capture: new.txt + .gitignore are untracked-not-ignored (2),
	// secret.env is ignored (0). So 2 files, and secret.env absent on disk.
	if cp.UntrackedFiles != 2 {
		t.Errorf("UntrackedFiles = %d, want 2 (new.txt + .gitignore)", cp.UntrackedFiles)
	}
	if _, err := os.Stat(filepath.Join(root, "chk_test", dirUntracked, "new.txt")); err != nil {
		t.Errorf("new.txt should have been captured: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "chk_test", dirUntracked, "secret.env")); !os.IsNotExist(err) {
		t.Error("ignored secret.env must NOT be captured")
	}
	// Manifest exists.
	if _, err := os.Stat(filepath.Join(root, "chk_test", fileMeta)); err != nil {
		t.Errorf("manifest missing: %v", err)
	}
}

// TestCaptureNonGit: a non-git cwd records metadata only, with a note, and
// no diff/untracked payload.
func TestCaptureNonGit(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	root, cp := capture(t, dir, nil)
	if cp.IsGit {
		t.Error("non-git cwd should have IsGit=false")
	}
	if cp.DiffBytes != 0 || cp.UntrackedFiles != 0 {
		t.Error("non-git cwd should capture no diff/untracked payload")
	}
	if !strings.Contains(cp.Note, "not a git repository") {
		t.Errorf("expected a not-a-repo note, got %q", cp.Note)
	}
	if _, err := os.Stat(filepath.Join(root, "chk_test", fileDiff)); !os.IsNotExist(err) {
		t.Error("no diff file should be written for a non-git cwd")
	}
}

// TestCaptureInputHistory: the input tail is written and sized.
func TestCaptureInputHistory(t *testing.T) {
	dir := t.TempDir()
	input := []byte("ls -la\ngit status\n")
	root, cp := capture(t, dir, input)
	if cp.InputBytes != int64(len(input)) {
		t.Errorf("InputBytes = %d, want %d", cp.InputBytes, len(input))
	}
	got, err := os.ReadFile(filepath.Join(root, "chk_test", fileInput))
	if err != nil {
		t.Fatalf("read input history: %v", err)
	}
	if string(got) != string(input) {
		t.Errorf("input history = %q, want %q", got, input)
	}
}

// TestCaptureUntrackedSizeCap: an untracked file larger than the per-file
// cap is skipped and the checkpoint is flagged Truncated.
func TestCaptureUntrackedSizeCap(t *testing.T) {
	dir := gitRepo(t)
	big := make([]byte, MaxUntrackedFileBytes+1)
	for i := range big {
		big[i] = 'x'
	}
	if err := os.WriteFile(filepath.Join(dir, "big.bin"), big, 0o644); err != nil {
		t.Fatal(err)
	}
	root, cp := capture(t, dir, nil)
	if cp.UntrackedFiles != 0 {
		t.Errorf("oversized untracked file should be skipped, got UntrackedFiles=%d", cp.UntrackedFiles)
	}
	if !cp.Truncated {
		t.Error("expected Truncated=true when a file is skipped by the size cap")
	}
	if _, err := os.Stat(filepath.Join(root, "chk_test", dirUntracked, "big.bin")); !os.IsNotExist(err) {
		t.Error("oversized file must not be copied")
	}
}

// TestSafeRel guards the path-escape defence.
func TestSafeRel(t *testing.T) {
	ok := []string{"a.txt", "dir/b.txt", "./c.txt"}
	bad := []string{"", "/etc/passwd", "../escape", "../../x", "a/../../b"}
	for _, p := range ok {
		if !safeRel(p) {
			t.Errorf("safeRel(%q) = false, want true", p)
		}
	}
	for _, p := range bad {
		if safeRel(p) {
			t.Errorf("safeRel(%q) = true, want false", p)
		}
	}
}

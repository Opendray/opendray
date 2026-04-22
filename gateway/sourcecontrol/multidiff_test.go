package sourcecontrol

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// newFixtureRepo inits a git repo with one committed file, then lets
// the caller mutate the tree for the scenario under test. Skips the
// test if `git` is unavailable.
func newFixtureRepo(t *testing.T) (repoPath string, cfg Config) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available")
	}
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		// Set minimal env so global git config (user, gpg signing) can't trip us up.
		cmd.Env = append(os.Environ(),
			"GIT_CONFIG_GLOBAL=/dev/null",
			"GIT_CONFIG_SYSTEM=/dev/null",
			"GIT_AUTHOR_NAME=test",
			"GIT_AUTHOR_EMAIL=t@example.com",
			"GIT_COMMITTER_NAME=test",
			"GIT_COMMITTER_EMAIL=t@example.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s / %v", args, string(out), err)
		}
	}
	run("init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-q", "-m", "initial")
	return dir, Config{AllowedRoots: []string{dir}}
}

func TestMultiDiff_UnstagedModificationsAcrossFiles(t *testing.T) {
	dir, cfg := newFixtureRepo(t)

	// Two files modified, one new (untracked is NOT in unstaged diff).
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello\nworld\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(dir, "other.txt")
	if err := os.WriteFile(other, []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Stage the new file so it appears in unstaged-then-modify flow.
	if out, err := exec.Command("git", "-C", dir, "add", "other.txt").CombinedOutput(); err != nil {
		t.Fatalf("git add: %s / %v", string(out), err)
	}
	// Now modify the staged file so it has both staged and unstaged changes.
	if err := os.WriteFile(other, []byte("a\nb\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := MultiDiff(context.Background(), cfg, dir,
		MultiDiffOptions{Mode: ModeUnstaged})
	if err != nil {
		t.Fatalf("MultiDiff: %v", err)
	}
	if got.Mode != ModeUnstaged {
		t.Errorf("mode = %q, want unstaged", got.Mode)
	}
	if len(got.Files) != 2 {
		t.Fatalf("want 2 files, got %d: %+v", len(got.Files), got.Files)
	}

	paths := make([]string, 0, len(got.Files))
	for _, f := range got.Files {
		paths = append(paths, f.Path)
		if f.Patch == "" {
			t.Errorf("%s: empty patch", f.Path)
		}
	}
	sort.Strings(paths)
	want := []string{"README.md", "other.txt"}
	if !reflect.DeepEqual(paths, want) {
		t.Errorf("paths = %v, want %v", paths, want)
	}

	// README.md: pure add (+1, -0).
	for _, f := range got.Files {
		if f.Path == "README.md" {
			if f.Add != 1 || f.Del != 0 {
				t.Errorf("README.md counts = +%d/-%d, want +1/-0", f.Add, f.Del)
			}
			if f.Status != "modified" {
				t.Errorf("README.md status = %q, want modified", f.Status)
			}
		}
	}
}

func TestMultiDiff_StagedShowsOnlyIndexedChanges(t *testing.T) {
	dir, cfg := newFixtureRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "staged.txt"), []byte("s1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", dir, "add", "staged.txt").CombinedOutput(); err != nil {
		t.Fatalf("git add: %s / %v", string(out), err)
	}

	// Also create an unstaged change that must NOT appear in the staged diff.
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello\nw\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := MultiDiff(context.Background(), cfg, dir,
		MultiDiffOptions{Mode: ModeStaged})
	if err != nil {
		t.Fatalf("MultiDiff staged: %v", err)
	}
	if len(got.Files) != 1 {
		t.Fatalf("want 1 staged file, got %d: %+v", len(got.Files), got.Files)
	}
	f := got.Files[0]
	if f.Path != "staged.txt" {
		t.Errorf("path = %q, want staged.txt", f.Path)
	}
	if f.Status != "added" {
		t.Errorf("status = %q, want added", f.Status)
	}
	if f.Add != 1 {
		t.Errorf("add count = %d, want 1", f.Add)
	}
}

func TestMultiDiff_EmptyDiffYieldsEmptyFiles(t *testing.T) {
	dir, cfg := newFixtureRepo(t)

	got, err := MultiDiff(context.Background(), cfg, dir,
		MultiDiffOptions{Mode: ModeUnstaged})
	if err != nil {
		t.Fatalf("MultiDiff: %v", err)
	}
	if len(got.Files) != 0 {
		t.Errorf("clean tree should yield 0 files, got %+v", got.Files)
	}
}

func TestMultiDiff_FullContextExpandsDiff(t *testing.T) {
	dir, cfg := newFixtureRepo(t)
	// Build a file with 20 lines and change the middle one so
	// default (3 context lines) and full context differ visibly.
	lines := make([]byte, 0, 40)
	for i := 0; i < 20; i++ {
		lines = append(lines, byte('a')+byte(i%26), '\n')
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), lines, 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", dir, "add", "README.md").CombinedOutput(); err != nil {
		t.Fatalf("git add: %s / %v", string(out), err)
	}
	if out, err := exec.Command("git",
		"-C", dir,
		"-c", "user.name=t",
		"-c", "user.email=t@example.com",
		"commit", "-q", "-m", "20 lines",
	).CombinedOutput(); err != nil {
		t.Fatalf("git commit: %s / %v", string(out), err)
	}
	// Modify the 10th line only.
	mutated := make([]byte, 0, 40)
	for i := 0; i < 20; i++ {
		if i == 10 {
			mutated = append(mutated, 'Z', '\n')
			continue
		}
		mutated = append(mutated, byte('a')+byte(i%26), '\n')
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), mutated, 0o644); err != nil {
		t.Fatal(err)
	}

	compact, err := MultiDiff(context.Background(), cfg, dir,
		MultiDiffOptions{Mode: ModeUnstaged, Full: false})
	if err != nil {
		t.Fatalf("compact: %v", err)
	}
	full, err := MultiDiff(context.Background(), cfg, dir,
		MultiDiffOptions{Mode: ModeUnstaged, Full: true})
	if err != nil {
		t.Fatalf("full: %v", err)
	}
	if len(compact.Files) != 1 || len(full.Files) != 1 {
		t.Fatalf("expected 1 file each; got compact=%d full=%d", len(compact.Files), len(full.Files))
	}
	if len(full.Files[0].Patch) <= len(compact.Files[0].Patch) {
		t.Errorf("full-context patch should be longer. compact=%d full=%d",
			len(compact.Files[0].Patch), len(full.Files[0].Patch))
	}
}

func TestParseNumstat(t *testing.T) {
	in := "3\t1\tREADME.md\n-\t-\tassets/logo.png\n7\t0\tcmd/main.go\n"
	got := parseNumstat(in)
	if got["README.md"].add != 3 || got["README.md"].del != 1 {
		t.Errorf("README.md counts wrong: %+v", got["README.md"])
	}
	if !got["assets/logo.png"].binary {
		t.Errorf("logo should be binary")
	}
	if got["cmd/main.go"].add != 7 {
		t.Errorf("main.go add wrong: %+v", got["cmd/main.go"])
	}
}

func TestParsePatchBlocks_DetectsStatus(t *testing.T) {
	patch := `diff --git a/new.txt b/new.txt
new file mode 100644
index 0000000..abc
--- /dev/null
+++ b/new.txt
@@ -0,0 +1,1 @@
+hello
diff --git a/gone.txt b/gone.txt
deleted file mode 100644
index abc..0000000
--- a/gone.txt
+++ /dev/null
@@ -1,1 +0,0 @@
-bye
diff --git a/old.txt b/renamed.txt
similarity index 100%
rename from old.txt
rename to renamed.txt
`
	got := parsePatchBlocks(patch)
	if len(got) != 3 {
		t.Fatalf("want 3 files, got %d", len(got))
	}
	byName := make(map[string]FileDiff)
	for _, f := range got {
		byName[f.Path] = f
	}
	if byName["new.txt"].Status != "added" {
		t.Errorf("new.txt status: %+v", byName["new.txt"])
	}
	if byName["gone.txt"].Status != "deleted" {
		t.Errorf("gone.txt status: %+v", byName["gone.txt"])
	}
	if byName["renamed.txt"].Status != "renamed" || byName["renamed.txt"].OldPath != "old.txt" {
		t.Errorf("rename parsing wrong: %+v", byName["renamed.txt"])
	}
}

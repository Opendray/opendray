package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// makeRepo initialises a throwaway git repo with a first commit and
// returns its absolute path.
func makeRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not installed")
	}

	dir := t.TempDir()
	commands := [][]string{
		{"init", "-q", "-b", "main"},
		{"config", "user.email", "test@ntc.local"},
		{"config", "user.name", "NTC Test"},
		{"config", "commit.gpgsign", "false"},
	}
	for _, args := range commands {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	readme := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readme, []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"add", "README.md"},
		{"commit", "-q", "-m", "initial"},
	} {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return dir
	}
	return resolved
}

func baseConfig(repo string) Config {
	return Config{
		AllowedRoots: []string{repo},
		DefaultPath:  repo,
		Timeout:      10 * time.Second,
		AllowCommit:  true,
	}
}

func TestStatusReportsCleanRepo(t *testing.T) {
	repo := makeRepo(t)
	cfg := baseConfig(repo)
	ctx := context.Background()

	res, err := Status(ctx, cfg, repo)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !res.Clean {
		t.Errorf("expected clean repo, got files: %+v", res.Files)
	}
	if res.Branch != "main" {
		t.Errorf("branch=%q, want main", res.Branch)
	}
	if res.Head == "" {
		t.Errorf("HEAD SHA is empty")
	}
}

func TestStatusReportsDirtyFiles(t *testing.T) {
	repo := makeRepo(t)
	cfg := baseConfig(repo)

	if err := os.WriteFile(filepath.Join(repo, "new.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Status(context.Background(), cfg, repo)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if res.Clean {
		t.Fatal("expected dirty repo")
	}
	foundUntracked, foundModified := false, false
	for _, f := range res.Files {
		if f.Path == "new.txt" && f.Untracked {
			foundUntracked = true
		}
		if f.Path == "README.md" && f.Unstaged {
			foundModified = true
		}
	}
	if !foundUntracked || !foundModified {
		t.Errorf("missing expected entries in %+v", res.Files)
	}
}

func TestDiffIncludesUnstagedChanges(t *testing.T) {
	repo := makeRepo(t)
	cfg := baseConfig(repo)
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Diff(context.Background(), cfg, repo, DiffOptions{})
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if !strings.Contains(res.Diff, "-hello") || !strings.Contains(res.Diff, "+changed") {
		t.Errorf("diff missing expected hunks:\n%s", res.Diff)
	}
}

func TestStageUnstageRoundTrip(t *testing.T) {
	repo := makeRepo(t)
	cfg := baseConfig(repo)
	ctx := context.Background()

	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Stage(ctx, cfg, repo, []string{"README.md"}); err != nil {
		t.Fatalf("Stage: %v", err)
	}
	status, _ := Status(ctx, cfg, repo)
	if len(status.Files) != 1 || !status.Files[0].Staged {
		t.Fatalf("expected staged README.md, got %+v", status.Files)
	}

	if err := Unstage(ctx, cfg, repo, []string{"README.md"}); err != nil {
		t.Fatalf("Unstage: %v", err)
	}
	status, _ = Status(ctx, cfg, repo)
	if len(status.Files) != 1 || status.Files[0].Staged {
		t.Fatalf("expected unstaged README.md, got %+v", status.Files)
	}
}

func TestCommitCreatesNewCommit(t *testing.T) {
	repo := makeRepo(t)
	cfg := baseConfig(repo)
	ctx := context.Background()

	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("v2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Stage(ctx, cfg, repo, []string{"README.md"}); err != nil {
		t.Fatalf("Stage: %v", err)
	}

	c, err := CreateCommit(ctx, cfg, repo, "second commit")
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if c.Subject != "second commit" {
		t.Errorf("subject=%q", c.Subject)
	}

	log, err := Log(ctx, cfg, repo, 5)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if len(log) != 2 || log[0].SHA != c.SHA {
		t.Errorf("log mismatch: %+v", log)
	}
}

func TestSecurePathRejectsOutsideRoots(t *testing.T) {
	repo := makeRepo(t)
	cfg := baseConfig(repo)

	outside := t.TempDir()
	if _, err := SecurePath(cfg, outside); err == nil {
		t.Error("expected rejection for path outside allowed roots")
	}
}

func TestReadOnlyRejectsMutations(t *testing.T) {
	repo := makeRepo(t)
	cfg := baseConfig(repo)
	cfg.AllowCommit = false

	if err := Stage(context.Background(), cfg, repo, []string{"README.md"}); err == nil {
		t.Error("expected Stage to be rejected in read-only mode")
	}
}

func TestSessionDiffFiltersToChangesSinceBaseline(t *testing.T) {
	repo := makeRepo(t)
	cfg := baseConfig(repo)
	ctx := context.Background()
	mgr := NewManager()

	baseline, err := mgr.Snapshot(ctx, cfg, "session-1", repo)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if baseline.HeadSHA == "" {
		t.Fatal("baseline HEAD is empty")
	}

	// First commit happens "inside" the session.
	if err := os.WriteFile(filepath.Join(repo, "added.go"), []byte("package x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Stage(ctx, cfg, repo, []string{"added.go"}); err != nil {
		t.Fatalf("Stage: %v", err)
	}
	if _, err := CreateCommit(ctx, cfg, repo, "in-session commit"); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	// Also modify a file but leave it unstaged — SessionDiff should still
	// show it because Diff with --since walks to the working tree.
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("working tree\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	diff, err := mgr.SessionDiff(ctx, cfg, "session-1")
	if err != nil {
		t.Fatalf("SessionDiff: %v", err)
	}
	if !strings.Contains(diff.Diff, "added.go") {
		t.Errorf("expected added.go in session diff:\n%s", diff.Diff)
	}
	if !strings.Contains(diff.Diff, "working tree") {
		t.Errorf("expected working-tree change in session diff:\n%s", diff.Diff)
	}
}

func TestValidateRefRejectsMetacharacters(t *testing.T) {
	cases := []string{"", "HEAD; rm -rf /", "foo$(whoami)", "bar\nbaz", "--upload-pack"}
	for _, c := range cases {
		if err := validateRef(c); err == nil {
			t.Errorf("expected rejection for ref %q", c)
		}
	}
	if err := validateRef("feat/new-thing"); err != nil {
		t.Errorf("valid ref rejected: %v", err)
	}
}

func TestValidateRelPathRejectsTraversal(t *testing.T) {
	cases := []string{"", "-rf", "../etc/passwd", "a\\b"}
	for _, c := range cases {
		if err := validateRelPath(c); err == nil {
			t.Errorf("expected rejection for path %q", c)
		}
	}
	if err := validateRelPath("gateway/git/git.go"); err != nil {
		t.Errorf("valid path rejected: %v", err)
	}
}

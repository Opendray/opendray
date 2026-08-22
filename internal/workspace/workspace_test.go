package workspace

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// initRepo creates a throwaway git repo with one commit and returns its
// root. Identity is set per-repo so tests run on hosts with no global
// git config.
func initRepo(t *testing.T) string {
	t.Helper()
	// Resolve symlinks up front (macOS TMPDIR lives under /var →
	// /private/var) so equality checks against git's resolved output
	// compare like with like.
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	run(t, root, "git", "init", "-q", "-b", "main")
	run(t, root, "git", "config", "user.email", "test@example.com")
	run(t, root, "git", "config", "user.name", "test")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "sub", "dir"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sub", "dir", "f.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, root, "git", "add", ".")
	run(t, root, "git", "commit", "-q", "-m", "init")
	return root
}

func run(t *testing.T, dir string, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
	return string(out)
}

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	return NewManager(t.TempDir(), nil)
}

func TestCreate(t *testing.T) {
	ctx := context.Background()

	t.Run("repo root cwd", func(t *testing.T) {
		repo := initRepo(t)
		m := newTestManager(t)
		info, err := m.Create(ctx, repo, "ses_abc123")
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if info.RepoRoot != repo {
			t.Errorf("RepoRoot = %q, want %q", info.RepoRoot, repo)
		}
		if info.Branch != "opendray/ses_abc123" {
			t.Errorf("Branch = %q", info.Branch)
		}
		if info.WorkDir != info.Root {
			t.Errorf("WorkDir = %q, want worktree root %q", info.WorkDir, info.Root)
		}
		if _, err := os.Stat(filepath.Join(info.WorkDir, "README.md")); err != nil {
			t.Errorf("worktree missing checked-out file: %v", err)
		}
		// The worktree must be on the session branch.
		br := strings.TrimSpace(run(t, info.Root, "git", "rev-parse", "--abbrev-ref", "HEAD"))
		if br != "opendray/ses_abc123" {
			t.Errorf("worktree branch = %q", br)
		}
	})

	t.Run("subdir cwd maps into worktree", func(t *testing.T) {
		repo := initRepo(t)
		m := newTestManager(t)
		sub := filepath.Join(repo, "sub", "dir")
		info, err := m.Create(ctx, sub, "ses_sub")
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		want := filepath.Join(info.Root, "sub", "dir")
		if info.WorkDir != want {
			t.Errorf("WorkDir = %q, want %q", info.WorkDir, want)
		}
		if _, err := os.Stat(info.WorkDir); err != nil {
			t.Errorf("subdir not present in worktree: %v", err)
		}
	})

	t.Run("non-git cwd rejected", func(t *testing.T) {
		m := newTestManager(t)
		if _, err := m.Create(ctx, t.TempDir(), "ses_x"); err == nil {
			t.Fatal("want error for non-git cwd")
		}
	})

	t.Run("linked worktree cwd rejected", func(t *testing.T) {
		repo := initRepo(t)
		m := newTestManager(t)
		info, err := m.Create(ctx, repo, "ses_first")
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if _, err := m.Create(ctx, info.Root, "ses_second"); err == nil {
			t.Fatal("want error when cwd is itself a linked worktree")
		}
	})
}

func TestReclaim(t *testing.T) {
	ctx := context.Background()

	t.Run("clean untouched worktree removed", func(t *testing.T) {
		repo := initRepo(t)
		m := newTestManager(t)
		info, err := m.Create(ctx, repo, "ses_clean")
		if err != nil {
			t.Fatal(err)
		}
		res, err := m.Reclaim(ctx, repo, info.Root, info.Branch)
		if err != nil {
			t.Fatalf("Reclaim: %v", err)
		}
		if res != Removed {
			t.Fatalf("result = %v, want Removed", res)
		}
		if _, err := os.Stat(info.Root); !os.IsNotExist(err) {
			t.Errorf("worktree dir still exists")
		}
		// Branch is gone too.
		cmd := exec.Command("git", "-C", repo, "rev-parse", "--verify", info.Branch)
		if out, err := cmd.CombinedOutput(); err == nil {
			t.Errorf("branch still exists: %s", out)
		}
	})

	t.Run("dirty worktree kept", func(t *testing.T) {
		repo := initRepo(t)
		m := newTestManager(t)
		info, err := m.Create(ctx, repo, "ses_dirty")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(info.Root, "wip.txt"), []byte("uncommitted\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		res, err := m.Reclaim(ctx, repo, info.Root, info.Branch)
		if err != nil {
			t.Fatalf("Reclaim: %v", err)
		}
		if res != KeptDirty {
			t.Fatalf("result = %v, want KeptDirty", res)
		}
		if _, err := os.Stat(info.Root); err != nil {
			t.Errorf("dirty worktree was deleted: %v", err)
		}
	})

	t.Run("unmerged commits kept", func(t *testing.T) {
		repo := initRepo(t)
		m := newTestManager(t)
		info, err := m.Create(ctx, repo, "ses_ahead")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(info.Root, "feature.txt"), []byte("new\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		run(t, info.Root, "git", "add", ".")
		run(t, info.Root, "git", "commit", "-q", "-m", "session work")
		res, err := m.Reclaim(ctx, repo, info.Root, info.Branch)
		if err != nil {
			t.Fatalf("Reclaim: %v", err)
		}
		if res != KeptUnmerged {
			t.Fatalf("result = %v, want KeptUnmerged", res)
		}
		if _, err := os.Stat(info.Root); err != nil {
			t.Errorf("unmerged worktree was deleted: %v", err)
		}
	})

	t.Run("merged branch removed", func(t *testing.T) {
		repo := initRepo(t)
		m := newTestManager(t)
		info, err := m.Create(ctx, repo, "ses_merged")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(info.Root, "feature.txt"), []byte("new\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		run(t, info.Root, "git", "add", ".")
		run(t, info.Root, "git", "commit", "-q", "-m", "session work")
		run(t, repo, "git", "merge", "-q", "--no-ff", "-m", "merge session", info.Branch)
		res, err := m.Reclaim(ctx, repo, info.Root, info.Branch)
		if err != nil {
			t.Fatalf("Reclaim: %v", err)
		}
		if res != Removed {
			t.Fatalf("result = %v, want Removed", res)
		}
	})

	t.Run("missing dir pruned", func(t *testing.T) {
		repo := initRepo(t)
		m := newTestManager(t)
		info, err := m.Create(ctx, repo, "ses_gone")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.RemoveAll(info.Root); err != nil {
			t.Fatal(err)
		}
		res, err := m.Reclaim(ctx, repo, info.Root, info.Branch)
		if err != nil {
			t.Fatalf("Reclaim: %v", err)
		}
		if res != Pruned {
			t.Fatalf("result = %v, want Pruned", res)
		}
	})
}

func TestResolver(t *testing.T) {
	r := NewResolver()
	r.Register("/wt/opendray-ses_a", "/wt/opendray-ses_a/sub", "/projects/opendray")

	tests := []struct {
		name, in, want string
	}{
		{"exact workdir", "/wt/opendray-ses_a/sub", "/projects/opendray"},
		{"worktree root", "/wt/opendray-ses_a", "/projects/opendray"},
		{"nested under root", "/wt/opendray-ses_a/deep/inner", "/projects/opendray"},
		{"unrelated path untouched", "/somewhere/else", "/somewhere/else"},
		{"prefix but not subpath", "/wt/opendray-ses_abc", "/wt/opendray-ses_abc"},
		{"empty", "", ""},
		{"integration scope key untouched", "integration:tg_bot", "integration:tg_bot"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := r.Canonical(tt.in); got != tt.want {
				t.Errorf("Canonical(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}

	r.Unregister("/wt/opendray-ses_a")
	if got := r.Canonical("/wt/opendray-ses_a/sub"); got != "/wt/opendray-ses_a/sub" {
		t.Errorf("after Unregister, Canonical = %q, want passthrough", got)
	}
}

func TestStatus(t *testing.T) {
	ctx := context.Background()
	repo := initRepo(t)
	m := newTestManager(t)
	info, err := m.Create(ctx, repo, "ses_status")
	if err != nil {
		t.Fatal(err)
	}

	st, err := m.Status(ctx, repo, info.Root, info.Branch)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if st.Dirty || st.Unmerged || st.Missing {
		t.Errorf("fresh worktree status = %+v, want all clear", st)
	}

	if err := os.WriteFile(filepath.Join(info.Root, "wip.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err = m.Status(ctx, repo, info.Root, info.Branch)
	if err != nil {
		t.Fatal(err)
	}
	if !st.Dirty {
		t.Errorf("want Dirty after uncommitted change, got %+v", st)
	}
}

// Package workspace manages per-session git worktrees for isolated
// sessions, plus the scope-key canonicalization that keeps worktree
// paths out of the project-identity namespace.
//
// A session created with isolation=worktree runs its CLI in a private
// checkout under <baseDir>/<repo>-<sessionID> on branch
// opendray/<sessionID>, while every project-keyed subsystem (memory,
// project docs, Cortex) keeps using the session's logical cwd. See
// docs/design/worktree-isolation.md.
package workspace

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// gitTimeout caps each git invocation; worktree add on a large repo is
// the slowest operation (checkout), so this is generous compared to the
// read-only inspection commands.
const gitTimeout = 30 * time.Second

// BranchPrefix namespaces every managed session branch.
const BranchPrefix = "opendray/"

var (
	// ErrNotGitRepo — the requested cwd is not inside a git work tree,
	// so no worktree can be created for it.
	ErrNotGitRepo = errors.New("workspace: cwd is not inside a git repository")
	// ErrLinkedWorktree — the requested cwd is itself a linked worktree;
	// isolation must anchor on the main checkout.
	ErrLinkedWorktree = errors.New("workspace: cwd is a linked worktree; use the main checkout")
)

// Info describes a created worktree.
type Info struct {
	// RepoRoot is the main checkout's top-level directory.
	RepoRoot string
	// Root is the worktree checkout's top-level directory.
	Root string
	// WorkDir is where the session process runs: Root plus the
	// cwd-relative subdirectory when the logical cwd was inside the
	// repo rather than at its root.
	WorkDir string
	// Branch is the session branch the worktree was created on.
	Branch string
}

// ReclaimResult says what Reclaim did with a worktree.
type ReclaimResult int

const (
	// Removed — worktree and branch deleted; nothing of value was lost
	// (clean tree, branch tip reachable from the main checkout's HEAD).
	Removed ReclaimResult = iota
	// KeptDirty — uncommitted changes present; everything preserved.
	KeptDirty
	// KeptUnmerged — committed work not reachable from the main
	// checkout's HEAD; everything preserved.
	KeptUnmerged
	// Pruned — the directory was already gone from disk; stale git
	// bookkeeping was pruned. The branch is preserved (it may carry
	// commits; deleting refs is never done blind).
	Pruned
)

func (r ReclaimResult) String() string {
	switch r {
	case Removed:
		return "removed"
	case KeptDirty:
		return "kept_dirty"
	case KeptUnmerged:
		return "kept_unmerged"
	case Pruned:
		return "pruned"
	}
	return fmt.Sprintf("ReclaimResult(%d)", int(r))
}

// Status is the inspection view of a managed worktree, used by the UI
// to flag sessions whose worktree needs attention.
type Status struct {
	// Missing — the worktree directory no longer exists on disk.
	Missing bool `json:"missing"`
	// Dirty — uncommitted changes in the worktree.
	Dirty bool `json:"dirty"`
	// Unmerged — the session branch has commits not reachable from the
	// main checkout's HEAD.
	Unmerged bool `json:"unmerged"`
}

// Manager creates and reclaims managed worktrees. Stateless beyond the
// base directory; all state lives in git and the sessions table.
type Manager struct {
	baseDir string
	log     *slog.Logger
}

// NewManager returns a Manager rooted at baseDir (created lazily).
func NewManager(baseDir string, log *slog.Logger) *Manager {
	if log == nil {
		log = slog.Default()
	}
	return &Manager{baseDir: baseDir, log: log.With("component", "workspace")}
}

// Create makes a worktree for the session: branch opendray/<sessionID>
// off the main checkout's current HEAD, checked out under
// <baseDir>/<repoBase>-<sessionID>. cwd may be the repo root or any
// directory inside it; WorkDir preserves the relative position.
func (m *Manager) Create(ctx context.Context, cwd, sessionID string) (Info, error) {
	repoRoot, err := m.repoRoot(ctx, cwd)
	if err != nil {
		return Info{}, err
	}
	if linked, err := m.isLinkedWorktree(ctx, cwd); err != nil {
		return Info{}, err
	} else if linked {
		return Info{}, ErrLinkedWorktree
	}

	// cwd's position relative to the repo root, from git itself —
	// filepath.Rel would be fooled when the two paths differ in
	// symlink resolution (macOS /var vs /private/var).
	prefix, err := m.git(ctx, cwd, "rev-parse", "--show-prefix")
	if err != nil {
		return Info{}, ErrNotGitRepo
	}
	rel := strings.TrimSuffix(strings.TrimSpace(prefix), "/")

	if err := os.MkdirAll(m.baseDir, 0o700); err != nil {
		return Info{}, fmt.Errorf("workspace: base dir: %w", err)
	}
	root := filepath.Join(m.baseDir, filepath.Base(repoRoot)+"-"+sessionID)
	branch := BranchPrefix + sessionID

	// Stale leftovers (a previous gateway crash mid-create) would make
	// worktree add fail; prune bookkeeping for paths that no longer
	// exist before trying.
	if _, err := os.Stat(root); err == nil {
		return Info{}, fmt.Errorf("workspace: worktree path already exists: %s", root)
	}
	_, _ = m.git(ctx, repoRoot, "worktree", "prune")

	if out, err := m.git(ctx, repoRoot, "worktree", "add", "-b", branch, root, "HEAD"); err != nil {
		return Info{}, fmt.Errorf("workspace: worktree add: %w: %s", err, strings.TrimSpace(out))
	}

	workDir := root
	if rel != "" && rel != "." {
		workDir = filepath.Join(root, rel)
		if info, err := os.Stat(workDir); err != nil || !info.IsDir() {
			// The subdir exists in the main checkout but not in HEAD
			// (e.g. untracked-only directory). Fall back to the root
			// rather than spawning into a missing dir.
			m.log.Warn("cwd subdir absent in worktree; using worktree root",
				"cwd", cwd, "workdir", workDir)
			workDir = root
		}
	}

	m.log.Info("worktree created", "session", sessionID, "root", root, "branch", branch)
	return Info{RepoRoot: repoRoot, Root: root, WorkDir: workDir, Branch: branch}, nil
}

// Reclaim disposes of a session's worktree if — and only if — doing so
// loses nothing: the tree is clean and the branch tip is reachable from
// the main checkout's HEAD (never committed, or already merged). Any
// other state is preserved and reported.
//
// repoCwd is the session's logical cwd (used to locate the main
// checkout); root/branch are the persisted worktree coordinates.
func (m *Manager) Reclaim(ctx context.Context, repoCwd, root, branch string) (ReclaimResult, error) {
	repoRoot, err := m.repoRoot(ctx, repoCwd)
	if err != nil {
		return KeptDirty, err
	}

	if _, err := os.Stat(root); os.IsNotExist(err) {
		_, _ = m.git(ctx, repoRoot, "worktree", "prune")
		m.log.Info("worktree dir already gone; pruned bookkeeping", "root", root)
		return Pruned, nil
	}

	st, err := m.Status(ctx, repoCwd, root, branch)
	if err != nil {
		return KeptDirty, err
	}
	if st.Dirty {
		return KeptDirty, nil
	}
	if st.Unmerged {
		return KeptUnmerged, nil
	}

	if out, err := m.git(ctx, repoRoot, "worktree", "remove", root); err != nil {
		return KeptDirty, fmt.Errorf("workspace: worktree remove: %w: %s", err, strings.TrimSpace(out))
	}
	// Safe by construction: Status verified the branch tip is an
	// ancestor of the main checkout's HEAD, so -D cannot lose commits.
	if out, err := m.git(ctx, repoRoot, "branch", "-D", branch); err != nil {
		m.log.Warn("worktree removed but branch delete failed", "branch", branch, "err", err, "out", strings.TrimSpace(out))
	}
	m.log.Info("worktree reclaimed", "root", root, "branch", branch)
	return Removed, nil
}

// Status inspects a managed worktree without changing anything.
func (m *Manager) Status(ctx context.Context, repoCwd, root, branch string) (Status, error) {
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return Status{Missing: true}, nil
	}
	var st Status
	out, err := m.git(ctx, root, "status", "--porcelain")
	if err != nil {
		return st, fmt.Errorf("workspace: status: %w", err)
	}
	st.Dirty = strings.TrimSpace(out) != ""

	repoRoot, err := m.repoRoot(ctx, repoCwd)
	if err != nil {
		return st, err
	}
	// Branch tip reachable from the main checkout's HEAD ⇒ nothing to
	// lose. merge-base --is-ancestor exits 1 when not an ancestor.
	if _, err := m.git(ctx, repoRoot, "merge-base", "--is-ancestor", branch, "HEAD"); err != nil {
		st.Unmerged = true
	}
	return st, nil
}

// RootOf maps a persisted work dir back to its worktree checkout root:
// the first path component under the manager's base directory. Returns
// "" when workDir is not under baseDir (not a managed worktree). Lets
// callers that only persisted WorkDir recover the root for reclaim /
// registry operations without shelling out to git.
func (m *Manager) RootOf(workDir string) string {
	rel, err := filepath.Rel(m.baseDir, workDir)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return ""
	}
	parts := strings.SplitN(rel, string(filepath.Separator), 2)
	return filepath.Join(m.baseDir, parts[0])
}

func (m *Manager) repoRoot(ctx context.Context, cwd string) (string, error) {
	out, err := m.git(ctx, cwd, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", ErrNotGitRepo
	}
	return strings.TrimSpace(out), nil
}

// isLinkedWorktree reports whether cwd belongs to a linked worktree
// (as opposed to the main checkout): in a linked worktree the repo's
// git dir differs from the common dir. Both paths come from a single
// git invocation with --path-format=absolute so symlinked temp dirs
// (macOS /var → /private/var) can't skew the comparison.
func (m *Manager) isLinkedWorktree(ctx context.Context, cwd string) (bool, error) {
	out, err := m.git(ctx, cwd, "rev-parse", "--path-format=absolute", "--git-dir", "--git-common-dir")
	if err != nil {
		return false, ErrNotGitRepo
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		return false, fmt.Errorf("workspace: unexpected rev-parse output: %q", out)
	}
	return strings.TrimSpace(lines[0]) != strings.TrimSpace(lines[1]), nil
}

func (m *Manager) git(ctx context.Context, dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, gitTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

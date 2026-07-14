package checkpoint

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// RestoreResult reports what a Restore actually did.
type RestoreResult struct {
	CheckpointID      string   `json:"checkpoint_id"`
	DiffApplied       bool     `json:"diff_applied"`
	UntrackedRestored int      `json:"untracked_restored"`
	UntrackedSkipped  []string `json:"untracked_skipped,omitempty"` // target already existed
}

// Restore errors. All are pre-write guards except a failure mid-apply,
// which git apply makes atomic (it applies all hunks or none).
var (
	// ErrNotGitCheckpoint — the checkpoint captured a non-git cwd (metadata
	// only), so there is nothing to apply.
	ErrNotGitCheckpoint = errors.New("checkpoint has no git payload to restore")
	// ErrNotWorkTree — the target cwd is no longer a git working tree.
	ErrNotWorkTree = errors.New("target cwd is not a git working tree")
	// ErrHeadMismatch — HEAD moved since capture; applying the diff would
	// be against a different base. Refused (strict posture).
	ErrHeadMismatch = errors.New("HEAD has moved since the checkpoint; refusing to restore")
	// ErrDirtyWorktree — there are uncommitted tracked changes; restoring
	// could clobber them or conflict. Refused (strict posture).
	ErrDirtyWorktree = errors.New("working tree has uncommitted changes; commit or stash them first")
	// ErrApplyCheckFailed — the stored diff does not apply cleanly to the
	// current tree (git apply --check dry-run failed). Refused before any
	// write. The wrapped message carries git's explanation.
	ErrApplyCheckFailed = errors.New("stored diff does not apply cleanly")
)

// Restore re-applies a checkpoint's captured context onto its cwd, under a
// strict, dry-run-first posture:
//
//  1. the cwd must be a git work tree whose HEAD equals the checkpoint's,
//  2. it must have no uncommitted tracked changes,
//  3. the stored diff must pass `git apply --check` (dry-run),
//
// only then is the diff applied. Untracked files are restored only where
// the target does not already exist (never overwriting). Any guard failing
// returns before a single byte is written.
func (s *Service) Restore(ctx context.Context, id string) (RestoreResult, error) {
	cp, err := s.store.get(ctx, id)
	if err != nil {
		return RestoreResult{}, err
	}
	res, err := applyCheckpoint(ctx, cp)
	if err == nil {
		s.log.Info("restored session checkpoint", "checkpoint_id", id, "cwd", cp.Cwd,
			"diff_applied", res.DiffApplied, "untracked_restored", res.UntrackedRestored,
			"untracked_skipped", len(res.UntrackedSkipped))
	}
	return res, err
}

// applyCheckpoint is the DB-free core of Restore: it enforces the strict
// guards and applies cp's payload onto cp.Cwd. Split out so the restore
// logic is testable without a database.
func applyCheckpoint(ctx context.Context, cp Checkpoint) (RestoreResult, error) {
	res := RestoreResult{CheckpointID: cp.ID}
	if !cp.IsGit {
		return res, ErrNotGitCheckpoint
	}
	cwd := cp.Cwd
	if !isGitWorkTree(ctx, cwd) {
		return res, ErrNotWorkTree
	}

	// Guard 1: HEAD must match the capture point.
	head := strings.TrimSpace(gitOut(ctx, cwd, "rev-parse", "HEAD"))
	if head != cp.GitHead {
		return res, fmt.Errorf("%w (checkpoint %s, now %s)", ErrHeadMismatch, shortSHA(cp.GitHead), shortSHA(head))
	}

	// Guard 2: no uncommitted *tracked* changes (untracked files are fine —
	// they don't conflict with the tracked-changes patch, and the untracked
	// restore below never overwrites).
	if dirty := strings.TrimSpace(gitOut(ctx, cwd, "status", "--porcelain", "--untracked-files=no")); dirty != "" {
		return res, ErrDirtyWorktree
	}

	// Apply the diff (if any) — dry-run first, then for real.
	diffPath := filepath.Join(cp.StoragePath, fileDiff)
	if fileExists(diffPath) {
		if out, err := gitRunCombined(ctx, cwd, "apply", "--check", diffPath); err != nil {
			return res, fmt.Errorf("%w: %s", ErrApplyCheckFailed, strings.TrimSpace(out))
		}
		if out, err := gitRunCombined(ctx, cwd, "apply", diffPath); err != nil {
			return res, fmt.Errorf("git apply: %s", strings.TrimSpace(out))
		}
		res.DiffApplied = true
	}

	// Restore untracked files without overwriting anything present.
	restored, skipped, err := restoreUntracked(filepath.Join(cp.StoragePath, dirUntracked), cwd)
	if err != nil {
		// The diff (if any) is already applied; report partial success with
		// the error so the operator knows the untracked restore was cut.
		res.UntrackedRestored = restored
		res.UntrackedSkipped = skipped
		return res, fmt.Errorf("restore untracked files: %w", err)
	}
	res.UntrackedRestored = restored
	res.UntrackedSkipped = skipped
	return res, nil
}

// restoreUntracked copies files from untrackedDir back into cwd at their
// captured relative paths, skipping any whose target already exists (no
// overwrite). Returns the restored count and the skipped relpaths.
func restoreUntracked(untrackedDir, cwd string) (int, []string, error) {
	info, err := os.Stat(untrackedDir)
	if os.IsNotExist(err) || (err == nil && !info.IsDir()) {
		return 0, nil, nil
	}
	var restored int
	var skipped []string
	walkErr := filepath.WalkDir(untrackedDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(untrackedDir, path)
		if err != nil {
			return err
		}
		if !safeRel(rel) {
			skipped = append(skipped, rel)
			return nil
		}
		target := filepath.Join(cwd, rel)
		if fileExists(target) {
			skipped = append(skipped, rel)
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return err
		}
		restored++
		return nil
	})
	return restored, skipped, walkErr
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func shortSHA(s string) string {
	if len(s) > 8 {
		return s[:8]
	}
	if s == "" {
		return "(none)"
	}
	return s
}

// gitRunCombined runs `git <args>` in cwd and returns combined stdout+stderr
// plus any error, so restore can surface git's explanation on failure.
// Unlike gitOut it does NOT swallow errors.
func gitRunCombined(ctx context.Context, cwd string, args ...string) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, gitTimeout)
	defer cancel()
	cmd := exec.CommandContext(cctx, "git", args...)
	cmd.Dir = cwd
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), err
}

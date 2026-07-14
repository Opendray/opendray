package checkpoint

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// CaptureRequest is the input to Capture: everything needed to snapshot a
// session's context, with no dependency on the session package.
type CaptureRequest struct {
	CheckpointID string  // pre-generated id (also the on-disk dir name)
	SessionID    string  // owning session
	Cwd          string  // working directory to snapshot
	Trigger      Trigger // interrupted | manual
	InputHistory []byte  // operator input tail (may be nil)
	Note         string  // optional operator note
	Now          time.Time
}

// gitTimeout bounds each git invocation so a wedged repo can't stall a
// capture (which, for the interrupt trigger, runs during shutdown).
const gitTimeout = 5 * time.Second

// Capture snapshots the request's cwd into storageRoot/<checkpointID>/ and
// returns the manifest. It never mutates the cwd. Capture is best-effort
// within its caps: hitting a cap sets Truncated but still produces a valid
// checkpoint. A capture-fatal error (cannot create the dir, cannot write
// the manifest) returns a non-nil error and removes any partial dir.
func Capture(ctx context.Context, storageRoot string, req CaptureRequest) (Checkpoint, error) {
	if storageRoot == "" {
		return Checkpoint{}, ErrNoStorageDir
	}
	if !req.Trigger.Valid() {
		return Checkpoint{}, ErrInvalidTrigger
	}
	dir := filepath.Join(storageRoot, req.CheckpointID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return Checkpoint{}, fmt.Errorf("checkpoint: mkdir: %w", err)
	}

	cp := Checkpoint{
		ID:          req.CheckpointID,
		SessionID:   req.SessionID,
		CreatedAt:   req.Now.UTC(),
		Trigger:     req.Trigger,
		Cwd:         req.Cwd,
		StoragePath: dir,
		Note:        req.Note,
	}

	// Input history first — it's cheap and independent of the git state.
	if len(req.InputHistory) > 0 {
		if err := os.WriteFile(filepath.Join(dir, fileInput), req.InputHistory, 0o600); err != nil {
			_ = os.RemoveAll(dir)
			return Checkpoint{}, fmt.Errorf("checkpoint: write input history: %w", err)
		}
		cp.InputBytes = int64(len(req.InputHistory))
	}

	if isGitWorkTree(ctx, req.Cwd) {
		cp.IsGit = true
		if err := captureGit(ctx, dir, req.Cwd, &cp); err != nil {
			// A git capture failure is non-fatal: keep the metadata-only
			// checkpoint (input history is already written) and note why.
			cp.Note = strings.TrimSpace(cp.Note + "\n[git capture error: " + err.Error() + "]")
		}
	} else {
		cp.Note = strings.TrimSpace(cp.Note + "\n[cwd is not a git repository; file snapshot skipped]")
	}

	if err := writeManifest(dir, cp); err != nil {
		_ = os.RemoveAll(dir)
		return Checkpoint{}, err
	}
	return cp, nil
}

// captureGit writes the uncommitted diff and untracked (non-ignored) files
// into dir, updating cp's git fields. Respects .gitignore via
// --exclude-standard, so ignored paths (secrets, node_modules, build
// output) are never copied.
func captureGit(ctx context.Context, dir, cwd string, cp *Checkpoint) error {
	cp.GitHead = strings.TrimSpace(gitOut(ctx, cwd, "rev-parse", "HEAD"))

	// Uncommitted tracked changes. With a HEAD, diff against it (staged +
	// unstaged); on an unborn branch (no commits yet) fall back to the
	// worktree diff so a brand-new repo still captures something.
	diffArgs := []string{"diff", "--no-color", "HEAD"}
	if cp.GitHead == "" {
		diffArgs = []string{"diff", "--no-color"}
	}
	diff := []byte(gitOut(ctx, cwd, diffArgs...))
	if len(diff) > MaxDiffBytes {
		diff = diff[:MaxDiffBytes]
		cp.Truncated = true
	}
	if len(diff) > 0 {
		if err := os.WriteFile(filepath.Join(dir, fileDiff), diff, 0o600); err != nil {
			return fmt.Errorf("write diff: %w", err)
		}
		cp.DiffBytes = int64(len(diff))
	}

	// Untracked, non-ignored files (NUL-separated, so paths with spaces or
	// newlines are handled). git already applied .gitignore + exclude rules.
	out := gitOut(ctx, cwd, "ls-files", "--others", "--exclude-standard", "-z")
	var copied int
	var total int64
	for _, rel := range strings.Split(out, "\x00") {
		if rel == "" {
			continue
		}
		if copied >= MaxUntrackedFiles || total >= MaxUntrackedTotalBytes {
			cp.Truncated = true
			break
		}
		n, skipped, err := copyUntracked(cwd, dir, rel, MaxUntrackedTotalBytes-total)
		if err != nil {
			continue // unreadable file: skip, don't fail the whole capture
		}
		if skipped {
			cp.Truncated = true
			continue
		}
		copied++
		total += n
	}
	cp.UntrackedFiles = copied
	cp.UntrackedBytes = total
	cp.GitDirty = cp.DiffBytes > 0 || copied > 0
	return nil
}

// copyUntracked copies cwd/rel into dir/untracked/rel, preserving the
// relative path. Returns the bytes written, whether it was skipped by a cap
// (too large, or would exceed remaining budget), and any hard error. rel
// comes from `git ls-files` so it is repo-relative and clean, but we still
// reject any path that would escape the untracked dir.
func copyUntracked(cwd, dir, rel string, remaining int64) (int64, bool, error) {
	if !safeRel(rel) {
		return 0, true, nil
	}
	src := filepath.Join(cwd, rel)
	info, err := os.Lstat(src)
	if err != nil {
		return 0, false, err
	}
	// Only regular files: skip symlinks (escape risk) and anything special.
	if !info.Mode().IsRegular() {
		return 0, true, nil
	}
	if info.Size() > MaxUntrackedFileBytes || info.Size() > remaining {
		return 0, true, nil
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return 0, false, err
	}
	dst := filepath.Join(dir, dirUntracked, rel)
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return 0, false, err
	}
	if err := os.WriteFile(dst, data, 0o600); err != nil {
		return 0, false, err
	}
	return int64(len(data)), false, nil
}

// safeRel rejects absolute paths and any path that escapes its root via
// "..", defence-in-depth over git's already-clean output.
func safeRel(rel string) bool {
	if rel == "" || filepath.IsAbs(rel) {
		return false
	}
	clean := filepath.Clean(rel)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return false
	}
	return true
}

func writeManifest(dir string, cp Checkpoint) error {
	data, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return fmt.Errorf("checkpoint: marshal manifest: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, fileMeta), data, 0o600); err != nil {
		return fmt.Errorf("checkpoint: write manifest: %w", err)
	}
	return nil
}

func isGitWorkTree(ctx context.Context, cwd string) bool {
	out := strings.TrimSpace(gitOut(ctx, cwd, "rev-parse", "--is-inside-work-tree"))
	return out == "true"
}

// gitOut runs `git <args>` in cwd and returns stdout (stderr discarded),
// bounded by gitTimeout. Returns "" on any error — callers treat a git
// failure as "no data", never fatal.
func gitOut(ctx context.Context, cwd string, args ...string) string {
	cctx, cancel := context.WithTimeout(ctx, gitTimeout)
	defer cancel()
	cmd := exec.CommandContext(cctx, "git", args...)
	cmd.Dir = cwd
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return ""
	}
	return out.String()
}

package session

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// migrateClaudeTranscript makes the conversation transcript file for
// sessionID visible inside newConfigDir's projects tree, so the next
// `claude --resume <sessionID>` spawned under that config dir finds
// and replays it. The file lives at
// <oldConfigDir>/projects/<workspace>/<sessionID>.jsonl — we don't
// know the workspace name a priori (Claude derives it from cwd via
// its own normalization), so we glob.
//
// Behavior:
//   - oldConfigDir == newConfigDir → nothing to do (no-op).
//   - sessionID == "" → nothing to migrate (no-op).
//   - source not found → no error; the conversation simply hasn't been
//     persisted yet (this can happen for sessions <1 turn old).
//   - destination already exists → leave it alone (prior switch already
//     migrated; both inodes are valid).
//   - hard-link attempt first (atomic, same-inode so both accounts'
//     views stay in sync going forward); copy fallback if cross-fs or
//     other Link() failure (preserves first migration but later writes
//     diverge — acceptable since switches are infrequent).
//
// Returns the first non-nil error from any required filesystem step;
// caller logs and falls through to a no-transcript respawn.
func migrateClaudeTranscript(oldConfigDir, newConfigDir, sessionID string) error {
	if sessionID == "" || oldConfigDir == "" || newConfigDir == "" {
		return nil
	}
	if oldConfigDir == newConfigDir {
		return nil
	}

	pattern := filepath.Join(oldConfigDir, "projects", "*", sessionID+".jsonl")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("glob old transcripts: %w", err)
	}
	if len(matches) == 0 {
		return nil // nothing to migrate
	}
	// Glob may return multiple workspaces if (somehow) the same UUID
	// existed under more than one — pick the newest by mtime so we
	// migrate the most recently-touched conversation.
	src := matches[0]
	if len(matches) > 1 {
		newest := src
		newestT := mtime(src)
		for _, m := range matches[1:] {
			if t := mtime(m); t > newestT {
				newest, newestT = m, t
			}
		}
		src = newest
	}

	workspace := filepath.Base(filepath.Dir(src))
	destDir := filepath.Join(newConfigDir, "projects", workspace)
	if err := os.MkdirAll(destDir, 0o700); err != nil {
		return fmt.Errorf("mkdir new transcript dir: %w", err)
	}
	dst := filepath.Join(destDir, sessionID+".jsonl")

	// If destination already exists (a prior switch already migrated
	// this conversation), leave it. Don't overwrite — the existing
	// file might be the same inode (hard-link) or a copy with later
	// updates from this account.
	if _, err := os.Lstat(dst); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat new transcript: %w", err)
	}

	// Hard-link first. Both accounts' projects/ trees then reference
	// one inode, so further writes are mutually visible — switching
	// BACK preserves whatever the new account added.
	if err := os.Link(src, dst); err == nil {
		return nil
	}
	// Fallback: physical copy. Same-inode sharing is lost, but at
	// least the conversation is restored on the next --resume.
	return copyFile(src, dst)
}

// mtime returns a file's modification time as a comparable int64. A
// missing or unreadable file sorts oldest (Unix(0)).
func mtime(path string) int64 {
	st, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return st.ModTime().UnixNano()
}

// copyFile streams src → dst. dst is created chmod 0600 (matches the
// Claude CLI's own perm bits on .jsonl files). Used only as the
// hard-link fallback in migrateClaudeTranscript.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open src: %w", err)
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create dst: %w", err)
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy bytes: %w", err)
	}
	return out.Sync()
}

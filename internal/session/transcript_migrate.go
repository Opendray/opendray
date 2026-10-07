package session

import (
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
// `claude --resume` only needs the *.jsonl under the target config
// dir — verified against Claude Code 2.1.292: a transcript authored
// under account A and copied into account B's projects/ resumes under
// B with full history. (#331 concluded otherwise and dropped this
// migration, which is why switching accounts lost the conversation.)
//
// The bool reports whether the conversation is resumable under
// newConfigDir after the call. The caller resumes only on true, so a
// missing transcript can never produce the doomed `--resume` loop
// #331 fixed.
//
// Behavior:
//   - sessionID / a config dir empty → (false, nil).
//   - oldConfigDir == newConfigDir → true iff the transcript exists.
//   - source not found → (false, nil); the conversation hasn't been
//     persisted yet (sessions <1 turn old).
//   - destination is the same inode as the source (a prior hard-link)
//     → (true, nil).
//   - destination is a different file (a stale copy from an earlier
//     switch the other way) → replaced. The source is authoritative:
//     it belongs to the account that just ran the conversation, so
//     keeping the old destination would resume a truncated history.
//   - hard-link first (same inode, so later writes by either account
//     stay mutually visible); copy fallback if cross-fs. Both land via
//     a temp file + rename, so dst is never observed half-written.
func migrateClaudeTranscript(oldConfigDir, newConfigDir, sessionID string) (bool, error) {
	if sessionID == "" || oldConfigDir == "" || newConfigDir == "" {
		return false, nil
	}

	pattern := filepath.Join(oldConfigDir, "projects", "*", sessionID+".jsonl")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return false, fmt.Errorf("glob old transcripts: %w", err)
	}
	if len(matches) == 0 {
		return false, nil // nothing to migrate
	}
	if oldConfigDir == newConfigDir {
		return true, nil
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

	// Defense in depth: reject any source that isn't a plain regular
	// file before linking. On Linux, os.Link() creates a hard-link to
	// the *symlink itself*, which means dst would also resolve to
	// whatever the symlink targets — a planted symlink at
	// <projects>/<ws>/<sid>.jsonl → /var/.../memory.key (or any other
	// file the opendray user can read) would be migrated into the
	// new account's tree and then read by `claude --resume` as
	// "conversation history", potentially exfiltrating its contents
	// to anthropic.com on the next agent message. Lstat-reject
	// non-regular files (symlinks, devices, fifos) up front.
	srcStat, err := os.Lstat(src)
	if err != nil {
		return false, fmt.Errorf("lstat src: %w", err)
	}
	if srcStat.Mode()&os.ModeSymlink != 0 {
		return false, fmt.Errorf("refusing to migrate symlinked transcript at %s", src)
	}
	if !srcStat.Mode().IsRegular() {
		return false, fmt.Errorf("refusing to migrate non-regular file at %s", src)
	}

	workspace := filepath.Base(filepath.Dir(src))
	destDir := filepath.Join(newConfigDir, "projects", workspace)
	if err := os.MkdirAll(destDir, 0o700); err != nil {
		return false, fmt.Errorf("mkdir new transcript dir: %w", err)
	}
	dst := filepath.Join(destDir, sessionID+".jsonl")

	if dstStat, err := os.Lstat(dst); err == nil && os.SameFile(srcStat, dstStat) {
		return true, nil // already linked by a prior switch
	}

	tmp := dst + ".migrate.tmp"
	_ = os.Remove(tmp)
	if err := os.Link(src, tmp); err != nil {
		// Fallback: physical copy. Same-inode sharing is lost, but the
		// conversation still resumes; a later switch back re-copies.
		if err := copyFile(src, tmp); err != nil {
			_ = os.Remove(tmp)
			return false, err
		}
	}
	if err := os.Rename(tmp, dst); err != nil {
		_ = os.Remove(tmp)
		return false, fmt.Errorf("install transcript: %w", err)
	}
	return true, nil
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

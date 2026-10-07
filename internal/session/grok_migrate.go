package session

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// migrateGrokSession makes grok session sessionID for cwd resumable under
// newHome by copying its directory out of oldHome:
//
//	<GROK_HOME>/sessions/<percent-encoded-cwd>/<sessionID>/
//
// `grok --resume <id>` needs only that directory under the target
// GROK_HOME — verified against grok 1.0.46: a session authored under one
// account home, copied into another, resumed there with full history.
//
// The bool reports whether the session is resumable under newHome after
// the call; the caller resumes only on true and otherwise falls back to a
// recap. Same contract as migrateClaudeTranscript:
//   - empty input or no source session → (false, nil).
//   - oldHome == newHome → true iff the session exists.
//   - an existing destination (a stale copy from an earlier switch the
//     other way) is replaced: the source belongs to the account that just
//     ran the conversation, so it is authoritative.
//   - only regular files are copied; a symlink anywhere in the source
//     aborts the copy (it would be read back as conversation history and
//     sent to the provider). *.lock files are runtime state and skipped.
//   - the copy is staged in a sibling temp dir and renamed into place, so
//     a half-copied session is never observed.
func migrateGrokSession(oldHome, newHome, cwd, sessionID string) (bool, error) {
	if oldHome == "" || newHome == "" || cwd == "" || !safePathElement(sessionID) {
		return false, nil
	}
	srcCwdDir := findGrokCwdDir(filepath.Join(oldHome, "sessions"), cwd)
	if srcCwdDir == "" {
		return false, nil
	}
	src := filepath.Join(srcCwdDir, sessionID)
	st, err := os.Lstat(src)
	if err != nil || !st.IsDir() {
		return false, nil // nothing to migrate
	}
	if filepath.Clean(oldHome) == filepath.Clean(newHome) {
		return true, nil
	}

	// Reuse the exact encoded dir name grok chose under the old home, or
	// the one the new home already has for this cwd.
	dstCwdDir := findGrokCwdDir(filepath.Join(newHome, "sessions"), cwd)
	if dstCwdDir == "" {
		dstCwdDir = filepath.Join(newHome, "sessions", filepath.Base(srcCwdDir))
	}
	if err := os.MkdirAll(dstCwdDir, 0o700); err != nil {
		return false, fmt.Errorf("mkdir new session dir: %w", err)
	}
	dst := filepath.Join(dstCwdDir, sessionID)
	tmp := dst + ".migrate.tmp"
	_ = os.RemoveAll(tmp)
	if err := copyGrokSessionDir(src, tmp); err != nil {
		_ = os.RemoveAll(tmp)
		return false, err
	}
	if err := os.RemoveAll(dst); err != nil {
		_ = os.RemoveAll(tmp)
		return false, fmt.Errorf("remove stale session: %w", err)
	}
	if err := os.Rename(tmp, dst); err != nil {
		_ = os.RemoveAll(tmp)
		return false, fmt.Errorf("install session: %w", err)
	}
	return true, nil
}

// copyGrokSessionDir recursively copies src into a new dir dst, regular
// files and dirs only. Any symlink, device or fifo aborts the copy.
func copyGrokSessionDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0o700); err != nil {
		return fmt.Errorf("mkdir %s: %w", dst, err)
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("read session dir: %w", err)
	}
	for _, e := range entries {
		s := filepath.Join(src, e.Name())
		d := filepath.Join(dst, e.Name())
		info, err := os.Lstat(s)
		if err != nil {
			return fmt.Errorf("lstat %s: %w", s, err)
		}
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			return fmt.Errorf("refusing to migrate symlink in grok session at %s", s)
		case info.IsDir():
			if err := copyGrokSessionDir(s, d); err != nil {
				return err
			}
		case info.Mode().IsRegular():
			if strings.HasSuffix(e.Name(), ".lock") {
				continue
			}
			if err := copyRegularFile(s, d); err != nil {
				return err
			}
		default:
			return fmt.Errorf("refusing to migrate non-regular file in grok session at %s", s)
		}
	}
	return nil
}

// copyRegularFile copies src → dst (created 0600, must not exist).
func copyRegularFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open src: %w", err)
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create dst: %w", err)
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return fmt.Errorf("copy bytes: %w", err)
	}
	if err := out.Sync(); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

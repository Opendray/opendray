package session

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// codex_rollout.go: keep a codex conversation alive across restarts and
// account switches.
//
// codex records each conversation ("thread") as one rollout file:
//
//	<CODEX_HOME>/sessions/YYYY/MM/DD/rollout-<ts>-<thread-id>.jsonl
//
// and `codex resume <thread-id>` finds it by scanning <CODEX_HOME>/sessions
// — the file is all it needs (verified against codex-cli 0.149.1: a
// rollout copied into a different CODEX_HOME passes the lookup that a
// missing id fails with "no rollout found for thread id").
//
// opendray runs every codex session under a per-session scratch
// CODEX_HOME inside the session temp dir (it injects MCP config +
// AGENTS.md there), and that temp dir is removed when the session ends.
// So the durable copy of a conversation lives in the session's
// *credential* home — the bound account's CODEX_HOME, or ~/.codex — and
// the scratch home only ever holds copies:
//
//   - spawn:   SeedCodexRollout copies the thread being resumed from the
//     credential home into the scratch home.
//   - exit:    SyncCodexHome copies every rollout (new or appended) plus a
//     refreshed auth.json back into the credential home, and
//     reports the thread id so the manager can persist it.
//   - switch:  MigrateCodexRollout copies the thread from the old
//     account's home into the new account's home.
//
// Every copy rejects symlinked / non-regular sources (a planted symlink
// must never be read back as "conversation history") and lands via a
// temp file + rename so a reader never sees a half-written rollout.

// codexRolloutSuffix is the filename suffix identifying threadID's rollout.
func codexRolloutSuffix(threadID string) string { return "-" + threadID + ".jsonl" }

// FindCodexRollout returns the path of threadID's rollout under
// <home>/sessions, or "" when there is none. The newest by mtime wins in
// the (pathological) case of several matches.
func FindCodexRollout(home, threadID string) string {
	if home == "" || threadID == "" || !safePathElement(threadID) {
		return ""
	}
	suffix := codexRolloutSuffix(threadID)
	var best string
	var bestT int64
	_ = filepath.WalkDir(filepath.Join(home, "sessions"), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !d.Type().IsRegular() {
			return nil
		}
		if !strings.HasPrefix(d.Name(), "rollout-") || !strings.HasSuffix(d.Name(), suffix) {
			return nil
		}
		if t := codexMtime(path); best == "" || t > bestT {
			best, bestT = path, t
		}
		return nil
	})
	return best
}

// MigrateCodexRollout makes threadID's rollout present under newHome at
// the same <sessions>-relative path it has under oldHome, so `codex resume
// <threadID>` spawned against newHome finds it. The bool reports whether
// the conversation is resumable under newHome afterwards; the caller
// resumes only on true, so a missing rollout falls back to a fresh start
// instead of a resume that exits with "no rollout found".
//
// A differing destination (a stale copy from an earlier switch the other
// way) is replaced: the source belongs to the account that just ran the
// conversation, so it's authoritative.
func MigrateCodexRollout(oldHome, newHome, threadID string) (bool, error) {
	if oldHome == "" || newHome == "" || threadID == "" {
		return false, nil
	}
	src := FindCodexRollout(oldHome, threadID)
	if src == "" {
		return false, nil
	}
	if filepath.Clean(oldHome) == filepath.Clean(newHome) {
		return true, nil
	}
	rel, err := filepath.Rel(filepath.Join(oldHome, "sessions"), src)
	if err != nil || strings.HasPrefix(rel, "..") {
		return false, fmt.Errorf("codex rollout outside sessions root: %s", src)
	}
	if err := copyRegularFileAtomic(src, filepath.Join(newHome, "sessions", rel)); err != nil {
		return false, err
	}
	return true, nil
}

// SeedCodexRollout copies the rollout being resumed from the credential
// home into the session's scratch CODEX_HOME. Same contract as
// MigrateCodexRollout (true = resumable there).
func SeedCodexRollout(credHome, scratchHome, threadID string) (bool, error) {
	return MigrateCodexRollout(credHome, scratchHome, threadID)
}

// SyncCodexHome copies the durable state a codex run wrote into its
// scratch CODEX_HOME back into the credential home before the scratch dir
// is removed:
//
//   - every rollout under <scratch>/sessions that is missing from, or
//     newer than, its counterpart under <cred>/sessions;
//   - auth.json, when codex refreshed it during the run (content differs
//     AND the scratch copy is newer). codex rotates OAuth refresh tokens,
//     so without this write-back the credential home keeps a refresh
//     token that was already spent and the login dies on its next use.
//
// Returns the thread id of the most recently written rollout in the
// scratch home — the scratch home is private to one opendray session, so
// that is this session's conversation. "" when codex wrote none.
// Best-effort: per-file failures are skipped so one bad file can't block
// the rest of the sync.
func SyncCodexHome(scratchHome, credHome string) string {
	if scratchHome == "" || credHome == "" || filepath.Clean(scratchHome) == filepath.Clean(credHome) {
		return ""
	}
	root := filepath.Join(scratchHome, "sessions")
	var newestID string
	var newestT int64
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !d.Type().IsRegular() {
			return nil
		}
		name := d.Name()
		if !strings.HasPrefix(name, "rollout-") || !strings.HasSuffix(name, ".jsonl") {
			return nil
		}
		srcInfo, err := os.Lstat(path)
		if err != nil {
			return nil
		}
		if t := srcInfo.ModTime().UnixNano(); newestID == "" || t > newestT {
			newestID, newestT = codexSessionIDFromName(name), t
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		dst := filepath.Join(credHome, "sessions", rel)
		if dstInfo, err := os.Stat(dst); err == nil &&
			dstInfo.Size() == srcInfo.Size() && !srcInfo.ModTime().After(dstInfo.ModTime()) {
			return nil // already in sync
		}
		_ = copyRegularFileAtomic(path, dst)
		return nil
	})
	syncCodexAuthBack(scratchHome, credHome)
	return newestID
}

// syncCodexAuthBack writes <scratch>/auth.json back to <cred>/auth.json
// when codex refreshed (or created) it during the run. A copy made at spawn
// is byte-identical to the credential home's file, so an untouched login
// is never rewritten; a login another session refreshed meanwhile is newer
// than our scratch copy and is left alone.
func syncCodexAuthBack(scratchHome, credHome string) {
	src := filepath.Join(scratchHome, "auth.json")
	dst := filepath.Join(credHome, "auth.json")
	srcInfo, err := os.Lstat(src)
	if err != nil || !srcInfo.Mode().IsRegular() || srcInfo.Size() == 0 {
		return
	}
	if dstInfo, err := os.Lstat(dst); err == nil {
		if !dstInfo.Mode().IsRegular() {
			return // never write through a symlink planted in the credential home
		}
		a, errA := os.ReadFile(src)
		b, errB := os.ReadFile(dst)
		if errA != nil || errB != nil || bytes.Equal(a, b) {
			return
		}
		if !srcInfo.ModTime().After(dstInfo.ModTime()) {
			return
		}
	}
	_ = copyRegularFileAtomic(src, dst)
}

// codexMtime returns a file's modification time as a comparable int64; a
// missing or unreadable file sorts oldest.
func codexMtime(path string) int64 {
	st, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return st.ModTime().UnixNano()
}

// copyRegularFileAtomic copies src to dst via a temp file + rename,
// creating dst's parent dirs (0700) and the file 0600, and carrying src's
// mtime over so later "is the copy newer?" checks compare like with like.
// Refuses symlinked or non-regular sources.
func copyRegularFileAtomic(src, dst string) error {
	st, err := os.Lstat(src)
	if err != nil {
		return fmt.Errorf("lstat src: %w", err)
	}
	if st.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to copy symlinked file at %s", src)
	}
	if !st.Mode().IsRegular() {
		return fmt.Errorf("refusing to copy non-regular file at %s", src)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return fmt.Errorf("mkdir dst dir: %w", err)
	}
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open src: %w", err)
	}
	defer in.Close()
	tmp, err := os.CreateTemp(filepath.Dir(dst), ".codex-copy-*")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op after a successful rename
	if _, err := io.Copy(tmp, in); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("copy bytes: %w", err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	_ = os.Chtimes(tmpName, st.ModTime(), st.ModTime())
	if err := os.Rename(tmpName, dst); err != nil {
		return fmt.Errorf("install %s: %w", dst, err)
	}
	return nil
}

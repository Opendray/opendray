// Package checkpoint captures and restores the context-level state of a
// session: the working tree's uncommitted git diff, untracked (non-ignored)
// files, and the operator input history. It is priority ② of the
// state-machine-hardening plan — context backup/restore layered on the now
// concurrency-safe, audit-tracked session state machine.
//
// Design (frozen 2026-07-14, see docs/design/session-state-machine.md):
//   - git-aware, .gitignore-respecting: only `git diff HEAD` (tracked
//     changes) and untracked-but-not-ignored files are captured, so
//     node_modules / build artifacts / secrets under .gitignore are never
//     copied. A non-git cwd records metadata only.
//   - filesystem + DB-metadata split: the heavy payload lands under a
//     per-checkpoint dir on disk; the DB row is the queryable manifest.
//   - triggers: on session interruption (gateway shutdown) and a manual API.
package checkpoint

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"
)

// Trigger records what caused a checkpoint to be captured.
type Trigger string

const (
	// TriggerInterrupted — captured because the gateway is shutting down
	// and the session's live work is about to be interrupted.
	TriggerInterrupted Trigger = "interrupted"
	// TriggerManual — captured on an explicit operator/API request.
	TriggerManual Trigger = "manual"
)

// Valid reports whether t is a known trigger.
func (t Trigger) Valid() bool {
	return t == TriggerInterrupted || t == TriggerManual
}

// Capture caps. Bound the on-disk payload so a checkpoint can never run
// away with disk (a huge generated diff, a stray large untracked file, a
// directory of thousands of untracked files). When a cap clips the capture
// the checkpoint is still written and its Truncated flag is set.
const (
	// MaxDiffBytes bounds the stored `git diff HEAD` output.
	MaxDiffBytes = 5 << 20 // 5 MiB
	// MaxUntrackedFileBytes skips any single untracked file larger than this.
	MaxUntrackedFileBytes = 1 << 20 // 1 MiB
	// MaxUntrackedTotalBytes bounds the sum of copied untracked files.
	MaxUntrackedTotalBytes = 20 << 20 // 20 MiB
	// MaxUntrackedFiles bounds how many untracked files are copied.
	MaxUntrackedFiles = 500
	// InputHistoryRingBytes is the per-session input-history ring capacity
	// the session manager keeps; the tail is what a checkpoint stores.
	InputHistoryRingBytes = 64 << 10 // 64 KiB
)

// On-disk layout under a checkpoint's storage dir.
const (
	fileDiff     = "uncommitted.diff"
	fileInput    = "input_history.log"
	fileMeta     = "meta.json"
	dirUntracked = "untracked"
)

// Checkpoint is the manifest for one captured context snapshot. The heavy
// payload lives under StoragePath; this is the DB row.
type Checkpoint struct {
	ID             string    `json:"id"`
	SessionID      string    `json:"session_id"`
	CreatedAt      time.Time `json:"created_at"`
	Trigger        Trigger   `json:"trigger"`
	Cwd            string    `json:"cwd"`
	IsGit          bool      `json:"is_git"`
	GitHead        string    `json:"git_head,omitempty"`
	GitDirty       bool      `json:"git_dirty"`
	DiffBytes      int64     `json:"diff_bytes"`
	UntrackedFiles int       `json:"untracked_files"`
	UntrackedBytes int64     `json:"untracked_bytes"`
	InputBytes     int64     `json:"input_bytes"`
	Truncated      bool      `json:"truncated"`
	StoragePath    string    `json:"-"`
	Note           string    `json:"note,omitempty"`
}

// Errors surfaced by the service/handler layers.
var (
	ErrNotFound       = errors.New("checkpoint not found")
	ErrInvalidTrigger = errors.New("invalid checkpoint trigger")
	ErrNoStorageDir   = errors.New("checkpoint storage dir not configured")
)

func newID() string {
	var b [9]byte
	if _, err := rand.Read(b[:]); err != nil {
		t := time.Now().UnixNano()
		for i := range b {
			b[i] = byte(t >> (i * 8))
		}
	}
	return "chk_" + base64.RawURLEncoding.EncodeToString(b[:])
}

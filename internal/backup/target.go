package backup

import (
	"context"
	"io"
)

// TargetRef is a stable reference to a blob written into a
// BackupTarget. It is persisted in backups.target_path / .sha256 /
// .bytes and used by Get/Delete to round-trip the same blob.
//
// Path is target-relative and never absolute — LocalTarget joins it
// onto its root, SMBTarget joins it onto its share, etc.
type TargetRef struct {
	Target string `json:"target"`
	Path   string `json:"path"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

// BackupTarget abstracts a place where backup blobs live.
//
// Put streams `r` into the target at `path`. Implementations must:
//   - write atomically (rename-over or transactional commit) so a
//     crash mid-write never leaves a half-written blob visible at
//     the final path,
//   - compute SHA-256 + byte count over the bytes actually written
//     and return them in the TargetRef,
//   - reject `path` values that try to escape the target's root
//     (e.g. "..", absolute paths) with ErrTargetRejectedPath.
//
// `size` is a hint for pre-allocation; -1 means unknown.
type BackupTarget interface {
	Name() string
	Kind() TargetKind
	Put(ctx context.Context, path string, r io.Reader, size int64) (TargetRef, error)
	Get(ctx context.Context, ref TargetRef) (io.ReadCloser, error)
	Delete(ctx context.Context, ref TargetRef) error
	// HealthCheck verifies the target is reachable + writable. It
	// is called from the connection-test button in the UI and from
	// the scheduler before each run.
	HealthCheck(ctx context.Context) error
}

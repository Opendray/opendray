package backup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ErrTargetRejectedPath is returned when a relative path tries to
// escape the target's root, contains a null byte, or is otherwise
// suspicious. Implementations of BackupTarget.Put must check.
var ErrTargetRejectedPath = errors.New("backup target: rejected path")

// LocalTarget stores backup blobs under a directory on the same
// machine running opendray. It is the default target and the only
// one available with no extra config.
//
// On Put, data is streamed into a sibling temp file (".<basename>.
// part") and atomically renamed onto the final path on success;
// crash-during-write leaves only the temp file behind, which the
// next scheduler tick may garbage-collect (TODO once retention is
// in place).
type LocalTarget struct {
	id   string
	root string
}

// NewLocalTarget creates the root directory if it doesn't exist
// (mode 0700, since blobs may be unencrypted in dev).
func NewLocalTarget(id, root string) (*LocalTarget, error) {
	if id == "" {
		return nil, errors.New("local target: id required")
	}
	if root == "" {
		return nil, errors.New("local target: root required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("local target: abs(%q): %w", root, err)
	}
	if err := os.MkdirAll(abs, 0o700); err != nil {
		return nil, fmt.Errorf("local target: mkdir %s: %w", abs, err)
	}
	return &LocalTarget{id: id, root: abs}, nil
}

func (t *LocalTarget) Name() string     { return t.id }
func (t *LocalTarget) Kind() TargetKind { return TargetLocal }
func (t *LocalTarget) Root() string     { return t.root }

func (t *LocalTarget) resolve(p string) (string, error) {
	if p == "" {
		return "", fmt.Errorf("%w: empty path", ErrTargetRejectedPath)
	}
	if strings.ContainsRune(p, 0) {
		return "", fmt.Errorf("%w: null byte", ErrTargetRejectedPath)
	}
	if filepath.IsAbs(p) {
		return "", fmt.Errorf("%w: absolute path %q", ErrTargetRejectedPath, p)
	}
	cleaned := filepath.Clean(p)
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: escapes root: %q", ErrTargetRejectedPath, p)
	}
	full := filepath.Join(t.root, cleaned)
	// Final containment check after Clean+Join — defends against
	// symlink games where root itself contains a symlink.
	rel, err := filepath.Rel(t.root, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: %q resolves outside root", ErrTargetRejectedPath, p)
	}
	return full, nil
}

func (t *LocalTarget) Put(ctx context.Context, path string, r io.Reader, _ int64) (TargetRef, error) {
	full, err := t.resolve(path)
	if err != nil {
		return TargetRef{}, err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o700); err != nil {
		return TargetRef{}, fmt.Errorf("local target: mkdir parent: %w", err)
	}
	tmp := filepath.Join(filepath.Dir(full), "."+filepath.Base(full)+".part")

	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return TargetRef{}, fmt.Errorf("local target: open tmp: %w", err)
	}
	// Cleanup tmp on any failure path; rename below replaces this.
	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(tmp)
		}
	}()

	hasher := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(f, hasher), &ctxReader{ctx: ctx, r: r})
	if cerr := f.Close(); cerr != nil && copyErr == nil {
		copyErr = cerr
	}
	if copyErr != nil {
		return TargetRef{}, fmt.Errorf("local target: write: %w", copyErr)
	}

	if err := os.Rename(tmp, full); err != nil {
		return TargetRef{}, fmt.Errorf("local target: rename: %w", err)
	}
	committed = true

	return TargetRef{
		Target: t.id,
		Path:   filepath.ToSlash(filepath.Clean(path)),
		Bytes:  written,
		SHA256: hex.EncodeToString(hasher.Sum(nil)),
	}, nil
}

func (t *LocalTarget) Get(_ context.Context, ref TargetRef) (io.ReadCloser, error) {
	full, err := t.resolve(ref.Path)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(full)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrBackupNotFound
		}
		return nil, fmt.Errorf("local target: open: %w", err)
	}
	return f, nil
}

func (t *LocalTarget) Delete(_ context.Context, ref TargetRef) error {
	full, err := t.resolve(ref.Path)
	if err != nil {
		return err
	}
	if err := os.Remove(full); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil // idempotent
		}
		return fmt.Errorf("local target: delete: %w", err)
	}
	return nil
}

func (t *LocalTarget) HealthCheck(_ context.Context) error {
	probe := filepath.Join(t.root, ".healthcheck")
	if err := os.WriteFile(probe, []byte("ok"), 0o600); err != nil {
		return fmt.Errorf("local target: write probe: %w", err)
	}
	if err := os.Remove(probe); err != nil {
		return fmt.Errorf("local target: remove probe: %w", err)
	}
	return nil
}

// ctxReader cancels io.Copy when the context is done. io.Copy
// otherwise has no native ctx hook.
type ctxReader struct {
	ctx context.Context
	r   io.Reader
}

func (c *ctxReader) Read(p []byte) (int, error) {
	if err := c.ctx.Err(); err != nil {
		return 0, err
	}
	return c.r.Read(p)
}

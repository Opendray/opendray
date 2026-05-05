package backup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"
)

// RcloneConfig drives a passthrough target backed by an external
// `rclone` binary. The operator pre-configures their remote in
// rclone (`rclone config`) and points us at it by name.
//
// This buys access to ≈70 backends (Google Drive, OneDrive,
// Dropbox, 百度网盘, 阿里云盘, iCloud-via-WebDAV, Backblaze B2,
// Mega, pCloud, Hetzner Storage Box, Wasabi, Google Cloud Storage,
// Azure Blob, etc.) without us writing per-backend code.
//
// Tradeoffs vs. native targets:
//   - Requires rclone binary on the opendray host (free, single
//     static binary, but external dependency).
//   - Streaming is via `rclone rcat` (Put) and `rclone cat` (Get) —
//     reliable but spawns a subprocess per op.
//   - HealthCheck uses `rclone lsd` to confirm auth + reachability.
type RcloneConfig struct {
	Remote       string // rclone remote name (e.g. "gdrive" or "r2-prod")
	PathPrefix   string // sub-folder under the remote root
	BinaryPath   string // optional override; default is exec.LookPath("rclone")
	ConfigPath   string // optional --config <path> override; default rclone's own
	ExtraArgs    []string // optional global flags (e.g. ["--bwlimit=10M"])
}

// RcloneTarget shells out to an external rclone binary.
type RcloneTarget struct {
	id        string
	cfg       RcloneConfig
	dialTO    time.Duration
	requestTO time.Duration
}

// NewRcloneTarget validates the binary is present and the remote
// looks like a non-empty name. It does NOT verify the remote
// actually works — that's HealthCheck's job.
func NewRcloneTarget(id string, cfg RcloneConfig) (*RcloneTarget, error) {
	if id == "" {
		return nil, errors.New("rclone target: id required")
	}
	if cfg.Remote == "" {
		return nil, errors.New("rclone target: remote name required (e.g. 'gdrive')")
	}
	if strings.ContainsAny(cfg.Remote, ":/\\ \t\n") {
		return nil, errors.New("rclone target: remote name must not contain ':', '/', whitespace")
	}
	bin := cfg.BinaryPath
	if bin == "" {
		resolved, err := exec.LookPath("rclone")
		if err != nil {
			return nil, fmt.Errorf("rclone target: binary not found on PATH (install from https://rclone.org or set binary_path): %w", err)
		}
		bin = resolved
	} else {
		if _, err := exec.LookPath(bin); err != nil {
			return nil, fmt.Errorf("rclone target: binary at %q unusable: %w", bin, err)
		}
	}
	cfg.BinaryPath = bin
	cfg.PathPrefix = strings.Trim(cfg.PathPrefix, "/")
	return &RcloneTarget{
		id:        id,
		cfg:       cfg,
		dialTO:    30 * time.Second,
		requestTO: 30 * time.Minute,
	}, nil
}

func (t *RcloneTarget) Name() string     { return t.id }
func (t *RcloneTarget) Kind() TargetKind { return TargetRclone }

// remotePath assembles "remote:prefix/p" — rclone's URI shape.
func (t *RcloneTarget) remotePath(p string) (string, error) {
	if p == "" {
		return "", fmt.Errorf("%w: empty path", ErrTargetRejectedPath)
	}
	if strings.ContainsRune(p, 0) {
		return "", fmt.Errorf("%w: null byte", ErrTargetRejectedPath)
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == ".." {
			return "", fmt.Errorf("%w: traversal segment in %q", ErrTargetRejectedPath, p)
		}
	}
	cleaned := path.Clean("/" + p)
	cleaned = strings.TrimPrefix(cleaned, "/")
	if t.cfg.PathPrefix != "" {
		cleaned = t.cfg.PathPrefix + "/" + cleaned
	}
	return t.cfg.Remote + ":" + cleaned, nil
}

func (t *RcloneTarget) baseArgs() []string {
	args := []string{}
	if t.cfg.ConfigPath != "" {
		args = append(args, "--config", t.cfg.ConfigPath)
	}
	args = append(args, t.cfg.ExtraArgs...)
	return args
}

func (t *RcloneTarget) Put(ctx context.Context, p string, r io.Reader, _ int64) (TargetRef, error) {
	dest, err := t.remotePath(p)
	if err != nil {
		return TargetRef{}, err
	}

	tctx, cancel := context.WithTimeout(ctx, t.requestTO)
	defer cancel()

	args := append(t.baseArgs(), "rcat", dest)
	cmd := exec.CommandContext(tctx, t.cfg.BinaryPath, args...)

	hasher := sha256.New()
	tee := io.TeeReader(&ctxReader{ctx: ctx, r: r}, hasher)

	cmd.Stdin = tee
	stderrBuf := newCappedBuf(8 << 10)
	cmd.Stderr = stderrBuf

	// rclone rcat doesn't tell us bytes-written. We track via tee.
	// To get the count we wrap tee in a counting writer.
	counter := &countWriter{}
	cmd.Stdin = io.TeeReader(tee, counter)

	if err := cmd.Run(); err != nil {
		return TargetRef{}, fmt.Errorf("rclone rcat: %w; stderr: %s", err, stderrBuf.String())
	}

	return TargetRef{
		Target: t.id,
		Path:   p,
		Bytes:  counter.n,
		SHA256: hex.EncodeToString(hasher.Sum(nil)),
	}, nil
}

func (t *RcloneTarget) Get(ctx context.Context, ref TargetRef) (io.ReadCloser, error) {
	src, err := t.remotePath(ref.Path)
	if err != nil {
		return nil, err
	}
	args := append(t.baseArgs(), "cat", src)
	cmd := exec.CommandContext(ctx, t.cfg.BinaryPath, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("rclone cat stdout: %w", err)
	}
	stderrBuf := newCappedBuf(8 << 10)
	cmd.Stderr = stderrBuf
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("rclone cat start: %w", err)
	}
	return &rcloneReader{
		stdout:    stdout,
		stderrBuf: stderrBuf,
		wait:      cmd.Wait,
	}, nil
}

type rcloneReader struct {
	stdout    io.ReadCloser
	stderrBuf *cappedBuf
	wait      func() error
	closed    bool
}

func (r *rcloneReader) Read(p []byte) (int, error) {
	n, err := r.stdout.Read(p)
	if errors.Is(err, io.EOF) {
		// Surface "file not found" stderr lines as a distinct error.
		if !r.closed {
			_ = r.stdout.Close()
			r.closed = true
			if waitErr := r.wait(); waitErr != nil {
				stderr := r.stderrBuf.String()
				if strings.Contains(stderr, "not found") || strings.Contains(stderr, "doesn't exist") {
					return n, ErrBackupNotFound
				}
				return n, fmt.Errorf("rclone cat exit: %w; stderr: %s", waitErr, stderr)
			}
		}
		return n, io.EOF
	}
	return n, err
}

func (r *rcloneReader) Close() error {
	if r.closed {
		return nil
	}
	r.closed = true
	_ = r.stdout.Close()
	return r.wait()
}

func (t *RcloneTarget) Delete(ctx context.Context, ref TargetRef) error {
	dest, err := t.remotePath(ref.Path)
	if err != nil {
		return err
	}
	tctx, cancel := context.WithTimeout(ctx, t.requestTO)
	defer cancel()
	args := append(t.baseArgs(), "deletefile", dest)
	cmd := exec.CommandContext(tctx, t.cfg.BinaryPath, args...)
	stderrBuf := newCappedBuf(8 << 10)
	cmd.Stderr = stderrBuf
	if err := cmd.Run(); err != nil {
		stderr := stderrBuf.String()
		if strings.Contains(stderr, "not found") || strings.Contains(stderr, "doesn't exist") {
			return nil // idempotent
		}
		return fmt.Errorf("rclone deletefile: %w; stderr: %s", err, stderr)
	}
	return nil
}

func (t *RcloneTarget) HealthCheck(ctx context.Context) error {
	tctx, cancel := context.WithTimeout(ctx, t.dialTO)
	defer cancel()

	// `lsd <remote>:` lists top-level dirs — confirms auth + reachable.
	probeRoot := t.cfg.Remote + ":"
	if t.cfg.PathPrefix != "" {
		probeRoot = t.cfg.Remote + ":" + t.cfg.PathPrefix
	}
	args := append(t.baseArgs(), "lsd", probeRoot, "--max-depth=1")
	cmd := exec.CommandContext(tctx, t.cfg.BinaryPath, args...)
	stderrBuf := newCappedBuf(8 << 10)
	cmd.Stderr = stderrBuf
	cmd.Stdout = io.Discard
	if err := cmd.Run(); err != nil {
		stderr := stderrBuf.String()
		// "directory not found" with PathPrefix is recoverable —
		// the prefix may not exist yet. Try mkdir then re-probe.
		if t.cfg.PathPrefix != "" &&
			(strings.Contains(stderr, "directory not found") || strings.Contains(stderr, "doesn't exist")) {
			mkArgs := append(t.baseArgs(), "mkdir", probeRoot)
			mkCmd := exec.CommandContext(tctx, t.cfg.BinaryPath, mkArgs...)
			mkCmd.Stderr = io.Discard
			if mkErr := mkCmd.Run(); mkErr != nil {
				return fmt.Errorf("rclone probe: mkdir %s: %w; stderr: %s", probeRoot, mkErr, stderr)
			}
			return nil
		}
		return fmt.Errorf("rclone lsd %s: %w; stderr: %s", probeRoot, err, stderr)
	}
	_ = os.Stdout // silences linter; unused import path
	return nil
}

// countWriter counts bytes written. Used to derive Bytes when the
// upstream tool (rclone rcat) doesn't surface a written-count.
type countWriter struct {
	n int64
}

func (c *countWriter) Write(p []byte) (int, error) {
	c.n += int64(len(p))
	return len(p), nil
}

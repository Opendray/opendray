// Package install implements the plugin install lifecycle (T6).
//
// The install package owns four concerns:
//   - source.go      — fetching a bundle directory from a URL (local / https / marketplace)
//   - hash.go        — sha256 helpers: file-content + canonical-manifest
//   - consent.go     — in-memory pending-install store gated by a hex token
//   - install.go     — the Installer, which ties sources + DB + runtime together
//
// Scope (M1): only "local:" + bare absolute paths work. https:// and
// marketplace:// sources parse but their Fetch returns ErrNotImplemented —
// M4 will wire them. All mutations go through Installer, never directly on
// the store or runtime, to keep atomicity + audit in one place.
package install

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// ErrNotImplemented is returned by M4-only Source.Fetch implementations.
var ErrNotImplemented = errors.New("install: source scheme not implemented in M1")

// Source knows how to produce a bundle directory for installation.
// Fetch returns a local path containing manifest.json at its root,
// plus a cleanup func the caller must run when done.
type Source interface {
	Fetch(ctx context.Context) (bundlePath string, cleanup func(), err error)
	Describe() string
}

// ParseSource dispatches on the URL scheme. Accepted inputs:
//
//   - "local:/abs/path"              → LocalSource (absolute path required)
//   - "/abs/path"                    → LocalSource (bare absolute path)
//   - "https://…"                    → HTTPSSource (Fetch returns ErrNotImplemented in M1)
//   - "marketplace://…"              → MarketplaceSource (ditto)
//
// Relative paths, unknown schemes, and the empty string return an error.
func ParseSource(raw string) (Source, error) {
	if raw == "" {
		return nil, fmt.Errorf("install: empty source string")
	}

	// Bare absolute path is LocalSource — no URL parsing needed.
	if filepath.IsAbs(raw) {
		return LocalSource{Path: filepath.Clean(raw)}, nil
	}

	// Explicit "local:/…" form. We peel off the scheme and require the
	// remainder to be an absolute path.
	if strings.HasPrefix(raw, "local:") {
		rest := strings.TrimPrefix(raw, "local:")
		if !filepath.IsAbs(rest) {
			return nil, fmt.Errorf("install: local: requires absolute path, got %q", rest)
		}
		return LocalSource{Path: filepath.Clean(rest)}, nil
	}

	// url.Parse does not error on many malformed inputs; we explicitly
	// require a scheme component before routing.
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" {
		return nil, fmt.Errorf("install: unrecognised source %q", raw)
	}

	switch u.Scheme {
	case "https":
		return HTTPSSource{URL: raw}, nil
	case "marketplace":
		return MarketplaceSource{Raw: raw}, nil
	default:
		return nil, fmt.Errorf("install: unsupported scheme %q", u.Scheme)
	}
}

// ─── LocalSource ────────────────────────────────────────────────────────────

// LocalSource reads a bundle from a local filesystem directory. We COPY
// the directory into a temp staging area so the caller can safely delete
// or rewrite the original without corrupting the in-flight install.
type LocalSource struct {
	Path string
}

// Fetch copies the directory at s.Path into a fresh os.MkdirTemp dir and
// returns that copy's path. cleanup removes the copy; the caller must call
// it when done with the bundle (on success after Confirm's Rename, on
// failure after Stage's error handling).
func (s LocalSource) Fetch(ctx context.Context) (string, func(), error) {
	if s.Path == "" {
		return "", nil, fmt.Errorf("install: LocalSource.Path is empty")
	}
	fi, err := os.Stat(s.Path)
	if err != nil {
		return "", nil, fmt.Errorf("install: stat local source %q: %w", s.Path, err)
	}
	if !fi.IsDir() {
		return "", nil, fmt.Errorf("install: local source %q is not a directory", s.Path)
	}

	// Staging lives under os.TempDir() — the Installer will move it to
	// DataDir during Confirm. Using TempDir keeps the fetch stage
	// filesystem-independent of DataDir so callers can discard stale
	// bundles via os.RemoveAll without worrying about cross-device rename.
	staging, err := os.MkdirTemp("", "opendray-fetch-*")
	if err != nil {
		return "", nil, fmt.Errorf("install: mkdir fetch staging: %w", err)
	}

	cleanup := func() { _ = os.RemoveAll(staging) }

	if err := copyTree(ctx, s.Path, staging); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("install: copy tree %q: %w", s.Path, err)
	}
	return staging, cleanup, nil
}

// Describe returns a human-readable label used for audit hashing.
func (s LocalSource) Describe() string { return "local:" + s.Path }

// ─── HTTPSSource (M4) ───────────────────────────────────────────────────────

// HTTPSSource will eventually download + verify a signed bundle from the
// marketplace CDN. In M1 Fetch returns ErrNotImplemented so callers fail
// loudly rather than silently.
type HTTPSSource struct {
	URL string
}

// Fetch is not implemented in M1. Returns ErrNotImplemented.
func (s HTTPSSource) Fetch(_ context.Context) (string, func(), error) {
	return "", nil, fmt.Errorf("install: https source %q: %w", s.URL, ErrNotImplemented)
}

// Describe returns a human-readable label.
func (s HTTPSSource) Describe() string { return s.URL }

// ─── MarketplaceSource (M4) ─────────────────────────────────────────────────

// MarketplaceSource resolves a marketplace:// URL to a download URL + hash
// via the marketplace index. Not wired in M1.
type MarketplaceSource struct {
	Raw string
}

// Fetch is not implemented in M1. Returns ErrNotImplemented.
func (s MarketplaceSource) Fetch(_ context.Context) (string, func(), error) {
	return "", nil, fmt.Errorf("install: marketplace source %q: %w", s.Raw, ErrNotImplemented)
}

// Describe returns a human-readable label.
func (s MarketplaceSource) Describe() string { return s.Raw }

// ─── copyTree ───────────────────────────────────────────────────────────────

// copyTree recursively copies srcDir into dstDir using filepath.Walk and
// io.Copy. We never shell out (see T6 spec "do not shell out for
// extraction"). Symlinks are followed for files and preserved for dirs; we
// do not try to faithfully reproduce permission bits beyond the low 9 bits
// (Unix mode) because plugin bundles are source-only.
func copyTree(ctx context.Context, srcDir, dstDir string) error {
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		// Honour caller-initiated cancellation on every node so huge bundles
		// don't delay shutdown.
		if err := ctx.Err(); err != nil {
			return err
		}

		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dstDir, rel)

		switch {
		case info.IsDir():
			return os.MkdirAll(target, info.Mode().Perm())
		case info.Mode().IsRegular():
			return copyFile(path, target, info.Mode().Perm())
		default:
			// Skip sockets, devices, etc. Plugins should never ship these,
			// and silently dropping them keeps the copy robust.
			return nil
		}
	})
}

// copyFile copies src to dst with the given mode. The destination is created
// fresh; if it exists it is truncated.
func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	// Ensure parent dir exists.
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

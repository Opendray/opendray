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
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
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

// ─── HTTPSSource (M4.1 T4) ──────────────────────────────────────────────────

// HTTPSSource downloads a plugin bundle zip from an HTTP(S) URL,
// verifies its SHA-256 against ExpectedSHA256, and extracts it to
// a staging directory ready for Installer.Stage to move into the
// data dir.
//
// The caller (gateway's install handler in T9) populates every
// field from a market.Entry — the marketplace-fetched per-version
// JSON carries the artifact URL, size, and canonical hash.
//
// Security model:
//   - MaxBytes caps the download so an adversary can't eat unbounded
//     memory / disk on a compromised artifact URL.
//   - SHA-256 is checked AFTER the full bytes land on disk but
//     BEFORE extraction, so a mismatch never creates files.
//   - Zip entries are validated per-entry: no absolute paths, no
//     "..", no symlinks, no setuid/setgid. A hostile zip can't
//     climb out of the staging dir.
//   - Content-Type is left unenforced — GitHub Releases serves
//     octet-stream, CDNs vary, and sha256 verification already
//     proves we got the intended bytes.
type HTTPSSource struct {
	// URL is the HTTPS endpoint to GET. HTTP is accepted for tests
	// that use a httptest.Server; production wiring requires HTTPS.
	URL string

	// ExpectedSHA256 is the canonical lowercase-hex digest from the
	// marketplace per-version record. Required — a zero-length
	// value fails Fetch with an obvious error so callers can't
	// accidentally skip verification.
	ExpectedSHA256 string

	// ExpectedSize, when non-zero, is the byte count the registry
	// advertised. Fetch compares it against the Content-Length
	// header (when present) for an early reject. The post-download
	// size check against MaxBytes is the authoritative cap.
	ExpectedSize int64

	// MaxBytes is the download ceiling. Zero means
	// defaultHTTPSMaxBytes (200 MiB, matching 09-marketplace.md's
	// host-form cap).
	MaxBytes int64

	// HTTPClient overrides the default client. Zero uses a
	// DefaultTransport wrapped with a 5-minute timeout — large
	// bundles over slow links need the headroom.
	HTTPClient *http.Client
}

// defaultHTTPSMaxBytes is the per-bundle download cap. Matches the
// spec's host-form upper bound (09-marketplace.md §Plugin release
// artifact); webview-only bundles land well below this so one
// constant is enough.
const defaultHTTPSMaxBytes int64 = 200 << 20

// defaultHTTPSTimeout is the per-request ceiling. Big enough for a
// 200 MiB zip over a mediocre link without being so long that a
// silent stall hangs the install flow indefinitely.
const defaultHTTPSTimeout = 5 * time.Minute

// Fetch downloads the artifact into a temporary file, verifies its
// SHA-256, extracts the archive into a fresh staging directory,
// and returns the staging path plus a cleanup func the caller must
// invoke when done.
//
// Errors are wrapped with sufficient context to diagnose without
// inspecting the URL — the download target is included in every
// error message.
func (s HTTPSSource) Fetch(ctx context.Context) (string, func(), error) {
	if s.URL == "" {
		return "", nil, fmt.Errorf("install: HTTPSSource.URL is empty")
	}
	if s.ExpectedSHA256 == "" {
		return "", nil, fmt.Errorf("install: HTTPSSource.ExpectedSHA256 is required for %s", s.URL)
	}
	maxBytes := s.MaxBytes
	if maxBytes <= 0 {
		maxBytes = defaultHTTPSMaxBytes
	}
	client := s.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: defaultHTTPSTimeout}
	}

	// 1) Issue the GET.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.URL, nil)
	if err != nil {
		return "", nil, fmt.Errorf("install: build request %s: %w", s.URL, err)
	}
	req.Header.Set("User-Agent", "opendray/install-https")
	resp, err := client.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("install: GET %s: %w", s.URL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", nil, fmt.Errorf("install: GET %s: HTTP %d", s.URL, resp.StatusCode)
	}
	// Early-reject mismatched advertised size. Server lies are
	// possible; the MaxBytes stream check is the authoritative cap.
	if s.ExpectedSize > 0 && resp.ContentLength > 0 && resp.ContentLength != s.ExpectedSize {
		return "", nil, fmt.Errorf("install: GET %s: size %d != expected %d",
			s.URL, resp.ContentLength, s.ExpectedSize)
	}
	if resp.ContentLength > maxBytes {
		return "", nil, fmt.Errorf("install: GET %s: Content-Length %d exceeds MaxBytes %d",
			s.URL, resp.ContentLength, maxBytes)
	}

	// 2) Stream to a temp file + sha256 in parallel. Using one
	// MultiWriter avoids a second pass over the bytes for hashing.
	tmp, err := os.CreateTemp("", "opendray-fetch-zip-*.zip")
	if err != nil {
		return "", nil, fmt.Errorf("install: create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	rmTmp := func() { _ = os.Remove(tmpPath) }
	hasher := sha256.New()
	limiter := &io.LimitedReader{R: resp.Body, N: maxBytes + 1}
	n, copyErr := io.Copy(io.MultiWriter(tmp, hasher), limiter)
	closeErr := tmp.Close()
	if copyErr != nil {
		rmTmp()
		return "", nil, fmt.Errorf("install: download %s: %w", s.URL, copyErr)
	}
	if closeErr != nil {
		rmTmp()
		return "", nil, fmt.Errorf("install: close temp %s: %w", s.URL, closeErr)
	}
	if n > maxBytes {
		rmTmp()
		return "", nil, fmt.Errorf("install: %s body exceeds MaxBytes %d", s.URL, maxBytes)
	}
	if s.ExpectedSize > 0 && n != s.ExpectedSize {
		rmTmp()
		return "", nil, fmt.Errorf("install: %s body %d != expected %d", s.URL, n, s.ExpectedSize)
	}
	gotHash := hex.EncodeToString(hasher.Sum(nil))
	if !strings.EqualFold(gotHash, s.ExpectedSHA256) {
		rmTmp()
		return "", nil, fmt.Errorf("install: %s sha256 mismatch: got %s want %s",
			s.URL, gotHash, s.ExpectedSHA256)
	}

	// 3) Extract into a staging directory. Zip validation runs
	// per-entry in extractZipBundle; on any security reject we
	// wipe both the tmp file and the partial extraction.
	staging, err := os.MkdirTemp("", "opendray-fetch-*")
	if err != nil {
		rmTmp()
		return "", nil, fmt.Errorf("install: mkdir staging: %w", err)
	}
	if err := extractZipBundle(ctx, tmpPath, staging); err != nil {
		rmTmp()
		_ = os.RemoveAll(staging)
		return "", nil, fmt.Errorf("install: extract %s: %w", s.URL, err)
	}
	rmTmp()

	cleanup := func() { _ = os.RemoveAll(staging) }
	return staging, cleanup, nil
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

// ─── extractZipBundle ───────────────────────────────────────────────────────

// extractZipBundle opens the zip at srcZip and extracts every entry
// into dstDir after validating its path + mode. Security rules:
//
//   - Entry names must not be absolute, contain "..", or escape dstDir.
//   - Symlinks are rejected outright — we never follow or recreate them.
//   - Executable mode bits are preserved (plugins with sidecar
//     binaries need +x) but setuid/setgid (0o4000 / 0o2000) are
//     stripped as defence-in-depth.
//   - A per-entry file-size cap is enforced so a zip-bomb with one
//     tiny header expanding to gigabytes aborts cleanly.
//
// All errors abort the extraction; the caller is responsible for
// wiping dstDir on failure.
func extractZipBundle(ctx context.Context, srcZip, dstDir string) error {
	const maxEntryBytes int64 = 200 << 20 // same ceiling as download cap

	r, err := zip.OpenReader(srcZip)
	if err != nil {
		return fmt.Errorf("zip open: %w", err)
	}
	defer r.Close()

	absDst, err := filepath.Abs(dstDir)
	if err != nil {
		return fmt.Errorf("abs dstDir: %w", err)
	}

	for _, zf := range r.File {
		if err := ctx.Err(); err != nil {
			return err
		}

		// Normalise to a forward-slash relative path and reject
		// any attempt to escape the destination.
		name := zf.Name
		if name == "" {
			return fmt.Errorf("zip entry: empty name")
		}
		if strings.ContainsRune(name, '\x00') {
			return fmt.Errorf("zip entry %q: null byte in name", name)
		}
		if filepath.IsAbs(name) || strings.HasPrefix(name, "/") {
			return fmt.Errorf("zip entry %q: absolute path not allowed", name)
		}
		// filepath.Clean + a prefix check catches every "..",
		// including nested ones like "a/../../b".
		cleaned := filepath.Clean(name)
		if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
			return fmt.Errorf("zip entry %q: escapes archive root", name)
		}
		target := filepath.Join(absDst, cleaned)
		if !strings.HasPrefix(target, absDst+string(filepath.Separator)) && target != absDst {
			return fmt.Errorf("zip entry %q: resolved path %q escapes %q", name, target, absDst)
		}

		mode := zf.Mode()
		switch {
		case mode&os.ModeSymlink != 0:
			return fmt.Errorf("zip entry %q: symlinks not allowed", name)
		case mode&os.ModeDevice != 0, mode&os.ModeNamedPipe != 0, mode&os.ModeSocket != 0:
			return fmt.Errorf("zip entry %q: irregular mode %v", name, mode)
		}

		// Strip setuid / setgid regardless; plugin bundles never
		// need them and the marketplace CI already scans for them,
		// so this is belt-and-braces.
		perm := mode.Perm() &^ (os.ModeSetuid | os.ModeSetgid)

		if zf.FileInfo().IsDir() || strings.HasSuffix(name, "/") {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("mkdir %q: %w", target, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("mkdir %q: %w", filepath.Dir(target), err)
		}

		rc, err := zf.Open()
		if err != nil {
			return fmt.Errorf("open entry %q: %w", name, err)
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm)
		if err != nil {
			rc.Close()
			return fmt.Errorf("create %q: %w", target, err)
		}
		n, copyErr := io.Copy(out, io.LimitReader(rc, maxEntryBytes+1))
		closeErr := out.Close()
		rc.Close()
		if copyErr != nil {
			return fmt.Errorf("extract %q: %w", name, copyErr)
		}
		if closeErr != nil {
			return fmt.Errorf("close %q: %w", target, closeErr)
		}
		if n > maxEntryBytes {
			return fmt.Errorf("zip entry %q: size exceeds %d bytes", name, maxEntryBytes)
		}
	}
	return nil
}

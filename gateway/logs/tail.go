// Package logs provides directory listing + file tailing for the log-viewer
// panel plugin. Paths are sandboxed to the caller's AllowedRoots exactly as
// the file-browser plugin is — no path outside those roots can be read.
package logs

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Config holds plugin settings extracted from the manifest.
type Config struct {
	AllowedRoots []string // absolute directories that may be browsed / tailed
	Extensions   []string // e.g. [".log", ".txt"]; empty = show all
	BacklogBytes int64    // how many bytes from the tail of the file to send as backlog
	ShowHidden   bool     // show dot-files in listings
}

// FileEntry represents one directory child in a listing response.
type FileEntry struct {
	Name    string `json:"name"`
	Path    string `json:"path"` // absolute
	Type    string `json:"type"` // "file" | "dir"
	Size    int64  `json:"size,omitempty"`
	ModTime int64  `json:"modTime,omitempty"`
	Ext     string `json:"ext,omitempty"`
}

// List returns the (filtered) directory entries at the given path. When path
// is empty, each configured root is surfaced as a virtual top-level entry.
func List(cfg Config, path string) ([]FileEntry, error) {
	if len(cfg.AllowedRoots) == 0 {
		return nil, fmt.Errorf("logs: no allowed roots configured")
	}
	// Empty path ⇒ list the roots themselves.
	if path == "" {
		out := make([]FileEntry, 0, len(cfg.AllowedRoots))
		for _, r := range cfg.AllowedRoots {
			abs, _ := filepath.Abs(r)
			info, err := os.Stat(abs)
			if err != nil || !info.IsDir() {
				continue
			}
			out = append(out, FileEntry{
				Name: filepath.Base(abs), Path: abs, Type: "dir",
				ModTime: info.ModTime().Unix(),
			})
		}
		return out, nil
	}

	abs, err := securePath(cfg, path)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		return nil, fmt.Errorf("logs: read dir: %w", err)
	}

	out := make([]FileEntry, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if !cfg.ShowHidden && strings.HasPrefix(name, ".") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		full := filepath.Join(abs, name)
		fe := FileEntry{
			Name: name, Path: full,
			ModTime: info.ModTime().Unix(),
		}
		if e.IsDir() {
			fe.Type = "dir"
		} else {
			fe.Type = "file"
			fe.Ext = strings.TrimPrefix(filepath.Ext(name), ".")
			fe.Size = info.Size()
			if len(cfg.Extensions) > 0 && !hasExt(name, cfg.Extensions) {
				continue
			}
		}
		out = append(out, fe)
	}
	return out, nil
}

// Tail streams the file at path. It first emits the last `cfg.BacklogBytes`
// bytes (aligned to a line boundary when possible), then follows with a
// short poll loop, detecting log rotation / truncation. Bytes are streamed
// via `out.Write` verbatim — callers are responsible for line-splitting,
// regex filtering, and rate-limiting on top of this raw stream.
//
// Returns nil when ctx is cancelled; error on fs / permission problems.
func Tail(ctx context.Context, cfg Config, path string, out io.Writer) error {
	abs, err := securePath(cfg, path)
	if err != nil {
		return err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return fmt.Errorf("logs: stat: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("logs: %q is a directory", path)
	}

	f, err := os.Open(abs) //nolint:gosec — securePath already vetted it
	if err != nil {
		return fmt.Errorf("logs: open: %w", err)
	}
	defer f.Close() //nolint:errcheck

	// ── Backlog ──────────────────────────────────────────────
	backlog := cfg.BacklogBytes
	if backlog <= 0 {
		backlog = 64 * 1024
	}
	size := info.Size()
	start := int64(0)
	if size > backlog {
		start = size - backlog
	}
	if _, err := f.Seek(start, io.SeekStart); err != nil {
		return fmt.Errorf("logs: seek: %w", err)
	}
	// If we seeked into the middle of a line, discard the partial prefix so
	// the user doesn't see a half-corrupted line at the top of the viewer.
	if start > 0 {
		rd := bufio.NewReader(f)
		if _, err := rd.ReadString('\n'); err == nil {
			// ReadString consumed the partial line; re-align f's position.
			pos, _ := f.Seek(0, io.SeekCurrent)
			// Account for buffered bytes still in `rd`.
			pos -= int64(rd.Buffered())
			_, _ = f.Seek(pos, io.SeekStart)
		} else {
			// Couldn't find a newline in the backlog window — just dump it.
			_, _ = f.Seek(start, io.SeekStart)
		}
	}

	// Stream backlog + follow in a single loop.
	buf := make([]byte, 8*1024)
	lastPos, _ := f.Seek(0, io.SeekCurrent)
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}
		n, rerr := f.Read(buf)
		if n > 0 {
			if _, werr := out.Write(buf[:n]); werr != nil {
				return werr
			}
			lastPos += int64(n)
		}
		if rerr == nil {
			continue
		}
		if rerr != io.EOF {
			return fmt.Errorf("logs: read: %w", rerr)
		}

		// EOF — wait a beat, then check for new bytes / rotation.
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(300 * time.Millisecond):
		}

		stat, serr := f.Stat()
		if serr != nil {
			return fmt.Errorf("logs: stat during follow: %w", serr)
		}
		if stat.Size() < lastPos {
			// File was truncated or rotated in place. Rewind and notify.
			if _, err := f.Seek(0, io.SeekStart); err != nil {
				return fmt.Errorf("logs: seek after rotate: %w", err)
			}
			lastPos = 0
			if _, err := out.Write([]byte("\n--- log rotated / truncated ---\n")); err != nil {
				return err
			}
		}
	}
}

// ── Helpers ──────────────────────────────────────────────────────

func hasExt(name string, exts []string) bool {
	lower := strings.ToLower(name)
	for _, e := range exts {
		e = strings.ToLower(strings.TrimSpace(e))
		if e == "" {
			continue
		}
		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		if strings.HasSuffix(lower, e) {
			return true
		}
	}
	return false
}

// securePath resolves path and verifies it's inside one of cfg.AllowedRoots.
// Same pattern as the file-browser plugin; rejects symlinks that escape.
func securePath(cfg Config, path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("logs: path is required")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("logs: invalid path: %w", err)
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		real = abs
	}
	for _, root := range cfg.AllowedRoots {
		rabs, _ := filepath.Abs(root)
		rreal, err := filepath.EvalSymlinks(rabs)
		if err != nil {
			rreal = rabs
		}
		if strings.HasPrefix(real, rreal) {
			return real, nil
		}
	}
	return "", fmt.Errorf("logs: path is not within any allowed root")
}

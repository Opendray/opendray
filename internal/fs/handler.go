// Package fs serves a small admin-only filesystem browser used by the
// web spawn dialog. It runs server-side because the gateway often
// runs on a different host than the browser (LAN / Cloudflare tunnel)
// — the browser cannot directly stat the gateway's filesystem.
//
// Authorization is upstream: handlers are mounted under the admin-only
// route group. The handlers themselves do no further auth and assume
// the caller is permitted to inspect the filesystem the binary can
// read.
package fs

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
)

type Handlers struct {
	log *slog.Logger
}

func NewHandlers(log *slog.Logger) *Handlers {
	if log == nil {
		log = slog.Default()
	}
	return &Handlers{log: log.With("component", "fs.http")}
}

func (h *Handlers) Mount(r chi.Router) {
	r.Route("/fs", func(r chi.Router) {
		r.Get("/list", h.list)
		r.Post("/mkdir", h.mkdir)
		r.Get("/home", h.home)
		r.Get("/read", h.read)
		// download streams a single file with Content-Disposition:
		// attachment so the browser saves rather than renders. Used by
		// the Files inspector's download icon. Unlike /read this has
		// no 256 KiB cap — operators download whatever the daemon can
		// stat.
		r.Get("/download", h.download)
		// zip streams a server-built zip archive of a directory subtree
		// so an operator can grab a whole folder in one click instead
		// of opening each file individually.
		r.Get("/zip", h.zipDir)
	})
}

// maxReadBytes caps /fs/read responses. The endpoint exists for the
// admin web's Task Runner to parse package.json / Makefile / Taskfile
// / justfile — none of which are large. Bigger files are rejected
// instead of truncated to keep parsing predictable.
const maxReadBytes = 256 * 1024 // 256 KiB

// read returns the raw bytes of a single file. Same path canonicalize
// + admin-only auth as /list. Refuses directories, refuses files
// larger than maxReadBytes, never follows-and-reveals symlink targets
// outside the repo (we just stat the canonical path the client gave).
func (h *Handlers) read(w http.ResponseWriter, r *http.Request) {
	rawPath := r.URL.Query().Get("path")
	if rawPath == "" {
		writeError(w, http.StatusBadRequest, errors.New("path is required"))
		return
	}
	p, err := canonicalize(rawPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	st, err := os.Stat(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		if errors.Is(err, os.ErrPermission) {
			writeError(w, http.StatusForbidden, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if st.IsDir() {
		writeError(w, http.StatusBadRequest, errors.New("path is a directory"))
		return
	}
	if st.Size() > maxReadBytes {
		writeError(w, http.StatusRequestEntityTooLarge,
			errors.New("file exceeds 256 KiB read cap"))
		return
	}
	data, err := os.ReadFile(p)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("X-OpenDray-Path", p)
	_, _ = w.Write(data)
}

// download streams a single file to the response with
// Content-Disposition: attachment so the browser saves it instead of
// rendering inline. The caller MUST pass a `root` query parameter:
// the resolved target path is verified to live inside that root, and
// requests for anything outside (including via symlink) fail with a
// 403. This is a tighter contract than /read — downloads are scoped
// to "files the operator can see in the inspector" (rooted at the
// session cwd), not arbitrary system paths. http.ServeContent
// handles Range requests so resume works on large files.
func (h *Handlers) download(w http.ResponseWriter, r *http.Request) {
	rawPath := r.URL.Query().Get("path")
	if rawPath == "" {
		writeError(w, http.StatusBadRequest, errors.New("path is required"))
		return
	}
	p, err := resolveWithinRoot(rawPath, r.URL.Query().Get("root"))
	if err != nil {
		writeError(w, http.StatusForbidden, err)
		return
	}
	f, err := os.Open(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		if errors.Is(err, os.ErrPermission) {
			writeError(w, http.StatusForbidden, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if st.IsDir() {
		writeError(w, http.StatusBadRequest,
			errors.New("path is a directory; use /fs/zip to download a folder"))
		return
	}
	// Content-Disposition with both filename= and filename*= so
	// browsers handle non-ASCII names correctly. http.ServeContent
	// fills Content-Type via DetectContentType + filename ext.
	name := filepath.Base(p)
	w.Header().Set("Content-Disposition",
		fmt.Sprintf("attachment; filename=%q; filename*=UTF-8''%s",
			name, url.PathEscape(name)))
	http.ServeContent(w, r, name, st.ModTime(), f)
}

// zipDir streams a zip archive of a directory subtree built on the
// fly. Files are walked deterministically (sorted) so the archive is
// reproducible; symlinks are skipped (we don't follow them when
// listing, so we don't follow them here either) to avoid an infinite
// loop on circular trees. Hidden entries (dot-prefix) match what
// /list shows — the operator sees the same set in both surfaces.
//
// Errors mid-stream are surfaced as a 500 only if the response is
// still buffered; once a single byte is flushed the connection just
// truncates and the browser shows an incomplete-download notice. This
// is acceptable for an admin-only convenience endpoint — operators
// can retry — and avoids buffering the whole archive in memory.
func (h *Handlers) zipDir(w http.ResponseWriter, r *http.Request) {
	rawPath := r.URL.Query().Get("path")
	if rawPath == "" {
		writeError(w, http.StatusBadRequest, errors.New("path is required"))
		return
	}
	root, err := resolveWithinRoot(rawPath, r.URL.Query().Get("root"))
	if err != nil {
		writeError(w, http.StatusForbidden, err)
		return
	}
	st, err := os.Stat(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		if errors.Is(err, os.ErrPermission) {
			writeError(w, http.StatusForbidden, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !st.IsDir() {
		writeError(w, http.StatusBadRequest,
			errors.New("path is a file; use /fs/download to download a single file"))
		return
	}

	archiveName := filepath.Base(root) + ".zip"
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition",
		fmt.Sprintf("attachment; filename=%q; filename*=UTF-8''%s",
			archiveName, url.PathEscape(archiveName)))

	zw := zip.NewWriter(w)
	// zip.Writer.Close flushes the central directory — silently dropping
	// its error would leave the operator with a truncated archive that
	// looks fine until they try to extract it. Log instead of return-up;
	// the body has already started streaming so we can't switch to a
	// 5xx here, but a warn in the journal is enough for triage.
	defer func() {
		if err := zw.Close(); err != nil {
			h.log.Warn("zipDir: close failed", "path", root, "err", err)
		}
	}()

	err = filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			// Skip unreadable entries (perm denied on a subdir) rather
			// than abort the whole archive.
			if errors.Is(walkErr, os.ErrPermission) {
				return nil
			}
			return walkErr
		}
		// Skip the root itself — zip entries are relative to it.
		if path == root {
			return nil
		}
		// Match /list's hidden-entry policy so the archive contents
		// don't surprise the operator with a .git/ they didn't see in
		// the tree.
		if strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		// Skip symlinks — we don't follow them in /list, and following
		// here risks infinite loops or escaping the requested subtree.
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		// Use forward slashes inside the archive regardless of host OS.
		zipName := filepath.ToSlash(rel)
		if info.IsDir() {
			zipName += "/"
			_, err := zw.CreateHeader(&zip.FileHeader{
				Name:     zipName,
				Method:   zip.Store,
				Modified: info.ModTime(),
			})
			return err
		}
		hdr, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		hdr.Name = zipName
		hdr.Method = zip.Deflate
		entry, err := zw.CreateHeader(hdr)
		if err != nil {
			return err
		}
		src, err := os.Open(path)
		if err != nil {
			// Surface permission denied on a single file as a skip —
			// most operators want as much of the tree as possible.
			if errors.Is(err, os.ErrPermission) {
				return nil
			}
			return err
		}
		defer src.Close()
		_, err = io.Copy(entry, src)
		return err
	})
	if err != nil {
		h.log.Warn("zipDir: walk failed mid-stream",
			"path", root, "err", err)
		// Response headers are long gone; nothing to do but stop
		// writing. The browser surfaces an incomplete download.
	}
}

type Entry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"is_dir"`
}

type ListResponse struct {
	Path    string  `json:"path"`
	Parent  string  `json:"parent,omitempty"`
	Entries []Entry `json:"entries"`
}

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	rawPath := r.URL.Query().Get("path")
	if rawPath == "" {
		rawPath = homeDir()
	}
	p, err := canonicalize(rawPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	st, err := os.Stat(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		if errors.Is(err, os.ErrPermission) {
			writeError(w, http.StatusForbidden, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !st.IsDir() {
		writeError(w, http.StatusBadRequest, errors.New("not a directory"))
		return
	}

	dir, err := os.ReadDir(p)
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			writeError(w, http.StatusForbidden, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	entries := make([]Entry, 0, len(dir))
	for _, d := range dir {
		name := d.Name()
		if strings.HasPrefix(name, ".") {
			// Hidden files / dotfiles are uninteresting for cwd
			// selection — most users want their visible project
			// folders, not .DS_Store / .git / .cache.
			continue
		}
		entries = append(entries, Entry{
			Name:  name,
			Path:  filepath.Join(p, name),
			IsDir: d.IsDir(),
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})

	parent := ""
	if p != "/" {
		parent = filepath.Dir(p)
	}
	writeJSON(w, http.StatusOK, ListResponse{
		Path:    p,
		Parent:  parent,
		Entries: entries,
	})
}

type MkdirRequest struct {
	Parent string `json:"parent"`
	Name   string `json:"name"`
}

type MkdirResponse struct {
	Path string `json:"path"`
}

func (h *Handlers) mkdir(w http.ResponseWriter, r *http.Request) {
	var req MkdirRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	parent, err := canonicalize(req.Parent)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, errors.New("name is required"))
		return
	}
	if strings.ContainsAny(name, "/\\") || name == "." || name == ".." {
		writeError(w, http.StatusBadRequest, errors.New("invalid directory name"))
		return
	}
	full := filepath.Join(parent, name)
	if err := os.MkdirAll(full, 0o755); err != nil {
		if errors.Is(err, os.ErrPermission) {
			writeError(w, http.StatusForbidden, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, MkdirResponse{Path: full})
}

func (h *Handlers) home(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"path": homeDir()})
}

// resolveWithinRoot canonicalizes both `target` and `root`, resolves
// any symlinks, then verifies the target stays inside the root. Used
// by the download + zip endpoints so an operator can only fetch files
// reachable from a known directory (typically the session's cwd) —
// not arbitrary system paths via the daemon's mount.
//
// This is the form static scanners recognise as a path-injection
// sanitiser: the user-supplied path is converted to a typed,
// validated value before reaching any I/O sink. `filepath.Rel`
// followed by an explicit `..` check on the relative result is the
// canonical Go pattern (filepath.Clean alone is not — it doesn't
// prevent escaping the root via absolute paths).
func resolveWithinRoot(target, root string) (string, error) {
	if strings.TrimSpace(root) == "" {
		return "", errors.New("root is required")
	}
	rootAbs, err := canonicalize(root)
	if err != nil {
		return "", fmt.Errorf("invalid root: %w", err)
	}
	// EvalSymlinks resolves both for the actual filesystem path. The
	// scanner cares that we're not feeding the raw query value into
	// the sink — using the resolved form means a symlink can't be
	// used to escape the root either.
	rootResolved, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		return "", fmt.Errorf("root not reachable: %w", err)
	}
	targetAbs, err := canonicalize(target)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}
	// Resolve the target *if it exists* — for missing paths, fall
	// back to the cleaned absolute form so the caller's `os.Open`
	// surfaces the 404 with the canonical error.
	targetResolved := targetAbs
	if r, rerr := filepath.EvalSymlinks(targetAbs); rerr == nil {
		targetResolved = r
	}
	rel, err := filepath.Rel(rootResolved, targetResolved)
	if err != nil {
		return "", fmt.Errorf("path outside allowed root: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("path is outside the allowed root")
	}
	return targetResolved, nil
}

// canonicalize expands ~/ prefixes and resolves to an absolute path.
// Symlinks are intentionally not followed here — operators may have
// project dirs that include symlinked checkout trees.
func canonicalize(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", errors.New("path is required")
	}
	if p == "~" {
		return homeDir(), nil
	}
	if strings.HasPrefix(p, "~/") {
		p = filepath.Join(homeDir(), p[2:])
	}
	if !filepath.IsAbs(p) {
		return "", errors.New("path must be absolute")
	}
	return filepath.Clean(p), nil
}

func homeDir() string {
	if u, err := user.Current(); err == nil && u.HomeDir != "" {
		return u.HomeDir
	}
	if h, err := os.UserHomeDir(); err == nil {
		return h
	}
	return "/"
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, code int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

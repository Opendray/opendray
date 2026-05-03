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
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
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

// Package git serves admin-only git inspection endpoints used by the
// web Inspector. Like internal/fs, it shells out to system tools
// rather than implementing parsing in-process — git plumbing already
// gives us machine-readable output via --porcelain / --pretty=format.
//
// All operations are read-only. No commit / push / mutating commands
// are exposed; if/when that's needed it should be a separate package
// behind an explicit write capability.
package git

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

const (
	// gitTimeout caps each git invocation. Status/log shouldn't take
	// more than a fraction of a second on healthy repos; this stops a
	// pathological repo from hanging the gateway.
	gitTimeout = 5 * time.Second
	// maxLogCount caps the recent commits listing to keep responses
	// small for the inspector panel.
	maxLogCount = 50
	defaultLog  = 20
)

type Handlers struct {
	log *slog.Logger
}

func NewHandlers(log *slog.Logger) *Handlers {
	if log == nil {
		log = slog.Default()
	}
	return &Handlers{log: log.With("component", "git.http")}
}

func (h *Handlers) Mount(r chi.Router) {
	r.Route("/git", func(r chi.Router) {
		r.Get("/status", h.status)
		r.Get("/log", h.log_)
		r.Get("/diff", h.diff)
		r.Get("/show", h.show)
	})
}

const (
	// maxDiffBytes caps diff/show output. Big refactors can produce
	// MBs of patch text — for the inspector panel we'd rather truncate
	// than block the gateway streaming the whole thing.
	maxDiffBytes = 512 * 1024
)

// StatusResponse is the JSON returned by GET /git/status?path=<dir>.
// IsRepo=false means the path exists but is not a git working tree —
// the panel renders an "init" affordance from this rather than a
// stack-trace.
type StatusResponse struct {
	IsRepo   bool         `json:"is_repo"`
	Branch   string       `json:"branch,omitempty"`
	Ahead    int          `json:"ahead"`
	Behind   int          `json:"behind"`
	Upstream string       `json:"upstream,omitempty"`
	Files    []StatusFile `json:"files"`
}

// StatusFile mirrors `git status --porcelain=v1 -b -z`'s entries.
// XY = two-char status (e.g. "M ", " M", "MM", "??"). Path is the
// repo-relative path; a rename has both Path and OldPath populated.
type StatusFile struct {
	XY      string `json:"xy"`
	Path    string `json:"path"`
	OldPath string `json:"old_path,omitempty"`
}

// LogResponse is the JSON for GET /git/log?path=<dir>&n=<count>.
type LogResponse struct {
	IsRepo  bool        `json:"is_repo"`
	Commits []LogCommit `json:"commits"`
}

type LogCommit struct {
	Hash      string `json:"hash"`
	ShortHash string `json:"short_hash"`
	Author    string `json:"author"`
	When      string `json:"when"`
	Subject   string `json:"subject"`
}

func (h *Handlers) status(w http.ResponseWriter, r *http.Request) {
	dir, err := dirParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), gitTimeout)
	defer cancel()

	if !isWorkTree(ctx, dir) {
		writeJSON(w, http.StatusOK, StatusResponse{IsRepo: false, Files: []StatusFile{}})
		return
	}

	// `--porcelain=v1 -b -z` gives stable output: header line(s) then
	// NUL-separated entries. -z avoids quoting issues for paths with
	// special chars.
	out, err := run(ctx, dir, "status", "--porcelain=v1", "-b", "-z")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	resp := parseStatus(string(out))
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handlers) log_(w http.ResponseWriter, r *http.Request) {
	dir, err := dirParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	n := defaultLog
	if v := r.URL.Query().Get("n"); v != "" {
		if i, err := strconv.Atoi(v); err == nil && i > 0 {
			n = i
		}
	}
	if n > maxLogCount {
		n = maxLogCount
	}

	ctx, cancel := context.WithTimeout(r.Context(), gitTimeout)
	defer cancel()

	if !isWorkTree(ctx, dir) {
		writeJSON(w, http.StatusOK, LogResponse{IsRepo: false, Commits: []LogCommit{}})
		return
	}

	// Tab-separated, NUL-terminated lines for safe parsing of commit
	// subjects that contain almost any byte.
	fmtArg := "--pretty=format:%H%x09%h%x09%an%x09%ar%x09%s%x00"
	out, err := run(ctx, dir, "log", "-n", strconv.Itoa(n), fmtArg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	commits := parseLog(string(out))
	writeJSON(w, http.StatusOK, LogResponse{IsRepo: true, Commits: commits})
}

// diff handles GET /git/diff?path=<dir>&file=<file>&scope=<unstaged|staged|all>.
//
//	scope=unstaged (default) → `git diff -- <file>`
//	scope=staged             → `git diff --cached -- <file>`
//	scope=all                → `git diff HEAD -- <file>`
//
// `file` is optional; omit to diff the whole tree. Untracked files
// have no diff (git tracks them as "??") so callers should fall back
// to /fs/read for those — diff returns empty in that case.
func (h *Handlers) diff(w http.ResponseWriter, r *http.Request) {
	dir, err := dirParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	scope := r.URL.Query().Get("scope")
	file := strings.TrimSpace(r.URL.Query().Get("file"))

	args := []string{"diff", "--no-color"}
	switch scope {
	case "staged":
		args = append(args, "--cached")
	case "all":
		args = append(args, "HEAD")
	case "", "unstaged":
		// default
	default:
		writeError(w, http.StatusBadRequest,
			errors.New("scope must be unstaged|staged|all"))
		return
	}
	if file != "" {
		// `--` ensures git treats `file` as a path even if it starts
		// with a dash. The path is repo-relative; we don't need to
		// canonicalize it the way we do for /fs.
		args = append(args, "--", file)
	}

	ctx, cancel := context.WithTimeout(r.Context(), gitTimeout)
	defer cancel()
	if !isWorkTree(ctx, dir) {
		writeError(w, http.StatusBadRequest, errors.New("not a git repository"))
		return
	}
	out, err := run(ctx, dir, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeText(w, out)
}

// show returns `git show <hash>` output: commit metadata + the full
// patch. hash is required.
func (h *Handlers) show(w http.ResponseWriter, r *http.Request) {
	dir, err := dirParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	hash := strings.TrimSpace(r.URL.Query().Get("hash"))
	if hash == "" {
		writeError(w, http.StatusBadRequest, errors.New("hash is required"))
		return
	}
	if !isValidHash(hash) {
		writeError(w, http.StatusBadRequest,
			errors.New("hash must be hex (4-64 chars)"))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), gitTimeout)
	defer cancel()
	if !isWorkTree(ctx, dir) {
		writeError(w, http.StatusBadRequest, errors.New("not a git repository"))
		return
	}
	out, err := run(ctx, dir,
		"show", "--no-color", "--stat", "--patch", hash)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeText(w, out)
}

// isValidHash guards against shell-injection-shaped values reaching
// git. Hex chars only, length sanity-bounded so callers can pass full
// or short SHAs.
func isValidHash(s string) bool {
	if len(s) < 4 || len(s) > 64 {
		return false
	}
	for _, r := range s {
		ok := (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') ||
			(r >= 'A' && r <= 'F')
		if !ok {
			return false
		}
	}
	return true
}

func parseStatus(s string) StatusResponse {
	resp := StatusResponse{IsRepo: true, Files: []StatusFile{}}
	parts := strings.Split(s, "\x00")
	skipNext := false
	for i, p := range parts {
		if skipNext {
			skipNext = false
			continue
		}
		if p == "" {
			continue
		}
		if strings.HasPrefix(p, "## ") {
			parseBranchLine(strings.TrimPrefix(p, "## "), &resp)
			continue
		}
		if len(p) < 3 {
			continue
		}
		xy := p[:2]
		path := p[3:]
		entry := StatusFile{XY: xy, Path: path}
		// Renames produce two NUL-separated tokens: "R  newpath\0oldpath".
		if xy[0] == 'R' || xy[0] == 'C' {
			if i+1 < len(parts) {
				entry.OldPath = parts[i+1]
				skipNext = true
			}
		}
		resp.Files = append(resp.Files, entry)
	}
	return resp
}

// parseBranchLine handles outputs like:
//
//	main
//	main...origin/main
//	main...origin/main [ahead 2]
//	main...origin/main [ahead 2, behind 1]
//	HEAD (no branch)
func parseBranchLine(line string, resp *StatusResponse) {
	if strings.HasPrefix(line, "HEAD ") {
		resp.Branch = "HEAD"
		return
	}
	track := ""
	if i := strings.Index(line, " ["); i >= 0 {
		track = line[i+2 : len(line)-1] // strip trailing "]"
		line = line[:i]
	}
	if i := strings.Index(line, "..."); i >= 0 {
		resp.Branch = line[:i]
		resp.Upstream = line[i+3:]
	} else {
		resp.Branch = line
	}
	for _, part := range strings.Split(track, ", ") {
		switch {
		case strings.HasPrefix(part, "ahead "):
			resp.Ahead, _ = strconv.Atoi(strings.TrimPrefix(part, "ahead "))
		case strings.HasPrefix(part, "behind "):
			resp.Behind, _ = strconv.Atoi(strings.TrimPrefix(part, "behind "))
		}
	}
}

func parseLog(s string) []LogCommit {
	out := []LogCommit{}
	for _, rec := range strings.Split(s, "\x00") {
		rec = strings.TrimPrefix(rec, "\n")
		if rec == "" {
			continue
		}
		fields := strings.SplitN(rec, "\t", 5)
		if len(fields) < 5 {
			continue
		}
		out = append(out, LogCommit{
			Hash:      fields[0],
			ShortHash: fields[1],
			Author:    fields[2],
			When:      fields[3],
			Subject:   fields[4],
		})
	}
	return out
}

func isWorkTree(ctx context.Context, dir string) bool {
	out, err := run(ctx, dir, "rev-parse", "--is-inside-work-tree")
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "true"
}

func run(ctx context.Context, dir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	// Strip GIT_* env vars from the parent so a gateway started from a
	// shell with GIT_DIR set doesn't leak that into target repos.
	cmd.Env = filteredEnv()
	return cmd.Output()
}

func filteredEnv() []string {
	src := os.Environ()
	out := make([]string, 0, len(src))
	for _, kv := range src {
		if strings.HasPrefix(kv, "GIT_") {
			continue
		}
		out = append(out, kv)
	}
	return out
}

// dirParam validates the ?path= query param: must be present, absolute,
// existing, and a directory. We do not do project-membership checks
// here — the admin-only middleware upstream is the trust boundary.
func dirParam(r *http.Request) (string, error) {
	p := strings.TrimSpace(r.URL.Query().Get("path"))
	if p == "" {
		return "", errors.New("path is required")
	}
	if !filepath.IsAbs(p) {
		return "", errors.New("path must be absolute")
	}
	p = filepath.Clean(p)
	st, err := os.Stat(p)
	if err != nil {
		return "", err
	}
	if !st.IsDir() {
		return "", errors.New("path is not a directory")
	}
	return p, nil
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

// writeText emits diff/show output as plain text, truncated with a
// trailer if it exceeds maxDiffBytes.
func writeText(w http.ResponseWriter, body []byte) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if len(body) > maxDiffBytes {
		body = body[:maxDiffBytes]
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
		_, _ = w.Write([]byte("\n... (truncated, exceeded 512 KiB)\n"))
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

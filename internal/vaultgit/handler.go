// Package vaultgit wraps `git` for the notes vault. Unlike the
// general-purpose internal/git package which takes an arbitrary path
// query, every operation here is locked to a single fixed root (the
// vault) — fewer ways for the web client to misuse it, and the
// frontend doesn't have to plumb the vault root around on every call.
//
// Authentication for remote ops (push / pull) follows the host's
// `git` config: SSH agent, OS keychain, credential helper, ~/.netrc,
// etc. Opendray does not store git credentials of its own.
package vaultgit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/opendray/opendray-v2/internal/githost"
)

const (
	gitTimeout    = 60 * time.Second
	maxOutputSize = 256 * 1024
	defaultLog    = 20
	maxLogCount   = 100
)

type Handlers struct {
	root  string           // absolute path to the vault root
	hosts *githost.Service // optional; nil disables HTTPS token injection
	log   *slog.Logger
}

func NewHandlers(vaultRoot string, hosts *githost.Service, log *slog.Logger) (*Handlers, error) {
	if log == nil {
		log = slog.Default()
	}
	if vaultRoot == "" {
		return nil, errors.New("vault root is empty")
	}
	abs, err := filepath.Abs(vaultRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve vault root: %w", err)
	}
	if _, err := os.Stat(abs); err != nil {
		return nil, fmt.Errorf("vault root: %w", err)
	}
	return &Handlers{
		root:  abs,
		hosts: hosts,
		log:   log.With("component", "vaultgit.http"),
	}, nil
}

func (h *Handlers) Mount(r chi.Router) {
	r.Route("/vault/git", func(r chi.Router) {
		r.Get("/status", h.status)
		r.Get("/auth", h.auth)
		r.Post("/init", h.init)
		r.Post("/commit", h.commit)
		r.Post("/pull", h.pull)
		r.Post("/push", h.push)
		r.Post("/abort", h.abort)
		r.Post("/reset-to-remote", h.resetToRemote)
		r.Get("/log", h.log_)
		r.Get("/remote", h.getRemote)
		r.Post("/remote", h.setRemote)
	})
}

// ── Response shapes ─────────────────────────────────────────────

type StatusResponse struct {
	IsRepo   bool         `json:"is_repo"`
	Branch   string       `json:"branch,omitempty"`
	Upstream string       `json:"upstream,omitempty"`
	Ahead    int          `json:"ahead"`
	Behind   int          `json:"behind"`
	Files    []StatusFile `json:"files"`
	Root     string       `json:"root"`
	// State is populated only when the repo is mid-operation (rebase /
	// merge / cherry-pick interrupted by conflicts) so the UI can
	// surface a clear recovery banner with abort + force-reset.
	State *GitState `json:"state,omitempty"`
}

// GitState reports any in-progress git operation that's blocking
// further pull/commit/push activity. The web UI uses this to decide
// whether to show a recovery banner instead of the normal action row.
type GitState struct {
	RebaseInProgress     bool     `json:"rebase_in_progress,omitempty"`
	MergeInProgress      bool     `json:"merge_in_progress,omitempty"`
	CherryPickInProgress bool     `json:"cherry_pick_in_progress,omitempty"`
	ConflictedFiles      []string `json:"conflicted_files,omitempty"`
}

type StatusFile struct {
	XY   string `json:"xy"`
	Path string `json:"path"`
}

type CommitRequest struct {
	Message string   `json:"message"`
	// Files restricts what's added to the commit. Empty = `git add .`.
	Files []string `json:"files,omitempty"`
}

type CommitResponse struct {
	Hash    string `json:"hash"`
	Message string `json:"message"`
	Output  string `json:"output"`
}

type RemoteEntry struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type SetRemoteRequest struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type LogResponse struct {
	Commits []LogCommit `json:"commits"`
}

type LogCommit struct {
	Hash      string `json:"hash"`
	ShortHash string `json:"short_hash"`
	Author    string `json:"author"`
	When      string `json:"when"`
	Subject   string `json:"subject"`
}

// ── Handlers ────────────────────────────────────────────────────

func (h *Handlers) status(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), gitTimeout)
	defer cancel()
	if !h.isRepo(ctx) {
		writeJSON(w, http.StatusOK, StatusResponse{
			IsRepo: false,
			Root:   h.root,
			Files:  []StatusFile{},
		})
		return
	}
	out, err := h.run(ctx, "status", "--porcelain=v1", "-b", "-z")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	resp := parseStatus(string(out))
	resp.Root = h.root
	if state := h.detectState(); state != nil {
		resp.State = state
	}
	writeJSON(w, http.StatusOK, resp)
}

// detectState peeks at .git for the marker files git uses to track
// in-progress rebase / merge / cherry-pick. Returns nil when the
// working tree is in a clean operational state (no recovery needed).
func (h *Handlers) detectState() *GitState {
	gitDir := filepath.Join(h.root, ".git")
	out := &GitState{}
	any := false
	if existsAny(filepath.Join(gitDir, "rebase-merge"), filepath.Join(gitDir, "rebase-apply")) {
		out.RebaseInProgress = true
		any = true
	}
	if exists(filepath.Join(gitDir, "MERGE_HEAD")) {
		out.MergeInProgress = true
		any = true
	}
	if exists(filepath.Join(gitDir, "CHERRY_PICK_HEAD")) {
		out.CherryPickInProgress = true
		any = true
	}
	if !any {
		return nil
	}
	// List unmerged paths via `git diff --name-only --diff-filter=U`.
	if list, err := h.run(context.Background(), "diff", "--name-only", "--diff-filter=U"); err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(list)), "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				out.ConflictedFiles = append(out.ConflictedFiles, line)
			}
		}
	}
	return out
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func existsAny(paths ...string) bool {
	for _, p := range paths {
		if exists(p) {
			return true
		}
	}
	return false
}

// abort runs the appropriate `git xxx --abort` based on what's
// currently in flight. When kind=="auto" (default) we detect from
// .git markers; explicit kinds let the UI force a specific abort.
//
// Body: {kind: "rebase"|"merge"|"cherry-pick"|"auto"}
func (h *Handlers) abort(w http.ResponseWriter, r *http.Request) {
	if !h.requireRepo(w, r) {
		return
	}
	var req struct {
		Kind string `json:"kind"`
	}
	_ = json.NewDecoder(io.LimitReader(r.Body, 1024)).Decode(&req)
	kind := strings.TrimSpace(req.Kind)
	if kind == "" || kind == "auto" {
		state := h.detectState()
		switch {
		case state == nil:
			writeError(w, http.StatusConflict,
				errors.New("no in-progress operation to abort"))
			return
		case state.RebaseInProgress:
			kind = "rebase"
		case state.MergeInProgress:
			kind = "merge"
		case state.CherryPickInProgress:
			kind = "cherry-pick"
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), gitTimeout)
	defer cancel()
	var args []string
	switch kind {
	case "rebase":
		args = []string{"rebase", "--abort"}
	case "merge":
		args = []string{"merge", "--abort"}
	case "cherry-pick":
		args = []string{"cherry-pick", "--abort"}
	default:
		writeError(w, http.StatusBadRequest,
			fmt.Errorf("unsupported abort kind: %s", kind))
		return
	}
	out, err := h.run(ctx, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError,
			fmt.Errorf("git %s --abort: %w (output: %s)", kind, err, string(out)))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"output": string(out),
		"kind":   kind,
	})
}

// resetToRemote is the destructive recovery: fetches from origin
// then `git reset --hard <remote_branch>`. Local commits not yet
// pushed AND any uncommitted changes are wiped. UI must confirm
// loudly before invoking.
//
// Body: {remote_branch?: "origin/main"}  defaults to origin/<current>
func (h *Handlers) resetToRemote(w http.ResponseWriter, r *http.Request) {
	if !h.requireRepo(w, r) {
		return
	}
	var req struct {
		RemoteBranch string `json:"remote_branch"`
	}
	_ = json.NewDecoder(io.LimitReader(r.Body, 1024)).Decode(&req)

	ctx, cancel := context.WithTimeout(r.Context(), gitTimeout)
	defer cancel()

	remoteBranch := strings.TrimSpace(req.RemoteBranch)
	if remoteBranch == "" {
		branch := strings.TrimSpace(string(must(h.run(ctx, "rev-parse", "--abbrev-ref", "HEAD"))))
		if branch == "" || branch == "HEAD" {
			branch = "main"
		}
		remoteBranch = "origin/" + branch
	}

	// First: clean up any in-progress op so reset is clean.
	if state := h.detectState(); state != nil {
		switch {
		case state.RebaseInProgress:
			_, _ = h.run(ctx, "rebase", "--abort")
		case state.MergeInProgress:
			_, _ = h.run(ctx, "merge", "--abort")
		case state.CherryPickInProgress:
			_, _ = h.run(ctx, "cherry-pick", "--abort")
		}
	}

	// Fetch latest from origin first so the ref is current.
	fetchArgs, err := h.argsWithAuth(ctx, "fetch", "origin")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if out, ferr := h.run(ctx, fetchArgs...); ferr != nil {
		writeError(w, http.StatusInternalServerError,
			fmt.Errorf("git fetch: %w (output: %s)", ferr, string(out)))
		return
	}

	out, rerr := h.run(ctx, "reset", "--hard", remoteBranch)
	if rerr != nil {
		writeError(w, http.StatusInternalServerError,
			fmt.Errorf("git reset --hard %s: %w (output: %s)",
				remoteBranch, rerr, string(out)))
		return
	}
	// Drop any untracked junk too so the working tree fully matches
	// the remote — what users expect from a "reset to remote" button.
	_, _ = h.run(ctx, "clean", "-fd")
	writeJSON(w, http.StatusOK, map[string]string{
		"output":        string(out),
		"remote_branch": remoteBranch,
	})
}

func (h *Handlers) init(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), gitTimeout)
	defer cancel()
	if h.isRepo(ctx) {
		writeError(w, http.StatusConflict, errors.New("vault is already a git repository"))
		return
	}
	out, err := h.run(ctx, "init", "-b", "main")
	if err != nil {
		writeError(w, http.StatusInternalServerError,
			fmt.Errorf("git init: %w", err))
		return
	}
	// Drop a sane default .gitignore if missing — keeps macOS .DS_Store
	// out of the repo, plus any future opendray runtime artefacts.
	gi := filepath.Join(h.root, ".gitignore")
	if _, err := os.Stat(gi); errors.Is(err, os.ErrNotExist) {
		_ = os.WriteFile(gi, []byte(".DS_Store\n.cache/\n"), 0o600)
	}
	writeJSON(w, http.StatusOK, map[string]string{"output": string(out)})
}

func (h *Handlers) commit(w http.ResponseWriter, r *http.Request) {
	if !h.requireRepo(w, r) {
		return
	}
	var req CommitRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 64<<10)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	msg := strings.TrimSpace(req.Message)
	if msg == "" {
		msg = fmt.Sprintf("Notes: %s", time.Now().Format("2006-01-02 15:04"))
	}

	ctx, cancel := context.WithTimeout(r.Context(), gitTimeout)
	defer cancel()

	addArgs := []string{"add"}
	if len(req.Files) == 0 {
		addArgs = append(addArgs, ".")
	} else {
		// Sanity check paths — must be relative, no `..`. Vault is the
		// only working tree we ever touch.
		for _, f := range req.Files {
			f = strings.TrimSpace(f)
			if f == "" {
				continue
			}
			if filepath.IsAbs(f) || strings.Contains(f, "..") {
				writeError(w, http.StatusBadRequest,
					fmt.Errorf("invalid file path: %s", f))
				return
			}
			addArgs = append(addArgs, "--", f)
			break // -- terminator only needed once; rest follow as args
		}
		for _, f := range req.Files {
			f = strings.TrimSpace(f)
			if f == "" || filepath.IsAbs(f) || strings.Contains(f, "..") {
				continue
			}
			addArgs = append(addArgs, f)
		}
	}
	if _, err := h.run(ctx, addArgs...); err != nil {
		writeError(w, http.StatusInternalServerError,
			fmt.Errorf("git add: %w", err))
		return
	}
	out, err := h.run(ctx, "commit", "-m", msg)
	if err != nil {
		// Could be "nothing to commit" — git exits 1. Surface the
		// stderr so the UI can render a useful message.
		writeError(w, http.StatusUnprocessableEntity,
			fmt.Errorf("git commit: %w", err))
		return
	}
	hash, _ := h.run(ctx, "rev-parse", "--short", "HEAD")
	writeJSON(w, http.StatusOK, CommitResponse{
		Hash:    strings.TrimSpace(string(hash)),
		Message: msg,
		Output:  string(out),
	})
}

func (h *Handlers) pull(w http.ResponseWriter, r *http.Request) {
	if !h.requireRepo(w, r) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), gitTimeout)
	defer cancel()
	// Detect whether the current branch already has upstream tracking
	// configured. If yes, plain `git pull` works. If no, explicitly
	// pull from origin's matching branch — git --set-upstream-to via
	// `pull --set-upstream` isn't a thing, but adding `origin <branch>`
	// to the pull args avoids the "no tracking information" error and
	// causes git to record the relationship as a side effect.
	pullArgs := []string{"pull", "--rebase", "--autostash"}
	if !h.hasUpstream(ctx) {
		branch := strings.TrimSpace(string(must(h.run(ctx, "rev-parse", "--abbrev-ref", "HEAD"))))
		if branch == "" || branch == "HEAD" {
			branch = "main"
		}
		pullArgs = append(pullArgs, "origin", branch)
	}
	args, err := h.argsWithAuth(ctx, pullArgs...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	out, err := h.run(ctx, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError,
			fmt.Errorf("git pull: %w (output: %s)", err, string(out)))
		return
	}
	// Best-effort: set upstream tracking for next time so plain
	// `git pull` works after this. Ignore errors — the pull already
	// succeeded; this is just a quality-of-life follow-up.
	if !h.hasUpstream(ctx) {
		branch := strings.TrimSpace(string(must(h.run(ctx, "rev-parse", "--abbrev-ref", "HEAD"))))
		if branch != "" && branch != "HEAD" {
			_, _ = h.run(ctx, "branch", "--set-upstream-to=origin/"+branch, branch)
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"output": string(out)})
}

// hasUpstream returns true when the current branch has an upstream
// tracking ref configured. Cheap probe used to decide pull-arg shape.
func (h *Handlers) hasUpstream(ctx context.Context) bool {
	_, err := h.run(ctx, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	return err == nil
}

// must returns the byte payload from a `run` call, swallowing the
// error. Suitable for "best-effort" reads where a missing answer is
// fine and the caller has a fallback.
func must(b []byte, _ error) []byte { return b }

func (h *Handlers) push(w http.ResponseWriter, r *http.Request) {
	if !h.requireRepo(w, r) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), gitTimeout)
	defer cancel()
	// Auto-set upstream on first push so users don't need to know
	// `git push -u origin main`. Harmless when already set.
	args, err := h.argsWithAuth(ctx, "push", "-u", "origin", "HEAD")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	out, err := h.run(ctx, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError,
			fmt.Errorf("git push: %w (output: %s)", err, string(out)))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"output": string(out)})
}

// AuthInfo describes how the next pull/push will authenticate. The
// UI shows this so the user knows whether their token / SSH setup is
// being used before they hit the button.
type AuthInfo struct {
	HasRemote bool   `json:"has_remote"`
	RemoteURL string `json:"remote_url,omitempty"`
	Scheme    string `json:"scheme,omitempty"` // "ssh" | "https" | "git" | "http"
	Host      string `json:"host,omitempty"`
	// For HTTPS remotes: report whether opendray will inject a token
	// from the matching git_hosts row, and where it came from.
	UsingToken    bool   `json:"using_token,omitempty"`
	TokenSource   string `json:"token_source,omitempty"` // "git_hosts:<host>"
	TokenMissing  bool   `json:"token_missing,omitempty"`
	HelpfulHint   string `json:"helpful_hint,omitempty"`
}

func (h *Handlers) auth(w http.ResponseWriter, r *http.Request) {
	if !h.requireRepo(w, r) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	rawURL, err := h.run(ctx, "remote", "get-url", "origin")
	if err != nil {
		writeJSON(w, http.StatusOK, AuthInfo{HasRemote: false})
		return
	}
	urlStr := strings.TrimSpace(string(rawURL))
	scheme, host := parseRemote(urlStr)
	info := AuthInfo{
		HasRemote: true,
		RemoteURL: urlStr,
		Scheme:    scheme,
		Host:      host,
	}
	switch scheme {
	case "https", "http":
		if h.hosts != nil && host != "" {
			if hv, err := h.hosts.GetByHost(ctx, host); err == nil && hv.Token != "" {
				info.UsingToken = true
				info.TokenSource = "git_hosts:" + host
			} else {
				info.TokenMissing = true
				info.HelpfulHint =
					"Add a token for " + host + " in Plugins → Git hosts so " +
						"opendray can authenticate over HTTPS."
			}
		} else {
			info.TokenMissing = true
			info.HelpfulHint =
				"HTTPS remote — configure a token for " + host +
					" in Plugins → Git hosts."
		}
	case "ssh", "git":
		info.HelpfulHint =
			"SSH remote — uses the gateway host's ssh-agent / ~/.ssh keys. " +
				"Make sure ssh -T " + host + " works from the host shell."
	}
	writeJSON(w, http.StatusOK, info)
}

// argsWithAuth peeks at the current `origin` remote URL and, if it's
// an HTTPS remote with a matching git_hosts token, prepends git
// `-c credential.helper=...` flags that feed the token to git's auth
// machinery for this single invocation. SSH remotes get no
// modification — those flow through the host's ssh-agent.
func (h *Handlers) argsWithAuth(ctx context.Context, gitArgs ...string) ([]string, error) {
	if h.hosts == nil {
		return gitArgs, nil
	}
	rawURL, err := h.run(ctx, "remote", "get-url", "origin")
	if err != nil {
		// No origin yet, or git error — let the underlying command
		// surface a clearer message.
		return gitArgs, nil
	}
	scheme, host := parseRemote(strings.TrimSpace(string(rawURL)))
	if scheme != "https" && scheme != "http" {
		return gitArgs, nil
	}
	if host == "" {
		return gitArgs, nil
	}
	hv, err := h.hosts.GetByHost(ctx, host)
	if err != nil || hv.Token == "" {
		return gitArgs, nil
	}
	// Inline credential helper: an embedded `!sh -c '...'` that prints
	// the username + password git asks for. Quoting carefully because
	// git parses the helper string as a shell command after a leading
	// `!`. The token never lands on disk this way.
	username := credentialUsername(hv.Kind)
	helper := fmt.Sprintf(
		`!f() { printf 'username=%%s\npassword=%%s\n' '%s' '%s'; }; f`,
		shellEscape(username), shellEscape(hv.Token),
	)
	out := []string{
		"-c", "credential.helper=",                   // wipe any inherited helper
		"-c", "credential.helper=" + helper,           // ours
	}
	out = append(out, gitArgs...)
	return out, nil
}

// credentialUsername picks the username git expects per host kind.
// Most providers accept the literal token as the password and any
// non-empty username; GitLab needs `oauth2` for personal access
// tokens to authenticate.
func credentialUsername(kind githost.Kind) string {
	switch kind {
	case githost.KindGitLab:
		return "oauth2"
	default:
		return "opendray"
	}
}

// shellEscape wraps single quotes for the inline credential helper.
// Tokens shouldn't contain `'` in practice but we defend anyway.
func shellEscape(s string) string {
	return strings.ReplaceAll(s, "'", `'\''`)
}

// parseRemote returns the scheme + host for a git URL. Recognises
// HTTPS (`https://host/...`), HTTP, SSH (`ssh://...`), and the SCP
// short form (`user@host:path`). Returns empty strings on unknown
// shapes.
func parseRemote(raw string) (scheme, host string) {
	if raw == "" {
		return "", ""
	}
	if strings.HasPrefix(raw, "git@") || (strings.Contains(raw, "@") && strings.Contains(raw, ":") && !strings.Contains(raw, "://")) {
		// scp-like: user@host:path
		rest := raw
		if i := strings.Index(rest, "@"); i >= 0 {
			rest = rest[i+1:]
		}
		if i := strings.Index(rest, ":"); i >= 0 {
			return "ssh", rest[:i]
		}
		return "ssh", ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "", ""
	}
	return u.Scheme, u.Host
}

func (h *Handlers) log_(w http.ResponseWriter, r *http.Request) {
	if !h.requireRepo(w, r) {
		return
	}
	n := defaultLog
	if v := r.URL.Query().Get("n"); v != "" {
		if i, err := parseN(v); err == nil && i > 0 {
			n = i
		}
	}
	if n > maxLogCount {
		n = maxLogCount
	}
	ctx, cancel := context.WithTimeout(r.Context(), gitTimeout)
	defer cancel()
	out, err := h.run(ctx, "log", "-n", fmt.Sprint(n),
		"--pretty=format:%H%x09%h%x09%an%x09%ar%x09%s%x00")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, LogResponse{Commits: parseLog(string(out))})
}

func (h *Handlers) getRemote(w http.ResponseWriter, r *http.Request) {
	if !h.requireRepo(w, r) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), gitTimeout)
	defer cancel()
	out, err := h.run(ctx, "remote", "-v")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	remotes := []RemoteEntry{}
	seen := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 || seen[fields[0]] {
			continue
		}
		seen[fields[0]] = true
		remotes = append(remotes, RemoteEntry{Name: fields[0], URL: fields[1]})
	}
	writeJSON(w, http.StatusOK, map[string]any{"remotes": remotes})
}

func (h *Handlers) setRemote(w http.ResponseWriter, r *http.Request) {
	if !h.requireRepo(w, r) {
		return
	}
	var req SetRemoteRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 16<<10)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	name := strings.TrimSpace(req.Name)
	url := strings.TrimSpace(req.URL)
	if name == "" {
		name = "origin"
	}
	if url == "" {
		writeError(w, http.StatusBadRequest, errors.New("url is required"))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), gitTimeout)
	defer cancel()
	// Try set-url first (idempotent); fall back to add if remote
	// doesn't exist yet.
	if _, err := h.run(ctx, "remote", "set-url", name, url); err != nil {
		if _, addErr := h.run(ctx, "remote", "add", name, url); addErr != nil {
			writeError(w, http.StatusInternalServerError,
				fmt.Errorf("set remote: %w", addErr))
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Helpers ─────────────────────────────────────────────────────

func (h *Handlers) isRepo(ctx context.Context) bool {
	out, err := h.run(ctx, "rev-parse", "--is-inside-work-tree")
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "true"
}

func (h *Handlers) requireRepo(w http.ResponseWriter, r *http.Request) bool {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	if !h.isRepo(ctx) {
		writeError(w, http.StatusConflict,
			errors.New("vault is not a git repository — call /vault/git/init first"))
		return false
	}
	return true
}

// run executes git in the vault root with a clean env (strips parent
// GIT_* so a gateway started under unusual config doesn't leak it
// into the vault). Captures stdout+stderr together so the caller can
// surface the full git output for diagnostics.
func (h *Handlers) run(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = h.root
	cmd.Env = filteredEnv()
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		out := truncate(buf.Bytes(), maxOutputSize)
		return out, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return truncate(buf.Bytes(), maxOutputSize), nil
}

func truncate(b []byte, n int) []byte {
	if len(b) <= n {
		return b
	}
	return append(b[:n], []byte("\n... (truncated)\n")...)
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

func parseN(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

func parseStatus(s string) StatusResponse {
	resp := StatusResponse{IsRepo: true, Files: []StatusFile{}}
	parts := strings.Split(s, "\x00")
	for _, p := range parts {
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
		resp.Files = append(resp.Files, StatusFile{XY: p[:2], Path: p[3:]})
	}
	return resp
}

func parseBranchLine(line string, resp *StatusResponse) {
	if strings.HasPrefix(line, "HEAD ") {
		resp.Branch = "HEAD"
		return
	}
	track := ""
	if i := strings.Index(line, " ["); i >= 0 {
		track = line[i+2 : len(line)-1]
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
			fmt.Sscanf(part, "ahead %d", &resp.Ahead)
		case strings.HasPrefix(part, "behind "):
			fmt.Sscanf(part, "behind %d", &resp.Behind)
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
		f := strings.SplitN(rec, "\t", 5)
		if len(f) < 5 {
			continue
		}
		out = append(out, LogCommit{
			Hash:      f[0],
			ShortHash: f[1],
			Author:    f[2],
			When:      f[3],
			Subject:   f[4],
		})
	}
	return out
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

// Package git provides safe, read-only git repository inspection for the
// source-control panel plugin. Every operation is pinned to a caller-provided
// repository path that must resolve inside the plugin's allowedRoots, and
// every subprocess runs under a context timeout to guard the gateway from
// runaway git invocations. Write paths (stage/unstage/discard/commit) live
// in the Claude session workflow, not here — the panel observes only.
package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Config holds settings resolved from the plugin's configSchema.
type Config struct {
	AllowedRoots []string
	DefaultPath  string
	GitBinary    string
	LogLimit     int
	DiffContext  int
	Timeout      time.Duration
}

// FileStatus describes one entry in `git status --porcelain=v1`.
type FileStatus struct {
	Path     string `json:"path"`
	OldPath  string `json:"oldPath,omitempty"` // populated for renames
	Index    string `json:"index"`             // staged status code, e.g. "M", "A"
	WorkTree string `json:"workTree"`          // unstaged status code
	Staged   bool   `json:"staged"`
	Unstaged bool   `json:"unstaged"`
	Untracked bool  `json:"untracked"`
}

// StatusResult is the composite status view returned to the panel.
type StatusResult struct {
	Repo      string       `json:"repo"`
	Branch    string       `json:"branch"`
	Head      string       `json:"head"`
	Upstream  string       `json:"upstream,omitempty"`
	Ahead     int          `json:"ahead"`
	Behind    int          `json:"behind"`
	Files     []FileStatus `json:"files"`
	Clean     bool         `json:"clean"`
}

// Commit is one entry in the log view.
type Commit struct {
	SHA     string `json:"sha"`
	Short   string `json:"short"`
	Author  string `json:"author"`
	Email   string `json:"email"`
	Date    int64  `json:"date"` // unix seconds
	Subject string `json:"subject"`
}

// Branch describes a local branch.
type Branch struct {
	Name     string `json:"name"`
	Current  bool   `json:"current"`
	Upstream string `json:"upstream,omitempty"`
	Head     string `json:"head"`
}

// DiffOptions selects which diff to compute.
type DiffOptions struct {
	Staged bool   // --cached
	Since  string // commit-ish; when set, diff is <since>..HEAD + working tree
	Path   string // optional path filter (must resolve under the repo)
}

// DiffResult wraps the unified diff text.
type DiffResult struct {
	Repo string `json:"repo"`
	Diff string `json:"diff"`
}

// SessionBaseline records the commit a session started from so the panel
// can show only what changed during the current session.
type SessionBaseline struct {
	SessionID string    `json:"sessionId"`
	Repo      string    `json:"repo"`
	HeadSHA   string    `json:"headSha"`
	CreatedAt time.Time `json:"createdAt"`
}

// Manager owns the in-memory session → baseline map and is safe for
// concurrent use from the HTTP handlers.
type Manager struct {
	mu        sync.RWMutex
	baselines map[string]SessionBaseline // key = sessionID
}

// NewManager returns an empty Manager.
func NewManager() *Manager {
	return &Manager{baselines: make(map[string]SessionBaseline)}
}

// ── Security ────────────────────────────────────────────────────

// SecurePath resolves path against the allowed roots and returns the
// absolute, symlink-free path. When path is empty the default path (or the
// first allowed root) is used.
func SecurePath(cfg Config, path string) (string, error) {
	if path == "" {
		if cfg.DefaultPath != "" {
			path = cfg.DefaultPath
		} else if len(cfg.AllowedRoots) > 0 {
			path = cfg.AllowedRoots[0]
		} else {
			return "", errors.New("git: no allowed roots configured")
		}
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("git: invalid path: %w", err)
	}
	realPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		realPath = absPath
	}
	for _, root := range cfg.AllowedRoots {
		rootAbs, _ := filepath.Abs(root)
		rootReal, err := filepath.EvalSymlinks(rootAbs)
		if err != nil {
			rootReal = rootAbs
		}
		if strings.HasPrefix(realPath, rootReal) {
			return realPath, nil
		}
	}
	return "", fmt.Errorf("git: path %q is outside allowed roots", path)
}

// refPattern validates commit SHAs, branch names, and tags so user-supplied
// refs never leak shell metacharacters into exec args. Git itself rejects
// most of these via refname rules, but defence in depth is cheap.
var refPattern = regexp.MustCompile(`^[A-Za-z0-9._/\-]{1,200}$`)

// relPathPattern allows only forward-slash POSIX paths — the format git's
// porcelain always emits — with no leading slash or parent traversal.
var relPathPattern = regexp.MustCompile(`^[^\s\\][^\\]*$`)

func validateRef(ref string) error {
	if ref == "" {
		return errors.New("git: ref is empty")
	}
	if strings.HasPrefix(ref, "-") {
		return fmt.Errorf("git: ref %q must not start with -", ref)
	}
	if !refPattern.MatchString(ref) {
		return fmt.Errorf("git: invalid ref %q", ref)
	}
	return nil
}

func validateRelPath(p string) error {
	if p == "" {
		return errors.New("git: path is empty")
	}
	if strings.HasPrefix(p, "-") {
		return fmt.Errorf("git: path %q must not start with -", p)
	}
	if strings.Contains(p, "..") {
		return fmt.Errorf("git: path %q must not contain ..", p)
	}
	if !relPathPattern.MatchString(p) {
		return fmt.Errorf("git: invalid path %q", p)
	}
	return nil
}

// ── Command runner ──────────────────────────────────────────────

func (c Config) binary() string {
	if c.GitBinary != "" {
		return c.GitBinary
	}
	return "git"
}

func (c Config) timeout() time.Duration {
	if c.Timeout <= 0 {
		return 20 * time.Second
	}
	return c.Timeout
}

// run executes git with the provided args inside repo. stdin is optional.
// The `--` sentinel should be inserted by callers before any path-like
// arguments to prevent git from interpreting paths starting with `-`.
func run(ctx context.Context, cfg Config, repo string, stdin []byte, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, cfg.timeout())
	defer cancel()

	full := append([]string{"--no-pager", "-C", repo}, args...)
	cmd := exec.CommandContext(ctx, cfg.binary(), full...)
	cmd.Env = append(cmd.Environ(), "GIT_TERMINAL_PROMPT=0")
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return stdout.Bytes(), nil
}

// ── Status ──────────────────────────────────────────────────────

// Status runs `git status` and a few companion queries, returning a
// normalised snapshot.
func Status(ctx context.Context, cfg Config, repoPath string) (StatusResult, error) {
	repo, err := SecurePath(cfg, repoPath)
	if err != nil {
		return StatusResult{}, err
	}

	out, err := run(ctx, cfg, repo, nil, "status", "--porcelain=v1", "--branch", "--untracked-files=all")
	if err != nil {
		return StatusResult{}, err
	}

	res := StatusResult{Repo: repo, Files: []FileStatus{}}
	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "## ") {
			parseBranchLine(line[3:], &res)
			continue
		}
		if len(line) < 3 {
			continue
		}
		fs := FileStatus{
			Index:    strings.TrimSpace(line[0:1]),
			WorkTree: strings.TrimSpace(line[1:2]),
		}
		rest := line[3:]
		// Rename/copy lines have the form "orig -> new".
		if idx := strings.Index(rest, " -> "); idx >= 0 {
			fs.OldPath = rest[:idx]
			fs.Path = rest[idx+4:]
		} else {
			fs.Path = rest
		}
		fs.Staged = fs.Index != "" && fs.Index != "?"
		fs.Unstaged = fs.WorkTree != "" && fs.WorkTree != "?"
		fs.Untracked = line[0] == '?' && line[1] == '?'
		res.Files = append(res.Files, fs)
	}
	res.Clean = len(res.Files) == 0

	if head, err := run(ctx, cfg, repo, nil, "rev-parse", "HEAD"); err == nil {
		res.Head = strings.TrimSpace(string(head))
	}
	return res, nil
}

func parseBranchLine(line string, res *StatusResult) {
	// Examples:
	//   "main...origin/main [ahead 1]"
	//   "main"
	//   "HEAD (no branch)"
	if strings.HasPrefix(line, "HEAD ") {
		res.Branch = "HEAD"
		return
	}
	branch := line
	bracket := ""
	if i := strings.Index(branch, " ["); i >= 0 {
		bracket = branch[i+2:]
		bracket = strings.TrimSuffix(bracket, "]")
		branch = branch[:i]
	}
	if i := strings.Index(branch, "..."); i >= 0 {
		res.Branch = branch[:i]
		res.Upstream = branch[i+3:]
	} else {
		res.Branch = branch
	}
	for _, part := range strings.Split(bracket, ", ") {
		part = strings.TrimSpace(part)
		switch {
		case strings.HasPrefix(part, "ahead "):
			res.Ahead, _ = strconv.Atoi(strings.TrimPrefix(part, "ahead "))
		case strings.HasPrefix(part, "behind "):
			res.Behind, _ = strconv.Atoi(strings.TrimPrefix(part, "behind "))
		}
	}
}

// ── Diff ────────────────────────────────────────────────────────

// Diff computes a unified diff. Per DiffOptions it returns either the
// staged diff, the unstaged diff, or the combined diff since a baseline
// commit (session-scoped view).
func Diff(ctx context.Context, cfg Config, repoPath string, opts DiffOptions) (DiffResult, error) {
	repo, err := SecurePath(cfg, repoPath)
	if err != nil {
		return DiffResult{}, err
	}

	args := []string{"diff", fmt.Sprintf("-U%d", cfg.diffContext()), "--no-color"}
	if opts.Since != "" {
		if err := validateRef(opts.Since); err != nil {
			return DiffResult{}, err
		}
		args = append(args, opts.Since)
	} else if opts.Staged {
		args = append(args, "--cached")
	}

	args = append(args, "--")
	if opts.Path != "" {
		if err := validateRelPath(opts.Path); err != nil {
			return DiffResult{}, err
		}
		args = append(args, opts.Path)
	}

	out, err := run(ctx, cfg, repo, nil, args...)
	if err != nil {
		return DiffResult{}, err
	}
	return DiffResult{Repo: repo, Diff: string(out)}, nil
}

func (c Config) diffContext() int {
	if c.DiffContext <= 0 {
		return 3
	}
	return c.DiffContext
}

// ── Log / branches ──────────────────────────────────────────────

const logFormat = "%H%x1f%h%x1f%an%x1f%ae%x1f%at%x1f%s"

// Log returns the most recent commits.
func Log(ctx context.Context, cfg Config, repoPath string, limit int) ([]Commit, error) {
	repo, err := SecurePath(cfg, repoPath)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = cfg.LogLimit
	}
	if limit <= 0 {
		limit = 50
	}
	out, err := run(ctx, cfg, repo, nil,
		"log",
		fmt.Sprintf("--max-count=%d", limit),
		"--pretty=format:"+logFormat,
	)
	if err != nil {
		return nil, err
	}
	var result []Commit
	for _, line := range strings.Split(string(out), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\x1f")
		if len(parts) < 6 {
			continue
		}
		ts, _ := strconv.ParseInt(parts[4], 10, 64)
		result = append(result, Commit{
			SHA:     parts[0],
			Short:   parts[1],
			Author:  parts[2],
			Email:   parts[3],
			Date:    ts,
			Subject: parts[5],
		})
	}
	return result, nil
}

// Branches lists local branches and marks the current one.
func Branches(ctx context.Context, cfg Config, repoPath string) ([]Branch, error) {
	repo, err := SecurePath(cfg, repoPath)
	if err != nil {
		return nil, err
	}
	out, err := run(ctx, cfg, repo, nil,
		"for-each-ref", "refs/heads",
		"--format=%(refname:short)%09%(objectname)%09%(upstream:short)%09%(HEAD)",
	)
	if err != nil {
		return nil, err
	}
	var result []Branch
	for _, line := range strings.Split(string(out), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 4 {
			continue
		}
		result = append(result, Branch{
			Name:     parts[0],
			Head:     parts[1],
			Upstream: parts[2],
			Current:  strings.TrimSpace(parts[3]) == "*",
		})
	}
	return result, nil
}

// ── Session baselines ───────────────────────────────────────────

// Snapshot records HEAD for a session so SessionDiff can later show only
// the changes made during that session. Re-snapshotting a session ID
// overwrites the prior baseline.
func (m *Manager) Snapshot(ctx context.Context, cfg Config, sessionID, repoPath string) (SessionBaseline, error) {
	if sessionID == "" {
		return SessionBaseline{}, errors.New("git: sessionId is required")
	}
	repo, err := SecurePath(cfg, repoPath)
	if err != nil {
		return SessionBaseline{}, err
	}
	out, err := run(ctx, cfg, repo, nil, "rev-parse", "HEAD")
	if err != nil {
		return SessionBaseline{}, err
	}
	baseline := SessionBaseline{
		SessionID: sessionID,
		Repo:      repo,
		HeadSHA:   strings.TrimSpace(string(out)),
		CreatedAt: time.Now(),
	}
	m.mu.Lock()
	m.baselines[sessionID] = baseline
	m.mu.Unlock()
	return baseline, nil
}

// Baseline returns the recorded baseline (if any) for a session.
func (m *Manager) Baseline(sessionID string) (SessionBaseline, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	b, ok := m.baselines[sessionID]
	return b, ok
}

// ClearBaseline removes a session's baseline (called when a session ends).
func (m *Manager) ClearBaseline(sessionID string) {
	m.mu.Lock()
	delete(m.baselines, sessionID)
	m.mu.Unlock()
}

// SessionDiff returns a diff from the session's baseline HEAD to the
// current working tree. Caller must have snapshotted first.
func (m *Manager) SessionDiff(ctx context.Context, cfg Config, sessionID string) (DiffResult, error) {
	b, ok := m.Baseline(sessionID)
	if !ok {
		return DiffResult{}, errors.New("git: no baseline recorded for this session; snapshot first")
	}
	return Diff(ctx, cfg, b.Repo, DiffOptions{Since: b.HeadSHA})
}

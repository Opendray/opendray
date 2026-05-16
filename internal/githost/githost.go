// Package githost adds two layers on top of the read-only `git`
// helpers in internal/git:
//
//  1. CRUD for per-host API tokens (GitHub / Gitea / GitLab) so the
//     Inspector's Plugins page can manage credentials.
//  2. Detect-remote + list-PRs endpoints used by the Inspector's Git
//     tab to render an "open PRs" section for the session's repo.
//
// Tokens are stored plaintext in `git_hosts` — same trust model as the
// claude_accounts on-disk OAuth tokens. The handlers mount under the
// admin-only middleware group.
package githost

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
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	httpTimeout = 6 * time.Second
)

var (
	ErrNotFound       = errors.New("git host not found")
	ErrNoTokenForHost = errors.New("no git host configured for this remote")
)

// Kind is the API flavor for a host.
type Kind string

const (
	KindGitHub Kind = "github"
	KindGitea  Kind = "gitea"
	KindGitLab Kind = "gitlab"
)

// Host is the row stored in `git_hosts`. Token is exposed in JSON only
// when the caller is creating/updating; List/Get redact via TokenMask.
type Host struct {
	ID        string    `json:"id"`
	Kind      Kind      `json:"kind"`
	Host      string    `json:"host"`
	Name      string    `json:"name"`
	Token     string    `json:"token,omitempty"`
	TokenMask string    `json:"token_mask,omitempty"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateRequest / UpdateRequest are the JSON bodies for POST / PUT.
type CreateRequest struct {
	Kind  Kind   `json:"kind"`
	Host  string `json:"host"`
	Name  string `json:"name"`
	Token string `json:"token"`
}

type UpdateRequest struct {
	Kind    *Kind   `json:"kind,omitempty"`
	Host    *string `json:"host,omitempty"`
	Name    *string `json:"name,omitempty"`
	Token   *string `json:"token,omitempty"`
	Enabled *bool   `json:"enabled,omitempty"`
}

// Service owns the DB layer and the HTTP client used for upstream API
// calls. Held by the app for its lifetime.
type Service struct {
	pool *pgxpool.Pool
	log  *slog.Logger
	http *http.Client
}

func NewService(pool *pgxpool.Pool, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{
		pool: pool,
		log:  log.With("component", "githost"),
		http: &http.Client{Timeout: httpTimeout},
	}
}

// ── CRUD ────────────────────────────────────────────────────────

const hostSelect = `
    SELECT id, kind, host, name, token, enabled, created_at, updated_at
    FROM git_hosts`

func (s *Service) List(ctx context.Context) ([]Host, error) {
	rows, err := s.pool.Query(ctx, hostSelect+` ORDER BY host`)
	if err != nil {
		return nil, fmt.Errorf("list git hosts: %w", err)
	}
	defer rows.Close()
	out := []Host{}
	for rows.Next() {
		h, err := scanHost(rows)
		if err != nil {
			return nil, err
		}
		h = redact(h)
		out = append(out, h)
	}
	return out, rows.Err()
}

func (s *Service) Get(ctx context.Context, id string) (Host, error) {
	row := s.pool.QueryRow(ctx, hostSelect+` WHERE id=$1`, id)
	h, err := scanHost(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Host{}, ErrNotFound
	}
	return redact(h), err
}

// GetByHost is used by the PR endpoint to find the matching token for
// a remote URL's host. Returns the unredacted token because callers
// need it to authenticate upstream.
func (s *Service) GetByHost(ctx context.Context, host string) (Host, error) {
	row := s.pool.QueryRow(ctx, hostSelect+` WHERE host=$1`, host)
	h, err := scanHost(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Host{}, ErrNotFound
	}
	return h, err
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (Host, error) {
	if err := validateKind(req.Kind); err != nil {
		return Host{}, err
	}
	if strings.TrimSpace(req.Host) == "" {
		return Host{}, errors.New("host is required")
	}
	if strings.TrimSpace(req.Token) == "" {
		return Host{}, errors.New("token is required")
	}
	row := s.pool.QueryRow(ctx, `
        INSERT INTO git_hosts (kind, host, name, token)
        VALUES ($1, $2, $3, $4)
        RETURNING id, kind, host, name, token, enabled, created_at, updated_at`,
		string(req.Kind), strings.TrimSpace(req.Host),
		strings.TrimSpace(req.Name), req.Token)
	h, err := scanHost(row)
	if err != nil {
		return Host{}, fmt.Errorf("insert git host: %w", err)
	}
	return redact(h), nil
}

func (s *Service) Update(ctx context.Context, id string, req UpdateRequest) (Host, error) {
	current, err := s.GetByID(ctx, id)
	if err != nil {
		return Host{}, err
	}
	if req.Kind != nil {
		if err := validateKind(*req.Kind); err != nil {
			return Host{}, err
		}
		current.Kind = *req.Kind
	}
	if req.Host != nil {
		current.Host = strings.TrimSpace(*req.Host)
	}
	if req.Name != nil {
		current.Name = strings.TrimSpace(*req.Name)
	}
	if req.Token != nil && *req.Token != "" {
		current.Token = *req.Token
	}
	if req.Enabled != nil {
		current.Enabled = *req.Enabled
	}
	row := s.pool.QueryRow(ctx, `
        UPDATE git_hosts
        SET kind=$1, host=$2, name=$3, token=$4, enabled=$5, updated_at=NOW()
        WHERE id=$6
        RETURNING id, kind, host, name, token, enabled, created_at, updated_at`,
		string(current.Kind), current.Host, current.Name, current.Token,
		current.Enabled, id)
	h, err := scanHost(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Host{}, ErrNotFound
	}
	if err != nil {
		return Host{}, fmt.Errorf("update git host: %w", err)
	}
	return redact(h), nil
}

// GetByID fetches the unredacted host (including the raw token) for
// internal use during update / PR listing. Not exposed via HTTP.
func (s *Service) GetByID(ctx context.Context, id string) (Host, error) {
	row := s.pool.QueryRow(ctx, hostSelect+` WHERE id=$1`, id)
	h, err := scanHost(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Host{}, ErrNotFound
	}
	return h, err
}

func (s *Service) Delete(ctx context.Context, id string) error {
	res, err := s.pool.Exec(ctx, `DELETE FROM git_hosts WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("delete git host: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func validateKind(k Kind) error {
	switch k {
	case KindGitHub, KindGitea, KindGitLab:
		return nil
	default:
		return fmt.Errorf("kind must be github|gitea|gitlab")
	}
}

func redact(h Host) Host {
	if h.Token != "" {
		h.TokenMask = "•••• " + lastN(h.Token, 4)
	}
	h.Token = ""
	return h
}

func lastN(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}

type rowScanner interface{ Scan(dest ...any) error }

func scanHost(row rowScanner) (Host, error) {
	var h Host
	var kind string
	if err := row.Scan(&h.ID, &kind, &h.Host, &h.Name, &h.Token,
		&h.Enabled, &h.CreatedAt, &h.UpdatedAt); err != nil {
		return Host{}, err
	}
	h.Kind = Kind(kind)
	return h, nil
}

// ── Remote detection ────────────────────────────────────────────

// Remote is what we figure out from `git remote get-url origin` plus
// the hosts table. `Kind` is filled when a matching host row exists,
// otherwise empty (UI shows "configure a token for <Host>").
type Remote struct {
	URL      string `json:"url"`
	Host     string `json:"host"`
	Owner    string `json:"owner"`
	Repo     string `json:"repo"`
	Kind     Kind   `json:"kind,omitempty"`
	HasToken bool   `json:"has_token"`
	WebURL   string `json:"web_url,omitempty"`
}

// DetectRemote reads `git remote get-url origin` from `dir` and parses
// host / owner / repo. Falls through gracefully — non-git directories
// or repos without an `origin` produce a typed error.
func (s *Service) DetectRemote(ctx context.Context, dir string) (Remote, error) {
	if !filepath.IsAbs(dir) {
		return Remote{}, errors.New("path must be absolute")
	}
	cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(cctx, "git", "-C", dir,
		"remote", "get-url", "origin").Output()
	if err != nil {
		return Remote{}, fmt.Errorf("git remote: %w", err)
	}
	rawURL := strings.TrimSpace(string(out))
	host, owner, repo, web, err := parseRemoteURL(rawURL)
	if err != nil {
		return Remote{}, err
	}
	rem := Remote{URL: rawURL, Host: host, Owner: owner, Repo: repo, WebURL: web}
	hostRow, err := s.GetByHost(ctx, host)
	if err == nil {
		rem.Kind = hostRow.Kind
		rem.HasToken = hostRow.Token != ""
	}
	return rem, nil
}

// parseRemoteURL accepts both the SSH shorthand (`git@host:owner/repo`)
// and the HTTPS form (`https://host/owner/repo[.git]`). Returns host,
// owner, repo, and a best-effort web URL for the repo root.
func parseRemoteURL(raw string) (host, owner, repo, web string, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", "", "", errors.New("empty remote URL")
	}
	switch {
	case strings.HasPrefix(raw, "git@"):
		// git@host:owner/repo[.git]
		rest := strings.TrimPrefix(raw, "git@")
		i := strings.Index(rest, ":")
		if i < 0 {
			return "", "", "", "", fmt.Errorf("unrecognized SSH remote: %s", raw)
		}
		host = rest[:i]
		path := rest[i+1:]
		owner, repo, err = splitOwnerRepo(path)
		if err != nil {
			return "", "", "", "", err
		}
	default:
		u, perr := url.Parse(raw)
		if perr != nil || u.Host == "" {
			return "", "", "", "", fmt.Errorf("parse remote: %w", perr)
		}
		host = u.Host
		owner, repo, err = splitOwnerRepo(strings.TrimPrefix(u.Path, "/"))
		if err != nil {
			return "", "", "", "", err
		}
	}
	web = "https://" + host + "/" + owner + "/" + repo
	return host, owner, repo, web, nil
}

func splitOwnerRepo(path string) (string, string, error) {
	path = strings.TrimSuffix(path, ".git")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("remote path needs owner/repo: %s", path)
	}
	// GitLab supports nested groups (a/b/c/repo). Take everything but
	// the last segment as the owner namespace.
	owner := strings.Join(parts[:len(parts)-1], "/")
	return owner, parts[len(parts)-1], nil
}

// ── Pull request listing ────────────────────────────────────────

// PullRequest is the trimmed-down view we surface in the panel.
type PullRequest struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	State     string    `json:"state"` // open | closed | merged
	Author    string    `json:"author"`
	Head      string    `json:"head"` // source branch
	Base      string    `json:"base"` // target branch
	URL       string    `json:"url"`  // web URL
	Draft     bool      `json:"draft"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (s *Service) ListPullRequests(ctx context.Context, dir, state string) (Remote, []PullRequest, error) {
	rem, err := s.DetectRemote(ctx, dir)
	if err != nil {
		return Remote{}, nil, err
	}
	if !rem.HasToken {
		return rem, nil, ErrNoTokenForHost
	}
	hostRow, err := s.GetByHost(ctx, rem.Host)
	if err != nil {
		return rem, nil, err
	}
	if state == "" {
		state = "open"
	}
	switch hostRow.Kind {
	case KindGitHub:
		prs, err := s.listGitHubPRs(ctx, hostRow, rem, state)
		return rem, prs, err
	case KindGitea:
		prs, err := s.listGiteaPRs(ctx, hostRow, rem, state)
		return rem, prs, err
	case KindGitLab:
		prs, err := s.listGitLabMRs(ctx, hostRow, rem, state)
		return rem, prs, err
	default:
		return rem, nil, fmt.Errorf("unsupported host kind: %s", hostRow.Kind)
	}
}

// CreatePRRequest is the payload for a new pull request.
type CreatePRRequest struct {
	Dir   string `json:"dir"`
	Title string `json:"title"`
	Body  string `json:"body"`
	Head  string `json:"head"` // source branch (required)
	Base  string `json:"base"` // target branch — defaults to the repo's default branch when empty
	Draft bool   `json:"draft"`
}

// MergePRRequest is the payload for merging an existing PR.
type MergePRRequest struct {
	Dir           string `json:"dir"`
	Number        int    `json:"number"`
	Method        string `json:"method"` // "squash" | "merge" | "rebase" (GitHub naming; mapped per platform)
	CommitTitle   string `json:"commit_title"`
	CommitMessage string `json:"commit_message"`
	DeleteBranch  bool   `json:"delete_branch"`
}

// CheckRun is one CI/check entry attached to a PR's head commit.
// Status/Conclusion follow the GitHub Checks API vocabulary so the
// UI can render a single icon set across hosts; Gitea / GitLab
// values are normalised in their respective adapters.
type CheckRun struct {
	Name       string    `json:"name"`
	Status     string    `json:"status"`     // queued | in_progress | completed
	Conclusion string    `json:"conclusion"` // success | failure | neutral | cancelled | skipped | timed_out | action_required
	URL        string    `json:"url"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// CreatePullRequest opens a PR on the host that owns dir's remote.
// Returns the created PR's API representation. Token must exist for
// the host (ErrNoTokenForHost otherwise).
func (s *Service) CreatePullRequest(ctx context.Context, req CreatePRRequest) (PullRequest, error) {
	rem, hostRow, err := s.resolveHost(ctx, req.Dir)
	if err != nil {
		return PullRequest{}, err
	}
	if req.Head == "" {
		return PullRequest{}, errors.New("head branch required")
	}
	if req.Title == "" {
		return PullRequest{}, errors.New("title required")
	}
	switch hostRow.Kind {
	case KindGitHub:
		return s.createGitHubPR(ctx, hostRow, rem, req)
	case KindGitea:
		return s.createGiteaPR(ctx, hostRow, rem, req)
	case KindGitLab:
		return s.createGitLabMR(ctx, hostRow, rem, req)
	default:
		return PullRequest{}, fmt.Errorf("unsupported host kind: %s", hostRow.Kind)
	}
}

// MergePullRequest merges an existing PR with the requested method.
// Squash is the default. When DeleteBranch is true the head branch
// is removed after a successful merge (GitHub only — Gitea / GitLab
// have their own delete flag in the merge call).
func (s *Service) MergePullRequest(ctx context.Context, req MergePRRequest) (PullRequest, error) {
	rem, hostRow, err := s.resolveHost(ctx, req.Dir)
	if err != nil {
		return PullRequest{}, err
	}
	if req.Number <= 0 {
		return PullRequest{}, errors.New("number required")
	}
	if req.Method == "" {
		req.Method = "squash"
	}
	switch hostRow.Kind {
	case KindGitHub:
		return s.mergeGitHubPR(ctx, hostRow, rem, req)
	case KindGitea:
		return s.mergeGiteaPR(ctx, hostRow, rem, req)
	case KindGitLab:
		return s.mergeGitLabMR(ctx, hostRow, rem, req)
	default:
		return PullRequest{}, fmt.Errorf("unsupported host kind: %s", hostRow.Kind)
	}
}

// PRChecks returns the check runs attached to a PR's head commit.
// GitHub-only in this iteration; Gitea / GitLab return an empty
// slice with no error so the UI can degrade gracefully.
func (s *Service) PRChecks(ctx context.Context, dir string, number int) ([]CheckRun, error) {
	rem, hostRow, err := s.resolveHost(ctx, dir)
	if err != nil {
		return nil, err
	}
	if number <= 0 {
		return nil, errors.New("number required")
	}
	if hostRow.Kind != KindGitHub {
		// Gitea / GitLab have their own check APIs but the surface
		// differs enough that we'd lose more in normalisation than
		// we'd gain. Return empty rather than error so the UI hides
		// the section gracefully.
		return []CheckRun{}, nil
	}
	return s.githubChecks(ctx, hostRow, rem, number)
}

// resolveHost is the shared prelude for create / merge / checks:
// detect the remote, confirm a token exists, fetch the host row.
func (s *Service) resolveHost(ctx context.Context, dir string) (Remote, Host, error) {
	rem, err := s.DetectRemote(ctx, dir)
	if err != nil {
		return Remote{}, Host{}, err
	}
	if !rem.HasToken {
		return rem, Host{}, ErrNoTokenForHost
	}
	hostRow, err := s.GetByHost(ctx, rem.Host)
	if err != nil {
		return rem, Host{}, err
	}
	return rem, hostRow, nil
}

func (s *Service) listGitHubPRs(ctx context.Context, h Host, rem Remote, state string) ([]PullRequest, error) {
	// github.com → api.github.com; GitHub Enterprise stays at the
	// same host under /api/v3.
	base := "https://api.github.com"
	if h.Host != "github.com" {
		base = "https://" + h.Host + "/api/v3"
	}
	u := fmt.Sprintf("%s/repos/%s/%s/pulls?state=%s&per_page=20",
		base, rem.Owner, rem.Repo, url.QueryEscape(state))
	body, err := s.fetch(ctx, u, "Bearer "+h.Token, "application/vnd.github+json")
	if err != nil {
		return nil, err
	}
	var raw []struct {
		Number    int       `json:"number"`
		Title     string    `json:"title"`
		State     string    `json:"state"`
		Draft     bool      `json:"draft"`
		HTMLURL   string    `json:"html_url"`
		UpdatedAt time.Time `json:"updated_at"`
		User      struct {
			Login string `json:"login"`
		} `json:"user"`
		Head struct {
			Ref string `json:"ref"`
		} `json:"head"`
		Base struct {
			Ref string `json:"ref"`
		} `json:"base"`
		MergedAt *time.Time `json:"merged_at"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("github decode: %w", err)
	}
	out := make([]PullRequest, 0, len(raw))
	for _, p := range raw {
		st := p.State
		if p.MergedAt != nil {
			st = "merged"
		}
		out = append(out, PullRequest{
			Number:    p.Number,
			Title:     p.Title,
			State:     st,
			Author:    p.User.Login,
			Head:      p.Head.Ref,
			Base:      p.Base.Ref,
			URL:       p.HTMLURL,
			Draft:     p.Draft,
			UpdatedAt: p.UpdatedAt,
		})
	}
	return out, nil
}

func (s *Service) listGiteaPRs(ctx context.Context, h Host, rem Remote, state string) ([]PullRequest, error) {
	u := fmt.Sprintf("https://%s/api/v1/repos/%s/%s/pulls?state=%s&limit=20",
		h.Host, rem.Owner, rem.Repo, url.QueryEscape(state))
	body, err := s.fetch(ctx, u, "token "+h.Token, "application/json")
	if err != nil {
		return nil, err
	}
	var raw []struct {
		Number    int       `json:"number"`
		Title     string    `json:"title"`
		State     string    `json:"state"`
		HTMLURL   string    `json:"html_url"`
		UpdatedAt time.Time `json:"updated_at"`
		User      struct {
			Login string `json:"login"`
		} `json:"user"`
		Head struct {
			Ref string `json:"ref"`
		} `json:"head"`
		Base struct {
			Ref string `json:"ref"`
		} `json:"base"`
		Merged bool `json:"merged"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("gitea decode: %w", err)
	}
	out := make([]PullRequest, 0, len(raw))
	for _, p := range raw {
		st := p.State
		if p.Merged {
			st = "merged"
		}
		out = append(out, PullRequest{
			Number:    p.Number,
			Title:     p.Title,
			State:     st,
			Author:    p.User.Login,
			Head:      p.Head.Ref,
			Base:      p.Base.Ref,
			URL:       p.HTMLURL,
			UpdatedAt: p.UpdatedAt,
		})
	}
	return out, nil
}

func (s *Service) listGitLabMRs(ctx context.Context, h Host, rem Remote, state string) ([]PullRequest, error) {
	// GitLab "MR" map: opened/closed/merged. Allow `open` as alias.
	mapState := state
	if state == "open" {
		mapState = "opened"
	}
	projectID := url.PathEscape(rem.Owner + "/" + rem.Repo)
	u := fmt.Sprintf("https://%s/api/v4/projects/%s/merge_requests?state=%s&per_page=20",
		h.Host, projectID, url.QueryEscape(mapState))
	body, err := s.fetch(ctx, u, "Bearer "+h.Token, "application/json")
	if err != nil {
		return nil, err
	}
	var raw []struct {
		IID       int       `json:"iid"`
		Title     string    `json:"title"`
		State     string    `json:"state"`
		WebURL    string    `json:"web_url"`
		UpdatedAt time.Time `json:"updated_at"`
		Draft     bool      `json:"draft"`
		Author    struct {
			Username string `json:"username"`
		} `json:"author"`
		SourceBranch string `json:"source_branch"`
		TargetBranch string `json:"target_branch"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("gitlab decode: %w", err)
	}
	out := make([]PullRequest, 0, len(raw))
	for _, p := range raw {
		out = append(out, PullRequest{
			Number:    p.IID,
			Title:     p.Title,
			State:     normaliseGitlabState(p.State),
			Author:    p.Author.Username,
			Head:      p.SourceBranch,
			Base:      p.TargetBranch,
			URL:       p.WebURL,
			Draft:     p.Draft,
			UpdatedAt: p.UpdatedAt,
		})
	}
	return out, nil
}

func normaliseGitlabState(s string) string {
	switch s {
	case "opened":
		return "open"
	default:
		return s
	}
}

// ── GitHub: create / merge / checks ────────────────────────────────

// githubAPIBase returns the host's GitHub API base URL. github.com
// uses api.github.com; GitHub Enterprise serves /api/v3 from the
// same hostname.
func githubAPIBase(host string) string {
	if host == "github.com" {
		return "https://api.github.com"
	}
	return "https://" + host + "/api/v3"
}

// pullRequestFromGitHubResponse normalises a single PR JSON envelope
// returned by either /pulls or /pulls/{n}/merge into the unified
// PullRequest shape the UI consumes.
type githubPRResponse struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	State     string    `json:"state"`
	Draft     bool      `json:"draft"`
	HTMLURL   string    `json:"html_url"`
	UpdatedAt time.Time `json:"updated_at"`
	User      struct {
		Login string `json:"login"`
	} `json:"user"`
	Head struct {
		Ref string `json:"ref"`
	} `json:"head"`
	Base struct {
		Ref string `json:"ref"`
	} `json:"base"`
	MergedAt *time.Time `json:"merged_at"`
}

func (p githubPRResponse) toPullRequest() PullRequest {
	st := p.State
	if p.MergedAt != nil {
		st = "merged"
	}
	return PullRequest{
		Number:    p.Number,
		Title:     p.Title,
		State:     st,
		Author:    p.User.Login,
		Head:      p.Head.Ref,
		Base:      p.Base.Ref,
		URL:       p.HTMLURL,
		Draft:     p.Draft,
		UpdatedAt: p.UpdatedAt,
	}
}

func (s *Service) createGitHubPR(ctx context.Context, h Host, rem Remote, req CreatePRRequest) (PullRequest, error) {
	payload := map[string]any{
		"title": req.Title,
		"body":  req.Body,
		"head":  req.Head,
		"draft": req.Draft,
	}
	if req.Base != "" {
		payload["base"] = req.Base
	} else {
		// Resolve default branch from the repo metadata so the
		// caller doesn't have to know whether it's main / master /
		// trunk. One extra request, cached implicitly via HTTP.
		def, err := s.githubDefaultBranch(ctx, h, rem)
		if err != nil {
			return PullRequest{}, fmt.Errorf("resolve base: %w", err)
		}
		payload["base"] = def
	}
	u := fmt.Sprintf("%s/repos/%s/%s/pulls", githubAPIBase(h.Host), rem.Owner, rem.Repo)
	body, err := s.do(ctx, http.MethodPost, u, "Bearer "+h.Token, "application/vnd.github+json", payload)
	if err != nil {
		return PullRequest{}, err
	}
	var raw githubPRResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return PullRequest{}, fmt.Errorf("github decode: %w", err)
	}
	return raw.toPullRequest(), nil
}

func (s *Service) mergeGitHubPR(ctx context.Context, h Host, rem Remote, req MergePRRequest) (PullRequest, error) {
	// GitHub: PUT /repos/{o}/{r}/pulls/{n}/merge. merge_method is
	// "merge" | "squash" | "rebase". The endpoint returns a small
	// status envelope, NOT the PR — we re-fetch the PR afterwards
	// so the caller gets a complete record (and the post-merge
	// state). Delete-branch is a separate DELETE call.
	mergePayload := map[string]any{
		"merge_method": req.Method,
	}
	if req.CommitTitle != "" {
		mergePayload["commit_title"] = req.CommitTitle
	}
	if req.CommitMessage != "" {
		mergePayload["commit_message"] = req.CommitMessage
	}
	mergeURL := fmt.Sprintf("%s/repos/%s/%s/pulls/%d/merge",
		githubAPIBase(h.Host), rem.Owner, rem.Repo, req.Number)
	if _, err := s.do(ctx, http.MethodPut, mergeURL, "Bearer "+h.Token, "application/vnd.github+json", mergePayload); err != nil {
		return PullRequest{}, err
	}
	// Re-fetch the PR to capture merged_at + final state.
	prURL := fmt.Sprintf("%s/repos/%s/%s/pulls/%d",
		githubAPIBase(h.Host), rem.Owner, rem.Repo, req.Number)
	prBody, err := s.do(ctx, http.MethodGet, prURL, "Bearer "+h.Token, "application/vnd.github+json", nil)
	if err != nil {
		// Merge succeeded; failing to refetch is non-fatal — caller
		// gets a minimal PR record reflecting what we know.
		return PullRequest{Number: req.Number, State: "merged"}, nil
	}
	var raw githubPRResponse
	if err := json.Unmarshal(prBody, &raw); err != nil {
		return PullRequest{Number: req.Number, State: "merged"}, nil
	}
	merged := raw.toPullRequest()
	// Best-effort branch deletion after the merge. Errors are
	// logged (via the caller) but never block the merge result.
	if req.DeleteBranch && merged.Head != "" {
		delURL := fmt.Sprintf("%s/repos/%s/%s/git/refs/heads/%s",
			githubAPIBase(h.Host), rem.Owner, rem.Repo,
			url.PathEscape(merged.Head))
		_, _ = s.do(ctx, http.MethodDelete, delURL, "Bearer "+h.Token, "application/vnd.github+json", nil)
	}
	return merged, nil
}

// githubDefaultBranch returns the repo's configured default branch
// (e.g. "main" or "master"). Used by CreatePR when the caller
// omitted Base.
func (s *Service) githubDefaultBranch(ctx context.Context, h Host, rem Remote) (string, error) {
	u := fmt.Sprintf("%s/repos/%s/%s", githubAPIBase(h.Host), rem.Owner, rem.Repo)
	body, err := s.do(ctx, http.MethodGet, u, "Bearer "+h.Token, "application/vnd.github+json", nil)
	if err != nil {
		return "", err
	}
	var raw struct {
		DefaultBranch string `json:"default_branch"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return "", err
	}
	if raw.DefaultBranch == "" {
		return "", errors.New("repo has no default_branch")
	}
	return raw.DefaultBranch, nil
}

func (s *Service) githubChecks(ctx context.Context, h Host, rem Remote, number int) ([]CheckRun, error) {
	// GET the PR to learn the head SHA, then fetch its check runs.
	// Two requests is wasteful vs storing PRs in DB, but we want
	// fresh check state on every poll.
	prURL := fmt.Sprintf("%s/repos/%s/%s/pulls/%d",
		githubAPIBase(h.Host), rem.Owner, rem.Repo, number)
	prBody, err := s.do(ctx, http.MethodGet, prURL, "Bearer "+h.Token, "application/vnd.github+json", nil)
	if err != nil {
		return nil, err
	}
	var pr struct {
		Head struct {
			SHA string `json:"sha"`
		} `json:"head"`
	}
	if err := json.Unmarshal(prBody, &pr); err != nil {
		return nil, fmt.Errorf("github decode pr: %w", err)
	}
	if pr.Head.SHA == "" {
		return nil, errors.New("pr has no head sha")
	}
	checksURL := fmt.Sprintf("%s/repos/%s/%s/commits/%s/check-runs?per_page=50",
		githubAPIBase(h.Host), rem.Owner, rem.Repo, pr.Head.SHA)
	body, err := s.do(ctx, http.MethodGet, checksURL, "Bearer "+h.Token, "application/vnd.github+json", nil)
	if err != nil {
		return nil, err
	}
	var raw struct {
		CheckRuns []struct {
			Name        string     `json:"name"`
			Status      string     `json:"status"`
			Conclusion  string     `json:"conclusion"`
			HTMLURL     string     `json:"html_url"`
			CompletedAt *time.Time `json:"completed_at"`
			StartedAt   *time.Time `json:"started_at"`
		} `json:"check_runs"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("github decode checks: %w", err)
	}
	out := make([]CheckRun, 0, len(raw.CheckRuns))
	for _, c := range raw.CheckRuns {
		ts := time.Time{}
		if c.CompletedAt != nil {
			ts = *c.CompletedAt
		} else if c.StartedAt != nil {
			ts = *c.StartedAt
		}
		out = append(out, CheckRun{
			Name:       c.Name,
			Status:     c.Status,
			Conclusion: c.Conclusion,
			URL:        c.HTMLURL,
			UpdatedAt:  ts,
		})
	}
	return out, nil
}

// ── Gitea: create / merge ─────────────────────────────────────────

func (s *Service) createGiteaPR(ctx context.Context, h Host, rem Remote, req CreatePRRequest) (PullRequest, error) {
	payload := map[string]any{
		"title": req.Title,
		"body":  req.Body,
		"head":  req.Head,
	}
	if req.Base != "" {
		payload["base"] = req.Base
	} else {
		// Gitea exposes default_branch in the repo metadata too.
		def, err := s.giteaDefaultBranch(ctx, h, rem)
		if err != nil {
			return PullRequest{}, fmt.Errorf("resolve base: %w", err)
		}
		payload["base"] = def
	}
	u := fmt.Sprintf("https://%s/api/v1/repos/%s/%s/pulls",
		h.Host, rem.Owner, rem.Repo)
	body, err := s.do(ctx, http.MethodPost, u, "token "+h.Token, "application/json", payload)
	if err != nil {
		return PullRequest{}, err
	}
	return decodeGiteaPR(body)
}

func (s *Service) mergeGiteaPR(ctx context.Context, h Host, rem Remote, req MergePRRequest) (PullRequest, error) {
	// Gitea merge: POST /repos/{o}/{r}/pulls/{n}/merge with
	// Do=squash|merge|rebase. delete_branch_after_merge is part
	// of the same payload.
	payload := map[string]any{
		"Do":                        giteaMergeMethod(req.Method),
		"delete_branch_after_merge": req.DeleteBranch,
	}
	if req.CommitTitle != "" {
		payload["MergeTitleField"] = req.CommitTitle
	}
	if req.CommitMessage != "" {
		payload["MergeMessageField"] = req.CommitMessage
	}
	mergeURL := fmt.Sprintf("https://%s/api/v1/repos/%s/%s/pulls/%d/merge",
		h.Host, rem.Owner, rem.Repo, req.Number)
	if _, err := s.do(ctx, http.MethodPost, mergeURL, "token "+h.Token, "application/json", payload); err != nil {
		return PullRequest{}, err
	}
	// Re-fetch the PR like the GitHub path. Errors return a minimal
	// record so the caller still knows the merge succeeded.
	prURL := fmt.Sprintf("https://%s/api/v1/repos/%s/%s/pulls/%d",
		h.Host, rem.Owner, rem.Repo, req.Number)
	prBody, err := s.do(ctx, http.MethodGet, prURL, "token "+h.Token, "application/json", nil)
	if err != nil {
		return PullRequest{Number: req.Number, State: "merged"}, nil
	}
	pr, err := decodeGiteaPR(prBody)
	if err != nil {
		return PullRequest{Number: req.Number, State: "merged"}, nil
	}
	return pr, nil
}

func (s *Service) giteaDefaultBranch(ctx context.Context, h Host, rem Remote) (string, error) {
	u := fmt.Sprintf("https://%s/api/v1/repos/%s/%s", h.Host, rem.Owner, rem.Repo)
	body, err := s.do(ctx, http.MethodGet, u, "token "+h.Token, "application/json", nil)
	if err != nil {
		return "", err
	}
	var raw struct {
		DefaultBranch string `json:"default_branch"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return "", err
	}
	if raw.DefaultBranch == "" {
		return "", errors.New("repo has no default_branch")
	}
	return raw.DefaultBranch, nil
}

func giteaMergeMethod(m string) string {
	switch m {
	case "squash", "merge", "rebase":
		return m
	default:
		return "squash"
	}
}

func decodeGiteaPR(body []byte) (PullRequest, error) {
	var p struct {
		Number    int       `json:"number"`
		Title     string    `json:"title"`
		State     string    `json:"state"`
		HTMLURL   string    `json:"html_url"`
		UpdatedAt time.Time `json:"updated_at"`
		User      struct {
			Login string `json:"login"`
		} `json:"user"`
		Head struct {
			Ref string `json:"ref"`
		} `json:"head"`
		Base struct {
			Ref string `json:"ref"`
		} `json:"base"`
		Merged bool `json:"merged"`
	}
	if err := json.Unmarshal(body, &p); err != nil {
		return PullRequest{}, fmt.Errorf("gitea decode: %w", err)
	}
	st := p.State
	if p.Merged {
		st = "merged"
	}
	return PullRequest{
		Number:    p.Number,
		Title:     p.Title,
		State:     st,
		Author:    p.User.Login,
		Head:      p.Head.Ref,
		Base:      p.Base.Ref,
		URL:       p.HTMLURL,
		UpdatedAt: p.UpdatedAt,
	}, nil
}

// ── GitLab: create / merge (merge requests) ───────────────────────

func (s *Service) createGitLabMR(ctx context.Context, h Host, rem Remote, req CreatePRRequest) (PullRequest, error) {
	payload := map[string]any{
		"source_branch": req.Head,
		"title":         req.Title,
		"description":   req.Body,
	}
	if req.Base != "" {
		payload["target_branch"] = req.Base
	} else {
		def, err := s.gitlabDefaultBranch(ctx, h, rem)
		if err != nil {
			return PullRequest{}, fmt.Errorf("resolve target_branch: %w", err)
		}
		payload["target_branch"] = def
	}
	projectID := url.PathEscape(rem.Owner + "/" + rem.Repo)
	u := fmt.Sprintf("https://%s/api/v4/projects/%s/merge_requests",
		h.Host, projectID)
	body, err := s.do(ctx, http.MethodPost, u, "Bearer "+h.Token, "application/json", payload)
	if err != nil {
		return PullRequest{}, err
	}
	return decodeGitLabMR(body)
}

func (s *Service) mergeGitLabMR(ctx context.Context, h Host, rem Remote, req MergePRRequest) (PullRequest, error) {
	// PUT /projects/{id}/merge_requests/{iid}/merge. Squash and
	// delete_branch are separate flags; method has no direct
	// equivalent for "rebase" so we map: squash → squash=true,
	// merge → squash=false, rebase falls back to squash=false +
	// the caller doing a rebase separately.
	payload := map[string]any{
		"should_remove_source_branch": req.DeleteBranch,
	}
	if req.Method == "squash" {
		payload["squash"] = true
	}
	if req.CommitMessage != "" {
		payload["merge_commit_message"] = req.CommitMessage
	}
	if req.CommitTitle != "" && req.Method == "squash" {
		payload["squash_commit_message"] = req.CommitTitle
	}
	projectID := url.PathEscape(rem.Owner + "/" + rem.Repo)
	mergeURL := fmt.Sprintf("https://%s/api/v4/projects/%s/merge_requests/%d/merge",
		h.Host, projectID, req.Number)
	body, err := s.do(ctx, http.MethodPut, mergeURL, "Bearer "+h.Token, "application/json", payload)
	if err != nil {
		return PullRequest{}, err
	}
	// GitLab returns the updated MR directly from the merge call.
	return decodeGitLabMR(body)
}

func (s *Service) gitlabDefaultBranch(ctx context.Context, h Host, rem Remote) (string, error) {
	projectID := url.PathEscape(rem.Owner + "/" + rem.Repo)
	u := fmt.Sprintf("https://%s/api/v4/projects/%s", h.Host, projectID)
	body, err := s.do(ctx, http.MethodGet, u, "Bearer "+h.Token, "application/json", nil)
	if err != nil {
		return "", err
	}
	var raw struct {
		DefaultBranch string `json:"default_branch"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return "", err
	}
	if raw.DefaultBranch == "" {
		return "", errors.New("project has no default_branch")
	}
	return raw.DefaultBranch, nil
}

func decodeGitLabMR(body []byte) (PullRequest, error) {
	var p struct {
		IID       int       `json:"iid"`
		Title     string    `json:"title"`
		State     string    `json:"state"`
		WebURL    string    `json:"web_url"`
		UpdatedAt time.Time `json:"updated_at"`
		Author    struct {
			Username string `json:"username"`
		} `json:"author"`
		SourceBranch string `json:"source_branch"`
		TargetBranch string `json:"target_branch"`
		Draft        bool   `json:"draft"`
	}
	if err := json.Unmarshal(body, &p); err != nil {
		return PullRequest{}, fmt.Errorf("gitlab decode: %w", err)
	}
	return PullRequest{
		Number:    p.IID,
		Title:     p.Title,
		State:     normaliseGitlabState(p.State),
		Author:    p.Author.Username,
		Head:      p.SourceBranch,
		Base:      p.TargetBranch,
		URL:       p.WebURL,
		Draft:     p.Draft,
		UpdatedAt: p.UpdatedAt,
	}, nil
}

func (s *Service) fetch(ctx context.Context, u, auth, accept string) ([]byte, error) {
	return s.do(ctx, http.MethodGet, u, auth, accept, nil)
}

// do is the shared HTTP transport for read + write calls against
// GitHub / Gitea / GitLab. Pass body=nil for GET / DELETE; supply a
// JSON-serializable payload for POST / PUT / PATCH and we encode it.
// The 2 MiB response cap matches the original fetch — sane upstream
// payloads stay well under, runaway responses get rejected.
func (s *Service) do(ctx context.Context, method, u, auth, accept string, body any) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", auth)
	req.Header.Set("Accept", accept)
	req.Header.Set("User-Agent", "opendray-inspector")
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("upstream %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return respBody, nil
}

// ── HTTP handlers ───────────────────────────────────────────────

type Handlers struct {
	svc *Service
	log *slog.Logger
}

func NewHandlers(svc *Service, log *slog.Logger) *Handlers {
	if log == nil {
		log = slog.Default()
	}
	return &Handlers{svc: svc, log: log.With("component", "githost.http")}
}

func (h *Handlers) Mount(r chi.Router) {
	r.Route("/git-hosts", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.create)
		r.Route("/{id}", func(r chi.Router) {
			r.Put("/", h.update)
			r.Delete("/", h.del)
		})
	})
	// Remote detection + PR listing live under /git/* alongside the
	// status / log / diff endpoints in internal/git for symmetry on
	// the client side.
	r.Get("/git/remote", h.remote)
	r.Get("/git/prs", h.prs)
	r.Post("/git/prs", h.createPR)
	r.Post("/git/prs/{number}/merge", h.mergePR)
	r.Get("/git/prs/{number}/checks", h.prChecks)
}

// createPR mounts POST /git/prs. Body matches CreatePRRequest.
// Returns 201 with the created PR envelope on success.
func (h *Handlers) createPR(w http.ResponseWriter, r *http.Request) {
	var req CreatePRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("decode body: %w", err))
		return
	}
	if req.Dir == "" {
		writeError(w, http.StatusBadRequest, errors.New("dir required"))
		return
	}
	pr, err := h.svc.CreatePullRequest(r.Context(), req)
	if err != nil {
		writeError(w, statusFromGitErr(err), err)
		return
	}
	writeJSON(w, http.StatusCreated, pr)
}

// mergePR mounts POST /git/prs/{number}/merge. URL number wins
// over body number when both are present.
func (h *Handlers) mergePR(w http.ResponseWriter, r *http.Request) {
	var req MergePRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("decode body: %w", err))
		return
	}
	if num := chi.URLParam(r, "number"); num != "" {
		n, err := strconv.Atoi(num)
		if err != nil || n <= 0 {
			writeError(w, http.StatusBadRequest, errors.New("invalid number"))
			return
		}
		req.Number = n
	}
	if req.Dir == "" {
		writeError(w, http.StatusBadRequest, errors.New("dir required"))
		return
	}
	pr, err := h.svc.MergePullRequest(r.Context(), req)
	if err != nil {
		writeError(w, statusFromGitErr(err), err)
		return
	}
	writeJSON(w, http.StatusOK, pr)
}

// prChecks mounts GET /git/prs/{number}/checks?path=<dir>.
func (h *Handlers) prChecks(w http.ResponseWriter, r *http.Request) {
	dir := r.URL.Query().Get("path")
	if dir == "" {
		writeError(w, http.StatusBadRequest, errors.New("path required"))
		return
	}
	num := chi.URLParam(r, "number")
	n, err := strconv.Atoi(num)
	if err != nil || n <= 0 {
		writeError(w, http.StatusBadRequest, errors.New("invalid number"))
		return
	}
	checks, err := h.svc.PRChecks(r.Context(), dir, n)
	if err != nil {
		writeError(w, statusFromGitErr(err), err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"checks": checks})
}

// statusFromGitErr maps the common sentinel errors to HTTP codes so
// the UI can react (e.g. show "configure token" hint for 404).
func statusFromGitErr(err error) int {
	if errors.Is(err, ErrNoTokenForHost) {
		return http.StatusUnauthorized
	}
	return http.StatusInternalServerError
}

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	hosts, err := h.svc.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"hosts": hosts})
}

func (h *Handlers) create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	host, err := h.svc.Create(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, host)
}

func (h *Handlers) update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	host, err := h.svc.Update(r.Context(), id, req)
	if err != nil {
		respond(w, err)
		return
	}
	writeJSON(w, http.StatusOK, host)
}

func (h *Handlers) del(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.Delete(r.Context(), id); err != nil {
		respond(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) remote(w http.ResponseWriter, r *http.Request) {
	dir := strings.TrimSpace(r.URL.Query().Get("path"))
	if dir == "" {
		writeError(w, http.StatusBadRequest, errors.New("path is required"))
		return
	}
	rem, err := h.svc.DetectRemote(r.Context(), dir)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, rem)
}

func (h *Handlers) prs(w http.ResponseWriter, r *http.Request) {
	dir := strings.TrimSpace(r.URL.Query().Get("path"))
	if dir == "" {
		writeError(w, http.StatusBadRequest, errors.New("path is required"))
		return
	}
	state := r.URL.Query().Get("state")
	rem, prs, err := h.svc.ListPullRequests(r.Context(), dir, state)
	if errors.Is(err, ErrNoTokenForHost) {
		writeJSON(w, http.StatusOK, map[string]any{
			"remote":     rem,
			"prs":        []PullRequest{},
			"need_token": true,
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"remote": rem,
			"prs":    []PullRequest{},
			"error":  err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"remote": rem,
		"prs":    prs,
	})
}

func respond(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, err)
	default:
		writeError(w, http.StatusInternalServerError, err)
	}
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

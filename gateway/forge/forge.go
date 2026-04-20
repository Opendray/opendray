// Package forge implements a read-only Git-forge pull-request viewer
// backing the git-forge panel plugin. Three forges are supported:
// Gitea, GitHub, and GitLab. The user picks one via the plugin's
// configSchema field `forgeType`; the adapter dispatch in this file
// routes each public call to the matching per-forge file.
//
// The package exposes normalised types — [PullRequest], [DiffFile],
// [Comment] — so HTTP handlers and the Flutter client see one shape
// regardless of which forge answered. Per-forge JSON shapes are
// private to gitea.go / github.go / gitlab.go.
//
// Writes (create PR, merge, comment, approve) are intentionally out
// of scope. The product position is that writes flow through the
// Claude session, not the panel (see git-viewer's README).
package forge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ─── Configuration ───────────────────────────────────────────────

// Config holds the settings the HTTP handler resolves from the
// git-forge plugin's configSchema before calling any adapter. Empty
// BaseURL / Repo are caller errors — callers validate before dispatch.
type Config struct {
	// ForgeType selects the adapter: "gitea" | "github" | "gitlab".
	// Empty / unknown is rejected by Dispatch so a misconfigured
	// plugin fails fast instead of silently defaulting.
	ForgeType string

	// BaseURL is the forge's API root (or web root — GitHub.com
	// accepts both "https://api.github.com" and "https://github.com"
	// and we coerce to the API host internally).
	BaseURL string

	// Repo is "owner/name" (Gitea, GitHub) or the full project path
	// (GitLab — it gets URL-encoded by the adapter). Leading slashes
	// are trimmed.
	Repo string

	// Token is the forge API token. Empty is allowed for public
	// repos on GitHub; Gitea and GitLab PR endpoints typically need
	// one. Never logged.
	Token string

	// Timeout caps every HTTP call. The handler passes this through
	// from the plugin's commandTimeoutSec config field.
	Timeout time.Duration
}

// State is the PR state filter passed to List.
type State string

const (
	StateOpen   State = "open"
	StateClosed State = "closed"
	StateAll    State = "all"
)

// ─── Normalised domain types ─────────────────────────────────────

// PullRequest is the cross-forge PR summary. Fields not supplied by
// a particular forge are left zero-valued — callers render "—" or
// omit them. Numeric ID is always the forge's PR/MR number, not the
// internal database id.
type PullRequest struct {
	Number       int       `json:"number"`
	Title        string    `json:"title"`
	State        string    `json:"state"` // "open" | "closed" | "merged"
	Author       string    `json:"author"`
	AuthorAvatar string    `json:"authorAvatar,omitempty"`
	HeadRef      string    `json:"headRef"`
	HeadSHA      string    `json:"headSha,omitempty"` // commit SHA for checks lookup
	BaseRef      string    `json:"baseRef"`
	URL          string    `json:"url"`
	Body         string    `json:"body,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	CommentCount int       `json:"commentCount"`
	Draft        bool      `json:"draft,omitempty"`
}

// DiffFile is one file's entry inside a PR diff. Patch is the raw
// unified-diff text (what `git diff` emits for that file). Adapters
// that only return a single concatenated diff parse it into per-file
// entries before handing off.
type DiffFile struct {
	Path      string `json:"path"`
	OldPath   string `json:"oldPath,omitempty"`
	Status    string `json:"status"` // "added" | "modified" | "deleted" | "renamed"
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Patch     string `json:"patch"`
}

// Comment is a top-level PR / issue comment.
type Comment struct {
	ID        int64     `json:"id"`
	Author    string    `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
	URL       string    `json:"url,omitempty"`
}

// Review is one reviewer's verdict on a PR. Each forge has a slightly
// different state machine; we normalise to four values the UI can
// colour consistently:
//
//	"approved"          — explicit LGTM
//	"changes_requested" — blocks merge
//	"commented"         — left remarks without approving/blocking
//	"dismissed"         — review was dismissed by an admin
//
// Author + SubmittedAt identify the entry; Body is optional summary
// text the reviewer typed. For GitLab's approvals API (which doesn't
// carry commentary), Body will be empty and State will be "approved".
type Review struct {
	Author       string    `json:"author"`
	AuthorAvatar string    `json:"authorAvatar,omitempty"`
	State        string    `json:"state"`
	Body         string    `json:"body,omitempty"`
	SubmittedAt  time.Time `json:"submittedAt"`
}

// ReviewComment is one inline comment attached to a specific file
// and line inside a PR's diff. Kept separate from the top-level
// Comment type so callers can colocate it with the matching diff hunk
// rather than threading it into a chronological timeline.
//
// Line is the line number in the NEW file (post-change) that the
// comment targets. When a comment is on a deleted line the adapter
// may report the old-file line instead — forges differ; we accept
// whichever number the forge gives and trust the client to match.
// ReplyToID allows threading; zero means the comment starts a thread.
type ReviewComment struct {
	ID        int64     `json:"id"`
	Author    string    `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
	Path      string    `json:"path"`
	Line      int       `json:"line"`
	ReplyToID int64     `json:"replyToId,omitempty"`
	URL       string    `json:"url,omitempty"`
}

// CheckRun summarises one CI check at the PR's head commit. The
// forge-native statuses collapse to five buckets so the UI can render
// a single traffic-light badge:
//
//	"success" — all assertions green
//	"failure" — at least one failed assertion
//	"error"   — CI system itself errored (config / infrastructure)
//	"pending" — still running or queued
//	"skipped" — check was skipped (often branch-filter)
//
// Context is the CI system + pipeline name ("ci/jenkins",
// "GitHub Actions / build", etc) so the user can distinguish multiple
// checks at a glance. TargetURL goes to the check's UI when present.
type CheckRun struct {
	Name        string     `json:"name"`
	Status      string     `json:"status"`
	Context     string     `json:"context,omitempty"`
	TargetURL   string     `json:"targetUrl,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

// ─── Adapter interface + dispatch ────────────────────────────────

// Adapter is the internal contract each forge implementation fills.
// Not exported — callers use the package-level List/Detail/Diff/
// Comments functions, which dispatch based on cfg.ForgeType.
type adapter interface {
	list(ctx context.Context, state State, limit int) ([]PullRequest, error)
	detail(ctx context.Context, number int) (PullRequest, error)
	diff(ctx context.Context, number int) ([]DiffFile, error)
	comments(ctx context.Context, number int) ([]Comment, error)
	reviews(ctx context.Context, number int) ([]Review, error)
	reviewComments(ctx context.Context, number int) ([]ReviewComment, error)
	// checks takes headSHA — adapter may hit a /statuses or
	// /check-runs endpoint that needs the commit hash directly
	// instead of the PR number.
	checks(ctx context.Context, number int, headSHA string) ([]CheckRun, error)
}

// pick returns the adapter matching cfg.ForgeType, or an error if
// the type is missing / unknown.
func pick(cfg Config) (adapter, error) {
	if cfg.BaseURL == "" {
		return nil, errors.New("forge: baseUrl is required")
	}
	if cfg.Repo == "" {
		return nil, errors.New("forge: repo is required")
	}
	switch strings.ToLower(cfg.ForgeType) {
	case "gitea":
		return &giteaAdapter{cfg: cfg, http: httpClient(cfg.Timeout)}, nil
	case "github":
		return &githubAdapter{cfg: cfg, http: httpClient(cfg.Timeout)}, nil
	case "gitlab":
		return &gitlabAdapter{cfg: cfg, http: httpClient(cfg.Timeout)}, nil
	case "":
		return nil, errors.New("forge: forgeType is required (one of: gitea, github, gitlab)")
	default:
		return nil, fmt.Errorf("forge: unknown forgeType %q", cfg.ForgeType)
	}
}

// List returns PRs matching the state filter. limit is clamped to
// [1, 100] — forges disagree on server-side limits and callers don't
// need more than a page for the panel list view.
func List(ctx context.Context, cfg Config, state State, limit int) ([]PullRequest, error) {
	if state == "" {
		state = StateOpen
	}
	if limit <= 0 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}
	a, err := pick(cfg)
	if err != nil {
		return nil, err
	}
	prs, err := a.list(ctx, state, limit)
	if err != nil {
		return nil, err
	}
	// Never return nil — the handler surfaces a JSON array and the
	// client prefers [] over null.
	if prs == nil {
		prs = []PullRequest{}
	}
	return prs, nil
}

// Detail returns one PR by number.
func Detail(ctx context.Context, cfg Config, number int) (PullRequest, error) {
	if number <= 0 {
		return PullRequest{}, errors.New("forge: pull number must be positive")
	}
	a, err := pick(cfg)
	if err != nil {
		return PullRequest{}, err
	}
	return a.detail(ctx, number)
}

// Diff returns the file-level diff for one PR.
func Diff(ctx context.Context, cfg Config, number int) ([]DiffFile, error) {
	if number <= 0 {
		return nil, errors.New("forge: pull number must be positive")
	}
	a, err := pick(cfg)
	if err != nil {
		return nil, err
	}
	files, err := a.diff(ctx, number)
	if err != nil {
		return nil, err
	}
	if files == nil {
		files = []DiffFile{}
	}
	return files, nil
}

// Comments returns top-level PR comments.
func Comments(ctx context.Context, cfg Config, number int) ([]Comment, error) {
	if number <= 0 {
		return nil, errors.New("forge: pull number must be positive")
	}
	a, err := pick(cfg)
	if err != nil {
		return nil, err
	}
	cs, err := a.comments(ctx, number)
	if err != nil {
		return nil, err
	}
	if cs == nil {
		cs = []Comment{}
	}
	return cs, nil
}

// Reviews returns one row per reviewer verdict (approved / changes
// requested / commented / dismissed). GitLab's approvals API lacks a
// body field — those rows come back with State="approved" and Body="".
func Reviews(ctx context.Context, cfg Config, number int) ([]Review, error) {
	if number <= 0 {
		return nil, errors.New("forge: pull number must be positive")
	}
	a, err := pick(cfg)
	if err != nil {
		return nil, err
	}
	rs, err := a.reviews(ctx, number)
	if err != nil {
		return nil, err
	}
	if rs == nil {
		rs = []Review{}
	}
	return rs, nil
}

// ReviewComments returns inline (per-file, per-line) review comments.
// Distinct from [Comments] which returns only the PR's top-level
// discussion thread.
func ReviewComments(ctx context.Context, cfg Config, number int) ([]ReviewComment, error) {
	if number <= 0 {
		return nil, errors.New("forge: pull number must be positive")
	}
	a, err := pick(cfg)
	if err != nil {
		return nil, err
	}
	rcs, err := a.reviewComments(ctx, number)
	if err != nil {
		return nil, err
	}
	if rcs == nil {
		rcs = []ReviewComment{}
	}
	return rcs, nil
}

// Checks returns CI run statuses for the PR's head commit. headSHA is
// looked up via Detail() internally when empty, so callers usually just
// pass the PR number.
func Checks(ctx context.Context, cfg Config, number int, headSHA string) ([]CheckRun, error) {
	if number <= 0 {
		return nil, errors.New("forge: pull number must be positive")
	}
	a, err := pick(cfg)
	if err != nil {
		return nil, err
	}
	// Resolve the head SHA if the caller didn't pre-fetch it. Saves
	// an HTTP round-trip on the common "just tapped into PR detail"
	// flow — callers that already hold the SHA pass it through.
	if headSHA == "" {
		pr, err := a.detail(ctx, number)
		if err != nil {
			return nil, fmt.Errorf("forge: resolve head sha: %w", err)
		}
		headSHA = pr.HeadSHA
	}
	if headSHA == "" {
		// Some forges (e.g. GitLab without diff_refs) don't surface
		// a head SHA; return an empty list rather than erroring so
		// the client can display "no checks reported" cleanly.
		return []CheckRun{}, nil
	}
	crs, err := a.checks(ctx, number, headSHA)
	if err != nil {
		return nil, err
	}
	if crs == nil {
		crs = []CheckRun{}
	}
	return crs, nil
}

// ─── HTTP plumbing shared by all adapters ────────────────────────

// httpClient returns a client with the caller-supplied timeout. A
// nil/zero timeout defaults to 20s so we never hang the gateway on
// a silently-unreachable forge.
func httpClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	return &http.Client{Timeout: timeout}
}

// doJSON runs req and decodes a JSON response into out. Non-2xx
// responses are turned into an error carrying the status code and
// the first 512 bytes of the body — enough for log triage without
// letting a hostile server flood gateway memory.
func doJSON(client *http.Client, req *http.Request, out any) error {
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("forge: http request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("forge: %s %s -> %d: %s",
			req.Method, req.URL.Path, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(out); err != nil {
		return fmt.Errorf("forge: decode response: %w", err)
	}
	return nil
}

// doText runs req and returns the raw body as a string. Used by the
// GitHub / GitLab adapters to pull the `.diff` / `.patch` endpoints.
func doText(client *http.Client, req *http.Request) (string, error) {
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("forge: http request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("forge: %s %s -> %d: %s",
			req.Method, req.URL.Path, resp.StatusCode, strings.TrimSpace(string(snippet)))
	}
	// Cap at 8 MB — a single PR's diff should comfortably fit. An
	// attacker-controlled forge can't OOM the gateway.
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return "", fmt.Errorf("forge: read body: %w", err)
	}
	return string(body), nil
}

// newRequest is a tiny wrapper to attach auth + User-Agent uniformly.
func newRequest(ctx context.Context, method, url, token, accept string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}
	if accept == "" {
		accept = "application/json"
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("User-Agent", "opendray-git-forge/1.0")
	if token != "" {
		// Gitea accepts "token <...>", GitHub accepts "token" or
		// "Bearer", GitLab expects "Bearer" (PAT) or its own header.
		// Each adapter overrides this in its own newRequest wrapper
		// when the default shape isn't right.
		req.Header.Set("Authorization", "token "+token)
	}
	return req, nil
}

// splitRepo splits cfg.Repo into (owner, name). Empty owner or name
// is an error — callers should pass "owner/name" verbatim.
func splitRepo(repo string) (string, string, error) {
	r := strings.TrimPrefix(strings.TrimSpace(repo), "/")
	r = strings.TrimSuffix(r, "/")
	parts := strings.SplitN(r, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("forge: repo must be owner/name, got %q", repo)
	}
	return parts[0], parts[1], nil
}

// trimSlash removes trailing slashes from the base URL so adapters
// can concat `/path` without doubling up.
func trimSlash(s string) string {
	return strings.TrimRight(s, "/")
}

// normaliseReviewState folds each forge's review vocabulary into the
// four buckets [Review.State] documents. Unknown strings fall through
// to "commented" so a forthcoming forge schema change doesn't crash
// the merge panel.
func normaliseReviewState(raw string) string {
	switch strings.ToUpper(raw) {
	case "APPROVED", "APPROVE", "APPROVAL":
		return "approved"
	case "CHANGES_REQUESTED", "REQUEST_CHANGES", "REJECTED":
		return "changes_requested"
	case "DISMISSED":
		return "dismissed"
	case "COMMENTED", "COMMENT", "":
		return "commented"
	default:
		return "commented"
	}
}

// normaliseCheckStatus collapses every forge's CI status enum into
// the five buckets [CheckRun.Status] documents. Behaviour for
// unrecognised strings is "pending" — conservative; running checks
// ought not be classified as success/failure without evidence.
func normaliseCheckStatus(raw string) string {
	switch strings.ToLower(raw) {
	case "success", "passed", "completed":
		return "success"
	case "failure", "failed", "error":
		return "failure"
	case "skipped", "neutral", "cancelled", "canceled":
		return "skipped"
	case "pending", "running", "queued", "in_progress", "":
		return "pending"
	case "warning":
		// Gitea reports CI style-lint etc. as "warning" — closer to
		// a soft failure than a pass.
		return "failure"
	default:
		return "pending"
	}
}

// firstNonEmpty returns the first argument that's a non-empty string,
// or "" if all are empty. Useful for picking a display name from
// forge fields that are often redundant (description vs context).
func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

package forge

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// githubAdapter talks to the GitHub v3 REST API. The adapter accepts
// both "https://api.github.com" and "https://github.com" as BaseURL
// — the latter is coerced to the API host so users who paste the
// web URL get the expected behaviour.
//
//   list / detail: /repos/{owner}/{name}/pulls
//   diff:          /repos/{owner}/{name}/pulls/{n} with
//                  Accept: application/vnd.github.v3.diff
//   comments:      /repos/{owner}/{name}/issues/{n}/comments
type githubAdapter struct {
	cfg  Config
	http *http.Client
}

func (a *githubAdapter) apiBase() string {
	base := trimSlash(a.cfg.BaseURL)
	// Common mispaste: "https://github.com" (web UI) → correct to
	// the API host. Enterprise installs use
	// "https://ghe.example.com/api/v3" which the user is expected
	// to type verbatim; we only rewrite the public host.
	if strings.EqualFold(base, "https://github.com") {
		return "https://api.github.com"
	}
	return base
}

func (a *githubAdapter) apiURL(format string, args ...any) string {
	return fmt.Sprintf("%s"+format,
		append([]any{a.apiBase()}, args...)...)
}

// githubAPIRequest wraps [newRequest] with GitHub-recommended headers
// (Accept + X-GitHub-Api-Version) so the adapter pins the response
// schema across GHE / github.com upgrades.
func (a *githubAdapter) newReq(ctx context.Context, method, url, accept string) (*http.Request, error) {
	if accept == "" {
		accept = "application/vnd.github+json"
	}
	req, err := newRequest(ctx, method, url, a.cfg.Token, accept)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	return req, nil
}

type githubPR struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	State     string    `json:"state"`           // "open" | "closed"
	MergedAt  *time.Time `json:"merged_at"`
	HTMLURL   string    `json:"html_url"`
	Body      string    `json:"body"`
	Comments  int       `json:"comments"`
	Draft     bool      `json:"draft"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	User      struct {
		Login     string `json:"login"`
		AvatarURL string `json:"avatar_url"`
	} `json:"user"`
	Head struct {
		Ref string `json:"ref"`
		SHA string `json:"sha"`
	} `json:"head"`
	Base struct {
		Ref string `json:"ref"`
	} `json:"base"`
}

func (a *githubAdapter) translate(p githubPR) PullRequest {
	state := p.State
	if p.MergedAt != nil {
		state = "merged"
	}
	return PullRequest{
		Number:       p.Number,
		Title:        p.Title,
		State:        state,
		Author:       p.User.Login,
		AuthorAvatar: p.User.AvatarURL,
		HeadRef:      p.Head.Ref,
		HeadSHA:      p.Head.SHA,
		BaseRef:      p.Base.Ref,
		URL:          p.HTMLURL,
		Body:         p.Body,
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
		CommentCount: p.Comments,
		Draft:        p.Draft,
	}
}

func (a *githubAdapter) list(ctx context.Context, state State, limit int) ([]PullRequest, error) {
	owner, name, err := splitRepo(a.cfg.Repo)
	if err != nil {
		return nil, err
	}
	// GitHub's state values match ours except "all" — which it also
	// calls "all", so direct passthrough. per_page caps at 100.
	url := a.apiURL("/repos/%s/%s/pulls?state=%s&per_page=%d", owner, name, string(state), limit)
	req, err := a.newReq(ctx, http.MethodGet, url, "")
	if err != nil {
		return nil, err
	}
	var raw []githubPR
	if err := doJSON(a.http, req, &raw); err != nil {
		return nil, err
	}
	out := make([]PullRequest, 0, len(raw))
	for _, p := range raw {
		out = append(out, a.translate(p))
	}
	return out, nil
}

func (a *githubAdapter) detail(ctx context.Context, number int) (PullRequest, error) {
	owner, name, err := splitRepo(a.cfg.Repo)
	if err != nil {
		return PullRequest{}, err
	}
	url := a.apiURL("/repos/%s/%s/pulls/%d", owner, name, number)
	req, err := a.newReq(ctx, http.MethodGet, url, "")
	if err != nil {
		return PullRequest{}, err
	}
	var raw githubPR
	if err := doJSON(a.http, req, &raw); err != nil {
		return PullRequest{}, err
	}
	return a.translate(raw), nil
}

func (a *githubAdapter) diff(ctx context.Context, number int) ([]DiffFile, error) {
	owner, name, err := splitRepo(a.cfg.Repo)
	if err != nil {
		return nil, err
	}
	url := a.apiURL("/repos/%s/%s/pulls/%d", owner, name, number)
	// Accept: v3.diff switches the response from JSON to raw unified
	// diff — the simplest cross-forge representation.
	req, err := a.newReq(ctx, http.MethodGet, url, "application/vnd.github.v3.diff")
	if err != nil {
		return nil, err
	}
	body, err := doText(a.http, req)
	if err != nil {
		return nil, err
	}
	return parseUnifiedDiff(body), nil
}

type githubComment struct {
	ID        int64     `json:"id"`
	HTMLURL   string    `json:"html_url"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	User      struct {
		Login string `json:"login"`
	} `json:"user"`
}

// ── Reviews / review-comments / checks (Tier A) ─────────────────

type githubReview struct {
	ID          int64     `json:"id"`
	State       string    `json:"state"` // APPROVED | CHANGES_REQUESTED | COMMENTED | DISMISSED | PENDING
	Body        string    `json:"body"`
	SubmittedAt time.Time `json:"submitted_at"`
	User        struct {
		Login     string `json:"login"`
		AvatarURL string `json:"avatar_url"`
	} `json:"user"`
}

func (a *githubAdapter) reviews(ctx context.Context, number int) ([]Review, error) {
	owner, name, err := splitRepo(a.cfg.Repo)
	if err != nil {
		return nil, err
	}
	url := a.apiURL("/repos/%s/%s/pulls/%d/reviews", owner, name, number)
	req, err := a.newReq(ctx, http.MethodGet, url, "")
	if err != nil {
		return nil, err
	}
	var raw []githubReview
	if err := doJSON(a.http, req, &raw); err != nil {
		return nil, err
	}
	out := make([]Review, 0, len(raw))
	for _, r := range raw {
		if r.State == "PENDING" {
			continue
		}
		out = append(out, Review{
			Author:       r.User.Login,
			AuthorAvatar: r.User.AvatarURL,
			State:        normaliseReviewState(r.State),
			Body:         r.Body,
			SubmittedAt:  r.SubmittedAt,
		})
	}
	return out, nil
}

// githubReviewComment mirrors GitHub's inline review-comment shape.
// Unlike Gitea's per-review comments, GitHub exposes a flat endpoint
// returning every review comment on the PR.
type githubReviewComment struct {
	ID                  int64     `json:"id"`
	Body                string    `json:"body"`
	CreatedAt           time.Time `json:"created_at"`
	Path                string    `json:"path"`
	Line                int       `json:"line"`
	OriginalLine        int       `json:"original_line"`
	InReplyToID         int64     `json:"in_reply_to_id"`
	HTMLURL             string    `json:"html_url"`
	User                struct {
		Login string `json:"login"`
	} `json:"user"`
}

func (a *githubAdapter) reviewComments(ctx context.Context, number int) ([]ReviewComment, error) {
	owner, name, err := splitRepo(a.cfg.Repo)
	if err != nil {
		return nil, err
	}
	// /pulls/{n}/comments is the review-comment stream; /issues/{n}
	// /comments is top-level discussion (Comments()). GitHub keeps
	// them separate so plugins that want one without the other don't
	// have to filter.
	url := a.apiURL("/repos/%s/%s/pulls/%d/comments?per_page=100", owner, name, number)
	req, err := a.newReq(ctx, http.MethodGet, url, "")
	if err != nil {
		return nil, err
	}
	var raw []githubReviewComment
	if err := doJSON(a.http, req, &raw); err != nil {
		return nil, err
	}
	out := make([]ReviewComment, 0, len(raw))
	for _, c := range raw {
		line := c.Line
		if line == 0 {
			line = c.OriginalLine
		}
		out = append(out, ReviewComment{
			ID: c.ID, Author: c.User.Login, Body: c.Body,
			CreatedAt: c.CreatedAt, Path: c.Path, Line: line,
			ReplyToID: c.InReplyToID, URL: c.HTMLURL,
		})
	}
	return out, nil
}

// githubCheckRun mirrors one entry from /commits/{sha}/check-runs.
// Newer than the /statuses endpoint Gitea exposes — GitHub Apps
// publish check-runs, older CI integrations publish plain statuses.
// We query check-runs first; statuses can be added as a fallback
// later if real-world repos leak past.
type githubCheckRun struct {
	Name        string     `json:"name"`
	Status      string     `json:"status"`     // queued | in_progress | completed
	Conclusion  string     `json:"conclusion"` // success | failure | neutral | cancelled | timed_out | action_required | stale | skipped
	HTMLURL     string     `json:"html_url"`
	CompletedAt *time.Time `json:"completed_at"`
	App         struct {
		Name string `json:"name"`
	} `json:"app"`
}

type githubCheckRunsResp struct {
	CheckRuns []githubCheckRun `json:"check_runs"`
}

func (a *githubAdapter) checks(ctx context.Context, _ int, headSHA string) ([]CheckRun, error) {
	owner, name, err := splitRepo(a.cfg.Repo)
	if err != nil {
		return nil, err
	}
	url := a.apiURL("/repos/%s/%s/commits/%s/check-runs?per_page=100", owner, name, headSHA)
	req, err := a.newReq(ctx, http.MethodGet, url, "")
	if err != nil {
		return nil, err
	}
	var raw githubCheckRunsResp
	if err := doJSON(a.http, req, &raw); err != nil {
		return nil, err
	}
	out := make([]CheckRun, 0, len(raw.CheckRuns))
	for _, c := range raw.CheckRuns {
		// check-runs expose two fields (`status` + `conclusion`);
		// collapse to our single enum. When the run isn't complete
		// yet the conclusion is empty and the status wins.
		status := c.Conclusion
		if status == "" {
			status = c.Status
		}
		out = append(out, CheckRun{
			Name:        c.Name,
			Context:     c.App.Name,
			Status:      normaliseCheckStatus(status),
			TargetURL:   c.HTMLURL,
			CompletedAt: c.CompletedAt,
		})
	}
	return out, nil
}

func (a *githubAdapter) comments(ctx context.Context, number int) ([]Comment, error) {
	owner, name, err := splitRepo(a.cfg.Repo)
	if err != nil {
		return nil, err
	}
	// PR top-level comments live on the issue API (GitHub treats
	// every PR as an issue for discussion purposes). Inline review
	// comments are on /pulls/{n}/comments — deferred to Phase 2.2.
	url := a.apiURL("/repos/%s/%s/issues/%d/comments", owner, name, number)
	req, err := a.newReq(ctx, http.MethodGet, url, "")
	if err != nil {
		return nil, err
	}
	var raw []githubComment
	if err := doJSON(a.http, req, &raw); err != nil {
		return nil, err
	}
	out := make([]Comment, 0, len(raw))
	for _, c := range raw {
		out = append(out, Comment{
			ID: c.ID, Author: c.User.Login, Body: c.Body,
			CreatedAt: c.CreatedAt, URL: c.HTMLURL,
		})
	}
	return out, nil
}

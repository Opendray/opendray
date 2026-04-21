package forge

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// giteaAdapter talks to Gitea v1 API.
//   list / detail: /repos/{owner}/{name}/pulls
//   diff:          /repos/{owner}/{name}/pulls/{n}.diff (raw unified text)
//   comments:      /repos/{owner}/{name}/issues/{n}/comments (PRs share
//                  the issue comment stream on Gitea; review threads
//                  live under /reviews which we defer to Phase 2.2).
type giteaAdapter struct {
	cfg  Config
	http *http.Client
}

func (a *giteaAdapter) apiURL(format string, args ...any) string {
	return fmt.Sprintf("%s/api/v1"+format,
		append([]any{trimSlash(a.cfg.BaseURL)}, args...)...)
}

// giteaPR mirrors the subset of Gitea's PullRequest JSON the adapter
// reads. Fields left out are intentional — keep the decode shape
// tight so forge upgrades can't silently change meaning on us.
type giteaPR struct {
	Number    int    `json:"number"`
	Title     string `json:"title"`
	State     string `json:"state"`     // "open" | "closed"
	Merged    bool   `json:"merged"`
	HTMLURL   string `json:"html_url"`
	Body      string `json:"body"`
	Comments  int    `json:"comments"`
	Draft     bool   `json:"draft"`
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

func (a *giteaAdapter) translate(p giteaPR) PullRequest {
	state := p.State
	if p.Merged {
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

func (a *giteaAdapter) list(ctx context.Context, state State, limit int) ([]PullRequest, error) {
	owner, name, err := splitRepo(a.cfg.Repo)
	if err != nil {
		return nil, err
	}
	// Gitea uses "state=open|closed|all" matching our State enum
	// directly. limit maps to Gitea's `limit=` (page size); we don't
	// paginate beyond the first page here.
	url := a.apiURL("/repos/%s/%s/pulls?state=%s&limit=%d", owner, name, string(state), limit)
	req, err := newRequest(ctx, http.MethodGet, url, a.cfg.Token, "")
	if err != nil {
		return nil, err
	}
	var raw []giteaPR
	if err := doJSON(a.http, req, &raw); err != nil {
		return nil, err
	}
	out := make([]PullRequest, 0, len(raw))
	for _, p := range raw {
		out = append(out, a.translate(p))
	}
	return out, nil
}

func (a *giteaAdapter) detail(ctx context.Context, number int) (PullRequest, error) {
	owner, name, err := splitRepo(a.cfg.Repo)
	if err != nil {
		return PullRequest{}, err
	}
	url := a.apiURL("/repos/%s/%s/pulls/%d", owner, name, number)
	req, err := newRequest(ctx, http.MethodGet, url, a.cfg.Token, "")
	if err != nil {
		return PullRequest{}, err
	}
	var raw giteaPR
	if err := doJSON(a.http, req, &raw); err != nil {
		return PullRequest{}, err
	}
	return a.translate(raw), nil
}

// diff pulls the raw unified diff and splits it into per-file
// [DiffFile] entries via [parseUnifiedDiff]. Gitea does expose a
// structured `/files` endpoint too, but the raw-diff route is
// available on every Gitea version we care about and avoids a
// double round-trip for additions/deletions (we derive those from
// the patch instead).
func (a *giteaAdapter) diff(ctx context.Context, number int) ([]DiffFile, error) {
	owner, name, err := splitRepo(a.cfg.Repo)
	if err != nil {
		return nil, err
	}
	url := a.apiURL("/repos/%s/%s/pulls/%d.diff", owner, name, number)
	req, err := newRequest(ctx, http.MethodGet, url, a.cfg.Token, "text/plain")
	if err != nil {
		return nil, err
	}
	body, err := doText(a.http, req)
	if err != nil {
		return nil, err
	}
	return parseUnifiedDiff(body), nil
}

// giteaComment mirrors the issue-comment JSON (Gitea stores PR
// discussion comments on the issue's timeline).
type giteaComment struct {
	ID        int64     `json:"id"`
	HTMLURL   string    `json:"html_url"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	User      struct {
		Login string `json:"login"`
	} `json:"user"`
}

// ── Reviews / review-comments / checks (Tier A) ─────────────────

// giteaReview mirrors the review JSON. Gitea's state vocab matches
// what we want with one exception: "APPROVED" vs "CHANGES_REQUESTED"
// come in uppercase, which we lowercase before handing to the UI.
type giteaReview struct {
	ID           int64     `json:"id"`
	State        string    `json:"state"` // APPROVED | COMMENT | REQUEST_CHANGES | DISMISSED | PENDING
	Body         string    `json:"body"`
	SubmittedAt  time.Time `json:"submitted_at"`
	CommentsCount int      `json:"comments_count"`
	User         struct {
		Login     string `json:"login"`
		AvatarURL string `json:"avatar_url"`
	} `json:"user"`
}

func (a *giteaAdapter) reviews(ctx context.Context, number int) ([]Review, error) {
	owner, name, err := splitRepo(a.cfg.Repo)
	if err != nil {
		return nil, err
	}
	url := a.apiURL("/repos/%s/%s/pulls/%d/reviews", owner, name, number)
	req, err := newRequest(ctx, http.MethodGet, url, a.cfg.Token, "")
	if err != nil {
		return nil, err
	}
	var raw []giteaReview
	if err := doJSON(a.http, req, &raw); err != nil {
		return nil, err
	}
	out := make([]Review, 0, len(raw))
	for _, r := range raw {
		// Skip PENDING reviews — the reviewer drafted something but
		// hasn't submitted yet. Including them would double-count
		// the authoring user in "who reviewed this" summaries.
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

// giteaReviewComment mirrors one comment inside a review's
// /reviews/{id}/comments endpoint. Gitea also exposes a flat
// /issues/{n}/comments stream for non-review discussion — that's
// what the plain Comments() method returns.
type giteaReviewComment struct {
	ID        int64     `json:"id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	Path      string    `json:"path"`
	Line      int       `json:"line"`
	OldLine   int       `json:"old_line"`
	HTMLURL   string    `json:"html_url"`
	User      struct {
		Login string `json:"login"`
	} `json:"user"`
}

func (a *giteaAdapter) reviewComments(ctx context.Context, number int) ([]ReviewComment, error) {
	owner, name, err := splitRepo(a.cfg.Repo)
	if err != nil {
		return nil, err
	}
	// Gitea exposes review comments one-reviews-at-a-time. We walk
	// all reviews on this PR and flatten. For a PR with dozens of
	// reviews this is N round-trips — acceptable for now; a /files
	// endpoint with all comments could replace this but isn't
	// documented consistently across Gitea versions.
	reviewsURL := a.apiURL("/repos/%s/%s/pulls/%d/reviews", owner, name, number)
	req, err := newRequest(ctx, http.MethodGet, reviewsURL, a.cfg.Token, "")
	if err != nil {
		return nil, err
	}
	var reviewList []giteaReview
	if err := doJSON(a.http, req, &reviewList); err != nil {
		return nil, err
	}
	var out []ReviewComment
	for _, r := range reviewList {
		if r.CommentsCount == 0 {
			continue
		}
		cURL := a.apiURL("/repos/%s/%s/pulls/%d/reviews/%d/comments",
			owner, name, number, r.ID)
		cReq, err := newRequest(ctx, http.MethodGet, cURL, a.cfg.Token, "")
		if err != nil {
			return nil, err
		}
		var raw []giteaReviewComment
		if err := doJSON(a.http, cReq, &raw); err != nil {
			return nil, err
		}
		for _, c := range raw {
			line := c.Line
			if line == 0 {
				line = c.OldLine
			}
			out = append(out, ReviewComment{
				ID: c.ID, Author: c.User.Login, Body: c.Body,
				CreatedAt: c.CreatedAt, Path: c.Path, Line: line,
				URL: c.HTMLURL,
			})
		}
	}
	return out, nil
}

// giteaStatus mirrors /commits/{sha}/statuses. Gitea has both the
// newer "commit statuses" (v1 API) and "combined status" endpoints;
// /statuses is the portable one.
type giteaStatus struct {
	ID          int64     `json:"id"`
	Status      string    `json:"status"` // pending | success | failure | error | warning
	TargetURL   string    `json:"target_url"`
	Description string    `json:"description"`
	Context     string    `json:"context"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (a *giteaAdapter) checks(ctx context.Context, _ int, headSHA string) ([]CheckRun, error) {
	owner, name, err := splitRepo(a.cfg.Repo)
	if err != nil {
		return nil, err
	}
	url := a.apiURL("/repos/%s/%s/commits/%s/statuses", owner, name, headSHA)
	req, err := newRequest(ctx, http.MethodGet, url, a.cfg.Token, "")
	if err != nil {
		return nil, err
	}
	var raw []giteaStatus
	if err := doJSON(a.http, req, &raw); err != nil {
		return nil, err
	}
	out := make([]CheckRun, 0, len(raw))
	for _, s := range raw {
		completed := s.UpdatedAt
		out = append(out, CheckRun{
			Name:        firstNonEmpty(s.Description, s.Context),
			Context:     s.Context,
			Status:      normaliseCheckStatus(s.Status),
			TargetURL:   s.TargetURL,
			CompletedAt: &completed,
		})
	}
	return out, nil
}

func (a *giteaAdapter) comments(ctx context.Context, number int) ([]Comment, error) {
	owner, name, err := splitRepo(a.cfg.Repo)
	if err != nil {
		return nil, err
	}
	url := a.apiURL("/repos/%s/%s/issues/%d/comments", owner, name, number)
	req, err := newRequest(ctx, http.MethodGet, url, a.cfg.Token, "")
	if err != nil {
		return nil, err
	}
	var raw []giteaComment
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

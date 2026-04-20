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

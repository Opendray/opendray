package forge

// Per-adapter listRepos implementations. Each forge has its own
// "show me all repos I can see" endpoint with a slightly different
// shape — this file keeps the translations close to the interface
// contract (RepoSummary) without touching the PR-flow adapter code.

import (
	"context"
	"fmt"
	"net/http"
)

// ── Gitea ───────────────────────────────────────────────────────

// giteaRepo mirrors the fields RepoSummary cares about. Gitea's /repos/
// search payload is huge; keep the decode narrow so upstream additions
// can't surprise us.
type giteaRepo struct {
	FullName      string `json:"full_name"`
	Description   string `json:"description"`
	Private       bool   `json:"private"`
	DefaultBranch string `json:"default_branch"`
	HTMLURL       string `json:"html_url"`
}

// giteaRepoSearchResp is Gitea's /repos/search envelope.
type giteaRepoSearchResp struct {
	Data []giteaRepo `json:"data"`
	OK   bool        `json:"ok"`
}

func (a *giteaAdapter) listRepos(ctx context.Context, limit int) ([]RepoSummary, error) {
	// /repos/search pages at 50 by default; clamp to limit.
	url := a.apiURL("/repos/search?limit=%d", limit)
	req, err := newRequest(ctx, http.MethodGet, url, a.cfg.Token, "")
	if err != nil {
		return nil, err
	}
	var env giteaRepoSearchResp
	if err := doJSON(a.http, req, &env); err != nil {
		return nil, err
	}
	out := make([]RepoSummary, 0, len(env.Data))
	for _, r := range env.Data {
		out = append(out, RepoSummary{
			FullName:      r.FullName,
			Description:   r.Description,
			Private:       r.Private,
			DefaultBranch: r.DefaultBranch,
			HTMLURL:       r.HTMLURL,
		})
	}
	return out, nil
}

// ── GitHub ──────────────────────────────────────────────────────

type githubRepo struct {
	FullName      string `json:"full_name"`
	Description   string `json:"description"`
	Private       bool   `json:"private"`
	DefaultBranch string `json:"default_branch"`
	HTMLURL       string `json:"html_url"`
}

func (a *githubAdapter) listRepos(ctx context.Context, limit int) ([]RepoSummary, error) {
	// /user/repos returns what the token can see (needs `repo` scope
	// for privates). Empty token → public-only endpoint is useless
	// here; surface a clear error rather than a silent empty list.
	if a.cfg.Token == "" {
		return nil, fmt.Errorf("github: token required to list accessible repos")
	}
	perPage := limit
	if perPage > 100 {
		perPage = 100
	}
	url := fmt.Sprintf("%s/user/repos?per_page=%d&sort=updated",
		trimSlash(a.apiBase()), perPage)
	req, err := newRequest(ctx, http.MethodGet, url, a.cfg.Token, "application/vnd.github+json")
	if err != nil {
		return nil, err
	}
	var raw []githubRepo
	if err := doJSON(a.http, req, &raw); err != nil {
		return nil, err
	}
	out := make([]RepoSummary, 0, len(raw))
	for _, r := range raw {
		out = append(out, RepoSummary{
			FullName:      r.FullName,
			Description:   r.Description,
			Private:       r.Private,
			DefaultBranch: r.DefaultBranch,
			HTMLURL:       r.HTMLURL,
		})
	}
	return out, nil
}

// ── GitLab ──────────────────────────────────────────────────────

// gitlabProject is the subset of /projects response we read. GitLab's
// "full name" is path_with_namespace (e.g. "group/subgroup/project")
// — callers can pass that straight back to PR routes because GitLab's
// adapter already URL-encodes the full path.
type gitlabProject struct {
	PathWithNamespace string `json:"path_with_namespace"`
	Description       string `json:"description"`
	Visibility        string `json:"visibility"` // "private" | "internal" | "public"
	DefaultBranch     string `json:"default_branch"`
	WebURL            string `json:"web_url"`
}

func (a *gitlabAdapter) listRepos(ctx context.Context, limit int) ([]RepoSummary, error) {
	perPage := limit
	if perPage > 100 {
		perPage = 100
	}
	url := a.apiURL("/projects?membership=true&simple=true&per_page=%d", perPage)
	req, err := newRequest(ctx, http.MethodGet, url, a.cfg.Token, "")
	if err != nil {
		return nil, err
	}
	var raw []gitlabProject
	if err := doJSON(a.http, req, &raw); err != nil {
		return nil, err
	}
	out := make([]RepoSummary, 0, len(raw))
	for _, p := range raw {
		out = append(out, RepoSummary{
			FullName:      p.PathWithNamespace,
			Description:   p.Description,
			Private:       p.Visibility != "public",
			DefaultBranch: p.DefaultBranch,
			HTMLURL:       p.WebURL,
		})
	}
	return out, nil
}

package githost

// Issue viewing — a read-only sibling of the pull-request surface in
// githost.go. Issues mirror PRs (list + detail + comments across
// GitHub / Gitea / GitLab) but drop the PR-only concepts: no commits,
// checks, files, or merge. The conversation reuses PRComment.
//
// Two cross-provider quirks shape this file:
//
//  1. GitHub and Gitea expose issues and PRs through the SAME /issues
//     endpoint; PR items carry a `pull_request` field. We skip those so
//     a repo's PRs don't double-show as issues. GitLab keeps issues on a
//     separate endpoint, so no filtering there.
//  2. Labels differ: GitHub/Gitea return [{name,color}] objects; GitLab
//     returns a flat []string. Both normalise to []Label (GitLab colour
//     is left empty).

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Label is one issue label. Color is a hex string without the leading
// '#'; it may be empty (GitLab labels carry only a name in the issue
// payload).
type Label struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// Issue is the trimmed-down view surfaced in the panel. Body is the
// description (markdown); like PullRequest.Body it is only populated by
// the single-issue detail fetch (GetIssue), so the list endpoint keeps
// its payload lean. Labels has no omitempty so the JSON always carries
// an array rather than null.
type Issue struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	State     string    `json:"state"` // open | closed
	Author    string    `json:"author"`
	URL       string    `json:"url"`
	Body      string    `json:"body,omitempty"`
	Labels    []Label   `json:"labels"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ── label normalisation ───────────────────────────────────────────

// githubStyleLabels maps GitHub/Gitea {name,color} label objects to
// []Label. Returns a non-nil empty slice so JSON emits [].
func githubStyleLabels(raw []struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}) []Label {
	out := make([]Label, 0, len(raw))
	for _, l := range raw {
		out = append(out, Label{Name: l.Name, Color: l.Color})
	}
	return out
}

// gitlabStringLabels maps GitLab's flat []string labels to []Label with
// empty Color. Returns a non-nil empty slice so JSON emits [].
func gitlabStringLabels(raw []string) []Label {
	out := make([]Label, 0, len(raw))
	for _, name := range raw {
		out = append(out, Label{Name: name})
	}
	return out
}

// ── service dispatch ───────────────────────────────────────────────

// ListIssues lists issues for dir's remote in the given state
// (open|closed|all, default open). PRs are filtered out for GitHub /
// Gitea; see the per-provider adapters.
func (s *Service) ListIssues(ctx context.Context, dir, state string) (Remote, []Issue, error) {
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
		body, ferr := s.fetch(ctx, fmt.Sprintf("%s/repos/%s/%s/issues?state=%s&per_page=20",
			githubAPIBase(hostRow.Host), rem.Owner, rem.Repo, url.QueryEscape(state)),
			"Bearer "+hostRow.Token, "application/vnd.github+json")
		if ferr != nil {
			return rem, nil, ferr
		}
		issues, derr := decodeGitHubIssues(body)
		return rem, issues, derr
	case KindGitea:
		body, ferr := s.fetch(ctx, fmt.Sprintf("https://%s/api/v1/repos/%s/%s/issues?state=%s&type=issues&limit=20",
			hostRow.Host, rem.Owner, rem.Repo, url.QueryEscape(state)),
			"token "+hostRow.Token, "application/json")
		if ferr != nil {
			return rem, nil, ferr
		}
		issues, derr := decodeGitHubIssues(body)
		return rem, issues, derr
	case KindGitLab:
		// GitLab's issues API accepts only opened|closed for the state
		// param; "all" is expressed by omitting it entirely.
		projectID := url.PathEscape(rem.Owner + "/" + rem.Repo)
		u := fmt.Sprintf("https://%s/api/v4/projects/%s/issues?per_page=20",
			hostRow.Host, projectID)
		switch state {
		case "open":
			u += "&state=opened"
		case "closed":
			u += "&state=closed"
		}
		body, ferr := s.fetch(ctx, u, "Bearer "+hostRow.Token, "application/json")
		if ferr != nil {
			return rem, nil, ferr
		}
		issues, derr := decodeGitLabIssues(body)
		return rem, issues, derr
	default:
		return rem, nil, fmt.Errorf("unsupported host kind: %s", hostRow.Kind)
	}
}

// GetIssue fetches a single issue — including its body — for the detail
// surface. Mirrors GetPullRequest.
func (s *Service) GetIssue(ctx context.Context, dir string, number int) (Issue, error) {
	rem, hostRow, err := s.resolveHost(ctx, dir)
	if err != nil {
		return Issue{}, err
	}
	if number <= 0 {
		return Issue{}, errors.New("number required")
	}
	switch hostRow.Kind {
	case KindGitHub:
		body, ferr := s.do(ctx, http.MethodGet, fmt.Sprintf("%s/repos/%s/%s/issues/%d",
			githubAPIBase(hostRow.Host), rem.Owner, rem.Repo, number),
			"Bearer "+hostRow.Token, "application/vnd.github+json", nil)
		if ferr != nil {
			return Issue{}, ferr
		}
		return decodeGitHubIssue(body)
	case KindGitea:
		body, ferr := s.do(ctx, http.MethodGet, fmt.Sprintf("https://%s/api/v1/repos/%s/%s/issues/%d",
			hostRow.Host, rem.Owner, rem.Repo, number),
			"token "+hostRow.Token, "application/json", nil)
		if ferr != nil {
			return Issue{}, ferr
		}
		return decodeGitHubIssue(body)
	case KindGitLab:
		projectID := url.PathEscape(rem.Owner + "/" + rem.Repo)
		body, ferr := s.do(ctx, http.MethodGet, fmt.Sprintf("https://%s/api/v4/projects/%s/issues/%d",
			hostRow.Host, projectID, number),
			"Bearer "+hostRow.Token, "application/json", nil)
		if ferr != nil {
			return Issue{}, ferr
		}
		return decodeGitLabIssue(body)
	default:
		return Issue{}, fmt.Errorf("unsupported host kind: %s", hostRow.Kind)
	}
}

// IssueComments returns an issue's conversation oldest-first. Reuses the
// PRComment type and the shared comment decoders. Unlike PRComments it
// never fetches reviews — issues have none, and the /pulls/{n}/reviews
// path would error for a non-PR number.
func (s *Service) IssueComments(ctx context.Context, dir string, number int) ([]PRComment, error) {
	rem, hostRow, err := s.resolveHost(ctx, dir)
	if err != nil {
		return nil, err
	}
	if number <= 0 {
		return nil, errors.New("number required")
	}
	switch hostRow.Kind {
	case KindGitHub:
		return s.githubIssueComments(ctx, hostRow, rem, number)
	case KindGitea:
		return s.giteaIssueComments(ctx, hostRow, rem, number)
	case KindGitLab:
		return s.gitlabIssueComments(ctx, hostRow, rem, number)
	default:
		return []PRComment{}, nil
	}
}

// ── decode: list ───────────────────────────────────────────────────

// githubIssueItem is one entry from GitHub/Gitea's /issues list. The
// PullRequest field is the discriminator: when present the item is a PR,
// not an issue, and must be skipped.
type githubIssueItem struct {
	Number      int             `json:"number"`
	Title       string          `json:"title"`
	State       string          `json:"state"`
	HTMLURL     string          `json:"html_url"`
	UpdatedAt   time.Time       `json:"updated_at"`
	PullRequest json.RawMessage `json:"pull_request"`
	User        struct {
		Login string `json:"login"`
	} `json:"user"`
	Labels []struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	} `json:"labels"`
}

// isPullRequestItem reports whether a GitHub/Gitea /issues entry is
// actually a pull request. The discriminator is a non-empty,
// non-null pull_request field; a missing field decodes to a zero-length
// RawMessage, and a literal `null` (some hosts) must also be treated as
// "not a PR".
func isPullRequestItem(pr json.RawMessage) bool {
	return len(pr) > 0 && string(pr) != "null"
}

// decodeGitHubIssues decodes a GitHub/Gitea /issues list, dropping any
// item that is actually a pull request (carries a pull_request field).
func decodeGitHubIssues(body []byte) ([]Issue, error) {
	var raw []githubIssueItem
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("github issues decode: %w", err)
	}
	out := make([]Issue, 0, len(raw))
	for _, it := range raw {
		if isPullRequestItem(it.PullRequest) {
			continue // a PR masquerading as an issue — skip it
		}
		out = append(out, Issue{
			Number:    it.Number,
			Title:     it.Title,
			State:     it.State,
			Author:    it.User.Login,
			URL:       it.HTMLURL,
			Labels:    githubStyleLabels(it.Labels),
			UpdatedAt: it.UpdatedAt,
		})
	}
	return out, nil
}

// gitlabIssueItem is one entry from GitLab's /issues list/detail. Labels
// are a flat []string and the identifier is the project-scoped iid.
type gitlabIssueItem struct {
	IID         int       `json:"iid"`
	Title       string    `json:"title"`
	State       string    `json:"state"`
	WebURL      string    `json:"web_url"`
	Description string    `json:"description"`
	UpdatedAt   time.Time `json:"updated_at"`
	Labels      []string  `json:"labels"`
	Author      struct {
		Username string `json:"username"`
	} `json:"author"`
}

func (it gitlabIssueItem) toIssue() Issue {
	return Issue{
		Number:    it.IID,
		Title:     it.Title,
		State:     normaliseGitlabState(it.State),
		Author:    it.Author.Username,
		URL:       it.WebURL,
		Body:      it.Description,
		Labels:    gitlabStringLabels(it.Labels),
		UpdatedAt: it.UpdatedAt,
	}
}

func decodeGitLabIssues(body []byte) ([]Issue, error) {
	var raw []gitlabIssueItem
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("gitlab issues decode: %w", err)
	}
	out := make([]Issue, 0, len(raw))
	for _, it := range raw {
		iss := it.toIssue()
		iss.Body = "" // list payloads stay lean; detail fetch fills Body
		out = append(out, iss)
	}
	return out, nil
}

// ── decode: detail ─────────────────────────────────────────────────

// decodeGitHubIssue decodes a single GitHub/Gitea issue, including the
// body. Shares the githubIssueItem shape plus a body field.
func decodeGitHubIssue(body []byte) (Issue, error) {
	var raw struct {
		githubIssueItem
		Body string `json:"body"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return Issue{}, fmt.Errorf("github issue decode: %w", err)
	}
	if isPullRequestItem(raw.PullRequest) {
		return Issue{}, errors.New("requested number is a pull request, not an issue")
	}
	return Issue{
		Number:    raw.Number,
		Title:     raw.Title,
		State:     raw.State,
		Author:    raw.User.Login,
		URL:       raw.HTMLURL,
		Body:      raw.Body,
		Labels:    githubStyleLabels(raw.Labels),
		UpdatedAt: raw.UpdatedAt,
	}, nil
}

func decodeGitLabIssue(body []byte) (Issue, error) {
	var raw gitlabIssueItem
	if err := json.Unmarshal(body, &raw); err != nil {
		return Issue{}, fmt.Errorf("gitlab issue decode: %w", err)
	}
	return raw.toIssue(), nil
}

// ── comments (issue-only, no reviews) ──────────────────────────────

func (s *Service) githubIssueComments(ctx context.Context, h Host, rem Remote, number int) ([]PRComment, error) {
	u := fmt.Sprintf("%s/repos/%s/%s/issues/%d/comments?per_page=100",
		githubAPIBase(h.Host), rem.Owner, rem.Repo, number)
	body, err := s.do(ctx, http.MethodGet, u, "Bearer "+h.Token, "application/vnd.github+json", nil)
	if err != nil {
		return nil, err
	}
	comments, err := decodeGitHubStyleComments(body)
	if err != nil {
		return nil, err
	}
	sortPRComments(comments)
	return comments, nil
}

func (s *Service) giteaIssueComments(ctx context.Context, h Host, rem Remote, number int) ([]PRComment, error) {
	u := fmt.Sprintf("https://%s/api/v1/repos/%s/%s/issues/%d/comments?limit=100",
		h.Host, rem.Owner, rem.Repo, number)
	body, err := s.do(ctx, http.MethodGet, u, "token "+h.Token, "application/json", nil)
	if err != nil {
		return nil, err
	}
	comments, err := decodeGitHubStyleComments(body)
	if err != nil {
		return nil, err
	}
	sortPRComments(comments)
	return comments, nil
}

func (s *Service) gitlabIssueComments(ctx context.Context, h Host, rem Remote, number int) ([]PRComment, error) {
	projectID := url.PathEscape(rem.Owner + "/" + rem.Repo)
	u := fmt.Sprintf("https://%s/api/v4/projects/%s/issues/%d/notes?sort=asc&per_page=100",
		h.Host, projectID, number)
	body, err := s.do(ctx, http.MethodGet, u, "Bearer "+h.Token, "application/json", nil)
	if err != nil {
		return nil, err
	}
	comments, err := decodeGitLabNotes(body)
	if err != nil {
		return nil, err
	}
	sortPRComments(comments)
	return comments, nil
}

// ── HTTP handlers ──────────────────────────────────────────────────

// issues mounts GET /git/issues?path=<dir>&state=<open|closed|all>.
// Mirrors prs; the no-token / error envelopes match so the client can
// reuse its handling. The list key is "issues".
func (h *Handlers) issues(w http.ResponseWriter, r *http.Request) {
	dir := strings.TrimSpace(r.URL.Query().Get("path"))
	if dir == "" {
		writeError(w, http.StatusBadRequest, errors.New("path is required"))
		return
	}
	state := r.URL.Query().Get("state")
	rem, issues, err := h.svc.ListIssues(r.Context(), dir, state)
	if errors.Is(err, ErrNoTokenForHost) {
		writeJSON(w, http.StatusOK, map[string]any{
			"remote":     rem,
			"issues":     []Issue{},
			"need_token": true,
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"remote": rem,
			"issues": []Issue{},
			"error":  err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"remote": rem,
		"issues": issues,
	})
}

// issueDetail mounts GET /git/issues/{number}?path=<dir>. Returns the
// full Issue envelope — including body — for the detail surface.
func (h *Handlers) issueDetail(w http.ResponseWriter, r *http.Request) {
	dir, n, ok := h.prTabParams(w, r)
	if !ok {
		return
	}
	iss, err := h.svc.GetIssue(r.Context(), dir, n)
	if err != nil {
		writeError(w, statusFromGitErr(err), err)
		return
	}
	writeJSON(w, http.StatusOK, iss)
}

// issueComments mounts GET /git/issues/{number}/comments?path=<dir>.
func (h *Handlers) issueComments(w http.ResponseWriter, r *http.Request) {
	dir, n, ok := h.prTabParams(w, r)
	if !ok {
		return
	}
	comments, err := h.svc.IssueComments(r.Context(), dir, n)
	if err != nil {
		writeError(w, statusFromGitErr(err), err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"comments": comments})
}

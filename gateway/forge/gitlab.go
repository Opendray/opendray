package forge

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// gitlabAdapter talks to GitLab's v4 REST API. GitLab diverges from
// GitHub / Gitea on three fronts:
//
//   1. Projects are addressed by URL-encoded path ("owner/name" →
//      "owner%2Fname") rather than literal slashes.
//   2. Pull requests are called "merge requests"; the path segment
//      is `/merge_requests/{iid}` and the number is `iid` (internal).
//   3. Auth uses the `PRIVATE-TOKEN` header for PATs, not the
//      generic Authorization header, so the adapter overrides
//      [newRequest]'s default.
type gitlabAdapter struct {
	cfg  Config
	http *http.Client
}

func (a *gitlabAdapter) apiURL(format string, args ...any) string {
	return fmt.Sprintf("%s/api/v4"+format,
		append([]any{trimSlash(a.cfg.BaseURL)}, args...)...)
}

func (a *gitlabAdapter) projectPath() (string, error) {
	// GitLab accepts the full namespaced path (e.g. "group/sub/repo")
	// as long as it's URL-encoded. We validate via splitRepo for the
	// minimum "owner/name" shape, then encode the whole thing.
	if _, _, err := splitRepo(a.cfg.Repo); err != nil {
		return "", err
	}
	return url.PathEscape(a.cfg.Repo), nil
}

// newReq overrides the default Authorization header; GitLab PATs go
// in PRIVATE-TOKEN. We strip the Authorization header [newRequest]
// set so a misconfigured forge (e.g. Authorization denied) can't
// double-auth into a 401 loop.
func (a *gitlabAdapter) newReq(ctx context.Context, method, url, accept string) (*http.Request, error) {
	req, err := newRequest(ctx, method, url, "", accept) // empty token = no Authorization header
	if err != nil {
		return nil, err
	}
	if a.cfg.Token != "" {
		req.Header.Set("PRIVATE-TOKEN", a.cfg.Token)
	}
	return req, nil
}

// gitlabMR mirrors the subset of MergeRequest JSON we need.
// DiffRefs.HeadSHA is the commit the checks + review-comments APIs
// need — surfaced on the wire as `diff_refs.head_sha`. Older GitLab
// versions also populate `sha` on the top-level MR; we fall back to
// that when diff_refs is absent.
type gitlabMR struct {
	IID          int       `json:"iid"`
	Title        string    `json:"title"`
	State        string    `json:"state"` // "opened" | "closed" | "merged"
	WebURL       string    `json:"web_url"`
	Description  string    `json:"description"`
	UserNotesCnt int       `json:"user_notes_count"`
	Draft        bool      `json:"draft"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	SHA          string    `json:"sha"`
	DiffRefs     struct {
		HeadSHA string `json:"head_sha"`
	} `json:"diff_refs"`
	Author struct {
		Username  string `json:"username"`
		AvatarURL string `json:"avatar_url"`
	} `json:"author"`
	SourceBranch string `json:"source_branch"`
	TargetBranch string `json:"target_branch"`
}

func (a *gitlabAdapter) translate(m gitlabMR) PullRequest {
	// Normalise GitLab's "opened" to "open" so the UI has one
	// vocabulary across forges.
	state := m.State
	if state == "opened" {
		state = "open"
	}
	headSHA := m.DiffRefs.HeadSHA
	if headSHA == "" {
		headSHA = m.SHA
	}
	return PullRequest{
		Number:       m.IID,
		Title:        m.Title,
		State:        state,
		Author:       m.Author.Username,
		AuthorAvatar: m.Author.AvatarURL,
		HeadRef:      m.SourceBranch,
		HeadSHA:      headSHA,
		BaseRef:      m.TargetBranch,
		URL:          m.WebURL,
		Body:         m.Description,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
		CommentCount: m.UserNotesCnt,
		Draft:        m.Draft,
	}
}

// gitlabStateParam maps our shared State enum to GitLab's naming.
func gitlabStateParam(s State) string {
	switch s {
	case StateOpen:
		return "opened"
	case StateClosed:
		return "closed"
	case StateAll:
		return "all"
	default:
		return "opened"
	}
}

func (a *gitlabAdapter) list(ctx context.Context, state State, limit int) ([]PullRequest, error) {
	project, err := a.projectPath()
	if err != nil {
		return nil, err
	}
	url := a.apiURL("/projects/%s/merge_requests?state=%s&per_page=%d",
		project, gitlabStateParam(state), limit)
	req, err := a.newReq(ctx, http.MethodGet, url, "")
	if err != nil {
		return nil, err
	}
	var raw []gitlabMR
	if err := doJSON(a.http, req, &raw); err != nil {
		return nil, err
	}
	out := make([]PullRequest, 0, len(raw))
	for _, m := range raw {
		out = append(out, a.translate(m))
	}
	return out, nil
}

func (a *gitlabAdapter) detail(ctx context.Context, number int) (PullRequest, error) {
	project, err := a.projectPath()
	if err != nil {
		return PullRequest{}, err
	}
	url := a.apiURL("/projects/%s/merge_requests/%d", project, number)
	req, err := a.newReq(ctx, http.MethodGet, url, "")
	if err != nil {
		return PullRequest{}, err
	}
	var raw gitlabMR
	if err := doJSON(a.http, req, &raw); err != nil {
		return PullRequest{}, err
	}
	return a.translate(raw), nil
}

// gitlabDiffChange mirrors one entry of /merge_requests/:iid/changes.
// GitLab gives us per-file metadata directly, so we don't need to
// parse a unified diff blob like Gitea/GitHub do.
type gitlabDiffChange struct {
	OldPath     string `json:"old_path"`
	NewPath     string `json:"new_path"`
	NewFile     bool   `json:"new_file"`
	DeletedFile bool   `json:"deleted_file"`
	RenamedFile bool   `json:"renamed_file"`
	Diff        string `json:"diff"`
}

type gitlabChangesResponse struct {
	Changes []gitlabDiffChange `json:"changes"`
}

func (a *gitlabAdapter) diff(ctx context.Context, number int) ([]DiffFile, error) {
	project, err := a.projectPath()
	if err != nil {
		return nil, err
	}
	url := a.apiURL("/projects/%s/merge_requests/%d/changes", project, number)
	req, err := a.newReq(ctx, http.MethodGet, url, "")
	if err != nil {
		return nil, err
	}
	var raw gitlabChangesResponse
	if err := doJSON(a.http, req, &raw); err != nil {
		return nil, err
	}
	out := make([]DiffFile, 0, len(raw.Changes))
	for _, c := range raw.Changes {
		status := "modified"
		switch {
		case c.NewFile:
			status = "added"
		case c.DeletedFile:
			status = "deleted"
		case c.RenamedFile:
			status = "renamed"
		}
		adds, dels := countHunkLines(c.Diff)
		oldPath := c.OldPath
		if oldPath == c.NewPath {
			oldPath = ""
		}
		out = append(out, DiffFile{
			Path:      c.NewPath,
			OldPath:   oldPath,
			Status:    status,
			Additions: adds,
			Deletions: dels,
			Patch:     c.Diff,
		})
	}
	return out, nil
}

type gitlabNote struct {
	ID         int64     `json:"id"`
	Body       string    `json:"body"`
	CreatedAt  time.Time `json:"created_at"`
	System     bool      `json:"system"`
	Author     struct {
		Username string `json:"username"`
	} `json:"author"`
}

// ── Reviews / review-comments / checks (Tier A) ─────────────────

// gitlabApproval mirrors /merge_requests/:iid/approvals. GitLab
// doesn't model "approve vs changes_requested" — it's a binary
// "approved or not" with a list of approvers. We emit one Review
// row per approver at State="approved"; reviewers who left inline
// comments without approving surface under ReviewComments, not
// here, because GitLab's data model doesn't link them.
type gitlabApproval struct {
	ApprovedBy []struct {
		User struct {
			Username  string `json:"username"`
			AvatarURL string `json:"avatar_url"`
		} `json:"user"`
	} `json:"approved_by"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (a *gitlabAdapter) reviews(ctx context.Context, number int) ([]Review, error) {
	project, err := a.projectPath()
	if err != nil {
		return nil, err
	}
	url := a.apiURL("/projects/%s/merge_requests/%d/approvals", project, number)
	req, err := a.newReq(ctx, http.MethodGet, url, "")
	if err != nil {
		return nil, err
	}
	var raw gitlabApproval
	if err := doJSON(a.http, req, &raw); err != nil {
		// Approvals API is CE-gated on some versions — fall back to
		// an empty list rather than surfacing the error, since
		// "we don't know" is a better UX than "API broken" when
		// the feature just isn't available on this GitLab edition.
		return []Review{}, nil
	}
	out := make([]Review, 0, len(raw.ApprovedBy))
	for _, a := range raw.ApprovedBy {
		out = append(out, Review{
			Author:       a.User.Username,
			AuthorAvatar: a.User.AvatarURL,
			State:        "approved",
			SubmittedAt:  raw.UpdatedAt,
		})
	}
	return out, nil
}

// gitlabNote for inline comments is the same /notes endpoint
// comments() uses, but we keep only the notes that have a `position`
// block (line + file) — those are the inline ones.
type gitlabPositionNote struct {
	ID        int64     `json:"id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	System    bool      `json:"system"`
	Author    struct {
		Username string `json:"username"`
	} `json:"author"`
	Position *struct {
		NewPath string `json:"new_path"`
		OldPath string `json:"old_path"`
		NewLine int    `json:"new_line"`
		OldLine int    `json:"old_line"`
	} `json:"position"`
}

func (a *gitlabAdapter) reviewComments(ctx context.Context, number int) ([]ReviewComment, error) {
	project, err := a.projectPath()
	if err != nil {
		return nil, err
	}
	url := a.apiURL("/projects/%s/merge_requests/%d/notes?sort=asc&per_page=100",
		project, number)
	req, err := a.newReq(ctx, http.MethodGet, url, "")
	if err != nil {
		return nil, err
	}
	var raw []gitlabPositionNote
	if err := doJSON(a.http, req, &raw); err != nil {
		return nil, err
	}
	out := make([]ReviewComment, 0, len(raw))
	for _, n := range raw {
		if n.System || n.Position == nil {
			continue
		}
		path := n.Position.NewPath
		if path == "" {
			path = n.Position.OldPath
		}
		line := n.Position.NewLine
		if line == 0 {
			line = n.Position.OldLine
		}
		out = append(out, ReviewComment{
			ID: n.ID, Author: n.Author.Username, Body: n.Body,
			CreatedAt: n.CreatedAt, Path: path, Line: line,
		})
	}
	return out, nil
}

// gitlabPipeline mirrors one pipeline entry. GitLab pipelines are
// per-commit (not per-MR), so we filter the project's pipeline list
// by SHA then flatten each pipeline into a single CheckRun. Job-level
// granularity is available via /pipelines/:id/jobs but that's one
// extra round-trip we defer until a user asks.
type gitlabPipeline struct {
	ID         int64      `json:"id"`
	Status     string     `json:"status"` // created | waiting_for_resource | preparing | pending | running | success | failed | canceled | skipped | manual | scheduled
	Ref        string     `json:"ref"`
	SHA        string     `json:"sha"`
	WebURL     string     `json:"web_url"`
	Source     string     `json:"source"` // push | merge_request_event | ...
	UpdatedAt  time.Time  `json:"updated_at"`
	FinishedAt *time.Time `json:"finished_at"`
}

func (a *gitlabAdapter) checks(ctx context.Context, _ int, headSHA string) ([]CheckRun, error) {
	project, err := a.projectPath()
	if err != nil {
		return nil, err
	}
	url := a.apiURL("/projects/%s/pipelines?sha=%s&per_page=20", project, headSHA)
	req, err := a.newReq(ctx, http.MethodGet, url, "")
	if err != nil {
		return nil, err
	}
	var raw []gitlabPipeline
	if err := doJSON(a.http, req, &raw); err != nil {
		return nil, err
	}
	out := make([]CheckRun, 0, len(raw))
	for _, p := range raw {
		name := firstNonEmpty(p.Source, p.Ref)
		out = append(out, CheckRun{
			Name:        name,
			Context:     "GitLab CI",
			Status:      normaliseCheckStatus(p.Status),
			TargetURL:   p.WebURL,
			CompletedAt: p.FinishedAt,
		})
	}
	return out, nil
}

func (a *gitlabAdapter) comments(ctx context.Context, number int) ([]Comment, error) {
	project, err := a.projectPath()
	if err != nil {
		return nil, err
	}
	// GitLab mixes "system" notes (auto-generated: "mentioned in
	// commit…", "assigned to…") with human notes in the same feed.
	// Users want the human ones in a review surface, so we filter
	// system notes out.
	url := a.apiURL("/projects/%s/merge_requests/%d/notes?sort=asc", project, number)
	req, err := a.newReq(ctx, http.MethodGet, url, "")
	if err != nil {
		return nil, err
	}
	var raw []gitlabNote
	if err := doJSON(a.http, req, &raw); err != nil {
		return nil, err
	}
	out := make([]Comment, 0, len(raw))
	for _, n := range raw {
		if n.System {
			continue
		}
		out = append(out, Comment{
			ID: n.ID, Author: n.Author.Username, Body: n.Body,
			CreatedAt: n.CreatedAt,
		})
	}
	return out, nil
}

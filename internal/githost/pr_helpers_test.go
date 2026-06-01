package githost

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGithubAPIBase(t *testing.T) {
	cases := []struct{ in, want string }{
		{"github.com", "https://api.github.com"},
		{"git.example.com", "https://git.example.com/api/v3"},
		{"github.acme.io", "https://github.acme.io/api/v3"},
	}
	for _, c := range cases {
		if got := githubAPIBase(c.in); got != c.want {
			t.Errorf("githubAPIBase(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestGiteaMergeMethod(t *testing.T) {
	cases := []struct{ in, want string }{
		{"squash", "squash"},
		{"merge", "merge"},
		{"rebase", "rebase"},
		{"", "squash"},        // empty → default
		{"unknown", "squash"}, // garbage → default
	}
	for _, c := range cases {
		if got := giteaMergeMethod(c.in); got != c.want {
			t.Errorf("giteaMergeMethod(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestDecodeGiteaPR_MergedState(t *testing.T) {
	body := []byte(`{
		"number": 42,
		"title": "fix: thing",
		"state": "closed",
		"html_url": "https://git.example.com/o/r/pulls/42",
		"updated_at": "2026-05-04T10:00:00Z",
		"user": {"login": "alice"},
		"head": {"ref": "feat/x"},
		"base": {"ref": "main"},
		"merged": true
	}`)
	got, err := decodeGiteaPR(body)
	if err != nil {
		t.Fatal(err)
	}
	if got.Number != 42 || got.Title != "fix: thing" {
		t.Errorf("wrong meta: %+v", got)
	}
	if got.State != "merged" {
		t.Errorf("merged:true should override state=closed → merged, got %q", got.State)
	}
	if got.Author != "alice" || got.Head != "feat/x" || got.Base != "main" {
		t.Errorf("wrong attribution / branches: %+v", got)
	}
}

func TestDecodeGitLabMR_StateNormalised(t *testing.T) {
	body := []byte(`{
		"iid": 7,
		"title": "feat: thing",
		"state": "opened",
		"web_url": "https://gitlab.com/g/r/-/merge_requests/7",
		"updated_at": "2026-05-04T10:00:00Z",
		"author": {"username": "bob"},
		"source_branch": "feat/x",
		"target_branch": "main",
		"draft": false
	}`)
	got, err := decodeGitLabMR(body)
	if err != nil {
		t.Fatal(err)
	}
	if got.Number != 7 {
		t.Errorf("iid → number wrong: %d", got.Number)
	}
	if got.State != "open" {
		t.Errorf("GitLab 'opened' should normalise to 'open', got %q", got.State)
	}
	if got.Author != "bob" || got.Head != "feat/x" || got.Base != "main" {
		t.Errorf("wrong attribution / branches: %+v", got)
	}
	if !strings.Contains(got.URL, "merge_requests/7") {
		t.Errorf("url wrong: %q", got.URL)
	}
}

func TestGithubPRResponse_ToPullRequest_MergedTimestamp(t *testing.T) {
	body := []byte(`{
		"number": 100,
		"title": "merged thing",
		"state": "closed",
		"html_url": "https://github.com/o/r/pull/100",
		"user": {"login": "carol"},
		"head": {"ref": "feat/y"},
		"base": {"ref": "main"},
		"merged_at": "2026-05-04T10:00:00Z"
	}`)
	var raw githubPRResponse
	if err := jsonUnmarshalTestHelper(body, &raw); err != nil {
		t.Fatal(err)
	}
	got := raw.toPullRequest()
	if got.State != "merged" {
		t.Errorf("merged_at!=nil should produce state=merged, got %q", got.State)
	}
}

// jsonUnmarshalTestHelper keeps `encoding/json` referenced from the
// test file (most of the other assertions go through pure helpers
// that don't import json directly).
func jsonUnmarshalTestHelper(body []byte, v any) error {
	return json.Unmarshal(body, v)
}

func TestCommitStatusState_MappingMatrix(t *testing.T) {
	cases := []struct {
		in             string
		wantStatus     string
		wantConclusion string
	}{
		// Gitea
		{"success", "completed", "success"},
		{"failure", "completed", "failure"},
		{"error", "completed", "failure"},
		{"warning", "completed", "neutral"},
		{"pending", "queued", ""},
		// GitLab
		{"running", "in_progress", ""},
		{"failed", "completed", "failure"},
		{"canceled", "completed", "cancelled"},
		{"cancelled", "completed", "cancelled"},
		{"skipped", "completed", "skipped"},
		{"manual", "completed", "action_required"},
		{"scheduled", "queued", ""},
		// Mixed case + unknowns
		{"SUCCESS", "completed", "success"},
		{"in_progress", "in_progress", ""},
		{"weird", "completed", "neutral"},
		{"", "completed", "neutral"},
	}
	for _, c := range cases {
		gotStatus, gotConclusion := commitStatusState(c.in)
		if gotStatus != c.wantStatus || gotConclusion != c.wantConclusion {
			t.Errorf("commitStatusState(%q) = (%q, %q), want (%q, %q)",
				c.in, gotStatus, gotConclusion, c.wantStatus, c.wantConclusion)
		}
	}
}

// ── PR body / description capture ──────────────────────────────────
// The detail surface (web drawer / mobile screen) needs the PR body.
// Each host adapter must surface it on PullRequest.Body: GitHub and
// Gitea call it "body"; GitLab calls it "description".

func TestDecodeGiteaPR_Body(t *testing.T) {
	body := []byte(`{
		"number": 7,
		"title": "feat: x",
		"state": "open",
		"body": "## Summary\nDoes the thing.",
		"html_url": "https://git.example.com/o/r/pulls/7",
		"updated_at": "2026-05-04T10:00:00Z",
		"user": {"login": "alice"},
		"head": {"ref": "feat/x"},
		"base": {"ref": "main"}
	}`)
	got, err := decodeGiteaPR(body)
	if err != nil {
		t.Fatal(err)
	}
	if got.Body != "## Summary\nDoes the thing." {
		t.Errorf("gitea body not captured, got %q", got.Body)
	}
}

func TestDecodeGitLabMR_DescriptionToBody(t *testing.T) {
	body := []byte(`{
		"iid": 9,
		"title": "feat: y",
		"state": "opened",
		"description": "Closes #1. Adds the widget.",
		"web_url": "https://gitlab.com/g/r/-/merge_requests/9",
		"updated_at": "2026-05-04T10:00:00Z",
		"author": {"username": "bob"},
		"source_branch": "feat/y",
		"target_branch": "main"
	}`)
	got, err := decodeGitLabMR(body)
	if err != nil {
		t.Fatal(err)
	}
	if got.Body != "Closes #1. Adds the widget." {
		t.Errorf("gitlab description should map to body, got %q", got.Body)
	}
}

func TestGithubPRResponse_BodyPassthrough(t *testing.T) {
	body := []byte(`{
		"number": 11,
		"title": "fix: z",
		"state": "open",
		"body": "Fixes the crash on startup.",
		"html_url": "https://github.com/o/r/pull/11",
		"updated_at": "2026-05-04T10:00:00Z",
		"user": {"login": "carol"},
		"head": {"ref": "fix/z"},
		"base": {"ref": "main"}
	}`)
	var raw githubPRResponse
	if err := jsonUnmarshalTestHelper(body, &raw); err != nil {
		t.Fatal(err)
	}
	got := raw.toPullRequest()
	if got.Body != "Fixes the crash on startup." {
		t.Errorf("github body should pass through toPullRequest, got %q", got.Body)
	}
}

// A description-less PR comes back with "body": null (GitHub) or the
// field omitted (Gitea / GitLab). All three must decode to an empty
// Body, never the literal "null".
func TestDecoders_NullOrMissingBody(t *testing.T) {
	var gh githubPRResponse
	if err := jsonUnmarshalTestHelper([]byte(`{
		"number": 1, "title": "t", "state": "open", "body": null,
		"user": {"login": "a"}, "head": {"ref": "x"}, "base": {"ref": "main"}
	}`), &gh); err != nil {
		t.Fatal(err)
	}
	if got := gh.toPullRequest(); got.Body != "" {
		t.Errorf("github null body should decode to empty, got %q", got.Body)
	}

	gitea, err := decodeGiteaPR([]byte(`{
		"number": 1, "title": "t", "state": "open",
		"user": {"login": "a"}, "head": {"ref": "x"}, "base": {"ref": "main"}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if gitea.Body != "" {
		t.Errorf("gitea missing body should be empty, got %q", gitea.Body)
	}

	gitlab, err := decodeGitLabMR([]byte(`{
		"iid": 1, "title": "t", "state": "opened",
		"author": {"username": "a"},
		"source_branch": "x", "target_branch": "main"
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if gitlab.Body != "" {
		t.Errorf("gitlab missing description should be empty, got %q", gitlab.Body)
	}
}

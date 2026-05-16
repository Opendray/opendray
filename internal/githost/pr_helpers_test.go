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

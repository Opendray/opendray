package githost

import (
	"testing"
)

// TestDecodeGitHubIssues_FiltersPRs is the highest-value test: GitHub's
// /issues endpoint returns pull requests too (each carrying a
// pull_request field). The list adapter must drop them so PRs don't
// double-show as issues.
func TestDecodeGitHubIssues_FiltersPRs(t *testing.T) {
	body := []byte(`[
		{"number": 7, "title": "real issue", "state": "open",
		 "html_url": "https://github.com/o/r/issues/7",
		 "updated_at": "2026-05-04T10:00:00Z", "user": {"login": "alice"},
		 "labels": [{"name": "bug", "color": "d73a4a"}]},
		{"number": 8, "title": "a PR", "state": "open",
		 "html_url": "https://github.com/o/r/pull/8",
		 "updated_at": "2026-05-04T11:00:00Z", "user": {"login": "bob"},
		 "pull_request": {"url": "https://api.github.com/repos/o/r/pulls/8"}},
		{"number": 9, "title": "another issue", "state": "closed",
		 "html_url": "https://github.com/o/r/issues/9",
		 "updated_at": "2026-05-04T12:00:00Z", "user": {"login": "carol"},
		 "labels": []}
	]`)
	got, err := decodeGitHubIssues(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 issues (PR #8 filtered), got %d: %+v", len(got), got)
	}
	if got[0].Number != 7 || got[1].Number != 9 {
		t.Errorf("wrong issues survived: %+v", got)
	}
	if got[0].Author != "alice" || got[0].State != "open" {
		t.Errorf("issue #7 meta wrong: %+v", got[0])
	}
}

func TestDecodeGitHubIssues_Labels(t *testing.T) {
	body := []byte(`[
		{"number": 1, "title": "t", "state": "open", "html_url": "u",
		 "updated_at": "2026-05-04T10:00:00Z", "user": {"login": "a"},
		 "labels": [{"name": "bug", "color": "d73a4a"}, {"name": "p1", "color": "ff0000"}]}
	]`)
	got, err := decodeGitHubIssues(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || len(got[0].Labels) != 2 {
		t.Fatalf("want 1 issue with 2 labels, got %+v", got)
	}
	if got[0].Labels[0].Name != "bug" || got[0].Labels[0].Color != "d73a4a" {
		t.Errorf("label[0] wrong: %+v", got[0].Labels[0])
	}
	if got[0].Labels[1].Name != "p1" || got[0].Labels[1].Color != "ff0000" {
		t.Errorf("label[1] wrong: %+v", got[0].Labels[1])
	}
}

// TestDecodeGitHubIssues_EmptyLabelsNonNil guards the JSON contract: the
// Labels field has no omitempty, so it must serialise as [] not null.
func TestDecodeGitHubIssues_EmptyLabelsNonNil(t *testing.T) {
	body := []byte(`[
		{"number": 1, "title": "t", "state": "open", "html_url": "u",
		 "updated_at": "2026-05-04T10:00:00Z", "user": {"login": "a"}}
	]`)
	got, err := decodeGitHubIssues(body)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Labels == nil {
		t.Error("Labels should be non-nil empty slice, got nil")
	}
}

func TestDecodeGitHubIssue_Detail(t *testing.T) {
	body := []byte(`{
		"number": 12, "title": "detail issue", "state": "open",
		"html_url": "https://github.com/o/r/issues/12",
		"updated_at": "2026-05-04T10:00:00Z", "user": {"login": "dora"},
		"body": "the description",
		"labels": [{"name": "enhancement", "color": "a2eeef"}]
	}`)
	got, err := decodeGitHubIssue(body)
	if err != nil {
		t.Fatal(err)
	}
	if got.Number != 12 || got.Body != "the description" {
		t.Errorf("detail meta/body wrong: %+v", got)
	}
	if len(got.Labels) != 1 || got.Labels[0].Name != "enhancement" {
		t.Errorf("detail labels wrong: %+v", got.Labels)
	}
}

func TestDecodeGitLabIssues_StringLabelsAndState(t *testing.T) {
	body := []byte(`[
		{"iid": 3, "title": "gl issue", "state": "opened",
		 "web_url": "https://gitlab.com/o/r/-/issues/3",
		 "updated_at": "2026-05-04T10:00:00Z", "author": {"username": "eve"},
		 "labels": ["bug", "p1"]}
	]`)
	got, err := decodeGitLabIssues(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 issue, got %d", len(got))
	}
	is := got[0]
	if is.Number != 3 || is.Author != "eve" {
		t.Errorf("gitlab meta wrong (iid→number, author): %+v", is)
	}
	if is.State != "open" {
		t.Errorf("gitlab state should normalise opened→open, got %q", is.State)
	}
	if len(is.Labels) != 2 || is.Labels[0].Name != "bug" || is.Labels[0].Color != "" {
		t.Errorf("gitlab string labels should map to {name, empty color}: %+v", is.Labels)
	}
	// list payloads must stay lean — Body is detail-only.
	if is.Body != "" {
		t.Errorf("list issue should not carry body, got %q", is.Body)
	}
}

func TestDecodeGitLabIssue_Detail(t *testing.T) {
	body := []byte(`{
		"iid": 5, "title": "gl detail", "state": "closed",
		"web_url": "https://gitlab.com/o/r/-/issues/5",
		"description": "gl body text",
		"updated_at": "2026-05-04T10:00:00Z", "author": {"username": "frank"},
		"labels": ["wontfix"]
	}`)
	got, err := decodeGitLabIssue(body)
	if err != nil {
		t.Fatal(err)
	}
	if got.Number != 5 || got.State != "closed" || got.Body != "gl body text" {
		t.Errorf("gitlab detail wrong: %+v", got)
	}
	if len(got.Labels) != 1 || got.Labels[0].Name != "wontfix" {
		t.Errorf("gitlab detail labels wrong: %+v", got.Labels)
	}
}

func TestGitlabStringLabels_EmptyNonNil(t *testing.T) {
	if got := gitlabStringLabels(nil); got == nil {
		t.Error("gitlabStringLabels(nil) should return non-nil empty slice")
	}
}

// TestIsPullRequestItem covers the discriminator's three input shapes:
// missing (zero-length), literal null, and a real object.
func TestIsPullRequestItem(t *testing.T) {
	cases := []struct {
		name string
		in   string // raw bytes; "" means a missing field (zero-length)
		want bool
	}{
		{"missing", "", false},
		{"null", "null", false},
		{"object", `{"url":"https://api.github.com/repos/o/r/pulls/8"}`, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var raw []byte
			if c.in != "" {
				raw = []byte(c.in)
			}
			if got := isPullRequestItem(raw); got != c.want {
				t.Errorf("isPullRequestItem(%q) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

// TestDecodeGitHubIssues_NullPullRequestKept guards the robustness fix:
// a real issue carrying "pull_request": null must NOT be filtered out.
func TestDecodeGitHubIssues_NullPullRequestKept(t *testing.T) {
	body := []byte(`[
		{"number": 7, "title": "real issue", "state": "open", "html_url": "u",
		 "updated_at": "2026-05-04T10:00:00Z", "user": {"login": "alice"},
		 "pull_request": null}
	]`)
	got, err := decodeGitHubIssues(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Number != 7 {
		t.Errorf("issue with null pull_request should be kept, got %+v", got)
	}
}

// TestDecodeGitHubIssue_RejectsPR guards the detail-fetch guard: asking
// for a number that resolves to a PR returns an error, not a fake issue.
func TestDecodeGitHubIssue_RejectsPR(t *testing.T) {
	body := []byte(`{
		"number": 8, "title": "a PR", "state": "open", "html_url": "u",
		"updated_at": "2026-05-04T10:00:00Z", "user": {"login": "bob"},
		"pull_request": {"url": "https://api.github.com/repos/o/r/pulls/8"}
	}`)
	if _, err := decodeGitHubIssue(body); err == nil {
		t.Error("decodeGitHubIssue should reject a pull request, got nil error")
	}
}

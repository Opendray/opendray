package githost

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
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

// ── detail tabs: commits / files / comments decoders ───────────────

func TestDecodeGitHubStyleCommits(t *testing.T) {
	body := []byte(`[
	  {"sha":"abcdef1234567890","html_url":"https://gh/c/abcdef1",
	   "commit":{"message":"feat: x\n\nbody","author":{"name":"Alice","date":"2026-05-01T10:00:00Z"}},
	   "author":{"login":"alice"}},
	  {"sha":"0123456","html_url":"",
	   "commit":{"message":"fix","author":{"name":"Bob","date":"2026-05-02T10:00:00Z"}},
	   "author":{"login":""}}
	]`)
	got, err := decodeGitHubStyleCommits(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 commits, got %d", len(got))
	}
	if got[0].ShortSHA != "abcdef1" {
		t.Errorf("short sha = %q, want abcdef1", got[0].ShortSHA)
	}
	if got[0].Author != "alice" {
		t.Errorf("author = %q, want alice", got[0].Author)
	}
	if got[1].Author != "Bob" {
		t.Errorf("missing login should fall back to commit author name, got %q", got[1].Author)
	}
	if got[1].ShortSHA != "0123456" {
		t.Errorf("7-char sha should be returned as-is, got %q", got[1].ShortSHA)
	}
}

func TestDecodeGitLabCommits(t *testing.T) {
	body := []byte(`[{"id":"deadbeefcafe","short_id":"deadbee","title":"t","message":"",
	   "author_name":"Carol","created_at":"2026-05-01T10:00:00Z","web_url":"https://gl/c"}]`)
	got, err := decodeGitLabCommits(body)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].SHA != "deadbeefcafe" || got[0].ShortSHA != "deadbee" {
		t.Errorf("ids wrong: %+v", got[0])
	}
	if got[0].Message != "t" {
		t.Errorf("empty message should fall back to title, got %q", got[0].Message)
	}
	if got[0].Author != "Carol" {
		t.Errorf("author = %q", got[0].Author)
	}
}

func TestDecodeGitHubStyleFiles(t *testing.T) {
	body := []byte(`[
	  {"filename":"a.go","status":"modified","additions":3,"deletions":1,"patch":"@@ -1 +1,3 @@"},
	  {"filename":"b.go","status":"removed","additions":0,"deletions":9}
	]`)
	got, err := decodeGitHubStyleFiles(body)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Patch == "" {
		t.Errorf("patch should be captured for github files")
	}
	if got[1].Status != "removed" {
		t.Errorf("status = %q, want removed", got[1].Status)
	}
	if got[1].Patch != "" {
		t.Errorf("absent patch should decode to empty, got %q", got[1].Patch)
	}
}

func TestDecodeGitLabDiffs_StatusAndCounts(t *testing.T) {
	body := []byte(`[
	  {"old_path":"a.go","new_path":"a.go","new_file":false,"renamed_file":false,"deleted_file":false,
	   "diff":"@@\n+added one\n+added two\n-removed one\n unchanged\n"},
	  {"old_path":"gone.go","new_path":"gone.go","deleted_file":true,
	   "diff":"--- a/gone.go\n+++ /dev/null\n-x\n"}
	]`)
	got, err := decodeGitLabDiffs(body)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Additions != 2 || got[0].Deletions != 1 {
		t.Errorf("counts derived from diff wrong: +%d -%d, want +2 -1", got[0].Additions, got[0].Deletions)
	}
	if got[1].Status != "removed" || got[1].Filename != "gone.go" {
		t.Errorf("deleted file: status=%q name=%q", got[1].Status, got[1].Filename)
	}
	// The +++ / --- headers in file 2 must not be counted as add/del.
	if got[1].Additions != 0 || got[1].Deletions != 1 {
		t.Errorf("diff headers miscounted: +%d -%d, want +0 -1", got[1].Additions, got[1].Deletions)
	}
}

func TestDecodeComments_AndReviews(t *testing.T) {
	comments, err := decodeGitHubStyleComments([]byte(
		`[{"user":{"login":"alice"},"body":"hi","created_at":"2026-05-01T10:00:00Z","html_url":"u"}]`))
	if err != nil {
		t.Fatal(err)
	}
	if len(comments) != 1 || comments[0].Author != "alice" || comments[0].Body != "hi" {
		t.Errorf("comment decode: %+v", comments)
	}
	reviews, err := decodeGitHubStyleReviews([]byte(`[
	  {"user":{"login":"bob"},"body":"please fix","state":"CHANGES_REQUESTED","submitted_at":"2026-05-01T11:00:00Z"},
	  {"user":{"login":"carol"},"body":"","state":"APPROVED","submitted_at":"2026-05-01T12:00:00Z"},
	  {"user":{"login":"dave"},"body":"changes","state":"REQUEST_CHANGES","submitted_at":"2026-05-01T13:00:00Z"}
	]`))
	if err != nil {
		t.Fatal(err)
	}
	if len(reviews) != 2 {
		t.Fatalf("bodyless review should be dropped; want 2 got %d", len(reviews))
	}
	if reviews[0].State != "changes_requested" {
		t.Errorf("github CHANGES_REQUESTED normalize: %q", reviews[0].State)
	}
	if reviews[1].State != "changes_requested" {
		t.Errorf("gitea REQUEST_CHANGES should normalize to changes_requested: %q", reviews[1].State)
	}
}

func TestDecodeGitLabNotes_DropSystem(t *testing.T) {
	got, err := decodeGitLabNotes([]byte(`[
	  {"author":{"username":"alice"},"body":"real comment","created_at":"2026-05-01T10:00:00Z","system":false},
	  {"author":{"username":"bot"},"body":"changed the description","created_at":"2026-05-01T10:01:00Z","system":true}
	]`))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Author != "alice" {
		t.Errorf("system notes must be dropped, got %+v", got)
	}
}

func TestSortPRComments(t *testing.T) {
	mk := func(iso string) PRComment {
		ts, _ := time.Parse(time.RFC3339, iso)
		return PRComment{CreatedAt: ts}
	}
	c := []PRComment{
		mk("2026-05-03T00:00:00Z"),
		mk("2026-05-01T00:00:00Z"),
		mk("2026-05-02T00:00:00Z"),
	}
	sortPRComments(c)
	if !c[0].CreatedAt.Before(c[1].CreatedAt) || !c[1].CreatedAt.Before(c[2].CreatedAt) {
		t.Errorf("comments not sorted ascending by time")
	}
}

func TestCountDiffLines(t *testing.T) {
	add, del := countDiffLines("@@ -1,2 +1,3 @@\n context\n+new line\n+another\n-gone\n")
	if add != 2 || del != 1 {
		t.Errorf("countDiffLines = +%d -%d, want +2 -1", add, del)
	}
}

func TestCountDiffLines_NoNewlineMarker(t *testing.T) {
	// The "\ No newline at end of file" marker must not be counted.
	add, del := countDiffLines(
		"@@ -1 +1 @@\n-old\n\\ No newline at end of file\n+new\n\\ No newline at end of file\n",
	)
	if add != 1 || del != 1 {
		t.Errorf("no-newline marker miscounted: +%d -%d, want +1 -1", add, del)
	}
}

func TestNormalizeReviewState(t *testing.T) {
	cases := map[string]string{
		"APPROVED":          "approved",
		"CHANGES_REQUESTED": "changes_requested", // GitHub
		"REQUEST_CHANGES":   "changes_requested", // Gitea
		"COMMENTED":         "commented",
		"COMMENT":           "commented", // Gitea
		"DISMISSED":         "dismissed",
		"weird":             "weird", // unknown → lowercased
	}
	for in, want := range cases {
		if got := normalizeReviewState(in); got != want {
			t.Errorf("normalizeReviewState(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeFileStatus(t *testing.T) {
	cases := map[string]string{
		"added":    "added",
		"new":      "added",
		"removed":  "removed",
		"deleted":  "removed",
		"renamed":  "renamed",
		"modified": "modified",
		"":         "modified",
		"changed":  "changed", // unknown → passed through lowercased
	}
	for in, want := range cases {
		if got := normalizeFileStatus(in); got != want {
			t.Errorf("normalizeFileStatus(%q) = %q, want %q", in, got, want)
		}
	}
}

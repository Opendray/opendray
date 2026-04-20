package forge

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newTestConfig builds a Config pinned at the given httptest URL
// for the given forge. Token is fixed to "t-test" so the auth
// header propagation is observable in the handler.
func newTestConfig(forgeType, baseURL string) Config {
	return Config{
		ForgeType: forgeType,
		BaseURL:   baseURL,
		Repo:      "kev/opendray",
		Token:     "t-test",
		Timeout:   3 * time.Second,
	}
}

func TestPick_RejectsMissingOrUnknownForgeType(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
	}{
		{"missing type", Config{BaseURL: "https://x", Repo: "a/b"}},
		{"unknown type", Config{ForgeType: "bitbucket", BaseURL: "https://x", Repo: "a/b"}},
		{"missing baseURL", Config{ForgeType: "gitea", Repo: "a/b"}},
		{"missing repo", Config{ForgeType: "gitea", BaseURL: "https://x"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := pick(tc.cfg); err == nil {
				t.Error("expected pick to reject config")
			}
		})
	}
}

func TestGitea_ListDetailDiffComments(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Auth propagation: Gitea uses "token <PAT>" on the
		// Authorization header. Fail fast if the adapter regresses.
		if got := r.Header.Get("Authorization"); got != "token t-test" {
			t.Errorf("Authorization = %q, want %q", got, "token t-test")
		}
		switch {
		case r.URL.Path == "/api/v1/repos/kev/opendray/pulls":
			if r.URL.Query().Get("state") != "open" {
				t.Errorf("state = %q", r.URL.Query().Get("state"))
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]giteaPR{{
				Number: 42, Title: "add forge plugin",
				State: "open", HTMLURL: "https://gitea.test/kev/opendray/pulls/42",
				Comments: 3,
				CreatedAt: time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2026, 4, 21, 9, 0, 0, 0, time.UTC),
			}})
		case r.URL.Path == "/api/v1/repos/kev/opendray/pulls/42":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(giteaPR{
				Number: 42, Title: "add forge plugin",
				State: "open", Merged: false,
				Body: "See RFC for motivation.",
			})
		case r.URL.Path == "/api/v1/repos/kev/opendray/pulls/42.diff":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte(sampleDiff))
		case r.URL.Path == "/api/v1/repos/kev/opendray/issues/42/comments":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]giteaComment{{
				ID: 7, Body: "LGTM",
				HTMLURL:   "https://gitea.test/kev/opendray/pulls/42#issuecomment-7",
				CreatedAt: time.Now(),
			}})
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	cfg := newTestConfig("gitea", srv.URL)
	ctx := context.Background()

	prs, err := List(ctx, cfg, StateOpen, 30)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(prs) != 1 || prs[0].Number != 42 || prs[0].State != "open" {
		t.Fatalf("List = %+v", prs)
	}
	if prs[0].URL == "" {
		t.Errorf("URL not populated from html_url")
	}

	pr, err := Detail(ctx, cfg, 42)
	if err != nil {
		t.Fatalf("Detail: %v", err)
	}
	if pr.Body == "" || pr.Number != 42 {
		t.Fatalf("Detail = %+v", pr)
	}

	files, err := Diff(ctx, cfg, 42)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected at least one DiffFile")
	}
	if files[0].Additions == 0 && files[0].Deletions == 0 {
		t.Errorf("line counts not parsed from patch: %+v", files[0])
	}

	cs, err := Comments(ctx, cfg, 42)
	if err != nil {
		t.Fatalf("Comments: %v", err)
	}
	if len(cs) != 1 || cs[0].Body != "LGTM" {
		t.Fatalf("Comments = %+v", cs)
	}
}

func TestGitea_MergedStateNormalised(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(giteaPR{
			Number: 1, State: "closed", Merged: true,
		})
	}))
	defer srv.Close()
	pr, err := Detail(context.Background(), newTestConfig("gitea", srv.URL), 1)
	if err != nil {
		t.Fatal(err)
	}
	if pr.State != "merged" {
		t.Errorf("State = %q, want merged (Gitea Merged=true overrides closed)", pr.State)
	}
}

func TestGitHub_DispatchAndMergedState(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// GitHub keeps our "token <PAT>" default — verify.
		if got := r.Header.Get("Authorization"); got != "token t-test" {
			t.Errorf("Authorization = %q", got)
		}
		// Also verify the versioned API header is pinned.
		if got := r.Header.Get("X-Github-Api-Version"); got != "2022-11-28" {
			t.Errorf("X-GitHub-Api-Version = %q", got)
		}
		switch r.URL.Path {
		case "/repos/kev/opendray/pulls":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]githubPR{{
				Number: 5, Title: "t", State: "closed",
				MergedAt: ptrTime(time.Now()),
			}})
		case "/repos/kev/opendray/pulls/5":
			// Detail path — if client asked for diff, return diff.
			if strings.Contains(r.Header.Get("Accept"), "vnd.github.v3.diff") {
				w.Header().Set("Content-Type", "text/plain")
				_, _ = w.Write([]byte(sampleDiff))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(githubPR{Number: 5, Title: "t", State: "closed"})
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	cfg := newTestConfig("github", srv.URL)

	prs, err := List(context.Background(), cfg, StateAll, 10)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(prs) != 1 || prs[0].State != "merged" {
		t.Fatalf("expected merged state, got %+v", prs)
	}

	files, err := Diff(context.Background(), cfg, 5)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if len(files) == 0 {
		t.Error("Diff returned no files")
	}
}

func TestGitLab_AuthHeaderAndStateMapping(t *testing.T) {
	// GitLab uses PRIVATE-TOKEN; Authorization must be empty so a
	// proxy doesn't accidentally double-handle credentials.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "" {
			t.Errorf("Authorization should be empty for GitLab, got %q", got)
		}
		if got := r.Header.Get("Private-Token"); got != "t-test" {
			t.Errorf("PRIVATE-TOKEN = %q", got)
		}
		// Only /api/v4/projects/kev%2Fopendray/merge_requests is
		// valid: encoded slash.
		if !strings.Contains(r.URL.Path, "/projects/kev%2Fopendray/") &&
			!strings.Contains(r.URL.EscapedPath(), "/projects/kev%2Fopendray/") {
			t.Errorf("project path not URL-encoded: %q", r.URL.Path)
		}
		// "open" from our enum must be translated to GitLab's
		// "opened" before hitting the wire.
		if r.URL.Path == "/api/v4/projects/kev%2Fopendray/merge_requests" &&
			r.URL.Query().Get("state") != "opened" {
			t.Errorf("state param = %q; want opened",
				r.URL.Query().Get("state"))
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/merge_requests"):
			_ = json.NewEncoder(w).Encode([]gitlabMR{{
				IID: 9, Title: "t", State: "opened",
				SourceBranch: "feat/x", TargetBranch: "main",
			}})
		case strings.HasSuffix(r.URL.Path, "/changes"):
			_ = json.NewEncoder(w).Encode(gitlabChangesResponse{
				Changes: []gitlabDiffChange{{
					NewPath: "app/lib/x.dart", OldPath: "app/lib/x.dart",
					Diff: "@@ -1,1 +1,2 @@\n hello\n+world\n",
				}},
			})
		case strings.HasSuffix(r.URL.Path, "/notes"):
			_ = json.NewEncoder(w).Encode([]gitlabNote{
				{ID: 1, Body: "auto", System: true},          // filtered
				{ID: 2, Body: "ship it", System: false,       // kept
					CreatedAt: time.Now()},
			})
		default:
			// Detail
			_ = json.NewEncoder(w).Encode(gitlabMR{
				IID: 9, Title: "t", State: "opened",
				SourceBranch: "feat/x", TargetBranch: "main",
			})
		}
	}))
	defer srv.Close()
	cfg := newTestConfig("gitlab", srv.URL)

	prs, err := List(context.Background(), cfg, StateOpen, 30)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(prs) != 1 || prs[0].State != "open" || prs[0].Number != 9 {
		t.Fatalf("List = %+v (state should be normalised from opened→open)", prs)
	}

	files, err := Diff(context.Background(), cfg, 9)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if len(files) != 1 || files[0].Additions != 1 {
		t.Fatalf("Diff = %+v", files)
	}
	if files[0].OldPath != "" {
		t.Errorf("OldPath = %q, should be '' when same as Path", files[0].OldPath)
	}

	cs, err := Comments(context.Background(), cfg, 9)
	if err != nil {
		t.Fatalf("Comments: %v", err)
	}
	if len(cs) != 1 || cs[0].Body != "ship it" {
		t.Fatalf("Comments filter failed: %+v", cs)
	}
}

func TestGitHub_CoercesWebHostToAPI(t *testing.T) {
	cfg := Config{
		ForgeType: "github",
		BaseURL:   "https://github.com",
		Repo:      "a/b",
	}
	a := &githubAdapter{cfg: cfg, http: httpClient(time.Second)}
	if got := a.apiBase(); got != "https://api.github.com" {
		t.Errorf("apiBase = %q, want https://api.github.com", got)
	}
	enterprise := &githubAdapter{
		cfg: Config{BaseURL: "https://ghe.example.com/api/v3"},
	}
	if got := enterprise.apiBase(); got != "https://ghe.example.com/api/v3" {
		t.Errorf("enterprise apiBase was coerced: %q", got)
	}
}

func TestParseUnifiedDiff_NewAndDeletedAndRename(t *testing.T) {
	blob := `diff --git a/old.txt b/new.txt
similarity index 95%
rename from old.txt
rename to new.txt
index aaa..bbb 100644
--- a/old.txt
+++ b/new.txt
@@ -1,1 +1,2 @@
 hello
+world
diff --git a/added.md b/added.md
new file mode 100644
index 0000000..fff
--- /dev/null
+++ b/added.md
@@ -0,0 +1,1 @@
+brand new
diff --git a/gone.txt b/gone.txt
deleted file mode 100644
index aaa..0000000
--- a/gone.txt
+++ /dev/null
@@ -1,1 +0,0 @@
-bye
`
	files := parseUnifiedDiff(blob)
	if len(files) != 3 {
		t.Fatalf("expected 3 files, got %d", len(files))
	}
	want := []struct {
		path, oldPath, status string
		adds, dels            int
	}{
		{"new.txt", "old.txt", "renamed", 1, 0},
		{"added.md", "", "added", 1, 0},
		{"gone.txt", "", "deleted", 0, 1},
	}
	for i, w := range want {
		got := files[i]
		if got.Path != w.path || got.OldPath != w.oldPath ||
			got.Status != w.status || got.Additions != w.adds ||
			got.Deletions != w.dels {
			t.Errorf("file %d: got %+v, want %+v", i, got, w)
		}
	}
}

func TestPickValidatesPullNumber(t *testing.T) {
	cfg := newTestConfig("gitea", "http://example.invalid")
	if _, err := Detail(context.Background(), cfg, 0); err == nil {
		t.Error("Detail should reject number 0")
	}
	if _, err := Diff(context.Background(), cfg, -1); err == nil {
		t.Error("Diff should reject negative number")
	}
	if _, err := Comments(context.Background(), cfg, 0); err == nil {
		t.Error("Comments should reject number 0")
	}
}

// ── helpers ──────────────────────────────────────────────────────

func ptrTime(t time.Time) *time.Time { return &t }

const sampleDiff = `diff --git a/README.md b/README.md
index aaaaaaa..bbbbbbb 100644
--- a/README.md
+++ b/README.md
@@ -1,3 +1,4 @@
 # OpenDray
-old line
+new line
+added line
`

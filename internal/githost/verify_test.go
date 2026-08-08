package githost

import "testing"

// Each forge answers "who am I" at a different path with a different
// auth scheme, and getting one wrong means the verification reports a
// bad token where the token was fine — worse than not verifying at all.
func TestVerifyEndpoints(t *testing.T) {
	tests := []struct {
		name         string
		host         Host
		repo         string
		wantAuth     string
		wantWhoami   string
		wantRepo     string
		wantContents string
	}{
		{
			name:         "github.com",
			host:         Host{Kind: KindGitHub, Host: "github.com", Token: "t"},
			repo:         "linivek/HomeLabDoc",
			wantAuth:     "Bearer t",
			wantWhoami:   "https://api.github.com/user",
			wantRepo:     "https://api.github.com/repos/linivek/HomeLabDoc",
			wantContents: "https://api.github.com/repos/linivek/HomeLabDoc/contents/",
		},
		{
			name:         "github enterprise serves /api/v3 from its own host",
			host:         Host{Kind: KindGitHub, Host: "git.corp.example", Token: "t"},
			repo:         "team/app",
			wantAuth:     "Bearer t",
			wantWhoami:   "https://git.corp.example/api/v3/user",
			wantRepo:     "https://git.corp.example/api/v3/repos/team/app",
			wantContents: "https://git.corp.example/api/v3/repos/team/app/contents/",
		},
		{
			name:         "gitea uses the token scheme, not Bearer",
			host:         Host{Kind: KindGitea, Host: "tea.example", Token: "t"},
			repo:         "linivek/notes",
			wantAuth:     "token t",
			wantWhoami:   "https://tea.example/api/v1/user",
			wantRepo:     "https://tea.example/api/v1/repos/linivek/notes",
			wantContents: "https://tea.example/api/v1/repos/linivek/notes/contents",
		},
		{
			name:         "gitlab addresses projects by url-encoded path",
			host:         Host{Kind: KindGitLab, Host: "gitlab.com", Token: "t"},
			repo:         "group/sub/app",
			wantAuth:     "Bearer t",
			wantWhoami:   "https://gitlab.com/api/v4/user",
			wantRepo:     "https://gitlab.com/api/v4/projects/group%2Fsub%2Fapp",
			wantContents: "https://gitlab.com/api/v4/projects/group%2Fsub%2Fapp/repository/tree",
		},
		{
			name:       "no repo asked for means no repo call",
			host:       Host{Kind: KindGitHub, Host: "github.com", Token: "t"},
			repo:       "",
			wantAuth:   "Bearer t",
			wantWhoami: "https://api.github.com/user",
			wantRepo:   "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth, accept, whoami, repoURL, contentsURL := verifyEndpoints(tt.host, tt.repo)
			if auth != tt.wantAuth {
				t.Errorf("auth = %q, want %q", auth, tt.wantAuth)
			}
			if whoami != tt.wantWhoami {
				t.Errorf("whoami = %q, want %q", whoami, tt.wantWhoami)
			}
			if repoURL != tt.wantRepo {
				t.Errorf("repo url = %q, want %q", repoURL, tt.wantRepo)
			}
			// The contents URL is the one that proves git will work: a
			// fine-grained token holding only Metadata answers the repo
			// URL and still makes git return 403.
			if contentsURL != tt.wantContents {
				t.Errorf("contents url = %q, want %q", contentsURL, tt.wantContents)
			}
			if accept == "" {
				t.Error("accept header must be set")
			}
		})
	}
}

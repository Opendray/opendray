package githost

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
)

// roundTripFunc lets a test answer the probe without a real forge. The
// probe talks https to a real hostname, so intercepting at the
// transport is cleaner than rewriting the URL for tests — and it keeps
// the URL under test the one production actually builds.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// probeService answers every request with a fixed status. Request
// shape is asserted separately, below.
func probeService(status int) *Service {
	return &Service{
		log: slog.New(slog.DiscardHandler),
		http: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: status,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     make(http.Header),
			}, nil
		})},
	}
}

// The case this probe exists for: a fine-grained token with Contents =
// Read-only. Everything the old verification checked passes; only the
// receive-pack advertisement says no.
func TestProbePush_ForbiddenMeansReadOnly(t *testing.T) {
	s := probeService(http.StatusForbidden)
	ok, err := s.probePush(context.Background(),
		Host{Kind: KindGitHub, Host: "github.com", Token: "t"}, "octo/handbook")
	if err != nil {
		t.Fatalf("a clean 403 is a verdict, not an error: %v", err)
	}
	if ok {
		t.Fatal("403 must report cannot-push")
	}
}

func TestProbePush_OKMeansWritable(t *testing.T) {
	s := probeService(http.StatusOK)
	ok, err := s.probePush(context.Background(),
		Host{Kind: KindGitHub, Host: "github.com", Token: "t"}, "octo/handbook")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("200 must report can-push")
	}
}

// Anything that is not a clean yes/no must surface as unknown. A 500
// from the forge, or a captive-portal 302, says nothing about the
// token — reporting "read-only" there would send the operator to edit
// permissions that were already correct.
func TestProbePush_UnexpectedStatusIsUnknownNotDenied(t *testing.T) {
	for _, status := range []int{http.StatusInternalServerError, http.StatusNotFound,
		http.StatusTooManyRequests} {
		s := probeService(status)
		if _, err := s.probePush(context.Background(),
			Host{Kind: KindGitHub, Host: "github.com", Token: "t"}, "octo/handbook"); err == nil {
			t.Errorf("status %d: want an error (unknown), got a verdict", status)
		}
	}
}

// The probe must speak git's protocol, not the forge's API: basic
// auth, and the receive-pack advertisement path. Get either wrong and
// the answer describes something other than push.
func TestProbePush_SendsGitCredentialsToTheReceivePackEndpoint(t *testing.T) {
	var seen *http.Request
	s := &Service{
		log: slog.New(slog.DiscardHandler),
		http: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			seen = r
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     make(http.Header),
			}, nil
		})},
	}
	if _, err := s.probePush(context.Background(),
		Host{Kind: KindGitLab, Host: "gitlab.com", Token: "secret"}, "group/app"); err != nil {
		t.Fatal(err)
	}
	if seen == nil {
		t.Fatal("no request made")
	}
	if got, want := seen.URL.String(),
		"https://gitlab.com/group/app.git/info/refs?service=git-receive-pack"; got != want {
		t.Errorf("url = %q, want %q", got, want)
	}
	if seen.Method != http.MethodGet {
		t.Errorf("method = %s — the probe must not mutate anything", seen.Method)
	}
	user, pass, ok := seen.BasicAuth()
	if !ok {
		t.Fatal("smart HTTP is basic auth; the API's Bearer scheme will not do")
	}
	if pass != "secret" {
		t.Errorf("password = %q, want the token", pass)
	}
	// GitLab is the forge that actually cares about the username.
	if user != "oauth2" {
		t.Errorf("username = %q, want oauth2 for GitLab", user)
	}
}

func TestReceivePackURL(t *testing.T) {
	tests := []struct {
		name, host, repo, want string
	}{
		{
			name: "github", host: "github.com", repo: "octo/handbook",
			want: "https://github.com/octo/handbook.git/info/refs?service=git-receive-pack",
		},
		{
			name: "self-hosted gitea with a port", host: "tea.example:3000", repo: "octo/notes",
			want: "https://tea.example:3000/octo/notes.git/info/refs?service=git-receive-pack",
		},
		{
			name: "gitlab nested groups keep their full path", host: "gitlab.com", repo: "group/sub/app",
			want: "https://gitlab.com/group/sub/app.git/info/refs?service=git-receive-pack",
		},
		{
			name: "stray slashes are trimmed", host: "github.com", repo: "/octo/handbook/",
			want: "https://github.com/octo/handbook.git/info/refs?service=git-receive-pack",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := receivePackURL(tt.host, tt.repo); got != tt.want {
				t.Fatalf("got  %q\nwant %q", got, tt.want)
			}
		})
	}
}

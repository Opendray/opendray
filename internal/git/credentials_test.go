package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

type fakeResolver struct {
	username, token string
	ok              bool
	gotHost         string
	gotOwner        string
}

func (f *fakeResolver) CredentialFor(
	_ context.Context, host, owner string,
) (string, string, bool) {
	f.gotHost, f.gotOwner = host, owner
	return f.username, f.token, f.ok
}

// initRepo makes a repo whose origin is remoteURL.
func initRepo(t *testing.T, remoteURL string) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"remote", "add", "origin", remoteURL},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("git unavailable or failed (%v): %s", err, out)
		}
	}
	return dir
}

func hasWipe(args []string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == "-c" && args[i+1] == "credential.helper=" {
			return true
		}
	}
	return false
}

func hasInlineHelper(args []string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == "-c" && len(args[i+1]) > len("credential.helper=") &&
			args[i+1][:len("credential.helper=!")] == "credential.helper=!" {
			return true
		}
	}
	return false
}

// The point of this file is that opendray's configured credential — and
// nothing the host happens to remember — answers an HTTPS prompt. The
// no-credential case matters most: that is where a stale osxkeychain
// entry used to answer silently, producing a 403 about a token the
// operator never configured here.
func TestAuthArgs(t *testing.T) {
	tests := []struct {
		name        string
		remote      string
		resolverOK  bool
		wantWipe    bool
		wantHelper  bool
		wantHost    string
		wantOwner   string
		wantUntouch bool
	}{
		{
			name: "https with a registered credential", remote: "https://github.com/octo-org/handbook.git",
			resolverOK: true, wantWipe: true, wantHelper: true,
			wantHost: "github.com", wantOwner: "octo-org",
		},
		{
			name: "https with NO credential still blanks inherited helpers",
			// Without the wipe, git would fall through to osxkeychain /
			// ~/.netrc and authenticate as somebody else.
			remote:     "https://github.com/octo-org/handbook.git",
			resolverOK: false, wantWipe: true, wantHelper: false,
			wantHost: "github.com", wantOwner: "octo-org",
		},
		{
			name:       "owner is passed through so the right row is chosen",
			remote:     "https://github.com/Opendray/opendray.git",
			resolverOK: true, wantWipe: true, wantHelper: true,
			wantHost: "github.com", wantOwner: "Opendray",
		},
		{
			name:       "self-hosted with a port",
			remote:     "https://gitea.example.com:3000/octo-org/notes.git",
			resolverOK: true, wantWipe: true, wantHelper: true,
			wantHost: "gitea.example.com:3000", wantOwner: "octo-org",
		},
		{
			name:       "ssh remote is left to the agent",
			remote:     "git@github.com:Opendray/opendray.git",
			resolverOK: true, wantUntouch: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := initRepo(t, tt.remote)
			r := &fakeResolver{username: "opendray", token: "tok", ok: tt.resolverOK}
			got := authArgs(context.Background(), r, dir, "push", "origin", "main")

			if tt.wantUntouch {
				if !slices.Equal(got, []string{"push", "origin", "main"}) {
					t.Fatalf("ssh remote args were modified: %v", got)
				}
				return
			}
			if hasWipe(got) != tt.wantWipe {
				t.Fatalf("wipe present = %v, want %v (args %v)", hasWipe(got), tt.wantWipe, got)
			}
			if hasInlineHelper(got) != tt.wantHelper {
				t.Fatalf("inline helper present = %v, want %v", hasInlineHelper(got), tt.wantHelper)
			}
			if r.gotHost != tt.wantHost || r.gotOwner != tt.wantOwner {
				t.Fatalf("resolved for (%q, %q), want (%q, %q)",
					r.gotHost, r.gotOwner, tt.wantHost, tt.wantOwner)
			}
			// The original command must still be there, after the flags.
			if got[len(got)-3] != "push" {
				t.Fatalf("git args mangled: %v", got)
			}
		})
	}
}

func TestAuthArgs_NoResolverOrNoRepo(t *testing.T) {
	base := []string{"push", "origin", "main"}

	// No resolver wired: previous behaviour, untouched.
	dir := initRepo(t, "https://github.com/octo-org/x.git")
	if got := authArgs(context.Background(), nil, dir, base...); !slices.Equal(got, base) {
		t.Fatalf("nil resolver modified args: %v", got)
	}

	// Not a repo / no origin: let git produce its own error.
	empty := filepath.Join(t.TempDir(), "nope")
	r := &fakeResolver{ok: true, token: "tok"}
	if got := authArgs(context.Background(), r, empty, base...); !slices.Equal(got, base) {
		t.Fatalf("missing repo modified args: %v", got)
	}
}

// A token must not be able to break out of the inline helper's quoting.
func TestShellEscape(t *testing.T) {
	if got := shellEscape("a'b"); got != `a'\''b` {
		t.Fatalf("shellEscape(%q) = %q", "a'b", got)
	}
}

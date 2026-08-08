package git

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"strings"
)

// Git operations that touch the network must authenticate with the
// credential the operator configured in opendray, and with nothing
// else.
//
// Left alone, git answers an HTTPS prompt from whatever the host
// offers: an osxkeychain entry (Xcode ships that helper enabled), a
// ~/.netrc, a credential store from some unrelated tool. Those are
// invisible from inside opendray and outlive the reason they were
// created, so a stale one keeps working long enough to be trusted and
// then fails with an error that describes a token nobody remembers
// configuring. The operator changes the token in opendray, nothing
// improves, and the evidence points nowhere.
//
// So: for HTTPS remotes we blank the inherited helper list first, then
// add ours. Blanking happens EVEN WHEN opendray has no credential for
// the host — an operation that fails saying "no credentials" is worth
// more than one that quietly succeeds as an identity the operator
// never chose, and is the only way the Git-hosts page can be trusted
// as the single place that decides.
//
// SSH remotes are untouched: they authenticate through the agent and
// ~/.ssh, which is a deliberate, visible configuration rather than an
// ambient cache.

// TokenResolver hands back the credential registered for a host+owner.
// Implemented by *githost.Service; an interface so this package does
// not depend on it (and so tests can supply their own).
type TokenResolver interface {
	CredentialFor(ctx context.Context, host, owner string) (username, token string, ok bool)
}

// authArgs returns the `-c` flags to prepend to a git invocation in
// dir so that opendray's configured credential — and only it — answers
// an HTTPS auth prompt.
//
// Returns the args unchanged for SSH remotes, for repos with no
// origin, and when there is no resolver.
func authArgs(
	ctx context.Context,
	resolver TokenResolver,
	dir string,
	gitArgs ...string,
) []string {
	if resolver == nil {
		return gitArgs
	}
	rawURL, err := runQuiet(ctx, dir, "remote", "get-url", "origin")
	if err != nil {
		return gitArgs // no origin, or not a repo — let git say so
	}
	scheme, host, owner := parseRemoteURL(strings.TrimSpace(rawURL))
	if scheme != "https" && scheme != "http" {
		return gitArgs // ssh / git: the agent's job, deliberately
	}
	if host == "" {
		return gitArgs
	}

	// Blank the inherited helpers whether or not we have a token —
	// see the note above; a silent fallback is the failure this
	// function exists to prevent.
	out := []string{"-c", "credential.helper="}
	username, token, ok := resolver.CredentialFor(ctx, host, owner)
	if ok && token != "" {
		// Inline helper: an embedded shell function that prints the
		// answer git asks for. Keeps the token off disk entirely.
		out = append(out, "-c", "credential.helper="+fmt.Sprintf(
			`!f() { printf 'username=%%s\npassword=%%s\n' '%s' '%s'; }; f`,
			shellEscape(username), shellEscape(token),
		))
	}
	return append(out, gitArgs...)
}

// parseRemoteURL splits a remote into scheme, host and owner (the
// first path segment — what selects among several credentials on one
// host).
func parseRemoteURL(raw string) (scheme, host, owner string) {
	if raw == "" {
		return "", "", ""
	}
	// scp-like: [user@]host:owner/repo.git
	if !strings.Contains(raw, "://") && strings.Contains(raw, ":") {
		rest := raw
		if i := strings.Index(rest, "@"); i >= 0 {
			rest = rest[i+1:]
		}
		if i := strings.Index(rest, ":"); i >= 0 {
			return "ssh", rest[:i], firstSegment(rest[i+1:])
		}
		return "ssh", "", ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "", "", ""
	}
	return u.Scheme, u.Host, firstSegment(u.Path)
}

func firstSegment(path string) string {
	path = strings.TrimPrefix(path, "/")
	if i := strings.Index(path, "/"); i >= 0 {
		return path[:i]
	}
	return path
}

// shellEscape wraps single quotes so a token can never break out of
// the inline helper's quoting.
func shellEscape(s string) string {
	return strings.ReplaceAll(s, `'`, `'\''`)
}

// runQuiet executes git in dir and returns stdout, discarding stderr.
func runQuiet(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = filteredEnv()
	out, err := cmd.Output()
	return string(out), err
}

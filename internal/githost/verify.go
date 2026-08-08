package githost

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// A stored token is a claim: the operator says "this credential belongs
// to <owner> and can reach its repos". Nothing checked that claim, and
// the forges make the mistake easy to hide.
//
// A GitHub fine-grained token is the worst offender. Its two settings —
// which repositories it covers, and which permissions it holds — are
// separate sections of the same form, and permissions default to NONE.
// Grant "All repositories", skip the permissions block, and you get a
// token that authenticates perfectly (`GET /user` → 200) yet cannot
// read a single repo. Git then answers a plain fetch with
//
//	remote: Write access to repository not granted ... 403
//
// which names the wrong permission, on the wrong operation, and never
// mentions the token's settings. From inside opendray it is
// indistinguishable from a gateway bug — and it cost this project three
// rounds of debugging before the token was suspected.
//
// Verify closes that loop: it asks the forge who the token actually
// belongs to, and whether it can reach a specific repository. One call
// turns "why doesn't it work" into a sentence.

// VerifyResult reports what the forge says about a stored credential.
type VerifyResult struct {
	// Login is the account the token authenticates as, per the forge.
	Login string `json:"login,omitempty"`
	// OwnerMatches is false when Login differs from the row's Owner —
	// a mislabelled credential, which resolves for repos it has no
	// business touching.
	OwnerMatches bool `json:"owner_matches"`
	// Reachable reports whether Repo (when given) can be read.
	Reachable bool   `json:"reachable,omitempty"`
	Repo      string `json:"repo,omitempty"`
	// Hint explains a failure in the forge's own terms, since the raw
	// message usually points somewhere unhelpful.
	Hint  string `json:"hint,omitempty"`
	Error string `json:"error,omitempty"`
}

// Verify authenticates the stored token against its forge and, when
// repo ("owner/name") is supplied, checks that it can be read.
func (s *Service) Verify(ctx context.Context, id, repo string) (VerifyResult, error) {
	h, err := s.getWithToken(ctx, id)
	if err != nil {
		return VerifyResult{}, err
	}
	if h.TokenLocked {
		return VerifyResult{
			Error: "token is encrypted and cannot be decrypted right now",
			Hint:  "Re-enter the token, or arm backups with the original key.",
		}, nil
	}
	if h.Token == "" {
		return VerifyResult{Error: "no token stored"}, nil
	}

	auth, accept, whoami, repoURL := verifyEndpoints(h, repo)
	res := VerifyResult{Repo: repo}

	body, err := s.do(ctx, "GET", whoami, auth, accept, nil)
	if err != nil {
		res.Error = err.Error()
		res.Hint = "The forge rejected the token itself — it is wrong, revoked or expired."
		return res, nil
	}
	var who struct {
		Login    string `json:"login"`
		Username string `json:"username"` // Gitea/GitLab spell it differently
	}
	_ = json.Unmarshal(body, &who)
	res.Login = who.Login
	if res.Login == "" {
		res.Login = who.Username
	}
	// An empty Owner is the host-wide credential and matches by design.
	res.OwnerMatches = h.Owner == "" ||
		strings.EqualFold(h.Owner, res.Login)
	if !res.OwnerMatches {
		res.Hint = fmt.Sprintf(
			"This entry is scoped to %q but the token belongs to %q, so it "+
				"will be used for repos it may not cover.", h.Owner, res.Login)
	}

	if repoURL == "" {
		return res, nil
	}
	if _, err := s.do(ctx, "GET", repoURL, auth, accept, nil); err != nil {
		res.Reachable = false
		// The token authenticated but cannot see the repo: on GitHub
		// that is almost always a fine-grained token whose permissions
		// block was left at "No access", or whose repository list
		// excludes this one.
		res.Hint = fmt.Sprintf(
			"The token is valid (authenticated as %q) but cannot read %s. "+
				"For a fine-grained token, check BOTH sections: Repository "+
				"access must include this repo, AND Permissions → Contents "+
				"must be Read and write. Permissions default to none, which "+
				"is why a token can look correct and still fail.",
			res.Login, repo)
		return res, nil
	}
	res.Reachable = true
	return res, nil
}

// verifyEndpoints returns the auth header, Accept header, whoami URL
// and repo URL for a host's forge kind.
func verifyEndpoints(h Host, repo string) (auth, accept, whoami, repoURL string) {
	switch h.Kind {
	case KindGitea:
		auth = "token " + h.Token
		accept = "application/json"
		whoami = fmt.Sprintf("https://%s/api/v1/user", h.Host)
		if repo != "" {
			repoURL = fmt.Sprintf("https://%s/api/v1/repos/%s", h.Host, repo)
		}
	case KindGitLab:
		auth = "Bearer " + h.Token
		accept = "application/json"
		whoami = fmt.Sprintf("https://%s/api/v4/user", h.Host)
		if repo != "" {
			// GitLab addresses projects by URL-encoded full path.
			repoURL = fmt.Sprintf("https://%s/api/v4/projects/%s",
				h.Host, strings.ReplaceAll(repo, "/", "%2F"))
		}
	default: // GitHub / GitHub Enterprise
		auth = "Bearer " + h.Token
		accept = "application/vnd.github+json"
		base := githubAPIBase(h.Host)
		whoami = base + "/user"
		if repo != "" {
			repoURL = base + "/repos/" + repo
		}
	}
	return auth, accept, whoami, repoURL
}

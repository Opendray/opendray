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
//
// Reaching a repository takes TWO calls, because one is not enough to
// tell the truth. `GET /repos/{o}/{r}` answers for a fine-grained token
// holding only Metadata — which is mandatory and always present — so a
// single check reports "can read" for a token git will refuse. The
// contents endpoint is the one that needs Contents permission, the same
// permission git transport needs. Checking metadata first is still
// worth it: metadata-denied means the repo is outside the token's
// repository list, contents-denied means the list is right and the
// permission is missing. Those are different fixes.

// VerifyResult reports what the forge says about a stored credential.
type VerifyResult struct {
	// Login is the account the token authenticates as, per the forge.
	Login string `json:"login,omitempty"`
	// OwnerMatches is false when Login differs from the row's Owner.
	// NOT an error on its own: an organisation entry is reached with a
	// member's token, so github.com/Opendray legitimately authenticates
	// as the person who owns that org. Informational only.
	OwnerMatches bool `json:"owner_matches"`
	// Reachable reports whether Repo's CONTENTS can be read — the
	// permission git transport needs, not merely repo metadata.
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

	auth, accept, whoami, repoURL, contentsURL := verifyEndpoints(h, repo)
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
	// A login that differs from the entry's owner is NORMAL, not a
	// mistake: an organisation's repos are reached with a token issued
	// by a member — someone who owns the org still authenticates as
	// themselves. Treating that as "mislabelled" (as this did at first)
	// flags the correct configuration as broken. So it is reported, not
	// judged; whether the credential actually works is what the repo
	// check below answers.
	res.OwnerMatches = h.Owner == "" || strings.EqualFold(h.Owner, res.Login)

	if repoURL == "" {
		if !res.OwnerMatches {
			res.Hint = fmt.Sprintf(
				"Token belongs to %q while this entry is scoped to %q. That is "+
					"expected when %q is an organisation %q belongs to. Name a "+
					"repo above to confirm the credential actually reaches it.",
				res.Login, h.Owner, h.Owner, res.Login)
		}
		return res, nil
	}

	// Step 1 — metadata. Failing here means the repo is not in the
	// token's repository list at all (or does not exist).
	if _, err := s.do(ctx, "GET", repoURL, auth, accept, nil); err != nil {
		res.Reachable = false
		res.Hint = fmt.Sprintf(
			"The token is valid (authenticated as %q) but cannot even see %s. "+
				"Its Repository access does not include that repo — add it, or "+
				"switch the token to all repositories.", res.Login, repo)
		return res, nil
	}

	// Step 2 — contents. This is the permission git transport needs, and
	// the one that is missing whenever a token "looks right" yet every
	// clone, pull and push comes back 403.
	if _, err := s.do(ctx, "GET", contentsURL, auth, accept, nil); err != nil {
		res.Reachable = false
		res.Hint = fmt.Sprintf(
			"The token can SEE %s but not read its contents — so git will "+
				"refuse every pull and push, reporting \"Write access to "+
				"repository not granted\" even for a read. Fix: Permissions → "+
				"Repository permissions → Contents = Read and write. It "+
				"defaults to no access, which is why the repository list "+
				"looking correct is not enough.", repo)
		return res, nil
	}
	res.Reachable = true
	return res, nil
}

// verifyEndpoints returns the auth header, Accept header, whoami URL,
// repo-metadata URL and repo-contents URL for a host's forge kind.
func verifyEndpoints(
	h Host, repo string,
) (auth, accept, whoami, repoURL, contentsURL string) {
	switch h.Kind {
	case KindGitea:
		auth = "token " + h.Token
		accept = "application/json"
		whoami = fmt.Sprintf("https://%s/api/v1/user", h.Host)
		if repo != "" {
			repoURL = fmt.Sprintf("https://%s/api/v1/repos/%s", h.Host, repo)
			contentsURL = repoURL + "/contents"
		}
	case KindGitLab:
		auth = "Bearer " + h.Token
		accept = "application/json"
		whoami = fmt.Sprintf("https://%s/api/v4/user", h.Host)
		if repo != "" {
			// GitLab addresses projects by URL-encoded full path.
			repoURL = fmt.Sprintf("https://%s/api/v4/projects/%s",
				h.Host, strings.ReplaceAll(repo, "/", "%2F"))
			contentsURL = repoURL + "/repository/tree"
		}
	default: // GitHub / GitHub Enterprise
		auth = "Bearer " + h.Token
		accept = "application/vnd.github+json"
		base := githubAPIBase(h.Host)
		whoami = base + "/user"
		if repo != "" {
			repoURL = base + "/repos/" + repo
			contentsURL = repoURL + "/contents/"
		}
	}
	return auth, accept, whoami, repoURL, contentsURL
}

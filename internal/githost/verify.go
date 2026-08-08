package githost

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
// Reaching a repository takes THREE calls, because fewer is not enough
// to tell the truth.
//
// `GET /repos/{o}/{r}` answers for a fine-grained token holding only
// Metadata — which is mandatory and always present — so a single check
// reports "can read" for a token git will refuse. The contents endpoint
// is the one that needs Contents permission. Checking metadata first is
// still worth it: metadata-denied means the repo is outside the token's
// repository list, contents-denied means the list is right and the
// permission is missing. Those are different fixes.
//
// Contents itself has THREE levels — No access, Read-only, Read and
// write — and the first two checks pass identically for a Read-only
// token. That token pulls fine and then fails every push with
//
//	remote: Write access to repository not granted ... 403
//
// which is the same message a token with no Contents at all produces
// on a plain fetch. So a green verification followed by a red push is
// exactly the outcome this feature exists to prevent, and it happened:
// the first version of this file green-lit a Read-only token.
//
// Hence the third call, and note what it is NOT. It is not an API
// endpoint whose permissions might diverge from the git transport's —
// it is the literal first request `git push` makes, the receive-pack
// advertisement, asked with the same credential over the same
// protocol. If it answers 403, push will answer 403. It is also a
// plain GET: it lists refs and changes nothing. And because it is git
// rather than a forge API, one code path covers every forge.
//
// The rule this file keeps learning: verify the operation, not a
// proxy for it. A false green is worse than no check, because it sends
// the operator looking somewhere else.

// VerifyResult reports what the forge says about a stored credential.
type VerifyResult struct {
	// Login is the account the token authenticates as, per the forge.
	Login string `json:"login,omitempty"`
	// OwnerMatches is false when Login differs from the row's Owner.
	// NOT an error on its own: an organisation entry is reached with a
	// member's token, so an organisation entry legitimately authenticates
	// as the person who owns that org. Informational only.
	OwnerMatches bool `json:"owner_matches"`
	// Reachable reports whether Repo's CONTENTS can be read — the
	// permission git transport needs, not merely repo metadata.
	Reachable bool   `json:"reachable,omitempty"`
	Repo      string `json:"repo,omitempty"`
	// CanPush reports whether the credential may push to Repo, probed
	// against git's own receive-pack advertisement.
	//
	// Tri-state on purpose. nil means "not established" — no repo was
	// named, or the forge answered something we cannot read a verdict
	// from — and must never be rendered as "read-only". Claiming a
	// token cannot push when it can would send the operator editing
	// permissions that were already correct, which is the same class
	// of mistake as the false green, pointed the other way.
	CanPush *bool `json:"can_push,omitempty"`
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

	// Step 3 — push. Reads work at this point; a Read-only token is
	// indistinguishable from a read-write one until something asks git
	// for write access, so ask.
	switch canPush, err := s.probePush(ctx, h, repo); {
	case err != nil:
		// Unknown, not "no". Leave CanPush nil and say nothing — a
		// guess here is worse than silence.
		s.log.Debug("push probe inconclusive",
			"host", h.Host, "repo", repo, "err", err)
	case canPush:
		res.CanPush = &canPush
	default:
		res.CanPush = &canPush
		res.Hint = fmt.Sprintf(
			"The token can READ %s but not write to it. Pulls will work and "+
				"every push will fail with \"Write access to repository not "+
				"granted\". Fix: Permissions → Repository permissions → "+
				"Contents = Read and write (it is currently Read-only).", repo)
	}
	return res, nil
}

// probePush asks git — not the forge's API — whether this credential
// may push. The receive-pack advertisement is the first request of a
// real `git push`, so its answer is the answer, and it mutates nothing.
//
// Returns an error rather than a verdict when the forge says something
// that is not a clear yes or no. Callers must treat that as unknown.
func (s *Service) probePush(ctx context.Context, h Host, repo string) (bool, error) {
	u := receivePackURL(h.Host, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return false, err
	}
	// Smart HTTP is basic auth, not the API's Bearer/token schemes —
	// this is git's protocol, and every forge speaks it the same way.
	req.SetBasicAuth(credentialUsername(h.Kind), h.Token)
	req.Header.Set("User-Agent", "git/2.0 (opendray-inspector)")

	resp, err := s.http.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 8<<10))

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusUnauthorized, http.StatusForbidden:
		return false, nil
	default:
		return false, fmt.Errorf("receive-pack advertisement: unexpected status %d",
			resp.StatusCode)
	}
}

// receivePackURL builds the smart-HTTP endpoint `git push` opens
// first. Same shape on GitHub, Gitea and GitLab — it is git's wire
// protocol, not a forge API, which is why one probe covers them all.
func receivePackURL(host, repo string) string {
	return fmt.Sprintf("https://%s/%s.git/info/refs?service=git-receive-pack",
		host, strings.Trim(repo, "/"))
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

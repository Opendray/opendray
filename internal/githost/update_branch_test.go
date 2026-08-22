package githost

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// A PR whose branch is behind its base cannot be merged when the repo
// requires branches to be up to date — GitHub answers the merge with a
// 405 "required status checks are expected". The UI can only offer a way
// out if it knows the state, so mergeable_state has to survive parsing.
func TestGitHubPRResponse_CarriesMergeableState(t *testing.T) {
	const raw = `{
	  "number": 517,
	  "title": "fix: something",
	  "state": "open",
	  "mergeable": true,
	  "mergeable_state": "behind",
	  "html_url": "https://github.com/o/r/pull/517",
	  "user": {"login": "linivek"},
	  "head": {"ref": "fix/x"},
	  "base": {"ref": "main"}
	}`
	var p githubPRResponse
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	pr := p.toPullRequest()
	if pr.MergeableState != "behind" {
		t.Errorf("MergeableState = %q, want behind", pr.MergeableState)
	}
	if !pr.BehindBase() {
		t.Error("BehindBase() = false for mergeable_state=behind")
	}
}

func TestPullRequest_BehindBase(t *testing.T) {
	for state, want := range map[string]bool{
		"behind":   true,
		"clean":    false,
		"blocked":  false, // failing/pending checks, not staleness
		"dirty":    false, // real conflicts — updating won't help
		"unstable": false,
		"":         false, // unknown (list endpoints omit it)
	} {
		if got := (PullRequest{MergeableState: state}).BehindBase(); got != want {
			t.Errorf("BehindBase(%q) = %v, want %v", state, got, want)
		}
	}
}

// Only GitHub is wired up for now. Other hosts must say so plainly
// rather than fail with something the operator has to decode.
func TestUpdateBranch_UnsupportedHostIsExplicit(t *testing.T) {
	svc := &Service{}
	err := svc.updateBranchForKind(t.Context(), Host{Kind: KindGitea}, Remote{}, 1)
	if err == nil {
		t.Fatal("expected an error for a non-GitHub host")
	}
	// Names the host it IS on and the one it would work on, so the
	// operator knows both why it failed and what to do instead.
	lower := strings.ToLower(err.Error())
	for _, want := range []string{"gitea", "github", "locally"} {
		if !strings.Contains(lower, want) {
			t.Errorf("error missing %q; got %q", want, err)
		}
	}
}

// A merge refused because the branch is stale used to surface as the
// host's raw JSON — "Repository rule violations found\n\n3 of 3 required
// status checks are expected" — which points at checks and says nothing
// about what to actually do. The operator sees it verbatim.
func TestExplainMergeFailure(t *testing.T) {
	raw := errors.New(`github: 405: {"message":"Repository rule violations found\n\n3 of 3 required status checks are expected.","status":"405"}`)

	out := explainMergeFailure(raw, PullRequest{MergeableState: "behind"}).Error()
	for _, want := range []string{"behind", "update"} {
		if !strings.Contains(strings.ToLower(out), want) {
			t.Errorf("a BEHIND merge failure should say so and name the way out; got %q", out)
		}
	}

	// Anything else is passed through — inventing an explanation for a
	// failure we did not diagnose would be worse than the raw text.
	other := explainMergeFailure(raw, PullRequest{MergeableState: "dirty"})
	if other.Error() != raw.Error() {
		t.Errorf("non-BEHIND failure should pass through unchanged; got %q", other)
	}
	if explainMergeFailure(nil, PullRequest{MergeableState: "behind"}) != nil {
		t.Error("nil error must stay nil")
	}
}

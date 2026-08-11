package githost

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

// Merging the second of two PRs fails on repos that require branches to
// be up to date: once the first one lands, the second is BEHIND, and the
// host refuses with a rule violation about missing status checks — which
// reads like a checks problem rather than a staleness one.
//
// The fix is to merge the base branch into the PR branch so the checks
// re-run on the new base. GitHub exposes it as "Update branch"; without
// an equivalent here an operator working only through opendray has no
// way out of that state.

// explainMergeFailure prefixes a merge error with the reason when we can
// name it from the PR's own state.
//
// A stale branch is refused by GitHub with "Repository rule violations
// found — N of N required status checks are expected", which reads as a
// checks problem and offers no next step. The operator sees that text
// verbatim, so the diagnosis belongs here rather than in each client.
//
// Only the case we can actually identify is rewritten; everything else
// passes through, because an invented explanation is worse than raw
// host output.
func explainMergeFailure(err error, pr PullRequest) error {
	if err == nil || !pr.BehindBase() {
		return err
	}
	return fmt.Errorf(
		"this branch is behind %s, and the repository requires branches to be up to date before merging — update the branch first, then merge (%w)",
		orDefault(pr.Base, "its base branch"), err)
}

func orDefault(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

// UpdateBranchRequest asks the host to merge a PR's base into its head.
type UpdateBranchRequest struct {
	Dir    string `json:"dir"`
	Number int    `json:"number"`
}

// UpdatePullRequestBranch brings a PR's branch up to date with its base
// and returns the refreshed PR.
//
// GitHub only for now: Gitea and GitLab have their own shapes for this
// and are refused explicitly rather than half-supported.
func (s *Service) UpdatePullRequestBranch(ctx context.Context, req UpdateBranchRequest) (PullRequest, error) {
	if req.Number <= 0 {
		return PullRequest{}, errors.New("number required")
	}
	rem, hostRow, err := s.resolveHost(ctx, req.Dir)
	if err != nil {
		return PullRequest{}, err
	}
	if err := s.updateBranchForKind(ctx, hostRow, rem, req.Number); err != nil {
		return PullRequest{}, err
	}
	// Re-fetch so the caller sees the new mergeable_state. The update
	// itself succeeded, so a failed refetch is not worth failing on —
	// the caller re-polls anyway.
	pr, ferr := s.GetPullRequest(ctx, req.Dir, req.Number)
	if ferr != nil {
		return PullRequest{Number: req.Number}, nil
	}
	return pr, nil
}

// updateBranchForKind dispatches to the host implementation.
func (s *Service) updateBranchForKind(ctx context.Context, h Host, rem Remote, number int) error {
	switch h.Kind {
	case KindGitHub:
		return s.updateGitHubPRBranch(ctx, h, rem, number)
	default:
		return fmt.Errorf(
			"updating a pull request branch is only supported on GitHub — this repository is on %s; merge the base branch in locally and push",
			h.Kind)
	}
}

// updateGitHubPRBranch calls PUT /repos/{o}/{r}/pulls/{n}/update-branch,
// which merges the base into the head branch.
//
// A 422 here means the branch cannot be updated this way — most often
// because it is not actually behind, or the merge would conflict. Both
// need the operator, so the host's message is passed through.
func (s *Service) updateGitHubPRBranch(ctx context.Context, h Host, rem Remote, number int) error {
	u := fmt.Sprintf("%s/repos/%s/%s/pulls/%d/update-branch",
		githubAPIBase(h.Host), rem.Owner, rem.Repo, number)
	// expected_head_sha is deliberately omitted: it guards against
	// updating a branch that moved since the UI last looked, but the
	// operator's intent here is "make it current", and a stale SHA would
	// just fail a request they would immediately retry.
	_, err := s.do(ctx, http.MethodPut, u, "Bearer "+h.Token, "application/vnd.github+json", map[string]any{})
	return err
}

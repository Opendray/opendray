package api

import "context"

// Forge is a source-control hosting integration — GitHub, GitLab, Gitea,
// Bitbucket, etc. A forge plugin authenticates against the service and
// exposes the host's source-control surface (repo browser, PR list,
// webhook handling) backed by that service.
//
// The v1 surface is intentionally narrow — just enough for the gateway's
// existing source-control endpoints to be served from a registered
// forge instead of a hardcoded module. Phase 3 will extend it as the
// existing gateway/sourcecontrol/ functionality is migrated in.
type Forge interface {
	// ID is the stable registry key (e.g. "github", "gitea-self").
	// MUST equal the id declared in manifest.contributes.forges[].id.
	ID() string

	// ListRepositories returns repos visible to the given account.
	// accountID is opaque to the host; the plugin maps it to its own
	// account-credential storage.
	ListRepositories(ctx context.Context, accountID string) ([]ForgeRepository, error)

	// ListPullRequests returns pull/merge requests in the given repo.
	// state is one of "open" | "closed" | "merged" | "all".
	ListPullRequests(ctx context.Context, accountID, repo, state string) ([]ForgePullRequest, error)
}

// ForgeRepository is the neutral repo descriptor.
type ForgeRepository struct {
	ID            string `json:"id"`
	FullName      string `json:"fullName"` // e.g. "owner/repo"
	Description   string `json:"description,omitempty"`
	Private       bool   `json:"private"`
	DefaultBranch string `json:"defaultBranch,omitempty"`
	HTMLURL       string `json:"htmlUrl,omitempty"`
	CloneURL      string `json:"cloneUrl,omitempty"`
}

// ForgePullRequest is the neutral PR/MR descriptor.
type ForgePullRequest struct {
	ID      string `json:"id"`
	Number  int    `json:"number"`
	Title   string `json:"title"`
	Author  string `json:"author"`
	State   string `json:"state"` // "open", "closed", "merged"
	HTMLURL string `json:"htmlUrl,omitempty"`
	Head    string `json:"head,omitempty"` // branch name
	Base    string `json:"base,omitempty"`
}

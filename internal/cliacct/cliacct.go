// Package cliacct manages multi-account binding for the Claude Code
// CLI. Mirrors v1's design: account rows hold metadata only (name,
// display name, config_dir, token_path), while the actual OAuth token
// lives chmod 600 at token_path on the gateway host. The host tool
// `claude-acc` (or the Import-local endpoint) is the source of truth
// for token files; this package only reads them at session-spawn
// time and injects env vars (CLAUDE_CODE_OAUTH_TOKEN +
// CLAUDE_CONFIG_DIR) into the spawned PTY.
package cliacct

import (
	"errors"
	"time"
)

// Account describes one Claude Code account known to the gateway. The
// OAuth token is intentionally NOT stored in the database — TokenPath
// points at the file on disk that holds it.
type Account struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	ConfigDir   string    `json:"config_dir"`
	TokenPath   string    `json:"token_path"`
	Description string    `json:"description"`
	Enabled     bool      `json:"enabled"`
	TokenFilled bool      `json:"token_filled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Derived fields below are computed on each read, never persisted.
	// They give the panel enough signal to answer "which account is
	// safe to use right now?" without needing Phase-2 usage probes.
	// JSON omitempty so older clients keep working.

	// SubscriptionType comes from <configDir>/.credentials.json:
	// claudeAiOauth.subscriptionType — e.g. "max", "pro", "free".
	SubscriptionType string `json:"subscription_type,omitempty"`
	// RateLimitTier comes from the same file:
	// claudeAiOauth.rateLimitTier — e.g. "default_claude_max_5x".
	RateLimitTier string `json:"rate_limit_tier,omitempty"`
	// LastUsedAt is MAX(sessions.started_at) where claude_account_id
	// matches; nil when this account has never been pinned to a
	// session. Drives the "last used …" chip.
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	// ActiveSessions counts sessions currently in a non-terminal state
	// pinned to this account. Powers BOTH the "N sessions" chip and the
	// least-loaded auto-assign heuristic. Always emitted (never omitted)
	// so the UI can render "0 sessions" without special-casing.
	ActiveSessions int `json:"active_sessions"`
	// OAuthEmail is the Anthropic account currently logged in at this
	// account's configDir, read from <configDir>/.claude.json (or
	// <home>/.claude.json for the synthetic 'default'). Empty when the
	// metadata file is missing or the account is unauthenticated.
	OAuthEmail string `json:"oauth_email,omitempty"`
	// PreviousEmail + IdentityDrift fire when the on-disk OAuthEmail
	// differs from the one we first recorded for this account id —
	// catching the dangerous case where the operator runs
	// `claude login` (no CLAUDE_CONFIG_DIR) and silently swaps the
	// underlying identity of the default account. Cleared when the
	// operator accepts the new identity via the accept-identity API.
	PreviousEmail string `json:"previous_email,omitempty"`
	IdentityDrift bool   `json:"identity_drift,omitempty"`
}

// CreateRequest is the body for POST /api/v1/claude-accounts.
//
// Token is optional; when omitted the row is created in "empty" state
// and the operator is expected to populate TokenPath through the
// claude-acc host tool (or via the dedicated PUT /token endpoint).
type CreateRequest struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name,omitempty"`
	ConfigDir   string `json:"config_dir,omitempty"`
	TokenPath   string `json:"token_path,omitempty"`
	Description string `json:"description,omitempty"`
	Enabled     *bool  `json:"enabled,omitempty"`
	Token       string `json:"token,omitempty"`
}

// UpdateRequest is the body for PUT /api/v1/claude-accounts/{id}.
// Pointer fields preserve "leave alone" vs "set to empty" semantics.
type UpdateRequest struct {
	Name        *string `json:"name,omitempty"`
	DisplayName *string `json:"display_name,omitempty"`
	ConfigDir   *string `json:"config_dir,omitempty"`
	TokenPath   *string `json:"token_path,omitempty"`
	Description *string `json:"description,omitempty"`
	Enabled     *bool   `json:"enabled,omitempty"`
}

// SetTokenRequest is the body for PUT /api/v1/claude-accounts/{id}/token.
type SetTokenRequest struct {
	Token string `json:"token"`
}

var (
	ErrNotFound  = errors.New("claude account not found")
	ErrDuplicate = errors.New("claude account name already exists")
	ErrDisabled  = errors.New("claude account is disabled")
)

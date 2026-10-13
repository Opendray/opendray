// Package opencodeacct manages multi-account binding for the OpenCode CLI
// (`opencode`). Unlike claude (CLAUDE_CONFIG_DIR), antigravity (HOME) and
// grok (GROK_HOME), an opencode account is NOT a directory: opencode reads
// its entire credential set from the OPENCODE_AUTH_CONTENT env var when
// set — the same JSON shape as ~/.local/share/opencode/auth.json, and it
// fully replaces the on-disk set. So an account is a named credential
// bundle that the catalog adapter injects per spawn.
//
// Every account shares the single opencode session DB (default data home),
// which is what makes a live account switch lossless: respawn with the new
// bundle and `--session <id>` resumes the same conversation. We never set
// XDG_DATA_HOME — that would leak into every tool the agent runs.
//
// Bundles are stored encrypted at rest (FieldCipher "v1:" envelope, the
// live backup cipher) and are never serialized back to clients — the API
// exposes only provider ids, auth type and a masked hint.
package opencodeacct

import (
	"errors"
	"time"
)

// Account describes one OpenCode credential bundle known to the gateway.
// The bundle itself is never part of this struct's JSON.
//
// ConfigDir / TokenFilled mirror cliacct/agyacct/grokacct field names so
// the shared web AccountSwitcher renders every provider uniformly;
// ConfigDir is always empty (opencode accounts have no directory) and
// TokenFilled reports whether the stored bundle decrypts to at least one
// credential.
type Account struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	ConfigDir   string    `json:"config_dir"`
	Description string    `json:"description"`
	Enabled     bool      `json:"enabled"`
	TokenFilled bool      `json:"token_filled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Providers is the non-secret list of opencode provider ids in the
	// bundle (e.g. "moonshotai"), persisted alongside the ciphertext.
	Providers []string `json:"providers"`
	// Credentials is a masked, display-only view of the bundle, computed
	// on read. Never contains a usable secret.
	Credentials []CredentialSummary `json:"credentials"`

	// LastUsedAt is MAX(sessions.started_at) where opencode_account_id
	// matches; nil when never pinned to a session.
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	// ActiveSessions counts non-terminal sessions pinned to this account.
	ActiveSessions int `json:"active_sessions"`

	// credentialsEnc is the stored envelope; unexported so it can never
	// reach a JSON response.
	credentialsEnc string
}

// CredentialSummary is the masked view of one provider entry.
type CredentialSummary struct {
	Provider string `json:"provider"`
	Type     string `json:"type"`
	Hint     string `json:"hint,omitempty"` // e.g. "sk-…a1b2"; empty for oauth
}

// CreateRequest is the body for POST /api/v1/opencode-accounts.
//
// Credentials is an auth.json-shaped object:
// {"<providerID>": {"type": "api", "key": "..."}, ...} (oauth/wellknown
// entries are passed through opaque). As a convenience, ProviderID +
// APIKey builds a single {"<ProviderID>": {"type":"api","key":APIKey}}
// bundle instead.
type CreateRequest struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name,omitempty"`
	Description string `json:"description,omitempty"`
	Enabled     *bool  `json:"enabled,omitempty"`

	Credentials map[string]any `json:"credentials,omitempty"`
	ProviderID  string         `json:"provider_id,omitempty"`
	APIKey      string         `json:"api_key,omitempty"`
}

// UpdateRequest is the body for PUT /api/v1/opencode-accounts/{id}.
// Pointer fields preserve "leave alone" vs "set"; a non-nil Credentials
// (or ProviderID+APIKey) replaces the whole bundle.
type UpdateRequest struct {
	Name        *string `json:"name,omitempty"`
	DisplayName *string `json:"display_name,omitempty"`
	Description *string `json:"description,omitempty"`
	Enabled     *bool   `json:"enabled,omitempty"`

	Credentials map[string]any `json:"credentials,omitempty"`
	ProviderID  string         `json:"provider_id,omitempty"`
	APIKey      string         `json:"api_key,omitempty"`
}

// ImportLocalRequest is the body for POST /opencode-accounts/import-local.
// Name defaults to "local".
type ImportLocalRequest struct {
	Name string `json:"name,omitempty"`
}

var (
	ErrNotFound           = errors.New("opencode account not found")
	ErrDuplicate          = errors.New("opencode account name already exists")
	ErrDisabled           = errors.New("opencode account is disabled")
	ErrNoCredentials      = errors.New("opencode account has no usable credentials")
	ErrInvalidCredentials = errors.New("invalid opencode credentials")
	ErrCipherRequired     = errors.New("opencode accounts need the backup cipher armed to encrypt credentials at rest (set up backups / OPENDRAY_BACKUP_KEY)")
)

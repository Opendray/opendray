// Package codexacct manages multi-account binding for the OpenAI Codex
// CLI (`codex`). Like grok (grokacct), codex keys its credential + config
// state off a home directory with its own env var, CODEX_HOME, so opendray
// relocates ONLY codex's state, never the whole $HOME. An "account" is a
// dedicated CODEX_HOME holding its own login at <CODEX_HOME>/auth.json
// (ChatGPT OAuth tokens or an API key); binding a session to an account
// means mirroring that credential into the session's scratch CODEX_HOME.
//
// Account rows hold metadata only (name, display name, the CODEX_HOME
// dir). The login lives on disk under <CODEX_HOME>/auth.json and is
// created out-of-band by `CODEX_HOME=<dir> codex login --device-auth`, or
// via SetAPIKey (`codex login --with-api-key`). Never copy one auth.json
// into another account's home: codex rotates OAuth refresh tokens, so two
// homes sharing one token invalidate each other.
package codexacct

import (
	"errors"
	"time"
)

// codexAuthRelPath is the login-token location relative to an account's
// CODEX_HOME, as written by `codex login`. Used to tell whether an account
// dir is actually logged in.
const codexAuthRelPath = "auth.json"

// Account describes one Codex account known to the gateway. The login
// token is intentionally NOT stored in the database — it lives at
// <ConfigDir>/auth.json where `codex` reads and refreshes it.
//
// ConfigDir is the per-account CODEX_HOME directory. The field is named to
// mirror cliacct.Account / agyacct.Account so the web API client + the
// shared AccountSwitcher component render every provider without
// special-casing JSON field names (config_dir / token_filled mean the
// analogous thing).
type Account struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	ConfigDir   string    `json:"config_dir"` // per-account CODEX_HOME directory
	Description string    `json:"description"`
	Enabled     bool      `json:"enabled"`
	TokenFilled bool      `json:"token_filled"` // login token present under CODEX_HOME
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Derived fields below are computed on each read, never persisted.

	// LastUsedAt is MAX(sessions.started_at) where codex_account_id
	// matches; nil when this account has never been pinned to a session.
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	// ActiveSessions counts non-terminal sessions pinned to this account.
	ActiveSessions int `json:"active_sessions"`

	// OAuthEmail / OAuthName tag the account with the ChatGPT holder it is
	// currently signed in as, read from <ConfigDir>/auth.json on each
	// list (mirrors cliacct's oauth_email). Empty when logged out.
	OAuthEmail string `json:"oauth_email,omitempty"`
	OAuthName  string `json:"oauth_name,omitempty"`
}

// CreateRequest is the body for POST /api/v1/codex-accounts.
//
// ConfigDir (the account CODEX_HOME) is optional; when omitted it is
// derived as <accountsDir>/<name>. The directory and its login token are
// created out-of-band via `CODEX_HOME=<dir> codex login` — Create only
// records the metadata row.
type CreateRequest struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name,omitempty"`
	ConfigDir   string `json:"config_dir,omitempty"`
	Description string `json:"description,omitempty"`
	Enabled     *bool  `json:"enabled,omitempty"`
}

// UpdateRequest is the body for PUT /api/v1/codex-accounts/{id}. Pointer
// fields preserve "leave alone" vs "set to empty" semantics.
type UpdateRequest struct {
	Name        *string `json:"name,omitempty"`
	DisplayName *string `json:"display_name,omitempty"`
	ConfigDir   *string `json:"config_dir,omitempty"`
	Description *string `json:"description,omitempty"`
	Enabled     *bool   `json:"enabled,omitempty"`
}

var (
	ErrNotFound      = errors.New("codex account not found")
	ErrDuplicate     = errors.New("codex account name already exists")
	ErrDisabled      = errors.New("codex account is disabled")
	ErrNotLoggedIn   = errors.New("codex account is not logged in")
	ErrNoUsableCodex = errors.New("no logged-in codex account available")
	ErrNoCodexBinary = errors.New("codex executable not found on PATH")
)

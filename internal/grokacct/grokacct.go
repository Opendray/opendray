// Package grokacct manages multi-account binding for the Grok Build CLI
// (`grok`). Like Antigravity (agyacct), grok keys its credential + config
// state off a home directory — but grok has its own GROK_HOME env var, so
// opendray relocates ONLY grok's state (auth.json, config.toml,
// trusted_folders.toml, sessions/), never the whole $HOME. An "account"
// is a dedicated GROK_HOME holding its own xAI login token at
// <GROK_HOME>/auth.json; binding a session to an account means spawning
// `grok` with GROK_HOME pointed at that directory.
//
// Account rows hold metadata only (name, display name, the GROK_HOME dir).
// The login token lives on disk under <GROK_HOME>/auth.json and is created
// out-of-band by running `GROK_HOME=<dir> grok login` once. This package
// NEVER writes tokens — it only discovers account dirs and points spawns
// at them.
package grokacct

import (
	"errors"
	"time"
)

// grokAuthRelPath is the login-token location relative to an account's
// GROK_HOME, as written by `grok login`. Used to tell whether an account
// dir is actually logged in.
const grokAuthRelPath = "auth.json"

// Account describes one Grok account known to the gateway. The login
// token is intentionally NOT stored in the database — it lives at
// <ConfigDir>/auth.json where `grok` reads and refreshes it.
//
// ConfigDir is the per-account GROK_HOME directory. The field is named to
// mirror cliacct.Account / agyacct.Account so the web API client + the
// shared AccountSwitcher component render every provider without
// special-casing JSON field names (config_dir / token_filled mean the
// analogous thing).
type Account struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	ConfigDir   string    `json:"config_dir"` // per-account GROK_HOME directory
	Description string    `json:"description"`
	Enabled     bool      `json:"enabled"`
	TokenFilled bool      `json:"token_filled"` // login token present under GROK_HOME
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Derived fields below are computed on each read, never persisted.

	// LastUsedAt is MAX(sessions.started_at) where grok_account_id
	// matches; nil when this account has never been pinned to a session.
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	// ActiveSessions counts non-terminal sessions pinned to this account.
	ActiveSessions int `json:"active_sessions"`
}

// CreateRequest is the body for POST /api/v1/grok-accounts.
//
// ConfigDir (the account GROK_HOME) is optional; when omitted it is
// derived as <accountsDir>/<name>. The directory and its login token are
// created out-of-band via `GROK_HOME=<dir> grok login` — Create only
// records the metadata row.
type CreateRequest struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name,omitempty"`
	ConfigDir   string `json:"config_dir,omitempty"`
	Description string `json:"description,omitempty"`
	Enabled     *bool  `json:"enabled,omitempty"`
}

// UpdateRequest is the body for PUT /api/v1/grok-accounts/{id}. Pointer
// fields preserve "leave alone" vs "set to empty" semantics.
type UpdateRequest struct {
	Name        *string `json:"name,omitempty"`
	DisplayName *string `json:"display_name,omitempty"`
	ConfigDir   *string `json:"config_dir,omitempty"`
	Description *string `json:"description,omitempty"`
	Enabled     *bool   `json:"enabled,omitempty"`
}

var (
	ErrNotFound  = errors.New("grok account not found")
	ErrDuplicate = errors.New("grok account name already exists")
	ErrDisabled  = errors.New("grok account is disabled")
)

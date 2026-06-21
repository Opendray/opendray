// Package agyacct manages multi-account binding for the Antigravity CLI
// (`agy`). Unlike Claude (which isolates accounts via CLAUDE_CONFIG_DIR),
// agy keys its ENTIRE credential + conversation state off $HOME: it
// reads/writes <HOME>/.gemini/antigravity-cli/antigravity-oauth-token
// (alongside conversations/, brain/, knowledge/). So an "account" here
// is a dedicated HOME directory holding its own OAuth token; binding a
// session to an account means spawning `agy` with HOME pointed at that
// directory.
//
// Account rows hold metadata only (name, display name, the HOME dir).
// The OAuth token lives on disk under <HOME>/.gemini/antigravity-cli/
// and is created out-of-band by running `agy` once under that HOME (the
// guided-login flow surfaced in the UI). This package NEVER writes
// tokens — it only discovers account dirs and points spawns at them.
package agyacct

import (
	"errors"
	"time"
)

// agyTokenRelPath is the OAuth token location relative to an account's
// HOME, as written by `agy` on a successful Google sign-in. Used to tell
// whether an account dir is actually logged in.
const agyTokenRelPath = ".gemini/antigravity-cli/antigravity-oauth-token"

// Account describes one Antigravity account known to the gateway. The
// OAuth token is intentionally NOT stored in the database — it lives
// under <ConfigDir>/.gemini/antigravity-cli/ where `agy` reads and
// refreshes it.
//
// ConfigDir is the per-account HOME directory. The field is named to
// mirror cliacct.Account so the web API client + the shared
// AccountSwitcher component render both providers without special-casing
// JSON field names (config_dir / token_filled mean the analogous thing).
type Account struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	ConfigDir   string    `json:"config_dir"` // per-account HOME directory
	Description string    `json:"description"`
	Enabled     bool      `json:"enabled"`
	TokenFilled bool      `json:"token_filled"` // OAuth token present under the HOME
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Derived fields below are computed on each read, never persisted.

	// LastUsedAt is MAX(sessions.started_at) where antigravity_account_id
	// matches; nil when this account has never been pinned to a session.
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	// ActiveSessions counts non-terminal sessions pinned to this account.
	// Always emitted so the UI can render "0 sessions" without special
	// casing.
	ActiveSessions int `json:"active_sessions"`
}

// CreateRequest is the body for POST /api/v1/antigravity-accounts.
//
// ConfigDir (the account HOME) is optional; when omitted it is derived
// as <accountsDir>/<name>. The directory and its OAuth token are
// created out-of-band via the guided `agy` login — Create only records
// the metadata row.
type CreateRequest struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name,omitempty"`
	ConfigDir   string `json:"config_dir,omitempty"`
	Description string `json:"description,omitempty"`
	Enabled     *bool  `json:"enabled,omitempty"`
}

// UpdateRequest is the body for PUT /api/v1/antigravity-accounts/{id}.
// Pointer fields preserve "leave alone" vs "set to empty" semantics.
type UpdateRequest struct {
	Name        *string `json:"name,omitempty"`
	DisplayName *string `json:"display_name,omitempty"`
	ConfigDir   *string `json:"config_dir,omitempty"`
	Description *string `json:"description,omitempty"`
	Enabled     *bool   `json:"enabled,omitempty"`
}

var (
	ErrNotFound  = errors.New("antigravity account not found")
	ErrDuplicate = errors.New("antigravity account name already exists")
	ErrDisabled  = errors.New("antigravity account is disabled")
)

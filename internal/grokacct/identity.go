package grokacct

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// oauthIdentity is the account-holder identity surfaced as a tag in the
// panel — the xAI login the account is currently signed in as. It mirrors
// cliacct's oauth_email: derived on every read, never persisted, so it
// always reflects who is logged in on disk right now.
type oauthIdentity struct {
	Email string
	Name  string // "First Last", trimmed; empty when no name on file
}

// authFileEntry is one issuer entry inside <GROK_HOME>/auth.json. grok
// keys the file by "<oidc_issuer>::<client_id>"; each value carries the
// signed-in holder's details alongside the tokens (which we never read).
type authFileEntry struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// readOAuthIdentity extracts the account holder's email and name from
// <configDir>/auth.json. Returns a zero identity (no error) when the home
// is blank, the file is missing/unreadable, or no entry carries an email —
// tagging is best-effort and must never break account listing.
func readOAuthIdentity(configDir string) oauthIdentity {
	if configDir == "" {
		return oauthIdentity{}
	}
	raw, err := os.ReadFile(filepath.Join(configDir, grokAuthRelPath))
	if err != nil {
		return oauthIdentity{}
	}
	var doc map[string]authFileEntry
	if err := json.Unmarshal(raw, &doc); err != nil {
		return oauthIdentity{}
	}
	// Deterministic pick when several issuer entries exist: first by key
	// order that actually carries an email.
	keys := make([]string, 0, len(doc))
	for k := range doc {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		e := doc[k]
		if e.Email == "" {
			continue
		}
		name := strings.TrimSpace(e.FirstName + " " + e.LastName)
		return oauthIdentity{Email: e.Email, Name: name}
	}
	return oauthIdentity{}
}

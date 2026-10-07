package codexacct

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// oauthIdentity is the account-holder identity surfaced as a tag in the
// panel — the ChatGPT login the account is currently signed in as. It
// mirrors cliacct's oauth_email: derived on every read, never persisted,
// so it always reflects who is logged in on disk right now.
type oauthIdentity struct {
	Email string
	Name  string
}

// authFile is the subset of <CODEX_HOME>/auth.json we read. A ChatGPT
// login carries tokens.id_token (an OIDC JWT whose claims hold the
// holder's email/name); an API-key login carries OPENAI_API_KEY instead
// and has no identity to show. The secrets themselves are never read
// beyond the id_token's public claims.
type authFile struct {
	Tokens *struct {
		IDToken string `json:"id_token"`
	} `json:"tokens"`
}

// readOAuthIdentity extracts the account holder's email and name from the
// id_token in <configDir>/auth.json. Returns a zero identity (no error)
// when the home is blank, the file is missing/unreadable, it's an API-key
// login, or the token carries no email — tagging is best-effort and must
// never break account listing. The JWT signature is not verified: the
// value is a display label only, never an authorization decision.
func readOAuthIdentity(configDir string) oauthIdentity {
	if configDir == "" {
		return oauthIdentity{}
	}
	raw, err := os.ReadFile(filepath.Join(configDir, codexAuthRelPath))
	if err != nil {
		return oauthIdentity{}
	}
	var doc authFile
	if err := json.Unmarshal(raw, &doc); err != nil || doc.Tokens == nil {
		return oauthIdentity{}
	}
	return identityFromIDToken(doc.Tokens.IDToken)
}

// identityFromIDToken decodes the (unverified) claims segment of a JWT.
func identityFromIDToken(tok string) oauthIdentity {
	parts := strings.Split(tok, ".")
	if len(parts) < 2 {
		return oauthIdentity{}
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(parts[1], "="))
	if err != nil {
		return oauthIdentity{}
	}
	var claims struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return oauthIdentity{}
	}
	return oauthIdentity{Email: claims.Email, Name: strings.TrimSpace(claims.Name)}
}

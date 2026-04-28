package integration

import (
	"crypto/rand"
	"encoding/base64"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const (
	apiKeyPrefix       = "odk_live_"
	apiKeyRandomBytes  = 32
	apiKeyBcryptCost   = 12
)

// generateAPIKey returns a fresh "odk_live_<43chars>" string and its
// bcrypt hash. The plaintext is shown once at registration; only the
// hash is persisted.
func generateAPIKey() (token, hash string, err error) {
	var b [apiKeyRandomBytes]byte
	if _, err = rand.Read(b[:]); err != nil {
		return "", "", err
	}
	token = apiKeyPrefix + base64.RawURLEncoding.EncodeToString(b[:])
	h, err := bcrypt.GenerateFromPassword([]byte(token), apiKeyBcryptCost)
	if err != nil {
		return "", "", err
	}
	return token, string(h), nil
}

// looksLikeAPIKey is a cheap prefix check; full verification still
// requires bcrypt.CompareHashAndPassword against the per-row hash.
func looksLikeAPIKey(token string) bool {
	return strings.HasPrefix(token, apiKeyPrefix)
}

// verifyAPIKey returns nil iff the token matches the bcrypt hash.
func verifyAPIKey(hash, token string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(token))
}

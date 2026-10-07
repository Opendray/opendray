package codexacct

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func fakeIDToken(claims string) string {
	enc := base64.RawURLEncoding.EncodeToString
	return enc([]byte(`{"alg":"none"}`)) + "." + enc([]byte(claims)) + ".sig"
}

func TestReadOAuthIdentity_ChatGPTLogin(t *testing.T) {
	home := t.TempDir()
	body := `{"OPENAI_API_KEY":null,"tokens":{"id_token":"` +
		fakeIDToken(`{"email":"dev@example.com","name":" Dev Person "}`) +
		`","access_token":"a","refresh_token":"r"}}`
	if err := os.WriteFile(filepath.Join(home, codexAuthRelPath), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	got := readOAuthIdentity(home)
	if got.Email != "dev@example.com" || got.Name != "Dev Person" {
		t.Errorf("identity = %+v", got)
	}
}

func TestReadOAuthIdentity_APIKeyLoginHasNoIdentity(t *testing.T) {
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, codexAuthRelPath), []byte(`{"OPENAI_API_KEY":"sk-x"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := readOAuthIdentity(home); got != (oauthIdentity{}) {
		t.Errorf("api-key login should have no identity, got %+v", got)
	}
}

func TestReadOAuthIdentity_MalformedIsZero(t *testing.T) {
	for _, tok := range []string{"", "not-a-jwt", "a.!!!.c"} {
		if got := identityFromIDToken(tok); got != (oauthIdentity{}) {
			t.Errorf("token %q: want zero identity, got %+v", tok, got)
		}
	}
	if got := readOAuthIdentity(""); got != (oauthIdentity{}) {
		t.Errorf("blank home: %+v", got)
	}
}

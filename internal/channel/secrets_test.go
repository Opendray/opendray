package channel

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// fakeFieldCipher mirrors the backup field envelope: EncryptField emits
// a "v1:" value, DecryptField reverses it (or fails when broken, standing
// in for a rotated key).
type fakeFieldCipher struct{ broken bool }

func (f fakeFieldCipher) EncryptField(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	return encryptedFieldPrefix + base64.StdEncoding.EncodeToString([]byte(plain)), nil
}

func (f fakeFieldCipher) DecryptField(env string) (string, error) {
	if f.broken {
		return "", errors.New("wrong key")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(env, encryptedFieldPrefix))
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func field(t *testing.T, raw json.RawMessage, key string) string {
	t.Helper()
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	v, ok := obj[key]
	if !ok {
		return "<absent>"
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		t.Fatalf("field %q not a string: %s", key, v)
	}
	return s
}

func TestEncryptDecryptConfigSecrets_RoundTrip(t *testing.T) {
	c := fakeFieldCipher{}
	in := json.RawMessage(`{"bot_token":"123:ABC","muted":true,"chat_id":"42"}`)

	enc := encryptConfigSecrets(c, "telegram", in)
	if got := field(t, enc, "bot_token"); !strings.HasPrefix(got, encryptedFieldPrefix) {
		t.Fatalf("bot_token not encrypted: %q", got)
	}
	// Non-secret fields untouched.
	if got := field(t, enc, "chat_id"); got != "42" {
		t.Errorf("chat_id changed to %q", got)
	}
	var encObj map[string]json.RawMessage
	if err := json.Unmarshal(enc, &encObj); err != nil {
		t.Fatalf("unmarshal enc: %v", err)
	}
	if string(encObj["muted"]) != "true" {
		t.Errorf("muted field changed/dropped: %s", encObj["muted"])
	}

	dec := decryptConfigSecrets(c, "telegram", enc)
	if got := field(t, dec, "bot_token"); got != "123:ABC" {
		t.Errorf("round-trip bot_token = %q, want 123:ABC", got)
	}
}

func TestEncryptConfigSecrets_MultipleFields(t *testing.T) {
	c := fakeFieldCipher{}
	in := json.RawMessage(`{"bot_token":"b","app_token":"a"}`)
	enc := encryptConfigSecrets(c, "slack", in)
	for _, f := range []string{"bot_token", "app_token"} {
		if got := field(t, enc, f); !strings.HasPrefix(got, encryptedFieldPrefix) {
			t.Errorf("slack %s not encrypted: %q", f, got)
		}
	}
}

func TestEncryptConfigSecrets_NoCipher_Plaintext(t *testing.T) {
	in := json.RawMessage(`{"bot_token":"plain"}`)
	out := encryptConfigSecrets(nil, "telegram", in)
	if got := field(t, out, "bot_token"); got != "plain" {
		t.Errorf("no-cipher should stay plaintext, got %q", got)
	}
}

func TestEncryptConfigSecrets_SkipsAlreadyEncrypted(t *testing.T) {
	c := fakeFieldCipher{}
	enc := encryptConfigSecrets(c, "telegram", json.RawMessage(`{"bot_token":"sekret"}`))
	once := field(t, enc, "bot_token")
	// Re-encrypting must be a no-op (never double-wrap), which is what
	// makes a read-modify-write under a rotated key safe.
	twice := field(t, encryptConfigSecrets(c, "telegram", enc), "bot_token")
	if once != twice {
		t.Errorf("double-encrypt changed value: %q -> %q", once, twice)
	}
}

func TestDecryptConfigSecrets_RotatedKey_PreservesCiphertext(t *testing.T) {
	good := fakeFieldCipher{}
	enc := encryptConfigSecrets(good, "telegram", json.RawMessage(`{"bot_token":"x"}`))
	ciphertext := field(t, enc, "bot_token")

	// Decrypt with a broken (rotated) key: the ciphertext must survive,
	// not be blanked — otherwise a later re-store would erase the token.
	dec := decryptConfigSecrets(fakeFieldCipher{broken: true}, "telegram", enc)
	if got := field(t, dec, "bot_token"); got != ciphertext {
		t.Errorf("rotated-key decrypt should preserve ciphertext %q, got %q", ciphertext, got)
	}

	// And a re-encrypt of that preserved ciphertext is still a no-op.
	re := encryptConfigSecrets(fakeFieldCipher{broken: true}, "telegram", dec)
	if got := field(t, re, "bot_token"); got != ciphertext {
		t.Errorf("re-store under rotated key lost the secret: %q", got)
	}
}

func TestConfigSecrets_UnknownKind_Untouched(t *testing.T) {
	c := fakeFieldCipher{}
	in := json.RawMessage(`{"bot_token":"x"}`)
	if got := string(encryptConfigSecrets(c, "nosuchkind", in)); got != string(in) {
		t.Errorf("unknown kind should be untouched, got %s", got)
	}
}

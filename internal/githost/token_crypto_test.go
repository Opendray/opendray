package githost

import (
	"encoding/base64"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
)

// fakeCipher mimics backup.Cipher's field envelope: EncryptField emits a
// "v1:"-prefixed value, DecryptField reverses it (or fails when armed is
// false, standing in for a missing key / rotated passphrase).
type fakeCipher struct{ broken bool }

func (f fakeCipher) EncryptField(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	return encryptedTokenPrefix + base64.StdEncoding.EncodeToString([]byte(plain)), nil
}

func (f fakeCipher) DecryptField(env string) (string, error) {
	if f.broken {
		return "", errors.New("wrong key")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(env, encryptedTokenPrefix))
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func newTestService(c FieldCipher) *Service {
	s := NewService(nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	s.cipher = c
	return s
}

func TestEncodeDecodeToken_RoundTrip(t *testing.T) {
	s := newTestService(fakeCipher{})
	const tok = "ghp_secrettoken123"
	enc := s.encodeToken(tok)
	if !strings.HasPrefix(enc, encryptedTokenPrefix) {
		t.Fatalf("encoded token %q missing %q prefix", enc, encryptedTokenPrefix)
	}
	if enc == tok {
		t.Fatal("token was not encrypted")
	}
	if got := s.decodeToken(enc); got != tok {
		t.Errorf("decodeToken round-trip = %q, want %q", got, tok)
	}
}

func TestEncodeToken_NoCipher_Plaintext(t *testing.T) {
	s := newTestService(nil)
	const tok = "glpat-abc"
	if got := s.encodeToken(tok); got != tok {
		t.Errorf("no-cipher encode = %q, want plaintext %q", got, tok)
	}
}

func TestDecodeToken_LegacyPlaintextPassthrough(t *testing.T) {
	s := newTestService(fakeCipher{})
	const legacy = "ghp_plaintextlegacy" // no v1: prefix
	if got := s.decodeToken(legacy); got != legacy {
		t.Errorf("legacy plaintext should pass through, got %q", got)
	}
}

func TestDecodeToken_DecryptFailure_ReturnsEmpty(t *testing.T) {
	enc := fakeCipher{}.mustEncrypt(t, "tok")
	s := newTestService(fakeCipher{broken: true})
	if got := s.decodeToken(enc); got != "" {
		t.Errorf("decrypt failure should yield empty token, got %q", got)
	}
}

func TestDecodeToken_EncryptedButNoCipher_ReturnsEmpty(t *testing.T) {
	enc := fakeCipher{}.mustEncrypt(t, "tok")
	s := newTestService(nil)
	if got := s.decodeToken(enc); got != "" {
		t.Errorf("encrypted value with no cipher should yield empty, got %q", got)
	}
}

func TestEncodeToken_EmptyStaysEmpty(t *testing.T) {
	s := newTestService(fakeCipher{})
	if got := s.encodeToken(""); got != "" {
		t.Errorf("empty token should stay empty, got %q", got)
	}
}

func (f fakeCipher) mustEncrypt(t *testing.T, plain string) string {
	t.Helper()
	enc, err := f.EncryptField(plain)
	if err != nil {
		t.Fatalf("EncryptField: %v", err)
	}
	return enc
}

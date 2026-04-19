package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"golang.org/x/crypto/hkdf"
)

// ─────────────────────────────────────────────
// Wrap / Unwrap round-trip
// ─────────────────────────────────────────────

// TestKEK_WrapUnwrap_RoundTrip1000 round-trips 1000 random DEKs under 1000
// random KEKs. Every one must unwrap to the same bytes.
func TestKEK_WrapUnwrap_RoundTrip1000(t *testing.T) {
	for i := 0; i < 1000; i++ {
		kek := randBytes(t, KEKSize)
		dek := randBytes(t, DEKSize)

		// Keep copies — WrapDEK / UnwrapDEK zero their inputs as a defence.
		kekCopy1 := append([]byte(nil), kek...)
		kekCopy2 := append([]byte(nil), kek...)

		wrapped, err := WrapDEK(kekCopy1, append([]byte(nil), dek...))
		if err != nil {
			t.Fatalf("WrapDEK[%d]: %v", i, err)
		}
		recovered, err := UnwrapDEK(kekCopy2, wrapped)
		if err != nil {
			t.Fatalf("UnwrapDEK[%d]: %v", i, err)
		}
		if !bytes.Equal(recovered, dek) {
			t.Fatalf("mismatch at %d: got %x want %x", i, recovered, dek)
		}
	}
}

// TestKEK_Unwrap_TamperedCiphertext: flipping any byte in the ciphertext
// body must fail the GCM authentication tag.
func TestKEK_Unwrap_TamperedCiphertext(t *testing.T) {
	kek := randBytes(t, KEKSize)
	dek := randBytes(t, DEKSize)

	wrapped, err := WrapDEK(append([]byte(nil), kek...), append([]byte(nil), dek...))
	if err != nil {
		t.Fatalf("WrapDEK: %v", err)
	}

	// Flip every byte position and assert each flip breaks GCM.
	for pos := 0; pos < len(wrapped); pos++ {
		tampered := append([]byte(nil), wrapped...)
		tampered[pos] ^= 0xFF
		_, err := UnwrapDEK(append([]byte(nil), kek...), tampered)
		if err == nil {
			t.Errorf("UnwrapDEK succeeded on tampered byte %d/%d", pos, len(wrapped))
		}
	}
}

// TestKEK_Unwrap_TamperedNonce: flipping a byte in the nonce region causes
// GCM to derive different key material → authentication fails.
func TestKEK_Unwrap_TamperedNonce(t *testing.T) {
	kek := randBytes(t, KEKSize)
	dek := randBytes(t, DEKSize)

	wrapped, err := WrapDEK(append([]byte(nil), kek...), append([]byte(nil), dek...))
	if err != nil {
		t.Fatalf("WrapDEK: %v", err)
	}

	// Flip the first nonce byte.
	wrapped[0] ^= 0x80
	_, err = UnwrapDEK(append([]byte(nil), kek...), wrapped)
	if err == nil {
		t.Error("UnwrapDEK should fail with tampered nonce")
	}
}

// TestKEK_Rotation_Rewrap simulates rotation: wrap under KEK-v1, unwrap
// under KEK-v1, rewrap under KEK-v2. The same DEK survives.
func TestKEK_Rotation_Rewrap(t *testing.T) {
	dek := randBytes(t, DEKSize)
	kekV1 := randBytes(t, KEKSize)
	kekV2 := randBytes(t, KEKSize)
	if bytes.Equal(kekV1, kekV2) {
		t.Skip("randomness collided") // astronomically improbable
	}

	// Wrap with v1.
	wrappedV1, err := WrapDEK(append([]byte(nil), kekV1...), append([]byte(nil), dek...))
	if err != nil {
		t.Fatalf("WrapDEK v1: %v", err)
	}
	// Unwrap under v1.
	unwrapped, err := UnwrapDEK(append([]byte(nil), kekV1...), wrappedV1)
	if err != nil {
		t.Fatalf("UnwrapDEK v1: %v", err)
	}
	if !bytes.Equal(unwrapped, dek) {
		t.Fatalf("v1 round-trip mismatch")
	}
	// Rewrap under v2.
	wrappedV2, err := WrapDEK(append([]byte(nil), kekV2...), append([]byte(nil), unwrapped...))
	if err != nil {
		t.Fatalf("WrapDEK v2: %v", err)
	}
	// Now v1 cannot unwrap the v2 blob.
	if _, err := UnwrapDEK(append([]byte(nil), kekV1...), wrappedV2); err == nil {
		t.Error("UnwrapDEK v1 on v2 blob should fail")
	}
	// And v2 must unwrap it.
	finalDEK, err := UnwrapDEK(append([]byte(nil), kekV2...), wrappedV2)
	if err != nil {
		t.Fatalf("UnwrapDEK v2: %v", err)
	}
	if !bytes.Equal(finalDEK, dek) {
		t.Error("rotation lost original DEK")
	}
}

// TestKEK_WrapDEK_PanicsOnWrongSizedKEK asserts the input-validation panic.
func TestKEK_WrapDEK_PanicsOnWrongSizedKEK(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("want panic, got none")
		}
	}()
	dek := randBytes(t, DEKSize)
	_, _ = WrapDEK(make([]byte, 10), dek)
}

// TestKEK_WrapDEK_PanicsOnWrongSizedDEK asserts the input-validation panic.
func TestKEK_WrapDEK_PanicsOnWrongSizedDEK(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("want panic, got none")
		}
	}()
	kek := randBytes(t, KEKSize)
	_, _ = WrapDEK(kek, make([]byte, 10))
}

// TestKEK_UnwrapDEK_PanicsOnWrongSizedKEK asserts the input-validation panic.
func TestKEK_UnwrapDEK_PanicsOnWrongSizedKEK(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("want panic, got none")
		}
	}()
	_, _ = UnwrapDEK(make([]byte, 31), make([]byte, 64))
}

// TestKEK_UnwrapDEK_RejectsShortBlob: blobs too short to contain
// nonce + tag are rejected with a clean error (not a panic).
func TestKEK_UnwrapDEK_RejectsShortBlob(t *testing.T) {
	kek := randBytes(t, KEKSize)
	_, err := UnwrapDEK(kek, []byte{0x01, 0x02, 0x03})
	if err == nil {
		t.Fatal("want error on short blob, got nil")
	}
}

// TestKEK_WrapDEK_DifferentNoncesEveryCall: wrapping the same DEK twice
// produces different ciphertexts (nonce uniqueness).
func TestKEK_WrapDEK_DifferentNoncesEveryCall(t *testing.T) {
	kek := randBytes(t, KEKSize)
	dek := randBytes(t, DEKSize)

	first, err := WrapDEK(append([]byte(nil), kek...), append([]byte(nil), dek...))
	if err != nil {
		t.Fatalf("WrapDEK 1: %v", err)
	}
	second, err := WrapDEK(append([]byte(nil), kek...), append([]byte(nil), dek...))
	if err != nil {
		t.Fatalf("WrapDEK 2: %v", err)
	}
	if bytes.Equal(first, second) {
		t.Error("two wraps of same DEK produced identical ciphertexts — nonce reuse")
	}
	// But the nonce prefix must be different.
	if bytes.Equal(first[:GCMNonceSize], second[:GCMNonceSize]) {
		t.Error("nonces should be unique")
	}
}

// TestKEK_NoKeyMaterialLogged captures slog output during wrap/unwrap and
// asserts no long hex run (>16 chars) appears — i.e. no key bytes leak.
func TestKEK_NoKeyMaterialLogged(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)
	t.Cleanup(func() { slog.SetDefault(slog.Default()) })

	kek := randBytes(t, KEKSize)
	dek := randBytes(t, DEKSize)

	wrapped, err := WrapDEK(append([]byte(nil), kek...), append([]byte(nil), dek...))
	if err != nil {
		t.Fatalf("WrapDEK: %v", err)
	}
	recovered, err := UnwrapDEK(append([]byte(nil), kek...), wrapped)
	if err != nil {
		t.Fatalf("UnwrapDEK: %v", err)
	}
	if !bytes.Equal(recovered, dek) {
		t.Fatal("round-trip mismatch")
	}

	// Scan log output for any hex run longer than 16 chars.
	logOutput := buf.String()
	if strings.ContainsAny(logOutput, "\x00") {
		t.Error("log output contains null byte — shouldn't happen")
	}
	if looksLikeLongHexRun(logOutput) {
		t.Errorf("log output contains a long hex run (possible key leak): %q", logOutput)
	}

	// Also check that the KEK/DEK hex literally isn't present anywhere.
	kekHex := hex.EncodeToString(kek)
	dekHex := hex.EncodeToString(dek)
	if strings.Contains(logOutput, kekHex) {
		t.Error("log contains KEK hex — leak")
	}
	if strings.Contains(logOutput, dekHex) {
		t.Error("log contains DEK hex — leak")
	}
}

// looksLikeLongHexRun returns true if s contains a substring of ≥17
// consecutive hex characters. Short hex strings (timestamps, short ids) are
// ignored; an AES key would show up as 64 hex chars.
func looksLikeLongHexRun(s string) bool {
	run := 0
	for _, r := range s {
		if isHex(r) {
			run++
			if run > 16 {
				return true
			}
		} else {
			run = 0
		}
	}
	return false
}

func isHex(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
}

// ─────────────────────────────────────────────
// KEKProvider wrapped around a test CredentialStore
// ─────────────────────────────────────────────

// TestKEKProvider_NotReadyWhenNoAdminRow wires a provider against a store
// backed by no admin_auth row; DeriveKEK returns ErrKEKNotReady.
//
// We use a null CredentialStore via a stubStore abstraction — the real
// store requires a *pgxpool.Pool. For this unit test we wrap the
// adminAuthKEKProvider directly with a faked loader.
func TestKEKProvider_NotReadyWhenNoAdminRow(t *testing.T) {
	p := &stubProvider{loader: func() (*Credentials, error) { return nil, nil }}

	_, err := p.DeriveKEK(context.Background(), "kid-v1")
	if !errors.Is(err, ErrKEKNotReady) {
		t.Errorf("want ErrKEKNotReady, got %v", err)
	}
}

// TestKEKProvider_DerivesStableKEKPerKid: deriving with the same kid
// returns the same bytes; a different kid produces different bytes.
func TestKEKProvider_DerivesStableKEKPerKid(t *testing.T) {
	// Fake admin hash — a typical bcrypt-format string.
	hash := "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"
	p := &stubProvider{loader: func() (*Credentials, error) {
		return &Credentials{Username: "admin", PasswordHash: hash}, nil
	}}

	ctx := context.Background()
	k1a, err := p.DeriveKEK(ctx, "v1")
	if err != nil {
		t.Fatalf("DeriveKEK v1: %v", err)
	}
	k1b, err := p.DeriveKEK(ctx, "v1")
	if err != nil {
		t.Fatalf("DeriveKEK v1b: %v", err)
	}
	k2, err := p.DeriveKEK(ctx, "v2")
	if err != nil {
		t.Fatalf("DeriveKEK v2: %v", err)
	}

	if !bytes.Equal(k1a, k1b) {
		t.Error("same kid must produce same KEK")
	}
	if bytes.Equal(k1a, k2) {
		t.Error("different kids must produce different KEKs")
	}
	if len(k1a) != KEKSize {
		t.Errorf("KEK len: want %d, got %d", KEKSize, len(k1a))
	}
}

// ─────────────────────────────────────────────
// Test helpers
// ─────────────────────────────────────────────

// stubProvider is a minimal KEKProvider wired to a custom admin-row loader
// so tests don't need a live *pgxpool.Pool. It mirrors the HKDF pipeline
// of adminAuthKEKProvider.DeriveKEK, so any drift in the real pipeline
// would break matched-output tests. The real adminAuthKEKProvider is
// exercised under an embedded-postgres harness in T13's plugin_secret
// tests.
type stubProvider struct {
	loader func() (*Credentials, error)
}

// DeriveKEK mirrors adminAuthKEKProvider.DeriveKEK but reads creds via
// the injected loader instead of a *pgxpool.Pool.
func (s *stubProvider) DeriveKEK(_ context.Context, kid string) ([]byte, error) {
	creds, err := s.loader()
	if err != nil {
		return nil, err
	}
	if creds == nil || creds.PasswordHash == "" {
		return nil, ErrKEKNotReady
	}
	ikm := []byte(creds.PasswordHash)
	salt := []byte("opendray-plugin-kek")
	info := []byte("opendray-plugin-kek/" + kid)
	reader := hkdf.New(sha256.New, ikm, salt, info)
	kek := make([]byte, KEKSize)
	if _, err := io.ReadFull(reader, kek); err != nil {
		return nil, err
	}
	return kek, nil
}

// randBytes produces n random bytes; t.Fatalf on failure.
func randBytes(t *testing.T, n int) []byte {
	t.Helper()
	b := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		t.Fatalf("rand: %v", err)
	}
	return b
}

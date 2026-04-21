// Package auth — Key-Encryption-Key (KEK) derivation and DEK wrap/unwrap
// primitives for opendray.secret.* (M3 T7).
//
// # Design
//
// The host never persists the KEK. Instead it derives one on demand via
// HKDF-SHA256 from the admin bcrypt hash (already stored in admin_auth).
// Rotating the admin password rotates the KEK; a host-side walk rewraps
// every plugin_secret_kek row on next login (see T13).
//
// Data-encryption keys (DEKs) are 32 random bytes, one per plugin, wrapped
// under the KEK and stored in plugin_secret_kek.wrapped_dek. WrapDEK /
// UnwrapDEK use AES-256-GCM with a fresh 12-byte random nonce per wrap.
//
// # Key leak mitigation
//
// Every function that takes a KEK defers a zeroBytes on its local copy so
// the memory is blanked before the slice is garbage-collected. This is a
// best-effort defence: Go's garbage collector may copy backing arrays, so
// zeroBytes cannot absolutely eliminate key residue. It does, however,
// eliminate the common case where a debug print or a panic stack dump
// exposes the key. No key material is ever logged.
package auth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
)

// KEKSize is the fixed KEK length (32 bytes for AES-256-GCM).
const KEKSize = 32

// DEKSize is the fixed DEK length (also 32 bytes — we encrypt DEKs with
// AES-256-GCM under the KEK, then use the DEK itself with AES-256-GCM on
// individual secret values).
const DEKSize = 32

// GCMNonceSize is the nonce size for AES-GCM (RFC 5116).
const GCMNonceSize = 12

// ErrKEKNotReady is returned by DeriveKEK when the admin_auth row hasn't
// been populated yet. Callers (T13) defer KEK-dependent operations until
// admin setup completes.
var ErrKEKNotReady = errors.New("auth: KEK not ready — admin credentials not yet initialised")

// ─────────────────────────────────────────────
// KEKProvider
// ─────────────────────────────────────────────

// KEKProvider derives the host KEK on demand, keyed by a key-id (kid).
// Different kids let callers rotate the KEK without touching persisted
// wrapped DEKs — a rewrap walk can unwrap with the old kid and rewrap
// with the new.
type KEKProvider interface {
	// DeriveKEK returns a 32-byte KEK keyed by kid. On a fresh install
	// (no admin credentials yet) it returns ErrKEKNotReady so callers
	// can surface a clean EUNAVAIL to plugins.
	DeriveKEK(ctx context.Context, kid string) ([]byte, error)
}

// adminAuthKEKProvider derives the KEK from the admin password hash stored
// in admin_auth. Construction is deferred until admin setup completes —
// until then DeriveKEK returns ErrKEKNotReady.
type adminAuthKEKProvider struct {
	store *CredentialStore
}

// NewKEKProviderFromAdminAuth wires a KEKProvider to read the admin bcrypt
// hash from store. The returned provider is safe for concurrent use.
func NewKEKProviderFromAdminAuth(store *CredentialStore) KEKProvider {
	return &adminAuthKEKProvider{store: store}
}

// DeriveKEK implements KEKProvider. Pipeline:
//
//  1. Load admin credentials; error if store is unreachable.
//  2. If no row yet → ErrKEKNotReady.
//  3. HKDF-SHA256 over the bcrypt hash bytes with:
//     salt = "opendray-plugin-kek"
//     info = "opendray-plugin-kek/" + kid
//     Output 32 bytes.
//
// The admin bcrypt hash is a stable high-entropy secret suitable as HKDF
// input keying material (IKM). Using it directly (rather than the plaintext
// password) means the KEK survives admin logins — we never see the password
// again after the first bcrypt round.
func (p *adminAuthKEKProvider) DeriveKEK(ctx context.Context, kid string) ([]byte, error) {
	creds, err := p.store.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("auth: load admin creds for KEK derivation: %w", err)
	}
	if creds == nil {
		return nil, ErrKEKNotReady
	}
	if creds.PasswordHash == "" {
		return nil, ErrKEKNotReady
	}

	ikm := []byte(creds.PasswordHash)
	salt := []byte("opendray-plugin-kek")
	info := []byte("opendray-plugin-kek/" + kid)

	reader := hkdf.New(sha256.New, ikm, salt, info)
	kek := make([]byte, KEKSize)
	if _, err := io.ReadFull(reader, kek); err != nil {
		return nil, fmt.Errorf("auth: hkdf expand KEK: %w", err)
	}
	return kek, nil
}

// ─────────────────────────────────────────────
// Wrap / Unwrap DEK
// ─────────────────────────────────────────────

// WrapDEK encrypts dek under kek using AES-256-GCM. Output shape:
//
//	wrapped = nonce (12 B) || ciphertext (len(dek)+16 for the GCM tag)
//
// Panics on wrong-sized keys: both kek and dek MUST be 32 bytes. A
// wrong-sized key is a programmer error, never a runtime condition.
func WrapDEK(kek, dek []byte) ([]byte, error) {
	if len(kek) != KEKSize {
		panic(fmt.Sprintf("auth: WrapDEK: kek must be %d bytes, got %d", KEKSize, len(kek)))
	}
	if len(dek) != DEKSize {
		panic(fmt.Sprintf("auth: WrapDEK: dek must be %d bytes, got %d", DEKSize, len(dek)))
	}
	// Defensive zeroing of our local kek view after use.
	defer zeroBytes(kek)

	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, fmt.Errorf("auth: aes.NewCipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("auth: cipher.NewGCM: %w", err)
	}

	nonce := make([]byte, GCMNonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("auth: read nonce: %w", err)
	}

	// Seal writes ciphertext+tag. We prepend the nonce so Unwrap can
	// recover both from a single blob.
	ct := gcm.Seal(nil, nonce, dek, nil)
	out := make([]byte, 0, GCMNonceSize+len(ct))
	out = append(out, nonce...)
	out = append(out, ct...)
	return out, nil
}

// UnwrapDEK is the inverse of WrapDEK. Returns the 32-byte DEK or an error
// on authentication failure (tampered ciphertext, wrong KEK, malformed
// blob).
//
// Panics if kek is not 32 bytes (programmer error).
func UnwrapDEK(kek, wrapped []byte) ([]byte, error) {
	if len(kek) != KEKSize {
		panic(fmt.Sprintf("auth: UnwrapDEK: kek must be %d bytes, got %d", KEKSize, len(kek)))
	}
	defer zeroBytes(kek)

	if len(wrapped) < GCMNonceSize+16 /* GCM tag */ {
		return nil, errors.New("auth: UnwrapDEK: wrapped blob too short")
	}
	nonce := wrapped[:GCMNonceSize]
	ct := wrapped[GCMNonceSize:]

	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, fmt.Errorf("auth: aes.NewCipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("auth: cipher.NewGCM: %w", err)
	}

	dek, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		// Do NOT include KEK or ciphertext bytes in the error — they are
		// secret material. A generic message is enough for the caller to
		// surface EINTERNAL; the operator can correlate by timestamp.
		return nil, fmt.Errorf("auth: UnwrapDEK: GCM open failed: %w", err)
	}
	if len(dek) != DEKSize {
		return nil, fmt.Errorf("auth: UnwrapDEK: unwrapped DEK has wrong size %d", len(dek))
	}
	return dek, nil
}

// ─────────────────────────────────────────────
// zeroBytes — best-effort key erasure
// ─────────────────────────────────────────────

// zeroBytes overwrites the byte slice with zeros. Go's compiler may elide
// this in hot loops if it can prove the slice is dead, but for the
// defer-after-use pattern in WrapDEK/UnwrapDEK the side-effect survives
// optimisation in practice. This is a defence-in-depth measure — the true
// protection is "don't log keys" and "don't panic with them in scope".
func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

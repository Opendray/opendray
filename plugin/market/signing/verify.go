// Package signing implements Ed25519 signature verification for
// OpenDray marketplace plugin entries.
//
// Canonical signed bytes:
//
//	[]byte(entry.SHA256)
//
// The lowercase-hex artifact hash is the full signed payload. Since
// the artifact (zip) embeds manifest.json and HTTPSSource verifies
// the hash before extracting, this single-hash signature binds the
// entire bundle. The marketplace CI (pr-validate.yml →
// manifest-match job) guarantees the registry's manifest copy
// equals the bundle's manifest.json, so we don't need a separate
// manifest-hash signature layer.
//
// Policy (which trust levels REQUIRE a verified signature) lives in
// the install handler — this package is pure verification.
package signing

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/opendray/opendray/plugin/market"
)

// ErrNoSignature signals the entry has no attached signature.
// Callers decide whether to treat that as an error based on the
// publisher's trust level.
var ErrNoSignature = errors.New("signing: entry has no signature")

// ErrNoMatchingKey signals the signature references a key the
// publisher record doesn't acknowledge. Treat as a hard failure.
var ErrNoMatchingKey = errors.New("signing: signature key not in publisher record")

// ErrBadSignature signals ed25519.Verify rejected the bytes. Hard
// failure — either the artifact changed or the signer's key
// doesn't own it.
var ErrBadSignature = errors.New("signing: signature failed verification")

// VerifySignature checks that `entry.Signature` was produced by a
// key registered on `publisher`. Returns nil on success, a typed
// error on any failure.
//
// Validation rules:
//
//   - entry.Signature != nil; otherwise ErrNoSignature.
//   - entry.Signature.Alg == "ed25519"; other algorithms rejected
//     outright (the schema only allows ed25519 anyway; defence in
//     depth).
//   - entry.Signature.PublicKey (base64) decodes to exactly 32
//     bytes (Ed25519 pubkey size).
//   - entry.Signature.Value (base64) decodes to exactly 64 bytes
//     (Ed25519 signature size).
//   - entry.SHA256 is a 64-hex string (already validated by
//     Resolve; defence).
//   - publisher.Keys contains a row whose PublicKey matches the
//     signature's PublicKey AND whose ExpiresAt (if set) is in the
//     future AND whose RevokedAt is empty.
//   - ed25519.Verify(key, []byte(entry.SHA256), sig) returns true.
//
// `now` lets tests exercise expiry without a clock-mock package;
// production callers pass time.Now.
func VerifySignature(entry market.Entry, publisher market.PublisherRecord, now time.Time) error {
	if entry.Signature == nil {
		return ErrNoSignature
	}
	sig := entry.Signature
	if sig.Alg != "ed25519" {
		return fmt.Errorf("%w: unsupported alg %q", ErrBadSignature, sig.Alg)
	}
	if entry.SHA256 == "" || len(entry.SHA256) != 64 {
		return fmt.Errorf("%w: malformed entry.SHA256", ErrBadSignature)
	}
	pubBytes, err := base64.StdEncoding.DecodeString(sig.PublicKey)
	if err != nil {
		return fmt.Errorf("%w: publicKey base64: %v", ErrBadSignature, err)
	}
	if len(pubBytes) != ed25519.PublicKeySize {
		return fmt.Errorf("%w: publicKey size %d != %d",
			ErrBadSignature, len(pubBytes), ed25519.PublicKeySize)
	}
	sigBytes, err := base64.StdEncoding.DecodeString(sig.Value)
	if err != nil {
		return fmt.Errorf("%w: value base64: %v", ErrBadSignature, err)
	}
	if len(sigBytes) != ed25519.SignatureSize {
		return fmt.Errorf("%w: value size %d != %d",
			ErrBadSignature, len(sigBytes), ed25519.SignatureSize)
	}

	// Publisher must register this key, and it must be usable.
	if !hasUsableKey(publisher, sig.PublicKey, now) {
		return ErrNoMatchingKey
	}

	if !ed25519.Verify(pubBytes, []byte(entry.SHA256), sigBytes) {
		return ErrBadSignature
	}
	return nil
}

// hasUsableKey searches the publisher record for a key matching
// the base64 pubkey, whose window includes `now`. Empty ExpiresAt
// means no expiry.
func hasUsableKey(publisher market.PublisherRecord, base64PubKey string, now time.Time) bool {
	for _, k := range publisher.Keys {
		if k.Alg != "ed25519" || k.PublicKey != base64PubKey {
			continue
		}
		if k.RevokedAt != "" {
			// Any revocation timestamp makes the key unusable
			// regardless of when it was set — defensive; the
			// alternative is a "revoked before signed" comparison
			// that the marketplace repo doesn't currently support.
			continue
		}
		if k.ExpiresAt != "" {
			exp, err := time.Parse(time.RFC3339, k.ExpiresAt)
			if err != nil {
				// An unparseable expiresAt is a registry bug but
				// shouldn't cause a false-positive trust; skip.
				continue
			}
			if !now.Before(exp) {
				continue
			}
		}
		return true
	}
	return false
}

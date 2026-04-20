package signing

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/opendray/opendray/plugin/market"
)

const sampleSHA256 = "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"

// keypair generates an Ed25519 keypair for tests. Returns the
// base64 public key + a sign func that produces base64 signatures
// over []byte(sampleSHA256).
func keypair(t *testing.T) (b64pub string, sign func(data []byte) string) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	b64pub = base64.StdEncoding.EncodeToString(pub)
	sign = func(data []byte) string {
		return base64.StdEncoding.EncodeToString(ed25519.Sign(priv, data))
	}
	return
}

func newPublisher(b64pub, expiresAt, revokedAt string) market.PublisherRecord {
	return market.PublisherRecord{
		Name:  "acme",
		Trust: "verified",
		Keys: []market.PublisherKey{
			{
				Alg:       "ed25519",
				PublicKey: b64pub,
				AddedAt:   "2024-01-01T00:00:00Z",
				ExpiresAt: expiresAt,
				RevokedAt: revokedAt,
			},
		},
	}
}

func signedEntry(b64pub string, sigValue string) market.Entry {
	return market.Entry{
		Name:      "plug",
		Publisher: "acme",
		Version:   "1.0.0",
		SHA256:    sampleSHA256,
		Signature: &market.Signature{
			Alg:       "ed25519",
			PublicKey: b64pub,
			Value:     sigValue,
		},
	}
}

// ─── Happy path ────────────────────────────────────────────────────────────

func TestVerify_HappyPath(t *testing.T) {
	pub, sign := keypair(t)
	entry := signedEntry(pub, sign([]byte(sampleSHA256)))
	publisher := newPublisher(pub, "", "")

	if err := VerifySignature(entry, publisher, time.Now()); err != nil {
		t.Fatalf("VerifySignature: %v", err)
	}
}

// ─── No signature ──────────────────────────────────────────────────────────

func TestVerify_NoSignature(t *testing.T) {
	entry := market.Entry{SHA256: sampleSHA256}
	publisher := market.PublisherRecord{Name: "acme", Trust: "community"}

	err := VerifySignature(entry, publisher, time.Now())
	if !errors.Is(err, ErrNoSignature) {
		t.Errorf("err = %v, want ErrNoSignature", err)
	}
}

// ─── Algorithm mismatch ────────────────────────────────────────────────────

func TestVerify_BadAlg(t *testing.T) {
	pub, sign := keypair(t)
	entry := signedEntry(pub, sign([]byte(sampleSHA256)))
	entry.Signature.Alg = "rsa"

	err := VerifySignature(entry, newPublisher(pub, "", ""), time.Now())
	if !errors.Is(err, ErrBadSignature) {
		t.Errorf("err = %v, want ErrBadSignature (bad alg)", err)
	}
}

// ─── Tampered bytes ────────────────────────────────────────────────────────

func TestVerify_WrongSHA(t *testing.T) {
	pub, sign := keypair(t)
	// Sign "honest" bytes but put a different SHA in the entry.
	entry := signedEntry(pub, sign([]byte("honest-bytes-not-the-real-sha-256-value-64char-hex-value-xxxxxx")))
	publisher := newPublisher(pub, "", "")

	err := VerifySignature(entry, publisher, time.Now())
	if !errors.Is(err, ErrBadSignature) {
		t.Errorf("err = %v, want ErrBadSignature", err)
	}
}

func TestVerify_TamperedSignature(t *testing.T) {
	pub, sign := keypair(t)
	good := sign([]byte(sampleSHA256))
	// Flip the last base64 char to something valid but different.
	tampered := good[:len(good)-1]
	if strings.HasSuffix(good, "=") {
		tampered = good[:len(good)-2] + "A="
	} else {
		tampered = good[:len(good)-1] + "A"
	}
	entry := signedEntry(pub, tampered)
	publisher := newPublisher(pub, "", "")

	err := VerifySignature(entry, publisher, time.Now())
	if !errors.Is(err, ErrBadSignature) {
		t.Errorf("err = %v, want ErrBadSignature on tampered signature", err)
	}
}

// ─── Key not in publisher record ──────────────────────────────────────────

func TestVerify_KeyNotRegistered(t *testing.T) {
	// Sign with pubA but publisher only knows pubB.
	pubA, signA := keypair(t)
	pubB, _ := keypair(t)
	entry := signedEntry(pubA, signA([]byte(sampleSHA256)))
	publisher := newPublisher(pubB, "", "") // different key

	err := VerifySignature(entry, publisher, time.Now())
	if !errors.Is(err, ErrNoMatchingKey) {
		t.Errorf("err = %v, want ErrNoMatchingKey", err)
	}
}

// ─── Key expiry / revocation ──────────────────────────────────────────────

func TestVerify_KeyExpired(t *testing.T) {
	pub, sign := keypair(t)
	entry := signedEntry(pub, sign([]byte(sampleSHA256)))
	publisher := newPublisher(pub, "2020-01-01T00:00:00Z", "")

	err := VerifySignature(entry, publisher, time.Now())
	if !errors.Is(err, ErrNoMatchingKey) {
		t.Errorf("err = %v, want ErrNoMatchingKey (key expired)", err)
	}
}

func TestVerify_KeyRevoked(t *testing.T) {
	pub, sign := keypair(t)
	entry := signedEntry(pub, sign([]byte(sampleSHA256)))
	publisher := newPublisher(pub, "", "2025-06-01T00:00:00Z")

	err := VerifySignature(entry, publisher, time.Now())
	if !errors.Is(err, ErrNoMatchingKey) {
		t.Errorf("err = %v, want ErrNoMatchingKey (key revoked)", err)
	}
}

func TestVerify_KeyBeforeExpiry(t *testing.T) {
	// Regression: key is still valid so long as now < ExpiresAt.
	pub, sign := keypair(t)
	entry := signedEntry(pub, sign([]byte(sampleSHA256)))
	future := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)
	publisher := newPublisher(pub, future, "")

	if err := VerifySignature(entry, publisher, time.Now()); err != nil {
		t.Fatalf("VerifySignature: %v", err)
	}
}

// ─── Malformed fields ─────────────────────────────────────────────────────

func TestVerify_BadBase64(t *testing.T) {
	entry := market.Entry{
		SHA256: sampleSHA256,
		Signature: &market.Signature{
			Alg:       "ed25519",
			PublicKey: "!!!not-base64!!!",
			Value:     "also-not-base64",
		},
	}
	publisher := market.PublisherRecord{Name: "acme", Trust: "verified"}

	err := VerifySignature(entry, publisher, time.Now())
	if !errors.Is(err, ErrBadSignature) {
		t.Errorf("err = %v, want ErrBadSignature", err)
	}
}

func TestVerify_WrongKeySize(t *testing.T) {
	// Valid base64 of 10 bytes — not a 32-byte Ed25519 pubkey.
	entry := market.Entry{
		SHA256: sampleSHA256,
		Signature: &market.Signature{
			Alg:       "ed25519",
			PublicKey: base64.StdEncoding.EncodeToString([]byte("short-key!")),
			Value:     base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize)),
		},
	}
	publisher := market.PublisherRecord{Name: "acme", Trust: "verified"}

	err := VerifySignature(entry, publisher, time.Now())
	if !errors.Is(err, ErrBadSignature) || !strings.Contains(err.Error(), "publicKey size") {
		t.Errorf("err = %v, want ErrBadSignature publicKey size", err)
	}
}

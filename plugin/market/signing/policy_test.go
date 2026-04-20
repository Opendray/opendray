package signing

import (
	"errors"
	"testing"
	"time"

	"github.com/opendray/opendray/plugin/market"
)

// validSignedEntry + validPublisher — happy-path shared fixtures.
// Reuses the keypair / signedEntry helpers from verify_test.go.
func validSignedEntry(t *testing.T, trust string) (market.Entry, market.PublisherRecord) {
	t.Helper()
	pub, sign := keypair(t)
	entry := signedEntry(pub, sign([]byte(sampleSHA256)))
	publisher := market.PublisherRecord{
		Name:  "acme",
		Trust: trust,
		Keys: []market.PublisherKey{{
			Alg: "ed25519", PublicKey: pub, AddedAt: "2024-01-01T00:00:00Z",
		}},
	}
	return entry, publisher
}

// ─── Truth table by trust level ────────────────────────────────────────────

func TestPolicy_Official_RequiresValidSignature(t *testing.T) {
	// Valid signature → allowed.
	entry, publisher := validSignedEntry(t, TrustOfficial)
	if err := EnforcePolicy(entry, publisher, time.Now()); err != nil {
		t.Errorf("official + valid sig: got %v, want nil", err)
	}
}

func TestPolicy_Official_MissingSignatureRejected(t *testing.T) {
	entry := market.Entry{SHA256: sampleSHA256} // no Signature
	publisher := market.PublisherRecord{Name: "acme", Trust: TrustOfficial}
	err := EnforcePolicy(entry, publisher, time.Now())
	if !errors.Is(err, ErrSignatureRequired) {
		t.Errorf("official + no sig: err=%v, want ErrSignatureRequired", err)
	}
}

func TestPolicy_Official_BadSignatureRejected(t *testing.T) {
	entry, publisher := validSignedEntry(t, TrustOfficial)
	// Tamper — flip one byte of the signature value.
	if len(entry.Signature.Value) > 1 {
		runes := []byte(entry.Signature.Value)
		if runes[0] == 'A' {
			runes[0] = 'B'
		} else {
			runes[0] = 'A'
		}
		entry.Signature.Value = string(runes)
	}
	err := EnforcePolicy(entry, publisher, time.Now())
	if err == nil {
		t.Error("official + tampered sig: want error, got nil")
	}
}

func TestPolicy_Verified_RequiresValidSignature(t *testing.T) {
	// Same gate as official.
	entry := market.Entry{SHA256: sampleSHA256}
	publisher := market.PublisherRecord{Name: "acme", Trust: TrustVerified}
	err := EnforcePolicy(entry, publisher, time.Now())
	if !errors.Is(err, ErrSignatureRequired) {
		t.Errorf("verified + no sig: err=%v, want ErrSignatureRequired", err)
	}
}

func TestPolicy_Community_AllowsNoSignature(t *testing.T) {
	entry := market.Entry{SHA256: sampleSHA256}
	publisher := market.PublisherRecord{Name: "acme", Trust: TrustCommunity}
	if err := EnforcePolicy(entry, publisher, time.Now()); err != nil {
		t.Errorf("community + no sig: want nil, got %v", err)
	}
}

func TestPolicy_Community_ValidSignatureAllowed(t *testing.T) {
	entry, publisher := validSignedEntry(t, TrustCommunity)
	if err := EnforcePolicy(entry, publisher, time.Now()); err != nil {
		t.Errorf("community + valid sig: got %v, want nil", err)
	}
}

func TestPolicy_Community_BrokenSignatureRejected(t *testing.T) {
	// Even on community, a broken signature is rejected. Prevents
	// an attacker from attaching a garbage signature to take the
	// "optional" off-ramp and slip through.
	entry, publisher := validSignedEntry(t, TrustCommunity)
	entry.Signature.Value = "AAAAAAAA==" // bogus
	err := EnforcePolicy(entry, publisher, time.Now())
	if err == nil {
		t.Error("community + broken sig: want error, got nil")
	}
}

// ─── Unknown trust defaults strictly ──────────────────────────────────────

func TestPolicy_UnknownTrust_DefaultsToRequired(t *testing.T) {
	entry := market.Entry{SHA256: sampleSHA256}
	publisher := market.PublisherRecord{Name: "acme", Trust: "ultra-official"}
	err := EnforcePolicy(entry, publisher, time.Now())
	if !errors.Is(err, ErrSignatureRequired) {
		t.Errorf("unknown trust: err=%v, want ErrSignatureRequired", err)
	}
}

func TestPolicy_EmptyTrust_DefaultsToRequired(t *testing.T) {
	entry := market.Entry{SHA256: sampleSHA256}
	publisher := market.PublisherRecord{Name: "acme"} // Trust=""
	err := EnforcePolicy(entry, publisher, time.Now())
	if !errors.Is(err, ErrSignatureRequired) {
		t.Errorf("empty trust: err=%v, want ErrSignatureRequired", err)
	}
}

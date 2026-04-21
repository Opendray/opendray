package signing

import (
	"errors"
	"fmt"
	"time"

	"github.com/opendray/opendray/plugin/market"
)

// Trust levels from docs/plugin-platform/09-marketplace.md §Trust
// levels. Match the PublisherRecord.Trust wire value verbatim.
const (
	TrustOfficial  = "official"
	TrustVerified  = "verified"
	TrustCommunity = "community"
)

// ErrSignatureRequired signals a policy decision: the publisher's
// trust level mandates a verified signature but the entry shipped
// without one. Hard failure at install time.
var ErrSignatureRequired = errors.New("signing: publisher trust level requires a verified signature")

// EnforcePolicy runs the install-time signature gate. Called
// exactly once per install, after market.Catalog.Resolve has
// filled in the entry and market.Catalog.FetchPublisher has
// produced the publisher record.
//
// Policy table:
//
//	trust=="official"  → signature REQUIRED, must verify.
//	trust=="verified"  → signature REQUIRED, must verify.
//	trust=="community" → signature OPTIONAL.
//	                     - present ⇒ must verify
//	                     - absent  ⇒ allowed
//	trust=="" / unknown → treated strictly (same as "official")
//	                      so a future schema addition can never
//	                      silently downgrade signing posture.
//
// `now` lets tests exercise key expiry without a clock-mock
// package; production callers pass time.Now().
func EnforcePolicy(entry market.Entry, publisher market.PublisherRecord, now time.Time) error {
	required := signatureRequired(publisher.Trust)

	if entry.Signature == nil {
		if required {
			return fmt.Errorf("%w: trust=%s", ErrSignatureRequired, publisher.Trust)
		}
		// Community plugin without a signature — allowed.
		return nil
	}

	// Signature present — always verify regardless of trust level.
	// "verify what you can" prevents a community plugin from
	// attaching a broken signature and slipping through on the
	// optional path.
	if err := VerifySignature(entry, publisher, now); err != nil {
		return err
	}
	return nil
}

// signatureRequired returns true for trust levels that mandate a
// verified signature. Unknown / empty trust levels are treated
// strictly — default-deny keeps the invariant that an incompletely
// configured registry fails closed.
func signatureRequired(trust string) bool {
	switch trust {
	case TrustCommunity:
		return false
	default:
		return true
	}
}

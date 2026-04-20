// Package market is the abstract interface over the OpenDray plugin
// marketplace. Two backends live under this tree:
//
//   - market/local   — reads catalog.json + packages/ from a directory
//   - market/remote  — fetches index.json + per-version JSON from a
//                      registry URL (github.com/Opendray/opendray-marketplace
//                      in production; file:// for tests, localhost HTTP
//                      during M4 rollout).
//
// Both backends produce the same Entry + return the same errors, so
// gateway handlers can accept `market.Catalog` without caring which
// backend is in use.
//
// Entry is the wire shape surfaced by GET /api/marketplace/plugins.
// The field set is the union needed by both backends; backend-specific
// fields (ArtifactURL / SHA256 / Signature) stay zero on local
// catalogs, which the install layer interprets as "bundle is already
// on disk, skip HTTPS fetch".
package market

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/opendray/opendray/plugin"
)

// ErrNotFound signals a ref that doesn't match any catalog entry.
// Callers surface this as HTTP 404.
var ErrNotFound = errors.New("market: entry not found")

// ErrBadRef is returned by ParseRef for malformed marketplace URIs.
var ErrBadRef = errors.New("market: bad ref")

// Entry is one installable plugin advertised by the catalog. The
// JSON tags match docs/plugin-platform/09-marketplace.md index.json
// schema so a remote index entry parses directly into this struct.
type Entry struct {
	// Core identity.
	Name      string `json:"name"`
	Version   string `json:"version"`
	Publisher string `json:"publisher"`

	// Display metadata rendered by the Hub card.
	DisplayName string   `json:"displayName,omitempty"`
	Description string   `json:"description,omitempty"`
	Icon        string   `json:"icon,omitempty"`
	Form        string   `json:"form,omitempty"`
	Tags        []string `json:"tags,omitempty"`

	// Permissions + config schema pulled from the bundled manifest so
	// the Hub can render consent + config dialogs without a second
	// fetch.
	Permissions  json.RawMessage      `json:"permissions,omitempty"`
	ConfigSchema []plugin.ConfigField `json:"configSchema,omitempty"`

	// Trust level surfaced as a badge on the Hub card. One of
	// "official" / "verified" / "community". Local catalogs default
	// to "community" unless the catalog JSON sets it explicitly.
	Trust string `json:"trust,omitempty"`

	// Remote-only fields. Zero on local-backed entries.
	//   ArtifactURL — HTTPS URL the install layer downloads from.
	//   SHA256      — lowercase hex, 64 chars. Verified against the
	//                 downloaded bytes before staging.
	//   Signature   — optional Ed25519 signature + publisher key.
	ArtifactURL string     `json:"artifactUrl,omitempty"`
	SHA256      string     `json:"sha256,omitempty"`
	Signature   *Signature `json:"signature,omitempty"`
}

// Signature is the Ed25519 signature record matching the marketplace
// per-version JSON schema.
type Signature struct {
	Alg       string `json:"alg"`       // "ed25519"
	PublicKey string `json:"publicKey"` // base64
	Value     string `json:"value"`     // base64
}

// Ref is the canonical ref pair the install flow passes around.
// Version may be empty — the catalog resolves "latest" client-side.
type Ref struct {
	Publisher string
	Name      string
	Version   string
}

// String renders a Ref back into the wire form the user types or
// the API accepts.
func (r Ref) String() string {
	if r.Publisher == "" {
		if r.Version == "" {
			return r.Name
		}
		return r.Name + "@" + r.Version
	}
	if r.Version == "" {
		return r.Publisher + "/" + r.Name
	}
	return r.Publisher + "/" + r.Name + "@" + r.Version
}

// Catalog is what both backends implement. Plain methods — no
// context-aware variants yet; the remote backend will thread ctx
// through its HTTP client internally in T2.
type Catalog interface {
	// List returns every entry visible to this catalog. Stable order
	// across calls (publisher, name, version) so clients can diff.
	List(ctx context.Context) ([]Entry, error)

	// Resolve looks up one entry by ref. An empty Version picks the
	// latest the catalog knows about. ErrNotFound when nothing matches.
	Resolve(ctx context.Context, ref Ref) (Entry, error)

	// BundlePath returns the on-disk location of the bundle for
	// entries the backend can serve locally. Returns ("", false) when
	// the backend can only produce the artifact URL — the install
	// layer then downloads via HTTPSSource instead of copying.
	BundlePath(ctx context.Context, ref Ref) (string, bool, error)
}

// ParseRef splits a marketplace reference into a Ref.
//
// Accepted shapes:
//
//	"foo"                           → {"", "foo", ""}
//	"foo@1.0.0"                     → {"", "foo", "1.0.0"}
//	"publisher/name"                → {"publisher", "name", ""}
//	"publisher/name@1.0.0"          → {"publisher", "name", "1.0.0"}
//	"marketplace://..."             → same as above, with the scheme peeled
//
// Rejects empty strings, path separators (beyond the single
// publisher/name split), and embedded "..".
func ParseRef(ref string) (Ref, error) {
	raw := strings.TrimSpace(ref)
	if raw == "" {
		return Ref{}, fmt.Errorf("%w: empty ref", ErrBadRef)
	}
	raw = strings.TrimPrefix(raw, "marketplace://")

	// Split the optional @version first so the name is easy to
	// inspect for the publisher/name split.
	var version string
	if at := strings.IndexByte(raw, '@'); at >= 0 {
		version = raw[at+1:]
		raw = raw[:at]
	}

	if raw == "" {
		return Ref{}, fmt.Errorf("%w: missing name", ErrBadRef)
	}
	if strings.Contains(raw, `\`) || strings.Contains(raw, "..") {
		return Ref{}, fmt.Errorf("%w: unsafe chars in %q", ErrBadRef, ref)
	}

	// publisher/name — one slash maximum. Bare name (no slash) means
	// publisher is unresolved; callers can default-fill
	// "opendray-examples" for M3 back-compat.
	parts := strings.Split(raw, "/")
	switch len(parts) {
	case 1:
		return Ref{Name: parts[0], Version: version}, nil
	case 2:
		if parts[0] == "" || parts[1] == "" {
			return Ref{}, fmt.Errorf("%w: empty publisher or name in %q", ErrBadRef, ref)
		}
		return Ref{Publisher: parts[0], Name: parts[1], Version: version}, nil
	default:
		return Ref{}, fmt.Errorf("%w: too many '/' in %q", ErrBadRef, ref)
	}
}

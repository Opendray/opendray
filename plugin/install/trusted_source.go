package install

import "context"

// TrustedSource is a local-filesystem bundle source that bypasses the
// AllowLocal gate in Stage. Use it for trusted intra-host paths — e.g.
// a marketplace entry whose on-disk location the server itself resolved
// from its catalog. Client-supplied `local:` / absolute-path sources
// must continue to flow through LocalSource so the AllowLocal check
// still fires.
//
// Fetch behaves identically to LocalSource.Fetch: the directory at
// Path is copied into a fresh os.MkdirTemp staging dir.
type TrustedSource struct {
	// Path is the absolute path to a bundle directory containing
	// manifest.json at its root.
	Path string
	// Label, when non-empty, is used by Describe() instead of the
	// raw path so audit logs don't leak user-filesystem layout. For a
	// catalog-resolved bundle this is typically "marketplace://<name>@<version>".
	Label string
}

// Fetch copies the directory at s.Path into a fresh staging dir.
func (s TrustedSource) Fetch(ctx context.Context) (string, func(), error) {
	return LocalSource{Path: s.Path}.Fetch(ctx)
}

// Describe returns a human-readable label used for audit hashing.
// Prefer Label when set — it keeps audit logs marketplace-identifier-
// shaped rather than filesystem-shaped.
func (s TrustedSource) Describe() string {
	if s.Label != "" {
		return s.Label
	}
	return "trusted:" + s.Path
}

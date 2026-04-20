// Package remote implements [market.Catalog] against an HTTP(S)
// registry URL.
//
// The concrete fetch / verify / cache logic lands in T2–T7 of the
// M4-PLAN. This file currently ships the type skeleton and returns
// ErrNotImplemented on every method so callers can swap it in at
// construction time and validate the plumbing before any real
// network I/O exists.
package remote

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/opendray/opendray/plugin/market"
)

// ErrNotImplemented signals a method that hasn't been filled in yet.
// T2 (FetchIndex) replaces the List error site; T3 (FetchVersion)
// replaces Resolve; T4 (HTTPSSource) is what BundlePath delegates to
// via the install layer rather than returning a path directly.
var ErrNotImplemented = errors.New("market/remote: not implemented yet")

// Config carries the knobs the remote backend needs. Reasonable
// defaults mean callers can leave most of it zero during M4.1
// iteration.
type Config struct {
	// RegistryURL is the base from which index.json + per-version
	// files resolve. For launch: the GitHub raw URL; later the CDN.
	// Must be absolute; trailing slash optional.
	RegistryURL string

	// Mirrors is an ordered list of fallback base URLs used when the
	// primary returns 5xx / times out. Populated in T6.
	Mirrors []string

	// HTTPClient overrides the default client. Zero uses a sane
	// 10-second timeout. Tests inject a client that hits a
	// httptest.Server.
	HTTPClient *http.Client

	// CacheDir is where fetched index + per-version JSON live on
	// disk for stale-while-revalidate. Zero disables disk caching.
	// Filled by T7.
	CacheDir string
}

// Catalog is the remote-backed implementation of [market.Catalog].
// It is safe to construct without network access; network calls
// happen on List / Resolve / BundlePath as needed.
type Catalog struct {
	cfg    Config
	client *http.Client
}

// New constructs a remote Catalog from cfg. A zero Config is
// rejected — the RegistryURL is mandatory so the caller can't
// silently fall through to an empty catalog.
func New(cfg Config) (*Catalog, error) {
	if cfg.RegistryURL == "" {
		return nil, errors.New("market/remote: RegistryURL is required")
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &Catalog{cfg: cfg, client: client}, nil
}

// List implements market.Catalog.
func (c *Catalog) List(_ context.Context) ([]market.Entry, error) {
	return nil, ErrNotImplemented
}

// Resolve implements market.Catalog.
func (c *Catalog) Resolve(_ context.Context, _ market.Ref) (market.Entry, error) {
	return market.Entry{}, ErrNotImplemented
}

// BundlePath implements market.Catalog. Remote-backed catalogs
// never hand out a local path — the install layer fetches via
// HTTPSSource using Entry.ArtifactURL + Entry.SHA256. Returning
// ("", false, nil) signals exactly that.
func (c *Catalog) BundlePath(_ context.Context, _ market.Ref) (string, bool, error) {
	return "", false, nil
}

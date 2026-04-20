// Package remote implements [market.Catalog] against an HTTP(S)
// registry URL — production is Opendray/opendray-marketplace via
// GitHub raw URLs.
//
// Wire contract: docs/plugin-platform/09-marketplace.md + the
// JSON Schemas at schemas/ in the marketplace repo. This file
// stays in lockstep with those schemas; any field change requires
// a matching update on both ends.
//
// Fills in progressively:
//
//	T2 (this commit) — List via index.json
//	T3               — Resolve via per-version JSON
//	T4               — BundlePath via HTTPSSource download
//	T5               — Ed25519 signature verification
//	T6               — mirror fallback round-robin
//	T7               — on-disk cache + stale-while-revalidate
package remote

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/opendray/opendray/plugin/market"
)

// ErrNotImplemented signals a method that hasn't been filled in yet.
var ErrNotImplemented = errors.New("market/remote: not implemented yet")

// maxIndexBytes caps the index.json download so a pathological
// registry can't eat unbounded memory. 8 MiB fits thousands of
// summary entries with room to spare.
const maxIndexBytes = 8 << 20

// defaultHTTPTimeout is the per-request ceiling. Applied to the
// http.Client when the caller doesn't supply their own.
const defaultHTTPTimeout = 10 * time.Second

// Config carries the knobs the remote backend needs. Reasonable
// defaults mean callers can leave most of it zero during M4.1
// iteration.
type Config struct {
	// RegistryURL is the base from which index.json + per-version
	// files resolve. Must be absolute; trailing slash optional.
	// For launch:
	//   https://raw.githubusercontent.com/Opendray/opendray-marketplace/main/
	// Later: CDN front-door at https://marketplace.opendray.dev/ .
	RegistryURL string

	// Mirrors is an ordered list of fallback base URLs used when
	// the primary returns 5xx / times out. Populated in T6.
	Mirrors []string

	// HTTPClient overrides the default client. Zero uses a 10-second
	// timeout. Tests inject a client that hits a httptest.Server.
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
	base   *url.URL
}

// New constructs a remote Catalog from cfg. A zero Config is
// rejected — the RegistryURL is mandatory so the caller can't
// silently fall through to an empty catalog.
func New(cfg Config) (*Catalog, error) {
	if cfg.RegistryURL == "" {
		return nil, errors.New("market/remote: RegistryURL is required")
	}
	base, err := url.Parse(cfg.RegistryURL)
	if err != nil {
		return nil, fmt.Errorf("market/remote: parse RegistryURL: %w", err)
	}
	if base.Scheme != "http" && base.Scheme != "https" {
		return nil, fmt.Errorf("market/remote: RegistryURL scheme must be http(s), got %q", base.Scheme)
	}
	if !strings.HasSuffix(base.Path, "/") {
		base.Path += "/"
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: defaultHTTPTimeout}
	}
	return &Catalog{cfg: cfg, client: client, base: base}, nil
}

// indexResponse mirrors the wire shape the marketplace repo's
// publish.yml emits (schemas/index.schema.json).
type indexResponse struct {
	Version     int              `json:"version"`
	GeneratedAt string           `json:"generatedAt"`
	Plugins     []indexPluginRow `json:"plugins"`
}

// indexPluginRow is one entry from index.json — a SUMMARY view.
// Full data (permissions, configSchema, artifactUrl, sha256,
// signature) lives in the per-version JSON fetched by Resolve
// (T3), not here.
type indexPluginRow struct {
	Name        string   `json:"name"`
	Publisher   string   `json:"publisher"`
	DisplayName string   `json:"displayName,omitempty"`
	Description string   `json:"description,omitempty"`
	Icon        string   `json:"icon,omitempty"`
	Form        string   `json:"form,omitempty"`
	Categories  []string `json:"categories,omitempty"`
	Keywords    []string `json:"keywords,omitempty"`
	Latest      string   `json:"latest"`
	Path        string   `json:"path,omitempty"`
	Trust       string   `json:"trust,omitempty"`
	Downloads   int      `json:"downloads,omitempty"`
	Stars       int      `json:"stars,omitempty"`
}

// List implements market.Catalog. Performs a GET against
// `{RegistryURL}/index.json`, validates the response shape, and
// maps every plugin row into a market.Entry. Permissions,
// ConfigSchema, ArtifactURL, SHA256, and Signature stay empty on
// summary entries — those are filled when the install flow calls
// Resolve (T3).
func (c *Catalog) List(ctx context.Context) ([]market.Entry, error) {
	u, err := c.resolveRelative("index.json")
	if err != nil {
		return nil, err
	}

	body, err := c.fetch(ctx, u, maxIndexBytes)
	if err != nil {
		return nil, fmt.Errorf("market/remote: fetch index: %w", err)
	}
	var idx indexResponse
	if err := json.Unmarshal(body, &idx); err != nil {
		return nil, fmt.Errorf("market/remote: parse index: %w", err)
	}
	if idx.Version != 1 {
		return nil, fmt.Errorf("market/remote: unsupported index version %d", idx.Version)
	}

	out := make([]market.Entry, 0, len(idx.Plugins))
	for i, p := range idx.Plugins {
		if p.Name == "" || p.Publisher == "" || p.Latest == "" {
			// Drop rows with missing critical fields rather than
			// failing the whole list — a single bad row shouldn't
			// break the Hub. Log via errorf-wrap at T7 when we add
			// a caller-visible logger.
			return nil, fmt.Errorf("market/remote: index row %d missing name/publisher/latest", i)
		}
		trust := p.Trust
		if trust == "" {
			trust = "community"
		}
		out = append(out, market.Entry{
			Name:        p.Name,
			Publisher:   p.Publisher,
			Version:     p.Latest,
			DisplayName: p.DisplayName,
			Description: p.Description,
			Icon:        p.Icon,
			Form:        p.Form,
			Tags:        mergeTags(p.Categories, p.Keywords),
			Trust:       trust,
			// Permissions / ConfigSchema / ArtifactURL / SHA256 /
			// Signature intentionally unset on summary entries.
			// Resolve (T3) fills them by fetching the per-version
			// JSON when a caller actually needs them.
		})
	}
	return out, nil
}

// mergeTags concatenates categories + keywords into a single
// de-duplicated ordered slice for the Hub card's tag chip.
func mergeTags(a, b []string) []string {
	if len(a) == 0 && len(b) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(a)+len(b))
	out := make([]string, 0, len(a)+len(b))
	for _, s := range a {
		if _, ok := seen[s]; ok || s == "" {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	for _, s := range b {
		if _, ok := seen[s]; ok || s == "" {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

// Resolve implements market.Catalog. Placeholder until T3 lands
// FetchVersion + Ed25519 verification.
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

// ─── helpers ────────────────────────────────────────────────────────────────

// resolveRelative joins a relative path onto the configured base
// URL without losing its trailing slash semantics. Rejects absolute
// paths so callers can't escape the registry root.
func (c *Catalog) resolveRelative(rel string) (*url.URL, error) {
	if strings.HasPrefix(rel, "/") {
		return nil, fmt.Errorf("market/remote: relative path must not start with /, got %q", rel)
	}
	ref, err := url.Parse(rel)
	if err != nil {
		return nil, fmt.Errorf("market/remote: parse relative %q: %w", rel, err)
	}
	return c.base.ResolveReference(ref), nil
}

// fetch performs a bounded GET. Caps the downloaded size at maxBytes
// and requires an application/json-ish Content-Type. Returns the
// body bytes verbatim so callers can json.Unmarshal into their own
// type.
func (c *Catalog) fetch(ctx context.Context, u *url.URL, maxBytes int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json, application/vnd.github.v3+json;q=0.9")
	req.Header.Set("User-Agent", "opendray/market-remote")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("registry GET %s: HTTP %d", u.String(), resp.StatusCode)
	}

	// Accept "application/json", "text/plain" (GitHub raw serves
	// the latter for .json), "application/octet-stream" — anything
	// that isn't explicitly HTML. An HTML body at a raw URL means
	// an IdP redirect or error page; refuse to parse it as JSON.
	ct := strings.ToLower(strings.TrimSpace(resp.Header.Get("Content-Type")))
	if strings.HasPrefix(ct, "text/html") {
		return nil, fmt.Errorf("registry GET %s: unexpected text/html response", u.String())
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > maxBytes {
		return nil, fmt.Errorf("registry GET %s: body exceeds %d bytes", u.String(), maxBytes)
	}
	return body, nil
}

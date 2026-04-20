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
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/opendray/opendray/plugin"
	"github.com/opendray/opendray/plugin/market"
)

// ErrNotImplemented signals a method that hasn't been filled in yet.
var ErrNotImplemented = errors.New("market/remote: not implemented yet")

// maxIndexBytes caps the index.json download so a pathological
// registry can't eat unbounded memory. 8 MiB fits thousands of
// summary entries with room to spare.
const maxIndexBytes = 8 << 20

// maxVersionBytes caps per-version JSON downloads. These carry a
// full manifest copy but no binary; 2 MiB is plenty and catches
// obvious padding attacks.
const maxVersionBytes = 2 << 20

// defaultPublisher fills in the Publisher field when callers pass
// a bare-name ref like `marketplace://fs-readme`. Keeps the M3
// install URLs working during the M4 transition without forcing
// every client to adopt publisher/name notation up front.
const defaultPublisher = "opendray-examples"

// sha256HexRE is the lowercase-hex-64-char shape the registry
// schema enforces on `sha256` fields. Validated before we hand
// the value to the install layer so a malformed registry entry
// trips here rather than inside the HTTPSSource verifier.
var sha256HexRE = regexp.MustCompile(`^[a-f0-9]{64}$`)

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

	// CacheDir is where fetched index + per-version JSON will live
	// on disk once the post-v1 stale-while-revalidate layer lands.
	// Zero disables disk caching. Memory-only TTL cache is always on.
	CacheDir string

	// CacheTTL controls the in-memory response cache. Zero uses
	// defaultCacheTTL (5 min). Set to -1 to disable caching (tests
	// that need every call to hit the network).
	CacheTTL time.Duration
}

// defaultCacheTTL bounds how long a fetched registry JSON stays
// warm in memory before we re-fetch. Short enough that a revoked
// plugin's revocations.json update propagates within a few
// minutes; long enough to spare GitHub raw from per-app-launch
// traffic.
const defaultCacheTTL = 5 * time.Minute

// HTTPStatusError is returned by fetch for any non-2xx response.
// The status code lets the retry loop decide whether to try the
// next mirror (5xx = yes; 4xx = no, it's definitive).
type HTTPStatusError struct {
	Status int
	URL    string
}

func (e *HTTPStatusError) Error() string {
	return fmt.Sprintf("registry GET %s: HTTP %d", e.URL, e.Status)
}

// Retryable returns true for server-side failures (5xx). 4xx are
// definitive — 404 means the resource genuinely isn't there, and
// retrying against mirrors won't change that.
func (e *HTTPStatusError) Retryable() bool {
	return e.Status >= 500 && e.Status <= 599
}

// Catalog is the remote-backed implementation of [market.Catalog].
// It is safe to construct without network access; network calls
// happen on List / Resolve / BundlePath as needed.
type Catalog struct {
	cfg    Config
	client *http.Client
	// bases is the primary RegistryURL followed by the configured
	// mirrors, all normalised to a trailing slash. The fetch
	// helper iterates this in order on retryable failures.
	bases []*url.URL

	// ttl is the cache window; 0 = use defaultCacheTTL; <0 disables.
	ttl time.Duration
	// cache is keyed on the relative path (e.g. "index.json",
	// "publishers/acme.json") — no base URL differentiation, since
	// mirror fallback surfaces through fetch and the cached bytes
	// are identical across mirrors.
	cacheMu sync.RWMutex
	cache   map[string]cacheEntry
}

type cacheEntry struct {
	body    []byte
	expires time.Time
}

// New constructs a remote Catalog from cfg. A zero Config is
// rejected — the RegistryURL is mandatory so the caller can't
// silently fall through to an empty catalog.
func New(cfg Config) (*Catalog, error) {
	if cfg.RegistryURL == "" {
		return nil, errors.New("market/remote: RegistryURL is required")
	}
	primary, err := parseBase(cfg.RegistryURL)
	if err != nil {
		return nil, fmt.Errorf("market/remote: parse RegistryURL: %w", err)
	}
	bases := []*url.URL{primary}
	for i, m := range cfg.Mirrors {
		b, err := parseBase(m)
		if err != nil {
			return nil, fmt.Errorf("market/remote: parse Mirrors[%d]: %w", i, err)
		}
		bases = append(bases, b)
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: defaultHTTPTimeout}
	}
	ttl := cfg.CacheTTL
	if ttl == 0 {
		ttl = defaultCacheTTL
	}
	return &Catalog{
		cfg:    cfg,
		client: client,
		bases:  bases,
		ttl:    ttl,
		cache:  make(map[string]cacheEntry),
	}, nil
}

// cacheLookup returns cached bytes if the entry is still fresh.
// Returns (nil, false) when the cache is disabled (ttl < 0), the
// key is absent, or the entry has expired.
func (c *Catalog) cacheLookup(rel string) ([]byte, bool) {
	if c.ttl < 0 {
		return nil, false
	}
	c.cacheMu.RLock()
	defer c.cacheMu.RUnlock()
	e, ok := c.cache[rel]
	if !ok {
		return nil, false
	}
	if time.Now().After(e.expires) {
		return nil, false
	}
	return e.body, true
}

// cacheStore records a freshly-fetched body. No-op when caching is
// disabled. Expired entries are evicted lazily on cacheLookup.
func (c *Catalog) cacheStore(rel string, body []byte) {
	if c.ttl < 0 {
		return
	}
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	c.cache[rel] = cacheEntry{
		body:    append([]byte(nil), body...),
		expires: time.Now().Add(c.ttl),
	}
}

// InvalidateCache drops every cached entry. Used by the revocation
// poller (T8) and the "Refresh cache now" button (T12) so a fresh
// network fetch runs on the next List / Resolve.
func (c *Catalog) InvalidateCache() {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	c.cache = make(map[string]cacheEntry)
}

// parseBase parses one base URL and guarantees a trailing slash
// for clean url.ResolveReference semantics.
func parseBase(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("scheme must be http(s), got %q", u.Scheme)
	}
	if !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}
	return u, nil
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
	body, err := c.fetch(ctx, "index.json", maxIndexBytes)
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

// versionResponse mirrors the marketplace version-JSON wire shape
// (schemas/version.schema.json in the marketplace repo).
type versionResponse struct {
	Name         string           `json:"name"`
	Publisher    string           `json:"publisher"`
	Version      string           `json:"version"`
	ReleaseNotes string           `json:"releaseNotes,omitempty"`
	Artifact     versionArtifact  `json:"artifact"`
	SHA256       string           `json:"sha256"`
	Signature    *market.Signature `json:"signature,omitempty"`
	Manifest     versionManifest  `json:"manifest"`
	Engines      map[string]any   `json:"engines,omitempty"`
	Platforms    []string         `json:"platforms,omitempty"`
}

type versionArtifact struct {
	URL     string   `json:"url"`
	Size    int64    `json:"size"`
	Mirrors []string `json:"mirrors,omitempty"`
}

// versionManifest is the subset of the full plugin manifest that
// Resolve needs to surface to the Hub (for consent + config
// dialogs). The manifest is accepted as a `map` first so we don't
// tightly couple every manifest-schema addition to this package;
// the fields we render are pulled explicitly.
type versionManifest struct {
	DisplayName  string          `json:"displayName,omitempty"`
	Description  string          `json:"description,omitempty"`
	Icon         string          `json:"icon,omitempty"`
	Form         string          `json:"form,omitempty"`
	Permissions  json.RawMessage `json:"permissions,omitempty"`
	ConfigSchema []plugin.ConfigField `json:"configSchema,omitempty"`
}

// Resolve implements market.Catalog. Fetches the per-version JSON
// for the ref (discovering `latest` via index.json when the ref
// doesn't pin a version), validates wire-level invariants, and
// returns a fully-populated Entry.
//
// SHA-256 verification against the actual artifact bytes happens
// in the install layer (T4), not here — Resolve only validates
// that `sha256` is 64-char lowercase hex so a malformed registry
// entry fails at the registry boundary.
//
// Signatures (T5) are returned verbatim; the verifier sits in
// the install layer too so policy ("official/verified must sign")
// applies at the one decision point.
func (c *Catalog) Resolve(ctx context.Context, ref market.Ref) (market.Entry, error) {
	if ref.Publisher == "" {
		ref.Publisher = defaultPublisher
	}
	if ref.Name == "" {
		return market.Entry{}, fmt.Errorf("%w: empty name", market.ErrBadRef)
	}

	// If no version pinned, discover latest via the index. List()
	// is cheap — one extra HTTP call — and keeps the state
	// centralised in the generated index rather than requiring a
	// second meta.json endpoint.
	if ref.Version == "" {
		entries, err := c.List(ctx)
		if err != nil {
			return market.Entry{}, fmt.Errorf("market/remote: resolve latest: %w", err)
		}
		for _, e := range entries {
			if e.Publisher == ref.Publisher && e.Name == ref.Name {
				ref.Version = e.Version
				break
			}
		}
		if ref.Version == "" {
			return market.Entry{}, fmt.Errorf("%w: %s/%s", market.ErrNotFound, ref.Publisher, ref.Name)
		}
	}

	rel := fmt.Sprintf("plugins/%s/%s/%s.json", ref.Publisher, ref.Name, ref.Version)
	body, err := c.fetch(ctx, rel, maxVersionBytes)
	if err != nil {
		var hs *HTTPStatusError
		if errors.As(err, &hs) && hs.Status == http.StatusNotFound {
			return market.Entry{}, fmt.Errorf("%w: %s", market.ErrNotFound, ref)
		}
		return market.Entry{}, fmt.Errorf("market/remote: fetch version: %w", err)
	}
	var v versionResponse
	if err := json.Unmarshal(body, &v); err != nil {
		return market.Entry{}, fmt.Errorf("market/remote: parse version: %w", err)
	}

	// Cross-check: the filename path MUST match the body's
	// (publisher, name, version). Catches registry-side typos or
	// copy-paste errors that otherwise silently ship the wrong
	// manifest to clients.
	if v.Name != ref.Name || v.Publisher != ref.Publisher || v.Version != ref.Version {
		return market.Entry{}, fmt.Errorf("market/remote: version body %s/%s@%s doesn't match URL %s/%s@%s",
			v.Publisher, v.Name, v.Version, ref.Publisher, ref.Name, ref.Version)
	}
	if v.Artifact.URL == "" || v.Artifact.Size <= 0 {
		return market.Entry{}, fmt.Errorf("market/remote: %s missing artifact url or size", ref)
	}
	if !sha256HexRE.MatchString(v.SHA256) {
		return market.Entry{}, fmt.Errorf("market/remote: %s sha256 malformed %q", ref, v.SHA256)
	}

	return market.Entry{
		Name:         v.Name,
		Publisher:    v.Publisher,
		Version:      v.Version,
		DisplayName:  v.Manifest.DisplayName,
		Description:  v.Manifest.Description,
		Icon:         v.Manifest.Icon,
		Form:         v.Manifest.Form,
		Permissions:  v.Manifest.Permissions,
		ConfigSchema: v.Manifest.ConfigSchema,
		Trust:        "community", // T10 fills from publisher record
		ArtifactURL:  v.Artifact.URL,
		SHA256:       v.SHA256,
		Signature:    v.Signature,
	}, nil
}

// BundlePath implements market.Catalog. Remote-backed catalogs
// never hand out a local path — the install layer fetches via
// HTTPSSource using Entry.ArtifactURL + Entry.SHA256. Returning
// ("", false, nil) signals exactly that.
func (c *Catalog) BundlePath(_ context.Context, _ market.Ref) (string, bool, error) {
	return "", false, nil
}

// FetchRevocations implements market.Catalog. Pulls the
// revocations.json file from the registry. A 404 surfaces as
// (nil, nil) — an empty revocations list is the normal case; the
// marketplace repo ships revocations.json from day one but a
// mirror without it shouldn't cause a hard error.
func (c *Catalog) FetchRevocations(ctx context.Context) ([]byte, error) {
	body, err := c.fetch(ctx, "revocations.json", maxVersionBytes)
	if err != nil {
		var hs *HTTPStatusError
		if errors.As(err, &hs) && hs.Status == http.StatusNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("market/remote: fetch revocations: %w", err)
	}
	return body, nil
}

// FetchPublisher implements market.Catalog. Pulls
// publishers/<publisher>.json from the registry. A 404 surfaces
// as market.ErrNotFound so callers can branch cleanly.
func (c *Catalog) FetchPublisher(ctx context.Context, publisher string) (market.PublisherRecord, error) {
	if publisher == "" {
		return market.PublisherRecord{}, fmt.Errorf("%w: empty publisher", market.ErrBadRef)
	}
	body, err := c.fetch(ctx, "publishers/"+publisher+".json", maxVersionBytes)
	if err != nil {
		var hs *HTTPStatusError
		if errors.As(err, &hs) && hs.Status == http.StatusNotFound {
			return market.PublisherRecord{}, fmt.Errorf("%w: publisher %q", market.ErrNotFound, publisher)
		}
		return market.PublisherRecord{}, fmt.Errorf("market/remote: fetch publisher: %w", err)
	}
	var rec market.PublisherRecord
	if err := json.Unmarshal(body, &rec); err != nil {
		return market.PublisherRecord{}, fmt.Errorf("market/remote: parse publisher: %w", err)
	}
	if rec.Name == "" {
		rec.Name = publisher
	}
	if rec.Trust == "" {
		rec.Trust = "community"
	}
	return rec, nil
}

// ─── helpers ────────────────────────────────────────────────────────────────

// resolveAt joins rel onto the given base, preserving trailing
// slash semantics. Rejects absolute paths so callers can't escape
// the registry root.
func resolveAt(base *url.URL, rel string) (*url.URL, error) {
	if strings.HasPrefix(rel, "/") {
		return nil, fmt.Errorf("market/remote: relative path must not start with /, got %q", rel)
	}
	ref, err := url.Parse(rel)
	if err != nil {
		return nil, fmt.Errorf("market/remote: parse relative %q: %w", rel, err)
	}
	return base.ResolveReference(ref), nil
}

// fetch performs a bounded GET of the given relative path, falling
// back through the configured mirrors on retryable failures (5xx,
// network errors, timeouts). 4xx errors — especially 404 — are
// definitive and short-circuit the loop so the caller surfaces
// "not found" rather than trying mirrors that will all 404 too.
//
// Cache-first: serves recent hits straight from memory so repeated
// List / Resolve / FetchPublisher calls within the CacheTTL window
// don't retouch the network. Cache is populated only by a
// successful full-size response; 4xx / 5xx never poison the cache.
func (c *Catalog) fetch(ctx context.Context, rel string, maxBytes int64) ([]byte, error) {
	if body, ok := c.cacheLookup(rel); ok {
		return body, nil
	}

	var lastErr error
	for i, base := range c.bases {
		u, err := resolveAt(base, rel)
		if err != nil {
			return nil, err
		}
		body, err := c.fetchOne(ctx, u, maxBytes)
		if err == nil {
			c.cacheStore(rel, body)
			return body, nil
		}
		lastErr = err

		// Don't burn through the mirror list on caller-initiated
		// cancellation.
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		// Definitive 4xx — the path genuinely isn't there. Surface
		// it directly so callers (Resolve / FetchPublisher) can
		// map to ErrNotFound.
		var hs *HTTPStatusError
		if errors.As(err, &hs) && !hs.Retryable() {
			return nil, err
		}
		// Retryable — advance to next mirror if one exists.
		if i+1 < len(c.bases) {
			continue
		}
	}
	return nil, lastErr
}

// fetchOne performs a single bounded GET. Caps the downloaded size
// at maxBytes, rejects HTML responses (GitHub raw's rate-limit /
// auth pages return HTML on .json paths), and returns bytes
// verbatim so callers json.Unmarshal into their own type.
func (c *Catalog) fetchOne(ctx context.Context, u *url.URL, maxBytes int64) ([]byte, error) {
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
		return nil, &HTTPStatusError{Status: resp.StatusCode, URL: u.String()}
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

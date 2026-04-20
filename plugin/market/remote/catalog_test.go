package remote

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/opendray/opendray/plugin/market"
)

// newTestCatalog spins up a httptest.Server with the given handler
// and returns a Catalog wired to it. Keeps tests terse and ensures
// every server is cleaned up via t.Cleanup.
func newTestCatalog(t *testing.T, handler http.HandlerFunc) *Catalog {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c, err := New(Config{RegistryURL: srv.URL, HTTPClient: srv.Client()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

// indexHandler responds with body on GET /index.json and 404 on
// every other path. Content-Type defaults to application/json;
// callers can override for the content-type tests.
func indexHandler(body string, opts ...func(http.Header)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/index.json" {
			http.NotFound(w, r)
			return
		}
		h := w.Header()
		h.Set("Content-Type", "application/json")
		for _, f := range opts {
			f(h)
		}
		_, _ = fmt.Fprint(w, body)
	}
}

// ─── Construction ───────────────────────────────────────────────────────────

func TestNew_Rejects(t *testing.T) {
	cases := []struct {
		name   string
		cfg    Config
		errSub string
	}{
		{"empty", Config{}, "RegistryURL is required"},
		{"badScheme", Config{RegistryURL: "ftp://example"}, "scheme must be http"},
		{"unparseable", Config{RegistryURL: "://:bad"}, "parse RegistryURL"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := New(tc.cfg)
			if err == nil || !strings.Contains(err.Error(), tc.errSub) {
				t.Errorf("err = %v; want contains %q", err, tc.errSub)
			}
		})
	}
}

func TestNew_NormalisesTrailingSlash(t *testing.T) {
	c, err := New(Config{RegistryURL: "https://example.com/marketplace"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(c.bases[0].Path, "/") {
		t.Errorf("base.Path = %q, want trailing slash", c.bases[0].Path)
	}
}

// ─── List — happy path ──────────────────────────────────────────────────────

func TestList_HappyPath(t *testing.T) {
	c := newTestCatalog(t, indexHandler(`{
		"version": 1,
		"generatedAt": "2026-04-20T00:00:00Z",
		"plugins": [
			{
				"name": "fs-readme",
				"publisher": "opendray-examples",
				"displayName": "FS Readme",
				"description": "reads README",
				"icon": "📖",
				"form": "host",
				"categories": ["examples"],
				"keywords": ["reference", "sidecar"],
				"latest": "1.0.0",
				"path": "plugins/opendray-examples/fs-readme",
				"trust": "official"
			},
			{
				"name": "alpha",
				"publisher": "acme",
				"latest": "2.1.0"
			}
		]
	}`))

	entries, err := c.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len=%d want 2", len(entries))
	}

	// First row: fully populated.
	e := entries[0]
	if e.Name != "fs-readme" || e.Publisher != "opendray-examples" || e.Version != "1.0.0" {
		t.Errorf("entry[0] identity = %+v", e)
	}
	if e.Form != "host" || e.Trust != "official" {
		t.Errorf("entry[0] form/trust = %q/%q", e.Form, e.Trust)
	}
	// Tags = categories ∪ keywords, deduped, ordered.
	wantTags := []string{"examples", "reference", "sidecar"}
	if len(e.Tags) != len(wantTags) {
		t.Errorf("entry[0].Tags = %v, want %v", e.Tags, wantTags)
	}
	for i, got := range e.Tags {
		if got != wantTags[i] {
			t.Errorf("entry[0].Tags[%d] = %q, want %q", i, got, wantTags[i])
		}
	}
	// Summary entries never carry the full-detail fields; those come from Resolve.
	if len(e.Permissions) != 0 {
		t.Errorf("entry[0].Permissions = %s; summary should be empty", e.Permissions)
	}
	if e.ArtifactURL != "" || e.SHA256 != "" {
		t.Errorf("entry[0] artifact/sha256 leaked into summary: %+v", e)
	}

	// Second row: default trust fills in.
	if entries[1].Trust != "community" {
		t.Errorf("entry[1].Trust = %q, want community (default)", entries[1].Trust)
	}
}

// ─── List — error paths ─────────────────────────────────────────────────────

func TestList_NotFound(t *testing.T) {
	c := newTestCatalog(t, func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	if _, err := c.List(context.Background()); err == nil {
		t.Fatal("want error on 404")
	} else if !strings.Contains(err.Error(), "HTTP 404") {
		t.Errorf("err = %v; want HTTP 404", err)
	}
}

func TestList_MalformedJSON(t *testing.T) {
	c := newTestCatalog(t, indexHandler(`{not json`))
	if _, err := c.List(context.Background()); err == nil {
		t.Fatal("want error on malformed JSON")
	} else if !strings.Contains(err.Error(), "parse index") {
		t.Errorf("err = %v; want parse index error", err)
	}
}

func TestList_UnsupportedVersion(t *testing.T) {
	c := newTestCatalog(t, indexHandler(`{"version":99,"generatedAt":"","plugins":[]}`))
	if _, err := c.List(context.Background()); err == nil {
		t.Fatal("want error on version mismatch")
	} else if !strings.Contains(err.Error(), "unsupported index version") {
		t.Errorf("err = %v; want unsupported version error", err)
	}
}

func TestList_MissingFields(t *testing.T) {
	c := newTestCatalog(t, indexHandler(`{
		"version":1,
		"generatedAt":"",
		"plugins":[{"name":"","publisher":"acme","latest":"1.0.0"}]
	}`))
	if _, err := c.List(context.Background()); err == nil {
		t.Fatal("want error on row with empty name")
	}
}

func TestList_RejectsHTMLResponse(t *testing.T) {
	// GitHub raw occasionally serves HTML when the URL is wrong
	// (rate-limited, auth page, etc). Refuse to parse those as JSON.
	c := newTestCatalog(t, indexHandler(`<html><body>rate limited</body></html>`,
		func(h http.Header) { h.Set("Content-Type", "text/html; charset=utf-8") }))
	if _, err := c.List(context.Background()); err == nil {
		t.Fatal("want error on html response")
	} else if !strings.Contains(err.Error(), "text/html") {
		t.Errorf("err = %v; want text/html refusal", err)
	}
}

func TestList_BodySizeCap(t *testing.T) {
	// Serve a body larger than maxIndexBytes. We don't need a
	// full 8 MiB response — patch the cap via a one-off Catalog
	// built against a small limit. The real cap is in maxIndexBytes
	// (const); here we rely on the same code path refusing at the
	// configured ceiling by feeding a body one byte over.
	big := strings.Repeat("x", maxIndexBytes+1)
	c := newTestCatalog(t, indexHandler(big))
	if _, err := c.List(context.Background()); err == nil {
		t.Fatal("want error on oversized body")
	} else if !strings.Contains(err.Error(), "exceeds") {
		t.Errorf("err = %v; want exceeds error", err)
	}
}

// ─── Resolve ───────────────────────────────────────────────────────────────

// indexAndVersionHandler serves the minimal set of URLs T3 + T2
// exercise: /index.json → indexBody, and
// /plugins/{pub}/{name}/{ver}.json → versionBody (verbatim).
func indexAndVersionHandler(indexBody, versionBody string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/index.json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, indexBody)
		case strings.HasPrefix(r.URL.Path, "/plugins/") && strings.HasSuffix(r.URL.Path, ".json"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, versionBody)
		default:
			http.NotFound(w, r)
		}
	}
}

const fakeSHA256 = "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"

func versionBody(pub, name, ver, sha, artifactURL string) string {
	return fmt.Sprintf(`{
		"name": %q,
		"publisher": %q,
		"version": %q,
		"artifact": {"url": %q, "size": 1024},
		"sha256": %q,
		"manifest": {
			"name": %q,
			"version": %q,
			"publisher": %q,
			"displayName": "FS Readme",
			"description": "reads README",
			"icon": "📖",
			"form": "host",
			"permissions": {"fs": {"read": ["${home}/**"]}}
		}
	}`, name, pub, ver, artifactURL, sha, name, ver, pub)
}

func TestResolve_HappyPath_ExplicitVersion(t *testing.T) {
	c := newTestCatalog(t, indexAndVersionHandler("",
		versionBody("opendray-examples", "fs-readme", "1.0.0", fakeSHA256,
			"https://example.com/fs-readme-1.0.0.zip")))

	entry, err := c.Resolve(context.Background(), market.Ref{
		Publisher: "opendray-examples", Name: "fs-readme", Version: "1.0.0",
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if entry.ArtifactURL != "https://example.com/fs-readme-1.0.0.zip" {
		t.Errorf("ArtifactURL = %q", entry.ArtifactURL)
	}
	if entry.SHA256 != fakeSHA256 {
		t.Errorf("SHA256 = %q", entry.SHA256)
	}
	if entry.Form != "host" {
		t.Errorf("Form = %q, want host (from manifest)", entry.Form)
	}
	if len(entry.Permissions) == 0 {
		t.Errorf("Permissions empty; want raw JSON from manifest")
	}
	if entry.Trust != "community" {
		t.Errorf("Trust default = %q, want community (T10 fills from publisher record)", entry.Trust)
	}
}

func TestResolve_LatestFromIndex(t *testing.T) {
	// When Version is empty, Resolve looks up the latest from the
	// index and then fetches that version's JSON.
	c := newTestCatalog(t, indexAndVersionHandler(`{
		"version": 1,
		"generatedAt": "",
		"plugins": [
			{"name": "fs-readme", "publisher": "opendray-examples", "latest": "2.3.0"}
		]
	}`,
		versionBody("opendray-examples", "fs-readme", "2.3.0", fakeSHA256,
			"https://example.com/fs-readme-2.3.0.zip")))

	entry, err := c.Resolve(context.Background(), market.Ref{
		Publisher: "opendray-examples", Name: "fs-readme", // no version
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if entry.Version != "2.3.0" {
		t.Errorf("Version = %q, want 2.3.0 (from index latest)", entry.Version)
	}
}

func TestResolve_DefaultPublisher(t *testing.T) {
	// Bare-name ref (M3 back-compat): Publisher defaults to
	// opendray-examples before any HTTP call.
	c := newTestCatalog(t, indexAndVersionHandler(`{
		"version": 1,
		"generatedAt": "",
		"plugins": [
			{"name": "fs-readme", "publisher": "opendray-examples", "latest": "1.0.0"}
		]
	}`,
		versionBody("opendray-examples", "fs-readme", "1.0.0", fakeSHA256,
			"https://example.com/fs-readme-1.0.0.zip")))

	entry, err := c.Resolve(context.Background(), market.Ref{Name: "fs-readme"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if entry.Publisher != "opendray-examples" {
		t.Errorf("Publisher = %q, want opendray-examples default", entry.Publisher)
	}
}

func TestResolve_NotFound(t *testing.T) {
	// 404 on the version JSON maps to market.ErrNotFound.
	c := newTestCatalog(t, func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	_, err := c.Resolve(context.Background(), market.Ref{
		Publisher: "acme", Name: "missing", Version: "1.0.0",
	})
	if err == nil || !errors.Is(err, market.ErrNotFound) {
		t.Errorf("err = %v; want ErrNotFound", err)
	}
}

func TestResolve_BodyMismatchesURL(t *testing.T) {
	// Registry-side typo: the version body claims "evil" but the
	// URL asked for "fs-readme". Must reject rather than silently
	// ship the wrong manifest.
	c := newTestCatalog(t, indexAndVersionHandler("",
		versionBody("opendray-examples", "evil", "1.0.0", fakeSHA256,
			"https://example.com/x.zip")))

	_, err := c.Resolve(context.Background(), market.Ref{
		Publisher: "opendray-examples", Name: "fs-readme", Version: "1.0.0",
	})
	if err == nil || !strings.Contains(err.Error(), "doesn't match URL") {
		t.Errorf("err = %v; want mismatch rejection", err)
	}
}

func TestResolve_BadSHA256(t *testing.T) {
	c := newTestCatalog(t, indexAndVersionHandler("",
		versionBody("acme", "plugin", "1.0.0", "not-a-valid-sha",
			"https://example.com/x.zip")))

	_, err := c.Resolve(context.Background(), market.Ref{
		Publisher: "acme", Name: "plugin", Version: "1.0.0",
	})
	if err == nil || !strings.Contains(err.Error(), "sha256 malformed") {
		t.Errorf("err = %v; want sha256 malformed", err)
	}
}

func TestResolve_MissingArtifact(t *testing.T) {
	// artifact.url empty = broken registry entry.
	body := `{
		"name": "x", "publisher": "acme", "version": "1.0.0",
		"artifact": {"url": "", "size": 0},
		"sha256": "` + fakeSHA256 + `",
		"manifest": {"name":"x","version":"1.0.0","publisher":"acme"}
	}`
	c := newTestCatalog(t, indexAndVersionHandler("", body))
	_, err := c.Resolve(context.Background(), market.Ref{
		Publisher: "acme", Name: "x", Version: "1.0.0",
	})
	if err == nil || !strings.Contains(err.Error(), "missing artifact") {
		t.Errorf("err = %v; want missing artifact", err)
	}
}

// ─── FetchPublisher ────────────────────────────────────────────────────────

func TestFetchPublisher_HappyPath(t *testing.T) {
	c := newTestCatalog(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/publishers/acme.json" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{
			"name": "acme",
			"trust": "verified",
			"keys": [{"alg":"ed25519","publicKey":"xyz","addedAt":"2024-01-01T00:00:00Z"}]
		}`)
	})
	rec, err := c.FetchPublisher(context.Background(), "acme")
	if err != nil {
		t.Fatal(err)
	}
	if rec.Name != "acme" || rec.Trust != "verified" || len(rec.Keys) != 1 {
		t.Errorf("rec = %+v", rec)
	}
}

func TestFetchPublisher_NotFound(t *testing.T) {
	c := newTestCatalog(t, func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	_, err := c.FetchPublisher(context.Background(), "missing")
	if !errors.Is(err, market.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

// ─── Mirror fallback ──────────────────────────────────────────────────────

// mirrorCatalog wires a Catalog against a primary URL + one or more
// mirror URLs (each served by its own httptest server). Keeps the
// call-site clean for multi-server tests.
func mirrorCatalog(t *testing.T, primary string, mirrors ...string) *Catalog {
	t.Helper()
	c, err := New(Config{
		RegistryURL: primary,
		Mirrors:     mirrors,
		HTTPClient:  &http.Client{}, // no global timeout — lets tests control
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

// TestMirror_PrimaryFailsMirrorSucceeds — primary returns 503
// (retryable), first mirror 200. Expect mirror body returned.
func TestMirror_PrimaryFailsMirrorSucceeds(t *testing.T) {
	primarySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	t.Cleanup(primarySrv.Close)

	mirrorSrv := httptest.NewServer(indexHandler(`{
		"version": 1,
		"generatedAt": "",
		"plugins": [{"name":"a","publisher":"b","latest":"1.0.0"}]
	}`))
	t.Cleanup(mirrorSrv.Close)

	c := mirrorCatalog(t, primarySrv.URL, mirrorSrv.URL)
	entries, err := c.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("want 1 entry from mirror, got %d", len(entries))
	}
}

// TestMirror_4xxDoesNotFallback — 404 on the primary is
// definitive. Mirrors aren't tried; Resolve surfaces ErrNotFound
// straight away. Otherwise every missing plugin would hit every
// mirror, multiplying load on a normal miss.
func TestMirror_4xxDoesNotFallback(t *testing.T) {
	// Mirror would succeed if reached. Shouldn't be.
	mirrorHit := 0
	mirrorSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mirrorHit++
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, versionBody("acme", "plug", "1.0.0", fakeSHA256, "https://x/y.zip"))
	}))
	t.Cleanup(mirrorSrv.Close)

	primarySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	t.Cleanup(primarySrv.Close)

	c := mirrorCatalog(t, primarySrv.URL, mirrorSrv.URL)
	_, err := c.Resolve(context.Background(), market.Ref{
		Publisher: "acme", Name: "plug", Version: "1.0.0",
	})
	if err == nil || !errors.Is(err, market.ErrNotFound) {
		t.Errorf("err = %v; want ErrNotFound", err)
	}
	if mirrorHit != 0 {
		t.Errorf("mirror hit %d times; want 0 (4xx should be definitive)", mirrorHit)
	}
}

// TestMirror_AllFailReturnsLastError — every base returns 503.
// Expect the last base's error surfaced to the caller.
func TestMirror_AllFailReturnsLastError(t *testing.T) {
	p := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "p", http.StatusServiceUnavailable)
	}))
	t.Cleanup(p.Close)
	m := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "m", http.StatusBadGateway)
	}))
	t.Cleanup(m.Close)

	c := mirrorCatalog(t, p.URL, m.URL)
	_, err := c.List(context.Background())
	if err == nil {
		t.Fatal("want error when all bases fail")
	}
	var hs *HTTPStatusError
	if !errors.As(err, &hs) || hs.Status != http.StatusBadGateway {
		t.Errorf("err = %v; want HTTPStatusError 502 from last base", err)
	}
}

// TestMirror_CtxCancelStopsRetry — caller cancel aborts the loop
// immediately; further mirrors are not tried.
func TestMirror_CtxCancelStopsRetry(t *testing.T) {
	mirrorHit := 0
	m := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mirrorHit++
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"version":1,"plugins":[]}`)
	}))
	t.Cleanup(m.Close)

	// Primary server never responds — client's ctx kills the request.
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	t.Cleanup(primary.Close)

	c, err := New(Config{
		RegistryURL: primary.URL,
		Mirrors:     []string{m.URL},
		HTTPClient:  &http.Client{},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately; primary will see ctx.Done.

	_, err = c.List(ctx)
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v; want context.Canceled", err)
	}
	if mirrorHit != 0 {
		t.Errorf("mirror hit %d times; want 0 under cancel", mirrorHit)
	}
}

// ─── Cache ────────────────────────────────────────────────────────────────

func TestCache_HitsAvoidSecondNetwork(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/index.json" {
			http.NotFound(w, r)
			return
		}
		hits++
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"version":1,"plugins":[]}`)
	}))
	t.Cleanup(srv.Close)

	c, err := New(Config{
		RegistryURL: srv.URL,
		HTTPClient:  srv.Client(),
		CacheTTL:    time.Minute, // long enough that both calls hit cache
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := c.List(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := c.List(context.Background()); err != nil {
		t.Fatal(err)
	}
	if hits != 1 {
		t.Errorf("upstream hit %d times; want 1 (second call served from cache)", hits)
	}
}

func TestCache_DisabledForcesEveryCall(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"version":1,"plugins":[]}`)
	}))
	t.Cleanup(srv.Close)

	c, _ := New(Config{
		RegistryURL: srv.URL,
		HTTPClient:  srv.Client(),
		CacheTTL:    -1, // disable
	})
	_, _ = c.List(context.Background())
	_, _ = c.List(context.Background())
	if hits != 2 {
		t.Errorf("upstream hit %d times; want 2 with cache disabled", hits)
	}
}

func TestCache_Invalidate(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"version":1,"plugins":[]}`)
	}))
	t.Cleanup(srv.Close)

	c, _ := New(Config{
		RegistryURL: srv.URL,
		HTTPClient:  srv.Client(),
		CacheTTL:    time.Minute,
	})
	_, _ = c.List(context.Background())
	_, _ = c.List(context.Background())
	c.InvalidateCache()
	_, _ = c.List(context.Background())
	if hits != 2 {
		t.Errorf("upstream hit %d times; want 2 (first hit + post-invalidate)", hits)
	}
}

func TestCache_ErrorsNotCached(t *testing.T) {
	// 503 response; next call should retry the network rather than
	// serving a poisoned cache entry.
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)

	c, _ := New(Config{
		RegistryURL: srv.URL,
		HTTPClient:  srv.Client(),
		CacheTTL:    time.Minute,
	})
	_, _ = c.List(context.Background())
	_, _ = c.List(context.Background())
	if hits != 2 {
		t.Errorf("upstream hit %d times on repeated 503; want 2", hits)
	}
}

// ─── BundlePath stays a no-op for remote ───────────────────────────────────

func TestBundlePath_RemoteOnlyReturnsFalse(t *testing.T) {
	c := newTestCatalog(t, indexHandler(`{"version":1,"plugins":[]}`))
	p, ok, err := c.BundlePath(context.Background(), market.Ref{Name: "x"})
	if err != nil {
		t.Fatalf("BundlePath err = %v", err)
	}
	if ok || p != "" {
		t.Errorf("BundlePath = (%q, %v), want ('', false)", p, ok)
	}
}

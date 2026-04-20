package remote

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
	if !strings.HasSuffix(c.base.Path, "/") {
		t.Errorf("base.Path = %q, want trailing slash", c.base.Path)
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

// ─── Resolve / BundlePath — still stubs ────────────────────────────────────

func TestResolve_StillStub(t *testing.T) {
	c := newTestCatalog(t, indexHandler(`{"version":1,"plugins":[]}`))
	_, err := c.Resolve(context.Background(), market.Ref{Name: "x"})
	if err == nil {
		t.Fatal("Resolve should still return ErrNotImplemented until T3")
	}
}

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

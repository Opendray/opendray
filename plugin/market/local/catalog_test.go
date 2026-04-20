package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/opendray/opendray/plugin/market"
)

func writeCatalog(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "catalog.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoad_Empty(t *testing.T) {
	// A missing catalog.json is not an error — the server should boot
	// in un-seeded environments and just serve an empty Hub.
	dir := t.TempDir()
	c, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	entries, err := c.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("empty catalog: want 0 entries, got %d", len(entries))
	}
}

func TestLoad_HappyPath(t *testing.T) {
	dir := t.TempDir()
	writeCatalog(t, dir, `{
		"schemaVersion": 1,
		"entries": [
			{"name": "fs-readme", "version": "1.0.0", "publisher": "opendray-examples", "form": "host"},
			{"name": "alpha", "version": "2.0.0", "publisher": "opendray"}
		]
	}`)
	c, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	entries, _ := c.List(context.Background())
	if len(entries) != 2 {
		t.Fatalf("want 2 entries, got %d", len(entries))
	}
	// Sort order: publisher, then name, then version.
	if entries[0].Publisher != "opendray" || entries[1].Publisher != "opendray-examples" {
		t.Errorf("sort order by publisher: got %q, %q", entries[0].Publisher, entries[1].Publisher)
	}
}

func TestLoad_DefaultPublisher(t *testing.T) {
	// Entries without a publisher get "opendray-examples" so the
	// M3-era catalogs (no publisher column) keep resolving.
	dir := t.TempDir()
	writeCatalog(t, dir, `{"entries":[{"name":"x","version":"1.0.0"}]}`)
	c, _ := Load(dir)
	entries, _ := c.List(context.Background())
	if entries[0].Publisher != "opendray-examples" {
		t.Errorf("default publisher = %q, want opendray-examples", entries[0].Publisher)
	}
}

func TestResolve_And_BundlePath(t *testing.T) {
	dir := t.TempDir()
	writeCatalog(t, dir, `{
		"entries": [
			{"name": "fs-readme", "version": "1.0.0", "publisher": "opendray-examples"}
		]
	}`)
	// Populate the M3 flat layout — Resolve should fall back to it
	// when the new namespaced path isn't present.
	legacyDir := filepath.Join(dir, "packages", "fs-readme", "1.0.0")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	c, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		ref     string
		wantDir string
	}{
		{"bareName", "fs-readme", "packages/fs-readme/1.0.0"},
		{"bareAt",   "fs-readme@1.0.0", "packages/fs-readme/1.0.0"},
		{"scheme",   "marketplace://fs-readme", "packages/fs-readme/1.0.0"},
		{"schemeAt", "marketplace://fs-readme@1.0.0", "packages/fs-readme/1.0.0"},
		{"pub",      "opendray-examples/fs-readme", "packages/fs-readme/1.0.0"},
		{"pubAt",    "opendray-examples/fs-readme@1.0.0", "packages/fs-readme/1.0.0"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ref, err := market.ParseRef(tc.ref)
			if err != nil {
				t.Fatalf("ParseRef: %v", err)
			}
			entry, err := c.Resolve(context.Background(), ref)
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if entry.Name != "fs-readme" {
				t.Errorf("entry.Name = %q", entry.Name)
			}
			got, haveLocal, err := c.BundlePath(context.Background(), ref)
			if err != nil {
				t.Fatalf("BundlePath: %v", err)
			}
			if !haveLocal {
				t.Errorf("want local bundle, got remote-only")
			}
			wantAbs := filepath.Join(dir, tc.wantDir)
			if got != wantAbs {
				t.Errorf("bundlePath = %q, want %q", got, wantAbs)
			}
		})
	}
}

func TestResolve_NotFound(t *testing.T) {
	dir := t.TempDir()
	writeCatalog(t, dir, `{"entries":[]}`)
	c, _ := Load(dir)
	ref, _ := market.ParseRef("missing")
	if _, err := c.Resolve(context.Background(), ref); !errors.Is(err, market.ErrNotFound) {
		t.Errorf("Resolve want ErrNotFound, got %v", err)
	}
}

func TestFetchPublisher(t *testing.T) {
	dir := t.TempDir()
	writeCatalog(t, dir, `{"entries":[]}`)
	if err := os.MkdirAll(filepath.Join(dir, "publishers"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{
		"name": "acme",
		"trust": "verified",
		"keys": [
			{"alg":"ed25519","publicKey":"base64pubkey==","addedAt":"2024-01-01T00:00:00Z"}
		]
	}`
	if err := os.WriteFile(filepath.Join(dir, "publishers", "acme.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	c, _ := Load(dir)

	rec, err := c.FetchPublisher(context.Background(), "acme")
	if err != nil {
		t.Fatalf("FetchPublisher: %v", err)
	}
	if rec.Trust != "verified" {
		t.Errorf("Trust = %q, want verified", rec.Trust)
	}
	if len(rec.Keys) != 1 || rec.Keys[0].PublicKey != "base64pubkey==" {
		t.Errorf("keys = %+v", rec.Keys)
	}
}

func TestFetchPublisher_NotFound(t *testing.T) {
	dir := t.TempDir()
	writeCatalog(t, dir, `{"entries":[]}`)
	c, _ := Load(dir)
	_, err := c.FetchPublisher(context.Background(), "missing")
	if !errors.Is(err, market.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestFetchPublisher_DefaultsTrust(t *testing.T) {
	dir := t.TempDir()
	writeCatalog(t, dir, `{"entries":[]}`)
	if err := os.MkdirAll(filepath.Join(dir, "publishers"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "publishers", "newbie.json"),
		[]byte(`{"name":"newbie","keys":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	c, _ := Load(dir)
	rec, err := c.FetchPublisher(context.Background(), "newbie")
	if err != nil {
		t.Fatal(err)
	}
	if rec.Trust != "community" {
		t.Errorf("default Trust = %q, want community", rec.Trust)
	}
}

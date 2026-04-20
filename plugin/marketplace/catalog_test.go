package marketplace

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
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
	if got := len(c.List()); got != 0 {
		t.Errorf("empty catalog: want 0 entries, got %d", got)
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
	entries := c.List()
	if len(entries) != 2 {
		t.Fatalf("want 2 entries, got %d", len(entries))
	}
	// Sorted alphabetically by name.
	if entries[0].Name != "alpha" || entries[1].Name != "fs-readme" {
		t.Errorf("sort order: got %s, %s", entries[0].Name, entries[1].Name)
	}
}

func TestResolve(t *testing.T) {
	dir := t.TempDir()
	writeCatalog(t, dir, `{
		"entries": [
			{"name": "fs-readme", "version": "1.0.0", "publisher": "opendray-examples"}
		]
	}`)
	c, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		ref      string
		wantDir  string
		wantName string
	}{
		{"fs-readme", "packages/fs-readme/1.0.0", "fs-readme"},
		{"fs-readme@1.0.0", "packages/fs-readme/1.0.0", "fs-readme"},
		{"marketplace://fs-readme", "packages/fs-readme/1.0.0", "fs-readme"},
		{"marketplace://fs-readme@1.0.0", "packages/fs-readme/1.0.0", "fs-readme"},
	}
	for _, tc := range tests {
		t.Run(tc.ref, func(t *testing.T) {
			entry, bundlePath, err := c.Resolve(tc.ref)
			if err != nil {
				t.Fatalf("Resolve(%q): %v", tc.ref, err)
			}
			if entry.Name != tc.wantName {
				t.Errorf("entry.Name = %q, want %q", entry.Name, tc.wantName)
			}
			wantAbs := filepath.Join(dir, tc.wantDir)
			if bundlePath != wantAbs {
				t.Errorf("bundlePath = %q, want %q", bundlePath, wantAbs)
			}
		})
	}
}

func TestResolve_NotFound(t *testing.T) {
	dir := t.TempDir()
	writeCatalog(t, dir, `{"entries":[]}`)
	c, _ := Load(dir)
	if _, _, err := c.Resolve("missing"); !errors.Is(err, ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestParseRef_Rejects(t *testing.T) {
	cases := []string{
		"",
		"   ",
		"marketplace://",
		"../etc/passwd",
		`\backslash`,
		"with/slash",
	}
	for _, raw := range cases {
		t.Run(raw, func(t *testing.T) {
			if _, _, err := ParseRef(raw); !errors.Is(err, ErrBadRef) {
				t.Errorf("ParseRef(%q) = %v, want ErrBadRef", raw, err)
			}
		})
	}
}

func TestParseRef_Accepts(t *testing.T) {
	cases := []struct{ in, name, ver string }{
		{"foo", "foo", ""},
		{"foo@1.0.0", "foo", "1.0.0"},
		{"marketplace://foo", "foo", ""},
		{"marketplace://foo@1.0.0", "foo", "1.0.0"},
	}
	for _, tc := range cases {
		n, v, err := ParseRef(tc.in)
		if err != nil {
			t.Errorf("ParseRef(%q) error: %v", tc.in, err)
			continue
		}
		if n != tc.name || v != tc.ver {
			t.Errorf("ParseRef(%q) = (%q,%q), want (%q,%q)", tc.in, n, v, tc.name, tc.ver)
		}
	}
}

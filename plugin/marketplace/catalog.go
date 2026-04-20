// Package marketplace owns the on-disk plugin catalog that backs the
// Hub page and the `marketplace://` install source.
//
// Layout (rooted at Catalog.Dir):
//
//	catalog.json
//	packages/
//	  <name>/
//	    <version>/
//	      manifest.json
//	      …bundle contents…
//
// `catalog.json` is the authoritative list of installable entries.
// Package directories not listed in catalog.json are ignored by Resolve
// so a half-extracted bundle can't accidentally become installable.
//
// The marketplace package intentionally has NO dependency on
// plugin/install — the gateway bridges the two. This keeps the install
// package's scope tight and lets tests stub one side without dragging
// the other.
package marketplace

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ErrNotFound is returned by Resolve when the ref doesn't match any
// catalog entry. Callers should surface this as HTTP 404.
var ErrNotFound = errors.New("marketplace: entry not found")

// ErrBadRef is returned by ParseRef for malformed marketplace URIs.
var ErrBadRef = errors.New("marketplace: bad ref")

// Entry is one installable plugin advertised by the catalog. Field
// names match the JSON wire shape so the catalog.json file can be
// hand-edited without a schema translator in between.
type Entry struct {
	Name        string          `json:"name"`
	Version     string          `json:"version"`
	Publisher   string          `json:"publisher"`
	DisplayName string          `json:"displayName,omitempty"`
	Description string          `json:"description,omitempty"`
	Icon        string          `json:"icon,omitempty"`
	// Form is the runtime shape ("host" / "declarative" / "webview") so
	// the Hub card can render a badge without fetching the full manifest.
	Form string `json:"form,omitempty"`
	// Tags are arbitrary classifier strings ("reference", "example",
	// "agent", "panel"). Hub uses them for filtering.
	Tags []string `json:"tags,omitempty"`
	// Permissions is the raw permissions block copied verbatim from the
	// bundled manifest so the Hub page can preview what the plugin will
	// ask for before the user hits install. Opaque JSON — the Hub card
	// renders it the same way the install-consent dialog does.
	Permissions json.RawMessage `json:"permissions,omitempty"`
}

// Catalog is the loaded, immutable snapshot of the on-disk catalog.
// Rebuild via Load when the file changes; do not mutate a live Catalog.
type Catalog struct {
	// Dir is the root the catalog was loaded from. Resolve() joins
	// packages/<name>/<version> onto this path.
	Dir string

	entries []Entry
	// byKey maps "<name>@<version>" → Entry index into entries. We also
	// map the bare "<name>" to the latest-seen version so callers can
	// pass either form to Resolve.
	byKey map[string]int
}

// Load reads dir/catalog.json and returns a Catalog. A missing file
// yields an empty catalog (not an error) so the server can boot in
// environments that haven't been seeded yet; the Hub page handles an
// empty list gracefully.
func Load(dir string) (*Catalog, error) {
	c := &Catalog{Dir: dir, byKey: make(map[string]int)}
	if dir == "" {
		return c, nil
	}
	data, err := os.ReadFile(filepath.Join(dir, "catalog.json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return c, nil
		}
		return nil, fmt.Errorf("marketplace: read catalog: %w", err)
	}
	var raw struct {
		SchemaVersion int     `json:"schemaVersion"`
		Entries       []Entry `json:"entries"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("marketplace: parse catalog: %w", err)
	}
	// Stable order for the Hub UI: alphabetical by name, then by
	// version. Matches the List() contract so the same response flows
	// through every call regardless of file layout.
	entries := append([]Entry(nil), raw.Entries...)
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Name != entries[j].Name {
			return entries[i].Name < entries[j].Name
		}
		return entries[i].Version < entries[j].Version
	})
	c.entries = entries
	for i, e := range entries {
		c.byKey[e.Name+"@"+e.Version] = i
		// Last-write wins for the bare name, which — because of the sort
		// above — ends up pointing at the highest version string.
		c.byKey[e.Name] = i
	}
	return c, nil
}

// List returns a copy of all entries. Callers can freely slice/mutate
// the returned slice; the catalog retains its own copy.
func (c *Catalog) List() []Entry {
	out := make([]Entry, len(c.entries))
	copy(out, c.entries)
	return out
}

// Resolve looks up a ref and returns the matching Entry + the
// absolute bundle directory path on disk. ref is one of:
//
//	"<name>"                — latest version in catalog order
//	"<name>@<version>"      — specific pin
//	"marketplace://<name>"  — with or without @<version>
//
// ErrNotFound is returned when no catalog entry matches.
func (c *Catalog) Resolve(ref string) (Entry, string, error) {
	name, version, err := ParseRef(ref)
	if err != nil {
		return Entry{}, "", err
	}
	key := name
	if version != "" {
		key = name + "@" + version
	}
	idx, ok := c.byKey[key]
	if !ok {
		return Entry{}, "", fmt.Errorf("%w: %s", ErrNotFound, ref)
	}
	entry := c.entries[idx]
	bundleDir := filepath.Join(c.Dir, "packages", entry.Name, entry.Version)
	return entry, bundleDir, nil
}

// ParseRef splits a marketplace reference into (name, version). Both
// "marketplace://foo" and bare "foo" shapes are accepted; version is
// returned empty when the ref elides it. Rejects empty strings and
// anything that can't be a plain name token.
func ParseRef(ref string) (name string, version string, err error) {
	raw := strings.TrimSpace(ref)
	if raw == "" {
		return "", "", fmt.Errorf("%w: empty ref", ErrBadRef)
	}
	raw = strings.TrimPrefix(raw, "marketplace://")
	// Disallow path separators so a hostile ref can't climb out of the
	// packages/ tree when joined below.
	if strings.ContainsAny(raw, `/\`) {
		return "", "", fmt.Errorf("%w: path separators not allowed in ref %q", ErrBadRef, ref)
	}
	if at := strings.IndexByte(raw, '@'); at >= 0 {
		name = raw[:at]
		version = raw[at+1:]
	} else {
		name = raw
	}
	if name == "" {
		return "", "", fmt.Errorf("%w: missing name in %q", ErrBadRef, ref)
	}
	return name, version, nil
}

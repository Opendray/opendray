// Package local implements [market.Catalog] against an on-disk
// directory.
//
// Layout (rooted at Catalog.Dir):
//
//	catalog.json
//	packages/
//	  <publisher>/<name>/<version>/
//	    manifest.json
//	    …bundle contents…
//
// Back-compat: M3's layout had `packages/<name>/<version>/` without a
// publisher level. When catalog.json lists an entry without a
// publisher, Load() fills `opendray-examples` and searches both
// `packages/opendray-examples/<name>/<version>/` and the legacy
// `packages/<name>/<version>/`.
//
// catalog.json is the authoritative list. Package directories not
// listed there are ignored by Resolve, so a half-extracted bundle
// can't accidentally become installable.
package local

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/opendray/opendray/plugin/market"
)

// Catalog is the loaded, immutable snapshot of the on-disk catalog.
// Rebuild via Load when the file changes; do not mutate a live
// Catalog.
type Catalog struct {
	// Dir is the root the catalog was loaded from. BundlePath joins
	// packages/<publisher>/<name>/<version> onto this path.
	Dir string

	entries []market.Entry
	// byKey maps "<publisher>/<name>@<version>" → Entry index. Also
	// maps "<publisher>/<name>" and bare "<name>" forms to the
	// latest-seen version so any ref shape resolves.
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
		return nil, fmt.Errorf("market/local: read catalog: %w", err)
	}
	var raw struct {
		SchemaVersion int            `json:"schemaVersion"`
		Entries       []market.Entry `json:"entries"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("market/local: parse catalog: %w", err)
	}

	// Fill missing publisher with the M3-era default so mixed
	// catalogs (old + new shape) resolve consistently.
	for i := range raw.Entries {
		if raw.Entries[i].Publisher == "" {
			raw.Entries[i].Publisher = "opendray-examples"
		}
		if raw.Entries[i].Trust == "" {
			raw.Entries[i].Trust = "community"
		}
	}

	// Stable order: publisher, name, version ascending.
	entries := append([]market.Entry(nil), raw.Entries...)
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Publisher != entries[j].Publisher {
			return entries[i].Publisher < entries[j].Publisher
		}
		if entries[i].Name != entries[j].Name {
			return entries[i].Name < entries[j].Name
		}
		return entries[i].Version < entries[j].Version
	})
	c.entries = entries
	for i, e := range entries {
		pn := e.Publisher + "/" + e.Name
		c.byKey[pn+"@"+e.Version] = i
		c.byKey[pn] = i       // last one wins → latest version
		c.byKey[e.Name] = i   // bare-name back-compat
	}
	return c, nil
}

// List implements market.Catalog.
func (c *Catalog) List(_ context.Context) ([]market.Entry, error) {
	out := make([]market.Entry, len(c.entries))
	copy(out, c.entries)
	return out, nil
}

// Resolve implements market.Catalog.
func (c *Catalog) Resolve(_ context.Context, ref market.Ref) (market.Entry, error) {
	key, ok := c.lookupKey(ref)
	if !ok {
		return market.Entry{}, fmt.Errorf("%w: %s", market.ErrNotFound, ref)
	}
	return c.entries[c.byKey[key]], nil
}

// FetchPublisher implements market.Catalog. Reads
// `publishers/<publisher>.json` relative to the catalog root. A
// missing record returns ErrNotFound so callers can surface a
// clean 404 rather than a stat error.
func (c *Catalog) FetchPublisher(_ context.Context, publisher string) (market.PublisherRecord, error) {
	if c.Dir == "" || publisher == "" {
		return market.PublisherRecord{}, fmt.Errorf("%w: publisher %q", market.ErrNotFound, publisher)
	}
	path := filepath.Join(c.Dir, "publishers", publisher+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return market.PublisherRecord{}, fmt.Errorf("%w: publisher %q", market.ErrNotFound, publisher)
		}
		return market.PublisherRecord{}, fmt.Errorf("market/local: read publisher %q: %w", publisher, err)
	}
	var rec market.PublisherRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return market.PublisherRecord{}, fmt.Errorf("market/local: parse publisher %q: %w", publisher, err)
	}
	if rec.Name == "" {
		rec.Name = publisher
	}
	if rec.Trust == "" {
		rec.Trust = "community"
	}
	return rec, nil
}

// BundlePath implements market.Catalog. The local backend always has
// the bundle on disk, so returns (path, true, nil) when the ref
// resolves.
func (c *Catalog) BundlePath(_ context.Context, ref market.Ref) (string, bool, error) {
	key, ok := c.lookupKey(ref)
	if !ok {
		return "", false, fmt.Errorf("%w: %s", market.ErrNotFound, ref)
	}
	entry := c.entries[c.byKey[key]]

	// Prefer the new namespaced layout
	// (packages/<publisher>/<name>/<version>/), fall back to the M3
	// flat layout (packages/<name>/<version>/).
	candidates := []string{
		filepath.Join(c.Dir, "packages", entry.Publisher, entry.Name, entry.Version),
		filepath.Join(c.Dir, "packages", entry.Name, entry.Version),
	}
	for _, p := range candidates {
		if fi, err := os.Stat(p); err == nil && fi.IsDir() {
			return p, true, nil
		}
	}
	return "", false, fmt.Errorf("%w: bundle dir missing for %s", market.ErrNotFound, ref)
}

func (c *Catalog) lookupKey(ref market.Ref) (string, bool) {
	// Try, in order: "<pub>/<name>@<ver>", "<pub>/<name>",
	// "<name>@<ver>" (bare back-compat), "<name>".
	candidates := make([]string, 0, 4)
	if ref.Publisher != "" {
		pn := ref.Publisher + "/" + ref.Name
		if ref.Version != "" {
			candidates = append(candidates, pn+"@"+ref.Version)
		}
		candidates = append(candidates, pn)
	}
	if ref.Version != "" {
		candidates = append(candidates, ref.Name+"@"+ref.Version)
	}
	candidates = append(candidates, ref.Name)

	for _, k := range candidates {
		if _, ok := c.byKey[k]; ok {
			return k, true
		}
	}
	return "", false
}

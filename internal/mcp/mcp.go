// Package mcp implements the MCP (Model Context Protocol) server
// registry — the unit a user bundles together as "an external tool
// surface" for their AI agent.
//
// Storage model — same philosophy as the skills + notes vaults:
//
//	<vault>/mcp/<id>/mcp.json   # user-defined MCP servers (filesystem)
//
// No built-ins are embedded in the binary. Users add servers via the
// Plugins page or by dropping a directory into the vault by hand.
//
// At session spawn time the catalog adapter calls List, resolves
// ${SECRET} placeholders against the secrets file, and merges the
// enabled servers with whatever the provider's own config declares;
// the per-CLI renderer (catalog.renderMCP) then writes the right
// claude-mcp.json / codex config.toml under the per-session scratch
// dir.
package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Server is one entry in the registry. Mirrors the existing
// catalog.MCPServer shape (so the renderer doesn't need a translation
// step) but adds id / description / enabled for the registry UI.
//
// Env / Headers values may contain ${KEY} placeholders that get
// substituted from the secrets file at spawn time. The raw values
// stored in mcp.json keep the placeholder so the file stays git-safe.
type Server struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Transport   string            `json:"transport,omitempty"` // stdio (default) | sse | http
	Command     string            `json:"command,omitempty"`
	Args        []string          `json:"args,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	URL         string            `json:"url,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Enabled     bool              `json:"enabled"`
	// SourcePath is the absolute fs path the server was loaded from.
	// Set by the loader, not persisted.
	SourcePath string `json:"-"`
}

// Loader walks the vault directory and returns Server entries. Mirrors
// the skills loader's API surface so the wiring is symmetric.
type Loader struct {
	vaultRoot string // absolute path to <vault>/mcp (may not exist)
}

// NewLoader points the loader at a directory that contains one
// subfolder per server (e.g. `<dir>/filesystem/mcp.json`). Empty `dir`
// disables loading entirely; List returns an empty slice.
func NewLoader(dir string) *Loader { return &Loader{vaultRoot: dir} }

// VaultRoot returns the absolute path the loader is reading from.
func (l *Loader) VaultRoot() string { return l.vaultRoot }

// List returns every server in the vault, sorted alphabetically by id.
// Missing / inaccessible directories return an empty list, not an
// error — the registry is optional for the gateway to start.
func (l *Loader) List() ([]Server, error) {
	if l.vaultRoot == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(l.vaultRoot)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read mcp vault: %w", err)
	}
	out := make([]Server, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		s, err := loadOne(l.vaultRoot, e.Name())
		if err != nil {
			// Skip malformed entries so a single bad mcp.json doesn't
			// stop the whole list from rendering. Caller can surface
			// the error per-id via Get.
			continue
		}
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// ListEnabled is List filtered to entries where Enabled=true. Used by
// the catalog adapter at spawn time so disabled servers don't get
// injected.
func (l *Loader) ListEnabled() ([]Server, error) {
	all, err := l.List()
	if err != nil {
		return nil, err
	}
	out := make([]Server, 0, len(all))
	for _, s := range all {
		if s.Enabled {
			out = append(out, s)
		}
	}
	return out, nil
}

// Get returns one server by id. Returns os.ErrNotExist when the
// directory doesn't exist so callers can disambiguate 404s.
func (l *Loader) Get(id string) (Server, error) {
	if l.vaultRoot == "" {
		return Server{}, fs.ErrNotExist
	}
	return loadOne(l.vaultRoot, id)
}

// loadOne reads <root>/<id>/mcp.json and parses it. Defaults `id` and
// `name` to the directory name if the file omits them.
func loadOne(root, id string) (Server, error) {
	path := filepath.Join(root, id, "mcp.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return Server{}, err
	}
	var s Server
	if err := json.Unmarshal(data, &s); err != nil {
		return Server{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if s.ID == "" {
		s.ID = id
	}
	if s.Name == "" {
		s.Name = id
	}
	if s.Transport == "" {
		s.Transport = "stdio"
	}
	s.SourcePath = filepath.Join(root, id)
	return s, nil
}

// ValidID enforces the directory naming rules: lowercase alphanumeric,
// dash, underscore. Same rules as skills so muscle memory carries over.
func ValidID(id string) bool {
	if id == "" || len(id) > 64 {
		return false
	}
	for _, r := range id {
		ok := (r >= 'a' && r <= 'z') ||
			(r >= '0' && r <= '9') ||
			r == '-' || r == '_'
		if !ok {
			return false
		}
	}
	return true
}

// Marshal serialises a Server back to indented JSON. Used by the HTTP
// handler so the on-disk file format stays human-readable / diffable.
func Marshal(s Server) ([]byte, error) {
	// Strip computed / non-persisted fields before writing so the file
	// only contains user-authored content.
	clean := s
	clean.SourcePath = ""
	if strings.TrimSpace(clean.Transport) == "stdio" {
		clean.Transport = ""
	}
	return json.MarshalIndent(clean, "", "  ")
}

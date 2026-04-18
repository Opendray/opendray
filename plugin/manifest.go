// Package plugin provides the plugin runtime for OpenDray.
package plugin

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Provider types.
const (
	ProviderTypeCLI   = "cli"   // Interactive CLI tool (claude, gemini, codex)
	ProviderTypeLocal = "local" // Local AI runtime (ollama, lmstudio)
	ProviderTypeShell = "shell" // Plain shell/terminal
	ProviderTypePanel = "panel" // UI panel plugin (docs viewer, preview, etc.)
)

// Provider is the unified model for all AI tools and terminal types.
// Every tool in OpenDray — including Claude and Terminal — is a Provider.
type Provider struct {
	// Identity
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Version     string `json:"version"`
	Type        string `json:"type"`               // cli | local | shell | panel
	Category    string `json:"category,omitempty"` // for panels: docs | files | custom

	// CLI specification
	CLI *CLISpec `json:"cli,omitempty"`

	// Capabilities
	Capabilities Capabilities `json:"capabilities"`

	// Configuration schema — drives the frontend form
	ConfigSchema []ConfigField `json:"configSchema"`
}

// CLISpec describes how to spawn this tool as a process.
type CLISpec struct {
	Command     string   `json:"command"`
	DefaultArgs []string `json:"defaultArgs,omitempty"`
	DetectCmd   string   `json:"detectCmd,omitempty"`
}

// Capabilities declares what this provider supports.
type Capabilities struct {
	Models         []ModelDef `json:"models"`
	SupportsResume bool       `json:"supportsResume"`
	SupportsStream bool       `json:"supportsStream"`
	SupportsImages bool       `json:"supportsImages"`
	SupportsMCP    bool       `json:"supportsMcp"`
	DynamicModels  bool       `json:"dynamicModels"` // models discovered at runtime (ollama, lmstudio)
}

// ModelDef describes an available model.
type ModelDef struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// ConfigField defines one configurable parameter.
// The frontend renders a form based on these fields.
type ConfigField struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Type        string `json:"type"` // string | secret | select | number | boolean | text | args
	Description string `json:"description,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
	Default     any    `json:"default,omitempty"`
	Options     []any  `json:"options,omitempty"` // for select type
	Required    bool   `json:"required,omitempty"`
	EnvVar      string `json:"envVar,omitempty"`    // maps to env var when launching
	CLIFlag     string `json:"cliFlag,omitempty"`   // when boolean=true or select value set, append this flag
	CLIValue    bool   `json:"cliValue,omitempty"`  // if true, append flag + value (--flag value); if false, flag only (--flag)
	Group       string `json:"group,omitempty"`     // visual grouping: "auth" | "runtime" | "advanced"
	DependsOn   string `json:"dependsOn,omitempty"` // only show when this key has a specific value
	DependsVal  string `json:"dependsVal,omitempty"`
}

// LoadManifest reads a manifest.json from the given plugin directory.
func LoadManifest(pluginDir string) (Provider, error) {
	path := filepath.Join(pluginDir, "manifest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return Provider{}, fmt.Errorf("plugin: read manifest %s: %w", path, err)
	}
	var p Provider
	if err := json.Unmarshal(data, &p); err != nil {
		return Provider{}, fmt.Errorf("plugin: parse manifest %s: %w", path, err)
	}
	if p.Name == "" {
		return Provider{}, fmt.Errorf("plugin: manifest %s: name is required", path)
	}
	if p.DisplayName == "" {
		p.DisplayName = p.Name
	}
	return p, nil
}

// ScanFS walks fsys under root recursively, mirroring [ScanPluginDir]
// but reading from any [fs.FS] rather than the local filesystem. This is
// how the runtime seeds itself from embedded plugin manifests on a
// fresh install where the `plugins/` directory isn't next to the binary.
//
// Walk rules match the filesystem version:
//   - stops descending into a plugin as soon as its manifest.json is found
//   - skips `_template/` and any directory whose name starts with "."
//   - surface-level errors (missing root, bad JSON) are logged by the
//     caller via a non-nil providers slice; this function only returns
//     an error for truly unexpected walk failures.
func ScanFS(fsys fs.FS, root string) ([]Provider, error) {
	entries, err := fs.ReadDir(fsys, root)
	if err != nil {
		// Embedded-only builds will always have a root; filesystem
		// fallbacks are allowed to be empty.
		return nil, nil
	}
	_ = entries

	var providers []Provider
	err = fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !d.IsDir() {
			return nil
		}
		name := d.Name()
		if path != root && (name == "_template" || strings.HasPrefix(name, ".")) {
			return fs.SkipDir
		}
		manifestPath := path + "/manifest.json"
		data, mErr := fs.ReadFile(fsys, manifestPath)
		if mErr != nil {
			return nil // not a plugin dir, keep descending
		}
		var p Provider
		if jErr := json.Unmarshal(data, &p); jErr != nil {
			return nil // corrupt manifest — skip, don't abort siblings
		}
		if p.Name == "" {
			return nil
		}
		if p.DisplayName == "" {
			p.DisplayName = p.Name
		}
		providers = append(providers, p)
		return fs.SkipDir // don't descend into a plugin
	})
	if err != nil {
		return nil, fmt.Errorf("plugin: scan fs %s: %w", root, err)
	}
	return providers, nil
}

// ScanPluginDir walks baseDir recursively and loads any directory that
// contains a manifest.json as a plugin. This allows grouping plugins into
// category subdirectories (e.g. plugins/agents/, plugins/panels/).
//
// Walking stops descending into a plugin once found — nested plugins are not
// supported. Directories named _template or starting with "." are skipped.
func ScanPluginDir(baseDir string) ([]Provider, error) {
	if _, err := os.Stat(baseDir); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("plugin: scan dir %s: %w", baseDir, err)
	}

	var providers []Provider
	err := filepath.WalkDir(baseDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !d.IsDir() {
			return nil
		}
		name := d.Name()
		if path != baseDir && (name == "_template" || strings.HasPrefix(name, ".")) {
			return filepath.SkipDir
		}
		if _, err := os.Stat(filepath.Join(path, "manifest.json")); err != nil {
			return nil // not a plugin dir, keep descending
		}
		p, err := LoadManifest(path)
		if err != nil {
			return nil // skip bad manifest, keep walking siblings
		}
		providers = append(providers, p)
		return filepath.SkipDir // don't descend into a plugin
	})
	if err != nil {
		return nil, fmt.Errorf("plugin: scan dir %s: %w", baseDir, err)
	}
	return providers, nil
}

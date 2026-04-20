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

// Plugin forms (v1). A plugin's runtime shape is declared by this
// field; legacy manifests that don't set it are mapped on the fly by
// [Provider.EffectiveForm] so the compat path works unchanged.
const (
	FormDeclarative = "declarative"
	FormWebview     = "webview"
	FormHost        = "host"
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

	// Required marks a plugin as load-bearing — it cannot be toggled off
	// or uninstalled via the normal API. Used for the three built-in
	// plugins (claude / terminal / file-browser) that the mobile shell
	// needs at all times. The flag is read from the manifest `required`
	// field and surfaced on /api/providers so Flutter can hide the
	// toggle/delete controls.
	Required bool `json:"required,omitempty"`

	// ─── v1 superset fields ──────────────────────────────────────
	// Every field below is optional. Legacy manifests leave them
	// zero and [Provider.IsV1] returns false so the existing compat
	// load path keeps working untouched.

	// Publisher is the marketplace-namespace owner id (e.g. "opendray").
	// Required on v1 manifests.
	Publisher string `json:"publisher,omitempty"`

	// Engines declares host-compatibility ranges. `engines.opendray`
	// is required on v1 manifests (semver range, e.g. "^1.0.0").
	Engines *EnginesV1 `json:"engines,omitempty"`

	// Form is the plugin runtime shape: declarative | webview | host.
	// Absent = derived from legacy Type via [Provider.EffectiveForm].
	Form string `json:"form,omitempty"`

	// Activation is the list of events that wake the plugin (empty =
	// lazy, never auto-activated; onStartup forces eager load).
	Activation []string `json:"activation,omitempty"`

	// Contributes declares every workbench slot this plugin fills.
	// M1 only interprets commands / statusBar / keybindings / menus;
	// other fields are accepted by schema but not yet rendered.
	Contributes *ContributesV1 `json:"contributes,omitempty"`

	// Permissions is the capability manifest. Host asks the user to
	// consent to these at install time and enforces them at every
	// bridge call.
	Permissions *PermissionsV1 `json:"permissions,omitempty"`

	// Host describes the sidecar runtime for form:"host" plugins. Ignored
	// (should be absent) for declarative and webview plugins. Validated
	// only when EffectiveForm() == FormHost and the current build supports
	// host-form (see plugin/host_os_*.go for the iOS gate).
	Host *HostV1 `json:"host,omitempty"`

	// V2Reserved is a forward-compat escape hatch: the host ignores
	// unknown keys inside it instead of erroring, so plugins written
	// against a later schema degrade gracefully on older hosts.
	V2Reserved json.RawMessage `json:"v2Reserved,omitempty"`
}

// HostV1 declares the sidecar configuration for form:"host" plugins.
// The supervisor (plugin/host.Supervisor, M3 T14) reads this to spawn
// and pipe stdio to the sidecar process. Absent on legacy manifests;
// absent on declarative / webview v1 manifests.
type HostV1 struct {
	// Entry is the command or script path executed by the supervisor.
	// For runtime="binary" it's an executable path; for runtime="node"
	// it's a JS file passed as argv[1] to `node`; similarly for deno /
	// python3 / bun. Must not contain "..".
	Entry string `json:"entry"`

	// Runtime is one of: binary | node | deno | python3 | bun | custom.
	// "custom" means entry is invoked directly and is expected to be
	// executable (shebang or native).
	Runtime string `json:"runtime,omitempty"`

	// Platforms maps "<os>-<arch>" to a platform-specific entry override
	// (binary downloads on different OSes). Keys match the regex
	// ^(linux|darwin|windows)-(x64|arm64)$. Empty map = Entry used
	// unchanged on every platform.
	Platforms map[string]string `json:"platforms,omitempty"`

	// Protocol is the stdio wire protocol. Only "jsonrpc-stdio"
	// (JSON-RPC 2.0 with LSP Content-Length framing) is accepted today.
	Protocol string `json:"protocol,omitempty"`

	// Restart is one of: on-failure (default) | always | never.
	Restart string `json:"restart,omitempty"`

	// Env extends the sidecar's environment. Keys must match
	// ^[A-Z_][A-Z0-9_]*$ (standard env-var name rules).
	Env map[string]string `json:"env,omitempty"`

	// Cwd overrides the sidecar's working directory. Relative paths are
	// resolved against the plugin's install dir. Absolute paths must
	// pass through the capability gate's fs.read/fs.write grants.
	Cwd string `json:"cwd,omitempty"`

	// IdleShutdownMinutes is the idle timeout before the supervisor
	// shuts the sidecar down. 0 = use supervisor default (10 min).
	IdleShutdownMinutes int `json:"idleShutdownMinutes,omitempty"`
}

// HostRuntimeNode / HostRuntimeDeno / ... name the allowed runtime values.
const (
	HostRuntimeBinary  = "binary"
	HostRuntimeNode    = "node"
	HostRuntimeDeno    = "deno"
	HostRuntimePython3 = "python3"
	HostRuntimeBun     = "bun"
	HostRuntimeCustom  = "custom"
)

// HostProtocolJSONRPCStdio is the only protocol shipped in M3.
const HostProtocolJSONRPCStdio = "jsonrpc-stdio"

// HostRestart* enumerates valid Restart values.
const (
	HostRestartOnFailure = "on-failure"
	HostRestartAlways    = "always"
	HostRestartNever     = "never"
)

// EnginesV1 declares plugin → host version compatibility.
type EnginesV1 struct {
	Opendray string `json:"opendray"`
	Node     string `json:"node,omitempty"`
	Deno     string `json:"deno,omitempty"`
}

// ContributesV1 holds every workbench-slot contribution.
// M1 honours commands/statusBar/keybindings/menus; M2 adds
// activityBar/views/panels (parsed here from day one, rendered
// by the Flutter shell once the workbench webview host lands).
type ContributesV1 struct {
	Commands    []CommandV1              `json:"commands,omitempty"`
	StatusBar   []StatusBarItemV1        `json:"statusBar,omitempty"`
	Keybindings []KeybindingV1           `json:"keybindings,omitempty"`
	Menus       map[string][]MenuEntryV1 `json:"menus,omitempty"`

	// ── M2 webview slots ───────────────────────────────────────────
	ActivityBar []ActivityBarItemV1 `json:"activityBar,omitempty"`
	Views       []ViewV1            `json:"views,omitempty"`
	Panels      []PanelV1           `json:"panels,omitempty"`
}

// CommandV1 registers a runnable command with the workbench.
type CommandV1 struct {
	ID       string        `json:"id"`
	Title    string        `json:"title"`
	Icon     string        `json:"icon,omitempty"`
	Category string        `json:"category,omitempty"`
	When     string        `json:"when,omitempty"`
	Run      *CommandRunV1 `json:"run,omitempty"`
}

// CommandRunV1 is the declarative action dispatcher. The `kind` field
// is the discriminator; only `notify` / `openUrl` / `exec` / `runTask`
// are live in M1 — `host` and `openView` return EUNAVAIL.
type CommandRunV1 struct {
	Kind    string            `json:"kind"`
	Method  string            `json:"method,omitempty"`
	Args    []json.RawMessage `json:"args,omitempty"`
	ViewID  string            `json:"viewId,omitempty"`
	URL     string            `json:"url,omitempty"`
	Message string            `json:"message,omitempty"`
	TaskID  string            `json:"taskId,omitempty"`
}

// StatusBarItemV1 renders a small label/icon in the workbench status strip.
type StatusBarItemV1 struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	Tooltip   string `json:"tooltip,omitempty"`
	Command   string `json:"command,omitempty"`
	Alignment string `json:"alignment,omitempty"`
	Priority  int    `json:"priority,omitempty"`
}

// KeybindingV1 binds a key (+ optional platform-specific mac key) to a command id.
type KeybindingV1 struct {
	Command string `json:"command"`
	Key     string `json:"key"`
	Mac     string `json:"mac,omitempty"`
	When    string `json:"when,omitempty"`
}

// MenuEntryV1 inserts a command (or submenu) into a named menu path.
type MenuEntryV1 struct {
	Command string `json:"command,omitempty"`
	Submenu string `json:"submenu,omitempty"`
	When    string `json:"when,omitempty"`
	Group   string `json:"group,omitempty"`
}

// ActivityBarItemV1 puts an icon on the workbench activity bar — the
// vertical strip of top-level entry points on the left (tablet/desktop)
// or the bottom tab row (phone). Tapping it opens the associated view.
type ActivityBarItemV1 struct {
	ID     string `json:"id"`
	Icon   string `json:"icon"`             // emoji OR relative path into the plugin's ui/
	Title  string `json:"title"`            // shown as tooltip + accessible label
	ViewID string `json:"viewId,omitempty"` // which view opens on activation; may be ommitted if an action handler is wired later
}

// ViewV1 declares a workbench view — either a webview-hosted plugin UI
// or a declarative schema-driven form. Rendered inside the activity
// bar's primary panel or (on phone) as a full-screen pane.
//
// Container selects where the view is anchored; Render selects the
// view backend. When Render is "webview", Entry must be a path
// relative to the plugin's ui/ directory.
type ViewV1 struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Container string `json:"container,omitempty"` // "activityBar" (default) | "panel" | "sidebar"
	Icon      string `json:"icon,omitempty"`
	When      string `json:"when,omitempty"`  // context expression, e.g. "workspaceOpen"
	Render    string `json:"render,omitempty"` // "webview" (default) | "declarative"
	Entry     string `json:"entry,omitempty"`
}

// PanelV1 declares a bottom / right panel, sibling to terminal and
// logs. Useful for transient workspaces that benefit from a strip view
// rather than a full primary view.
type PanelV1 struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Icon     string `json:"icon,omitempty"`
	Position string `json:"position,omitempty"` // "bottom" (default) | "right"
	Render   string `json:"render,omitempty"`   // "webview" (default)
	Entry    string `json:"entry,omitempty"`
}

// PermissionsV1 is the install-time capability grant.
//
// Polymorphic fields (fs/exec/http) accept either a boolean (grant-all
// or grant-none) or a more specific shape (allowed paths / commands /
// URLs). v1 parsing preserves the raw JSON so each call-site can
// interpret under the shape that bridge method expects — keeps T1
// additive and lets T2 (validator) pin down semantics without a
// schema rewrite.
type PermissionsV1 struct {
	Fs        json.RawMessage `json:"fs,omitempty"`
	Exec      json.RawMessage `json:"exec,omitempty"`
	HTTP      json.RawMessage `json:"http,omitempty"`
	Session   string          `json:"session,omitempty"`
	Storage   bool            `json:"storage,omitempty"`
	Secret    bool            `json:"secret,omitempty"`
	Clipboard string          `json:"clipboard,omitempty"`
	Telegram  bool            `json:"telegram,omitempty"`
	Git       string          `json:"git,omitempty"`
	LLM       bool            `json:"llm,omitempty"`
	Events    []string        `json:"events,omitempty"`
}

// IsV1 reports whether this manifest opted into the v1 plugin contract.
// Opting in requires both `publisher` and `engines.opendray` — the two
// fields that identify a plugin author and gate host compatibility.
// Zero-valued manifests (the legacy compat path) always return false.
func (p Provider) IsV1() bool {
	return p.Publisher != "" && p.Engines != nil && p.Engines.Opendray != ""
}

// EffectiveForm returns the runtime shape the host should use. Explicit
// `form` wins; otherwise legacy `type` is mapped:
//
//	cli | local | shell  →  host
//	panel                →  declarative  (M1 compat path; M2 introduces webview)
//	anything else        →  declarative
func (p Provider) EffectiveForm() string {
	if p.Form != "" {
		return p.Form
	}
	switch p.Type {
	case ProviderTypeCLI, ProviderTypeLocal, ProviderTypeShell:
		return FormHost
	case ProviderTypePanel:
		return FormDeclarative
	}
	return FormDeclarative
}

// HasHostBackend reports whether the plugin ships a sidecar the
// supervisor should spawn on demand. True for:
//
//   - Classic form:"host" plugins (the M3 case).
//   - form:"webview" plugins that ALSO declare a host:{} block —
//     the "combined form" M5 B1 introduces so webview UIs can call
//     their own privileged sidecar via opendray.commands.execute.
//
// The rule is uniform: a host block means a sidecar, regardless of
// which form drives the primary UI surface.
func (p Provider) HasHostBackend() bool {
	if p.Host == nil {
		return false
	}
	return p.EffectiveForm() == FormHost || p.EffectiveForm() == FormWebview
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

// ConfigField defines one configurable parameter. The frontend
// renders a form based on these fields; v1 plugins consume the
// saved values at runtime through the storage / secret bridge
// namespaces with the reserved key prefix "__config.<key>".
//
// v1 Type values:
//
//	string  → text input               → plugin_kv
//	number  → numeric input             → plugin_kv (stored as JSON string)
//	bool    → switch                    → plugin_kv ("true" / "false")
//	select  → dropdown                  → plugin_kv (must be in Options)
//	secret  → password input            → plugin_secret (AES-GCM)
//
// Legacy Type values ("boolean", "text", "args") remain accepted for
// pre-v1 plugins but are not rendered by the v1 Hub config form. Use
// "bool" on new manifests. EnvVar / CLIFlag / CLIValue / DependsOn
// are legacy-only and ignored by the v1 config pipeline.
type ConfigField struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
	Default     any    `json:"default,omitempty"`
	Options     []any  `json:"options,omitempty"` // for select type
	Required    bool   `json:"required,omitempty"`
	EnvVar      string `json:"envVar,omitempty"`    // legacy: maps to env var when launching
	CLIFlag     string `json:"cliFlag,omitempty"`   // legacy: CLI flag append
	CLIValue    bool   `json:"cliValue,omitempty"`  // legacy: append value after flag
	Group       string `json:"group,omitempty"`     // visual grouping: "auth" | "runtime" | "advanced"
	DependsOn   string `json:"dependsOn,omitempty"` // legacy: show-when gate
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

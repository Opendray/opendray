// Package catalog serves the embedded CLI provider manifests
// (claude, codex, gemini, shell) plus per-provider user config.
//
// Per ADR 0004 the catalog backend ships only 4 manifests — anything
// that v1 had as a "plugin" but is actually a UI tool / messaging
// channel / external app belongs in the client / channel hub /
// integration registry instead.
//
// Manifests are embedded as JSON via go:embed; the providers table
// stores per-id user state (enabled flag, user config values) and a
// hash of the loaded manifest for change detection.
package catalog

// Manifest is the v2 declarative manifest schema. Compared to v1 we
// drop publisher / engines / form / activation / required / detectCmd
// / dynamicModels (see ADR 0004) and gain mcpServers as a *user
// config* (not a manifest field) since MCP injection is a session
// spawn-time hook, not a separate plugin.
type Manifest struct {
	ID            string `json:"id"`
	DisplayName   string `json:"displayName"`
	DisplayNameZh string `json:"displayName_zh,omitempty"`
	Description   string `json:"description"`
	DescriptionZh string `json:"description_zh,omitempty"`
	Icon          string `json:"icon"`
	Version       string `json:"version"`
	Kind          string `json:"kind"` // "cli" | "shell"
	Executable    string `json:"executable"`
	// NpmPackage is the npm package the executable ships in, used to
	// probe the latest published version (and, in Phase 2, to update
	// the CLI). Empty for non-npm providers such as the builtin shell.
	NpmPackage string `json:"npmPackage,omitempty"`
	// ModelFlag is the CLI flag used to select a model (e.g. "--model").
	// When set and the operator has configured a default `model`, it is
	// passed on every spawn. Empty for providers with no model concept.
	ModelFlag string `json:"modelFlag,omitempty"`
	// KnownModels is a maintained suggestion list (kept current per
	// release) the UI offers as "Suggested" — the CLIs expose no live
	// model-list command, so this is the auto-populate source. Operators
	// edit the actual list (provider config `models`) on top.
	KnownModels  []string      `json:"knownModels,omitempty"`
	DefaultArgs  []string      `json:"defaultArgs,omitempty"`
	Capabilities Capabilities  `json:"capabilities"`
	ConfigSchema []ConfigField `json:"configSchema,omitempty"`
	UI           *UI           `json:"ui,omitempty"`
}

// Capabilities is descriptive metadata used by clients to render the
// right affordances (e.g. show image upload only when supportsImages).
// Backend does not enforce these — they are advisory.
type Capabilities struct {
	SupportsResume bool `json:"supportsResume"`
	SupportsStream bool `json:"supportsStream"`
	SupportsImages bool `json:"supportsImages"`
	SupportsMcp    bool `json:"supportsMcp"`
}

// ConfigField describes one user-configurable setting that the client
// renders as a form input. Mirrors v1's configSchema entries minus
// fields tied to v1's removed plugin runtime.
type ConfigField struct {
	Key           string   `json:"key"`
	Label         string   `json:"label"`
	LabelZh       string   `json:"label_zh,omitempty"`
	Type          string   `json:"type"` // string | number | boolean | select | secret | args
	Group         string   `json:"group,omitempty"`
	Default       any      `json:"default,omitempty"`
	Options       []string `json:"options,omitempty"`
	Placeholder   string   `json:"placeholder,omitempty"`
	Description   string   `json:"description,omitempty"`
	DescriptionZh string   `json:"description_zh,omitempty"`
	EnvVar        string   `json:"envVar,omitempty"`
	CliFlag       string   `json:"cliFlag,omitempty"`
	CliValue      bool     `json:"cliValue,omitempty"`
	DependsOn     string   `json:"dependsOn,omitempty"`
	DependsVal    any      `json:"dependsVal,omitempty"`
}

// UI is the client-render contribution. Activity bar item + view
// list. M5 client decides how to render these.
type UI struct {
	ActivityBar *ActivityBarItem `json:"activityBar,omitempty"`
	Views       []View           `json:"views,omitempty"`
}

type ActivityBarItem struct {
	ID    string `json:"id"`
	Icon  string `json:"icon"`
	Title string `json:"title"`
}

type View struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Container string `json:"container,omitempty"` // default: "activityBar"
}

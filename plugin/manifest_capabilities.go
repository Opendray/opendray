package plugin

// manifest_capabilities.go — M6 capability declarations.
//
// These types live under contributes.{providers,channels,forges,mcpServers}
// and are the *static* declaration of every runtime capability a plugin
// will register at activation. They mirror but do not embed the
// plugin/api capability interfaces — the manifest is the control plane,
// the api package is the data plane.
//
// Validation rule: every id present in a capability declaration MUST be
// registered by the plugin's Register call (or vice versa, depending on
// the host's strictness setting). Host enforces this at activation time.

// ProviderContributionV1 declares one LLM provider this plugin owns.
//
// id is the registry key — must be unique across all loaded plugins
// and must be the same string the plugin's api.Provider impl returns
// from ID().
//
// displayName/description/icon are surfaced in the model picker UI and
// the marketplace listing. They may be omitted for headless deployments.
type ProviderContributionV1 struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName,omitempty"`
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon,omitempty"`

	// Categories is a list of opaque tags used by the UI to group
	// providers (e.g. "cloud", "on-device", "free", "vision").
	Categories []string `json:"categories,omitempty"`
}

// ChannelContributionV1 declares one messaging channel this plugin owns.
type ChannelContributionV1 struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName,omitempty"`
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon,omitempty"`
}

// ForgeContributionV1 declares one source-control forge this plugin owns.
type ForgeContributionV1 struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName,omitempty"`
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon,omitempty"`

	// Kind is one of "github" | "gitlab" | "gitea" | "bitbucket" |
	// "custom". Used by the source-control UI to pick the right
	// metadata layout. Empty defaults to "custom".
	Kind string `json:"kind,omitempty"`
}

// McpServerContributionV1 declares one MCP server this plugin owns.
type McpServerContributionV1 struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName,omitempty"`
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon,omitempty"`
}

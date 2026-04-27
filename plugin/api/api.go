package api

import (
	"context"
	"log/slog"
)

// PluginAPI is the per-plugin handle the host hands to a plugin's
// Register function at activation time. Every registration the plugin
// performs flows through this interface — there is no other path into
// the host's capability registry.
//
// The host MUST construct a fresh PluginAPI per plugin so that
// registrations carry an implicit owner. A plugin cannot register on
// behalf of another plugin.
//
// Lifetime: PluginAPI is valid until the plugin is unloaded. After
// unload, calling any method other than Plugin() returns
// ErrPluginUnloaded. Capabilities should observe Context() cancellation
// for cooperative shutdown.
type PluginAPI interface {
	// Plugin returns metadata about the plugin this API handle is
	// bound to. Always safe to call.
	Plugin() PluginInfo

	// Logger returns a structured logger pre-tagged with the plugin's
	// id. Plugins should prefer this over creating their own.
	Logger() *slog.Logger

	// Context returns the plugin lifetime context. Cancelled when the
	// plugin is being unloaded. Long-running goroutines launched by
	// the plugin must respect this and exit cleanly.
	Context() context.Context

	// ── Capability registrars ───────────────────────────────────────
	//
	// Each Register* call validates that the capability id matches one
	// the plugin declared in its manifest's contributes section. Calls
	// for an undeclared id return ErrUndeclaredCapability — this is
	// what gives manifests their "control plane" status (the host can
	// know what a plugin owns without loading code).

	RegisterProvider(p Provider) error
	RegisterChannel(c Channel) error
	RegisterForge(f Forge) error
	RegisterMcpServer(m McpServer) error

	// Hooks returns the host event/hook bus scoped to this plugin.
	// Subscriptions are auto-cancelled at plugin unload.
	Hooks() HookBus
}

// PluginInfo carries identity fields the plugin may want to log or
// surface in diagnostics. It is a value type — copies are cheap and
// safe.
type PluginInfo struct {
	// Name is the manifest `name` field (the registry key).
	Name string
	// Version is the manifest `version` field (semver).
	Version string
	// Publisher is the marketplace namespace owner id.
	Publisher string
}

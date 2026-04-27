// Package api defines the stable Go contract every OpenDray plugin uses
// to register capabilities (providers, channels, forges, MCP servers) and
// hooks with the host runtime.
//
// In-process built-in plugins call this contract directly. External
// host-form plugins call it indirectly: the bridge translates their
// JSON-RPC envelopes into method calls on the same interface, so a
// capability registered from either side ends up in the same registry
// (plugin/capreg) and is indistinguishable to the gateway.
//
// This package is the single source of truth for the plugin contract.
// Adding or changing methods here is a deliberate, versioned event —
// see APIVersion below.
package api

// APIVersion is the semantic version of the PluginAPI contract.
//
// MAJOR — incompatible changes (removing methods, changing signatures).
// MINOR — backwards-compatible additions (new optional capabilities).
// PATCH — clarifications, doc fixes, no surface change.
//
// External plugins declare their target version in
// manifest.engines.pluginAPI; the host refuses to load plugins built
// against a major version it does not implement.
const APIVersion = "1.0.0"

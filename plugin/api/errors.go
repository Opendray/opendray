package api

import "errors"

// Sentinel errors returned by PluginAPI implementations. Callers check
// with errors.Is. Implementations may wrap these with additional
// context using fmt.Errorf("...: %w", err).
var (
	// ErrPluginUnloaded is returned when a plugin tries to use its
	// PluginAPI handle after the host has begun unloading it. The
	// only safe call after unload is Plugin().
	ErrPluginUnloaded = errors.New("plugin: api handle is unloaded")

	// ErrUndeclaredCapability is returned by Register* methods when
	// the capability id is not present in the plugin's manifest
	// contributes section. Manifests are the control-plane source of
	// truth; runtime registrations cannot exceed declared scope.
	ErrUndeclaredCapability = errors.New("plugin: capability id not declared in manifest")

	// ErrDuplicateCapability is returned when a capability id is
	// already registered (by this plugin or another). Capability ids
	// are globally unique across all loaded plugins.
	ErrDuplicateCapability = errors.New("plugin: capability id already registered")
)

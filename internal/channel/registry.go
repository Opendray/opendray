package channel

import (
	"encoding/json"
	"log/slog"
	"sort"
	"sync"
)

// Factory builds a Channel from its DB id and raw config JSON.
type Factory func(id string, config json.RawMessage, log *slog.Logger) (Channel, error)

var (
	registryMu sync.RWMutex
	registry   = make(map[string]Factory)
)

// Register adds a kind→factory mapping. Called from package init() in
// telegram / slack / etc. impl packages.
func Register(kind string, f Factory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[kind] = f
}

// Lookup returns the factory for kind, or nil if not registered.
func Lookup(kind string) Factory {
	registryMu.RLock()
	defer registryMu.RUnlock()
	return registry[kind]
}

// KnownKinds returns the registered kinds in sorted order. Used by
// REST to surface the supported channel types.
func KnownKinds() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]string, 0, len(registry))
	for k := range registry {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

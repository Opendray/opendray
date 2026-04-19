package gateway

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/opendray/opendray/plugin"
	"github.com/opendray/opendray/plugin/bridge"
)

// PathVarResolver implements bridge.PathVarResolver for the live
// gateway. It reads a plugin's current data directory from
// ${PluginsDataDir}/<plugin>/data/ (created lazily with 0700), the
// host user's home dir from os.UserHomeDir, tmp from os.TempDir, and
// — in M3 — leaves workspace empty so fs grants anchored on
// ${workspace} fail closed until M4 threads the active session's cwd
// through here.
//
// Plugins that need per-call workspace context can stash it in their
// own storage (via opendray.storage) and resolve paths client-side.
type PathVarResolver struct {
	// DataDir is the root for every plugin's per-install data
	// directory. Matches cfg.PluginsDataDir from config.toml.
	DataDir string

	// Providers looks up the installed version so the plugin-scoped
	// data path is ${DataDir}/<plugin>/<version>/data/. Optional: if
	// nil or the lookup misses, the version suffix is dropped.
	Providers providerLookup

	once       sync.Once
	cachedHome string
}

// providerLookup is a narrow interface over *plugin.Runtime.Get so
// tests don't need to wire a full Runtime. Only the Get method is
// used — a v1 match is fine; legacy manifests have Version too.
type providerLookup interface {
	Get(name string) (plugin.Provider, bool)
}

// Resolve implements bridge.PathVarResolver. Returned PathVarCtx is
// ready to pass to bridge.Gate.CheckExpanded.
func (r *PathVarResolver) Resolve(_ context.Context, pluginName string) (bridge.PathVarCtx, error) {
	r.once.Do(func() {
		if h, err := os.UserHomeDir(); err == nil {
			r.cachedHome = h
		}
	})

	version := ""
	if r.Providers != nil {
		if p, ok := r.Providers.Get(pluginName); ok {
			version = p.Version
		}
	}
	dataDir := filepath.Join(r.DataDir, pluginName)
	if version != "" {
		dataDir = filepath.Join(dataDir, version)
	}
	dataDir = filepath.Join(dataDir, "data")
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return bridge.PathVarCtx{}, fmt.Errorf("path_vars: mkdir %s: %w", dataDir, err)
	}

	return bridge.PathVarCtx{
		// Workspace intentionally empty in M3 — M4 threads the
		// active session's cwd via a session-aware variant.
		Workspace: "",
		Home:      r.cachedHome,
		DataDir:   dataDir,
		Tmp:       os.TempDir(),
	}, nil
}

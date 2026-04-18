// Package plugins bundles the on-disk plugin manifests into the Go
// binary. Every subdirectory of plugins/ that contains a manifest.json
// is exposed through FS so the plugin runtime can seed the DB even on a
// fresh install where the `plugins/` directory isn't next to the binary
// (LXC / release-binary deploys, Docker, etc.).
//
// Users can still drop extra plugins into a filesystem pluginDir —
// plugin.Runtime.LoadAll merges both sources with filesystem taking
// precedence so forks / overrides are honoured.
package plugins

import "embed"

//go:embed all:agents all:panels
var FS embed.FS

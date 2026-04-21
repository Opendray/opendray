// Package plugins bundles the on-disk plugin manifests into the Go
// binary. Every manifest under plugins/builtin/ is exposed through FS
// so the plugin runtime can seed the DB even on a fresh install where
// the `plugins/` directory isn't next to the binary (LXC /
// release-binary deploys, Docker, etc.).
//
// The `builtin/` subdir is the single home for everything that ships
// with OpenDray — agents (CLI-style) and panels (GUI-style) live
// side-by-side. Third-party plugins don't land here; they flow through
// the install pipeline into the runtime data dir. plugins/examples/
// stays outside the embed and is a reference-only tree for publisher
// CLI work.
//
// Users can still drop extra plugins into a filesystem pluginDir —
// plugin.Runtime.LoadAll merges both sources with filesystem taking
// precedence so forks / overrides are honoured.
package plugins

import "embed"

//go:embed all:builtin
var FS embed.FS

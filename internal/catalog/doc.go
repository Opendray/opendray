// Package catalog serves the embedded CLI provider manifests
// (claude, codex, gemini, ...) plus per-provider user config.
//
// Responsibilities (per design §8.4):
//   - Load manifests from internal/catalog/builtin/ via go:embed.
//   - Persist per-provider user config + enabled state in postgres.
//   - Expose a read-only manifest API and a config-write API.
//
// Replaces v1's plugin/manifest + plugins/builtin without bridge / install /
// market / host machinery. v1 had 16 declarative manifests and 0 host /
// bridge plugins, so we keep only the part that was actually used.
package catalog

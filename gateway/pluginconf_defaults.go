package gateway

// Plugin config defaults & path expansion.
//
// Built-in panel plugins (file-browser, log-viewer, task-runner) declare
// `allowedRoots` / `defaultPath` in their manifest. Before this helper,
// users hit HTTP 400 "plugin not configured" immediately after install
// because effectiveConfig does not inject manifest Default values.
//
// resolveRoots / resolveDefaultPath consult the user-saved value first,
// then fall back to the field's manifest Default, and finally expand
// $HOME / ~ so the value is a real absolute path at runtime.

import (
	"os"
	"strings"

	"github.com/opendray/opendray/plugin"
)

// expandUserPath turns a user-facing path string into an absolute path
// with environment variables and `~` expanded.
//
// Rules:
//   - Empty / whitespace-only input returns "".
//   - Leading `~` or `~/` is replaced by $HOME.
//   - `~other-user` forms are left untouched (no /etc/passwd lookup).
//   - $VAR and ${VAR} are expanded via os.ExpandEnv. Undefined vars
//     become empty strings (stdlib behaviour).
//   - Surrounding whitespace is trimmed.
func expandUserPath(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if s == "~" {
		s = "$HOME"
	} else if strings.HasPrefix(s, "~/") {
		s = "$HOME" + s[1:]
	}
	return os.ExpandEnv(s)
}

// resolveRoots reads a comma-separated directory list from cfg[key]. If
// the user value is empty / whitespace / missing, the manifest field's
// string Default is used instead. Each entry is trimmed, empty entries
// filtered, and each survivor is passed through expandUserPath so the
// caller gets absolute on-disk paths.
//
// Returns nil when neither the user value nor the schema default yield
// any entries — callers keep their existing empty-slice behaviour for
// backwards compatibility.
func resolveRoots(cfg map[string]any, schema []plugin.ConfigField, key string) []string {
	raw := stringVal(cfg, key, "")
	if strings.TrimSpace(raw) == "" {
		if def, ok := schemaStringDefault(schema, key); ok {
			raw = def
		}
	}
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(raw, ",") {
		p := expandUserPath(part)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

// resolveDefaultPath reads a single path value from cfg[key], falling
// back to the manifest field's string Default when missing. The result
// is expanded via expandUserPath so downstream code never sees a literal
// "$HOME" or "~".
func resolveDefaultPath(cfg map[string]any, schema []plugin.ConfigField, key string) string {
	raw := stringVal(cfg, key, "")
	if strings.TrimSpace(raw) == "" {
		if def, ok := schemaStringDefault(schema, key); ok {
			raw = def
		}
	}
	return expandUserPath(raw)
}

// schemaStringDefault returns the string form of a field's manifest
// Default. Non-string defaults (numbers, booleans) are ignored — those
// field types don't participate in path resolution.
func schemaStringDefault(schema []plugin.ConfigField, key string) (string, bool) {
	for _, f := range schema {
		if f.Key != key {
			continue
		}
		if s, ok := f.Default.(string); ok {
			return s, true
		}
		return "", false
	}
	return "", false
}

// Package plugin — manifest v1 validator (T2).
//
// ValidateV1 checks a [Provider] against every rule defined in
// docs/plugin-platform/02-manifest.md §JSON Schema.
// Regex patterns are copy-pasted verbatim from the doc; each compile site
// cites the line number so drift is immediately visible in diff.
package plugin

import (
	"fmt"
	"regexp"
	"strings"
)

// ─── package-level compiled regexes ─────────────────────────────────────────
// All patterns come verbatim from docs/plugin-platform/02-manifest.md §JSON Schema.
// Note: JSON source uses \\d; Go regex uses \d.

// reName — 02-manifest.md line 18.
var reName = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,62}[a-z0-9])?$`)

// reVersion — 02-manifest.md line 19.
var reVersion = regexp.MustCompile(`^\d+\.\d+\.\d+(-[A-Za-z0-9.-]+)?$`)

// rePublisher — 02-manifest.md line 20.
var rePublisher = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,38}[a-z0-9])?$`)

// reActivation — 02-manifest.md line 84 (items.pattern).
// JSON source: "^(onStartup|onCommand:[a-z0-9._-]+|onView:[a-z0-9._-]+|onSession:(start|stop|idle|output)|onLanguage:[a-z0-9_-]+|onFile:[^\\s]+|onSchedule:cron:.+)$"
// cron: segment accepts any non-empty tail so five-field expressions with
// spaces ("0 * * * *") are valid — the original [^\s]+ shape rejected them.
var reActivation = regexp.MustCompile(`^(onStartup|onCommand:[a-z0-9._-]+|onView:[a-z0-9._-]+|onSession:(start|stop|idle|output)|onLanguage:[a-z0-9_-]+|onFile:[^\s]+|onSchedule:cron:.+)$`)

// reCommandID — derived from the activation pattern's command-id fragment and
// used in 02-manifest.md §command $defs to describe id format: ^[a-z0-9._-]+$
var reCommandID = regexp.MustCompile(`^[a-z0-9._-]+$`)

// ─── ValidationError ────────────────────────────────────────────────────────

// ValidationError names a single failed rule at a path inside the manifest.
// Pretty-print format: "<path>: <msg>" for join-into-one-error reporting.
type ValidationError struct {
	Path string
	Msg  string
}

// Error implements the error interface.
// Format: "<Path>: <Msg>"
func (v ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", v.Path, v.Msg)
}

// ─── ValidateV1 ─────────────────────────────────────────────────────────────

// ValidateV1 returns nil for legacy manifests (IsV1()==false) — it
// short-circuits so the compat path never fails validation. For v1 manifests
// it returns a slice of every rule violation found; empty slice == valid.
func ValidateV1(p Provider) []ValidationError {
	if !p.IsV1() {
		return nil
	}

	var errs []ValidationError

	// name — required, pattern
	if err := validateName(p.Name); err != nil {
		errs = append(errs, ValidationError{Path: "name", Msg: err.Error()})
	}

	// publisher — required for v1, pattern
	if err := validatePublisher(p.Publisher); err != nil {
		errs = append(errs, ValidationError{Path: "publisher", Msg: err.Error()})
	}

	// version — required, semver pattern
	if err := validateSemver(p.Version); err != nil {
		errs = append(errs, ValidationError{Path: "version", Msg: err.Error()})
	}

	// engines.opendray — required on v1, non-empty
	if err := validateEngines(p.Engines); err != nil {
		errs = append(errs, ValidationError{Path: "engines.opendray", Msg: err.Error()})
	}

	// form — if set must be "declarative" | "webview" | "host"
	if p.Form != "" {
		switch p.Form {
		case FormDeclarative, FormWebview, FormHost:
			// valid
		default:
			errs = append(errs, ValidationError{
				Path: "form",
				Msg:  fmt.Sprintf("must be one of declarative|webview|host, got %q", p.Form),
			})
		}
	}

	// activation[*]
	for i, ev := range p.Activation {
		if err := validateActivationEvent(ev); err != nil {
			errs = append(errs, ValidationError{
				Path: fmt.Sprintf("activation[%d]", i),
				Msg:  err.Error(),
			})
		}
	}

	// contributes.*
	if p.Contributes != nil {
		errs = append(errs, validateContributes(p.Contributes)...)
	}

	// permissions.*
	if p.Permissions != nil {
		errs = append(errs, validatePermissions(p.Permissions)...)
	}

	// host.* — only inspected when the effective form is "host". The
	// iOS build-tag gate (plugin/host_os_*.go) additionally refuses
	// host-form entirely; validateHostV1 returns a single error in
	// that case.
	if p.EffectiveForm() == FormHost {
		errs = append(errs, validateHostV1(p.Host)...)
	}

	// configSchema.*
	errs = append(errs, validateConfigSchema(p.ConfigSchema)...)

	return errs
}

// configFieldKeyPattern caps the identifier shape so sidecar code can
// read each field back as a plain JSON property. Matches common env-
// var / JS-identifier conventions.
var configFieldKeyPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)

// configFieldTypes is the closed set of widget types the v1 Hub knows
// how to render. "boolean" is accepted as a legacy alias for "bool";
// "text" / "args" are also kept for legacy-provider manifests that
// pre-date the v1 form schema — they render as plain string inputs.
var configFieldTypes = map[string]struct{}{
	"string":  {},
	"text":    {}, // legacy alias of string (multi-line hint only)
	"number":  {},
	"bool":    {},
	"boolean": {}, // legacy alias of bool
	"select":  {},
	"secret":  {},
	"args":    {}, // legacy CLI-args editor; ignored by v1 pipeline
}

// validateConfigSchema enforces key uniqueness, a stable Key shape, a
// closed type enum, and Options presence for type=="select". Missing
// or empty schemas are valid (no config surface = no form).
func validateConfigSchema(schema []ConfigField) []ValidationError {
	if len(schema) == 0 {
		return nil
	}
	var errs []ValidationError
	seen := make(map[string]struct{}, len(schema))
	for i, f := range schema {
		path := fmt.Sprintf("configSchema[%d]", i)
		if f.Key == "" {
			errs = append(errs, ValidationError{Path: path + ".key", Msg: "required"})
		} else if !configFieldKeyPattern.MatchString(f.Key) {
			errs = append(errs, ValidationError{
				Path: path + ".key",
				Msg:  fmt.Sprintf("must match %s, got %q", configFieldKeyPattern.String(), f.Key),
			})
		} else if _, dup := seen[f.Key]; dup {
			errs = append(errs, ValidationError{
				Path: path + ".key",
				Msg:  fmt.Sprintf("duplicate key %q", f.Key),
			})
		} else {
			seen[f.Key] = struct{}{}
		}

		if f.Label == "" {
			errs = append(errs, ValidationError{Path: path + ".label", Msg: "required"})
		}

		if _, ok := configFieldTypes[f.Type]; !ok {
			errs = append(errs, ValidationError{
				Path: path + ".type",
				Msg:  fmt.Sprintf("must be one of string|number|bool|select|secret, got %q", f.Type),
			})
		}

		if f.Type == "select" && len(f.Options) == 0 {
			errs = append(errs, ValidationError{
				Path: path + ".options",
				Msg:  "required when type is select",
			})
		}
	}
	return errs
}

// ─── Exported helpers ────────────────────────────────────────────────────────

// validateName checks the name field against the pattern on 02-manifest.md line 18.
// Pattern: ^[a-z0-9]([a-z0-9-]{0,62}[a-z0-9])?$
func validateName(name string) error {
	if name == "" {
		return fmt.Errorf("must not be empty")
	}
	if !reName.MatchString(name) {
		return fmt.Errorf("must match ^[a-z0-9]([a-z0-9-]{0,62}[a-z0-9])?$, got %q", name)
	}
	return nil
}

// validatePublisher checks the publisher field against the pattern on 02-manifest.md line 20.
// Pattern: ^[a-z0-9]([a-z0-9-]{0,38}[a-z0-9])?$
func validatePublisher(pub string) error {
	if pub == "" {
		return fmt.Errorf("must not be empty (required for v1)")
	}
	if !rePublisher.MatchString(pub) {
		return fmt.Errorf("must match ^[a-z0-9]([a-z0-9-]{0,38}[a-z0-9])?$, got %q", pub)
	}
	return nil
}

// validateSemver checks the version field against the pattern on 02-manifest.md line 19.
// Pattern: ^\d+\.\d+\.\d+(-[A-Za-z0-9.-]+)?$
func validateSemver(v string) error {
	if v == "" {
		return fmt.Errorf("must not be empty")
	}
	if !reVersion.MatchString(v) {
		return fmt.Errorf("must match ^\\d+\\.\\d+\\.\\d+(-[A-Za-z0-9.-]+)?$, got %q", v)
	}
	return nil
}

// validateCommandID checks a command id against the format implied by
// 02-manifest.md §command $defs and its use in the activation pattern:
// ^[a-z0-9._-]+$
func validateCommandID(id string) error {
	if id == "" {
		return fmt.Errorf("must not be empty")
	}
	if !reCommandID.MatchString(id) {
		return fmt.Errorf("must match ^[a-z0-9._-]+$, got %q", id)
	}
	return nil
}

// validateEngines checks that engines.opendray is present and non-empty.
// The value is a semver range (format validation is semver syntax, not range evaluation).
// 02-manifest.md lines 34–41.
func validateEngines(e *EnginesV1) error {
	if e == nil || e.Opendray == "" {
		return fmt.Errorf("engines.opendray is required on v1 manifests and must be non-empty")
	}
	return nil
}

// validateActivationEvent checks one activation event against the pattern
// from 02-manifest.md line 84 (items.pattern).
func validateActivationEvent(ev string) error {
	if !reActivation.MatchString(ev) {
		return fmt.Errorf("must match onStartup|onCommand:<id>|onView:<id>|onSession:(start|stop|idle|output)|onLanguage:<lang>|onFile:<glob>|onSchedule:cron:<expr>, got %q", ev)
	}
	return nil
}

// validateContributes checks every contribution point.
// Paths use dot+bracket notation matching 02-manifest.md field names.
func validateContributes(c *ContributesV1) []ValidationError {
	var errs []ValidationError

	// commands
	for i, cmd := range c.Commands {
		base := fmt.Sprintf("contributes.commands[%d]", i)
		if cmd.ID == "" || cmd.Title == "" {
			// Report on the whole entry if either required field is missing.
			var missing []string
			if cmd.ID == "" {
				missing = append(missing, "id")
			}
			if cmd.Title == "" {
				missing = append(missing, "title")
			}
			errs = append(errs, ValidationError{
				Path: base,
				Msg:  fmt.Sprintf("required fields missing: %v", missing),
			})
		} else {
			// id pattern check (only when id is non-empty)
			if err := validateCommandID(cmd.ID); err != nil {
				errs = append(errs, ValidationError{Path: base + ".id", Msg: err.Error()})
			}
		}
		// run.kind — if run is set, validate kind
		if cmd.Run != nil && cmd.Run.Kind != "" {
			switch cmd.Run.Kind {
			case "host", "notify", "openView", "runTask", "exec", "openUrl":
				// valid — 02-manifest.md line 161 (run.$defs enum)
			default:
				errs = append(errs, ValidationError{
					Path: base + ".run.kind",
					Msg:  fmt.Sprintf("must be one of host|notify|openView|runTask|exec|openUrl, got %q", cmd.Run.Kind),
				})
			}
		}
	}

	// statusBar
	for i, sb := range c.StatusBar {
		base := fmt.Sprintf("contributes.statusBar[%d]", i)
		if sb.ID == "" || sb.Text == "" {
			var missing []string
			if sb.ID == "" {
				missing = append(missing, "id")
			}
			if sb.Text == "" {
				missing = append(missing, "text")
			}
			errs = append(errs, ValidationError{
				Path: base,
				Msg:  fmt.Sprintf("required fields missing: %v", missing),
			})
		}
		// alignment — if set must be "left" | "right" (02-manifest.md line 187)
		if sb.Alignment != "" && sb.Alignment != "left" && sb.Alignment != "right" {
			errs = append(errs, ValidationError{
				Path: base + ".alignment",
				Msg:  fmt.Sprintf("must be left|right, got %q", sb.Alignment),
			})
		}
	}

	// keybindings
	for i, kb := range c.Keybindings {
		base := fmt.Sprintf("contributes.keybindings[%d]", i)
		// command + key both required (02-manifest.md line 203: required:["command","key"])
		if kb.Command == "" || kb.Key == "" {
			var missing []string
			if kb.Command == "" {
				missing = append(missing, "command")
			}
			if kb.Key == "" {
				missing = append(missing, "key")
			}
			errs = append(errs, ValidationError{
				Path: base,
				Msg:  fmt.Sprintf("required fields missing: %v", missing),
			})
		}
	}

	// ── M2: activityBar ──────────────────────────────────────────────────────
	// Limit: max 4 items (03-contribution-points.md §1).
	if len(c.ActivityBar) > 4 {
		errs = append(errs, ValidationError{
			Path: "contributes.activityBar",
			Msg:  fmt.Sprintf("too many (max 4), got %d", len(c.ActivityBar)),
		})
	}

	// Build a set of declared view IDs for cross-reference checks.
	viewIDs := make(map[string]struct{}, len(c.Views))
	for _, v := range c.Views {
		if v.ID != "" {
			viewIDs[v.ID] = struct{}{}
		}
	}

	for i, ab := range c.ActivityBar {
		base := fmt.Sprintf("contributes.activityBar[%d]", i)

		// id: required + regex ^[a-z0-9._-]+$
		if ab.ID == "" {
			errs = append(errs, ValidationError{Path: base + ".id", Msg: "must not be empty"})
		} else if err := validateCommandID(ab.ID); err != nil {
			errs = append(errs, ValidationError{Path: base + ".id", Msg: err.Error()})
		}

		// icon: required, non-empty
		if ab.Icon == "" {
			errs = append(errs, ValidationError{Path: base + ".icon", Msg: "must not be empty"})
		}

		// title: required, 1–48 chars
		if ab.Title == "" {
			errs = append(errs, ValidationError{Path: base + ".title", Msg: "must not be empty"})
		} else if len([]rune(ab.Title)) > 48 {
			errs = append(errs, ValidationError{
				Path: base + ".title",
				Msg:  fmt.Sprintf("must be 1–48 chars, got %d", len([]rune(ab.Title))),
			})
		}

		// viewId: optional; if set must reference a declared view id
		if ab.ViewID != "" {
			if _, ok := viewIDs[ab.ViewID]; !ok {
				errs = append(errs, ValidationError{
					Path: base + ".viewId",
					Msg:  fmt.Sprintf("references unknown view %q", ab.ViewID),
				})
			}
		}
	}

	// ── M2: views ────────────────────────────────────────────────────────────
	// Limit: max 8 items (03-contribution-points.md §2).
	if len(c.Views) > 8 {
		errs = append(errs, ValidationError{
			Path: "contributes.views",
			Msg:  fmt.Sprintf("too many (max 8), got %d", len(c.Views)),
		})
	}

	for i, v := range c.Views {
		base := fmt.Sprintf("contributes.views[%d]", i)

		// id: required + regex
		if v.ID == "" {
			errs = append(errs, ValidationError{Path: base + ".id", Msg: "must not be empty"})
		} else if err := validateCommandID(v.ID); err != nil {
			errs = append(errs, ValidationError{Path: base + ".id", Msg: err.Error()})
		}

		// title: required, 1–64 chars
		if v.Title == "" {
			errs = append(errs, ValidationError{Path: base + ".title", Msg: "must not be empty"})
		} else if len([]rune(v.Title)) > 64 {
			errs = append(errs, ValidationError{
				Path: base + ".title",
				Msg:  fmt.Sprintf("must be 1–64 chars, got %d", len([]rune(v.Title))),
			})
		}

		// container: optional; if set must be "activityBar" | "panel" | "sidebar"
		if v.Container != "" {
			switch v.Container {
			case "activityBar", "panel", "sidebar":
				// valid — 03-contribution-points.md §2
			default:
				errs = append(errs, ValidationError{
					Path: base + ".container",
					Msg:  fmt.Sprintf("must be one of activityBar|panel|sidebar, got %q", v.Container),
				})
			}
		}

		// render: optional; if set must be "webview" | "declarative"
		if v.Render != "" {
			switch v.Render {
			case "webview", "declarative":
				// valid
			default:
				errs = append(errs, ValidationError{
					Path: base + ".render",
					Msg:  fmt.Sprintf("must be one of webview|declarative, got %q", v.Render),
				})
			}
		}

		// entry: required when render=webview; must be a relative path
		if v.Render == "webview" {
			if v.Entry == "" {
				errs = append(errs, ValidationError{
					Path: base + ".entry",
					Msg:  "required when render=webview",
				})
			} else if err := validateRelativeBundlePath(v.Entry); err != nil {
				errs = append(errs, ValidationError{Path: base + ".entry", Msg: err.Error()})
			}
		}
	}

	// ── M2: panels ───────────────────────────────────────────────────────────
	// Limit: max 4 items (03-contribution-points.md §3).
	if len(c.Panels) > 4 {
		errs = append(errs, ValidationError{
			Path: "contributes.panels",
			Msg:  fmt.Sprintf("too many (max 4), got %d", len(c.Panels)),
		})
	}

	for i, panel := range c.Panels {
		base := fmt.Sprintf("contributes.panels[%d]", i)

		// id: required + regex
		if panel.ID == "" {
			errs = append(errs, ValidationError{Path: base + ".id", Msg: "must not be empty"})
		} else if err := validateCommandID(panel.ID); err != nil {
			errs = append(errs, ValidationError{Path: base + ".id", Msg: err.Error()})
		}

		// title: required, 1–64 chars
		if panel.Title == "" {
			errs = append(errs, ValidationError{Path: base + ".title", Msg: "must not be empty"})
		} else if len([]rune(panel.Title)) > 64 {
			errs = append(errs, ValidationError{
				Path: base + ".title",
				Msg:  fmt.Sprintf("must be 1–64 chars, got %d", len([]rune(panel.Title))),
			})
		}

		// position: optional; if set must be "bottom" | "right"
		if panel.Position != "" {
			switch panel.Position {
			case "bottom", "right":
				// valid — 03-contribution-points.md §3
			default:
				errs = append(errs, ValidationError{
					Path: base + ".position",
					Msg:  fmt.Sprintf("must be one of bottom|right, got %q", panel.Position),
				})
			}
		}

		// render: optional; if set must be "webview" | "declarative"
		if panel.Render != "" {
			switch panel.Render {
			case "webview", "declarative":
				// valid
			default:
				errs = append(errs, ValidationError{
					Path: base + ".render",
					Msg:  fmt.Sprintf("must be one of webview|declarative, got %q", panel.Render),
				})
			}
		}

		// entry: required when render=webview; must be a relative path
		if panel.Render == "webview" {
			if panel.Entry == "" {
				errs = append(errs, ValidationError{
					Path: base + ".entry",
					Msg:  "required when render=webview",
				})
			} else if err := validateRelativeBundlePath(panel.Entry); err != nil {
				errs = append(errs, ValidationError{Path: base + ".entry", Msg: err.Error()})
			}
		}
	}

	return errs
}

// validateRelativeBundlePath enforces that p is a legitimate reference into a
// plugin's ui/ directory:
//   - non-empty
//   - does not start with '/' (must be relative)
//   - does not contain '..' (defence-in-depth; the asset server also cleans
//     paths at serve time)
//   - does not contain newlines or other ASCII control characters
func validateRelativeBundlePath(p string) error {
	if p == "" {
		return fmt.Errorf("must not be empty")
	}
	if strings.HasPrefix(p, "/") {
		return fmt.Errorf("must be relative (must not start with '/')")
	}
	if strings.Contains(p, "..") {
		return fmt.Errorf("must not contain '..'")
	}
	for _, r := range p {
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("must not contain control characters")
		}
	}
	return nil
}

// validatePermissions checks the permission values against their allowed enums.
// 02-manifest.md lines 141–149 (permissions.$defs properties).
func validatePermissions(p *PermissionsV1) []ValidationError {
	var errs []ValidationError

	// session: enum [false, "read", "write"] — 02-manifest.md line 141
	// In Go the PermissionsV1.Session is a string; JSON false becomes "" (omitempty).
	// Only non-empty values need enum-checking.
	if p.Session != "" {
		switch p.Session {
		case "read", "write":
			// valid
		default:
			errs = append(errs, ValidationError{
				Path: "permissions.session",
				Msg:  fmt.Sprintf("must be read|write (or omitted for false), got %q", p.Session),
			})
		}
	}

	// clipboard: enum [false, "read", "write", "readwrite"] — 02-manifest.md line 144
	if p.Clipboard != "" {
		switch p.Clipboard {
		case "read", "write", "readwrite":
			// valid
		default:
			errs = append(errs, ValidationError{
				Path: "permissions.clipboard",
				Msg:  fmt.Sprintf("must be read|write|readwrite (or omitted for false), got %q", p.Clipboard),
			})
		}
	}

	// git: enum [false, "read", "write"] — 02-manifest.md line 146
	if p.Git != "" {
		switch p.Git {
		case "read", "write":
			// valid
		default:
			errs = append(errs, ValidationError{
				Path: "permissions.git",
				Msg:  fmt.Sprintf("must be read|write (or omitted for false), got %q", p.Git),
			})
		}
	}

	return errs
}

// ─── host.* ───────────────────────────────────────────────────────────────────

var (
	hostPlatformKeyRE = regexp.MustCompile(`^(linux|darwin|windows)-(x64|arm64)$`)
	hostEnvKeyRE      = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)
)

// validateHostV1 checks a form:"host" manifest's .host block.
//
// iOS builds refuse host-form plugins outright — a nil-safe check keeps
// the error message actionable when the host field is set but the OS
// doesn't allow it. Everywhere else, this validates the field set per
// 02-manifest.md §host.
func validateHostV1(h *HostV1) []ValidationError {
	if !HostFormAllowed {
		return []ValidationError{{
			Path: "host",
			Msg:  "host-form plugins are not supported on this platform",
		}}
	}
	if h == nil {
		return []ValidationError{{
			Path: "host",
			Msg:  `required when form:"host"`,
		}}
	}

	var errs []ValidationError

	if h.Entry == "" {
		errs = append(errs, ValidationError{Path: "host.entry", Msg: "required"})
	} else if strings.Contains(h.Entry, "..") {
		errs = append(errs, ValidationError{Path: "host.entry", Msg: `must not contain ".."`})
	}

	switch h.Runtime {
	case "", HostRuntimeBinary, HostRuntimeNode, HostRuntimeDeno,
		HostRuntimePython3, HostRuntimeBun, HostRuntimeCustom:
		// valid (empty → defaults to binary)
	default:
		errs = append(errs, ValidationError{
			Path: "host.runtime",
			Msg: fmt.Sprintf("must be one of binary|node|deno|python3|bun|custom, got %q",
				h.Runtime),
		})
	}

	if h.Protocol != "" && h.Protocol != HostProtocolJSONRPCStdio {
		errs = append(errs, ValidationError{
			Path: "host.protocol",
			Msg:  fmt.Sprintf("must be %q, got %q", HostProtocolJSONRPCStdio, h.Protocol),
		})
	}

	switch h.Restart {
	case "", HostRestartOnFailure, HostRestartAlways, HostRestartNever:
		// valid
	default:
		errs = append(errs, ValidationError{
			Path: "host.restart",
			Msg:  fmt.Sprintf("must be on-failure|always|never, got %q", h.Restart),
		})
	}

	for k := range h.Platforms {
		if !hostPlatformKeyRE.MatchString(k) {
			errs = append(errs, ValidationError{
				Path: fmt.Sprintf("host.platforms[%q]", k),
				Msg:  `key must match ^(linux|darwin|windows)-(x64|arm64)$`,
			})
		}
	}
	for k := range h.Env {
		if !hostEnvKeyRE.MatchString(k) {
			errs = append(errs, ValidationError{
				Path: fmt.Sprintf("host.env[%q]", k),
				Msg:  `key must match ^[A-Z_][A-Z0-9_]*$`,
			})
		}
	}

	if strings.Contains(h.Cwd, "..") {
		errs = append(errs, ValidationError{Path: "host.cwd", Msg: `must not contain ".."`})
	}
	if h.IdleShutdownMinutes < 0 {
		errs = append(errs, ValidationError{
			Path: "host.idleShutdownMinutes",
			Msg:  "must be ≥ 0 (0 = use supervisor default)",
		})
	}

	return errs
}

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
// JSON source: "^(onStartup|onCommand:[a-z0-9._-]+|onView:[a-z0-9._-]+|onSession:(start|stop|idle|output)|onLanguage:[a-z0-9_-]+|onFile:[^\\s]+|onSchedule:(cron:[^\\s]+))$"
var reActivation = regexp.MustCompile(`^(onStartup|onCommand:[a-z0-9._-]+|onView:[a-z0-9._-]+|onSession:(start|stop|idle|output)|onLanguage:[a-z0-9_-]+|onFile:[^\s]+|onSchedule:(cron:[^\s]+))$`)

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

	return errs
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

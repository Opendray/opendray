package plugin

import (
	"encoding/json"
	"strings"
	"testing"

	bundled "github.com/opendray/opendray/plugins"
)

// ─── Helper ─────────────────────────────────────────────────────────────────

// mustParseProvider unmarshals a JSON string into a Provider or fatals.
func mustParseProvider(t *testing.T, raw string) Provider {
	t.Helper()
	var p Provider
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	return p
}

// hasError returns true if any ValidationError in errs has a Path containing
// pathSubstr.
func hasError(errs []ValidationError, pathSubstr string) bool {
	for _, e := range errs {
		if strings.Contains(e.Path, pathSubstr) {
			return true
		}
	}
	return false
}

// ─── T1: Legacy short-circuit ────────────────────────────────────────────────

// TestValidateV1_LegacyShortCircuit verifies that ValidateV1 returns nil for
// every bundled legacy manifest (agents + panels). These manifests have
// IsV1()==false so the validator must short-circuit immediately — even if the
// legacy manifest would fail v1 rules (e.g. missing publisher).
func TestValidateV1_LegacyShortCircuit(t *testing.T) {
	var providers []Provider
	for _, root := range []string{"agents", "panels"} {
		ps, err := ScanFS(bundled.FS, root)
		if err != nil {
			t.Fatalf("ScanFS %s: %v", root, err)
		}
		providers = append(providers, ps...)
	}
	if len(providers) == 0 {
		t.Fatal("no bundled plugins found — ScanFS regression")
	}

	for _, p := range providers {
		p := p
		t.Run(p.Name, func(t *testing.T) {
			errs := ValidateV1(p)
			if errs != nil {
				t.Errorf("legacy manifest %q: ValidateV1 returned non-nil %v (want nil)", p.Name, errs)
			}
		})
	}
}

// ─── T2: Minimal valid v1 ────────────────────────────────────────────────────

// TestValidateV1_MinimalValid verifies that a hand-crafted minimal v1 manifest
// (name/version/publisher/engines) passes validation with an empty slice.
func TestValidateV1_MinimalValid(t *testing.T) {
	t.Run("minimal fields only", func(t *testing.T) {
		p := mustParseProvider(t, `{
			"name": "my-plugin",
			"version": "1.0.0",
			"publisher": "acme",
			"engines": { "opendray": "^1.0.0" }
		}`)
		errs := ValidateV1(p)
		if len(errs) != 0 {
			t.Errorf("expected no errors, got %v", errs)
		}
	})

	t.Run("single-char name and publisher", func(t *testing.T) {
		p := mustParseProvider(t, `{
			"name": "a",
			"version": "0.0.1",
			"publisher": "b",
			"engines": { "opendray": ">=0.1.0" }
		}`)
		errs := ValidateV1(p)
		if len(errs) != 0 {
			t.Errorf("expected no errors for single-char name/publisher, got %v", errs)
		}
	})
}

// ─── T3: time-ninja full fixture ─────────────────────────────────────────────

// TestValidateV1_TimeNinja verifies that the reference time-ninja manifest
// (with every contribution point + permissions) passes validation cleanly.
func TestValidateV1_TimeNinja(t *testing.T) {
	const raw = `{
		"$schema": "https://opendray.dev/schemas/plugin-manifest-v1.json",
		"name": "time-ninja",
		"version": "1.0.0",
		"publisher": "opendray-examples",
		"displayName": "Time Ninja",
		"description": "Pomodoro reminder that lives in the status bar.",
		"icon": "🍅",
		"engines": { "opendray": "^1.0.0" },
		"form": "declarative",
		"activation": ["onStartup"],
		"contributes": {
			"commands": [
				{
					"id": "time.start",
					"title": "Start Pomodoro",
					"category": "Time Ninja",
					"run": { "kind": "notify", "message": "Pomodoro started — 25 minutes" }
				}
			],
			"statusBar": [
				{
					"id": "time.bar",
					"text": "🍅 25:00",
					"tooltip": "Start a pomodoro",
					"command": "time.start",
					"alignment": "right",
					"priority": 50
				}
			],
			"keybindings": [
				{ "command": "time.start", "key": "ctrl+alt+p", "mac": "cmd+alt+p" }
			],
			"menus": {
				"appBar/right": [
					{ "command": "time.start", "group": "timer@1" }
				]
			}
		},
		"permissions": {}
	}`

	p := mustParseProvider(t, raw)
	errs := ValidateV1(p)
	if len(errs) != 0 {
		t.Errorf("time-ninja fixture: expected no errors, got:\n%v", errs)
	}
}

// ─── T4: Invalid cases ───────────────────────────────────────────────────────

// TestValidateV1_InvalidCases is the comprehensive table of bad manifests.
// Each case specifies one or more error path substrings that must be present.
func TestValidateV1_InvalidCases(t *testing.T) {
	type tc struct {
		name          string
		json          string
		wantPathParts []string // all must appear as error paths
	}

	cases := []tc{
		// ── name ──────────────────────────────────────────────────────────────
		{
			name: "name empty",
			json: `{"name":"","version":"1.0.0","publisher":"acme","engines":{"opendray":"^1"}}`,
			wantPathParts: []string{"name"},
		},
		{
			name: "name uppercase",
			json: `{"name":"MyPlugin","version":"1.0.0","publisher":"acme","engines":{"opendray":"^1"}}`,
			wantPathParts: []string{"name"},
		},
		{
			name: "name starts with hyphen",
			json: `{"name":"-bad","version":"1.0.0","publisher":"acme","engines":{"opendray":"^1"}}`,
			wantPathParts: []string{"name"},
		},
		{
			name: "name too long (65 chars)",
			// 65 lowercase letters: max allowed is 64 (1 start + up to 62 middle + 1 end).
			// The string below is exactly 65 'a' characters.
			json: `{"name":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","version":"1.0.0","publisher":"acme","engines":{"opendray":"^1"}}`,
			wantPathParts: []string{"name"},
		},
		{
			name: "name ends with hyphen",
			json: `{"name":"bad-","version":"1.0.0","publisher":"acme","engines":{"opendray":"^1"}}`,
			wantPathParts: []string{"name"},
		},

		// ── publisher ─────────────────────────────────────────────────────────
		{
			name: "publisher empty",
			// publisher="" means IsV1() won't return true, so we must set engines
			// and name but leave publisher blank — however IsV1 requires publisher != ""
			// The way to test publisher validation is to have publisher set to an invalid
			// value. Here we test that publisher pattern rejects uppercase.
			json: `{"name":"ok","version":"1.0.0","publisher":"Acme","engines":{"opendray":"^1"}}`,
			wantPathParts: []string{"publisher"},
		},
		{
			name: "publisher starts with hyphen",
			json: `{"name":"ok","version":"1.0.0","publisher":"-bad","engines":{"opendray":"^1"}}`,
			wantPathParts: []string{"publisher"},
		},
		{
			name: "publisher too long (41 chars)",
			// max publisher length: 1 + 38 middle + 1 = 40 chars
			json: `{"name":"ok","version":"1.0.0","publisher":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","engines":{"opendray":"^1"}}`,
			wantPathParts: []string{"publisher"},
		},
		{
			name: "publisher ends with hyphen",
			json: `{"name":"ok","version":"1.0.0","publisher":"bad-","engines":{"opendray":"^1"}}`,
			wantPathParts: []string{"publisher"},
		},

		// ── version / semver ──────────────────────────────────────────────────
		{
			name: "version missing patch",
			json: `{"name":"ok","version":"1.0","publisher":"acme","engines":{"opendray":"^1"}}`,
			wantPathParts: []string{"version"},
		},
		{
			name: "version with v prefix",
			json: `{"name":"ok","version":"v1.0.0","publisher":"acme","engines":{"opendray":"^1"}}`,
			wantPathParts: []string{"version"},
		},
		{
			name: "version four parts",
			json: `{"name":"ok","version":"1.0.0.0","publisher":"acme","engines":{"opendray":"^1"}}`,
			wantPathParts: []string{"version"},
		},

		// ── engines ───────────────────────────────────────────────────────────
		// NOTE (spec drift): engines.opendray being empty causes IsV1()==false, so
		// ValidateV1 short-circuits before the engines check is reached. The engines
		// check inside ValidateV1 is therefore a defence-in-depth guard for future
		// callers that bypass IsV1(). To test the engines.opendray rule we must use
		// a manifest that IS v1 (publisher+engines both present) but has engines
		// with a whitespace-only value — however the current IsV1() requires
		// engines.Opendray != "", so we cannot construct such a case via the public
		// API without internal mutation. We instead test validateEngines() directly.
		// For the invalid-cases table we substitute a test that IS reachable:
		// a name that is valid but version is bad (additional engines coverage via
		// TestValidateEngines below).
		//
		// Engines guard: explicitly test that a manifest with publisher + empty
		// engines struct is classified as legacy (IsV1==false) by using nil engines.
		// We cover the validateEngines path via TestValidateEngines separately.
		// A v1 manifest with pre-release version containing an invalid char (space)
		// exercises the semver validator on the pre-release portion.
		{
			name: "version semver pre-release with space",
			json: `{"name":"ok","version":"1.0.0-alpha 1","publisher":"acme","engines":{"opendray":"^1"}}`,
			wantPathParts: []string{"version"},
		},

		// ── form ──────────────────────────────────────────────────────────────
		{
			name: "form invalid value native",
			json: `{"name":"ok","version":"1.0.0","publisher":"acme","engines":{"opendray":"^1"},"form":"native"}`,
			wantPathParts: []string{"form"},
		},
		{
			name: "form invalid value empty string with explicit set",
			// Only rejected if form is present but not one of the three values.
			// An empty form field is fine (omitempty), so use an unknown value.
			json: `{"name":"ok","version":"1.0.0","publisher":"acme","engines":{"opendray":"^1"},"form":"plugin"}`,
			wantPathParts: []string{"form"},
		},

		// ── activation ────────────────────────────────────────────────────────
		{
			name: "activation event onFoo unknown",
			json: `{"name":"ok","version":"1.0.0","publisher":"acme","engines":{"opendray":"^1"},"activation":["onFoo"]}`,
			wantPathParts: []string{"activation[0]"},
		},
		{
			name: "activation onCommand without id",
			// "onCommand:" with no id after colon is invalid
			json: `{"name":"ok","version":"1.0.0","publisher":"acme","engines":{"opendray":"^1"},"activation":["onCommand:"]}`,
			wantPathParts: []string{"activation[0]"},
		},

		// ── contributes.commands ──────────────────────────────────────────────
		{
			name: "command missing title",
			json: `{"name":"ok","version":"1.0.0","publisher":"acme","engines":{"opendray":"^1"},"contributes":{"commands":[{"id":"foo.bar"}]}}`,
			wantPathParts: []string{"contributes.commands[0]"},
		},
		{
			name: "command missing id",
			json: `{"name":"ok","version":"1.0.0","publisher":"acme","engines":{"opendray":"^1"},"contributes":{"commands":[{"title":"Do thing"}]}}`,
			wantPathParts: []string{"contributes.commands[0]"},
		},
		{
			name: "command id uppercase",
			json: `{"name":"ok","version":"1.0.0","publisher":"acme","engines":{"opendray":"^1"},"contributes":{"commands":[{"id":"Foo.Bar","title":"Do thing"}]}}`,
			wantPathParts: []string{"contributes.commands[0].id"},
		},
		{
			name: "command run.kind invalid",
			json: `{"name":"ok","version":"1.0.0","publisher":"acme","engines":{"opendray":"^1"},"contributes":{"commands":[{"id":"foo.bar","title":"T","run":{"kind":"magic"}}]}}`,
			wantPathParts: []string{"contributes.commands[0].run.kind"},
		},

		// ── contributes.statusBar ─────────────────────────────────────────────
		{
			name: "statusBar item missing id",
			json: `{"name":"ok","version":"1.0.0","publisher":"acme","engines":{"opendray":"^1"},"contributes":{"statusBar":[{"text":"hi"}]}}`,
			wantPathParts: []string{"contributes.statusBar[0]"},
		},
		{
			name: "statusBar item missing text",
			json: `{"name":"ok","version":"1.0.0","publisher":"acme","engines":{"opendray":"^1"},"contributes":{"statusBar":[{"id":"s.bar"}]}}`,
			wantPathParts: []string{"contributes.statusBar[0]"},
		},
		{
			name: "statusBar alignment invalid",
			json: `{"name":"ok","version":"1.0.0","publisher":"acme","engines":{"opendray":"^1"},"contributes":{"statusBar":[{"id":"s.bar","text":"T","alignment":"center"}]}}`,
			wantPathParts: []string{"contributes.statusBar[0].alignment"},
		},

		// ── contributes.keybindings ───────────────────────────────────────────
		{
			name: "keybinding missing key",
			json: `{"name":"ok","version":"1.0.0","publisher":"acme","engines":{"opendray":"^1"},"contributes":{"keybindings":[{"command":"foo.bar"}]}}`,
			wantPathParts: []string{"contributes.keybindings[0]"},
		},

		// ── permissions ───────────────────────────────────────────────────────
		{
			name: "permissions.session admin not in enum",
			json: `{"name":"ok","version":"1.0.0","publisher":"acme","engines":{"opendray":"^1"},"permissions":{"session":"admin"}}`,
			wantPathParts: []string{"permissions.session"},
		},
		{
			name: "permissions.clipboard copy not in enum",
			json: `{"name":"ok","version":"1.0.0","publisher":"acme","engines":{"opendray":"^1"},"permissions":{"clipboard":"copy"}}`,
			wantPathParts: []string{"permissions.clipboard"},
		},
		{
			name: "permissions.git superuser not in enum",
			json: `{"name":"ok","version":"1.0.0","publisher":"acme","engines":{"opendray":"^1"},"permissions":{"git":"superuser"}}`,
			wantPathParts: []string{"permissions.git"},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			p := mustParseProvider(t, tc.json)
			errs := ValidateV1(p)
			if len(errs) == 0 {
				t.Errorf("expected validation errors, got none")
				return
			}
			for _, pathPart := range tc.wantPathParts {
				if !hasError(errs, pathPart) {
					t.Errorf("expected error with path containing %q, got errors: %v", pathPart, errs)
				}
			}
		})
	}
}

// ─── T5: Helper unit tests ────────────────────────────────────────────────────

// TestValidateName verifies regex boundary conditions for the name field.
// Pattern: ^[a-z0-9]([a-z0-9-]{0,62}[a-z0-9])?$
// (doc 02-manifest.md line 18)
func TestValidateName(t *testing.T) {
	type tc struct {
		name  string
		input string
		valid bool
	}
	cases := []tc{
		// valid
		{"single char a", "a", true},
		{"single char 0", "0", true},
		{"two chars", "ab", true},
		{"with hyphen middle", "my-plugin", true},
		{"all digits", "123", true},
		{"max length 64", strings.Repeat("a", 64), true},
		// invalid
		{"empty string", "", false},
		{"single hyphen", "-", false},
		{"starts with hyphen", "-plugin", false},
		{"ends with hyphen", "plugin-", false},
		{"uppercase", "MyPlugin", false},
		{"spaces", "my plugin", false},
		{"too long 65", strings.Repeat("a", 65), false},
		{"double hyphen (valid per regex)", "my--plugin", true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := validateName(tc.input)
			if tc.valid && err != nil {
				t.Errorf("validateName(%q) = %v, want nil", tc.input, err)
			}
			if !tc.valid && err == nil {
				t.Errorf("validateName(%q) = nil, want error", tc.input)
			}
		})
	}
}

// TestValidatePublisher verifies regex boundary conditions for publisher field.
// Pattern: ^[a-z0-9]([a-z0-9-]{0,38}[a-z0-9])?$
// (doc 02-manifest.md line 20)
func TestValidatePublisher(t *testing.T) {
	type tc struct {
		name  string
		input string
		valid bool
	}
	cases := []tc{
		// valid
		{"single char", "a", true},
		{"simple name", "acme", true},
		{"with hyphen", "my-org", true},
		{"max length 40", strings.Repeat("a", 40), true},
		// invalid
		{"empty", "", false},
		{"starts with hyphen", "-org", false},
		{"ends with hyphen", "org-", false},
		{"uppercase", "Acme", false},
		{"too long 41", strings.Repeat("a", 41), false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := validatePublisher(tc.input)
			if tc.valid && err != nil {
				t.Errorf("validatePublisher(%q) = %v, want nil", tc.input, err)
			}
			if !tc.valid && err == nil {
				t.Errorf("validatePublisher(%q) = nil, want error", tc.input)
			}
		})
	}
}

// TestValidateSemver verifies regex boundary conditions for semver field.
// Pattern: ^\d+\.\d+\.\d+(-[A-Za-z0-9.-]+)?$
// (doc 02-manifest.md line 19)
func TestValidateSemver(t *testing.T) {
	type tc struct {
		name  string
		input string
		valid bool
	}
	cases := []tc{
		// valid
		{"basic semver", "1.0.0", true},
		{"zero version", "0.0.0", true},
		{"large numbers", "10.20.300", true},
		{"pre-release alpha", "1.0.0-alpha", true},
		{"pre-release alpha.1", "1.0.0-alpha.1", true},
		{"pre-release with dots", "2.0.0-rc.1.final", true},
		// invalid
		{"empty", "", false},
		{"v prefix", "v1.0.0", false},
		{"missing patch", "1.0", false},
		{"four parts", "1.0.0.0", false},
		{"trailing dot", "1.0.", false},
		{"no numbers", "major.minor.patch", false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := validateSemver(tc.input)
			if tc.valid && err != nil {
				t.Errorf("validateSemver(%q) = %v, want nil", tc.input, err)
			}
			if !tc.valid && err == nil {
				t.Errorf("validateSemver(%q) = nil, want error", tc.input)
			}
		})
	}
}

// TestValidateCommandID verifies regex boundary conditions for command id field.
// Pattern: ^[a-z0-9._-]+$
// (doc 02-manifest.md §command definition)
func TestValidateCommandID(t *testing.T) {
	type tc struct {
		name  string
		input string
		valid bool
	}
	cases := []tc{
		// valid
		{"simple", "foo", true},
		{"dot separated", "time.start", true},
		{"with hyphen", "my-cmd", true},
		{"with underscore", "my_cmd", true},
		{"all chars", "a0._-", true},
		{"single char", "a", true},
		// invalid
		{"empty", "", false},
		{"uppercase", "Foo", false},
		{"space", "foo bar", false},
		{"slash", "foo/bar", false},
		{"colon", "foo:bar", false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := validateCommandID(tc.input)
			if tc.valid && err != nil {
				t.Errorf("validateCommandID(%q) = %v, want nil", tc.input, err)
			}
			if !tc.valid && err == nil {
				t.Errorf("validateCommandID(%q) = nil, want error", tc.input)
			}
		})
	}
}

// TestValidationError_Error verifies the Error() string format.
func TestValidationError_Error(t *testing.T) {
	t.Run("format is path colon space msg", func(t *testing.T) {
		e := ValidationError{Path: "contributes.commands[0].id", Msg: "invalid format"}
		want := "contributes.commands[0].id: invalid format"
		if got := e.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})
}

// TestValidateV1_ValidActivationEvents verifies that all documented activation
// event patterns are accepted.
func TestValidateV1_ValidActivationEvents(t *testing.T) {
	base := `{"name":"ok","version":"1.0.0","publisher":"acme","engines":{"opendray":"^1"},"activation":[%s]}`

	valid := []string{
		`"onStartup"`,
		`"onCommand:my.cmd"`,
		`"onCommand:time.start"`,
		`"onView:my.view"`,
		`"onSession:start"`,
		`"onSession:stop"`,
		`"onSession:idle"`,
		`"onSession:output"`,
		`"onLanguage:rust"`,
		`"onLanguage:go"`,
		`"onFile:*.rs"`,
		`"onFile:src/main.go"`,
		// Standard five-field cron with spaces — 02-manifest.md L84 was
		// widened to `cron:.+` after T2 surfaced the drift; real plugin
		// schedules need this shape.
		`"onSchedule:cron:0 * * * *"`,
		`"onSchedule:cron:@hourly"`,
	}

	for _, ev := range valid {
		ev := ev
		t.Run(ev, func(t *testing.T) {
			p := mustParseProvider(t, strings.ReplaceAll(base, "%s", ev))
			errs := ValidateV1(p)
			if len(errs) != 0 {
				t.Errorf("expected no errors for activation event %s, got %v", ev, errs)
			}
		})
	}
}

// TestValidateV1_InvalidActivationEvents verifies that bad activation event
// strings produce errors at the right path.
func TestValidateV1_InvalidActivationEvents(t *testing.T) {
	base := `{"name":"ok","version":"1.0.0","publisher":"acme","engines":{"opendray":"^1"},"activation":[%s]}`

	invalid := []string{
		`"onFoo"`,
		`"onCommand:"`,
		`"onSession:crash"`,
		`"onSession:"`,
		`"startup"`,
		`"ON_STARTUP"`,
	}

	for _, ev := range invalid {
		ev := ev
		t.Run(ev, func(t *testing.T) {
			p := mustParseProvider(t, strings.ReplaceAll(base, "%s", ev))
			errs := ValidateV1(p)
			if !hasError(errs, "activation[0]") {
				t.Errorf("expected activation[0] error for %s, got %v", ev, errs)
			}
		})
	}
}

// TestValidateV1_NilContributesAndPermissions covers the nil-check branches
// inside ValidateV1 where contributes and permissions are not set.
func TestValidateV1_NilContributesAndPermissions(t *testing.T) {
	t.Run("no contributes no permissions", func(t *testing.T) {
		p := mustParseProvider(t, `{
			"name": "bare",
			"version": "1.0.0",
			"publisher": "acme",
			"engines": { "opendray": "^1.0.0" }
		}`)
		errs := ValidateV1(p)
		if len(errs) != 0 {
			t.Errorf("expected no errors, got %v", errs)
		}
	})
}

// TestValidateContributes_KeybindingMissingCommand covers the keybinding
// path where command is missing (not just key).
func TestValidateContributes_KeybindingMissingCommand(t *testing.T) {
	t.Run("keybinding missing command and key", func(t *testing.T) {
		c := &ContributesV1{
			Keybindings: []KeybindingV1{
				{Command: "", Key: ""},
			},
		}
		errs := validateContributes(c)
		if len(errs) == 0 {
			t.Error("expected error for keybinding missing command and key, got none")
		}
		found := false
		for _, e := range errs {
			if strings.Contains(e.Path, "keybindings[0]") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected keybindings[0] path in errors, got %v", errs)
		}
	})
}

// TestValidateEngines verifies the validateEngines helper directly, covering
// the nil engine and empty-opendray branches that cannot be reached via ValidateV1
// (because IsV1() gates on both conditions). This provides coverage for the
// defence-in-depth code path in validateEngines.
func TestValidateEngines(t *testing.T) {
	t.Run("nil engines returns error", func(t *testing.T) {
		if err := validateEngines(nil); err == nil {
			t.Error("validateEngines(nil) = nil, want error")
		}
	})
	t.Run("empty opendray returns error", func(t *testing.T) {
		if err := validateEngines(&EnginesV1{Opendray: ""}); err == nil {
			t.Error("validateEngines(&EnginesV1{}) = nil, want error")
		}
	})
	t.Run("valid opendray range passes", func(t *testing.T) {
		if err := validateEngines(&EnginesV1{Opendray: "^1.0.0"}); err != nil {
			t.Errorf("validateEngines(^1.0.0) = %v, want nil", err)
		}
	})
}

// TestValidateV1_ValidCommandRunKinds checks that all allowed run.kind values pass.
func TestValidateV1_ValidCommandRunKinds(t *testing.T) {
	kinds := []string{"host", "notify", "openView", "runTask", "exec", "openUrl"}
	for _, kind := range kinds {
		kind := kind
		t.Run(kind, func(t *testing.T) {
			raw := `{"name":"ok","version":"1.0.0","publisher":"acme","engines":{"opendray":"^1"},"contributes":{"commands":[{"id":"foo.bar","title":"T","run":{"kind":"` + kind + `"}}]}}`
			p := mustParseProvider(t, raw)
			errs := ValidateV1(p)
			if hasError(errs, "run.kind") {
				t.Errorf("kind=%q: unexpected run.kind error in %v", kind, errs)
			}
		})
	}
}

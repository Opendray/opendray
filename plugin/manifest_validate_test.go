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

// ─── M2 T2: webview contribution-point validator ────────────────────────────

// hasErrorMsg returns true if any ValidationError contains both pathSubstr in
// Path and msgSubstr in Msg.
func hasErrorMsg(errs []ValidationError, pathSubstr, msgSubstr string) bool {
	for _, e := range errs {
		if strings.Contains(e.Path, pathSubstr) && strings.Contains(e.Msg, msgSubstr) {
			return true
		}
	}
	return false
}

// baseV1 is a minimal valid v1 JSON preamble — append contributes to build fixtures.
const baseV1 = `{"name":"ok","version":"1.0.0","publisher":"acme","engines":{"opendray":"^1"}}`

// withContributes wraps a contributes JSON fragment into a full v1 manifest.
func withContributes(t *testing.T, contributes string) Provider {
	t.Helper()
	return mustParseProvider(t, `{"name":"ok","version":"1.0.0","publisher":"acme","engines":{"opendray":"^1"},"contributes":`+contributes+`}`)
}

// ── Count limits ──────────────────────────────────────────────────────────────

// TestValidate_Webview_ActivityBarCountLimit asserts that 5 activityBar entries
// (exceeding the max-4 limit from 03-contribution-points.md §1) yields an error
// path containing "contributes.activityBar" and message containing "4".
func TestValidate_Webview_ActivityBarCountLimit(t *testing.T) {
	p := withContributes(t, `{"activityBar":[
		{"id":"a.1","icon":"i","title":"T1"},
		{"id":"a.2","icon":"i","title":"T2"},
		{"id":"a.3","icon":"i","title":"T3"},
		{"id":"a.4","icon":"i","title":"T4"},
		{"id":"a.5","icon":"i","title":"T5"}
	]}`)
	errs := ValidateV1(p)
	if !hasErrorMsg(errs, "contributes.activityBar", "4") {
		t.Errorf("expected contributes.activityBar too-many error containing '4', got: %v", errs)
	}
}

// TestValidate_Webview_ViewsCountLimit asserts that 9 views (exceeding max-8
// from 03-contribution-points.md §2) yield an error with "max 8".
func TestValidate_Webview_ViewsCountLimit(t *testing.T) {
	p := withContributes(t, `{"views":[
		{"id":"v.1","title":"V1"},
		{"id":"v.2","title":"V2"},
		{"id":"v.3","title":"V3"},
		{"id":"v.4","title":"V4"},
		{"id":"v.5","title":"V5"},
		{"id":"v.6","title":"V6"},
		{"id":"v.7","title":"V7"},
		{"id":"v.8","title":"V8"},
		{"id":"v.9","title":"V9"}
	]}`)
	errs := ValidateV1(p)
	if !hasErrorMsg(errs, "contributes.views", "8") {
		t.Errorf("expected contributes.views too-many error containing '8', got: %v", errs)
	}
}

// TestValidate_Webview_PanelsCountLimit asserts that 5 panels (exceeding max-4
// from 03-contribution-points.md §3) yield an error with "max 4".
func TestValidate_Webview_PanelsCountLimit(t *testing.T) {
	p := withContributes(t, `{"panels":[
		{"id":"p.1","title":"P1"},
		{"id":"p.2","title":"P2"},
		{"id":"p.3","title":"P3"},
		{"id":"p.4","title":"P4"},
		{"id":"p.5","title":"P5"}
	]}`)
	errs := ValidateV1(p)
	if !hasErrorMsg(errs, "contributes.panels", "4") {
		t.Errorf("expected contributes.panels too-many error containing '4', got: %v", errs)
	}
}

// ── ID regex ─────────────────────────────────────────────────────────────────

// TestValidate_Webview_ActivityBarIDRegex asserts that an activityBar item
// with id "Bad ID" (uppercase + space) fails at contributes.activityBar[0].id.
func TestValidate_Webview_ActivityBarIDRegex(t *testing.T) {
	p := withContributes(t, `{"activityBar":[
		{"id":"Bad ID","icon":"i","title":"T"}
	]}`)
	errs := ValidateV1(p)
	if !hasError(errs, "contributes.activityBar[0].id") {
		t.Errorf("expected contributes.activityBar[0].id error, got: %v", errs)
	}
}

// ── Enum validation ──────────────────────────────────────────────────────────

// TestValidate_Webview_ViewsRenderEnum asserts that render="native" (not in
// {webview, declarative}) fails at contributes.views[0].render.
func TestValidate_Webview_ViewsRenderEnum(t *testing.T) {
	p := withContributes(t, `{"views":[
		{"id":"v.1","title":"T","render":"native"}
	]}`)
	errs := ValidateV1(p)
	if !hasError(errs, "contributes.views[0].render") {
		t.Errorf("expected contributes.views[0].render error, got: %v", errs)
	}
}

// TestValidate_Webview_PanelsPositionEnum asserts that position="top" (not in
// {bottom, right}) fails at contributes.panels[0].position.
func TestValidate_Webview_PanelsPositionEnum(t *testing.T) {
	p := withContributes(t, `{"panels":[
		{"id":"p.1","title":"T","position":"top"}
	]}`)
	errs := ValidateV1(p)
	if !hasError(errs, "contributes.panels[0].position") {
		t.Errorf("expected contributes.panels[0].position error, got: %v", errs)
	}
}

// ── Entry path validation ─────────────────────────────────────────────────────

// TestValidate_Webview_EntryRequiredWhenWebview asserts that a view with
// render=webview but missing entry field produces an error at
// contributes.views[0].entry containing "required".
func TestValidate_Webview_EntryRequiredWhenWebview(t *testing.T) {
	t.Run("view missing entry", func(t *testing.T) {
		p := withContributes(t, `{"views":[
			{"id":"v.1","title":"T","render":"webview"}
		]}`)
		errs := ValidateV1(p)
		if !hasErrorMsg(errs, "contributes.views[0].entry", "required") {
			t.Errorf("expected views[0].entry required error, got: %v", errs)
		}
	})
	t.Run("panel missing entry", func(t *testing.T) {
		p := withContributes(t, `{"panels":[
			{"id":"p.1","title":"T","render":"webview"}
		]}`)
		errs := ValidateV1(p)
		if !hasErrorMsg(errs, "contributes.panels[0].entry", "required") {
			t.Errorf("expected panels[0].entry required error, got: %v", errs)
		}
	})
}

// TestValidate_Webview_EntryMustBeRelative tests the validateRelativeBundlePath
// helper via views and panels:
//   - absolute path "/etc/passwd" → fails
//   - path traversal "../escape" → fails
//   - valid relative path "ui/index.html" → passes
func TestValidate_Webview_EntryMustBeRelative(t *testing.T) {
	t.Run("absolute path rejected", func(t *testing.T) {
		p := withContributes(t, `{"views":[
			{"id":"v.1","title":"T","render":"webview","entry":"/etc/passwd"}
		]}`)
		errs := ValidateV1(p)
		if !hasErrorMsg(errs, "contributes.views[0].entry", "relative") {
			t.Errorf("expected 'must be relative' error, got: %v", errs)
		}
	})
	t.Run("dotdot path rejected", func(t *testing.T) {
		p := withContributes(t, `{"views":[
			{"id":"v.1","title":"T","render":"webview","entry":"../escape"}
		]}`)
		errs := ValidateV1(p)
		if !hasErrorMsg(errs, "contributes.views[0].entry", "..") {
			t.Errorf("expected must-not-contain-'..' error, got: %v", errs)
		}
	})
	t.Run("valid relative path passes", func(t *testing.T) {
		p := withContributes(t, `{"views":[
			{"id":"v.1","title":"T","render":"webview","entry":"ui/index.html"}
		]}`)
		errs := ValidateV1(p)
		if hasError(errs, "contributes.views[0].entry") {
			t.Errorf("expected no entry error for valid relative path, got: %v", errs)
		}
	})
	t.Run("panel absolute path rejected", func(t *testing.T) {
		p := withContributes(t, `{"panels":[
			{"id":"p.1","title":"T","render":"webview","entry":"/etc/passwd"}
		]}`)
		errs := ValidateV1(p)
		if !hasErrorMsg(errs, "contributes.panels[0].entry", "relative") {
			t.Errorf("expected 'must be relative' error for panel, got: %v", errs)
		}
	})
	t.Run("panel dotdot path rejected", func(t *testing.T) {
		p := withContributes(t, `{"panels":[
			{"id":"p.1","title":"T","render":"webview","entry":"../../etc/shadow"}
		]}`)
		errs := ValidateV1(p)
		if !hasErrorMsg(errs, "contributes.panels[0].entry", "..") {
			t.Errorf("expected must-not-contain-'..' error for panel, got: %v", errs)
		}
	})
}

// ── Cross-reference check ─────────────────────────────────────────────────────

// TestValidate_Webview_ActivityBarOrphanViewId asserts that an activityBar item
// referencing a viewId that does not appear in contributes.views fails with a
// message containing "unknown view".
func TestValidate_Webview_ActivityBarOrphanViewId(t *testing.T) {
	t.Run("orphan viewId with empty views list", func(t *testing.T) {
		p := withContributes(t, `{"activityBar":[
			{"id":"a.1","icon":"i","title":"T","viewId":"nope"}
		]}`)
		errs := ValidateV1(p)
		if !hasErrorMsg(errs, "contributes.activityBar[0].viewId", "nope") {
			t.Errorf("expected orphan-viewId error mentioning 'nope', got: %v", errs)
		}
	})
	t.Run("orphan viewId with non-matching views", func(t *testing.T) {
		p := withContributes(t, `{
			"activityBar":[{"id":"a.1","icon":"i","title":"T","viewId":"wrong"}],
			"views":[{"id":"right.view","title":"R"}]
		}`)
		errs := ValidateV1(p)
		if !hasErrorMsg(errs, "contributes.activityBar[0].viewId", "wrong") {
			t.Errorf("expected orphan-viewId error mentioning 'wrong', got: %v", errs)
		}
	})
	t.Run("valid viewId reference passes", func(t *testing.T) {
		p := withContributes(t, `{
			"activityBar":[{"id":"a.1","icon":"i","title":"T","viewId":"my.view"}],
			"views":[{"id":"my.view","title":"My View"}]
		}`)
		errs := ValidateV1(p)
		if hasError(errs, "contributes.activityBar[0].viewId") {
			t.Errorf("expected no viewId error for valid cross-reference, got: %v", errs)
		}
	})
}

// ── Full-manifest happy-path ──────────────────────────────────────────────────

// TestValidate_Webview_KanbanPasses verifies that the reference kanban manifest
// from M2-PLAN.md §10 passes ValidateV1 with zero errors.
func TestValidate_Webview_KanbanPasses(t *testing.T) {
	const kanbanManifest = `{
		"$schema": "https://opendray.dev/schemas/plugin-manifest-v1.json",
		"name": "kanban",
		"version": "1.0.0",
		"publisher": "opendray-examples",
		"displayName": "Kanban",
		"description": "A minimal kanban board.",
		"icon": "📋",
		"engines": { "opendray": "^1.0.0" },
		"form": "webview",
		"activation": ["onView:kanban.board"],
		"contributes": {
			"activityBar": [
				{ "id": "kanban.activity", "icon": "📋", "title": "Kanban", "viewId": "kanban.board" }
			],
			"views": [
				{ "id": "kanban.board", "title": "Kanban Board",
				  "container": "activityBar", "render": "webview", "entry": "index.html" }
			]
		},
		"permissions": {
			"storage": true
		}
	}`
	p := mustParseProvider(t, kanbanManifest)
	errs := ValidateV1(p)
	if len(errs) != 0 {
		t.Errorf("kanban manifest: expected no errors, got:\n%v", errs)
	}
}

// ── validateRelativeBundlePath unit tests ─────────────────────────────────────

// TestValidateRelativeBundlePath exercises the helper directly covering all
// edge cases: empty, absolute, dotdot, newline, and valid paths.
func TestValidateRelativeBundlePath(t *testing.T) {
	type tc struct {
		name  string
		input string
		valid bool
	}
	cases := []tc{
		// valid
		{"simple filename", "index.html", true},
		{"subdir path", "ui/index.html", true},
		{"deep path", "dist/assets/main.js", true},
		// invalid
		{"empty string", "", false},
		{"absolute path", "/etc/passwd", false},
		{"dotdot traversal", "../escape", false},
		{"dotdot in middle", "ui/../../../etc/passwd", false},
		{"newline injection", "ui/index.html\nX-Injected: bad", false},
		{"carriage return injection", "ui/index.html\rother", false},
		{"null byte", "ui/index\x00.html", false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := validateRelativeBundlePath(tc.input)
			if tc.valid && err != nil {
				t.Errorf("validateRelativeBundlePath(%q) = %v, want nil", tc.input, err)
			}
			if !tc.valid && err == nil {
				t.Errorf("validateRelativeBundlePath(%q) = nil, want error", tc.input)
			}
		})
	}
}

// ── Additional field-level validation ────────────────────────────────────────

// TestValidate_Webview_ActivityBarFieldRules tests per-field rules for
// activityBar items: icon required, title length bounds.
func TestValidate_Webview_ActivityBarFieldRules(t *testing.T) {
	t.Run("icon empty fails", func(t *testing.T) {
		p := withContributes(t, `{"activityBar":[{"id":"a.1","icon":"","title":"T"}]}`)
		errs := ValidateV1(p)
		if !hasError(errs, "contributes.activityBar[0].icon") {
			t.Errorf("expected icon required error, got: %v", errs)
		}
	})
	t.Run("title too long (49 chars) fails", func(t *testing.T) {
		title := strings.Repeat("x", 49)
		p := withContributes(t, `{"activityBar":[{"id":"a.1","icon":"i","title":"`+title+`"}]}`)
		errs := ValidateV1(p)
		if !hasError(errs, "contributes.activityBar[0].title") {
			t.Errorf("expected title-too-long error, got: %v", errs)
		}
	})
	t.Run("title at max (48 chars) passes", func(t *testing.T) {
		title := strings.Repeat("x", 48)
		p := withContributes(t, `{"activityBar":[{"id":"a.1","icon":"i","title":"`+title+`"}]}`)
		errs := ValidateV1(p)
		if hasError(errs, "contributes.activityBar[0].title") {
			t.Errorf("unexpected title error for 48-char title, got: %v", errs)
		}
	})
}

// TestValidate_Webview_ViewsContainerEnum asserts container="unknown" fails.
func TestValidate_Webview_ViewsContainerEnum(t *testing.T) {
	p := withContributes(t, `{"views":[{"id":"v.1","title":"T","container":"unknown"}]}`)
	errs := ValidateV1(p)
	if !hasError(errs, "contributes.views[0].container") {
		t.Errorf("expected views[0].container error, got: %v", errs)
	}
}

// TestValidate_Webview_ValidContainerValues checks all allowed container values pass.
func TestValidate_Webview_ValidContainerValues(t *testing.T) {
	for _, container := range []string{"activityBar", "panel", "sidebar"} {
		container := container
		t.Run(container, func(t *testing.T) {
			p := withContributes(t, `{"views":[{"id":"v.1","title":"T","container":"`+container+`"}]}`)
			errs := ValidateV1(p)
			if hasError(errs, "contributes.views[0].container") {
				t.Errorf("container=%q: unexpected error, got: %v", container, errs)
			}
		})
	}
}

// TestValidate_Webview_PanelRenderEnum asserts render="native" on a panel fails.
func TestValidate_Webview_PanelRenderEnum(t *testing.T) {
	p := withContributes(t, `{"panels":[{"id":"p.1","title":"T","render":"native"}]}`)
	errs := ValidateV1(p)
	if !hasError(errs, "contributes.panels[0].render") {
		t.Errorf("expected panels[0].render error, got: %v", errs)
	}
}

// ─── configSchema ───────────────────────────────────────────────────────────

// TestValidate_ConfigSchema_Accepts covers every v1 type + legacy aliases.
// Each field has a unique key because the validator also rejects duplicates.
func TestValidate_ConfigSchema_Accepts(t *testing.T) {
	cases := []ConfigField{
		{Key: "host", Label: "Host", Type: "string"},
		{Key: "port", Label: "Port", Type: "number"},
		{Key: "enabled", Label: "Enabled", Type: "bool"},
		{Key: "flag", Label: "Flag", Type: "boolean"}, // legacy alias
		{Key: "mode", Label: "Mode", Type: "select",
			Options: []any{"a", "b"}},
		{Key: "password", Label: "Password", Type: "secret"},
		{Key: "bio", Label: "Bio", Type: "text"}, // legacy alias
	}
	p := baseV1Provider()
	p.ConfigSchema = cases
	errs := ValidateV1(p)
	if hasError(errs, "configSchema") {
		t.Errorf("unexpected configSchema errors: %v", errs)
	}
}

func TestValidate_ConfigSchema_Rejects(t *testing.T) {
	tests := []struct {
		name   string
		schema []ConfigField
		want   string
	}{
		{
			name:   "missing key",
			schema: []ConfigField{{Label: "X", Type: "string"}},
			want:   "configSchema[0].key",
		},
		{
			name:   "bad key pattern",
			schema: []ConfigField{{Key: "has-dash", Label: "X", Type: "string"}},
			want:   "configSchema[0].key",
		},
		{
			name: "duplicate key",
			schema: []ConfigField{
				{Key: "host", Label: "A", Type: "string"},
				{Key: "host", Label: "B", Type: "string"},
			},
			want: "configSchema[1].key",
		},
		{
			name:   "missing label",
			schema: []ConfigField{{Key: "host", Type: "string"}},
			want:   "configSchema[0].label",
		},
		{
			name:   "unknown type",
			schema: []ConfigField{{Key: "x", Label: "X", Type: "widget"}},
			want:   "configSchema[0].type",
		},
		{
			name:   "select without options",
			schema: []ConfigField{{Key: "mode", Label: "Mode", Type: "select"}},
			want:   "configSchema[0].options",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := baseV1Provider()
			p.ConfigSchema = tc.schema
			errs := ValidateV1(p)
			if !hasError(errs, tc.want) {
				t.Errorf("want error at %s, got: %v", tc.want, errs)
			}
		})
	}
}

// baseV1Provider returns the minimum valid v1 manifest for tests that
// only want to exercise one sub-validator. Kept local to keep the
// fixture light.
func baseV1Provider() Provider {
	return Provider{
		Name:      "test",
		Version:   "1.0.0",
		Publisher: "opendray-examples",
		Engines:   &EnginesV1{Opendray: "^1.0.0"},
	}
}

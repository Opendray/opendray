package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	bundled "github.com/opendray/opendray/plugins"
)

// v1MigratedTier1 is the strict subset of bundled plugins that went
// through the M5 Phase 5 legacy → v1 migration. These three are
// guarded by a regression test that asserts the migration preserved
// their contributes block — i.e. someone can't strip contributes off
// claude/file-browser/terminal and still ship.
//
// Don't conflate with v1BundledAll: the other 14 bundled plugins
// (codex/gemini/pg-browser/etc.) were authored as v1 from the start
// inside the marketplace fixture, never had a legacy shape, and
// therefore have no "migration preserved contributes" invariant to
// guard — Flutter hand-routes them anyway.
var v1MigratedTier1 = map[string]bool{
	"terminal":     true, // M5 A1
	"file-browser": true, // M5 A2
	"claude":       true, // M5 A3.1
}

// v1BundledAll enumerates every plugin shipped under plugins/builtin/.
// Used to assert the universal "all bundled builtins report IsV1() ==
// true" invariant — if a new builtin lands, add it here so the
// bundled-scan test counts it.
var v1BundledAll = map[string]bool{
	"claude":            true,
	"codex":             true,
	"file-browser":      true,
	"gemini":            true,
	"log-viewer":        true,
	"mcp":               true,
	"obsidian-reader":   true,
	"opencode":          true,
	"pg-browser":        true,
	"qwen-code":         true,
	"simulator-preview": true,
	"source-control":    true,
	"task-runner":       true,
	"telegram":          true,
	"terminal":          true,
	"web-browser":       true,
}

func TestProvider_IsV1(t *testing.T) {
	cases := []struct {
		name string
		p    Provider
		want bool
	}{
		{"zero value is legacy", Provider{}, false},
		{"legacy typed manifest", Provider{Name: "x", Type: "cli", Version: "1.0.0"}, false},
		{"v1 minimal", Provider{Publisher: "me", Engines: &EnginesV1{Opendray: "^1.0.0"}}, true},
		{"publisher without engines is not v1", Provider{Publisher: "me"}, false},
		{"engines without opendray range is not v1", Provider{Publisher: "me", Engines: &EnginesV1{}}, false},
		{"engines without publisher is not v1", Provider{Engines: &EnginesV1{Opendray: "^1.0.0"}}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.p.IsV1(); got != tc.want {
				t.Errorf("IsV1() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestProvider_EffectiveForm(t *testing.T) {
	cases := []struct {
		name string
		p    Provider
		want string
	}{
		{"explicit declarative", Provider{Form: "declarative"}, "declarative"},
		{"explicit webview", Provider{Form: "webview"}, "webview"},
		{"explicit host", Provider{Form: "host"}, "host"},
		{"legacy panel maps to declarative", Provider{Type: "panel"}, "declarative"},
		{"legacy cli maps to host", Provider{Type: "cli"}, "host"},
		{"legacy local maps to host", Provider{Type: "local"}, "host"},
		{"legacy shell maps to host", Provider{Type: "shell"}, "host"},
		{"empty falls back to declarative", Provider{}, "declarative"},
		{"explicit form beats legacy type", Provider{Form: "webview", Type: "panel"}, "webview"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.p.EffectiveForm(); got != tc.want {
				t.Errorf("EffectiveForm() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestLoadManifest_LegacyCompat(t *testing.T) {
	// Every bundled manifest under plugins/builtin/ must load with its
	// identity fields intact AND report IsV1() == true. Post-plugin
	// consolidation (see v1BundledAll) there are no legacy-shaped
	// bundled manifests left; any new builtin that lands without
	// publisher+engines should fail this guard loudly so the PR author
	// knows to add them.
	providers, err := ScanFS(bundled.FS, "builtin")
	if err != nil {
		t.Fatalf("ScanFS builtin: %v", err)
	}
	if len(providers) == 0 {
		t.Fatal("no bundled plugins found — ScanFS regression")
	}

	for _, p := range providers {
		p := p
		t.Run(p.Name, func(t *testing.T) {
			if !v1BundledAll[p.Name] {
				t.Fatalf("bundled plugin %q is not in v1BundledAll — add it to the whitelist or remove the manifest", p.Name)
			}
			if !p.IsV1() {
				t.Errorf("bundled manifest %q must report IsV1()=true (needs publisher + engines.opendray)", p.Name)
			}
			if p.Name == "" {
				t.Errorf("manifest lost Name field")
			}
			if p.Version == "" {
				t.Errorf("manifest lost Version field")
			}
			if p.Type == "" {
				t.Errorf("manifest lost Type field")
			}
		})
	}
}

func TestLoadManifest_V1Superset(t *testing.T) {
	dir := t.TempDir()
	body := `{
		"name": "time-ninja",
		"displayName": "Time Ninja",
		"version": "0.1.0",
		"publisher": "opendray",
		"engines": { "opendray": "^1.0.0" },
		"form": "declarative",
		"activation": ["onCommand:time-ninja.show"],
		"contributes": {
			"commands": [
				{
					"id": "time-ninja.show",
					"title": "Time Ninja: Show Clock",
					"run": { "kind": "notify", "message": "It is now." }
				}
			],
			"statusBar": [
				{ "id": "time-ninja.clock", "text": "⏰", "command": "time-ninja.show", "alignment": "right", "priority": 100 }
			],
			"keybindings": [
				{ "command": "time-ninja.show", "key": "ctrl+alt+t" }
			],
			"menus": {
				"commandPalette": [ { "command": "time-ninja.show" } ]
			}
		},
		"permissions": {
			"exec": ["date"]
		},
		"v2Reserved": { "experimentalFlag": true }
	}`
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(body), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	p, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}

	if !p.IsV1() {
		t.Fatal("v1 manifest reports IsV1()=false")
	}
	if p.Name != "time-ninja" {
		t.Errorf("Name = %q, want time-ninja", p.Name)
	}
	if p.Publisher != "opendray" {
		t.Errorf("Publisher = %q, want opendray", p.Publisher)
	}
	if p.Engines == nil || p.Engines.Opendray != "^1.0.0" {
		t.Errorf("Engines = %+v, want Opendray=^1.0.0", p.Engines)
	}
	if p.EffectiveForm() != "declarative" {
		t.Errorf("EffectiveForm = %q, want declarative", p.EffectiveForm())
	}
	if len(p.Activation) != 1 || p.Activation[0] != "onCommand:time-ninja.show" {
		t.Errorf("Activation = %v", p.Activation)
	}
	if p.Contributes == nil {
		t.Fatal("Contributes is nil")
	}
	if len(p.Contributes.Commands) != 1 {
		t.Fatalf("Contributes.Commands = %d, want 1", len(p.Contributes.Commands))
	}
	cmd := p.Contributes.Commands[0]
	if cmd.ID != "time-ninja.show" || cmd.Title != "Time Ninja: Show Clock" {
		t.Errorf("command = %+v", cmd)
	}
	if cmd.Run == nil || cmd.Run.Kind != "notify" || cmd.Run.Message != "It is now." {
		t.Errorf("command.Run = %+v", cmd.Run)
	}
	if len(p.Contributes.StatusBar) != 1 || p.Contributes.StatusBar[0].Text != "⏰" {
		t.Errorf("StatusBar = %+v", p.Contributes.StatusBar)
	}
	if p.Contributes.StatusBar[0].Priority != 100 {
		t.Errorf("StatusBar[0].Priority = %d, want 100", p.Contributes.StatusBar[0].Priority)
	}
	if len(p.Contributes.Keybindings) != 1 || p.Contributes.Keybindings[0].Key != "ctrl+alt+t" {
		t.Errorf("Keybindings = %+v", p.Contributes.Keybindings)
	}
	if len(p.Contributes.Menus["commandPalette"]) != 1 {
		t.Errorf("Menus.commandPalette = %+v", p.Contributes.Menus)
	}
	if p.Permissions == nil {
		t.Fatal("Permissions is nil")
	}
	var execAllow []string
	if err := json.Unmarshal(p.Permissions.Exec, &execAllow); err != nil {
		t.Fatalf("permissions.exec did not round-trip as string array: %v", err)
	}
	if len(execAllow) != 1 || execAllow[0] != "date" {
		t.Errorf("permissions.exec = %v", execAllow)
	}
	if len(p.V2Reserved) == 0 {
		t.Error("V2Reserved lost during unmarshal")
	}
}

func TestLoadManifest_V1PermissionsExecBool(t *testing.T) {
	// permissions.exec may be a bare boolean too. RawMessage must preserve both shapes.
	dir := t.TempDir()
	body := `{
		"name": "trust-me",
		"version": "0.1.0",
		"publisher": "opendray",
		"engines": { "opendray": "^1.0.0" },
		"permissions": { "exec": true }
	}`
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(body), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	p, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	var asBool bool
	if err := json.Unmarshal(p.Permissions.Exec, &asBool); err != nil {
		t.Fatalf("permissions.exec did not round-trip as bool: %v", err)
	}
	if !asBool {
		t.Error("permissions.exec = false, want true")
	}
}

// ─── M2 T1 — webview contribution points ────────────────────────────

// TestLoadManifest_V1Webview loads a manifest that exercises
// activityBar / views / panels and asserts every field round-trips.
// These are the three slots M2 introduces.
func TestLoadManifest_V1Webview(t *testing.T) {
	dir := t.TempDir()
	body := `{
		"name": "kanban",
		"version": "1.0.0",
		"publisher": "opendray-examples",
		"engines": { "opendray": "^1.0.0" },
		"form": "webview",
		"contributes": {
			"activityBar": [
				{
					"id": "kanban.board",
					"icon": "📋",
					"title": "Kanban",
					"viewId": "kanban.board"
				}
			],
			"views": [
				{
					"id": "kanban.board",
					"title": "Kanban Board",
					"container": "activityBar",
					"icon": "📋",
					"when": "workspaceOpen",
					"render": "webview",
					"entry": "ui/index.html"
				}
			],
			"panels": [
				{
					"id": "kanban.console",
					"title": "Kanban Log",
					"icon": "📝",
					"position": "bottom",
					"render": "webview",
					"entry": "ui/console.html"
				}
			]
		}
	}`
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(body), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	p, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if p.Contributes == nil {
		t.Fatal("Contributes is nil")
	}

	if len(p.Contributes.ActivityBar) != 1 {
		t.Fatalf("ActivityBar = %d, want 1", len(p.Contributes.ActivityBar))
	}
	ab := p.Contributes.ActivityBar[0]
	if ab.ID != "kanban.board" || ab.Icon != "📋" || ab.Title != "Kanban" || ab.ViewID != "kanban.board" {
		t.Errorf("ActivityBar[0] = %+v", ab)
	}

	if len(p.Contributes.Views) != 1 {
		t.Fatalf("Views = %d, want 1", len(p.Contributes.Views))
	}
	v := p.Contributes.Views[0]
	if v.ID != "kanban.board" || v.Title != "Kanban Board" ||
		v.Container != "activityBar" || v.Render != "webview" ||
		v.Entry != "ui/index.html" || v.When != "workspaceOpen" {
		t.Errorf("Views[0] = %+v", v)
	}

	if len(p.Contributes.Panels) != 1 {
		t.Fatalf("Panels = %d, want 1", len(p.Contributes.Panels))
	}
	pn := p.Contributes.Panels[0]
	if pn.ID != "kanban.console" || pn.Position != "bottom" ||
		pn.Render != "webview" || pn.Entry != "ui/console.html" {
		t.Errorf("Panels[0] = %+v", pn)
	}
}

// TestLoadManifest_V1WebviewOmittedDefaultsEmpty confirms the new fields
// follow omitempty discipline — when a manifest leaves them off, the
// parsed struct must end up with nil/empty slices (not surprising
// default shapes that leak into the wire format).
func TestLoadManifest_V1WebviewOmittedDefaultsEmpty(t *testing.T) {
	dir := t.TempDir()
	body := `{
		"name": "minimal-webview",
		"version": "0.1.0",
		"publisher": "opendray",
		"engines": { "opendray": "^1.0.0" },
		"contributes": { "commands": [{ "id": "m.hi", "title": "Hi" }] }
	}`
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(body), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	p, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if p.Contributes == nil {
		t.Fatal("Contributes is nil despite commands being set")
	}
	if p.Contributes.ActivityBar != nil {
		t.Errorf("ActivityBar should be nil when absent, got %v", p.Contributes.ActivityBar)
	}
	if p.Contributes.Views != nil {
		t.Errorf("Views should be nil when absent, got %v", p.Contributes.Views)
	}
	if p.Contributes.Panels != nil {
		t.Errorf("Panels should be nil when absent, got %v", p.Contributes.Panels)
	}
}

// TestTier1Migrated_Tier1PluginsRemainV1 is the M5 A4 regression guard.
// Every bundled plugin in the v1MigratedTier1 set must continue to load
// as a fully-v1 manifest: IsV1() true, publisher non-empty, engines.opendray
// set, and the embedded contributes block honoured (not synthesised). A
// drift here means someone stripped required fields off a migrated manifest.
func TestTier1Migrated_Tier1PluginsRemainV1(t *testing.T) {
	ps, err := ScanFS(bundled.FS, "builtin")
	if err != nil {
		t.Fatalf("ScanFS builtin: %v", err)
	}
	found := make(map[string]bool)
	for _, p := range ps {
		if !v1MigratedTier1[p.Name] {
			continue
		}
		found[p.Name] = true
		if !p.IsV1() {
			t.Errorf("%s: IsV1() = false after Phase 5 migration", p.Name)
		}
		if p.Publisher == "" {
			t.Errorf("%s: Publisher empty", p.Name)
		}
		if p.Engines == nil || p.Engines.Opendray == "" {
			t.Errorf("%s: engines.opendray empty", p.Name)
		}
		if p.Contributes == nil {
			t.Errorf("%s: contributes block missing", p.Name)
		}
	}
	for name := range v1MigratedTier1 {
		if !found[name] {
			t.Errorf("tier-1 plugin %q vanished from bundled FS", name)
		}
	}
}

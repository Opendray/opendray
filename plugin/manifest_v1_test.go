package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	bundled "github.com/opendray/opendray/plugins"
)

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
	// Bundled plugin manifests under plugins/agents and plugins/panels must
	// continue to load without losing identity fields. Mixed shape is
	// expected: some are still legacy (no publisher/engines), others
	// migrated to v1 under M5 Phase 5 (A1+A2 so far — terminal,
	// file-browser). Both must retain Name/Version/Type; only the
	// *legacy* set is asserted to report IsV1()==false.
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

	// v1Migrated is the set of bundled manifests that intentionally opted
	// into the v1 contract. Extend as each Phase 5 task lands.
	v1Migrated := map[string]bool{
		"terminal":     true, // M5 A1
		"file-browser": true, // M5 A2
	}

	for _, p := range providers {
		p := p
		t.Run(p.Name, func(t *testing.T) {
			if v1Migrated[p.Name] {
				if !p.IsV1() {
					t.Errorf("migrated manifest %q must report IsV1()=true", p.Name)
				}
			} else if p.IsV1() {
				t.Errorf("legacy manifest %q reports IsV1()=true; v1 opt-in should require publisher+engines", p.Name)
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

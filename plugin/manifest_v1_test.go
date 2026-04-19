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
	// Every bundled plugin manifest under plugins/agents and plugins/panels
	// must continue to load unchanged: IsV1()==false, and Name/Version/Type
	// populated exactly like they are on main.
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
			if p.IsV1() {
				t.Errorf("legacy manifest reports IsV1()=true; v1 opt-in should require publisher+engines")
			}
			if p.Name == "" {
				t.Errorf("legacy manifest lost Name field")
			}
			if p.Version == "" {
				t.Errorf("legacy manifest lost Version field")
			}
			if p.Type == "" {
				t.Errorf("legacy manifest lost Type field")
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

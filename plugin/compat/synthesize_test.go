package compat_test

import (
	"io/fs"
	"testing"

	"github.com/opendray/opendray/plugin"
	"github.com/opendray/opendray/plugin/compat"
	bundled "github.com/opendray/opendray/plugins"
)

// ── T1: CLI agent legacy manifest ───────────────────────────────────────────

func TestSynthesize_CLIAgent(t *testing.T) {
	p := plugin.Provider{
		Name:        "my-agent",
		DisplayName: "My Agent",
		Version:     "1.0.0",
		Type:        plugin.ProviderTypeCLI,
		CLI: &plugin.CLISpec{
			Command: "my-agent-cli",
		},
		Capabilities: plugin.Capabilities{
			SupportsStream: true,
			SupportsResume: true,
		},
	}

	got := compat.Synthesize(p)

	// Identity-level v1 fields must be populated.
	if got.Publisher != "opendray-builtin" {
		t.Errorf("Publisher = %q; want %q", got.Publisher, "opendray-builtin")
	}
	if got.Engines == nil || got.Engines.Opendray != ">=0" {
		opendray := ""
		if got.Engines != nil {
			opendray = got.Engines.Opendray
		}
		t.Errorf("Engines.Opendray = %q; want %q", opendray, ">=0")
	}
	if got.EffectiveForm() != plugin.FormHost {
		t.Errorf("EffectiveForm() = %q; want %q", got.EffectiveForm(), plugin.FormHost)
	}
	if len(got.Activation) != 1 || got.Activation[0] != "onStartup" {
		t.Errorf("Activation = %v; want [onStartup]", got.Activation)
	}

	// ContributesV1 is M1-only; it has no AgentProviders field.
	// The overlay carries empty Contributes — the registry will accept it
	// (isZero == true means Set is a no-op, which is correct for builtins).
	// This is expected by design: M1 ContributesV1 has no AgentProviders.

	// Legacy fields must be preserved intact.
	if got.Type != "cli" {
		t.Errorf("Type = %q; want %q", got.Type, "cli")
	}
	if got.CLI == nil || got.CLI.Command != "my-agent-cli" {
		t.Errorf("CLI.Command not preserved")
	}
	if !got.Capabilities.SupportsStream {
		t.Errorf("Capabilities.SupportsStream not preserved")
	}
	if !got.Capabilities.SupportsResume {
		t.Errorf("Capabilities.SupportsResume not preserved")
	}

	// IsV1 must return true after synthesis.
	if !got.IsV1() {
		t.Errorf("IsV1() = false after Synthesize; want true")
	}

	// The result must be a new value — caller cannot assume pointer equality.
	// Since Provider is a value type this is structural. Just verify Name/Version
	// match to confirm identity copy.
	if got.Name != p.Name || got.Version != p.Version {
		t.Errorf("identity fields changed: Name=%q Version=%q", got.Name, got.Version)
	}
}

// ── T2: Panel legacy manifest ────────────────────────────────────────────────

func TestSynthesize_Panel(t *testing.T) {
	p := plugin.Provider{
		Name:        "my-panel",
		DisplayName: "My Panel",
		Version:     "2.0.0",
		Type:        plugin.ProviderTypePanel,
		Category:    "tools",
	}

	got := compat.Synthesize(p)

	if got.Publisher != "opendray-builtin" {
		t.Errorf("Publisher = %q; want %q", got.Publisher, "opendray-builtin")
	}
	if got.Engines == nil || got.Engines.Opendray != ">=0" {
		t.Errorf("Engines.Opendray not set to >=0")
	}
	// Panel → declarative form.
	if got.EffectiveForm() != plugin.FormDeclarative {
		t.Errorf("EffectiveForm() = %q; want %q", got.EffectiveForm(), plugin.FormDeclarative)
	}
	if len(got.Activation) != 1 || got.Activation[0] != "onStartup" {
		t.Errorf("Activation = %v; want [onStartup]", got.Activation)
	}
	// Legacy fields preserved.
	if got.Type != "panel" {
		t.Errorf("Type = %q; want %q", got.Type, "panel")
	}
	if got.Category != "tools" {
		t.Errorf("Category = %q; want %q", got.Category, "tools")
	}
	if !got.IsV1() {
		t.Errorf("IsV1() = false after Synthesize; want true")
	}
}

// ── T3: All 17 bundled manifests ────────────────────────────────────────────

func TestSynthesize_BundledManifests(t *testing.T) {
	var providers []plugin.Provider
	for _, root := range []string{"agents", "panels"} {
		ps, err := plugin.ScanFS(bundled.FS, root)
		if err != nil {
			t.Fatalf("ScanFS(%q): %v", root, err)
		}
		providers = append(providers, ps...)
	}

	if len(providers) == 0 {
		t.Fatal("no bundled providers found; check plugins.FS")
	}
	t.Logf("testing compat synthesis on %d bundled manifests", len(providers))

	for _, p := range providers {
		p := p // capture
		t.Run(p.Name, func(t *testing.T) {
			// Pre-condition: bundled manifests are legacy.
			if p.IsV1() {
				t.Fatalf("bundled manifest %q unexpectedly has IsV1()=true before synthesis", p.Name)
			}

			got := compat.Synthesize(p)

			if got.Publisher != "opendray-builtin" {
				t.Errorf("Publisher = %q; want opendray-builtin", got.Publisher)
			}
			if !got.IsV1() {
				t.Errorf("IsV1() = false after Synthesize")
			}
			if got.Name != p.Name {
				t.Errorf("Name changed: got %q, want %q", got.Name, p.Name)
			}
			if got.Version != p.Version {
				t.Errorf("Version changed: got %q, want %q", got.Version, p.Version)
			}
		})
	}
}

// ── T4: V1 pass-through ─────────────────────────────────────────────────────

func TestSynthesize_V1Passthrough(t *testing.T) {
	engines := &plugin.EnginesV1{Opendray: "^1.0.0"}
	p := plugin.Provider{
		Name:      "my-v1-plugin",
		Version:   "1.0.0",
		Publisher: "my-publisher",
		Engines:   engines,
		Form:      plugin.FormDeclarative,
		Contributes: &plugin.ContributesV1{
			Commands: []plugin.CommandV1{
				{ID: "my.cmd", Title: "My Command"},
			},
		},
	}

	if !p.IsV1() {
		t.Fatal("test setup: p.IsV1() must be true")
	}

	got := compat.Synthesize(p)

	// Publisher must be preserved (not overwritten with opendray-builtin).
	if got.Publisher != "my-publisher" {
		t.Errorf("Publisher = %q; want %q", got.Publisher, "my-publisher")
	}
	// Engines preserved.
	if got.Engines == nil || got.Engines.Opendray != "^1.0.0" {
		t.Errorf("Engines.Opendray = %q; want ^1.0.0", got.Engines.Opendray)
	}
	// Contributes preserved.
	if got.Contributes == nil || len(got.Contributes.Commands) != 1 {
		t.Errorf("Contributes.Commands not preserved")
	} else if got.Contributes.Commands[0].ID != "my.cmd" {
		t.Errorf("Commands[0].ID = %q; want %q", got.Contributes.Commands[0].ID, "my.cmd")
	}
	// IsV1 still true.
	if !got.IsV1() {
		t.Errorf("IsV1() = false after passthrough")
	}
}

// ── Additional coverage: deep-copy isolation ─────────────────────────────────

// TestSynthesize_LegacyWithConfigSchema verifies that ConfigSchema is preserved
// (and deep-copied) through the synthesizer on legacy manifests.
func TestSynthesize_LegacyWithConfigSchema(t *testing.T) {
	p := plugin.Provider{
		Name:    "agent-with-schema",
		Version: "1.0.0",
		Type:    plugin.ProviderTypeCLI,
		CLI:     &plugin.CLISpec{Command: "tool"},
		ConfigSchema: []plugin.ConfigField{
			{Key: "apiKey", Label: "API Key", Type: "secret"},
		},
	}

	got := compat.Synthesize(p)

	if len(got.ConfigSchema) != 1 {
		t.Fatalf("ConfigSchema length: got %d, want 1", len(got.ConfigSchema))
	}
	if got.ConfigSchema[0].Key != "apiKey" {
		t.Errorf("ConfigSchema[0].Key = %q; want apiKey", got.ConfigSchema[0].Key)
	}
	// Mutation of the result must not affect the source.
	got.ConfigSchema[0].Key = "mutated"
	if p.ConfigSchema[0].Key != "apiKey" {
		t.Errorf("deep-copy violated: source ConfigSchema[0].Key changed to %q", p.ConfigSchema[0].Key)
	}
}

// TestSynthesize_V1WithFullContribs exercises the ContributesV1 deep-copy path
// on a v1 passthrough that has all contribution slots populated.
func TestSynthesize_V1WithFullContribs(t *testing.T) {
	p := plugin.Provider{
		Name:      "full-plugin",
		Version:   "1.0.0",
		Publisher: "test-pub",
		Engines:   &plugin.EnginesV1{Opendray: "^1.0.0"},
		Activation: []string{"onStartup", "onCommand:do.thing"},
		Contributes: &plugin.ContributesV1{
			Commands: []plugin.CommandV1{
				{ID: "do.thing", Title: "Do Thing"},
			},
			StatusBar: []plugin.StatusBarItemV1{
				{ID: "do.bar", Text: "Do", Alignment: "right", Priority: 10},
			},
			Keybindings: []plugin.KeybindingV1{
				{Command: "do.thing", Key: "ctrl+d"},
			},
			Menus: map[string][]plugin.MenuEntryV1{
				"appBar/right": {{Command: "do.thing"}},
			},
		},
		Permissions: &plugin.PermissionsV1{Storage: true},
	}

	got := compat.Synthesize(p)

	if !got.IsV1() {
		t.Errorf("IsV1() = false after v1 passthrough")
	}
	if got.Contributes == nil {
		t.Fatal("Contributes nil after passthrough")
	}
	if len(got.Contributes.Commands) != 1 {
		t.Fatalf("Commands len = %d; want 1", len(got.Contributes.Commands))
	}
	if len(got.Contributes.StatusBar) != 1 {
		t.Fatalf("StatusBar len = %d; want 1", len(got.Contributes.StatusBar))
	}
	if len(got.Contributes.Keybindings) != 1 {
		t.Fatalf("Keybindings len = %d; want 1", len(got.Contributes.Keybindings))
	}
	if len(got.Contributes.Menus["appBar/right"]) != 1 {
		t.Fatalf("Menus[appBar/right] len = %d; want 1", len(got.Contributes.Menus["appBar/right"]))
	}
	if len(got.Activation) != 2 {
		t.Fatalf("Activation len = %d; want 2", len(got.Activation))
	}
	if got.Permissions == nil || !got.Permissions.Storage {
		t.Errorf("Permissions.Storage not preserved")
	}

	// Mutation isolation: mutating got.Contributes must not affect p.
	got.Contributes.Commands[0].ID = "mutated"
	if p.Contributes.Commands[0].ID != "do.thing" {
		t.Errorf("deep-copy violated: source Commands[0].ID = %q", p.Contributes.Commands[0].ID)
	}
}

// ── T5: No disk rewrite ──────────────────────────────────────────────────────

func TestCompat_NoDiskRewrite(t *testing.T) {
	// Read the raw bytes for one bundled manifest directly from the FS.
	manifestPath := "agents/claude/manifest.json"
	before, err := fs.ReadFile(bundled.FS, manifestPath)
	if err != nil {
		t.Fatalf("read bundled manifest: %v", err)
	}

	// Parse and synthesize.
	providers, err := plugin.ScanFS(bundled.FS, "agents")
	if err != nil {
		t.Fatalf("ScanFS: %v", err)
	}
	var claude plugin.Provider
	for _, p := range providers {
		if p.Name == "claude" {
			claude = p
			break
		}
	}
	if claude.Name == "" {
		t.Fatal("claude provider not found in bundled agents")
	}

	_ = compat.Synthesize(claude)

	// Re-read the bytes — must be byte-identical.
	after, err := fs.ReadFile(bundled.FS, manifestPath)
	if err != nil {
		t.Fatalf("re-read bundled manifest: %v", err)
	}

	if string(before) != string(after) {
		t.Errorf("manifest bytes changed after Synthesize — disk rewrite invariant violated")
	}
}

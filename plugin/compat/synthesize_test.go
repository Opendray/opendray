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

// ── M2 T4: Panel → synthesized view ─────────────────────────────────────────

// TestSynthesize_PanelGetsView verifies that a legacy panel provider gets a
// single synthesized Views entry with the correct fields.
func TestSynthesize_PanelGetsView(t *testing.T) {
	p := plugin.Provider{
		Name:        "git",
		DisplayName: "Git",
		Version:     "1.0.0",
		Type:        plugin.ProviderTypePanel,
	}

	got := compat.Synthesize(p)

	if got.Contributes == nil {
		t.Fatal("Contributes is nil after Synthesize for panel type")
	}
	if len(got.Contributes.Views) != 1 {
		t.Fatalf("Contributes.Views length = %d; want 1", len(got.Contributes.Views))
	}

	v := got.Contributes.Views[0]
	if v.ID != "git" {
		t.Errorf("Views[0].ID = %q; want %q", v.ID, "git")
	}
	if v.Title != "Git" {
		t.Errorf("Views[0].Title = %q; want %q", v.Title, "Git")
	}
	if v.Container != "activityBar" {
		t.Errorf("Views[0].Container = %q; want %q", v.Container, "activityBar")
	}
	if v.Render != "declarative" {
		t.Errorf("Views[0].Render = %q; want %q", v.Render, "declarative")
	}
	if v.Entry != "" {
		t.Errorf("Views[0].Entry = %q; want empty (legacy panels keep bespoke widget rendering)", v.Entry)
	}
}

// TestSynthesize_PanelFallbackDisplayName verifies that when DisplayName is
// empty, the synthesized view Title falls back to Name.
func TestSynthesize_PanelFallbackDisplayName(t *testing.T) {
	p := plugin.Provider{
		Name:        "my-panel",
		DisplayName: "", // intentionally empty
		Version:     "1.0.0",
		Type:        plugin.ProviderTypePanel,
	}

	got := compat.Synthesize(p)

	if got.Contributes == nil {
		t.Fatal("Contributes is nil")
	}
	if len(got.Contributes.Views) != 1 {
		t.Fatalf("Views length = %d; want 1", len(got.Contributes.Views))
	}
	if got.Contributes.Views[0].Title != "my-panel" {
		t.Errorf("Views[0].Title = %q; want %q (fallback to Name)", got.Contributes.Views[0].Title, "my-panel")
	}
}

// TestSynthesize_AgentDoesNotGetView verifies that a cli-type provider does
// not receive any synthesized view entries.
func TestSynthesize_AgentDoesNotGetView(t *testing.T) {
	p := plugin.Provider{
		Name:    "my-agent",
		Version: "1.0.0",
		Type:    plugin.ProviderTypeCLI,
		CLI:     &plugin.CLISpec{Command: "my-agent-cli"},
	}

	got := compat.Synthesize(p)

	if got.Contributes != nil && len(got.Contributes.Views) != 0 {
		t.Errorf("Contributes.Views = %v; want empty for cli type", got.Contributes.Views)
	}
}

// TestSynthesize_LocalShellDoNotGetViews verifies that local and shell type
// providers do not receive any synthesized view entries.
func TestSynthesize_LocalShellDoNotGetViews(t *testing.T) {
	for _, typ := range []string{plugin.ProviderTypeLocal, plugin.ProviderTypeShell} {
		typ := typ
		t.Run(typ, func(t *testing.T) {
			p := plugin.Provider{
				Name:    "my-provider",
				Version: "1.0.0",
				Type:    typ,
			}

			got := compat.Synthesize(p)

			if got.Contributes != nil && len(got.Contributes.Views) != 0 {
				t.Errorf("type=%q: Contributes.Views = %v; want empty", typ, got.Contributes.Views)
			}
		})
	}
}

// TestSynthesize_AllBundledPanelsGetOneView iterates all bundled panel manifests
// via ScanFS and verifies each gets exactly one synthesized view whose title
// matches the manifest's DisplayName.
func TestSynthesize_AllBundledPanelsGetOneView(t *testing.T) {
	panels, err := plugin.ScanFS(bundled.FS, "panels")
	if err != nil {
		t.Fatalf("ScanFS(panels): %v", err)
	}
	if len(panels) == 0 {
		t.Fatal("no bundled panels found; check plugins.FS")
	}
	t.Logf("testing view synthesis on %d bundled panel manifests", len(panels))

	for _, p := range panels {
		p := p
		t.Run(p.Name, func(t *testing.T) {
			got := compat.Synthesize(p)

			if got.Contributes == nil {
				t.Fatal("Contributes is nil")
			}
			if len(got.Contributes.Views) != 1 {
				t.Fatalf("Views length = %d; want 1", len(got.Contributes.Views))
			}

			v := got.Contributes.Views[0]
			if v.ID != p.Name {
				t.Errorf("Views[0].ID = %q; want %q", v.ID, p.Name)
			}
			// DisplayName is always set (ScanFS falls back to Name if empty).
			wantTitle := p.DisplayName
			if wantTitle == "" {
				wantTitle = p.Name
			}
			if v.Title != wantTitle {
				t.Errorf("Views[0].Title = %q; want %q", v.Title, wantTitle)
			}
			if v.Container != "activityBar" {
				t.Errorf("Views[0].Container = %q; want activityBar", v.Container)
			}
			if v.Render != "declarative" {
				t.Errorf("Views[0].Render = %q; want declarative", v.Render)
			}
		})
	}
}

// TestCompat_NoDiskRewrite_Panel verifies that synthesizing a bundled panel
// manifest leaves the on-disk bytes byte-identical.
func TestCompat_NoDiskRewrite_Panel(t *testing.T) {
	manifestPath := "panels/git/manifest.json"
	before, err := fs.ReadFile(bundled.FS, manifestPath)
	if err != nil {
		t.Fatalf("read bundled panel manifest: %v", err)
	}

	panels, err := plugin.ScanFS(bundled.FS, "panels")
	if err != nil {
		t.Fatalf("ScanFS: %v", err)
	}
	var gitPanel plugin.Provider
	for _, p := range panels {
		if p.Name == "git" {
			gitPanel = p
			break
		}
	}
	if gitPanel.Name == "" {
		t.Fatal("git panel not found in bundled panels")
	}

	_ = compat.Synthesize(gitPanel)

	after, err := fs.ReadFile(bundled.FS, manifestPath)
	if err != nil {
		t.Fatalf("re-read bundled panel manifest: %v", err)
	}

	if string(before) != string(after) {
		t.Errorf("panel manifest bytes changed after Synthesize — disk rewrite invariant violated")
	}
}

// TestSynthesize_PanelIdempotentGuard verifies the defensive idempotency guard:
// if the panel provider's Contributes.Views already contains a view with the
// same ID as p.Name (e.g. the caller accidentally re-synthesizes), the
// synthesizer does NOT add a duplicate.
func TestSynthesize_PanelIdempotentGuard(t *testing.T) {
	// Construct a legacy panel that already has a view with its own name.
	// This simulates a re-synthesis scenario.
	p := plugin.Provider{
		Name:        "git",
		DisplayName: "Git",
		Version:     "1.0.0",
		Type:        plugin.ProviderTypePanel,
		Contributes: &plugin.ContributesV1{
			Views: []plugin.ViewV1{
				{ID: "git", Title: "Git", Container: "activityBar", Render: "declarative"},
			},
		},
	}

	got := compat.Synthesize(p)

	if got.Contributes == nil {
		t.Fatal("Contributes is nil")
	}
	if len(got.Contributes.Views) != 1 {
		t.Errorf("Views length = %d; want 1 (no duplication)", len(got.Contributes.Views))
	}
}

// TestCopyProvider_ActivityBarPanels exercises the ActivityBar and Panels
// deep-copy paths in copyProvider via the v1 pass-through path (IsV1=true).
func TestCopyProvider_ActivityBarPanels(t *testing.T) {
	p := plugin.Provider{
		Name:      "webview-plugin",
		Version:   "1.0.0",
		Publisher: "test-pub",
		Engines:   &plugin.EnginesV1{Opendray: "^1.0.0"},
		Contributes: &plugin.ContributesV1{
			ActivityBar: []plugin.ActivityBarItemV1{
				{ID: "wb.bar", Icon: "icon.svg", Title: "WorkBench", ViewID: "wb.view"},
			},
			Views: []plugin.ViewV1{
				{ID: "wb.view", Title: "WorkBench", Container: "activityBar", Render: "webview", Entry: "ui/index.html"},
			},
			Panels: []plugin.PanelV1{
				{ID: "wb.panel", Title: "WorkBench Panel", Position: "bottom", Render: "webview", Entry: "ui/panel.html"},
			},
		},
	}

	if !p.IsV1() {
		t.Fatal("test setup: p.IsV1() must be true")
	}

	got := compat.Synthesize(p)

	if got.Contributes == nil {
		t.Fatal("Contributes is nil after passthrough")
	}
	if len(got.Contributes.ActivityBar) != 1 {
		t.Fatalf("ActivityBar len = %d; want 1", len(got.Contributes.ActivityBar))
	}
	if len(got.Contributes.Views) != 1 {
		t.Fatalf("Views len = %d; want 1", len(got.Contributes.Views))
	}
	if len(got.Contributes.Panels) != 1 {
		t.Fatalf("Panels len = %d; want 1", len(got.Contributes.Panels))
	}

	// Verify deep-copy isolation for all three new slice types.
	got.Contributes.ActivityBar[0].ID = "mutated"
	if p.Contributes.ActivityBar[0].ID != "wb.bar" {
		t.Errorf("ActivityBar deep-copy violated: source ID = %q", p.Contributes.ActivityBar[0].ID)
	}
	got.Contributes.Views[0].ID = "mutated"
	if p.Contributes.Views[0].ID != "wb.view" {
		t.Errorf("Views deep-copy violated: source ID = %q", p.Contributes.Views[0].ID)
	}
	got.Contributes.Panels[0].ID = "mutated"
	if p.Contributes.Panels[0].ID != "wb.panel" {
		t.Errorf("Panels deep-copy violated: source ID = %q", p.Contributes.Panels[0].ID)
	}
}

// TestSynthesize_V1ManifestViewsUntouched verifies that a v1 manifest with
// explicitly declared views is passed through unchanged (no duplication, no
// fabrication by the synthesizer).
func TestSynthesize_V1ManifestViewsUntouched(t *testing.T) {
	p := plugin.Provider{
		Name:      "my-v1-panel",
		Version:   "1.0.0",
		Publisher: "my-publisher",
		Engines:   &plugin.EnginesV1{Opendray: "^1.0.0"},
		Form:      plugin.FormDeclarative,
		Contributes: &plugin.ContributesV1{
			Views: []plugin.ViewV1{
				{
					ID:        "my-view",
					Title:     "My View",
					Container: "activityBar",
					Render:    "webview",
					Entry:     "ui/index.html",
				},
			},
		},
	}

	if !p.IsV1() {
		t.Fatal("test setup: p.IsV1() must be true")
	}

	got := compat.Synthesize(p)

	// Views must be preserved exactly — no new synthetic entry added.
	if got.Contributes == nil {
		t.Fatal("Contributes is nil after v1 passthrough")
	}
	if len(got.Contributes.Views) != 1 {
		t.Fatalf("Views length = %d; want exactly 1 (original, no duplication)", len(got.Contributes.Views))
	}
	v := got.Contributes.Views[0]
	if v.ID != "my-view" {
		t.Errorf("Views[0].ID = %q; want %q", v.ID, "my-view")
	}
	if v.Entry != "ui/index.html" {
		t.Errorf("Views[0].Entry = %q; want %q", v.Entry, "ui/index.html")
	}
}

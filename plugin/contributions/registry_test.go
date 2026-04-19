package contributions_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/opendray/opendray/kernel/store"
	"github.com/opendray/opendray/plugin"
	"github.com/opendray/opendray/plugin/contributions"
)

// ── helpers ────────────────────────────────────────────────────────────────

func makeContrib(
	cmds []plugin.CommandV1,
	sb []plugin.StatusBarItemV1,
	kb []plugin.KeybindingV1,
	menus map[string][]plugin.MenuEntryV1,
) plugin.ContributesV1 {
	return plugin.ContributesV1{
		Commands:    cmds,
		StatusBar:   sb,
		Keybindings: kb,
		Menus:       menus,
	}
}

// ── Test 1: Set then Flatten ───────────────────────────────────────────────

// TestRegistry_SetThenFlatten verifies that after setting two plugins with
// overlapping and disjoint commands, Flatten returns deterministic ordering
// and every entry carries the correct PluginName.
func TestRegistry_SetThenFlatten(t *testing.T) {
	r := contributions.NewRegistry()

	alphaContrib := makeContrib(
		[]plugin.CommandV1{
			{ID: "alpha.b", Title: "Alpha B"},
			{ID: "alpha.a", Title: "Alpha A"},
		},
		nil, nil, nil,
	)
	betaContrib := makeContrib(
		[]plugin.CommandV1{
			{ID: "beta.x", Title: "Beta X"},
		},
		nil, nil, nil,
	)

	r.Set("alpha", alphaContrib)
	r.Set("beta", betaContrib)

	flat := r.Flatten()

	if len(flat.Commands) != 3 {
		t.Fatalf("expected 3 commands, got %d: %+v", len(flat.Commands), flat.Commands)
	}

	// Sorted by (PluginName asc, ID asc) → alpha.a, alpha.b, beta.x
	wantOrder := []struct {
		pluginName string
		id         string
	}{
		{"alpha", "alpha.a"},
		{"alpha", "alpha.b"},
		{"beta", "beta.x"},
	}
	for i, w := range wantOrder {
		got := flat.Commands[i]
		if got.PluginName != w.pluginName {
			t.Errorf("Commands[%d].PluginName: got %q, want %q", i, got.PluginName, w.pluginName)
		}
		if got.ID != w.id {
			t.Errorf("Commands[%d].ID: got %q, want %q", i, got.ID, w.id)
		}
	}
}

// ── Test 2: Remove ─────────────────────────────────────────────────────────

// TestRegistry_Remove verifies that after removing one plugin, only the
// other's contributions remain in Flatten.
func TestRegistry_Remove(t *testing.T) {
	r := contributions.NewRegistry()

	r.Set("pluginA", makeContrib(
		[]plugin.CommandV1{{ID: "a.cmd", Title: "A Command"}},
		nil, nil, nil,
	))
	r.Set("pluginB", makeContrib(
		[]plugin.CommandV1{{ID: "b.cmd", Title: "B Command"}},
		nil, nil, nil,
	))

	r.Remove("pluginA")

	flat := r.Flatten()
	if len(flat.Commands) != 1 {
		t.Fatalf("expected 1 command after remove, got %d", len(flat.Commands))
	}
	if flat.Commands[0].PluginName != "pluginB" {
		t.Errorf("expected pluginB, got %q", flat.Commands[0].PluginName)
	}
	if flat.Commands[0].ID != "b.cmd" {
		t.Errorf("expected b.cmd, got %q", flat.Commands[0].ID)
	}
}

// ── Test 3: Remove Unknown ─────────────────────────────────────────────────

// TestRegistry_RemoveUnknown verifies removing a name that was never set
// is a no-op and does not panic.
func TestRegistry_RemoveUnknown(t *testing.T) {
	r := contributions.NewRegistry()

	// Should not panic.
	r.Remove("nobody")

	flat := r.Flatten()
	if len(flat.Commands) != 0 {
		t.Errorf("expected empty commands, got %d", len(flat.Commands))
	}
}

// ── Test 4: Has ────────────────────────────────────────────────────────────

// TestRegistry_Has verifies Has reports correctly before/after Set/Remove.
func TestRegistry_Has(t *testing.T) {
	r := contributions.NewRegistry()

	if r.Has("myPlugin") {
		t.Error("Has should return false before Set")
	}

	r.Set("myPlugin", makeContrib(
		[]plugin.CommandV1{{ID: "cmd.one", Title: "One"}},
		nil, nil, nil,
	))

	if !r.Has("myPlugin") {
		t.Error("Has should return true after Set")
	}

	r.Remove("myPlugin")

	if r.Has("myPlugin") {
		t.Error("Has should return false after Remove")
	}
}

// ── Test 5: Set empty clears ───────────────────────────────────────────────

// TestRegistry_SetEmptyClears verifies that Set with a zero ContributesV1
// causes the entry to be absent in Flatten and Has returns false.
func TestRegistry_SetEmptyClears(t *testing.T) {
	r := contributions.NewRegistry()

	r.Set("myPlugin", makeContrib(
		[]plugin.CommandV1{{ID: "cmd.one", Title: "One"}},
		nil, nil, nil,
	))

	// Overwrite with zero value.
	r.Set("myPlugin", plugin.ContributesV1{})

	flat := r.Flatten()
	if len(flat.Commands) != 0 {
		t.Errorf("expected 0 commands after clearing, got %d", len(flat.Commands))
	}

	if r.Has("myPlugin") {
		t.Error("Has should return false for empty contributions")
	}
}

// ── Test 6: Concurrent Set/Remove ─────────────────────────────────────────

// TestRegistry_ConcurrentSetRemove runs 100 goroutines (half setting, half
// removing) concurrently and verifies no data race occurs and Flatten does
// not panic. Run with -race.
func TestRegistry_ConcurrentSetRemove(t *testing.T) {
	r := contributions.NewRegistry()

	const n = 100
	names := []string{"pluginA", "pluginB", "pluginC", "pluginD", "pluginE"}
	contrib := makeContrib(
		[]plugin.CommandV1{{ID: "cmd.x", Title: "X"}},
		nil, nil, nil,
	)

	var wg sync.WaitGroup
	wg.Add(n)
	for i := range n {
		go func(i int) {
			defer wg.Done()
			name := names[i%len(names)]
			if i%2 == 0 {
				r.Set(name, contrib)
			} else {
				r.Remove(name)
			}
		}(i)
	}
	wg.Wait()

	// Must not panic; we can't assert exact state since concurrent ops are
	// non-deterministic, but the registry must be internally consistent.
	_ = r.Flatten()
}

// ── Test 7: Flatten stable order ──────────────────────────────────────────

// TestRegistry_FlattenStableOrder verifies that calling Flatten twice
// returns byte-identical results (no non-deterministic map iteration).
func TestRegistry_FlattenStableOrder(t *testing.T) {
	r := contributions.NewRegistry()

	contrib := makeContrib(
		[]plugin.CommandV1{
			{ID: "cmd.z", Title: "Z"},
			{ID: "cmd.a", Title: "A"},
		},
		[]plugin.StatusBarItemV1{
			{ID: "sb.one", Text: "One", Alignment: "left", Priority: 10},
			{ID: "sb.two", Text: "Two", Alignment: "right", Priority: 5},
		},
		[]plugin.KeybindingV1{
			{Command: "cmd.z", Key: "ctrl+z"},
			{Command: "cmd.a", Key: "ctrl+a"},
		},
		map[string][]plugin.MenuEntryV1{
			"appBar/right": {{Command: "cmd.z", Group: "z@1"}},
		},
	)
	r.Set("stable", contrib)

	flat1 := r.Flatten()
	flat2 := r.Flatten()

	b1, _ := json.Marshal(flat1)
	b2, _ := json.Marshal(flat2)
	if string(b1) != string(b2) {
		t.Errorf("Flatten not stable:\n  first:  %s\n  second: %s", b1, b2)
	}

	if !reflect.DeepEqual(flat1, flat2) {
		t.Errorf("Flatten not DeepEqual between calls")
	}
}

// ── Test 8: StatusBar sort by priority ────────────────────────────────────

// TestRegistry_StatusBarSortByPriority verifies the status bar sort rules:
//   - alignment "left" before "right"
//   - within same alignment: priority desc (higher first)
//   - tie-break on priority: PluginName asc
func TestRegistry_StatusBarSortByPriority(t *testing.T) {
	r := contributions.NewRegistry()

	// Plugin "alpha" contributes: left/10 and right/50
	r.Set("alpha", makeContrib(nil, []plugin.StatusBarItemV1{
		{ID: "alpha.left", Text: "AL", Alignment: "left", Priority: 10},
		{ID: "alpha.right", Text: "AR", Alignment: "right", Priority: 50},
	}, nil, nil))

	// Plugin "beta" contributes: left/100 and right/50 (tie with alpha.right → pluginName asc)
	r.Set("beta", makeContrib(nil, []plugin.StatusBarItemV1{
		{ID: "beta.left", Text: "BL", Alignment: "left", Priority: 100},
		{ID: "beta.right", Text: "BR", Alignment: "right", Priority: 50},
	}, nil, nil))

	// Plugin "gamma" contributes: right/5
	r.Set("gamma", makeContrib(nil, []plugin.StatusBarItemV1{
		{ID: "gamma.right", Text: "GR", Alignment: "right", Priority: 5},
	}, nil, nil))

	flat := r.Flatten()

	if len(flat.StatusBar) != 5 {
		t.Fatalf("expected 5 statusbar items, got %d: %+v", len(flat.StatusBar), flat.StatusBar)
	}

	// Expected order:
	// 1. beta.left  (left, priority 100 — highest left)
	// 2. alpha.left (left, priority 10)
	// 3. alpha.right (right, priority 50, plugin "alpha" < "beta")
	// 4. beta.right  (right, priority 50, plugin "beta")
	// 5. gamma.right (right, priority 5)
	wantIDs := []string{"beta.left", "alpha.left", "alpha.right", "beta.right", "gamma.right"}
	for i, id := range wantIDs {
		if flat.StatusBar[i].ID != id {
			t.Errorf("StatusBar[%d]: got ID %q, want %q (full: %+v)",
				i, flat.StatusBar[i].ID, id, flat.StatusBar)
		}
	}
}

// ── Embedded-pg helper ─────────────────────────────────────────────────────

// bootTestDB boots an embedded Postgres, runs migrations, and returns a
// *store.DB. The test is skipped under -short.
func bootTestDB(t *testing.T) *store.DB {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping embedded-postgres integration test in -short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	t.Cleanup(cancel)

	port := freePort(t)
	dataDir := t.TempDir()
	cacheDir := filepath.Join(os.TempDir(), "opendray-pg-cache")
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		t.Fatalf("mkdir cache: %v", err)
	}

	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Username("opendray").
			Password("testpw").
			Database("opendray").
			Port(uint32(port)).
			DataPath(dataDir).
			RuntimePath(filepath.Join(cacheDir, "runtime")).
			BinariesPath(cacheDir).
			StartTimeout(2 * time.Minute),
	)
	if err := pg.Start(); err != nil {
		t.Fatalf("pg start: %v", err)
	}
	t.Cleanup(func() { _ = pg.Stop() })

	db, err := store.New(ctx, store.Config{
		Host:     "127.0.0.1",
		Port:     fmt.Sprintf("%d", port),
		User:     "opendray",
		Password: "testpw",
		DBName:   "opendray",
	})
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	t.Cleanup(db.Close)

	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("db.Migrate: %v", err)
	}
	return db
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("freePort: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// ── Test 9: Runtime.WithContributions integration ─────────────────────────

// TestRuntime_WithContributions constructs a Runtime with WithContributions,
// registers a plugin with Contributes, asserts the registry has the entry,
// then removes and asserts it is gone.
//
// Requires embedded-postgres; skipped under -short.
func TestRuntime_WithContributions(t *testing.T) {
	db := bootTestDB(t)

	reg := contributions.NewRegistry()
	hookBus := plugin.NewHookBus(nil)

	rt := plugin.NewRuntime(db, hookBus, "", nil, plugin.WithContributions(reg))

	p := plugin.Provider{
		Name:        "time-ninja",
		DisplayName: "Time Ninja",
		Version:     "1.0.0",
		Type:        "panel",
		Contributes: &plugin.ContributesV1{
			Commands: []plugin.CommandV1{
				{ID: "time.start", Title: "Start Pomodoro"},
			},
		},
	}

	if err := rt.Register(context.Background(), p); err != nil {
		t.Fatalf("Register: %v", err)
	}

	if !reg.Has("time-ninja") {
		t.Fatal("registry should have time-ninja after Register")
	}

	flat := reg.Flatten()
	if len(flat.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d: %+v", len(flat.Commands), flat.Commands)
	}
	if flat.Commands[0].ID != "time.start" {
		t.Errorf("command ID: got %q, want %q", flat.Commands[0].ID, "time.start")
	}
	if flat.Commands[0].PluginName != "time-ninja" {
		t.Errorf("PluginName: got %q, want %q", flat.Commands[0].PluginName, "time-ninja")
	}

	if err := rt.Remove(context.Background(), "time-ninja"); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	if reg.Has("time-ninja") {
		t.Fatal("registry should NOT have time-ninja after Remove")
	}
}

// ── Test 10: Nil contributions registry doesn't panic ─────────────────────

// TestRuntime_NilContributionsRegistry constructs a Runtime WITHOUT
// WithContributions and verifies Register + Remove do not panic.
//
// Requires embedded-postgres; skipped under -short.
func TestRuntime_NilContributionsRegistry(t *testing.T) {
	db := bootTestDB(t)

	hookBus := plugin.NewHookBus(nil)
	// No WithContributions option → contributionsReg is nil.
	rt := plugin.NewRuntime(db, hookBus, "", nil)

	p := plugin.Provider{
		Name:    "time-ninja",
		Version: "1.0.0",
		Type:    "panel",
		Contributes: &plugin.ContributesV1{
			Commands: []plugin.CommandV1{{ID: "time.start", Title: "Start"}},
		},
	}

	if err := rt.Register(context.Background(), p); err != nil {
		t.Fatalf("Register (nil contrib reg): %v", err)
	}

	if err := rt.Remove(context.Background(), "time-ninja"); err != nil {
		t.Fatalf("Remove (nil contrib reg): %v", err)
	}
}

// ─── M2 T3 — activityBar / views / panels slots ────────────────────────

// TestFlatten_ActivityBar_Views_Panels_Sorted sets two plugins that each
// contribute one of each new slot type, and asserts the Flatten output
// sorts by (PluginName, ID) deterministically.
func TestFlatten_ActivityBar_Views_Panels_Sorted(t *testing.T) {
	r := contributions.NewRegistry()

	r.Set("zebra", plugin.ContributesV1{
		ActivityBar: []plugin.ActivityBarItemV1{
			{ID: "zebra.board", Icon: "🦓", Title: "Zebra", ViewID: "zebra.view"},
		},
		Views: []plugin.ViewV1{
			{ID: "zebra.view", Title: "Zebra View", Render: "webview", Entry: "ui/index.html"},
		},
		Panels: []plugin.PanelV1{
			{ID: "zebra.console", Title: "Zebra Log", Position: "bottom", Render: "webview", Entry: "ui/log.html"},
		},
	})
	r.Set("antelope", plugin.ContributesV1{
		ActivityBar: []plugin.ActivityBarItemV1{
			{ID: "antelope.a", Icon: "🦌", Title: "Antelope", ViewID: "antelope.v"},
		},
		Views: []plugin.ViewV1{
			{ID: "antelope.v", Title: "Antelope View", Render: "declarative"},
		},
		// no panels
	})

	flat := r.Flatten()

	// activityBar: antelope first (alphabetic by PluginName)
	if len(flat.ActivityBar) != 2 {
		t.Fatalf("ActivityBar len = %d, want 2", len(flat.ActivityBar))
	}
	if flat.ActivityBar[0].PluginName != "antelope" || flat.ActivityBar[0].ID != "antelope.a" {
		t.Errorf("ActivityBar[0] = %+v", flat.ActivityBar[0])
	}
	if flat.ActivityBar[1].PluginName != "zebra" || flat.ActivityBar[1].Icon != "🦓" {
		t.Errorf("ActivityBar[1] = %+v", flat.ActivityBar[1])
	}

	// views: antelope first
	if len(flat.Views) != 2 {
		t.Fatalf("Views len = %d, want 2", len(flat.Views))
	}
	if flat.Views[0].PluginName != "antelope" {
		t.Errorf("Views[0].PluginName = %q, want antelope", flat.Views[0].PluginName)
	}
	if flat.Views[1].PluginName != "zebra" || flat.Views[1].Entry != "ui/index.html" {
		t.Errorf("Views[1] = %+v", flat.Views[1])
	}

	// panels: only zebra contributed one
	if len(flat.Panels) != 1 {
		t.Fatalf("Panels len = %d, want 1", len(flat.Panels))
	}
	if flat.Panels[0].PluginName != "zebra" || flat.Panels[0].ID != "zebra.console" {
		t.Errorf("Panels[0] = %+v", flat.Panels[0])
	}
}

// TestFlatten_EmptyWebviewSlots confirms plugins with only M1
// contributions (no activityBar/views/panels) still register and
// Flatten returns empty — not nil — slices so JSON always emits `[]`
// rather than `null`. Flutter treats null as invalid DTO.
func TestFlatten_EmptyWebviewSlots(t *testing.T) {
	r := contributions.NewRegistry()
	r.Set("m1-only", plugin.ContributesV1{
		Commands: []plugin.CommandV1{{ID: "m.hi", Title: "Hi"}},
	})

	flat := r.Flatten()
	if flat.ActivityBar == nil {
		t.Error("ActivityBar = nil, want empty slice")
	}
	if flat.Views == nil {
		t.Error("Views = nil, want empty slice")
	}
	if flat.Panels == nil {
		t.Error("Panels = nil, want empty slice")
	}
	if len(flat.ActivityBar) != 0 || len(flat.Views) != 0 || len(flat.Panels) != 0 {
		t.Errorf("unexpected non-empty slices: %+v", flat)
	}
}

// TestIsZero_OnlyWebviewFieldsPresent guards an edge case: a plugin
// that ONLY contributes a view (nothing else) must not be treated as
// zero by Set — it's a legitimate webview-only plugin.
func TestIsZero_OnlyWebviewFieldsPresent(t *testing.T) {
	r := contributions.NewRegistry()
	r.Set("webview-only", plugin.ContributesV1{
		Views: []plugin.ViewV1{{ID: "v", Title: "V"}},
	})
	if !r.Has("webview-only") {
		t.Fatal("Has(webview-only) = false — isZero is too aggressive on webview-only plugins")
	}
	if len(r.Flatten().Views) != 1 {
		t.Error("view not reflected in Flatten")
	}
}

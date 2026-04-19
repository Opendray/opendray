package plugin

// Tests for T12: verifies that loadIntoMemory (called by Register) pushes
// contributions to the ContribRegistry for both legacy and v1 manifests.
//
// These tests live in package plugin (not package plugin_test) so they can
// directly observe the Runtime fields after Register — but they go through
// the public Register API, not unexported internals.
//
// DB requirement: Register unconditionally calls store.DB, so these tests
// boot embedded Postgres. They are skipped under -short.

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/opendray/opendray/kernel/store"
)

// fakeContribRegistry is a test-double for ContribRegistry that records calls.
type fakeContribRegistry struct {
	mu   sync.Mutex
	sets []fakeSet
	rems []string
}

type fakeSet struct {
	name string
	c    ContributesV1
}

func (f *fakeContribRegistry) Set(pluginName string, c ContributesV1) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sets = append(f.sets, fakeSet{name: pluginName, c: c})
}

func (f *fakeContribRegistry) Remove(pluginName string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rems = append(f.rems, pluginName)
}

// countSetsFor returns how many Set calls were recorded for the given plugin name.
func (f *fakeContribRegistry) countSetsFor(name string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, s := range f.sets {
		if s.name == name {
			n++
		}
	}
	return n
}

// lastSetFor returns the last ContributesV1 passed to Set for the given name.
func (f *fakeContribRegistry) lastSetFor(name string) (ContributesV1, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := len(f.sets) - 1; i >= 0; i-- {
		if f.sets[i].name == name {
			return f.sets[i].c, true
		}
	}
	return ContributesV1{}, false
}

// setupTestDB boots embedded Postgres and returns a migrated *store.DB.
// The DB is cleaned up in t.Cleanup. Panics if Postgres can't start.
func setupTestDB(t *testing.T) *store.DB {
	t.Helper()

	port, err := freeTestPort()
	if err != nil {
		t.Fatalf("freeTestPort: %v", err)
	}

	dataDir := t.TempDir()
	cacheDir := filepath.Join(os.TempDir(), "opendray-pg-cache")
	_ = os.MkdirAll(cacheDir, 0o700)
	// Each test gets its own runtime directory to avoid stale-pwfile conflicts
	// when tests run concurrently or a previous run left state behind.
	runtimeDir := t.TempDir()

	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Username("opendray").
			Password("testpw").
			Database("opendray").
			Port(uint32(port)).
			DataPath(dataDir).
			RuntimePath(runtimeDir).
			BinariesPath(cacheDir).
			StartTimeout(2 * time.Minute),
	)
	if err := pg.Start(); err != nil {
		t.Fatalf("embedded postgres start: %v", err)
	}
	t.Cleanup(func() { _ = pg.Stop() })

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

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
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	t.Cleanup(db.Close)
	return db
}

func freeTestPort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// TestRuntime_loadIntoMemory_PushesLegacyContribs verifies that registering
// a legacy manifest (IsV1()==false) causes the compat synthesizer to push its
// contributions into the registry via Set. Because ContributesV1 has no
// AgentProviders field in M1, the synthesized Contributes will be empty for a
// legacy CLI plugin — but the registry must still receive a call (even if it
// is a no-op Set from isZero==true). This test asserts the plumbing is wired.
func TestRuntime_loadIntoMemory_PushesLegacyContribs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping embedded-postgres test in -short mode")
	}

	db := setupTestDB(t)
	fake := &fakeContribRegistry{}
	bus := NewHookBus(nil)
	rt := NewRuntime(db, bus, "", nil, WithContributions(fake))

	legacy := Provider{
		Name:        "test-legacy-agent",
		DisplayName: "Test Legacy Agent",
		Version:     "1.0.0",
		Type:        ProviderTypeCLI,
		CLI:         &CLISpec{Command: "echo"},
		Capabilities: Capabilities{
			SupportsStream: true,
		},
	}
	// Pre-condition: this is a legacy manifest.
	if legacy.IsV1() {
		t.Fatal("test setup: legacy.IsV1() must be false")
	}

	ctx := context.Background()
	if err := rt.Register(ctx, legacy); err != nil {
		t.Fatalf("Register: %v", err)
	}

	// The registry should have received at least one Set call for this plugin.
	// (Even with empty Contributes the synthesizer path must execute.)
	count := fake.countSetsFor(legacy.Name)
	if count == 0 {
		t.Errorf("ContribRegistry.Set never called for legacy plugin %q; want at least one call", legacy.Name)
	}
}

// TestRuntime_loadIntoMemory_PushesV1Contribs verifies that a v1 manifest with
// real Contributes.Commands gets its contributions forwarded to the registry
// verbatim (not synthesized). The fake recorder lets us inspect the exact
// ContributesV1 value that was pushed.
func TestRuntime_loadIntoMemory_PushesV1Contribs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping embedded-postgres test in -short mode")
	}

	db := setupTestDB(t)
	fake := &fakeContribRegistry{}
	bus := NewHookBus(nil)
	rt := NewRuntime(db, bus, "", nil, WithContributions(fake))

	contribs := &ContributesV1{
		Commands: []CommandV1{
			{ID: "foo.bar", Title: "Foo Bar"},
		},
	}
	v1p := Provider{
		Name:        "test-v1-plugin",
		DisplayName: "Test V1 Plugin",
		Version:     "1.0.0",
		Publisher:   "test-publisher",
		Engines:     &EnginesV1{Opendray: ">=0"},
		Form:        FormDeclarative,
		Contributes: contribs,
	}
	if !v1p.IsV1() {
		t.Fatal("test setup: v1p.IsV1() must be true")
	}

	ctx := context.Background()
	if err := rt.Register(ctx, v1p); err != nil {
		t.Fatalf("Register: %v", err)
	}

	got, ok := fake.lastSetFor(v1p.Name)
	if !ok {
		t.Fatalf("ContribRegistry.Set never called for v1 plugin %q", v1p.Name)
	}
	if len(got.Commands) != 1 {
		t.Fatalf("Contributes.Commands: got %d entries, want 1", len(got.Commands))
	}
	if got.Commands[0].ID != "foo.bar" {
		t.Errorf("Commands[0].ID = %q; want %q", got.Commands[0].ID, "foo.bar")
	}
}

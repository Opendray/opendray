// Package install — T6 test suite.
//
// The integration tests below mirror the bootstrapping pattern used in
// kernel/store/plugin_tables_test.go: spin up an embedded Postgres
// once per test, migrate, and tear it down via t.Cleanup. Expensive
// tests are skipped under -short. Pure-logic tests (ParseSource,
// SHA256CanonicalManifest_Stable) always run.
package install

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/opendray/opendray/kernel/store"
	"github.com/opendray/opendray/plugin"
	"github.com/opendray/opendray/plugin/bridge"
)

// ─── helpers ────────────────────────────────────────────────────────────────

// freePort finds a free TCP port on localhost.
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("free port: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// bootDB brings up an embedded Postgres and runs every migration so the
// installer tests have plugins/plugin_consents/plugin_audit in place.
func bootDB(t *testing.T) *store.DB {
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
		t.Fatalf("Migrate: %v", err)
	}
	return db
}

// writeValidBundle creates a fresh v1 plugin bundle directory under base.
// Returns the bundle root path.
func writeValidBundle(t *testing.T, base, name, version string) string {
	t.Helper()
	dir := filepath.Join(base, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir bundle: %v", err)
	}
	m := map[string]any{
		"name":        name,
		"version":     version,
		"publisher":   "opendray-examples",
		"displayName": name,
		"description": "Test plugin",
		"type":        "panel",
		"form":        "declarative",
		"engines":     map[string]string{"opendray": "^1.0.0"},
		"permissions": map[string]any{},
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# "+name+"\n"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	return dir
}

// writeBadBundle writes a manifest missing a required v1 field (publisher).
// The manifest does set engines.opendray so IsV1() fires, and ValidateV1
// complains about the missing publisher.
func writeBadBundle(t *testing.T, base, name string) string {
	t.Helper()
	dir := filepath.Join(base, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir bad bundle: %v", err)
	}
	// Construct raw JSON: engines.opendray set, but NO publisher. IsV1()
	// requires both → manifest looks v1-ish to the validator dispatch but
	// will trip the publisher rule. We also keep it syntactically valid.
	raw := `{
		"name": "` + name + `",
		"version": "1.0.0",
		"engines": {"opendray": "^1.0.0"},
		"publisher": "x"
	}`
	// Intentionally strip publisher line: we want it to fail validation for
	// a missing publisher. Easiest: emit the manifest with publisher=="" via
	// raw string that doesn't include that field. But then IsV1()==false and
	// ValidateV1 short-circuits. To force v1 path failure we use a publisher
	// that fails the regex (leading dash).
	raw = `{
		"name": "` + name + `",
		"version": "1.0.0",
		"engines": {"opendray": "^1.0.0"},
		"publisher": "-bad-name"
	}`
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(raw), 0o644); err != nil {
		t.Fatalf("write bad manifest: %v", err)
	}
	return dir
}

// writeLegacyBundle writes a manifest with no publisher/engines (legacy shape).
func writeLegacyBundle(t *testing.T, base, name string) string {
	t.Helper()
	dir := filepath.Join(base, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir legacy bundle: %v", err)
	}
	m := map[string]any{
		"name":    name,
		"version": "1.0.0",
		"type":    "panel",
	}
	data, _ := json.MarshalIndent(m, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0o644); err != nil {
		t.Fatalf("write legacy manifest: %v", err)
	}
	return dir
}

// mustRuntime builds a minimal plugin.Runtime wired to db.
func mustRuntime(db *store.DB) *plugin.Runtime {
	hooks := plugin.NewHookBus(slog.Default())
	return plugin.NewRuntime(db, hooks, "", slog.Default())
}

// ─── TestParseSource ────────────────────────────────────────────────────────

func TestParseSource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string // concrete type name, or "" for error
		wantErr bool
	}{
		{"localScheme", "local:/abs/foo", "LocalSource", false},
		{"bareAbsolute", "/abs/foo", "LocalSource", false},
		{"https", "https://example.com/plugin.zip", "HTTPSSource", false},
		{"marketplace", "marketplace://publisher.plugin@1.0.0", "MarketplaceSource", false},
		{"garbageScheme", "garbage://foo", "", true},
		{"empty", "", "", true},
		{"relativePath", "foo/bar", "", true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseSource(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParseSource(%q) want error, got %T", tc.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseSource(%q) err: %v", tc.input, err)
			}
			typeName := ""
			switch got.(type) {
			case LocalSource, *LocalSource:
				typeName = "LocalSource"
			case HTTPSSource, *HTTPSSource:
				typeName = "HTTPSSource"
			case MarketplaceSource, *MarketplaceSource:
				typeName = "MarketplaceSource"
			default:
				typeName = fmt.Sprintf("%T", got)
			}
			if typeName != tc.want {
				t.Errorf("ParseSource(%q): got type %s, want %s", tc.input, typeName, tc.want)
			}
		})
	}
}

// TestHTTPS_Marketplace_ReturnErrNotImplemented ensures the M4-bound
// sources fail loudly rather than silently.
func TestHTTPS_Marketplace_ReturnErrNotImplemented(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	https := HTTPSSource{URL: "https://example.com/foo.zip"}
	if _, _, err := https.Fetch(ctx); !errors.Is(err, ErrNotImplemented) {
		t.Errorf("HTTPSSource.Fetch: want ErrNotImplemented, got %v", err)
	}

	mp := MarketplaceSource{Raw: "marketplace://foo"}
	if _, _, err := mp.Fetch(ctx); !errors.Is(err, ErrNotImplemented) {
		t.Errorf("MarketplaceSource.Fetch: want ErrNotImplemented, got %v", err)
	}
}

// TestSource_Describe sanity-checks the human-readable labels used by the
// audit path (never log raw paths, but Describe is the input to the hash).
func TestSource_Describe(t *testing.T) {
	t.Parallel()
	cases := []struct {
		src  Source
		want string
	}{
		{LocalSource{Path: "/tmp/foo"}, "local:/tmp/foo"},
		{HTTPSSource{URL: "https://x/y"}, "https://x/y"},
		{MarketplaceSource{Raw: "marketplace://z"}, "marketplace://z"},
	}
	for _, c := range cases {
		if got := c.src.Describe(); got != c.want {
			t.Errorf("Describe: got %q want %q", got, c.want)
		}
	}
}

// TestLocalSource_Fetch_ErrorPaths covers the bad-input branches: empty
// path, non-existent path, and regular-file (not a directory).
func TestLocalSource_Fetch_ErrorPaths(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	if _, _, err := (LocalSource{}).Fetch(ctx); err == nil {
		t.Error("empty Path: want error, got nil")
	}
	if _, _, err := (LocalSource{Path: "/no/such/path/abc123"}).Fetch(ctx); err == nil {
		t.Error("non-existent path: want error, got nil")
	}
	// Regular file (not dir).
	f := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := (LocalSource{Path: f}).Fetch(ctx); err == nil {
		t.Error("regular file path: want error, got nil")
	}
}

// TestLocalSource_Fetch_HappyPath verifies a successful copy-into-staging,
// including a nested directory so copyTree's dir branch is exercised.
func TestLocalSource_Fetch_HappyPath(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	src := writeValidBundle(t, t.TempDir(), "fetch-plugin", "1.0.0")

	// Add a nested file so copyTree walks into a subdirectory.
	if err := os.MkdirAll(filepath.Join(src, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "sub", "nested.txt"), []byte("n"), 0o644); err != nil {
		t.Fatal(err)
	}

	path, cleanup, err := LocalSource{Path: src}.Fetch(ctx)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	t.Cleanup(cleanup)
	if _, err := os.Stat(filepath.Join(path, "manifest.json")); err != nil {
		t.Errorf("fetched bundle missing manifest.json: %v", err)
	}
	if _, err := os.Stat(filepath.Join(path, "sub", "nested.txt")); err != nil {
		t.Errorf("fetched bundle missing nested file: %v", err)
	}
}

// TestPendingStore_ExpiredKeyGone verifies take returns (nil,false) for
// an entry that was put with an already-expired ExpiresAt.
func TestPendingStore_ExpiredKeyGone(t *testing.T) {
	t.Parallel()
	now := time.Now()
	ps := newPendingStore(10*time.Minute, func() time.Time { return now.Add(time.Hour) })
	ps.put(&PendingInstall{Token: "tkn", ExpiresAt: now})
	if got, ok := ps.take("tkn"); ok || got != nil {
		t.Errorf("take expired: got %+v ok=%v", got, ok)
	}
}

// TestPendingStore_TakeMissing covers the not-found branch.
func TestPendingStore_TakeMissing(t *testing.T) {
	t.Parallel()
	ps := newPendingStore(10*time.Minute, nil)
	if _, ok := ps.take("nope"); ok {
		t.Error("take missing: want ok=false")
	}
}

// TestPendingStore_NewWithNilNow verifies the default now fallback.
func TestPendingStore_NewWithNilNow(t *testing.T) {
	t.Parallel()
	ps := newPendingStore(time.Minute, nil)
	if ps.now == nil {
		t.Error("now fallback not applied")
	}
}

// ─── TestSHA256CanonicalManifest_Stable ─────────────────────────────────────

func TestSHA256CanonicalManifest_Stable(t *testing.T) {
	t.Parallel()

	jsonA := `{"name":"foo","version":"1.0.0","publisher":"p","engines":{"opendray":"^1.0.0"}}`
	jsonB := `{"engines":{"opendray":"^1.0.0"},"publisher":"p","version":"1.0.0","name":"foo"}`

	var pa, pb plugin.Provider
	if err := json.Unmarshal([]byte(jsonA), &pa); err != nil {
		t.Fatalf("unmarshal A: %v", err)
	}
	if err := json.Unmarshal([]byte(jsonB), &pb); err != nil {
		t.Fatalf("unmarshal B: %v", err)
	}

	hA, err := SHA256CanonicalManifest(pa)
	if err != nil {
		t.Fatalf("hash A: %v", err)
	}
	hB, err := SHA256CanonicalManifest(pb)
	if err != nil {
		t.Fatalf("hash B: %v", err)
	}
	if hA != hB {
		t.Errorf("canonical hash mismatch: A=%s B=%s", hA, hB)
	}
	if len(hA) != 64 {
		t.Errorf("hash length: got %d want 64", len(hA))
	}
}

// TestSHA256File verifies the file-hash helper.
func TestSHA256File(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	p := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(p, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	h, err := SHA256File(p)
	if err != nil {
		t.Fatalf("SHA256File: %v", err)
	}
	// Known sha256 of "hello"
	want := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if h != want {
		t.Errorf("SHA256File: got %s want %s", h, want)
	}
}

// TestSHA256File_Missing covers the open-error branch.
func TestSHA256File_Missing(t *testing.T) {
	t.Parallel()
	if _, err := SHA256File("/no/such/file/xyz.txt"); err == nil {
		t.Error("want open error for missing file, got nil")
	}
}

// TestSHA256CanonicalManifest_WithContributions verifies nested map/array
// canonicalisation: two manifests with the same contributions in different
// key order must hash identically.
func TestSHA256CanonicalManifest_WithContributions(t *testing.T) {
	t.Parallel()

	// Same payload, different key ordering at every level. We round-trip
	// through plugin.Provider so json tags are honoured.
	jsonA := `{
		"name":"t","version":"1.0.0","publisher":"p","engines":{"opendray":"^1.0.0"},
		"contributes":{"commands":[{"id":"a","title":"A","run":{"kind":"notify","message":"hi"}}]}
	}`
	jsonB := `{
		"version":"1.0.0","name":"t","engines":{"opendray":"^1.0.0"},"publisher":"p",
		"contributes":{"commands":[{"title":"A","run":{"message":"hi","kind":"notify"},"id":"a"}]}
	}`
	var pa, pb plugin.Provider
	if err := json.Unmarshal([]byte(jsonA), &pa); err != nil {
		t.Fatalf("A: %v", err)
	}
	if err := json.Unmarshal([]byte(jsonB), &pb); err != nil {
		t.Fatalf("B: %v", err)
	}
	hA, err := SHA256CanonicalManifest(pa)
	if err != nil {
		t.Fatal(err)
	}
	hB, err := SHA256CanonicalManifest(pb)
	if err != nil {
		t.Fatal(err)
	}
	if hA != hB {
		t.Errorf("nested canonical hash mismatch: A=%s B=%s", hA, hB)
	}
}

// ─── Installer scenarios ───────────────────────────────────────────────────

func TestInstaller_HappyPath(t *testing.T) {
	db := bootDB(t)
	ctx := context.Background()

	src := writeValidBundle(t, t.TempDir(), "happy-plugin", "1.0.0")
	dataDir := t.TempDir()

	rt := mustRuntime(db)
	gate := bridge.NewGate(nil, nil, slog.Default())
	inst := NewInstaller(dataDir, db, rt, gate, slog.Default())
	t.Cleanup(inst.Stop)
	inst.AllowLocal = true

	pend, err := inst.Stage(ctx, LocalSource{Path: src})
	if err != nil {
		t.Fatalf("Stage: %v", err)
	}
	if pend.Name != "happy-plugin" || pend.Version != "1.0.0" {
		t.Errorf("pending: name=%q version=%q", pend.Name, pend.Version)
	}
	if len(pend.Token) != 64 {
		t.Errorf("token length: got %d want 64", len(pend.Token))
	}
	if len(pend.ManifestHash) != 64 {
		t.Errorf("manifest hash length: got %d want 64", len(pend.ManifestHash))
	}

	if err := inst.Confirm(ctx, pend.Token); err != nil {
		t.Fatalf("Confirm: %v", err)
	}

	// Final dir exists.
	finalPath := filepath.Join(dataDir, "happy-plugin", "1.0.0")
	if _, err := os.Stat(finalPath); err != nil {
		t.Errorf("final dir missing: %v", err)
	}

	// plugins row written.
	var pluginCount int
	if err := db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM plugins WHERE name = $1`, "happy-plugin").
		Scan(&pluginCount); err != nil {
		t.Fatalf("plugins count: %v", err)
	}
	if pluginCount != 1 {
		t.Errorf("plugins row count: got %d want 1", pluginCount)
	}

	// plugin_consents row written.
	cons, err := db.GetConsent(ctx, "happy-plugin")
	if err != nil {
		t.Fatalf("GetConsent: %v", err)
	}
	if cons.ManifestHash != pend.ManifestHash {
		t.Errorf("consent hash: got %s want %s", cons.ManifestHash, pend.ManifestHash)
	}

	// Runtime.Get returns the provider.
	if _, ok := rt.Get("happy-plugin"); !ok {
		t.Errorf("Runtime.Get: provider not registered")
	}

	// Audit row exists.
	entries, err := db.TailAudit(ctx, "happy-plugin", 10)
	if err != nil {
		t.Fatalf("TailAudit: %v", err)
	}
	sawConfirmOK := false
	for _, e := range entries {
		if e.Ns == "install" && e.Method == "confirm" && e.Result == "ok" {
			sawConfirmOK = true
		}
	}
	if !sawConfirmOK {
		t.Errorf("no install/confirm/ok audit row; entries=%+v", entries)
	}
}

func TestInstaller_InvalidManifestRejected(t *testing.T) {
	db := bootDB(t)
	ctx := context.Background()

	bad := writeBadBundle(t, t.TempDir(), "bad-plugin")
	dataDir := t.TempDir()

	inst := NewInstaller(dataDir, db, mustRuntime(db), bridge.NewGate(nil, nil, slog.Default()), slog.Default())
	t.Cleanup(inst.Stop)
	inst.AllowLocal = true

	_, err := inst.Stage(ctx, LocalSource{Path: bad})
	if !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("Stage: want ErrInvalidManifest, got %v", err)
	}

	// No lingering staged dirs inside dataDir.
	entries, _ := os.ReadDir(dataDir)
	for _, e := range entries {
		if filepath.HasPrefix(e.Name(), "staging-") {
			t.Errorf("staged dir leaked: %s", e.Name())
		}
	}

	// No plugins row.
	var cnt int
	_ = db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM plugins WHERE name=$1`, "bad-plugin").Scan(&cnt)
	if cnt != 0 {
		t.Errorf("plugins row leaked: count=%d", cnt)
	}
}

func TestInstaller_LegacyManifestRejected(t *testing.T) {
	db := bootDB(t)
	ctx := context.Background()

	legacy := writeLegacyBundle(t, t.TempDir(), "legacy-plugin")
	dataDir := t.TempDir()

	inst := NewInstaller(dataDir, db, mustRuntime(db), bridge.NewGate(nil, nil, slog.Default()), slog.Default())
	t.Cleanup(inst.Stop)
	inst.AllowLocal = true

	_, err := inst.Stage(ctx, LocalSource{Path: legacy})
	if !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("Stage: want ErrInvalidManifest for legacy manifest, got %v", err)
	}
}

func TestInstaller_ExpiredTokenRejected(t *testing.T) {
	db := bootDB(t)
	ctx := context.Background()

	src := writeValidBundle(t, t.TempDir(), "expire-plugin", "1.0.0")
	dataDir := t.TempDir()

	inst := NewInstallerWithTTL(dataDir, db, mustRuntime(db),
		bridge.NewGate(nil, nil, slog.Default()), slog.Default(),
		100*time.Millisecond, 50*time.Millisecond)
	t.Cleanup(inst.Stop)
	inst.AllowLocal = true

	pend, err := inst.Stage(ctx, LocalSource{Path: src})
	if err != nil {
		t.Fatalf("Stage: %v", err)
	}
	time.Sleep(250 * time.Millisecond)

	if err := inst.Confirm(ctx, pend.Token); !errors.Is(err, ErrTokenNotFound) {
		t.Fatalf("Confirm: want ErrTokenNotFound, got %v", err)
	}

	// Staged dir should be cleaned up by the reaper.
	if _, statErr := os.Stat(pend.StagedPath); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("staged dir still exists after expiry: %s", pend.StagedPath)
	}
}

// TestInstaller_OnContributionsChangedFires — T15.
// Confirms that Installer fires OnContributionsChanged exactly once per
// successful Confirm AND per successful Uninstall, in that order, so the
// gateway SSE hub can publish contributionsChanged to Flutter clients.
func TestInstaller_OnContributionsChangedFires(t *testing.T) {
	db := bootDB(t)
	ctx := context.Background()

	src := writeValidBundle(t, t.TempDir(), "ccplug", "1.0.0")
	dataDir := t.TempDir()

	rt := mustRuntime(db)
	inst := NewInstaller(dataDir, db, rt, bridge.NewGate(nil, nil, slog.Default()), slog.Default())
	t.Cleanup(inst.Stop)
	inst.AllowLocal = true

	var (
		mu     sync.Mutex
		events []string
	)
	inst.OnContributionsChanged = func() {
		mu.Lock()
		events = append(events, fmt.Sprintf("t=%d", len(events)))
		mu.Unlock()
	}

	pend, err := inst.Stage(ctx, LocalSource{Path: src})
	if err != nil {
		t.Fatalf("Stage: %v", err)
	}
	if err := inst.Confirm(ctx, pend.Token); err != nil {
		t.Fatalf("Confirm: %v", err)
	}

	mu.Lock()
	if got := len(events); got != 1 {
		mu.Unlock()
		t.Fatalf("OnContributionsChanged after Confirm: got %d, want 1", got)
	}
	mu.Unlock()

	if err := inst.Uninstall(ctx, "ccplug"); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}

	mu.Lock()
	afterUninstall := len(events)
	mu.Unlock()
	if afterUninstall != 2 {
		t.Fatalf("OnContributionsChanged after Uninstall: got %d, want 2", afterUninstall)
	}

	// Second (idempotent) uninstall still fires — current contract is
	// "fire on every successful Uninstall call"; the gateway publishing
	// an extra no-op event is cheap (Flutter refetch is idempotent too).
	if err := inst.Uninstall(ctx, "ccplug"); err != nil {
		t.Fatalf("Uninstall (second): %v", err)
	}
	mu.Lock()
	afterSecond := len(events)
	mu.Unlock()
	if afterSecond != 3 {
		t.Fatalf("OnContributionsChanged after idempotent Uninstall: got %d, want 3", afterSecond)
	}
}

func TestInstaller_UninstallRemovesAllTraces(t *testing.T) {
	db := bootDB(t)
	ctx := context.Background()

	src := writeValidBundle(t, t.TempDir(), "un-plugin", "1.0.0")
	dataDir := t.TempDir()

	rt := mustRuntime(db)
	inst := NewInstaller(dataDir, db, rt, bridge.NewGate(nil, nil, slog.Default()), slog.Default())
	t.Cleanup(inst.Stop)
	inst.AllowLocal = true

	pend, err := inst.Stage(ctx, LocalSource{Path: src})
	if err != nil {
		t.Fatalf("Stage: %v", err)
	}
	if err := inst.Confirm(ctx, pend.Token); err != nil {
		t.Fatalf("Confirm: %v", err)
	}

	if err := inst.Uninstall(ctx, "un-plugin"); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}

	if _, ok := rt.Get("un-plugin"); ok {
		t.Errorf("Runtime still has provider after Uninstall")
	}

	var cnt int
	_ = db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM plugins WHERE name=$1`, "un-plugin").Scan(&cnt)
	if cnt != 0 {
		t.Errorf("plugins row remains after Uninstall: %d", cnt)
	}
	_ = db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM plugin_consents WHERE plugin_name=$1`, "un-plugin").Scan(&cnt)
	if cnt != 0 {
		t.Errorf("plugin_consents row remains after Uninstall: %d", cnt)
	}

	final := filepath.Join(dataDir, "un-plugin", "1.0.0")
	if _, err := os.Stat(final); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("final dir still exists after Uninstall")
	}

	// Audit rows preserved (historical).
	entries, _ := db.TailAudit(ctx, "un-plugin", 10)
	if len(entries) == 0 {
		t.Errorf("audit trail empty after Uninstall; expected historical rows")
	}

	// Idempotent second call.
	if err := inst.Uninstall(ctx, "un-plugin"); err != nil {
		t.Errorf("second Uninstall not idempotent: %v", err)
	}
}

func TestInstaller_ConcurrentStage(t *testing.T) {
	db := bootDB(t)
	ctx := context.Background()

	inst := NewInstaller(t.TempDir(), db, mustRuntime(db), bridge.NewGate(nil, nil, slog.Default()), slog.Default())
	t.Cleanup(inst.Stop)
	inst.AllowLocal = true

	// Each goroutine stages from its own temp dir so I/O does not collide.
	const N = 20
	tokens := make([]string, N)
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		i := i
		go func() {
			defer wg.Done()
			src := writeValidBundle(t, t.TempDir(), fmt.Sprintf("p%d", i), "1.0.0")
			p, err := inst.Stage(ctx, LocalSource{Path: src})
			if err != nil {
				t.Errorf("Stage[%d]: %v", i, err)
				return
			}
			tokens[i] = p.Token
		}()
	}
	wg.Wait()

	seen := make(map[string]bool)
	for i, tk := range tokens {
		if tk == "" {
			t.Errorf("token[%d] empty", i)
			continue
		}
		if seen[tk] {
			t.Errorf("duplicate token at %d: %s", i, tk)
		}
		seen[tk] = true
	}
}

func TestInstaller_ConfirmAtomicity(t *testing.T) {
	db := bootDB(t)
	ctx := context.Background()

	src := writeValidBundle(t, t.TempDir(), "atom-plugin", "1.0.0")
	dataDir := t.TempDir()

	// Make `${DataDir}/atom-plugin` a REGULAR FILE. Confirm will then
	// fail when it tries to MkdirAll the parent of `${finalDir}` (i.e.
	// this same path), because you cannot mkdir a file path. That
	// failure happens after the DB rows are written inside the tx but
	// before commit, so rollback must wipe them.
	blocker := filepath.Join(dataDir, "atom-plugin")
	if err := os.WriteFile(blocker, []byte("squatter"), 0o644); err != nil {
		t.Fatal(err)
	}

	inst := NewInstaller(dataDir, db, mustRuntime(db), bridge.NewGate(nil, nil, slog.Default()), slog.Default())
	t.Cleanup(inst.Stop)
	inst.AllowLocal = true

	pend, err := inst.Stage(ctx, LocalSource{Path: src})
	if err != nil {
		t.Fatalf("Stage: %v", err)
	}

	err = inst.Confirm(ctx, pend.Token)
	if err == nil {
		t.Fatalf("Confirm: expected failure, got nil")
	}

	// No partial state.
	var cnt int
	_ = db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM plugins WHERE name=$1`, "atom-plugin").Scan(&cnt)
	if cnt != 0 {
		t.Errorf("plugins row written despite failed Confirm: %d", cnt)
	}
	_ = db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM plugin_consents WHERE plugin_name=$1`, "atom-plugin").Scan(&cnt)
	if cnt != 0 {
		t.Errorf("plugin_consents row written despite failed Confirm: %d", cnt)
	}

	// Staged dir still present for retry.
	if _, statErr := os.Stat(pend.StagedPath); statErr != nil {
		t.Errorf("staged dir removed on failed Confirm (should be kept for retry): %v", statErr)
	}
}

// TestInstaller_ConfirmReplacesExisting verifies that re-confirming a
// plugin that is already installed moves the previous install to .trash
// and installs the new one atomically.
func TestInstaller_ConfirmReplacesExisting(t *testing.T) {
	db := bootDB(t)
	ctx := context.Background()

	dataDir := t.TempDir()
	rt := mustRuntime(db)
	inst := NewInstaller(dataDir, db, rt, bridge.NewGate(nil, nil, slog.Default()), slog.Default())
	t.Cleanup(inst.Stop)
	inst.AllowLocal = true

	// First install.
	src1 := writeValidBundle(t, t.TempDir(), "replace-plugin", "1.0.0")
	p1, err := inst.Stage(ctx, LocalSource{Path: src1})
	if err != nil {
		t.Fatalf("Stage1: %v", err)
	}
	if err := inst.Confirm(ctx, p1.Token); err != nil {
		t.Fatalf("Confirm1: %v", err)
	}

	// Second install (same name+version) should replace, via .trash.
	src2 := writeValidBundle(t, t.TempDir(), "replace-plugin", "1.0.0")
	p2, err := inst.Stage(ctx, LocalSource{Path: src2})
	if err != nil {
		t.Fatalf("Stage2: %v", err)
	}
	if err := inst.Confirm(ctx, p2.Token); err != nil {
		t.Fatalf("Confirm2: %v", err)
	}

	// .trash dir contains the old install.
	trashDir := filepath.Join(dataDir, ".trash")
	entries, err := os.ReadDir(trashDir)
	if err != nil || len(entries) == 0 {
		t.Errorf(".trash dir missing or empty after replace: err=%v entries=%d", err, len(entries))
	}
}

// TestInstaller_ConfirmStagedDirDeletedBetween covers the
// readStagedManifestJSON error branch: if somebody removes the staged
// dir between Stage and Confirm, Confirm returns a read error AND
// re-registers the pending entry so the caller can retry once they fix
// the filesystem.
func TestInstaller_ConfirmStagedDirDeletedBetween(t *testing.T) {
	db := bootDB(t)
	ctx := context.Background()

	src := writeValidBundle(t, t.TempDir(), "gone-plugin", "1.0.0")
	dataDir := t.TempDir()

	inst := NewInstaller(dataDir, db, mustRuntime(db), bridge.NewGate(nil, nil, slog.Default()), slog.Default())
	t.Cleanup(inst.Stop)
	inst.AllowLocal = true

	pend, err := inst.Stage(ctx, LocalSource{Path: src})
	if err != nil {
		t.Fatalf("Stage: %v", err)
	}
	// Simulate external interference: delete the staged bundle.
	if err := os.RemoveAll(pend.StagedPath); err != nil {
		t.Fatal(err)
	}
	if err := inst.Confirm(ctx, pend.Token); err == nil {
		t.Fatal("Confirm: want error when staged dir is missing, got nil")
	}
	// No DB rows should have been written.
	var cnt int
	_ = db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM plugins WHERE name=$1`, "gone-plugin").Scan(&cnt)
	if cnt != 0 {
		t.Errorf("plugins row leaked: %d", cnt)
	}
}

// TestInstaller_ConfirmTokenGoneAfterFirstConfirm covers the second-use
// rejection: tokens are single-use.
func TestInstaller_ConfirmTokenGoneAfterFirstConfirm(t *testing.T) {
	db := bootDB(t)
	ctx := context.Background()

	src := writeValidBundle(t, t.TempDir(), "once-plugin", "1.0.0")
	dataDir := t.TempDir()

	inst := NewInstaller(dataDir, db, mustRuntime(db), bridge.NewGate(nil, nil, slog.Default()), slog.Default())
	t.Cleanup(inst.Stop)
	inst.AllowLocal = true

	pend, err := inst.Stage(ctx, LocalSource{Path: src})
	if err != nil {
		t.Fatalf("Stage: %v", err)
	}
	if err := inst.Confirm(ctx, pend.Token); err != nil {
		t.Fatalf("Confirm1: %v", err)
	}
	if err := inst.Confirm(ctx, pend.Token); !errors.Is(err, ErrTokenNotFound) {
		t.Errorf("Confirm2: want ErrTokenNotFound, got %v", err)
	}
}

func TestInstaller_JanitorReapsExpired(t *testing.T) {
	db := bootDB(t)
	ctx := context.Background()

	dataDir := t.TempDir()
	// Short TTL and short reap tick so the janitor fires during the test.
	inst := NewInstallerWithTTL(dataDir, db, mustRuntime(db),
		bridge.NewGate(nil, nil, slog.Default()), slog.Default(),
		80*time.Millisecond, 30*time.Millisecond)
	t.Cleanup(inst.Stop)
	inst.AllowLocal = true

	var stagedPaths []string
	for i := 0; i < 3; i++ {
		src := writeValidBundle(t, t.TempDir(), fmt.Sprintf("j%d", i), "1.0.0")
		p, err := inst.Stage(ctx, LocalSource{Path: src})
		if err != nil {
			t.Fatalf("Stage[%d]: %v", i, err)
		}
		stagedPaths = append(stagedPaths, p.StagedPath)
	}

	// Wait for >TTL + at least one reap tick.
	time.Sleep(300 * time.Millisecond)

	sort.Strings(stagedPaths)
	for _, p := range stagedPaths {
		if _, err := os.Stat(p); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("staged dir not reaped: %s (err=%v)", p, err)
		}
	}

	if n := inst.pendingCount(); n != 0 {
		t.Errorf("pending entries remaining: %d", n)
	}
}

// ─── T25: AllowLocal gate ─────────────────────────────────────────────────

// TestInstaller_LocalSourceGatedByConfig verifies that an Installer with
// AllowLocal=false refuses to Stage a local-scheme source with
// ErrLocalDisabled, and that setting AllowLocal=true allows it to proceed
// past the gate (it may still fail for other reasons — we only assert that
// ErrLocalDisabled is NOT returned).
func TestInstaller_LocalSourceGatedByConfig(t *testing.T) {
	db := bootDB(t)
	ctx := context.Background()

	src := writeValidBundle(t, t.TempDir(), "gate-plugin", "1.0.0")
	dataDir := t.TempDir()

	rt := mustRuntime(db)
	gate := bridge.NewGate(nil, nil, slog.Default())

	// ── Case 1: AllowLocal=false → ErrLocalDisabled ───────────────────────
	instDeny := NewInstaller(dataDir, db, rt, gate, slog.Default())
	instDeny.AllowLocal = false
	t.Cleanup(instDeny.Stop)

	_, err := instDeny.Stage(ctx, LocalSource{Path: src})
	if !errors.Is(err, ErrLocalDisabled) {
		t.Errorf("Stage with AllowLocal=false: want ErrLocalDisabled, got %v", err)
	}

	// No staging artefacts should have been created.
	entries, _ := os.ReadDir(dataDir)
	for _, e := range entries {
		if len(e.Name()) > 8 && e.Name()[:8] == "staging-" {
			t.Errorf("staging dir leaked when AllowLocal=false: %s", e.Name())
		}
	}

	// ── Case 2: AllowLocal=true → gate passes, Stage succeeds ────────────
	instAllow := NewInstaller(dataDir, db, rt, gate, slog.Default())
	instAllow.AllowLocal = true
	t.Cleanup(instAllow.Stop)

	pend, err := instAllow.Stage(ctx, LocalSource{Path: src})
	if errors.Is(err, ErrLocalDisabled) {
		t.Errorf("Stage with AllowLocal=true: got unexpected ErrLocalDisabled")
	}
	if err != nil {
		t.Fatalf("Stage with AllowLocal=true: unexpected error %v", err)
	}
	if pend == nil || pend.Name != "gate-plugin" {
		t.Errorf("Stage with AllowLocal=true: unexpected pending %+v", pend)
	}
}

// TestInstaller_LocalSourceGatedByConfig_HTTPSUnaffected verifies that the
// AllowLocal gate does NOT affect non-local sources: an HTTPSSource with
// AllowLocal=false returns ErrNotImplemented (not ErrLocalDisabled).
func TestInstaller_LocalSourceGatedByConfig_HTTPSUnaffected(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// We only need an Installer with no DB for this; the gate fires before
	// any DB or file I/O when the source is local. For HTTPS the gate is
	// skipped entirely, so ErrNotImplemented comes from the source itself.
	inst := &Installer{
		DataDir:    t.TempDir(),
		AllowLocal: false,
		pending:    newPendingStore(10*time.Minute, nil),
		Log:        slog.Default(),
	}

	_, err := inst.Stage(ctx, HTTPSSource{URL: "https://example.com/x.zip"})
	if errors.Is(err, ErrLocalDisabled) {
		t.Errorf("HTTPS source should not be blocked by AllowLocal gate; got ErrLocalDisabled")
	}
	if !errors.Is(err, ErrNotImplemented) {
		t.Errorf("HTTPS source with AllowLocal=false: want ErrNotImplemented, got %v", err)
	}
}

// ─── compile-only guard: ensures I/O helpers used above keep referenced ───
var _ = io.Copy

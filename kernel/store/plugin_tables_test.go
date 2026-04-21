package store

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// TestMigrate_PluginTables boots a clean embedded Postgres, runs every
// migration in Migrate() in order, and asserts the four plugin_* tables
// added in M1 (plugin_consents, plugin_kv, plugin_secret, plugin_audit)
// exist with the columns their callers rely on. The test also re-runs
// Migrate() to confirm idempotency — required because OpenDray runs
// migrations at every boot.
//
// Boot cost is ~10s after the binary is cached under ~/.cache; first
// run on a fresh machine downloads ~100MB. Skipped under `go test -short`.
func TestMigrate_PluginTables(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping embedded-postgres integration test in -short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	port, err := freePort()
	if err != nil {
		t.Fatalf("free port: %v", err)
	}

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

	db, err := New(ctx, Config{
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
		t.Fatalf("first Migrate: %v", err)
	}
	// Idempotency: second run must succeed without error.
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("second Migrate (idempotency): %v", err)
	}

	got := listTables(t, ctx, db)
	for _, want := range []string{"plugin_consents", "plugin_kv", "plugin_secret", "plugin_audit"} {
		if !contains(got, want) {
			t.Errorf("table %q missing; have %v", want, got)
		}
	}

	wantCols := map[string][]string{
		"plugin_consents": {"plugin_name", "manifest_hash", "perms_json", "granted_at", "updated_at"},
		"plugin_kv":       {"plugin_name", "key", "value", "size_bytes", "updated_at"},
		"plugin_secret":   {"plugin_name", "key", "ciphertext", "updated_at"},
		"plugin_audit":    {"id", "ts", "plugin_name", "ns", "method", "caps", "result", "duration_ms", "args_hash", "message"},
	}
	for table, cols := range wantCols {
		have := listColumns(t, ctx, db, table)
		for _, c := range cols {
			if !contains(have, c) {
				t.Errorf("table %s: column %q missing; have %v", table, c, have)
			}
		}
	}
}

func TestMigrate_PluginConsentsFK(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping embedded-postgres integration test in -short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	port, err := freePort()
	if err != nil {
		t.Fatalf("free port: %v", err)
	}
	dataDir := t.TempDir()
	cacheDir := filepath.Join(os.TempDir(), "opendray-pg-cache")
	_ = os.MkdirAll(cacheDir, 0o700)

	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Username("opendray").Password("testpw").Database("opendray").
			Port(uint32(port)).DataPath(dataDir).
			RuntimePath(filepath.Join(cacheDir, "runtime")).
			BinariesPath(cacheDir).
			StartTimeout(2 * time.Minute),
	)
	if err := pg.Start(); err != nil {
		t.Fatalf("pg start: %v", err)
	}
	t.Cleanup(func() { _ = pg.Stop() })

	db, err := New(ctx, Config{
		Host: "127.0.0.1", Port: fmt.Sprintf("%d", port),
		User: "opendray", Password: "testpw", DBName: "opendray",
	})
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	t.Cleanup(db.Close)
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// Inserting a consent row for a plugin that doesn't exist must fail
	// because of the FK on plugin_consents.plugin_name → plugins(name).
	_, err = db.Pool.Exec(ctx,
		`INSERT INTO plugin_consents (plugin_name, manifest_hash, perms_json)
		 VALUES ($1, $2, $3::jsonb)`,
		"ghost", "deadbeef", "{}")
	if err == nil {
		t.Fatal("FK violation expected for unknown plugin_name, got nil error")
	}
}

// ─── helpers ────────────────────────────────────────────────────

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func listTables(t *testing.T, ctx context.Context, db *DB) []string {
	t.Helper()
	rows, err := db.Pool.Query(ctx, `
		SELECT table_name FROM information_schema.tables
		WHERE table_schema = 'public'
	`)
	if err != nil {
		t.Fatalf("list tables: %v", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatalf("scan: %v", err)
		}
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

func listColumns(t *testing.T, ctx context.Context, db *DB, table string) []string {
	t.Helper()
	rows, err := db.Pool.Query(ctx, `
		SELECT column_name FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1
	`, table)
	if err != nil {
		t.Fatalf("list columns: %v", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatalf("scan: %v", err)
		}
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

func contains(hay []string, needle string) bool {
	for _, s := range hay {
		if s == needle {
			return true
		}
	}
	return false
}

// Compile-time check that sql is imported so the test binary links
// against the pgx stdlib driver we wire in via the blank import above
// — embedded-postgres needs it to talk to the child.
var _ = sql.Drivers

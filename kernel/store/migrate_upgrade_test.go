package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// TestMigrate_UpgradeFromPrePluginSchema — M5 D4 smoke test.
//
// The default Migrate() always runs on an empty DB in CI. Production
// installs evolve over time: a v0.9 deployment upgrades to v1 by
// re-running `opendray` with a new binary, and the migration chain
// must not drop pre-existing data.
//
// Flow:
//  1. Apply migrations 001–009 only (pre-plugin schema, matches a
//     M0 deployment before the plugin platform shipped).
//  2. Insert one sessions row + one mcp_servers row with non-default
//     values — this is the user's "legacy data" snapshot.
//  3. Run the full Migrate() — bridges forward to the current HEAD
//     (010–016 add the plugin platform tables).
//  4. Assert the seeded rows are still present verbatim. Any migration
//     that forgets a `WHERE plugin_name IS NOT NULL` or does a sloppy
//     `DROP TABLE sessions CASCADE` will flunk this test.
//  5. Insert plugin_consents + plugin_kv rows.
//  6. Re-run Migrate() a third time — verify both the legacy rows AND
//     the plugin rows survive the idempotent re-run. This is what
//     happens every time OpenDray boots on an already-migrated DB.
//
// ~15s on a cached embedded-postgres binary; skipped under -short.
func TestMigrate_UpgradeFromPrePluginSchema(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping embedded-postgres integration test in -short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	db := bootUpgradeTestPG(t, ctx)

	// ── Phase 1: apply the pre-plugin subset (001–009) ───────────────────────
	prePlugin := []string{
		"migrations/001_init.sql",
		"migrations/002_plugins_version.sql",
		"migrations/003_plugin_config.sql",
		"migrations/004_mcp_servers.sql",
		"migrations/005_claude_accounts.sql",
		"migrations/006_claude_accounts_local_backend.sql",
		"migrations/007_rollback_local_backend.sql",
		"migrations/008_llm_providers.sql",
		"migrations/009_admin_auth.sql",
	}
	if err := applyMigrations(ctx, db, prePlugin); err != nil {
		t.Fatalf("apply pre-plugin subset: %v", err)
	}

	// ── Phase 2: seed legacy data ────────────────────────────────────────────
	var legacySessionID, legacyMCPID string
	err := db.Pool.QueryRow(ctx, `
		INSERT INTO sessions (name, session_type, cwd, status, model)
		VALUES ('legacy-session', 'claude', '/home/legacy', 'stopped', 'sonnet-4')
		RETURNING id`,
	).Scan(&legacySessionID)
	if err != nil {
		t.Fatalf("seed sessions: %v", err)
	}

	// mcp_servers schema (from 004_mcp_servers.sql)
	err = db.Pool.QueryRow(ctx, `
		INSERT INTO mcp_servers (name, command, args, env, enabled)
		VALUES ('legacy-mcp', 'mcp-server', '[]'::jsonb, '{}'::jsonb, true)
		RETURNING id`,
	).Scan(&legacyMCPID)
	if err != nil {
		t.Fatalf("seed mcp_servers: %v", err)
	}

	// ── Phase 3: bridge to HEAD via full Migrate() ───────────────────────────
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("bridge-to-HEAD Migrate(): %v", err)
	}

	// ── Phase 4: verify legacy rows survived the upgrade ─────────────────────
	var sessionName, sessionStatus string
	err = db.Pool.QueryRow(ctx,
		`SELECT name, status FROM sessions WHERE id=$1`, legacySessionID,
	).Scan(&sessionName, &sessionStatus)
	if err != nil {
		t.Fatalf("post-upgrade sessions lookup: %v", err)
	}
	if sessionName != "legacy-session" || sessionStatus != "stopped" {
		t.Errorf("legacy session mutated: name=%q status=%q", sessionName, sessionStatus)
	}

	var mcpName string
	var mcpEnabled bool
	err = db.Pool.QueryRow(ctx,
		`SELECT name, enabled FROM mcp_servers WHERE id=$1`, legacyMCPID,
	).Scan(&mcpName, &mcpEnabled)
	if err != nil {
		t.Fatalf("post-upgrade mcp_servers lookup: %v", err)
	}
	if mcpName != "legacy-mcp" || !mcpEnabled {
		t.Errorf("legacy mcp mutated: name=%q enabled=%v", mcpName, mcpEnabled)
	}

	// ── Phase 5: seed plugin-era data on the fully-migrated schema ───────────
	// plugins row first (FK target for plugin_consents).
	_, err = db.Pool.Exec(ctx,
		`INSERT INTO plugins (name, version, enabled) VALUES ('d4test', '1.0.0', true)`,
	)
	if err != nil {
		t.Fatalf("seed plugins: %v", err)
	}
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO plugin_consents (plugin_name, manifest_hash, perms_json)
		VALUES ('d4test', 'sha256:dummyhash', '{"storage":true}'::jsonb)`,
	)
	if err != nil {
		t.Fatalf("seed plugin_consents: %v", err)
	}
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO plugin_kv (plugin_name, key, value)
		VALUES ('d4test', 'hello', '"world"'::jsonb)`,
	)
	if err != nil {
		t.Fatalf("seed plugin_kv: %v", err)
	}

	// ── Phase 6: re-run Migrate() (idempotency with data present) ────────────
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("idempotent Migrate() with data: %v", err)
	}

	// Legacy rows still intact.
	err = db.Pool.QueryRow(ctx,
		`SELECT name FROM sessions WHERE id=$1`, legacySessionID,
	).Scan(&sessionName)
	if err != nil {
		t.Errorf("legacy session lost after idempotent re-run: %v", err)
	}

	// Plugin-era rows still intact.
	var val []byte
	err = db.Pool.QueryRow(ctx, `
		SELECT value FROM plugin_kv WHERE plugin_name=$1 AND key=$2`,
		"d4test", "hello",
	).Scan(&val)
	if err != nil {
		t.Errorf("plugin_kv row lost after idempotent re-run: %v", err)
	}
	if string(val) != `"world"` {
		t.Errorf("plugin_kv value mutated: got %q", val)
	}
}

// ─── helpers ────────────────────────────────────────────────────────────────

// bootUpgradeTestPG starts a fresh embedded-postgres and returns a live
// *DB. Separate from the plugin_tables_test helper so these two tests
// can run concurrently without trampling each other's DataDir.
func bootUpgradeTestPG(t *testing.T, ctx context.Context) *DB {
	t.Helper()

	port, err := freePort()
	if err != nil {
		t.Fatalf("free port: %v", err)
	}

	dataDir := t.TempDir()
	cacheDir := filepath.Join(os.TempDir(), "opendray-pg-cache")
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		t.Fatalf("mkdir cache: %v", err)
	}
	// Unique runtime dir per test so parallel test binaries don't race
	// on the shared pwfile / postmaster.pid.
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
	return db
}

// applyMigrations runs the given subset of migration files against db.
// Mirrors DB.Migrate but exposes the file list so callers can stop at
// an arbitrary point (used to simulate a pre-HEAD deployment).
func applyMigrations(ctx context.Context, db *DB, files []string) error {
	for _, path := range files {
		sql, err := migrationsFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", path, err)
		}
		if _, err := db.Pool.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("exec migration %s: %w", path, err)
		}
	}
	return nil
}

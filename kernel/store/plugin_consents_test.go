package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
)

// bootDBForConsents boots an embedded Postgres instance and runs all
// migrations. It returns a fully migrated *DB and registers cleanup.
// The binary is cached under os.TempDir()/opendray-pg-cache so the
// ~100 MB download only happens once per machine.
// Tests are skipped under -short.
func bootDBForConsents(t *testing.T) *DB {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping embedded-postgres integration test in -short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	t.Cleanup(cancel)

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

	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	return db
}

// insertTestPlugin inserts a minimal plugins row to satisfy the FK on
// plugin_consents. It is a no-op if the plugin already exists.
func insertTestPlugin(t *testing.T, ctx context.Context, db *DB, name, version string) {
	t.Helper()
	_, err := db.Pool.Exec(ctx,
		`INSERT INTO plugins (name, version) VALUES ($1, $2)
		 ON CONFLICT (name) DO NOTHING`,
		name, version,
	)
	if err != nil {
		t.Fatalf("insertTestPlugin(%q): %v", name, err)
	}
}

// ── TestConsentRoundTrip ─────────────────────────────────────────────────────

// TestConsentRoundTrip verifies that UpsertConsent stores a row that
// GetConsent retrieves with exact field values.
func TestConsentRoundTrip(t *testing.T) {
	db := bootDBForConsents(t)
	ctx := context.Background()

	insertTestPlugin(t, ctx, db, "roundtrip-plugin", "1.0.0")

	perms := json.RawMessage(`{"exec":{"globs":["git *"]},"storage":true}`)
	c := PluginConsent{
		PluginName:   "roundtrip-plugin",
		ManifestHash: "abc123def456abc123def456abc123def456abc123def456abc123def456abc1",
		PermsJSON:    perms,
	}

	if err := db.UpsertConsent(ctx, c); err != nil {
		t.Fatalf("UpsertConsent: %v", err)
	}

	got, err := db.GetConsent(ctx, "roundtrip-plugin")
	if err != nil {
		t.Fatalf("GetConsent: %v", err)
	}

	t.Run("PluginName", func(t *testing.T) {
		if got.PluginName != c.PluginName {
			t.Errorf("got %q want %q", got.PluginName, c.PluginName)
		}
	})
	t.Run("ManifestHash", func(t *testing.T) {
		if got.ManifestHash != c.ManifestHash {
			t.Errorf("got %q want %q", got.ManifestHash, c.ManifestHash)
		}
	})
	t.Run("PermsJSON", func(t *testing.T) {
		// JSONB may reorder keys; compare as normalised JSON.
		var wantObj, gotObj any
		if err := json.Unmarshal(c.PermsJSON, &wantObj); err != nil {
			t.Fatalf("unmarshal want: %v", err)
		}
		if err := json.Unmarshal(got.PermsJSON, &gotObj); err != nil {
			t.Fatalf("unmarshal got: %v", err)
		}
		wantB, _ := json.Marshal(wantObj)
		gotB, _ := json.Marshal(gotObj)
		if string(wantB) != string(gotB) {
			t.Errorf("PermsJSON: got %s want %s", gotB, wantB)
		}
	})
	t.Run("GrantedAt non-zero", func(t *testing.T) {
		if got.GrantedAt.IsZero() {
			t.Error("GrantedAt is zero")
		}
	})
	t.Run("UpdatedAt non-zero", func(t *testing.T) {
		if got.UpdatedAt.IsZero() {
			t.Error("UpdatedAt is zero")
		}
	})
}

// ── TestConsentUpsertUpdatesTimestamp ────────────────────────────────────────

// TestConsentUpsertUpdatesTimestamp verifies that a second UpsertConsent
// call leaves granted_at unchanged but bumps updated_at.
func TestConsentUpsertUpdatesTimestamp(t *testing.T) {
	db := bootDBForConsents(t)
	ctx := context.Background()

	insertTestPlugin(t, ctx, db, "upsert-ts-plugin", "1.0.0")

	c := PluginConsent{
		PluginName:   "upsert-ts-plugin",
		ManifestHash: "first-hash",
		PermsJSON:    json.RawMessage(`{}`),
	}
	if err := db.UpsertConsent(ctx, c); err != nil {
		t.Fatalf("first UpsertConsent: %v", err)
	}

	first, err := db.GetConsent(ctx, "upsert-ts-plugin")
	if err != nil {
		t.Fatalf("GetConsent (first): %v", err)
	}

	// Capture a checkpoint between the two upserts.
	between := time.Now()
	time.Sleep(10 * time.Millisecond) // ensure now() inside Postgres advances

	c.ManifestHash = "second-hash"
	c.PermsJSON = json.RawMessage(`{"storage":true}`)
	if err := db.UpsertConsent(ctx, c); err != nil {
		t.Fatalf("second UpsertConsent: %v", err)
	}

	second, err := db.GetConsent(ctx, "upsert-ts-plugin")
	if err != nil {
		t.Fatalf("GetConsent (second): %v", err)
	}

	t.Run("GrantedAt unchanged", func(t *testing.T) {
		if !second.GrantedAt.Equal(first.GrantedAt) {
			t.Errorf("GrantedAt changed: first=%v second=%v", first.GrantedAt, second.GrantedAt)
		}
	})
	t.Run("UpdatedAt bumped after checkpoint", func(t *testing.T) {
		if !second.UpdatedAt.After(between) {
			t.Errorf("UpdatedAt not bumped: checkpoint=%v updated_at=%v", between, second.UpdatedAt)
		}
	})
	t.Run("ManifestHash updated", func(t *testing.T) {
		if second.ManifestHash != "second-hash" {
			t.Errorf("ManifestHash: got %q want %q", second.ManifestHash, "second-hash")
		}
	})
}

// ── TestConsentGetMissing ────────────────────────────────────────────────────

// TestConsentGetMissing verifies GetConsent returns ErrConsentNotFound
// for an unknown plugin name.
func TestConsentGetMissing(t *testing.T) {
	db := bootDBForConsents(t)
	ctx := context.Background()

	_, err := db.GetConsent(ctx, "totally-unknown-plugin")
	if !errors.Is(err, ErrConsentNotFound) {
		t.Errorf("want ErrConsentNotFound, got %v", err)
	}
}

// ── TestConsentDelete ────────────────────────────────────────────────────────

// TestConsentDelete checks that DeleteConsent removes a row and that
// deleting a non-existent name is idempotent (returns nil).
func TestConsentDelete(t *testing.T) {
	db := bootDBForConsents(t)
	ctx := context.Background()

	t.Run("delete existing then GetConsent returns ErrConsentNotFound", func(t *testing.T) {
		insertTestPlugin(t, ctx, db, "delete-plugin", "1.0.0")

		if err := db.UpsertConsent(ctx, PluginConsent{
			PluginName:   "delete-plugin",
			ManifestHash: "hashval",
			PermsJSON:    json.RawMessage(`{}`),
		}); err != nil {
			t.Fatalf("UpsertConsent: %v", err)
		}
		if err := db.DeleteConsent(ctx, "delete-plugin"); err != nil {
			t.Fatalf("DeleteConsent: %v", err)
		}
		_, err := db.GetConsent(ctx, "delete-plugin")
		if !errors.Is(err, ErrConsentNotFound) {
			t.Errorf("after delete: want ErrConsentNotFound, got %v", err)
		}
	})

	t.Run("delete non-existent is nil (idempotent)", func(t *testing.T) {
		if err := db.DeleteConsent(ctx, "never-existed-plugin"); err != nil {
			t.Errorf("DeleteConsent non-existent: want nil, got %v", err)
		}
	})
}

// ── TestUpdateConsentPerms ──────────────────────────────────────────────────

// TestUpdateConsentPerms verifies that UpdateConsentPerms:
//   - overwrites only perms_json on an existing row,
//   - bumps updated_at,
//   - leaves manifest_hash and granted_at untouched,
//   - returns ErrConsentNotFound (wrapped) for an unknown plugin.
func TestUpdateConsentPerms(t *testing.T) {
	db := bootDBForConsents(t)
	ctx := context.Background()

	insertTestPlugin(t, ctx, db, "update-perms-plugin", "1.0.0")

	orig := PluginConsent{
		PluginName:   "update-perms-plugin",
		ManifestHash: "original-hash",
		PermsJSON:    json.RawMessage(`{"storage":true,"events":["session.*"]}`),
	}
	if err := db.UpsertConsent(ctx, orig); err != nil {
		t.Fatalf("UpsertConsent: %v", err)
	}
	before, err := db.GetConsent(ctx, orig.PluginName)
	if err != nil {
		t.Fatalf("GetConsent before update: %v", err)
	}

	// Make sure now() inside Postgres advances between upsert and update.
	time.Sleep(10 * time.Millisecond)

	newPerms := json.RawMessage(`{"events":["session.*"]}`)
	if err := db.UpdateConsentPerms(ctx, orig.PluginName, newPerms); err != nil {
		t.Fatalf("UpdateConsentPerms: %v", err)
	}

	after, err := db.GetConsent(ctx, orig.PluginName)
	if err != nil {
		t.Fatalf("GetConsent after update: %v", err)
	}

	t.Run("perms_json replaced", func(t *testing.T) {
		var want, got any
		if err := json.Unmarshal(newPerms, &want); err != nil {
			t.Fatalf("unmarshal want: %v", err)
		}
		if err := json.Unmarshal(after.PermsJSON, &got); err != nil {
			t.Fatalf("unmarshal got: %v", err)
		}
		wantB, _ := json.Marshal(want)
		gotB, _ := json.Marshal(got)
		if string(wantB) != string(gotB) {
			t.Errorf("perms: got %s want %s", gotB, wantB)
		}
	})
	t.Run("manifest_hash unchanged", func(t *testing.T) {
		if after.ManifestHash != before.ManifestHash {
			t.Errorf("ManifestHash changed: before=%q after=%q",
				before.ManifestHash, after.ManifestHash)
		}
	})
	t.Run("granted_at unchanged", func(t *testing.T) {
		if !after.GrantedAt.Equal(before.GrantedAt) {
			t.Errorf("GrantedAt changed: before=%v after=%v",
				before.GrantedAt, after.GrantedAt)
		}
	})
	t.Run("updated_at bumped", func(t *testing.T) {
		if !after.UpdatedAt.After(before.UpdatedAt) {
			t.Errorf("UpdatedAt not bumped: before=%v after=%v",
				before.UpdatedAt, after.UpdatedAt)
		}
	})
	t.Run("unknown plugin returns ErrConsentNotFound", func(t *testing.T) {
		err := db.UpdateConsentPerms(ctx, "never-existed-plugin",
			json.RawMessage(`{}`))
		if !errors.Is(err, ErrConsentNotFound) {
			t.Errorf("want ErrConsentNotFound, got %v", err)
		}
	})
}

// ── TestConsentCascade ───────────────────────────────────────────────────────

// TestConsentCascade verifies:
//   - Deleting a plugins row cascades to plugin_consents (FK ON DELETE CASCADE).
//   - plugin_audit rows are NOT removed — audit is historical, not FK-linked.
func TestConsentCascade(t *testing.T) {
	db := bootDBForConsents(t)
	ctx := context.Background()

	insertTestPlugin(t, ctx, db, "cascade-plugin", "1.0.0")

	if err := db.UpsertConsent(ctx, PluginConsent{
		PluginName:   "cascade-plugin",
		ManifestHash: "hashcascade",
		PermsJSON:    json.RawMessage(`{}`),
	}); err != nil {
		t.Fatalf("UpsertConsent: %v", err)
	}

	for i := range 3 {
		if err := db.AppendAudit(ctx, AuditEntry{
			PluginName: "cascade-plugin",
			Ns:         "exec",
			Method:     fmt.Sprintf("method%d", i),
			Caps:       []string{"exec"},
			Result:     "ok",
			DurationMs: 1,
			ArgsHash:   "aabbcc",
			Message:    "",
		}); err != nil {
			t.Fatalf("AppendAudit %d: %v", i, err)
		}
	}

	// Delete the plugins row — should cascade to plugin_consents.
	if _, err := db.Pool.Exec(ctx,
		`DELETE FROM plugins WHERE name = $1`, "cascade-plugin"); err != nil {
		t.Fatalf("delete plugins row: %v", err)
	}

	t.Run("consent row gone (FK cascade)", func(t *testing.T) {
		var count int
		if err := db.Pool.QueryRow(ctx,
			`SELECT count(*) FROM plugin_consents WHERE plugin_name = $1`,
			"cascade-plugin").Scan(&count); err != nil {
			t.Fatalf("count plugin_consents: %v", err)
		}
		if count != 0 {
			t.Errorf("plugin_consents: want 0 after cascade, got %d", count)
		}
	})

	t.Run("audit rows remain (no FK)", func(t *testing.T) {
		var count int
		if err := db.Pool.QueryRow(ctx,
			`SELECT count(*) FROM plugin_audit WHERE plugin_name = $1`,
			"cascade-plugin").Scan(&count); err != nil {
			t.Fatalf("count plugin_audit: %v", err)
		}
		if count != 3 {
			t.Errorf("plugin_audit: want 3 historical rows after cascade, got %d", count)
		}
	})
}

// ── TestAuditAppendAndTail ───────────────────────────────────────────────────

// TestAuditAppendAndTail appends 3 entries and verifies TailAudit returns
// them newest-first with every field intact.
func TestAuditAppendAndTail(t *testing.T) {
	db := bootDBForConsents(t)
	ctx := context.Background()

	entries := []AuditEntry{
		{
			PluginName: "audit-plugin",
			Ns:         "exec",
			Method:     "run",
			Caps:       []string{"exec", "fs.read"},
			Result:     "ok",
			DurationMs: 42,
			ArgsHash:   "aabbcc112233",
			Message:    "first",
		},
		{
			PluginName: "audit-plugin",
			Ns:         "http",
			Method:     "GET",
			Caps:       []string{"http"},
			Result:     "denied",
			DurationMs: 5,
			ArgsHash:   "ddeeff445566",
			Message:    "no http grant",
		},
		{
			PluginName: "audit-plugin",
			Ns:         "install",
			Method:     "install",
			Caps:       []string{},
			Result:     "ok",
			DurationMs: 200,
			ArgsHash:   "",
			Message:    "",
		},
	}

	for i, e := range entries {
		if err := db.AppendAudit(ctx, e); err != nil {
			t.Fatalf("AppendAudit[%d]: %v", i, err)
		}
	}

	rows, err := db.TailAudit(ctx, "audit-plugin", 10)
	if err != nil {
		t.Fatalf("TailAudit: %v", err)
	}

	t.Run("returns 3 rows", func(t *testing.T) {
		if len(rows) != 3 {
			t.Fatalf("want 3 rows, got %d", len(rows))
		}
	})

	t.Run("newest-first ordering", func(t *testing.T) {
		// entries[2] inserted last → rows[0]; entries[0] inserted first → rows[2].
		if rows[0].Method != "install" {
			t.Errorf("rows[0].Method: want %q got %q", "install", rows[0].Method)
		}
		if rows[2].Method != "run" {
			t.Errorf("rows[2].Method: want %q got %q", "run", rows[2].Method)
		}
	})

	t.Run("all fields round-trip", func(t *testing.T) {
		r, e := rows[2], entries[0]
		checks := []struct {
			name string
			got  string
			want string
		}{
			{"PluginName", r.PluginName, e.PluginName},
			{"Ns", r.Ns, e.Ns},
			{"Method", r.Method, e.Method},
			{"Result", r.Result, e.Result},
			{"ArgsHash", r.ArgsHash, e.ArgsHash},
			{"Message", r.Message, e.Message},
		}
		for _, c := range checks {
			if c.got != c.want {
				t.Errorf("%s: got %q want %q", c.name, c.got, c.want)
			}
		}
		if r.DurationMs != e.DurationMs {
			t.Errorf("DurationMs: got %d want %d", r.DurationMs, e.DurationMs)
		}
	})

	t.Run("Caps slice round-trips", func(t *testing.T) {
		// rows[2] = entries[0]: Caps=["exec","fs.read"]
		r := rows[2]
		if len(r.Caps) != 2 || r.Caps[0] != "exec" || r.Caps[1] != "fs.read" {
			t.Errorf("Caps: got %v want [exec fs.read]", r.Caps)
		}
		// rows[0] = entries[2]: empty Caps
		r0 := rows[0]
		if len(r0.Caps) != 0 {
			t.Errorf("empty Caps: got %v want []", r0.Caps)
		}
	})
}

// ── TestAuditTailLimitClamp ──────────────────────────────────────────────────

// TestAuditTailLimitClamp verifies the server-side clamping behaviour of
// TailAudit. The effective limit is clamped to [1, 1000]:
//
//   - limit <= 0  → treated as 1 (returns exactly 1 row, not 0 or all)
//   - limit > 1000 → treated as 1000
func TestAuditTailLimitClamp(t *testing.T) {
	db := bootDBForConsents(t)
	ctx := context.Background()

	// Insert 5 rows.
	for i := range 5 {
		if err := db.AppendAudit(ctx, AuditEntry{
			PluginName: "clamp-plugin",
			Ns:         "exec",
			Method:     fmt.Sprintf("m%d", i),
			Caps:       []string{},
			Result:     "ok",
			DurationMs: i,
			ArgsHash:   "",
			Message:    "",
		}); err != nil {
			t.Fatalf("AppendAudit: %v", err)
		}
	}

	t.Run("limit=0 clamps to 1", func(t *testing.T) {
		rows, err := db.TailAudit(ctx, "clamp-plugin", 0)
		if err != nil {
			t.Fatalf("TailAudit(0): %v", err)
		}
		if len(rows) != 1 {
			t.Errorf("limit=0: want 1 row (clamped), got %d", len(rows))
		}
	})

	t.Run("limit=2 returns exactly 2", func(t *testing.T) {
		rows, err := db.TailAudit(ctx, "clamp-plugin", 2)
		if err != nil {
			t.Fatalf("TailAudit(2): %v", err)
		}
		if len(rows) != 2 {
			t.Errorf("limit=2: want 2 rows, got %d", len(rows))
		}
	})

	t.Run("limit=2000 clamps to 1000 (5 rows in table returns 5)", func(t *testing.T) {
		rows, err := db.TailAudit(ctx, "clamp-plugin", 2000)
		if err != nil {
			t.Fatalf("TailAudit(2000): %v", err)
		}
		if len(rows) > 1000 {
			t.Errorf("limit=2000: got %d rows, want ≤1000", len(rows))
		}
		if len(rows) != 5 {
			t.Errorf("limit=2000 with 5 rows: got %d want 5", len(rows))
		}
	})
}

// ── TestAppendAuditRejectsSQLi ───────────────────────────────────────────────

// TestAppendAuditRejectsSQLi is a regression test that ensures AppendAudit
// uses parameterised queries. A plugin_name containing SQL metacharacters
// must be stored as a literal string — the table must not be dropped.
func TestAppendAuditRejectsSQLi(t *testing.T) {
	db := bootDBForConsents(t)
	ctx := context.Background()

	sqliName := `'; DROP TABLE plugin_audit; --`

	err := db.AppendAudit(ctx, AuditEntry{
		PluginName: sqliName,
		Ns:         `exec'; DROP TABLE plugin_audit; --`,
		Method:     `method'; DELETE FROM plugin_audit; --`,
		Caps:       []string{`exec'; DROP TABLE plugin_audit; --`},
		Result:     "ok",
		DurationMs: 0,
		ArgsHash:   `hash'; DROP TABLE plugin_audit; --`,
		Message:    `msg'; DROP TABLE plugin_audit; --`,
	})
	if err != nil {
		t.Fatalf("AppendAudit with SQLi payload: %v", err)
	}

	// The table must still exist and hold the row with the literal string.
	var count int
	if err := db.Pool.QueryRow(ctx,
		`SELECT count(*) FROM plugin_audit WHERE plugin_name = $1`,
		sqliName).Scan(&count); err != nil {
		t.Fatalf("count after SQLi payload: %v (table may have been dropped)", err)
	}
	if count != 1 {
		t.Errorf("want 1 literal row with SQLi plugin_name, got %d", count)
	}
}

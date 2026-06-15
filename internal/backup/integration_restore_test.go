package backup

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	storepkg "github.com/opendray/opendray-v2/internal/store"
)

// These tests exercise the real backup → verify → restore (dry-run +
// apply) path against a live PostgreSQL, using pg_dump / pg_restore.
// They're gated on OPENDRAY_DEV_DB_URL (a URL-form, writable DSN that
// can CREATE DATABASE) and skip cleanly when it's absent — so unit CI
// stays green while local/integration runs get real coverage.

func integrationDSN(t *testing.T) string {
	t.Helper()
	v := os.Getenv("OPENDRAY_DEV_DB_URL")
	if v == "" {
		t.Skip("OPENDRAY_DEV_DB_URL not set; export a writable URL-form Postgres DSN to run restore integration tests")
	}
	if u, err := url.Parse(v); err != nil || u.Scheme == "" {
		t.Skip("OPENDRAY_DEV_DB_URL is not a URL-form DSN; integration test needs one to swap the database name")
	}
	return v
}

func requireBinary(t *testing.T, name string) {
	t.Helper()
	if _, err := exec.LookPath(name); err != nil {
		t.Skipf("%s not on PATH; skipping restore integration test", name)
	}
}

// swapDBName clones a URL-form DSN with a different database name, so we
// can point pg_dump / pg_restore at throwaway databases on the same
// server without disturbing the dev DB.
func swapDBName(dsn, name string) string {
	u, _ := url.Parse(dsn) // integrationDSN already validated it parses
	u.Path = "/" + name
	return u.String()
}

func mustExec(t *testing.T, ctx context.Context, pool *pgxpool.Pool, sql string) {
	t.Helper()
	if _, err := pool.Exec(ctx, sql); err != nil {
		t.Fatalf("exec %q: %v", sql, err)
	}
}

func TestIntegration_FullInstanceBackupRestoreRoundtrip(t *testing.T) {
	dsn := integrationDSN(t)
	requireBinary(t, "pg_dump")
	requireBinary(t, "pg_restore")

	// Generous deadline: the test runs two full backup+verify+restore
	// cycles (the second is the apply's pre-restore safety snapshot).
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	// Metadata/store DB = the dev DB, with backup tables ensured via
	// migrations. Backup rows we insert here are cleaned up at the end.
	st, err := storepkg.Open(ctx, dsn, 4)
	if err != nil {
		t.Skipf("dev DB unreachable, skipping: %v", err)
	}
	// Close the store via Cleanup (registered FIRST so it runs LAST in
	// LIFO): the DROP DATABASE / DELETE cleanups below run through this
	// same pool and must outlive a plain `defer st.Close()`.
	t.Cleanup(st.Close)
	if err := st.Migrate(ctx, slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		t.Fatalf("migrate dev DB: %v", err)
	}
	pool := st.Pool()

	// Throwaway source (dumped) + destination (restored into) databases.
	// Names are digits/underscores only, but quote the identifier anyway.
	suffix := fmt.Sprintf("%d_%d", os.Getpid(), time.Now().UnixNano()%1_000_000)
	srcDB := "od_dr_src_" + suffix
	dstDB := "od_dr_dst_" + suffix
	mustExec(t, ctx, pool, `CREATE DATABASE "`+srcDB+`"`)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DROP DATABASE IF EXISTS "`+srcDB+`" WITH (FORCE)`)
	})
	mustExec(t, ctx, pool, `CREATE DATABASE "`+dstDB+`"`)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DROP DATABASE IF EXISTS "`+dstDB+`" WITH (FORCE)`)
	})

	srcDSN := swapDBName(dsn, srcDB)
	dstDSN := swapDBName(dsn, dstDB)

	// Seed the source DB with a marker row we can assert lands in the
	// destination after an apply-mode restore.
	srcPool, err := pgxpool.New(ctx, srcDSN)
	if err != nil {
		t.Fatalf("open source pool: %v", err)
	}
	defer srcPool.Close()
	mustExec(t, ctx, srcPool, `CREATE TABLE dr_marker (id int PRIMARY KEY, note text)`)
	mustExec(t, ctx, srcPool, `INSERT INTO dr_marker (id, note) VALUES (1, 'hello-dr')`)

	// On-disk inputs for a full_instance bundle: a vault note, a
	// secrets.env, and a config.toml.
	tmp := t.TempDir()
	localDir := filepath.Join(tmp, "backups")
	notesDir := filepath.Join(tmp, "vault", "notes")
	if err := os.MkdirAll(notesDir, 0o700); err != nil {
		t.Fatalf("mkdir notes: %v", err)
	}
	if err := os.WriteFile(filepath.Join(notesDir, "hello.md"), []byte("# note\n"), 0o600); err != nil {
		t.Fatalf("write note: %v", err)
	}
	secretsFile := filepath.Join(tmp, "secrets.env")
	if err := os.WriteFile(secretsFile, []byte("API_KEY=shh\n"), 0o600); err != nil {
		t.Fatalf("write secrets: %v", err)
	}
	configFile := filepath.Join(tmp, "config.toml")
	if err := os.WriteFile(configFile, []byte("[server]\nport=8770\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	svc, err := NewService(Config{
		Enabled:      true,
		LocalDir:     localDir,
		VaultSources: []VaultSource{{Logical: "notes", Dir: notesDir}},
		SecretsFile:  secretsFile,
	}, ServiceDeps{
		Pool:       pool,
		Passphrase: "integration-test-passphrase-0123456789",
		DSN:        srcDSN,
		ConfigPath: configFile,
		Log:        slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	if svc.pgrestore == nil {
		// requireBinary already confirmed pg_restore is on PATH, so a nil
		// here is a real wiring failure, not a reason to skip.
		t.Fatal("pg_restore on PATH but Service didn't wire it")
	}
	if err := svc.Bootstrap(ctx); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}

	// ── Backup (full_instance) ──────────────────────────────────────
	b, err := svc.RunBackupSync(ctx, RunBackupRequest{
		TargetID:    "local",
		TriggeredBy: TriggeredManual,
		Kind:        KindFullInstance,
	})
	if err != nil {
		t.Fatalf("RunBackupSync: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM backups WHERE id=$1`, b.ID)
	})
	if b.Status != BackupSucceeded {
		t.Fatalf("backup status=%s error=%s", b.Status, b.Error)
	}
	if b.Kind != KindFullInstance {
		t.Fatalf("backup kind=%s, want full_instance", b.Kind)
	}
	// doRunBackup verifies the blob (pg_restore --list) best-effort.
	if b.VerifiedAt == nil || b.VerifyError != "" {
		t.Errorf("expected backup verified, got verified_at=%v verify_error=%q",
			b.VerifiedAt, b.VerifyError)
	}

	// ── Restore dry-run: reports a plan, changes nothing ────────────
	rc, _, err := svc.DownloadBackup(ctx, b.ID)
	if err != nil {
		t.Fatalf("DownloadBackup (dry-run): %v", err)
	}
	defer rc.Close()
	dry, err := svc.RestoreBackup(ctx, RestoreRequest{
		Source: rc, TargetDSN: dstDSN, Apply: false,
	})
	if err != nil {
		t.Fatalf("dry-run restore: %v", err)
	}
	if !dry.Plan.DryRun || !dry.Plan.DumpPresent {
		t.Errorf("dry-run plan: DryRun=%v DumpPresent=%v", dry.Plan.DryRun, dry.Plan.DumpPresent)
	}
	if dry.Plan.VaultFiles < 1 {
		t.Errorf("dry-run plan VaultFiles=%d, want >=1", dry.Plan.VaultFiles)
	}
	if dry.Plan.ConfigPath != configFile {
		t.Errorf("dry-run plan ConfigPath=%q, want %q", dry.Plan.ConfigPath, configFile)
	}
	if dry.Plan.SecretsPath != secretsFile {
		t.Errorf("dry-run plan SecretsPath=%q, want %q", dry.Plan.SecretsPath, secretsFile)
	}
	if dry.Plan.SafetySnapshotID != "" {
		t.Errorf("dry-run took a safety snapshot (%s); it must not", dry.Plan.SafetySnapshotID)
	}
	// Destination still empty — dry-run wrote nothing.
	dstPool, err := pgxpool.New(ctx, dstDSN)
	if err != nil {
		t.Fatalf("open dest pool: %v", err)
	}
	defer dstPool.Close()
	var exists bool
	if err := dstPool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='dr_marker')`).
		Scan(&exists); err != nil {
		t.Fatalf("check dst pre-apply: %v", err)
	}
	if exists {
		t.Fatal("dry-run created dr_marker in the destination; it must not write")
	}

	// ── Restore apply: safety snapshot + write + pg_restore ─────────
	rc2, _, err := svc.DownloadBackup(ctx, b.ID)
	if err != nil {
		t.Fatalf("DownloadBackup (apply): %v", err)
	}
	defer rc2.Close()
	res, err := svc.RestoreBackup(ctx, RestoreRequest{
		Source: rc2, TargetDSN: dstDSN, Clean: false, Apply: true,
	})
	if err != nil {
		t.Fatalf("apply restore: %v\noutput: %s", err, res.PGRestoreOutput)
	}
	if !res.FingerprintOK {
		t.Error("apply: fingerprint did not match running cipher")
	}
	if res.Plan.SafetySnapshotID == "" {
		t.Error("apply: expected a pre-restore safety snapshot id")
	} else {
		snapID := res.Plan.SafetySnapshotID
		t.Cleanup(func() {
			_, _ = pool.Exec(context.Background(), `DELETE FROM backups WHERE id=$1`, snapID)
		})
	}
	if !containsAll(res.Plan.Applied, "database", "vault", "config", "secrets.env") {
		t.Errorf("apply: Plan.Applied=%v, want database+vault+config+secrets.env", res.Plan.Applied)
	}

	// The marker row replayed into the destination database.
	var note string
	if err := dstPool.QueryRow(ctx, `SELECT note FROM dr_marker WHERE id=1`).Scan(&note); err != nil {
		t.Fatalf("read restored marker: %v", err)
	}
	if note != "hello-dr" {
		t.Errorf("restored marker note=%q, want hello-dr", note)
	}
}

func containsAll(have []string, want ...string) bool {
	set := make(map[string]bool, len(have))
	for _, h := range have {
		set[h] = true
	}
	for _, w := range want {
		if !set[w] {
			return false
		}
	}
	return true
}

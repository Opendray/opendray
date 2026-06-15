package backup

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/opendray/opendray-v2/internal/version"
)

// SkipPreMigrateEnv disables the automatic pre-migration snapshot when
// set to any non-empty value. The escape hatch for operators who
// accept the upgrade risk (or have their own backup in front).
const SkipPreMigrateEnv = "OPENDRAY_SKIP_PREMIGRATE_BACKUP"

// PreMigrateOptions configures a pre-migration snapshot. It is
// deliberately self-contained — no Service, no targets, no DB row — so
// it can run on the migrate code path before the schema (and the
// backups table itself) exists.
type PreMigrateOptions struct {
	DSN        string // libpq conninfo for pg_dump
	Dir        string // directory to write the snapshot into
	PgDumpPath string // optional explicit pg_dump path; "" → resolve from PATH
	Passphrase string // "" → write a 0600 plaintext dump instead of a sealed bundle
	Log        *slog.Logger
}

// GuardPreMigrate takes a pre-migration snapshot when pending is
// non-empty. It is FAIL-CLOSED: a snapshot error is returned so the
// caller aborts the upgrade before the schema is touched. Operators who
// accept the risk set OPENDRAY_SKIP_PREMIGRATE_BACKUP to skip it.
func GuardPreMigrate(ctx context.Context, pending []string, o PreMigrateOptions) error {
	log := o.Log
	if log == nil {
		log = slog.Default()
	}
	if len(pending) == 0 {
		return nil
	}
	if os.Getenv(SkipPreMigrateEnv) != "" {
		log.Warn("pre-migrate snapshot skipped via env",
			"env", SkipPreMigrateEnv, "pending", len(pending))
		return nil
	}
	path, err := SnapshotBeforeMigrate(ctx, o)
	if err != nil {
		return fmt.Errorf("pre-migrate snapshot failed (set %s=1 to skip): %w", SkipPreMigrateEnv, err)
	}
	log.Info("pre-migrate snapshot written", "path", path, "pending", len(pending))
	return nil
}

// SnapshotBeforeMigrate dumps the database and writes either a sealed
// bundle (when Passphrase is set — restorable via the normal restore
// path) or a 0600 plaintext pg_dump (when no passphrase is available).
// Returns the path written.
func SnapshotBeforeMigrate(ctx context.Context, o PreMigrateOptions) (string, error) {
	if o.DSN == "" {
		return "", fmt.Errorf("premigrate: dsn empty")
	}
	if o.Dir == "" {
		return "", fmt.Errorf("premigrate: dir empty")
	}
	log := o.Log
	if log == nil {
		log = slog.Default()
	}
	pgdump, err := NewPgDump(o.PgDumpPath)
	if err != nil {
		return "", fmt.Errorf("premigrate: %w", err)
	}
	if err := os.MkdirAll(o.Dir, 0o700); err != nil {
		return "", fmt.Errorf("premigrate: mkdir: %w", err)
	}

	// pg_dump → temp file (a stable size is needed for the bundle header).
	tmpName, err := dumpToTemp(ctx, pgdump, o.DSN, o.Dir)
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpName)

	stamp := time.Now().UTC().Format("20060102-150405")

	// Plaintext fallback when no passphrase is configured. Mode 0600,
	// local only — it contains the whole DB (incl. plaintext tokens).
	if o.Passphrase == "" {
		dst := filepath.Join(o.Dir, fmt.Sprintf("pre-migrate-%s.dump", stamp))
		if err := copyFile(tmpName, dst, 0o600); err != nil {
			return "", fmt.Errorf("premigrate: write plaintext: %w", err)
		}
		log.Warn("pre-migrate snapshot is UNENCRYPTED (no backup passphrase configured)", "path", dst)
		return dst, nil
	}

	cipher, err := NewCipher(o.Passphrase)
	if err != nil {
		return "", fmt.Errorf("premigrate: cipher: %w", err)
	}
	dst := filepath.Join(o.Dir, fmt.Sprintf("pre-migrate-%s.tar.gz.enc", stamp))
	if err := sealDumpToBundle(tmpName, dst, cipher, stamp); err != nil {
		return "", err
	}
	return dst, nil
}

// dumpToTemp streams pg_dump into a temp file in dir and returns its path.
func dumpToTemp(ctx context.Context, pgdump *PgDump, dsn, dir string) (string, error) {
	tmp, err := os.CreateTemp(dir, ".premigrate-*.bin")
	if err != nil {
		return "", fmt.Errorf("premigrate: tmp: %w", err)
	}
	name := tmp.Name()
	res, err := pgdump.Dump(ctx, dsn)
	if err != nil {
		_ = tmp.Close()
		_ = os.Remove(name)
		return "", fmt.Errorf("premigrate: pg_dump: %w", err)
	}
	if _, err := io.Copy(tmp, res.Reader); err != nil {
		_ = tmp.Close()
		_ = os.Remove(name)
		return "", fmt.Errorf("premigrate: pg_dump copy: %w", err)
	}
	if err := res.Wait(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(name)
		return "", fmt.Errorf("premigrate: pg_dump wait: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(name)
		return "", fmt.Errorf("premigrate: close tmp: %w", err)
	}
	return name, nil
}

// sealDumpToBundle wraps a pg_dump file as a sealed manifest+dump bundle
// (the same layout RestoreBackup understands) at dst (mode 0600).
func sealDumpToBundle(dumpPath, dst string, cipher Cipher, stamp string) error {
	stat, err := os.Stat(dumpPath)
	if err != nil {
		return fmt.Errorf("premigrate: stat tmp: %w", err)
	}
	dumpFile, err := os.Open(dumpPath)
	if err != nil {
		return fmt.Errorf("premigrate: reopen tmp: %w", err)
	}
	defer dumpFile.Close()

	info := version.Current()
	manifest := BundleManifest{
		BackupID:        "premigrate-" + stamp,
		CreatedAt:       time.Now().UTC(),
		OpendrayVersion: info.Version,
		GitSHA:          info.Commit,
		Encryption: ManifestEncryption{
			Algo:        "aes-256-gcm-chunked",
			Fingerprint: cipher.Fingerprint(),
		},
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("premigrate: create out: %w", err)
	}
	bundleR, bundleW := io.Pipe()
	go func() {
		werr := WriteBundle(bundleW, manifest, []BundleSource{
			{Name: "dump.bin", Body: dumpFile, Size: stat.Size()},
		})
		_ = bundleW.CloseWithError(werr)
	}()
	_, copyErr := io.Copy(out, cipher.Seal(bundleR))
	closeErr := out.Close()
	if copyErr != nil {
		// Unblock the WriteBundle goroutine (and the cipher.Seal reader
		// it feeds) so it doesn't leak waiting on a reader we abandoned.
		_ = bundleR.CloseWithError(copyErr)
		_ = os.Remove(dst)
		return fmt.Errorf("premigrate: seal/write: %w", copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("premigrate: close out: %w", closeErr)
	}
	return nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		_ = os.Remove(dst) // don't leave a truncated dump behind
		return err
	}
	return out.Close()
}

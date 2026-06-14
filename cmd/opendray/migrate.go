// Migrate subcommand for the opendray binary.
//
// Runs DB migrations against the configured Postgres DSN and exits.
// Intentionally bypasses internal/app.New so it works on a fresh
// database where the schema doesn't exist yet — app.New's catalog
// subsystem upserts seed rows into a `providers` table that
// migration 0001 hasn't yet created, which previously crashed
// `opendray migrate` before any migration could run (see #162).
//
// This file deliberately mirrors the shape of `serve` only as far
// as the lifecycle surface (config + ctx + logger). It does not
// share `internal/app.run` because the whole point of the fix is
// that migrate must NOT compose the full app graph.

package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/opendray/opendray-v2/internal/backup"
	"github.com/opendray/opendray-v2/internal/config"
	"github.com/opendray/opendray-v2/internal/store"
)

func runMigrate(args []string) int {
	fs := flag.NewFlagSet("migrate", flag.ExitOnError)
	cfgPath := fs.String("config", "", "path to config.toml (env-only mode if empty)")
	_ = fs.Parse(args)

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	log := newMigrateLogger(cfg.Log)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	st, err := store.Open(ctx, cfg.Database.URL, cfg.Database.MaxConns)
	if err != nil {
		log.Error("open store", "err", err)
		return 1
	}
	defer st.Close()

	// Pre-migration safety snapshot (fail-closed) before touching the
	// schema. Set OPENDRAY_SKIP_PREMIGRATE_BACKUP to skip.
	pending, perr := st.PendingMigrations(ctx)
	if perr != nil {
		log.Error("check pending migrations", "err", perr)
		return 1
	}
	if len(pending) > 0 {
		preKey, kerr := backup.LoadPassphrase()
		if kerr != nil {
			// Don't silently fall back to a plaintext snapshot when a
			// key source is configured but unreadable — fail closed.
			log.Error("pre-migrate snapshot: backup key load", "err", kerr)
			return 1
		}
		if gerr := backup.GuardPreMigrate(ctx, pending, backup.PreMigrateOptions{
			DSN:        cfg.Database.URL,
			Dir:        premigrateDir(cfg.Backup.LocalDir),
			PgDumpPath: cfg.Backup.PgDumpPath,
			Passphrase: preKey.Passphrase,
			Log:        log,
		}); gerr != nil {
			log.Error("pre-migrate snapshot", "err", gerr)
			return 1
		}
	}

	if err := st.Migrate(ctx, log); err != nil {
		log.Error("apply migrations", "err", err)
		return 1
	}
	return 0
}

// premigrateDir resolves where pre-migration snapshots are written,
// mirroring app.defaultBackupDir: ~-expand a configured local_dir, or
// default to ~/.opendray/backups when unset.
func premigrateDir(configured string) string {
	base := configured
	if base != "" {
		if expanded, err := expandHome(base); err == nil {
			base = expanded
		}
	} else if home, err := os.UserHomeDir(); err == nil {
		base = filepath.Join(home, ".opendray", "backups")
	} else {
		base = "backups"
	}
	return filepath.Join(base, "premigrate")
}

// newMigrateLogger builds a minimal slog.Logger writing to stderr,
// honouring cfg.Log.Level and cfg.Log.Format. It deliberately skips
// the ring buffer and rotating-file outputs that internal/app's
// newLogger sets up — migrate is a short-lived one-shot, those
// surfaces add no value here.
func newMigrateLogger(cfg config.LogConfig) *slog.Logger {
	var level slog.Level
	switch strings.ToLower(strings.TrimSpace(cfg.Level)) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{Level: level}
	var h slog.Handler
	if strings.EqualFold(strings.TrimSpace(cfg.Format), "json") {
		h = slog.NewJSONHandler(os.Stderr, opts)
	} else {
		h = slog.NewTextHandler(os.Stderr, opts)
	}
	return slog.New(h).With("component", "migrate")
}

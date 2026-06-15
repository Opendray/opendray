package backup

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// PgRestore wraps the pg_restore CLI. Counterpart to PgDump in
// pgdump.go; same binary-on-PATH lookup conventions, same exec
// boundary, same major-version compatibility caveats.
type PgRestore struct {
	binPath string
}

// NewPgRestore resolves pg_restore. Empty binPath = PATH lookup.
// Returns ErrPgRestoreUnavailable on failure.
func NewPgRestore(binPath string) (*PgRestore, error) {
	if binPath == "" {
		resolved, err := exec.LookPath("pg_restore")
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrPgRestoreUnavailable, err)
		}
		return &PgRestore{binPath: resolved}, nil
	}
	if _, err := exec.LookPath(binPath); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPgRestoreUnavailable, err)
	}
	return &PgRestore{binPath: binPath}, nil
}

func (p *PgRestore) BinPath() string { return p.binPath }

// Version invokes `pg_restore --version`.
func (p *PgRestore) Version(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, p.binPath, "--version").Output()
	if err != nil {
		return "", fmt.Errorf("pg_restore --version: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// RestoreOptions controls Restore behaviour.
type RestoreOptions struct {
	// Clean drops existing schema objects (tables/indexes) before
	// restoring. Equivalent to --clean --if-exists. The default
	// (false) is "additive only" — restore fails on any conflict.
	Clean bool

	// SingleTransaction wraps the entire restore in BEGIN..COMMIT
	// so a partial failure rolls everything back. Adds memory
	// pressure on big dumps; default off.
	SingleTransaction bool

	// CreateDatabase issues CREATE DATABASE before the data load.
	// Requires the DSN to point at a different database (typically
	// 'postgres') because you can't create a DB you're connected
	// to. Off by default.
	CreateDatabase bool
}

// Restore replays a dump file into the given DSN. Returns the tail
// of pg_restore's combined stdout+stderr on success (helpful in UI
// to confirm "X restored, Y warnings"); on failure returns the same
// alongside the error.
func (p *PgRestore) Restore(ctx context.Context, dumpPath, dsn string, opts RestoreOptions) (string, error) {
	if dumpPath == "" {
		return "", errors.New("pg_restore: dumpPath required")
	}
	if dsn == "" {
		return "", errors.New("pg_restore: dsn required")
	}

	args := []string{
		"--no-owner",
		"--no-privileges",
		"--dbname=" + dsn,
	}
	if opts.Clean {
		args = append(args, "--clean", "--if-exists")
	}
	if opts.SingleTransaction {
		args = append(args, "--single-transaction")
	}
	if opts.CreateDatabase {
		args = append(args, "--create")
	}
	args = append(args, dumpPath)

	cmd := exec.CommandContext(ctx, p.binPath, args...)
	out, err := cmd.CombinedOutput()
	tail := tailString(string(out), 8<<10) // 8 KiB tail
	if err != nil {
		return tail, fmt.Errorf("pg_restore exit: %w", err)
	}
	return tail, nil
}

// List runs `pg_restore --list` against a dump file to confirm it is a
// readable archive, WITHOUT restoring anything or needing a database.
// Used to verify a freshly-written backup. A non-nil error means the
// dump is unreadable / corrupt (or pg_restore is incompatible); the
// returned string is the tail of pg_restore's output either way.
func (p *PgRestore) List(ctx context.Context, dumpPath string) (string, error) {
	if dumpPath == "" {
		return "", errors.New("pg_restore: dumpPath required")
	}
	cmd := exec.CommandContext(ctx, p.binPath, "--list", dumpPath)
	out, err := cmd.CombinedOutput()
	tail := tailString(string(out), 8<<10)
	if err != nil {
		return tail, fmt.Errorf("pg_restore --list: %w", err)
	}
	return tail, nil
}

// tailString returns the last n bytes of s (UTF-8 unaware — that's
// fine for pg_restore output which is ASCII / English-locale).
func tailString(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}

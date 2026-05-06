package backup

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"
)

// PgDump wraps the pg_dump CLI. We don't link libpq because every
// postgres distribution ships the binary anyway, and shelling out
// removes a CGO dependency from opendray.
//
// The binary's major version must be ≥ the server's. Mismatch is
// surfaced via Version() so callers can write the value into
// backups.pg_version and let restore-time tooling pick the right
// pg_restore.
type PgDump struct {
	binPath string
}

// NewPgDump resolves the pg_dump binary. Empty binPath falls back
// to PATH lookup. Returns ErrPgDumpUnavailable when nothing usable
// is found, so callers can errors.Is and surface a 503 instead of
// a generic exec error.
func NewPgDump(binPath string) (*PgDump, error) {
	if binPath == "" {
		resolved, err := exec.LookPath("pg_dump")
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrPgDumpUnavailable, err)
		}
		binPath = resolved
		return &PgDump{binPath: binPath}, nil
	}
	if _, err := exec.LookPath(binPath); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPgDumpUnavailable, err)
	}
	return &PgDump{binPath: binPath}, nil
}

// BinPath returns the absolute path to the resolved pg_dump binary.
func (p *PgDump) BinPath() string { return p.binPath }

// Version invokes `pg_dump --version` and returns the trimmed
// stdout (e.g. "pg_dump (PostgreSQL) 14.19 (Homebrew)").
func (p *PgDump) Version(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, p.binPath, "--version")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("pg_dump --version: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// DumpResult is returned by Dump. The caller must read Reader to
// EOF (or fully drain on cancellation) before calling Wait — pg_dump
// blocks on its stdout pipe otherwise.
type DumpResult struct {
	Reader io.Reader
	Wait   func() error
}

// Dump streams pg_dump --format=custom output for the given DSN.
//
// Fixed flag set:
//
//	--format=custom        — compressed, supports parallel pg_restore
//	--no-owner             — restore-target's user owns objects
//	--no-privileges        — strip GRANT statements
//	--dbname=<dsn>         — full conninfo URL or libpq connstring
//
// We DON'T pass --blobs or --jobs — both are out of scope for v1.
//
// Cancellation: ctx.Done() kills the child via CommandContext.
func (p *PgDump) Dump(ctx context.Context, dsn string) (*DumpResult, error) {
	if dsn == "" {
		return nil, errors.New("pg_dump: dsn required")
	}
	cmd := exec.CommandContext(ctx, p.binPath,
		"--format=custom",
		"--no-owner",
		"--no-privileges",
		"--dbname="+dsn,
	)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("pg_dump: stdout pipe: %w", err)
	}
	stderrBuf := newCappedBuf(32 << 10)
	cmd.Stderr = stderrBuf

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("pg_dump: start: %w", err)
	}

	wait := func() error {
		if err := cmd.Wait(); err != nil {
			return fmt.Errorf("pg_dump exit: %w; stderr: %s", err, stderrBuf.String())
		}
		return nil
	}
	return &DumpResult{Reader: stdout, Wait: wait}, nil
}

// cappedBuf retains only the last N bytes written. pg_dump's
// stderr on a malformed connstring can dump many KB; we only need
// the tail to surface in the error message.
type cappedBuf struct {
	cap int
	buf []byte
}

func newCappedBuf(cap int) *cappedBuf { return &cappedBuf{cap: cap} }

func (b *cappedBuf) Write(p []byte) (int, error) {
	b.buf = append(b.buf, p...)
	if len(b.buf) > b.cap {
		b.buf = b.buf[len(b.buf)-b.cap:]
	}
	return len(p), nil
}

func (b *cappedBuf) String() string { return string(b.buf) }

// ParsePGMajorMinor extracts "14.19" from a Version() string. If
// no version pattern is found returns "". Robust against varying
// suffixes like "(Homebrew)" or "(Debian 14.19-1.pgdg120+1)".
func ParsePGMajorMinor(versionLine string) string {
	m := versionMajorMinorRe.FindStringSubmatch(versionLine)
	if len(m) < 3 {
		return ""
	}
	return m[1] + "." + m[2]
}

var versionMajorMinorRe = regexp.MustCompile(`(\d+)\.(\d+)`)

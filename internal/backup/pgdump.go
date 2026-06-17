package backup

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
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

// pgDumpSearchGlobs lists locations probed (in addition to PATH) to find
// the newest installed pg_dump when no explicit path is configured.
// pg_dump is forward-compatible — a newer binary dumps same-or-older
// servers fine, but an OLDER binary refuses a newer server ("server
// version mismatch") — so "newest installed wins" is the safe default.
// This is what stops a stale PATH default (e.g. an old Homebrew pg_dump
// 14 ahead of an installed 17) from version-mismatching the server and
// failing the FAIL-CLOSED pre-migrate snapshot into a launchd restart
// loop. A package var so tests can neutralize filesystem discovery.
var pgDumpSearchGlobs = []string{
	"/opt/homebrew/opt/postgresql@*/bin/pg_dump", // macOS Homebrew (Apple silicon)
	"/opt/homebrew/bin/pg_dump",
	"/usr/local/opt/postgresql@*/bin/pg_dump", // macOS Homebrew (Intel)
	"/usr/local/bin/pg_dump",
	"/usr/lib/postgresql/*/bin/pg_dump", // Debian/Ubuntu
	"/usr/pgsql-*/bin/pg_dump",          // RHEL/PGDG
	"/usr/bin/pg_dump",
}

// NewPgDump resolves the pg_dump binary.
//   - A non-empty binPath is an explicit operator override (cfg.backup.
//     pg_dump_path); it is used as-is so operators stay in control.
//   - An empty binPath triggers AUTO-DISCOVERY: PATH plus the well-known
//     install locations are probed and the NEWEST major.minor wins, so a
//     fresh install prefers e.g. postgresql@17 over a PATH default of 14
//     with zero config. Falls back to a plain PATH lookup when discovery
//     turns up nothing usable.
//
// Returns ErrPgDumpUnavailable when nothing usable is found, so callers
// can errors.Is and surface a 503 instead of a generic exec error.
func NewPgDump(binPath string) (*PgDump, error) {
	if binPath != "" {
		if _, err := exec.LookPath(binPath); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrPgDumpUnavailable, err)
		}
		return &PgDump{binPath: binPath}, nil
	}
	if best := pickNewestPgDump(discoverPgDumpCandidates()); best != "" {
		return &PgDump{binPath: best}, nil
	}
	// Fallback: a plain PATH lookup (also covers odd environments where
	// `--version` probing failed for every discovered candidate).
	resolved, err := exec.LookPath("pg_dump")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPgDumpUnavailable, err)
	}
	return &PgDump{binPath: resolved}, nil
}

// pgDumpCandidate is a discovered pg_dump binary and its parsed version.
type pgDumpCandidate struct {
	path  string
	major int
	minor int
}

// discoverPgDumpCandidates probes PATH and pgDumpSearchGlobs for usable
// pg_dump binaries, de-duplicated by absolute path, each annotated with
// its `--version`. Binaries whose version can't be probed/parsed are
// skipped.
func discoverPgDumpCandidates() []pgDumpCandidate {
	seen := map[string]bool{}
	var out []pgDumpCandidate
	consider := func(path string) {
		if path == "" {
			return
		}
		if abs, err := filepath.Abs(path); err == nil {
			path = abs
		}
		if seen[path] {
			return
		}
		info, err := os.Stat(path)
		if err != nil || info.IsDir() || info.Mode()&0o111 == 0 {
			return
		}
		seen[path] = true
		if maj, min, ok := pgDumpVersionOf(path); ok {
			out = append(out, pgDumpCandidate{path: path, major: maj, minor: min})
		}
	}
	if p, err := exec.LookPath("pg_dump"); err == nil {
		consider(p)
	}
	for _, g := range pgDumpSearchGlobs {
		matches, _ := filepath.Glob(g)
		for _, m := range matches {
			consider(m)
		}
	}
	return out
}

// pgDumpVersionOf runs `<path> --version` and parses its major.minor.
func pgDumpVersionOf(path string) (int, int, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, path, "--version").Output()
	if err != nil {
		return 0, 0, false
	}
	return parsePGMajorMinorInts(strings.TrimSpace(string(out)))
}

// pickNewestPgDump returns the path of the candidate with the highest
// (major, minor), or "" when the list is empty. Pure — unit-tested.
func pickNewestPgDump(cands []pgDumpCandidate) string {
	best := ""
	bestMaj, bestMin := -1, -1
	for _, c := range cands {
		if c.major > bestMaj || (c.major == bestMaj && c.minor > bestMin) {
			best, bestMaj, bestMin = c.path, c.major, c.minor
		}
	}
	return best
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

// parsePGMajorMinorInts is ParsePGMajorMinor as integers, for version
// comparison. Returns ok=false when no version pattern is present.
func parsePGMajorMinorInts(versionLine string) (int, int, bool) {
	m := versionMajorMinorRe.FindStringSubmatch(versionLine)
	if len(m) < 3 {
		return 0, 0, false
	}
	maj, err1 := strconv.Atoi(m[1])
	min, err2 := strconv.Atoi(m[2])
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return maj, min, true
}

var versionMajorMinorRe = regexp.MustCompile(`(\d+)\.(\d+)`)

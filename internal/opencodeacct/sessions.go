package opencodeacct

import (
	"context"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"time"
)

// SessionLocator finds the opencode conversation a session should resume.
// opencode has no pre-assignable session id for a NEW conversation, so we
// look it up from opencode's own session index (`opencode session list
// --format json`), which lives in the shared DB every account uses.
type SessionLocator struct {
	// Binary is the opencode executable; "" → "opencode" on PATH.
	Binary string
	// Timeout bounds the list call; 0 → 15s.
	Timeout time.Duration
}

// listedSession is one row of `opencode session list --format json`.
// created / updated are unix milliseconds.
type listedSession struct {
	ID        string `json:"id"`
	Directory string `json:"directory"`
	Created   int64  `json:"created"`
	Updated   int64  `json:"updated"`
}

// LatestSessionID returns the most recently updated opencode session whose
// directory is workDir, preferring sessions active at/after since (the
// opendray session's start, so a concurrent opencode session in the same
// directory started earlier doesn't win). Falls back to the newest in
// workDir. "" when none / opencode unavailable — callers then spawn fresh.
func (l SessionLocator) LatestSessionID(ctx context.Context, workDir string, since time.Time) string {
	bin := l.Binary
	if bin == "" {
		bin = "opencode"
	}
	timeout := l.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(cctx, bin, "session", "list", "--format", "json")
	cmd.Dir = workDir
	// opencode reads stdin when it isn't a TTY — leave it nil (/dev/null)
	// or the call hangs until the timeout.
	cmd.Stdin = nil
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return pickSession(out, workDir, since)
}

// pickSession is the pure selection half of LatestSessionID.
func pickSession(raw []byte, workDir string, since time.Time) string {
	var rows []listedSession
	if err := json.Unmarshal(raw, &rows); err != nil {
		return ""
	}
	want := cleanDir(workDir)
	sinceMs := int64(0)
	if !since.IsZero() {
		sinceMs = since.UnixMilli()
	}
	var best, bestRecent listedSession
	for _, r := range rows {
		if r.ID == "" || cleanDir(r.Directory) != want {
			continue
		}
		if r.Updated > best.Updated || best.ID == "" {
			best = r
		}
		if sinceMs > 0 && r.Updated >= sinceMs && (r.Updated > bestRecent.Updated || bestRecent.ID == "") {
			bestRecent = r
		}
	}
	if bestRecent.ID != "" {
		return bestRecent.ID
	}
	return best.ID
}

// cleanDir normalizes a directory for comparison, resolving symlinks
// when possible (opencode records the real path).
func cleanDir(d string) string {
	if d == "" {
		return ""
	}
	if r, err := filepath.EvalSymlinks(d); err == nil {
		d = r
	}
	return filepath.Clean(d)
}

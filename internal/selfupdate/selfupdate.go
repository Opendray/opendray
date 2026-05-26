// Package selfupdate is the shared core for opendray's self-update:
// the read-only "what's the latest release" check (used by the CLI and the
// dashboard's About panel) and the on-disk update *request* that the
// unprivileged daemon drops for a privileged systemd oneshot to act on.
//
// Privilege model: the daemon runs as the unprivileged `opendray` user and
// cannot replace /usr/local/bin/opendray, restart the unit, or refresh the
// unit file. So an in-dashboard "Update now" does NOT update anything
// directly — it writes a request file into a daemon-owned directory. A
// root systemd path unit (opendray-selfupdate.path) watches that file and
// activates a root oneshot (opendray-selfupdate.service →
// `opendray self-update --apply`) which downloads the *official* signed
// release, verifies it, swaps the binary, refreshes the unit, and restarts.
// The request's only power is "trigger an upgrade to the official latest" —
// it cannot point the installer at arbitrary code.
package selfupdate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ReleasesAPI is the GitHub releases-latest endpoint for the canonical
// repo. Kept here so the CLI and server agree on the source of truth.
const ReleasesAPI = "https://api.github.com/repos/Opendray/opendray/releases/latest"

// RequestFile is the daemon-owned trigger the privileged oneshot watches.
// It lives under the daemon's writable data dir (ReadWritePaths).
const RequestFile = "selfupdate.request"

// Status is the result of a release check.
type Status struct {
	Current   string `json:"current"`
	Latest    string `json:"latest"`
	Available bool   `json:"updateAvailable"`
	NotesURL  string `json:"notesUrl"`
}

// Request is the on-disk upgrade trigger written by the daemon and consumed
// by the root oneshot. Version is advisory/audit only — the oneshot always
// installs the official latest and verifies it, so a tampered request can
// at most force an upgrade-to-latest, never arbitrary code.
type Request struct {
	Version     string    `json:"version"`
	RequestedBy string    `json:"requestedBy"`
	RequestedAt time.Time `json:"requestedAt"`
	// Force re-installs the latest even when already current (the
	// dashboard's "Re-install" action). Maps to `opendray update --force`.
	Force bool `json:"force,omitempty"`
}

type ghRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

// NormalizeVersion strips a leading "v" so "v2.1.1" and "2.1.1" compare
// equal, and trims any "+build" metadata (custom CT builds like
// "2.1.1+ct128.abc" compare against the release base).
func NormalizeVersion(v string) string {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexByte(v, '+'); i >= 0 {
		v = v[:i]
	}
	return v
}

// Check fetches the latest release and compares it to current.
func Check(ctx context.Context, current string) (Status, error) {
	rel, err := fetchLatest(ctx)
	if err != nil {
		return Status{}, err
	}
	cur := NormalizeVersion(current)
	latest := NormalizeVersion(rel.TagName)
	return Status{
		Current:   cur,
		Latest:    latest,
		Available: latest != "" && latest != cur,
		NotesURL:  rel.HTMLURL,
	}, nil
}

func fetchLatest(ctx context.Context) (*ghRelease, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ReleasesAPI, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "opendray-selfupdate")

	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("github API returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("decode release JSON: %w", err)
	}
	if rel.TagName == "" {
		return nil, fmt.Errorf("github returned an empty tag_name (rate-limited?)")
	}
	return &rel, nil
}

// RequestPath returns the trigger-file path inside dataDir.
func RequestPath(dataDir string) string {
	return filepath.Join(dataDir, RequestFile)
}

// WriteRequest atomically writes the upgrade trigger (temp + rename) so the
// watching path unit never sees a half-written file.
func WriteRequest(dataDir string, r Request) error {
	if r.RequestedAt.IsZero() {
		r.RequestedAt = time.Now().UTC()
	}
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	path := RequestPath(dataDir)
	tmp, err := os.CreateTemp(dataDir, RequestFile+".*")
	if err != nil {
		return fmt.Errorf("create temp request: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// ReadRequest loads and removes the trigger file. Removal is best-effort
// but logged by the caller; the oneshot also rm's it so the path unit
// doesn't re-fire.
func ReadRequest(path string) (Request, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Request{}, err
	}
	var r Request
	if err := json.Unmarshal(b, &r); err != nil {
		return Request{}, fmt.Errorf("parse request: %w", err)
	}
	return r, nil
}

// PendingRequest reports whether a trigger file is already present (so the
// API can avoid stacking duplicate requests).
func PendingRequest(dataDir string) bool {
	_, err := os.Stat(RequestPath(dataDir))
	return err == nil
}

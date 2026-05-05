package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

// RestoreRequest is what the HTTP handler hands to RestoreBackup.
//
// Source is the (encrypted) bundle reader — typically the body of
// a multipart upload. TargetDSN is where to restore to; empty
// means "use opendray's own database" (DANGEROUS, requires the
// double-confirm flow in the UI).
//
// Clean controls whether pg_restore drops existing objects first.
// On a fresh / parallel database leave it false; when restoring
// over the running opendray's own DB you almost always want true.
type RestoreRequest struct {
	Source       io.Reader
	TargetDSN    string
	Clean        bool
	OperatorNote string // free-form audit string from the UI confirm flow
}

// RestoreBackup decrypts the bundle, extracts dump.bin to a temp
// path, validates the manifest's key fingerprint, and replays the
// dump via pg_restore against TargetDSN. Returns a RestoreResult
// summarising what happened (stored only in slog/audit, not DB).
//
// pg_restore is best-effort: missing on PATH → ErrPgRestoreUnavailable.
func (s *Service) RestoreBackup(ctx context.Context, req RestoreRequest) (RestoreResult, error) {
	if s.pgrestore == nil {
		return RestoreResult{}, ErrPgRestoreUnavailable
	}
	if req.Source == nil {
		return RestoreResult{}, errors.New("restore: source reader is nil")
	}

	startedAt := time.Now().UTC()

	tmpDir, err := os.MkdirTemp(s.cfg.LocalDir, ".restore-*")
	if err != nil {
		return RestoreResult{}, fmt.Errorf("restore: tempdir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Decrypt → gunzip → tar → extract.
	plain := s.cipher.Open(req.Source)

	gzr, err := gzip.NewReader(plain)
	if err != nil {
		return RestoreResult{}, fmt.Errorf("restore: gzip: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	var (
		manifest  BundleManifest
		dumpPath  string
		bytesRead int64
	)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return RestoreResult{}, fmt.Errorf("restore: tar: %w", err)
		}
		switch hdr.Name {
		case "manifest.json":
			body, err := io.ReadAll(tr)
			if err != nil {
				return RestoreResult{}, fmt.Errorf("restore: read manifest: %w", err)
			}
			if err := json.Unmarshal(body, &manifest); err != nil {
				return RestoreResult{}, fmt.Errorf("restore: parse manifest: %w", err)
			}
			bytesRead += int64(len(body))
		case "dump.bin":
			dumpPath = filepath.Join(tmpDir, "dump.bin")
			f, err := os.Create(dumpPath)
			if err != nil {
				return RestoreResult{}, fmt.Errorf("restore: create dump tmp: %w", err)
			}
			n, copyErr := io.Copy(f, tr)
			closeErr := f.Close()
			if copyErr != nil {
				return RestoreResult{}, fmt.Errorf("restore: extract dump: %w", copyErr)
			}
			if closeErr != nil {
				return RestoreResult{}, fmt.Errorf("restore: close dump: %w", closeErr)
			}
			bytesRead += n
		default:
			// Discard config.toml and other auxiliaries — they're not
			// applied automatically; operators install them by hand
			// after deciding what to merge.
			n, _ := io.Copy(io.Discard, tr)
			bytesRead += n
		}
	}
	if dumpPath == "" {
		return RestoreResult{}, ErrRestoreNoDump
	}

	// Verify the bundle was encrypted with our running passphrase.
	// cipher.Open would have already failed otherwise, but the
	// fingerprint check gives the operator a clear "wrong key"
	// signal even when the bundle was empty.
	fingerprintOK := manifest.Encryption.Fingerprint == s.cipher.Fingerprint()
	if !fingerprintOK && manifest.Encryption.Fingerprint != "" {
		return RestoreResult{}, fmt.Errorf("%w: bundle=%s server=%s",
			ErrRestoreFingerprintMismatch,
			manifest.Encryption.Fingerprint, s.cipher.Fingerprint())
	}

	targetDSN := req.TargetDSN
	if targetDSN == "" {
		targetDSN = s.dsn
	}

	output, err := s.pgrestore.Restore(ctx, dumpPath, targetDSN, RestoreOptions{
		Clean:             req.Clean,
		SingleTransaction: false, // big dumps would OOM
	})
	finishedAt := time.Now().UTC()
	if err != nil {
		s.log.Warn("restore failed",
			"err", err,
			"output_tail", output,
			"manifest_id", manifest.BackupID)
		return RestoreResult{
			Manifest:        manifest,
			BytesRead:       bytesRead,
			TargetDSNUsed:   redactDSN(targetDSN),
			FingerprintOK:   fingerprintOK,
			PGRestoreOutput: output,
			StartedAt:       startedAt,
			FinishedAt:      finishedAt,
		}, err
	}

	s.log.Info("restore succeeded",
		"manifest_id", manifest.BackupID,
		"bytes", bytesRead,
		"target_dsn", redactDSN(targetDSN),
		"note", req.OperatorNote)

	return RestoreResult{
		Manifest:        manifest,
		BytesRead:       bytesRead,
		TargetDSNUsed:   redactDSN(targetDSN),
		FingerprintOK:   fingerprintOK,
		PGRestoreOutput: output,
		StartedAt:       startedAt,
		FinishedAt:      finishedAt,
	}, nil
}

// PgRestoreVersion exposes pg_restore's version string for the UI
// status banner (parallel to PGVersion for pg_dump). Empty string
// when pg_restore is unavailable — the UI hides the Restore button.
func (s *Service) PgRestoreVersion(ctx context.Context) string {
	if s.pgrestore == nil {
		return ""
	}
	v, _ := s.pgrestore.Version(ctx)
	return v
}

// redactDSN returns a host/db summary of a postgres URL so audit
// logs / API responses don't echo back passwords.
//
//	postgres://user:secret@host:5432/db?x → "host:5432/db"
//	host=h port=5432 dbname=d user=u …    → "host=h port=5432 dbname=d"
func redactDSN(dsn string) string {
	if u, err := url.Parse(dsn); err == nil && u.Scheme != "" {
		host := u.Host
		path := u.Path
		if path == "" {
			path = "/"
		}
		return host + path
	}
	// keyword=value form: strip password-bearing keys.
	parts := splitKeyword(dsn)
	keep := []string{}
	for _, p := range parts {
		if !startsWithAny(p, "password=", "passfile=") {
			keep = append(keep, p)
		}
	}
	return joinSpace(keep)
}

func splitKeyword(s string) []string {
	out := []string{}
	cur := ""
	for _, r := range s {
		if r == ' ' && cur != "" {
			out = append(out, cur)
			cur = ""
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

func startsWithAny(s string, prefixes ...string) bool {
	for _, p := range prefixes {
		if len(s) >= len(p) && s[:len(p)] == p {
			return true
		}
	}
	return false
}

func joinSpace(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += " "
		}
		out += p
	}
	return out
}

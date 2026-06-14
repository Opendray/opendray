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
	"strings"
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
	Source    io.Reader
	TargetDSN string
	Clean     bool
	// Apply commits the restore: it takes a pre-restore safety
	// snapshot, writes config.toml/vault/secrets.env into place and
	// replays the dump via pg_restore. When false (the DEFAULT)
	// RestoreBackup is a DRY RUN — it validates the bundle and reports
	// a RestorePlan but changes nothing on disk or in the database.
	Apply bool
	// Force lets an apply proceed even when the pre-restore safety
	// snapshot fails. Ignored in dry-run.
	Force        bool
	OperatorNote string // free-form audit string from the UI confirm flow
}

// RestoreBackup decrypts the bundle and either reports what a restore
// would do (dry-run, the default) or commits it (Apply).
//
// It is two-phase: every entry is first staged to a temp dir, then the
// manifest fingerprint + dump presence are validated, and only then —
// in Apply mode — are files moved into place and pg_restore run. This
// guarantees a corrupt or wrong-key bundle never leaves a half-written
// config/vault behind. Apply also takes a full_instance safety
// snapshot of the current instance first (see TriggeredPreRestore).
//
// pg_restore is required only for Apply; a dry-run works without it.
func (s *Service) RestoreBackup(ctx context.Context, req RestoreRequest) (RestoreResult, error) {
	if req.Source == nil {
		return RestoreResult{}, errors.New("restore: source reader is nil")
	}
	if req.Apply && s.pgrestore == nil {
		return RestoreResult{}, ErrPgRestoreUnavailable
	}

	startedAt := time.Now().UTC()
	plan := RestorePlan{DryRun: !req.Apply}

	tmpDir, err := os.MkdirTemp(s.cfg.LocalDir, ".restore-*")
	if err != nil {
		return RestoreResult{}, fmt.Errorf("restore: tempdir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	configName := "config.toml"
	if s.configPath != "" {
		configName = filepath.Base(s.configPath)
	}

	// ── Phase 1: decrypt → gunzip → untar, staging each entry to tmp ──
	plain := s.cipher.Open(req.Source)
	gzr, err := gzip.NewReader(plain)
	if err != nil {
		return RestoreResult{}, fmt.Errorf("restore: gzip: %w", err)
	}
	defer gzr.Close()
	tr := tar.NewReader(gzr)

	var (
		manifest                            BundleManifest
		dumpPath, vaultPath, cfgTmp, secTmp string
		bytesRead                           int64
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
			dumpPath, err = stageEntry(tmpDir, "dump.bin", tr, &bytesRead)
			if err != nil {
				return RestoreResult{}, err
			}
		case "vault.tar":
			vaultPath, err = stageEntry(tmpDir, "vault.tar", tr, &bytesRead)
			if err != nil {
				return RestoreResult{}, err
			}
		case "secrets.env":
			secTmp, err = stageEntry(tmpDir, "secrets.env", tr, &bytesRead)
			if err != nil {
				return RestoreResult{}, err
			}
		case configName:
			cfgTmp, err = stageEntry(tmpDir, "config.toml", tr, &bytesRead)
			if err != nil {
				return RestoreResult{}, err
			}
		default:
			n, _ := io.Copy(io.Discard, tr)
			bytesRead += n
		}
	}

	// ── Validate before committing anything ──
	plan.DumpPresent = dumpPath != ""
	if !plan.DumpPresent {
		return RestoreResult{}, ErrRestoreNoDump
	}
	if st, statErr := os.Stat(dumpPath); statErr == nil {
		plan.DumpBytes = st.Size()
	}
	// cipher.Open already authenticates (wrong key → read error above);
	// the fingerprint check just gives a clearer "wrong key" signal.
	fingerprintOK := manifest.Encryption.Fingerprint == s.cipher.Fingerprint()
	if !fingerprintOK && manifest.Encryption.Fingerprint != "" {
		return RestoreResult{}, fmt.Errorf("%w: bundle=%s server=%s",
			ErrRestoreFingerprintMismatch,
			manifest.Encryption.Fingerprint, s.cipher.Fingerprint())
	}
	if vaultPath != "" {
		roots, n, ierr := inspectVaultTar(vaultPath)
		if ierr != nil {
			return RestoreResult{}, fmt.Errorf("restore: inspect vault: %w", ierr)
		}
		plan.VaultRoots, plan.VaultFiles = roots, n
	}
	if cfgTmp != "" {
		plan.ConfigPath = s.configPath
	}
	if secTmp != "" {
		plan.SecretsPath = s.cfg.SecretsFile
	}

	targetDSN := req.TargetDSN
	if targetDSN == "" {
		targetDSN = s.dsn
	}
	result := RestoreResult{
		Manifest:      manifest,
		BytesRead:     bytesRead,
		TargetDSNUsed: redactDSN(targetDSN),
		FingerprintOK: fingerprintOK,
		Plan:          plan,
		StartedAt:     startedAt,
	}

	// ── Dry-run stops here: nothing written, nothing run. ──
	if !req.Apply {
		result.FinishedAt = time.Now().UTC()
		s.log.Info("restore dry-run",
			"manifest_id", manifest.BackupID,
			"vault_files", result.Plan.VaultFiles,
			"config", result.Plan.ConfigPath != "",
			"secrets", result.Plan.SecretsPath != "")
		return result, nil
	}

	// ── Phase 2: commit. From here every return is `result` so its
	// Plan reflects exactly what was committed (safety snapshot id +
	// the Applied list), even on a mid-phase failure. ──
	snapID, snErr := s.preRestoreSnapshot(ctx)
	if snErr != nil {
		if !req.Force {
			result.FinishedAt = time.Now().UTC()
			return result, fmt.Errorf(
				"restore: pre-restore safety snapshot failed (pass force to skip): %w", snErr)
		}
		s.log.Warn("restore: safety snapshot failed; proceeding (force)", "err", snErr)
	}
	result.Plan.SafetySnapshotID = snapID

	if cfgTmp != "" {
		if s.configPath == "" {
			s.log.Warn("restore: bundle has config but no config path configured; skipping")
		} else if err := installFile(cfgTmp, s.configPath); err != nil {
			result.FinishedAt = time.Now().UTC()
			return result, err
		} else {
			result.Plan.Applied = append(result.Plan.Applied, "config")
		}
	}
	if secTmp != "" {
		if s.cfg.SecretsFile == "" {
			s.log.Warn("restore: bundle has secrets.env but no secrets path configured; skipping")
		} else if err := installFile(secTmp, s.cfg.SecretsFile); err != nil {
			result.FinishedAt = time.Now().UTC()
			return result, err
		} else {
			result.Plan.Applied = append(result.Plan.Applied, "secrets.env")
		}
	}
	if vaultPath != "" {
		vf, oerr := os.Open(vaultPath)
		if oerr != nil {
			result.FinishedAt = time.Now().UTC()
			return result, fmt.Errorf("restore: open staged vault: %w", oerr)
		}
		// UnpackVault's count is files ACTUALLY written, which can be
		// fewer than the dry-run inspection if the bundle carries a
		// logical root this instance doesn't map — so we overwrite
		// Plan.VaultFiles deliberately.
		n, uerr := UnpackVault(vf, s.vaultDestFunc())
		if cerr := vf.Close(); cerr != nil && uerr == nil {
			uerr = cerr
		}
		if uerr != nil {
			result.FinishedAt = time.Now().UTC()
			return result, fmt.Errorf("restore: vault: %w", uerr)
		}
		result.Plan.VaultFiles = n
		result.Plan.Applied = append(result.Plan.Applied, "vault")
	}

	output, rerr := s.pgrestore.Restore(ctx, dumpPath, targetDSN, RestoreOptions{
		Clean:             req.Clean,
		SingleTransaction: false, // big dumps would OOM
	})
	result.PGRestoreOutput = output
	result.FinishedAt = time.Now().UTC()
	if rerr != nil {
		s.log.Warn("restore failed", "err", rerr, "output_tail", output, "manifest_id", manifest.BackupID)
		return result, rerr
	}
	result.Plan.Applied = append(result.Plan.Applied, "database")
	s.log.Info("restore succeeded",
		"manifest_id", manifest.BackupID,
		"bytes", bytesRead,
		"target_dsn", redactDSN(targetDSN),
		"applied", result.Plan.Applied,
		"safety_snapshot", snapID,
		"note", req.OperatorNote)
	return result, nil
}

// stageEntry copies the current tar entry to <tmpDir>/<name> and
// returns the path. bytesRead is accumulated for the result summary.
func stageEntry(tmpDir, name string, tr *tar.Reader, bytesRead *int64) (string, error) {
	path := filepath.Join(tmpDir, name)
	f, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("restore: stage %s: %w", name, err)
	}
	n, copyErr := io.Copy(f, tr)
	closeErr := f.Close()
	if copyErr != nil {
		return "", fmt.Errorf("restore: stage %s: %w", name, copyErr)
	}
	if closeErr != nil {
		return "", fmt.Errorf("restore: stage %s: %w", name, closeErr)
	}
	*bytesRead += n
	return path, nil
}

// inspectVaultTar reads a staged vault tar and reports the distinct
// logical roots and total regular-file count, without writing anything.
func inspectVaultTar(path string) (roots []string, files int, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, fmt.Errorf("vaulttar inspect: open: %w", err)
	}
	defer f.Close()
	seen := map[string]bool{}
	tr := tar.NewReader(f)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, 0, fmt.Errorf("vaulttar inspect: read: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		files++
		if logical, _, ok := strings.Cut(filepath.ToSlash(hdr.Name), "/"); ok && !seen[logical] {
			seen[logical] = true
			roots = append(roots, logical)
		}
	}
	return roots, files, nil
}

// installFile copies srcPath onto dst (mode 0600). Any existing dst is
// first renamed aside so a restore is reversible: to dst+".bak", or, if
// that already exists (a prior restore's backup — the last good copy we
// must not clobber), to a timestamped dst+".bak.<unixnano>". If the
// write fails after the rename, the original is rolled back into place
// so a failed restore never leaves the destination missing.
func installFile(srcPath, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return fmt.Errorf("restore: mkdir %s: %w", filepath.Dir(dst), err)
	}

	backedUp := "" // where the prior dst was moved, "" if none
	if _, err := os.Stat(dst); err == nil {
		bak := dst + ".bak"
		if _, err := os.Stat(bak); err == nil {
			bak = fmt.Sprintf("%s.bak.%d", dst, time.Now().UnixNano())
		}
		if err := os.Rename(dst, bak); err != nil {
			return fmt.Errorf("restore: back up existing %s: %w", dst, err)
		}
		backedUp = bak
	}
	rollback := func() {
		if backedUp != "" {
			_ = os.Remove(dst)
			_ = os.Rename(backedUp, dst)
		}
	}

	in, err := os.Open(srcPath)
	if err != nil {
		rollback()
		return fmt.Errorf("restore: open staged %s: %w", srcPath, err)
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		rollback()
		return fmt.Errorf("restore: create %s: %w", dst, err)
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		rollback()
		return fmt.Errorf("restore: write %s: %w", dst, copyErr)
	}
	if closeErr != nil {
		rollback()
		return fmt.Errorf("restore: close %s: %w", dst, closeErr)
	}
	return nil
}

// vaultDestFunc maps a logical vault root (notes/skills/mcp) to its
// configured on-disk destination for UnpackVault.
func (s *Service) vaultDestFunc() func(string) (string, bool) {
	m := make(map[string]string, len(s.cfg.VaultSources))
	for _, v := range s.cfg.VaultSources {
		m[v.Logical] = v.Dir
	}
	return func(logical string) (string, bool) {
		d, ok := m[logical]
		return d, ok && d != ""
	}
}

// preRestoreSnapshot captures the current instance as a full_instance
// backup to the local target before an apply overwrites it.
func (s *Service) preRestoreSnapshot(ctx context.Context) (string, error) {
	if s.targets.get("local") == nil {
		return "", errors.New("no 'local' backup target available")
	}
	b, err := s.RunBackupSync(ctx, RunBackupRequest{
		TargetID:    "local",
		TriggeredBy: TriggeredPreRestore,
		Kind:        KindFullInstance,
	})
	if err != nil {
		return "", err
	}
	if b.Status != BackupSucceeded {
		return b.ID, fmt.Errorf("safety snapshot %s status=%s: %s", b.ID, b.Status, b.Error)
	}
	return b.ID, nil
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

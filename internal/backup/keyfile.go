// Keyfile bridges the backup feature's master passphrase between
// "operator pastes an env var" (the original deployment model) and
// "operator clicks Enable in the admin UI" (the convenience model
// added in PR #49). The two coexist with explicit priority:
//
//	1. OPENDRAY_BACKUP_KEY            — env var, highest priority
//	2. $OPENDRAY_BACKUP_KEY_FILE      — path override env, file content
//	3. ~/.opendray/secrets/backup.key — default file location
//	4. (none — feature disabled)
//
// The default file lives under a dedicated `secrets/` directory at
// the same level as the backup blob output dirs (`backups/`,
// `exports/`), NOT inside them. That means an operator who runs
// `tar -czf snapshot.tar.gz ~/.opendray/backups` to ship a backup
// to cold storage will NOT accidentally include the key — anyone
// who can read the tarball would otherwise be able to decrypt every
// blob inside it. The separation is load-bearing.
//
// The env var path stays first-priority so existing systemd / docker
// deployments that already export OPENDRAY_BACKUP_KEY keep working
// without changes, and so an env-var setup can never be silently
// overridden by a stale file an operator forgot was there.

package backup

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// KeySource records how the passphrase was loaded, so the UI can
// tell the operator "you're running with the env var, the Disable
// button won't help you" or "I'll just remove the file."
type KeySource string

const (
	// KeySourceNone — feature is off, no passphrase available
	// from any of the three sources.
	KeySourceNone KeySource = ""
	// KeySourceEnv — OPENDRAY_BACKUP_KEY env var.
	KeySourceEnv KeySource = "env"
	// KeySourceFile — read from the default file or KEY_FILE override.
	KeySourceFile KeySource = "file"
)

// envKey is the long-standing env var an operator can use to opt
// into backups without ever touching the file-on-disk path. Mirrors
// internal/app/app.go which is the original consumer of the var.
const envKey = "OPENDRAY_BACKUP_KEY"

// envKeyFile lets advanced deployments (systemd LoadCredential,
// docker secrets mount, etc.) point at a custom file location
// rather than dropping the passphrase into the binary's env.
const envKeyFile = "OPENDRAY_BACKUP_KEY_FILE"

// LoadResult bundles the resolved passphrase plus its provenance.
// Callers should treat an empty Passphrase as "feature disabled,
// don't mount handlers."
type LoadResult struct {
	Passphrase string
	Source     KeySource
	// Path is the file the passphrase was loaded from (or would
	// have been loaded from / written to). Always populated even
	// when Source != KeySourceFile, so the UI can show the
	// operator the canonical location.
	Path string
}

// LoadPassphrase walks the priority chain. Returns an empty
// Passphrase + KeySourceNone when no source is configured (this is
// a normal state, not an error — the feature is just off). Returns
// a non-nil error only when a source IS configured but fails to
// read (permissions, malformed file).
func LoadPassphrase() (LoadResult, error) {
	path, err := defaultKeyFilePath()
	if err != nil {
		// Even without a resolvable home dir we still want to
		// honour the env-var path; just leave Path empty so the
		// UI doesn't try to render a misleading location.
		path = ""
	}
	res := LoadResult{Path: path}

	if v := strings.TrimSpace(os.Getenv(envKey)); v != "" {
		res.Passphrase = v
		res.Source = KeySourceEnv
		return res, nil
	}

	overridePath := strings.TrimSpace(os.Getenv(envKeyFile))
	if overridePath != "" {
		// An explicit override path is treated as authoritative —
		// missing file there is a configuration error, not a
		// silent fallback to the default. Operators who set this
		// var intend to use that exact file.
		passphrase, readErr := readKeyFile(overridePath)
		if readErr != nil {
			return res, fmt.Errorf("read %s: %w", envKeyFile, readErr)
		}
		res.Passphrase = passphrase
		res.Source = KeySourceFile
		res.Path = overridePath
		return res, nil
	}

	if path != "" {
		passphrase, readErr := readKeyFile(path)
		if errors.Is(readErr, os.ErrNotExist) {
			return res, nil
		}
		if readErr != nil {
			return res, fmt.Errorf("read default keyfile: %w", readErr)
		}
		res.Passphrase = passphrase
		res.Source = KeySourceFile
		return res, nil
	}

	return res, nil
}

// WriteKeyFile atomically writes the passphrase to the default
// keyfile location, creating parent directories as needed. Mode is
// 0600 on the file and 0700 on the parent dir.
//
// Refuses overwriting an existing file by default — the UI flow
// requires an explicit "rotate" path to avoid accidentally
// orphaning encrypted blobs whose ciphertext can no longer be
// recovered. Pass overwrite=true for rotation.
//
// Returns the canonical path written to so callers can echo it
// back in API responses ("we wrote your key to <path>").
func WriteKeyFile(passphrase string, overwrite bool) (string, error) {
	if strings.TrimSpace(passphrase) == "" {
		return "", errors.New("passphrase is empty")
	}
	path, err := defaultKeyFilePath()
	if err != nil {
		return "", fmt.Errorf("resolve key file path: %w", err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create secrets dir: %w", err)
	}
	// Tighten parent dir permissions even if it pre-existed with
	// looser ones (e.g. the operator made it themselves). 0700 is
	// the contract.
	if err := os.Chmod(dir, 0o700); err != nil {
		return "", fmt.Errorf("chmod secrets dir: %w", err)
	}

	if !overwrite {
		if _, statErr := os.Stat(path); statErr == nil {
			return "", fmt.Errorf("key file already exists at %s; use rotate flow to replace", path)
		} else if !errors.Is(statErr, os.ErrNotExist) {
			return "", fmt.Errorf("stat key file: %w", statErr)
		}
	}

	// Atomic write: tempfile in the same dir, fsync, rename. The
	// rename is atomic on the same filesystem so the file is
	// either fully there or absent — never half-written.
	tmp, err := os.CreateTemp(dir, ".backup.key.tmp-*")
	if err != nil {
		return "", fmt.Errorf("create temp key file: %w", err)
	}
	tmpPath := tmp.Name()
	// Best-effort cleanup if anything below this point fails.
	defer func() { _ = os.Remove(tmpPath) }()

	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return "", fmt.Errorf("chmod temp key file: %w", err)
	}
	if _, err := tmp.WriteString(passphrase + "\n"); err != nil {
		_ = tmp.Close()
		return "", fmt.Errorf("write temp key file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return "", fmt.Errorf("fsync temp key file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("close temp key file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return "", fmt.Errorf("rename temp key file into place: %w", err)
	}
	return path, nil
}

// RemoveKeyFile deletes the default keyfile. No-op when the file is
// already absent. Does NOT touch env-var-set passphrases — those
// live in the parent process's environment and the binary can't
// unset them.
func RemoveKeyFile() error {
	path, err := defaultKeyFilePath()
	if err != nil {
		return fmt.Errorf("resolve key file path: %w", err)
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove key file: %w", err)
	}
	return nil
}

// DefaultKeyFilePath exposes the canonical location for the UI to
// display ("your key file will be saved to ~/...") without having
// to read or write it.
func DefaultKeyFilePath() (string, error) {
	return defaultKeyFilePath()
}

func defaultKeyFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".opendray", "secrets", "backup.key"), nil
}

func readKeyFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	// Trim trailing newline + any stray whitespace from
	// hand-edited files. Empty after trim is a corruption
	// signal — better to fail loudly than silently disable.
	s := strings.TrimSpace(string(b))
	if s == "" {
		return "", fmt.Errorf("key file %s is empty after trim", path)
	}
	return s, nil
}

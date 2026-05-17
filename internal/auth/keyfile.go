// Keyfile lets the admin UI rotate the operator's username +
// password without touching config.toml — mirrors the
// backup/keyfile pattern introduced in PR #49 (load priority
// env > KEY_FILE > default file > config.toml, atomic 0600
// writes, separate secrets/ dir so a tar of the data dir doesn't
// leak credentials).
//
// On-disk schema is a single JSON object:
//
//	{"user":"admin","password_hash":"$2a$..."}
//
// Stored at ~/.opendray/secrets/admin.key with mode 0600 and
// parent dir mode 0700. password_hash is bcrypt at the standard
// cost; the value never reaches the wire after the operator
// submits the change.
//
// The env-var path (cfg.Admin.Password) stays plaintext for now;
// rotating that out is a separate breaking-change conversation
// (it would force every existing operator to do a one-shot UI
// setup before they can authenticate post-upgrade).

package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// envOverride lets advanced deployments (docker secrets, systemd
// LoadCredential, etc.) point at a custom file location without
// dropping plaintext into the binary's env.
const envOverride = "OPENDRAY_ADMIN_KEY_FILE"

// CredSource records how active credentials were loaded so the UI
// can decide whether the keyfile-write flow is meaningful. When
// the value is CredSourceEnv we don't refuse changes outright —
// the operator may still want to write a keyfile to take effect
// next restart — but the UI surfaces a warning that the env-var
// path remains authoritative until they unset it.
type CredSource string

const (
	CredSourceNone   CredSource = ""
	CredSourceConfig CredSource = "config"
	CredSourceFile   CredSource = "file"
)

// AdminCreds is the canonical in-memory shape used at runtime.
// Either PasswordHash OR PasswordPlaintext is populated depending
// on Source: file source carries a bcrypt hash, config source
// carries the original plaintext (for backward compatibility with
// existing config.toml deployments).
//
// Login uses these via auth.Service.verify; nothing else reads
// them.
type AdminCreds struct {
	User              string
	PasswordHash      string
	PasswordPlaintext string
	Source            CredSource
}

// keyFilePayload is the on-disk shape. Kept distinct from
// AdminCreds so the (de)serialiser doesn't accidentally pick up
// runtime-only fields.
type keyFilePayload struct {
	User         string `json:"user"`
	PasswordHash string `json:"password_hash"`
}

// LoadCreds walks the priority chain (env file override → default
// file → caller-supplied config fallback). Returns CredSourceNone
// + empty struct (NOT an error) when nothing is configured;
// callers decide what that means (auth.Service rejects all
// logins).
//
// configUser / configPassword are passed in from cfg.Admin so this
// package doesn't have to import internal/config.
func LoadCreds(configUser, configPassword string) (AdminCreds, error) {
	// 1. Explicit file override via env. An empty file there is an
	//    error (operator intended to use it).
	if path := strings.TrimSpace(os.Getenv(envOverride)); path != "" {
		creds, err := readKeyFile(path)
		if err != nil {
			return AdminCreds{}, fmt.Errorf("read %s: %w", envOverride, err)
		}
		return creds, nil
	}

	// 2. Default file location. Missing file → fall through;
	//    anything else (corrupt JSON, permissions) is loud.
	if path, err := DefaultKeyFilePath(); err == nil {
		creds, rerr := readKeyFile(path)
		if rerr == nil {
			return creds, nil
		}
		if !errors.Is(rerr, os.ErrNotExist) {
			return AdminCreds{}, fmt.Errorf("read default admin keyfile: %w", rerr)
		}
	}

	// 3. Config fallback. Both empty = feature disabled (matches
	//    auth.Service behaviour from before this file existed).
	if configUser == "" || configPassword == "" {
		return AdminCreds{}, nil
	}
	return AdminCreds{
		User:              configUser,
		PasswordPlaintext: configPassword,
		Source:            CredSourceConfig,
	}, nil
}

// DefaultKeyFilePath surfaces the canonical location so the UI can
// echo it back ("we'll write your new credentials to <path>")
// without duplicating the join logic.
func DefaultKeyFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".opendray", "secrets", "admin.key"), nil
}

// WriteKeyFile atomically writes a new user + password_hash pair
// to the default location. password is hashed here, never stored
// plaintext anywhere. Returns the canonical path on success so the
// caller can include it in the API response.
//
// Overwrite is always allowed for credentials (unlike the backup
// keyfile's refuse-overwrite default) — rotating an admin password
// is a normal operation, while rotating a backup key risks
// orphaning encrypted blobs.
func WriteKeyFile(user, password string) (string, error) {
	user = strings.TrimSpace(user)
	if user == "" {
		return "", errors.New("user is empty")
	}
	if len(password) < MinPasswordLen {
		return "", fmt.Errorf("password must be at least %d characters", MinPasswordLen)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	payload := keyFilePayload{User: user, PasswordHash: string(hash)}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}

	path, err := DefaultKeyFilePath()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create secrets dir: %w", err)
	}
	// Tighten perms even if the dir pre-existed.
	if err := os.Chmod(dir, 0o700); err != nil {
		return "", fmt.Errorf("chmod secrets dir: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".admin.key.tmp-*")
	if err != nil {
		return "", fmt.Errorf("create temp keyfile: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return "", fmt.Errorf("chmod temp keyfile: %w", err)
	}
	if _, err := tmp.Write(body); err != nil {
		_ = tmp.Close()
		return "", fmt.Errorf("write temp keyfile: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return "", fmt.Errorf("fsync temp keyfile: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("close temp keyfile: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return "", fmt.Errorf("rename temp keyfile: %w", err)
	}
	return path, nil
}

// MinPasswordLen is the floor enforced by both WriteKeyFile and
// the /auth/change-credentials handler. Eight is the operator-
// chosen minimum: low enough that an existing memorable password
// usually clears it, high enough to keep trivial guesses out.
// bcrypt covers the rest of the entropy story.
const MinPasswordLen = 8

func readKeyFile(path string) (AdminCreds, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return AdminCreds{}, err
	}
	var payload keyFilePayload
	if err := json.Unmarshal(b, &payload); err != nil {
		return AdminCreds{}, fmt.Errorf("parse keyfile %s: %w", path, err)
	}
	user := strings.TrimSpace(payload.User)
	hash := strings.TrimSpace(payload.PasswordHash)
	if user == "" || hash == "" {
		return AdminCreds{}, fmt.Errorf("keyfile %s missing user or password_hash", path)
	}
	return AdminCreds{
		User:         user,
		PasswordHash: hash,
		Source:       CredSourceFile,
	}, nil
}

// VerifyPassword runs constant-time on plaintext + checks against
// the active credentials. Returns true only when both user and
// password match. Used by both Login and the change-credentials
// handler's "verify current" step.
func (c AdminCreds) VerifyPassword(user, password string) bool {
	if c.User == "" || user != c.User {
		// Run a dummy bcrypt comparison anyway to avoid leaking
		// "user exists / doesn't exist" via timing. Cost matches
		// bcrypt.DefaultCost so the operation duration is
		// indistinguishable from a real verify.
		_ = bcrypt.CompareHashAndPassword(
			[]byte("$2a$10$3VPyywpwKPbWoaaIFTpWZ.OE3hrKwgZdvDt7yPMtRSXY7WhgVoP4G"),
			[]byte(password),
		)
		return false
	}
	switch c.Source {
	case CredSourceFile:
		err := bcrypt.CompareHashAndPassword(
			[]byte(c.PasswordHash), []byte(password),
		)
		return err == nil
	case CredSourceConfig:
		return constantTimeEqual(c.PasswordPlaintext, password)
	default:
		return false
	}
}

// constantTimeEqual is a thin wrapper around subtle.ConstantTimeCompare
// that handles unequal-length inputs without a length-based
// short-circuit (subtle.ConstantTimeCompare itself short-circuits
// on length mismatch, leaking the length oracle).
func constantTimeEqual(a, b string) bool {
	ab, bb := []byte(a), []byte(b)
	if len(ab) != len(bb) {
		// Still run the compare with the shorter padded so timing
		// stays roughly constant. Returning false unconditionally
		// is fine here — the timing channel narrowing matters far
		// more than the algorithmic correctness of the dummy op.
		dummy := make([]byte, len(ab))
		_ = constantTimeCompareBytes(dummy, ab)
		return false
	}
	return constantTimeCompareBytes(ab, bb)
}

// constantTimeCompareBytes is broken out so it can be inlined into
// the dummy-compare path above without dragging in subtle's
// length-mismatch branch.
func constantTimeCompareBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := range a {
		v |= a[i] ^ b[i]
	}
	return v == 0
}

package backup

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// The backup encryption key is deliberately never stored inside a
// backup (so shipping a backup dir can't leak it). That makes the
// passphrase a single point of failure: lose it and every encrypted
// backup is unrecoverable. A Recovery Kit closes that gap WITHOUT
// weakening the backups: it is the backup passphrase wrapped under a
// SEPARATE recovery passphrase the operator chooses and stores
// out-of-band (password manager, printout in a safe, etc.). To recover
// a dead host you need both the kit file and the recovery passphrase.

// RecoveryKitVersion is the on-disk kit format version.
const RecoveryKitVersion = 1

// MinRecoveryPassphraseLen is the floor for a recovery passphrase. The
// kit is only as strong as this secret, so we refuse trivially short
// ones.
const MinRecoveryPassphraseLen = 8

// ErrRecoveryPassphraseTooShort is returned by ExportRecoveryKit when
// the recovery passphrase is below MinRecoveryPassphraseLen.
var ErrRecoveryPassphraseTooShort = fmt.Errorf(
	"recovery: recovery passphrase must be at least %d characters", MinRecoveryPassphraseLen)

// ErrRecoveryKitVersion is returned when a kit's version is unknown.
var ErrRecoveryKitVersion = errors.New("recovery: unsupported recovery kit version")

// recoveryKit is the serialized Recovery Kit. WrappedKey is the backup
// passphrase encrypted (AES-GCM, "v1:" envelope) under a key derived
// from the recovery passphrase. KeyFingerprint identifies WHICH backup
// key this unlocks so an operator can match a kit to their backups
// without decrypting it.
type recoveryKit struct {
	Version        int    `json:"version"`
	CreatedAt      string `json:"created_at"`
	KeyFingerprint string `json:"key_fingerprint"`
	WrappedKey     string `json:"wrapped_key"`
}

// ExportRecoveryKit wraps backupPassphrase under recoveryPassphrase and
// returns a serialized kit. The two passphrases are independent: the
// recovery passphrase is what the operator must keep to use the kit.
//
// keyFingerprint is the BACKUP key's fingerprint (Cipher.Fingerprint),
// recorded so a kit can be matched against backups[].key_fingerprint.
// It is passed in rather than re-derived here so callers that already
// hold the backup Cipher don't pay a second 200k-round PBKDF2.
func ExportRecoveryKit(backupPassphrase, recoveryPassphrase, keyFingerprint string, now time.Time) ([]byte, error) {
	if backupPassphrase == "" {
		return nil, ErrCipherUnconfigured
	}
	if len(recoveryPassphrase) < MinRecoveryPassphraseLen {
		return nil, ErrRecoveryPassphraseTooShort
	}
	recCipher, err := NewCipher(recoveryPassphrase)
	if err != nil {
		return nil, fmt.Errorf("recovery: %w", err)
	}
	wrapped, err := recCipher.EncryptField(backupPassphrase)
	if err != nil {
		return nil, fmt.Errorf("recovery: wrap key: %w", err)
	}
	kit := recoveryKit{
		Version:        RecoveryKitVersion,
		CreatedAt:      now.UTC().Format(time.RFC3339),
		KeyFingerprint: keyFingerprint,
		WrappedKey:     wrapped,
	}
	body, err := json.MarshalIndent(kit, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("recovery: marshal kit: %w", err)
	}
	return body, nil
}

// ImportRecoveryKit reverses ExportRecoveryKit: given the kit bytes and
// the recovery passphrase, it returns the original backup passphrase.
// A wrong recovery passphrase surfaces as ErrCipherWrongKey.
func ImportRecoveryKit(kitBytes []byte, recoveryPassphrase string) (string, error) {
	var kit recoveryKit
	if err := json.Unmarshal(kitBytes, &kit); err != nil {
		return "", fmt.Errorf("recovery: parse kit: %w", err)
	}
	if kit.Version != RecoveryKitVersion {
		return "", fmt.Errorf("%w: %d", ErrRecoveryKitVersion, kit.Version)
	}
	if kit.WrappedKey == "" {
		return "", errors.New("recovery: kit has no wrapped key")
	}
	recCipher, err := NewCipher(recoveryPassphrase)
	if err != nil {
		return "", fmt.Errorf("recovery: %w", err)
	}
	pass, err := recCipher.DecryptField(kit.WrappedKey)
	if err != nil {
		return "", fmt.Errorf("recovery: unwrap key: %w", err)
	}
	return pass, nil
}

// RecoveryKitFingerprint reports the backup key fingerprint a kit
// unlocks, without needing the recovery passphrase — for the UI to show
// "this kit matches backups under key abcd…".
func RecoveryKitFingerprint(kitBytes []byte) (string, error) {
	var kit recoveryKit
	if err := json.Unmarshal(kitBytes, &kit); err != nil {
		return "", fmt.Errorf("recovery: parse kit: %w", err)
	}
	return kit.KeyFingerprint, nil
}

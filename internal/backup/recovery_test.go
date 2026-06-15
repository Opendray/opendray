package backup

import (
	"errors"
	"testing"
	"time"
)

// fpFor returns the backup key fingerprint for a passphrase (what
// callers pass to ExportRecoveryKit).
func fpFor(t *testing.T, pass string) string {
	t.Helper()
	c, err := NewCipher(pass)
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}
	return c.Fingerprint()
}

func TestRecoveryKit_Roundtrip(t *testing.T) {
	const backupPass = "the-backup-passphrase-xyz"
	const recoveryPass = "operator-recovery-secret-1"

	kit, err := ExportRecoveryKit(backupPass, recoveryPass, fpFor(t, backupPass), time.Unix(0, 0))
	if err != nil {
		t.Fatalf("ExportRecoveryKit: %v", err)
	}

	got, err := ImportRecoveryKit(kit, recoveryPass)
	if err != nil {
		t.Fatalf("ImportRecoveryKit: %v", err)
	}
	if got != backupPass {
		t.Errorf("recovered passphrase = %q, want %q", got, backupPass)
	}
}

func TestRecoveryKit_FingerprintMatchesBackupKey(t *testing.T) {
	const backupPass = "the-backup-passphrase-xyz"
	want := fpFor(t, backupPass)
	kit, err := ExportRecoveryKit(backupPass, "operator-recovery-secret-1", want, time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	fp, err := RecoveryKitFingerprint(kit)
	if err != nil {
		t.Fatal(err)
	}
	if fp != want {
		t.Errorf("kit fingerprint = %q, want %q", fp, want)
	}
}

func TestImportRecoveryKit_WrongPassphraseFails(t *testing.T) {
	const backupPass = "the-backup-passphrase-xyz"
	kit, err := ExportRecoveryKit(backupPass, "operator-recovery-secret-1", fpFor(t, backupPass), time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	_, err = ImportRecoveryKit(kit, "the-wrong-recovery-pass")
	if err == nil {
		t.Fatal("expected wrong recovery passphrase to fail")
	}
	if !errors.Is(err, ErrCipherWrongKey) {
		t.Errorf("want ErrCipherWrongKey, got %v", err)
	}
}

func TestExportRecoveryKit_Validation(t *testing.T) {
	if _, err := ExportRecoveryKit("", "operator-recovery-secret-1", "fp", time.Unix(0, 0)); !errors.Is(err, ErrCipherUnconfigured) {
		t.Errorf("empty backup passphrase: want ErrCipherUnconfigured, got %v", err)
	}
	if _, err := ExportRecoveryKit("backup-pass", "short", "fp", time.Unix(0, 0)); !errors.Is(err, ErrRecoveryPassphraseTooShort) {
		t.Errorf("short recovery passphrase: want ErrRecoveryPassphraseTooShort, got %v", err)
	}
}

func TestImportRecoveryKit_BadInput(t *testing.T) {
	if _, err := ImportRecoveryKit([]byte("not json"), "operator-recovery-secret-1"); err == nil {
		t.Error("expected parse error for non-JSON kit")
	}
	if _, err := ImportRecoveryKit([]byte(`{"version":99,"wrapped_key":"x"}`), "operator-recovery-secret-1"); !errors.Is(err, ErrRecoveryKitVersion) {
		t.Errorf("unknown version: want ErrRecoveryKitVersion, got %v", err)
	}
	if _, err := ImportRecoveryKit([]byte(`{"version":1}`), "operator-recovery-secret-1"); err == nil {
		t.Error("expected error for kit with no wrapped key")
	}
}

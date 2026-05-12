package auth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// All tests isolate themselves with t.TempDir + t.Setenv("HOME",
// ...) so they never touch the real ~/.opendray/secrets path.

func setTempHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv(envOverride, "")
	return dir
}

func TestLoadCreds_None(t *testing.T) {
	setTempHome(t)
	got, err := LoadCreds("", "")
	if err != nil {
		t.Fatalf("LoadCreds: %v", err)
	}
	if got.Source != CredSourceNone || got.User != "" {
		t.Fatalf("expected empty creds, got %+v", got)
	}
}

func TestLoadCreds_ConfigFallback(t *testing.T) {
	setTempHome(t)
	got, err := LoadCreds("admin", "secret-from-config")
	if err != nil {
		t.Fatalf("LoadCreds: %v", err)
	}
	if got.Source != CredSourceConfig {
		t.Fatalf("expected CredSourceConfig, got %q", got.Source)
	}
	if got.User != "admin" || got.PasswordPlaintext != "secret-from-config" {
		t.Fatalf("unexpected creds: %+v", got)
	}
}

func TestLoadCreds_FileBeatsConfig(t *testing.T) {
	setTempHome(t)
	if _, err := WriteKeyFile("from-file-user", "from-file-passphrase!!"); err != nil {
		t.Fatalf("WriteKeyFile: %v", err)
	}
	got, err := LoadCreds("config-user", "config-pass")
	if err != nil {
		t.Fatalf("LoadCreds: %v", err)
	}
	if got.Source != CredSourceFile {
		t.Fatalf("expected file source, got %q", got.Source)
	}
	if got.User != "from-file-user" {
		t.Fatalf("expected user from file, got %q", got.User)
	}
	if got.PasswordPlaintext != "" {
		t.Fatalf("file source should not carry plaintext")
	}
}

func TestLoadCreds_EnvOverrideBeatsDefault(t *testing.T) {
	setTempHome(t)
	customDir := t.TempDir()
	custom := filepath.Join(customDir, "my-admin.key")
	// Use WriteKeyFile-style payload but at a custom path.
	hash, _ := bcrypt.GenerateFromPassword([]byte("a-long-enough-passphrase"), bcrypt.MinCost)
	body := `{"user":"env-user","password_hash":"` + string(hash) + `"}`
	if err := os.WriteFile(custom, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(envOverride, custom)

	got, err := LoadCreds("config-user", "config-pass")
	if err != nil {
		t.Fatalf("LoadCreds: %v", err)
	}
	if got.User != "env-user" {
		t.Fatalf("expected env-overridden user, got %q", got.User)
	}
}

func TestLoadCreds_EnvOverrideMissingIsError(t *testing.T) {
	setTempHome(t)
	t.Setenv(envOverride, "/nonexistent/path.key")
	if _, err := LoadCreds("", ""); err == nil {
		t.Fatal("expected error when KEY_FILE override points to missing file")
	}
}

func TestWriteKeyFile_PermsAndAtomicity(t *testing.T) {
	setTempHome(t)
	path, err := WriteKeyFile("operator", "a-suitably-long-passphrase")
	if err != nil {
		t.Fatalf("WriteKeyFile: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("file perms: got %#o want 0600", mode)
	}
	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if mode := dirInfo.Mode().Perm(); mode != 0o700 {
		t.Errorf("dir perms: got %#o want 0700", mode)
	}
	// No stray tempfile after a successful write.
	entries, _ := os.ReadDir(filepath.Dir(path))
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".admin.key.tmp") {
			t.Errorf("stray tempfile: %s", e.Name())
		}
	}
}

func TestWriteKeyFile_AllowsOverwrite(t *testing.T) {
	setTempHome(t)
	if _, err := WriteKeyFile("u1", "first-passphrase-long!!"); err != nil {
		t.Fatalf("first write: %v", err)
	}
	if _, err := WriteKeyFile("u2", "rotated-passphrase-yes!"); err != nil {
		t.Fatalf("rotation: %v", err)
	}
	// Verify the rotated user is what gets loaded.
	got, err := LoadCreds("", "")
	if err != nil {
		t.Fatal(err)
	}
	if got.User != "u2" {
		t.Fatalf("expected rotated user, got %q", got.User)
	}
}

func TestWriteKeyFile_RejectsShortPassword(t *testing.T) {
	setTempHome(t)
	if _, err := WriteKeyFile("u", "short"); err == nil {
		t.Fatal("expected short password to be rejected")
	}
}

func TestWriteKeyFile_RejectsEmptyUser(t *testing.T) {
	setTempHome(t)
	if _, err := WriteKeyFile("  ", "a-long-enough-passphrase!"); err == nil {
		t.Fatal("expected empty user to be rejected")
	}
}

func TestAdminCreds_VerifyPassword(t *testing.T) {
	setTempHome(t)
	// File-source creds (bcrypt).
	if _, err := WriteKeyFile("operator", "the-real-passphrase-1234"); err != nil {
		t.Fatal(err)
	}
	creds, _ := LoadCreds("", "")
	if !creds.VerifyPassword("operator", "the-real-passphrase-1234") {
		t.Error("correct user+password should verify")
	}
	if creds.VerifyPassword("operator", "wrong") {
		t.Error("wrong password should fail")
	}
	if creds.VerifyPassword("intruder", "the-real-passphrase-1234") {
		t.Error("wrong user should fail")
	}

	// Config-source creds (plaintext compare).
	confCreds := AdminCreds{
		User:              "admin",
		PasswordPlaintext: "config-pass",
		Source:            CredSourceConfig,
	}
	if !confCreds.VerifyPassword("admin", "config-pass") {
		t.Error("config creds correct should verify")
	}
	if confCreds.VerifyPassword("admin", "wrong") {
		t.Error("config creds wrong password should fail")
	}
}

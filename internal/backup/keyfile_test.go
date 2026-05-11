package backup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Each test isolates itself by pointing HOME at a fresh temp dir;
// without that the default keyfile location would either leak the
// real user's key into the test or write into the real ~/.opendray.

func setTempHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	// Clear the two env-var inputs so individual tests can opt
	// them back in deterministically.
	t.Setenv(envKey, "")
	t.Setenv(envKeyFile, "")
	return dir
}

func TestLoadPassphrase_None(t *testing.T) {
	setTempHome(t)

	got, err := LoadPassphrase()
	if err != nil {
		t.Fatalf("LoadPassphrase returned err on empty env+disk: %v", err)
	}
	if got.Passphrase != "" {
		t.Fatalf("expected empty passphrase, got %q", got.Passphrase)
	}
	if got.Source != KeySourceNone {
		t.Fatalf("expected KeySourceNone, got %q", got.Source)
	}
	if !strings.HasSuffix(got.Path, "/.opendray/secrets/backup.key") {
		t.Fatalf("expected default path under fake home, got %q", got.Path)
	}
}

func TestLoadPassphrase_EnvVarWins(t *testing.T) {
	setTempHome(t)
	t.Setenv(envKey, "from-env-var-very-long-value-here")
	// Also write a file — env must win.
	written, err := WriteKeyFile("from-file-also-long-enough-yes", false)
	if err != nil {
		t.Fatalf("WriteKeyFile: %v", err)
	}
	if _, err := os.Stat(written); err != nil {
		t.Fatalf("written file missing: %v", err)
	}

	got, err := LoadPassphrase()
	if err != nil {
		t.Fatalf("LoadPassphrase: %v", err)
	}
	if got.Passphrase != "from-env-var-very-long-value-here" {
		t.Fatalf("expected env value, got %q", got.Passphrase)
	}
	if got.Source != KeySourceEnv {
		t.Fatalf("expected KeySourceEnv, got %q", got.Source)
	}
}

func TestLoadPassphrase_FileOverridePath(t *testing.T) {
	setTempHome(t)
	custom := filepath.Join(t.TempDir(), "my-key")
	if err := os.WriteFile(custom, []byte("explicit-override-passphrase\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(envKeyFile, custom)

	got, err := LoadPassphrase()
	if err != nil {
		t.Fatalf("LoadPassphrase: %v", err)
	}
	if got.Passphrase != "explicit-override-passphrase" {
		t.Fatalf("expected override value, got %q", got.Passphrase)
	}
	if got.Source != KeySourceFile {
		t.Fatalf("expected KeySourceFile, got %q", got.Source)
	}
	if got.Path != custom {
		t.Fatalf("expected Path to reflect override, got %q want %q", got.Path, custom)
	}
}

func TestLoadPassphrase_FileOverrideMissingIsError(t *testing.T) {
	setTempHome(t)
	t.Setenv(envKeyFile, "/nonexistent/path/that/should/not/exist.key")

	_, err := LoadPassphrase()
	if err == nil {
		t.Fatal("expected error when KEY_FILE override points to missing file")
	}
}

func TestLoadPassphrase_DefaultFile(t *testing.T) {
	setTempHome(t)
	written, err := WriteKeyFile("disk-loaded-passphrase-here-32+", false)
	if err != nil {
		t.Fatalf("WriteKeyFile: %v", err)
	}

	got, err := LoadPassphrase()
	if err != nil {
		t.Fatalf("LoadPassphrase: %v", err)
	}
	if got.Passphrase != "disk-loaded-passphrase-here-32+" {
		t.Fatalf("expected file passphrase, got %q", got.Passphrase)
	}
	if got.Source != KeySourceFile {
		t.Fatalf("expected KeySourceFile, got %q", got.Source)
	}
	if got.Path != written {
		t.Fatalf("Path mismatch: got %q want %q", got.Path, written)
	}
}

func TestWriteKeyFile_PermsAndAtomicity(t *testing.T) {
	setTempHome(t)
	path, err := WriteKeyFile("a-suitably-long-passphrase-for-testing", false)
	if err != nil {
		t.Fatalf("WriteKeyFile: %v", err)
	}

	// File perms must be 0600 — anyone else on the box reading the
	// key would defeat the whole encryption story.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("file perms: got %#o want 0600", mode)
	}
	// Parent dir 0700 for the same reason — a 0755 dir
	// containing a 0600 file is fine in theory but tightening
	// the dir is defence in depth.
	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if mode := dirInfo.Mode().Perm(); mode != 0o700 {
		t.Errorf("dir perms: got %#o want 0700", mode)
	}

	// No stray tempfile from the atomic write — defer-cleanup
	// only fires on error, so a successful write should leave a
	// pristine secrets dir.
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".backup.key.tmp") {
			t.Errorf("stray tempfile left behind: %s", e.Name())
		}
	}
}

func TestWriteKeyFile_RefusesOverwrite(t *testing.T) {
	setTempHome(t)
	if _, err := WriteKeyFile("first-passphrase-long-enough-here", false); err != nil {
		t.Fatalf("WriteKeyFile first call: %v", err)
	}
	if _, err := WriteKeyFile("second-passphrase-long-enough-no", false); err == nil {
		t.Fatal("expected second WriteKeyFile to refuse")
	}
}

func TestWriteKeyFile_OverwriteWhenForced(t *testing.T) {
	setTempHome(t)
	if _, err := WriteKeyFile("first-passphrase-long-enough-here", false); err != nil {
		t.Fatalf("WriteKeyFile first call: %v", err)
	}
	path, err := WriteKeyFile("rotated-passphrase-also-long-and-fresh", true)
	if err != nil {
		t.Fatalf("overwrite should succeed when force=true: %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(b)); got != "rotated-passphrase-also-long-and-fresh" {
		t.Fatalf("file content not rotated: got %q", got)
	}
}

func TestWriteKeyFile_EmptyPassphraseRejected(t *testing.T) {
	setTempHome(t)
	if _, err := WriteKeyFile("   ", false); err == nil {
		t.Fatal("expected empty passphrase to be rejected")
	}
}

func TestRemoveKeyFile_NoErrorWhenAbsent(t *testing.T) {
	setTempHome(t)
	if err := RemoveKeyFile(); err != nil {
		t.Fatalf("RemoveKeyFile on empty dir should be a no-op: %v", err)
	}
}

func TestRemoveKeyFile_Removes(t *testing.T) {
	setTempHome(t)
	path, err := WriteKeyFile("a-suitably-long-passphrase-for-testing", false)
	if err != nil {
		t.Fatalf("WriteKeyFile: %v", err)
	}
	if err := RemoveKeyFile(); err != nil {
		t.Fatalf("RemoveKeyFile: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected file removed, got stat err=%v", err)
	}
}

func TestReadKeyFile_TrimsAndRejectsEmpty(t *testing.T) {
	dir := t.TempDir()

	// Trailing newline + whitespace stripped.
	withWS := filepath.Join(dir, "ws.key")
	if err := os.WriteFile(withWS, []byte("  hello-world  \n\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := readKeyFile(withWS)
	if err != nil {
		t.Fatalf("readKeyFile: %v", err)
	}
	if got != "hello-world" {
		t.Fatalf("got %q, want %q", got, "hello-world")
	}

	// Pure whitespace → error, not silent empty success.
	empty := filepath.Join(dir, "empty.key")
	if err := os.WriteFile(empty, []byte("   \n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readKeyFile(empty); err == nil {
		t.Fatal("expected whitespace-only file to error")
	}
}

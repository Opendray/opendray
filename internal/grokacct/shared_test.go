package grokacct

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureSharedAssets_SymlinksHeavyDirs(t *testing.T) {
	src := t.TempDir()
	home := t.TempDir()
	// Shareable, account-independent install/cache dirs in the source.
	for _, d := range []string{"bin", "vendor", "bundled"} {
		if err := os.MkdirAll(filepath.Join(src, d), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	// A per-account file that must NEVER be shared.
	if err := os.WriteFile(filepath.Join(src, "auth.json"), []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := ensureSharedAssets(home, src); err != nil {
		t.Fatalf("ensureSharedAssets: %v", err)
	}

	for _, d := range []string{"bin", "vendor", "bundled"} {
		p := filepath.Join(home, d)
		fi, err := os.Lstat(p)
		if err != nil {
			t.Fatalf("%s not created: %v", d, err)
		}
		if fi.Mode()&os.ModeSymlink == 0 {
			t.Errorf("%s should be a symlink", d)
		}
		target, _ := os.Readlink(p)
		if target != filepath.Join(src, d) {
			t.Errorf("%s -> %q, want %q", d, target, filepath.Join(src, d))
		}
	}
	// auth.json is per-account: it must NOT be symlinked into the home.
	if _, err := os.Lstat(filepath.Join(home, "auth.json")); err == nil {
		t.Error("auth.json must never be shared into an account home")
	}
}

func TestEnsureSharedAssets_LeavesExistingUntouched(t *testing.T) {
	src := t.TempDir()
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(src, "bin"), 0o700); err != nil {
		t.Fatal(err)
	}
	// The account already has its own real bin/ — don't clobber it.
	realBin := filepath.Join(home, "bin")
	if err := os.MkdirAll(realBin, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(realBin, "marker"), []byte("mine"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := ensureSharedAssets(home, src); err != nil {
		t.Fatal(err)
	}

	fi, _ := os.Lstat(realBin)
	if fi.Mode()&os.ModeSymlink != 0 {
		t.Error("existing bin/ must not be replaced by a symlink")
	}
	if _, err := os.Stat(filepath.Join(realBin, "marker")); err != nil {
		t.Error("existing bin/ contents were lost")
	}
}

func TestEnsureSharedAssets_NoopWhenSrcMissingOrSame(t *testing.T) {
	home := t.TempDir()
	// src does not exist → no-op, no error.
	if err := ensureSharedAssets(home, filepath.Join(t.TempDir(), "nope")); err != nil {
		t.Errorf("missing src should be a no-op, got %v", err)
	}
	// home == src → no-op (never symlink the shared source into itself).
	same := t.TempDir()
	if err := os.MkdirAll(filepath.Join(same, "bin"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := ensureSharedAssets(same, same); err != nil {
		t.Errorf("home==src should be a no-op, got %v", err)
	}
	if fi, _ := os.Lstat(filepath.Join(same, "bin")); fi != nil && fi.Mode()&os.ModeSymlink != 0 {
		t.Error("home==src must not turn bin/ into a self-symlink")
	}
}

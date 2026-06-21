package agyacct

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeToken creates a logged-in agy HOME at home (token file + parents).
func writeAgyToken(t *testing.T, home string) {
	t.Helper()
	p := filepath.Join(home, agyTokenRelPath)
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(p, []byte("token-bytes"), 0o600); err != nil {
		t.Fatalf("write token: %v", err)
	}
}

func TestAccountHasCredentials(t *testing.T) {
	home := t.TempDir()
	if accountHasCredentials(home) {
		t.Fatal("empty HOME should have no credentials")
	}
	writeAgyToken(t, home)
	if !accountHasCredentials(home) {
		t.Fatal("HOME with token should report credentials present")
	}
	if accountHasCredentials("") {
		t.Fatal("empty path must be false")
	}

	// An empty (zero-byte) token file is not "logged in".
	empty := t.TempDir()
	p := filepath.Join(empty, agyTokenRelPath)
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if accountHasCredentials(empty) {
		t.Fatal("zero-byte token must not count as logged in")
	}
}

func TestDiscoverLocalAccounts(t *testing.T) {
	// Point HOME at a dir with no token so the synthetic "default" entry
	// is not emitted — keeps this test independent of the real host.
	t.Setenv("HOME", t.TempDir())

	accountsDir := t.TempDir()
	// work: logged in → discovered. half: dir exists but no token → skipped.
	writeAgyToken(t, filepath.Join(accountsDir, "work"))
	if err := os.MkdirAll(filepath.Join(accountsDir, "half"), 0o700); err != nil {
		t.Fatal(err)
	}
	// A symlinked account dir must be rejected even if it has a token.
	realTokenHome := t.TempDir()
	writeAgyToken(t, realTokenHome)
	if err := os.Symlink(realTokenHome, filepath.Join(accountsDir, "link")); err != nil {
		t.Fatal(err)
	}

	got, err := discoverLocalAccounts(accountsDir)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	names := map[string]bool{}
	for _, d := range got {
		names[d.name] = true
	}
	if !names["work"] {
		t.Error("expected 'work' (has token) to be discovered")
	}
	if names["half"] {
		t.Error("'half' has no token; must not be discovered")
	}
	if names["link"] {
		t.Error("symlinked account dir must be rejected")
	}
	if names["default"] {
		t.Error("HOME has no token; synthetic default must not appear")
	}
}

func TestSelectSpawnHome(t *testing.T) {
	// Empty HOME → configuration error.
	if _, err := selectSpawnHome("acct", ""); err == nil {
		t.Error("empty home should error")
	}
	// HOME exists but has no OAuth token → guided-login error.
	noTok := t.TempDir()
	if _, err := selectSpawnHome("acct", noTok); err == nil {
		t.Error("home without token should error (not logged in)")
	} else if !strings.Contains(err.Error(), "agy") {
		t.Errorf("error should hint the guided login, got: %v", err)
	}
	// HOME with a logged-in token → returns the HOME for injection.
	home := t.TempDir()
	writeAgyToken(t, home)
	got, err := selectSpawnHome("acct", home)
	if err != nil {
		t.Fatalf("valid logged-in home: %v", err)
	}
	if got != home {
		t.Errorf("got %q, want %q", got, home)
	}
}

func TestDiscoverDefaultAccountFromHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeAgyToken(t, home) // the gateway user's primary agy login

	got, err := discoverLocalAccounts(t.TempDir()) // empty accountsDir
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if len(got) != 1 || got[0].name != "default" || got[0].configDir != home {
		t.Fatalf("expected one synthetic default rooted at HOME, got %+v", got)
	}
}

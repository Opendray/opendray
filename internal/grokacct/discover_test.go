package grokacct

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeGrokToken creates a logged-in grok GROK_HOME at home: grok writes
// its xAI credentials to <GROK_HOME>/auth.json on `grok login`.
func writeGrokToken(t *testing.T, home string) {
	t.Helper()
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, grokAuthRelPath), []byte("{\"token\":\"x\"}"), 0o600); err != nil {
		t.Fatalf("write token: %v", err)
	}
}

func TestAccountHasCredentials(t *testing.T) {
	home := t.TempDir()
	if accountHasCredentials(home) {
		t.Fatal("empty GROK_HOME should have no credentials")
	}
	writeGrokToken(t, home)
	if !accountHasCredentials(home) {
		t.Fatal("GROK_HOME with auth.json should report credentials present")
	}
	if accountHasCredentials("") {
		t.Fatal("empty path must be false")
	}

	// A zero-byte auth.json is not "logged in".
	empty := t.TempDir()
	if err := os.WriteFile(filepath.Join(empty, grokAuthRelPath), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if accountHasCredentials(empty) {
		t.Fatal("zero-byte auth.json must not count as logged in")
	}
}

func TestSelectSpawnHome(t *testing.T) {
	// No configured home → error naming the account.
	if _, err := selectSpawnHome("work", ""); err == nil {
		t.Fatal("empty home should error")
	} else if !strings.Contains(err.Error(), "work") {
		t.Errorf("error should name the account; got %v", err)
	}

	// Home set but not logged in → guided-login error mentioning grok login.
	notLoggedIn := t.TempDir()
	_, err := selectSpawnHome("work", notLoggedIn)
	if err == nil {
		t.Fatal("home without auth.json should error")
	}
	if !strings.Contains(err.Error(), "grok login") || !strings.Contains(err.Error(), "GROK_HOME") {
		t.Errorf("error should guide GROK_HOME=... grok login; got %v", err)
	}

	// Logged-in home → returns the home to inject.
	ok := t.TempDir()
	writeGrokToken(t, ok)
	got, err := selectSpawnHome("work", ok)
	if err != nil {
		t.Fatalf("logged-in home should succeed: %v", err)
	}
	if got != ok {
		t.Errorf("want home %q, got %q", ok, got)
	}
}

func TestDiscoverLocalAccounts(t *testing.T) {
	// Point HOME at a dir with no grok home and clear GROK_HOME so the
	// synthetic "default" entry is not emitted — keeps this test
	// independent of the real host.
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GROK_HOME", "")

	accountsDir := t.TempDir()
	// work: logged in → discovered. half: dir exists but no token → skipped.
	writeGrokToken(t, filepath.Join(accountsDir, "work"))
	if err := os.MkdirAll(filepath.Join(accountsDir, "half"), 0o700); err != nil {
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
		t.Errorf("logged-in account 'work' should be discovered; got %+v", got)
	}
	if names["half"] {
		t.Errorf("half-set-up account 'half' (no auth.json) must be skipped; got %+v", got)
	}
}

func TestDiscoverLocalAccounts_DefaultFromGrokHome(t *testing.T) {
	// When the gateway user's own GROK_HOME (or ~/.grok) is logged in, a
	// synthetic "default" account is surfaced. Use GROK_HOME to avoid
	// touching the real ~/.grok.
	gh := t.TempDir()
	writeGrokToken(t, gh)
	t.Setenv("GROK_HOME", gh)
	t.Setenv("HOME", t.TempDir())

	got, err := discoverLocalAccounts(t.TempDir())
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	var haveDefault bool
	for _, d := range got {
		if d.name == "default" {
			haveDefault = true
		}
	}
	if !haveDefault {
		t.Errorf("logged-in GROK_HOME should surface a 'default' account; got %+v", got)
	}
}

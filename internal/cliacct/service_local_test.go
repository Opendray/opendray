package cliacct

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFile is a tiny helper for building the on-disk fixtures.
func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// collectNames is a small helper so existing assertions still read
// against a name-set even though discoverLocalAccounts now returns
// richer entries.
func collectNames(in []discoveredAccount) []string {
	out := make([]string, 0, len(in))
	for _, d := range in {
		out = append(out, d.name)
	}
	return out
}

func TestDiscoverLocalAccounts(t *testing.T) {
	// Isolate HOME so the test never sees the host's real ~/.claude
	// (which would change the result based on which machine you run
	// the suite on).
	t.Setenv("HOME", t.TempDir())

	dir := t.TempDir()

	// Config-dir accounts (the documented claude-login flow).
	writeFile(t, filepath.Join(dir, "kevin", ".credentials.json"), `{"claudeAiOauth":{}}`)
	writeFile(t, filepath.Join(dir, "work", ".credentials.json"), `{"claudeAiOauth":{}}`)
	// A directory that is NOT a config dir (no .credentials.json) → ignored.
	if err := os.MkdirAll(filepath.Join(dir, "notanaccount"), 0o700); err != nil {
		t.Fatal(err)
	}
	// Legacy token files, including one that duplicates a config-dir name.
	writeFile(t, filepath.Join(dir, "tokens", "legacy.token"), "sk-ant-oat01-xxx")
	writeFile(t, filepath.Join(dir, "tokens", "kevin.token"), "sk-ant-oat01-dup")

	discovered, err := discoverLocalAccounts(dir)
	if err != nil {
		t.Fatal(err)
	}
	names := collectNames(discovered)

	got := map[string]bool{}
	for _, n := range names {
		if got[n] {
			t.Errorf("duplicate name %q in result %v", n, names)
		}
		got[n] = true
	}
	for _, want := range []string{"kevin", "work", "legacy"} {
		if !got[want] {
			t.Errorf("expected account %q in %v", want, names)
		}
	}
	if got["notanaccount"] {
		t.Error("a dir without .credentials.json must not be imported")
	}
	if got["tokens"] {
		t.Error("the tokens dir itself must not be imported as an account")
	}
	if got["default"] {
		t.Error("no ~/.claude/.credentials.json was created — 'default' must not be emitted")
	}
	if len(names) != 3 {
		t.Errorf("expected 3 unique accounts, got %d: %v", len(names), names)
	}
}

func TestDiscoverLocalAccounts_MissingDirIsNotError(t *testing.T) {
	// Nonexistent accounts dir → empty result, no error (this is the bug
	// that produced "run `claude-acc init`").
	t.Setenv("HOME", t.TempDir())
	got, err := discoverLocalAccounts(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("missing dir should not error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected no entries, got %v", collectNames(got))
	}
}

func TestDiscoverLocalAccounts_EmitsDefaultWhenClaudeHomeHasCreds(t *testing.T) {
	// Set HOME to a tempdir and stage a ~/.claude/.credentials.json there.
	// The synthetic "default" entry should surface, pointing at HOME/.claude.
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeFile(t, filepath.Join(home, ".claude", ".credentials.json"), `{"claudeAiOauth":{}}`)

	// Empty accountsDir (no named accounts) → only 'default' should appear.
	got, err := discoverLocalAccounts(filepath.Join(t.TempDir(), "accounts-empty"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("expected exactly 1 entry (default), got %d: %v", len(got), collectNames(got))
	}
	if got[0].name != "default" {
		t.Errorf("name = %q, want 'default'", got[0].name)
	}
	if got[0].configDir != filepath.Join(home, ".claude") {
		t.Errorf("configDir = %q, want %q", got[0].configDir, filepath.Join(home, ".claude"))
	}
	if got[0].tokenPath != "" {
		t.Errorf("tokenPath = %q, want empty (default account has no legacy token file)",
			got[0].tokenPath)
	}
	if got[0].displayName == "" {
		t.Error("displayName should be set so the UI can render a friendly label")
	}
}

func TestDiscoverLocalAccounts_DefaultEmittedAlongsideNamedAccounts(t *testing.T) {
	// Mixed: a real ~/.claude/ default AND a named ~/.claude-accounts/kevin/
	// should both surface, default first.
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeFile(t, filepath.Join(home, ".claude", ".credentials.json"), `{"claudeAiOauth":{}}`)
	accountsDir := filepath.Join(t.TempDir(), "accounts")
	writeFile(t, filepath.Join(accountsDir, "kevin", ".credentials.json"), `{"claudeAiOauth":{}}`)

	got, err := discoverLocalAccounts(accountsDir)
	if err != nil {
		t.Fatal(err)
	}
	names := collectNames(got)
	if len(names) != 2 {
		t.Fatalf("expected 2 entries, got %d: %v", len(names), names)
	}
	if names[0] != "default" {
		t.Errorf("default should come first; got %v", names)
	}
	if names[1] != "kevin" {
		t.Errorf("kevin should come second; got %v", names)
	}
}

func TestSelectSpawnCreds(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, "kevin")
	writeFile(t, filepath.Join(configDir, ".credentials.json"), `{"claudeAiOauth":{}}`)

	t.Run("config-dir account: configDir set, no static token", func(t *testing.T) {
		cd, token, err := selectSpawnCreds("kevin", configDir, "")
		if err != nil {
			t.Fatal(err)
		}
		if cd != configDir {
			t.Errorf("configDir = %q, want %q", cd, configDir)
		}
		if token != "" {
			t.Errorf("config-dir account must not pin a static token, got %q", token)
		}
	})

	t.Run("legacy token account: token returned", func(t *testing.T) {
		tokPath := filepath.Join(dir, "tokens", "legacy.token")
		writeFile(t, tokPath, "  sk-ant-oat01-abc\n")
		cd, token, err := selectSpawnCreds("legacy", "", tokPath)
		if err != nil {
			t.Fatal(err)
		}
		if cd != "" {
			t.Errorf("configDir = %q, want empty", cd)
		}
		if token != "sk-ant-oat01-abc" {
			t.Errorf("token = %q, want trimmed legacy token", token)
		}
	})

	t.Run("no creds at all: error", func(t *testing.T) {
		if _, _, err := selectSpawnCreds("ghost", filepath.Join(dir, "missing"), ""); err == nil {
			t.Error("expected error when neither token nor config-dir creds exist")
		}
	})
}

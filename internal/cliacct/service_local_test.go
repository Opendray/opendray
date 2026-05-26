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

func TestDiscoverLocalAccountNames(t *testing.T) {
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

	names, err := discoverLocalAccountNames(dir)
	if err != nil {
		t.Fatal(err)
	}

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
	if len(names) != 3 {
		t.Errorf("expected 3 unique accounts, got %d: %v", len(names), names)
	}
}

func TestDiscoverLocalAccountNames_MissingDirIsNotError(t *testing.T) {
	// Nonexistent accounts dir → empty result, no error (this is the bug
	// that produced "run `claude-acc init`").
	names, err := discoverLocalAccountNames(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("missing dir should not error: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("expected no names, got %v", names)
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

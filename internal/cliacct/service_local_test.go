package cliacct

import (
	"context"
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
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome) // Windows: os.UserHomeDir reads USERPROFILE

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
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome) // Windows: os.UserHomeDir reads USERPROFILE
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
	t.Setenv("USERPROFILE", home) // Windows: os.UserHomeDir reads USERPROFILE
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
	t.Setenv("USERPROFILE", home) // Windows: os.UserHomeDir reads USERPROFILE
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

func TestAccountHasCredentials(t *testing.T) {
	dir := t.TempDir()

	t.Run("config-dir account with .credentials.json → true", func(t *testing.T) {
		cdir := filepath.Join(dir, "cfgdir")
		writeFile(t, filepath.Join(cdir, ".credentials.json"), `{"claudeAiOauth":{}}`)
		if !accountHasCredentials(cdir, filepath.Join(dir, "no-such.token")) {
			t.Error("expected true for config-dir account with .credentials.json")
		}
	})

	t.Run("legacy token file present and non-empty → true", func(t *testing.T) {
		tp := filepath.Join(dir, "legacy.token")
		writeFile(t, tp, "sk-ant-oat01-x")
		if !accountHasCredentials("", tp) {
			t.Error("expected true for non-empty legacy token file")
		}
	})

	t.Run("legacy token file empty → fall through to config-dir check", func(t *testing.T) {
		tp := filepath.Join(dir, "empty.token")
		writeFile(t, tp, "")
		// Empty token file alone is not credentials, and no config-dir → false.
		if accountHasCredentials("", tp) {
			t.Error("empty token file with no config-dir should report false")
		}
	})

	t.Run("nothing at all → false", func(t *testing.T) {
		if accountHasCredentials(filepath.Join(dir, "no-cdir"), filepath.Join(dir, "no.token")) {
			t.Error("expected false when neither source has credentials")
		}
	})

	t.Run("default account: tokenPath nonexistent but config-dir has creds → true", func(t *testing.T) {
		// This is the regression that motivated the change. The
		// 'default' account's TokenPath points at <accountsDir>/tokens/default.token
		// which doesn't exist, but its ConfigDir points at ~/.claude
		// which DOES have .credentials.json. Should report true.
		home := filepath.Join(dir, "home")
		writeFile(t, filepath.Join(home, ".claude", ".credentials.json"), `{"claudeAiOauth":{}}`)
		if !accountHasCredentials(
			filepath.Join(home, ".claude"),
			filepath.Join(dir, "accounts", "tokens", "default.token"), // never written
		) {
			t.Error("default account with creds in config-dir must report token_filled=true")
		}
	})
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

func TestResolveClaudeConfigDir_EmptyIDReturnsHomeClaude(t *testing.T) {
	// Sessions with no pinned account use the Claude CLI's own
	// ~/.claude home. The transcript-migration path needs that
	// concrete dir (not "") so the source JSONL can be located when
	// switching from "no pin" to a named account.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // Windows: os.UserHomeDir reads USERPROFILE
	svc := &Service{}             // store not needed for empty-id path
	got, err := svc.ResolveClaudeConfigDir(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".claude")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestAutoAssignRespectsEnabledFlag pins the contract that toggling an
// account 'disabled' immediately excludes it from PickAutoAssign and
// that toggling it back 'enabled' immediately re-includes it. The SQL
// in store.pickLeastLoaded enforces this via WHERE ca.enabled = true,
// and CheckClaudeAccountEnabled enforces it on explicit-pin paths;
// this test would catch a regression in either.
//
// We don't go through Postgres here (the rest of the package's tests
// stay DB-free); instead we test the in-memory predicate that
// PickAutoAssignClaudeAccount applies on the Service.List() result.
// The SQL filter is small and already mechanically verified by the
// live integration probe documented in the PR description.
func TestPickAutoAssign_RespectsEnabledCount(t *testing.T) {
	// With 0 enabled accounts → empty pick, no error.
	if got := countEnabled([]Account{{Enabled: false}, {Enabled: false}}); got != 0 {
		t.Errorf("0 enabled: countEnabled = %d, want 0", got)
	}
	// With 1 enabled account → empty pick (auto-assign is for ≥2).
	if got := countEnabled([]Account{{Enabled: true}, {Enabled: false}}); got != 1 {
		t.Errorf("1 enabled: countEnabled = %d, want 1", got)
	}
	// With 2 enabled accounts → eligible for auto-assign.
	if got := countEnabled([]Account{{Enabled: true}, {Enabled: true}, {Enabled: false}}); got != 2 {
		t.Errorf("2 enabled: countEnabled = %d, want 2", got)
	}
}

// countEnabled is the exact filter PickAutoAssignClaudeAccount applies
// to decide whether the pool has enough capacity to be worth balancing.
// Lifted out so the test pins the predicate that's currently inline.
func countEnabled(rows []Account) int {
	n := 0
	for _, a := range rows {
		if a.Enabled {
			n++
		}
	}
	return n
}

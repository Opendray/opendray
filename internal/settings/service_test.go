package settings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"

	"github.com/opendray/opendray-v2/internal/config"
)

func TestService_Get_StripsSensitive(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.toml")
	writeToml(t, path, config.Config{
		Listen: "0.0.0.0:8770",
		Database: config.DatabaseConfig{
			URL: "postgres://secret@localhost/db",
		},
		Admin: config.AdminConfig{
			User:     "admin",
			Password: "supersecret",
		},
	})

	svc := NewService(path, nil)
	got, err := svc.Get()
	if err != nil {
		t.Fatal(err)
	}
	if got.Database.URL != "" {
		t.Errorf("database url leaked: %q", got.Database.URL)
	}
	if got.Admin.Password != "" {
		t.Errorf("admin password leaked: %q", got.Admin.Password)
	}
	if got.Admin.User != "admin" {
		t.Errorf("admin user lost: %q", got.Admin.User)
	}
	if got.Listen != "0.0.0.0:8770" {
		t.Errorf("listen lost: %q", got.Listen)
	}
}

func TestService_Update_PreservesSensitiveOnEmpty(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.toml")
	writeToml(t, path, config.Config{
		Database: config.DatabaseConfig{URL: "postgres://original@localhost/db"},
		Admin: config.AdminConfig{
			User:     "admin",
			Password: "originalpass",
		},
	})

	svc := NewService(path, nil)
	// Patch with empty sensitive fields — should be preserved.
	patch := config.Config{
		Listen: "0.0.0.0:9999",
		Admin: config.AdminConfig{
			User:     "admin",
			Password: "", // empty → keep "originalpass"
			TokenTTL: "12h",
		},
		// Database absent → DB.URL = "" → keep original.
	}
	if err := svc.Update(&patch); err != nil {
		t.Fatal(err)
	}

	var got config.Config
	if _, err := toml.DecodeFile(path, &got); err != nil {
		t.Fatal(err)
	}
	if got.Database.URL != "postgres://original@localhost/db" {
		t.Errorf("db url not preserved: %q", got.Database.URL)
	}
	if got.Admin.Password != "originalpass" {
		t.Errorf("password not preserved: %q", got.Admin.Password)
	}
	if got.Listen != "0.0.0.0:9999" {
		t.Errorf("listen not updated: %q", got.Listen)
	}
	if got.Admin.TokenTTL != "12h" {
		t.Errorf("token_ttl not saved: %q", got.Admin.TokenTTL)
	}
}

func TestService_Update_OverwritesSensitiveWhenProvided(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.toml")
	writeToml(t, path, config.Config{
		Admin: config.AdminConfig{Password: "oldpass"},
	})

	svc := NewService(path, nil)
	patch := config.Config{
		Admin: config.AdminConfig{Password: "newpass"},
	}
	if err := svc.Update(&patch); err != nil {
		t.Fatal(err)
	}

	var got config.Config
	if _, err := toml.DecodeFile(path, &got); err != nil {
		t.Fatal(err)
	}
	if got.Admin.Password != "newpass" {
		t.Errorf("password not overwritten: %q", got.Admin.Password)
	}
}

func TestService_Update_AtomicWrite_NoDanglingTmp(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.toml")
	writeToml(t, path, config.Config{Listen: "0.0.0.0:8770"})

	svc := NewService(path, nil)
	if err := svc.Update(&config.Config{Listen: "0.0.0.0:9000"}); err != nil {
		t.Fatal(err)
	}

	entries, _ := os.ReadDir(tmp)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("dangling tmp file: %s", e.Name())
		}
	}
}

func TestService_Update_PreservesProvidersSection(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.toml")
	writeToml(t, path, config.Config{
		Providers: config.ProvidersConfig{
			Claude: config.ClaudeProviderConfig{
				HistoryRoots: []string{"/custom/claude"},
				AccountsDir:  "/custom/accounts",
			},
			Codex:  config.CodexProviderConfig{SessionsRoot: "/custom/codex"},
			Gemini: config.GeminiProviderConfig{TmpRoot: "/custom/gem"},
		},
	})

	svc := NewService(path, nil)
	got, err := svc.Get()
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Providers.Claude.HistoryRoots) != 1 ||
		got.Providers.Claude.HistoryRoots[0] != "/custom/claude" {
		t.Errorf("history_roots round-trip failed: %+v", got.Providers.Claude)
	}

	// Round-trip via Update.
	if err := svc.Update(got); err != nil {
		t.Fatal(err)
	}
	var roundTripped config.Config
	if _, err := toml.DecodeFile(path, &roundTripped); err != nil {
		t.Fatal(err)
	}
	if roundTripped.Providers.Codex.SessionsRoot != "/custom/codex" {
		t.Errorf("codex root lost on round-trip")
	}
	if roundTripped.Providers.Gemini.TmpRoot != "/custom/gem" {
		t.Errorf("gemini tmp lost on round-trip")
	}
}

func writeToml(t *testing.T, path string, c config.Config) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := toml.NewEncoder(f).Encode(c); err != nil {
		t.Fatal(err)
	}
}

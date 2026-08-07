package settings

import (
	"errors"
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
		Listen:   "0.0.0.0:8770",
		Database: config.DatabaseConfig{URL: "postgres://u@localhost/db"},
		Admin:    config.AdminConfig{Password: "oldpass"},
	})

	svc := NewService(path, nil)
	patch := config.Config{
		Listen: "0.0.0.0:8770",
		Admin:  config.AdminConfig{Password: "newpass"},
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
	writeToml(t, path, config.Config{
		Listen:   "0.0.0.0:8770",
		Database: config.DatabaseConfig{URL: "postgres://u@localhost/db"},
	})

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
		// listen + database.url are what the loader insists on; a
		// round-trip test has to start from a config that would
		// actually boot.
		Listen:   "0.0.0.0:8770",
		Database: config.DatabaseConfig{URL: "postgres://u@localhost/db"},
		Providers: config.ProvidersConfig{
			Claude: config.ClaudeProviderConfig{
				HistoryRoots: []string{"/custom/claude"},
				AccountsDir:  "/custom/accounts",
			},
			Codex: config.CodexProviderConfig{SessionsRoot: "/custom/codex"},
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

// A PUT that would write a config the loader rejects must fail at save
// time. Otherwise the write succeeds, the operator hits Restart, and
// the gateway never comes back — with the offending field unnamed.
func TestService_Update_RejectsConfigTheLoaderWouldReject(t *testing.T) {
	tests := []struct {
		name  string
		patch func(*config.Config)
	}{
		{
			name: "unknown host sleep mode",
			patch: func(c *config.Config) {
				c.Host.PreventIdleSleep = "sometimes"
			},
		},
		{
			name: "unparseable idle threshold",
			patch: func(c *config.Config) {
				c.Session.IdleThreshold = "half an hour"
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			path := filepath.Join(tmp, "config.toml")
			original := config.Config{
				Listen:   "0.0.0.0:8770",
				Database: config.DatabaseConfig{URL: "postgres://u@localhost/db"},
				Admin:    config.AdminConfig{User: "admin", Password: "pw"},
			}
			writeToml(t, path, original)

			patch := original
			patch.Database.URL = "" // as the UI sends it: stripped, merged back
			patch.Admin.Password = ""
			tt.patch(&patch)

			svc := NewService(path, nil)
			err := svc.Update(&patch)
			if err == nil {
				t.Fatal("Update accepted a config the loader would reject")
			}
			if !errors.Is(err, ErrInvalidConfig) {
				t.Fatalf("err = %v, want it to wrap ErrInvalidConfig", err)
			}

			// The on-disk config must be untouched, not half-written.
			var onDisk config.Config
			if _, err := toml.DecodeFile(path, &onDisk); err != nil {
				t.Fatalf("config no longer decodes: %v", err)
			}
			if err := onDisk.Validate(); err != nil {
				t.Fatalf("a rejected PUT corrupted the stored config: %v", err)
			}
			if onDisk.Admin.Password != "pw" {
				t.Fatalf("stored password changed: %q", onDisk.Admin.Password)
			}
		})
	}
}

// The valid modes must all survive a round-trip, so the settings UI
// can offer every one of them.
func TestService_Update_AcceptsEveryHostSleepMode(t *testing.T) {
	for _, mode := range []string{"", "ac", "always", "on_demand", "off"} {
		t.Run("mode="+mode, func(t *testing.T) {
			tmp := t.TempDir()
			path := filepath.Join(tmp, "config.toml")
			writeToml(t, path, config.Config{
				Listen:   "0.0.0.0:8770",
				Database: config.DatabaseConfig{URL: "postgres://u@localhost/db"},
			})

			svc := NewService(path, nil)
			patch := config.Config{Listen: "0.0.0.0:8770"}
			patch.Host.PreventIdleSleep = mode
			if err := svc.Update(&patch); err != nil {
				t.Fatalf("Update(%q) = %v", mode, err)
			}
			var onDisk config.Config
			if _, err := toml.DecodeFile(path, &onDisk); err != nil {
				t.Fatal(err)
			}
			if onDisk.Host.PreventIdleSleep != mode {
				t.Fatalf("stored mode = %q, want %q", onDisk.Host.PreventIdleSleep, mode)
			}
		})
	}
}

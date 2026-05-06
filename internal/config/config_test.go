package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_DefaultsAndFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(`
listen = "127.0.0.1:9999"

[database]
url = "postgres://x:y@localhost/z"

[admin]
user = "admin"
password = "secret"
`), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Listen != "127.0.0.1:9999" {
		t.Errorf("Listen = %q, want 127.0.0.1:9999", got.Listen)
	}
	if got.Database.URL != "postgres://x:y@localhost/z" {
		t.Errorf("Database.URL = %q", got.Database.URL)
	}
	if got.Log.Level != "info" {
		t.Errorf("Log.Level = %q, want default info", got.Log.Level)
	}
}

func TestLoad_EnvOverride(t *testing.T) {
	t.Setenv("OPENDRAY_LISTEN", "0.0.0.0:1234")
	t.Setenv("OPENDRAY_DATABASE_URL", "postgres://env/db")
	got, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Listen != "0.0.0.0:1234" {
		t.Errorf("Listen = %q", got.Listen)
	}
	if got.Database.URL != "postgres://env/db" {
		t.Errorf("Database.URL = %q", got.Database.URL)
	}
}

func TestLoad_MissingDatabaseURL(t *testing.T) {
	if _, err := Load(""); err == nil {
		t.Fatal("expected validation error for missing database url")
	}
}

func TestLoad_DatabaseMaxConns(t *testing.T) {
	t.Run("from toml", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.toml")
		if err := os.WriteFile(path, []byte(`
[database]
url = "postgres://x:y@localhost/z"
max_conns = 32

[admin]
user = "admin"
password = "secret"
`), 0o600); err != nil {
			t.Fatal(err)
		}
		got, err := Load(path)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if got.Database.MaxConns != 32 {
			t.Errorf("MaxConns = %d, want 32", got.Database.MaxConns)
		}
	})

	t.Run("env override beats toml", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.toml")
		if err := os.WriteFile(path, []byte(`
[database]
url = "postgres://x:y@localhost/z"
max_conns = 8
`), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Setenv("OPENDRAY_DATABASE_MAX_CONNS", "64")
		got, err := Load(path)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if got.Database.MaxConns != 64 {
			t.Errorf("MaxConns = %d, want 64", got.Database.MaxConns)
		}
	})

	t.Run("invalid env ignored", func(t *testing.T) {
		t.Setenv("OPENDRAY_DATABASE_URL", "postgres://x")
		t.Setenv("OPENDRAY_DATABASE_MAX_CONNS", "not-a-number")
		got, err := Load("")
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if got.Database.MaxConns != 0 {
			t.Errorf("MaxConns = %d, want 0 (invalid env value should be ignored)",
				got.Database.MaxConns)
		}
	})

	t.Run("negative env ignored", func(t *testing.T) {
		t.Setenv("OPENDRAY_DATABASE_URL", "postgres://x")
		t.Setenv("OPENDRAY_DATABASE_MAX_CONNS", "-5")
		got, err := Load("")
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if got.Database.MaxConns != 0 {
			t.Errorf("MaxConns = %d, want 0", got.Database.MaxConns)
		}
	})
}

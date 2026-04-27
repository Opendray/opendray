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

package config

import (
	"os"
	"path/filepath"
	"testing"
)

// SessionConfig.ResolveEnv is the generic, tool-agnostic mechanism for
// injecting env into every session. jev is just one consumer of it
// (TYPESAFE_API_KEY); the core knows nothing about jev.
func TestSessionConfigResolveEnv(t *testing.T) {
	t.Run("inline env", func(t *testing.T) {
		env, err := SessionConfig{Env: map[string]string{"FOO": "bar"}}.ResolveEnv()
		if err != nil {
			t.Fatal(err)
		}
		if env["FOO"] != "bar" {
			t.Errorf("got %v", env)
		}
	})

	t.Run("env_files read and trimmed (secret out of config)", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "key")
		if err := os.WriteFile(p, []byte("  apikey_fromfile\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		env, err := SessionConfig{EnvFiles: map[string]string{"TYPESAFE_API_KEY": p}}.ResolveEnv()
		if err != nil {
			t.Fatal(err)
		}
		if env["TYPESAFE_API_KEY"] != "apikey_fromfile" {
			t.Errorf("got %v, want trimmed file value", env)
		}
	})

	t.Run("env_files overrides inline on key conflict", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "key")
		if err := os.WriteFile(p, []byte("filewins"), 0o600); err != nil {
			t.Fatal(err)
		}
		env, _ := SessionConfig{
			Env:      map[string]string{"K": "inline"},
			EnvFiles: map[string]string{"K": p},
		}.ResolveEnv()
		if env["K"] != "filewins" {
			t.Errorf("env_files should win, got %q", env["K"])
		}
	})

	t.Run("missing file errors", func(t *testing.T) {
		_, err := SessionConfig{EnvFiles: map[string]string{"K": "/no/such/file"}}.ResolveEnv()
		if err == nil {
			t.Error("expected error for unreadable env_files path")
		}
	})

	t.Run("unset yields empty map (feature off)", func(t *testing.T) {
		env, err := SessionConfig{}.ResolveEnv()
		if err != nil {
			t.Fatal(err)
		}
		if len(env) != 0 {
			t.Errorf("expected empty, got %v", env)
		}
	})

	t.Run("blank keys/paths skipped", func(t *testing.T) {
		env, _ := SessionConfig{
			Env:      map[string]string{"  ": "x"},
			EnvFiles: map[string]string{"K": "  "},
		}.ResolveEnv()
		if len(env) != 0 {
			t.Errorf("blank entries should be skipped, got %v", env)
		}
	})
}

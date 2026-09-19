package config

import (
	"os"
	"path/filepath"
	"testing"
)

// The Jev integration turns an operator-supplied API key into a single
// env var injected into every session, so any provider's agent can call
// the `jev` CLI. ResolveEnv is the pure mapping from config to that env.
func TestJevConfigResolveEnv(t *testing.T) {
	t.Run("inline key maps to TYPESAFE_API_KEY", func(t *testing.T) {
		env, err := JevConfig{APIKey: "apikey_abc"}.ResolveEnv()
		if err != nil {
			t.Fatal(err)
		}
		if env["TYPESAFE_API_KEY"] != "apikey_abc" {
			t.Errorf("got %v, want TYPESAFE_API_KEY=apikey_abc", env)
		}
	})

	t.Run("api_key_file is read and trimmed", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "key")
		if err := os.WriteFile(p, []byte("  apikey_fromfile\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		env, err := JevConfig{APIKeyFile: p}.ResolveEnv()
		if err != nil {
			t.Fatal(err)
		}
		if env["TYPESAFE_API_KEY"] != "apikey_fromfile" {
			t.Errorf("got %v, want trimmed key from file", env)
		}
	})

	t.Run("custom env_var (e.g. OpenRouter)", func(t *testing.T) {
		env, _ := JevConfig{APIKey: "sk-or-x", EnvVar: "OPENROUTER_API_KEY"}.ResolveEnv()
		if env["OPENROUTER_API_KEY"] != "sk-or-x" {
			t.Errorf("got %v, want OPENROUTER_API_KEY=sk-or-x", env)
		}
	})

	t.Run("unset yields no env (feature off)", func(t *testing.T) {
		env, err := JevConfig{}.ResolveEnv()
		if err != nil {
			t.Fatal(err)
		}
		if len(env) != 0 {
			t.Errorf("expected empty env when unconfigured, got %v", env)
		}
	})

	t.Run("inline key wins over file", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "key")
		if err := os.WriteFile(p, []byte("filekey"), 0o600); err != nil {
			t.Fatal(err)
		}
		env, _ := JevConfig{APIKey: "inlinekey", APIKeyFile: p}.ResolveEnv()
		if env["TYPESAFE_API_KEY"] != "inlinekey" {
			t.Errorf("inline api_key should win, got %v", env)
		}
	})
}

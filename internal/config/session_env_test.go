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

	t.Run("dotenv file: KEY=VAL lines all injected", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "secrets.env")
		body := "" +
			"# a comment\n" +
			"\n" +
			"GH_TOKEN=ghp_abc\n" +
			"export CLOUDFLARE_API_TOKEN=cf_xyz\n" +
			"QUOTED=\"has spaces\"\n" +
			"SINGLE='sq'\n" +
			"  SPACED = trimmed \n"
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		env, err := SessionConfig{EnvDotenvFiles: []string{p}}.ResolveEnv()
		if err != nil {
			t.Fatal(err)
		}
		want := map[string]string{
			"GH_TOKEN":             "ghp_abc",
			"CLOUDFLARE_API_TOKEN": "cf_xyz",
			"QUOTED":               "has spaces",
			"SINGLE":               "sq",
			"SPACED":               "trimmed",
		}
		for k, v := range want {
			if env[k] != v {
				t.Errorf("%s = %q, want %q (full: %v)", k, env[k], v, env)
			}
		}
	})

	t.Run("precedence: dotenv < inline env < env_files", func(t *testing.T) {
		de := filepath.Join(t.TempDir(), "d.env")
		if err := os.WriteFile(de, []byte("K=fromdotenv\nONLYD=d\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		ef := filepath.Join(t.TempDir(), "k")
		if err := os.WriteFile(ef, []byte("fromenvfile"), 0o600); err != nil {
			t.Fatal(err)
		}
		env, err := SessionConfig{
			EnvDotenvFiles: []string{de},
			Env:            map[string]string{"K": "frominline", "ONLYI": "i"},
			EnvFiles:       map[string]string{"K": ef},
		}.ResolveEnv()
		if err != nil {
			t.Fatal(err)
		}
		if env["K"] != "fromenvfile" {
			t.Errorf("K = %q, want fromenvfile (env_files wins)", env["K"])
		}
		if env["ONLYD"] != "d" || env["ONLYI"] != "i" {
			t.Errorf("non-conflicting keys lost: %v", env)
		}
	})

	t.Run("missing dotenv file errors", func(t *testing.T) {
		_, err := SessionConfig{EnvDotenvFiles: []string{"/no/such/file.env"}}.ResolveEnv()
		if err == nil {
			t.Error("expected error for unreadable dotenv file")
		}
	})
}

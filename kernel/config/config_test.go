package config

import (
	"os"
	"path/filepath"
	"testing"
)

// withEnv sets env vars for the duration of the test. We use a closure
// rather than t.Setenv in a loop because we want to assert behaviour for
// both "unset" and "set to empty" which t.Setenv doesn't distinguish.
func withEnv(t *testing.T, kv map[string]string) {
	t.Helper()
	for k, v := range kv {
		t.Setenv(k, v)
	}
}

func TestDefaults(t *testing.T) {
	d := Defaults()
	if d.Server.ListenAddr != "127.0.0.1:8640" {
		t.Errorf("default listen addr = %q, want 127.0.0.1:8640", d.Server.ListenAddr)
	}
	if d.DB.Mode != "embedded" {
		t.Errorf("default db mode = %q, want embedded", d.DB.Mode)
	}
	if d.DB.Embedded.Port != 5433 {
		t.Errorf("default embedded port = %d, want 5433", d.DB.Embedded.Port)
	}
}

func TestIsComplete(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want bool
	}{
		{
			name: "empty",
			cfg:  Config{},
			want: false,
		},
		{
			name: "embedded with jwt",
			cfg: Config{
				Auth: Auth{JWTSecret: "x"},
				DB:   DB{Mode: "embedded", Embedded: EmbeddedDB{DataDir: "/tmp"}},
			},
			want: true,
		},
		{
			name: "embedded missing jwt",
			cfg: Config{
				DB: DB{Mode: "embedded", Embedded: EmbeddedDB{DataDir: "/tmp"}},
			},
			want: false,
		},
		{
			name: "external with all required",
			cfg: Config{
				Auth: Auth{JWTSecret: "x"},
				DB: DB{
					Mode:     "external",
					External: ExternalDB{Host: "localhost", User: "u", Name: "d"},
				},
			},
			want: true,
		},
		{
			name: "external missing host",
			cfg: Config{
				Auth: Auth{JWTSecret: "x"},
				DB:   DB{Mode: "external", External: ExternalDB{User: "u", Name: "d"}},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.IsComplete(); got != tt.want {
				t.Errorf("IsComplete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "default is valid",
			cfg:  Defaults(),
		},
		{
			name: "missing port in listen addr",
			cfg: func() Config {
				c := Defaults()
				c.Server.ListenAddr = "localhost"
				return c
			}(),
			wantErr: true,
		},
		{
			name: "bad log level",
			cfg: func() Config {
				c := Defaults()
				c.Server.LogLevel = "verbose"
				return c
			}(),
			wantErr: true,
		},
		{
			name: "embedded port out of range",
			cfg: func() Config {
				c := Defaults()
				c.DB.Embedded.Port = 0
				return c
			}(),
			wantErr: true,
		},
		{
			name: "external missing host",
			cfg: func() Config {
				c := Defaults()
				c.DB.Mode = "external"
				return c
			}(),
			wantErr: true,
		},
		{
			name: "unknown db mode",
			cfg: func() Config {
				c := Defaults()
				c.DB.Mode = "sqlite"
				return c
			}(),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Env vars should override file values in the merged config.
func TestLoadEnvOverridesFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	toml := `
schema_version = 1
[server]
listen_addr = ":8640"
[auth]
jwt_secret = "from-file"
[db]
mode = "embedded"
[db.embedded]
data_dir = "/tmp/pg"
port = 5433
version = "15.4.0"
`
	if err := os.WriteFile(cfgPath, []byte(toml), 0o600); err != nil {
		t.Fatal(err)
	}
	withEnv(t, map[string]string{
		"OPENDRAY_CONFIG": cfgPath,
		"JWT_SECRET":      "from-env",
		"LISTEN_ADDR":     ":9000",
	})

	cfg, src, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if src != SourceMixed {
		t.Errorf("source = %v, want mixed", src)
	}
	if cfg.Auth.JWTSecret != "from-env" {
		t.Errorf("JWT = %q, want env override", cfg.Auth.JWTSecret)
	}
	if cfg.Server.ListenAddr != ":9000" {
		t.Errorf("listen = %q, want env override", cfg.Server.ListenAddr)
	}
	if cfg.DB.Mode != "embedded" {
		t.Errorf("mode = %q, want from file", cfg.DB.Mode)
	}
}

// DB_HOST env var should force mode=external, even if the file says embedded.
// This preserves the existing LXC/Docker deployment's expectations.
func TestDBHostForcesExternal(t *testing.T) {
	cfg := Defaults()
	if cfg.DB.Mode != "embedded" {
		t.Fatalf("precondition: default mode = %q", cfg.DB.Mode)
	}
	withEnv(t, map[string]string{
		"OPENDRAY_CONFIG": filepath.Join(t.TempDir(), "nope.toml"),
		"DB_HOST":         "10.0.0.5",
		"DB_USER":         "u",
		"DB_NAME":         "d",
		"JWT_SECRET":      "x",
	})
	cfg, _, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DB.Mode != "external" {
		t.Errorf("mode = %q, want external (forced by DB_HOST)", cfg.DB.Mode)
	}
	if cfg.DB.External.Host != "10.0.0.5" {
		t.Errorf("host = %q, want 10.0.0.5", cfg.DB.External.Host)
	}
}

// SaveAndReload confirms the TOML round-trips without data loss.
func TestSaveAndReload(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	t.Setenv("OPENDRAY_CONFIG", cfgPath)

	original := Defaults()
	original.Auth.JWTSecret = "round-trip-secret"
	original.DB.Mode = "external"
	original.DB.External.Host = "10.0.0.7"
	original.DB.External.User = "opendray"
	original.DB.External.Password = "pw"
	original.DB.External.Name = "opendray"

	if err := Save(original); err != nil {
		t.Fatal(err)
	}
	// 0600 perms check — config contains secrets.
	info, err := os.Stat(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("perms = %o, want 0600", perm)
	}

	reloaded, _, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Auth.JWTSecret != original.Auth.JWTSecret {
		t.Errorf("jwt lost on round-trip: got %q", reloaded.Auth.JWTSecret)
	}
	if reloaded.DB.External.Host != "10.0.0.7" {
		t.Errorf("host lost: got %q", reloaded.DB.External.Host)
	}
}

// ─── T25: PluginsDataDir + AllowLocalPlugins ───────────────────────────────

// TestConfig_PluginsDataDir_Default verifies that when OPENDRAY_PLUGINS_DATA_DIR
// is not set, Load() returns a PluginsDataDir that ends with .opendray/plugins
// and is rooted under the user's home directory.
func TestConfig_PluginsDataDir_Default(t *testing.T) {
	// Isolate from any real config file and from the data-dir env var.
	t.Setenv("OPENDRAY_CONFIG", filepath.Join(t.TempDir(), "no-such-config.toml"))
	t.Setenv("OPENDRAY_PLUGINS_DATA_DIR", "")

	cfg, _, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("os.UserHomeDir: %v", err)
	}
	want := filepath.Join(home, ".opendray", "plugins")
	if cfg.PluginsDataDir != want {
		t.Errorf("PluginsDataDir = %q, want %q", cfg.PluginsDataDir, want)
	}
}

// TestConfig_PluginsDataDir_EnvOverride verifies that OPENDRAY_PLUGINS_DATA_DIR
// is respected when set.
func TestConfig_PluginsDataDir_EnvOverride(t *testing.T) {
	t.Setenv("OPENDRAY_CONFIG", filepath.Join(t.TempDir(), "no-such-config.toml"))
	t.Setenv("OPENDRAY_PLUGINS_DATA_DIR", "/custom/plugins/path")

	cfg, _, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.PluginsDataDir != "/custom/plugins/path" {
		t.Errorf("PluginsDataDir = %q, want /custom/plugins/path", cfg.PluginsDataDir)
	}
}

// TestConfig_AllowLocalPlugins_Truthy is a table-driven test covering every
// truthy and falsy variant documented on the AllowLocalPlugins field.
//
// Truthy (→ true):  "1", "true", "True", "TRUE", "yes", "Yes", "YES", "on", "On", "ON"
// Falsy  (→ false): "0", "false", "no", "off", "" (empty), unset, "bogus", "2"
func TestConfig_AllowLocalPlugins_Truthy(t *testing.T) {
	// This test calls Defaults() directly to avoid file I/O and avoid the
	// pre-existing TestLoadEnvOverridesFile env-leakage bug that affects Load().
	type row struct {
		val  string // env value to set ("" means set-to-empty, not unset)
		want bool
	}
	truthy := []row{
		{"1", true},
		{"true", true},
		{"True", true},
		{"TRUE", true},
		{"yes", true},
		{"Yes", true},
		{"YES", true},
		{"on", true},
		{"On", true},
		{"ON", true},
	}
	falsy := []row{
		{"0", false},
		{"false", false},
		{"no", false},
		{"off", false},
		{"", false},
		{"bogus", false},
		{"2", false},
	}

	for _, tc := range append(truthy, falsy...) {
		tc := tc
		t.Run("OPENDRAY_ALLOW_LOCAL_PLUGINS="+tc.val, func(t *testing.T) {
			t.Setenv("OPENDRAY_ALLOW_LOCAL_PLUGINS", tc.val)
			cfg := Defaults()
			changed := applyEnvOverrides(&cfg)
			_ = changed
			if cfg.AllowLocalPlugins != tc.want {
				t.Errorf("AllowLocalPlugins = %v, want %v (env=%q)", cfg.AllowLocalPlugins, tc.want, tc.val)
			}
		})
	}
}

// TestConfig_AllowLocalPlugins_Unset confirms the zero-value / unset case → false.
func TestConfig_AllowLocalPlugins_Unset(t *testing.T) {
	// Explicitly unset the env var using os.Unsetenv so we don't rely on
	// t.Setenv's restore semantics.
	orig, had := os.LookupEnv("OPENDRAY_ALLOW_LOCAL_PLUGINS")
	if had {
		defer os.Setenv("OPENDRAY_ALLOW_LOCAL_PLUGINS", orig)
		os.Unsetenv("OPENDRAY_ALLOW_LOCAL_PLUGINS")
	}
	cfg := Defaults()
	applyEnvOverrides(&cfg)
	if cfg.AllowLocalPlugins {
		t.Errorf("AllowLocalPlugins = true when env var is unset, want false")
	}
}

func TestGenerateJWTSecret(t *testing.T) {
	a, err := GenerateJWTSecret()
	if err != nil {
		t.Fatal(err)
	}
	b, err := GenerateJWTSecret()
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Error("two generated secrets collided — rand broken?")
	}
	if len(a) < 48 {
		t.Errorf("secret length = %d, want ≥ 48", len(a))
	}
}

// Package config is OpenDray's runtime configuration loader.
//
// The effective config is merged from three sources, listed here in order
// of increasing precedence:
//
//  1. config.toml — written by the setup wizard, hand-editable
//  2. environment variables — legacy `.env` values, keep LXC/Docker
//     deployments identical to today
//  3. explicit code-side defaults — set when neither source supplies
//     a value
//
// The boot sequence in cmd/opendray/main.go calls Load(), which returns
// a fully-merged [Config]. If the result isn't [Config.IsComplete] the
// binary enters setup mode — see kernel/setup.
package config

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

// SchemaVersion is bumped whenever the TOML layout changes in a way that
// isn't backward-compatible. Loader rejects files with a newer version.
const SchemaVersion = 1

// Config is the fully-merged runtime configuration.
type Config struct {
	SchemaVersion    int            `toml:"schema_version"`
	SetupCompletedAt string         `toml:"setup_completed_at,omitempty"`
	Server           Server         `toml:"server"`
	Auth             Auth           `toml:"auth"`
	DB               DB             `toml:"db"`
	Plugins          Plugins        `toml:"plugins"`
	Telegram         Telegram       `toml:"telegram"`
	CFAccess         CFAccess       `toml:"cf_access,omitempty"`
	PluginPlatform   PluginPlatform `toml:"plugin_platform,omitempty"`

	// Plugin platform top-level convenience fields (M1/T25).
	// These shadow PluginPlatform fields for direct access throughout the
	// codebase without chaining through the nested struct.
	//
	// PluginsDataDir is the root directory for installed plugin bundles.
	// Default: ${HOME}/.opendray/plugins. Env: OPENDRAY_PLUGINS_DATA_DIR.
	PluginsDataDir string `toml:"-"` // computed; not round-tripped in TOML

	// AllowLocalPlugins gates local-scheme plugin installs.
	// Default: false. Env: OPENDRAY_ALLOW_LOCAL_PLUGINS (truthy: 1|true|yes|on).
	AllowLocalPlugins bool `toml:"-"` // computed; not round-tripped in TOML

	// MarketplaceDir is the root of the on-disk plugin catalog that
	// backs /api/marketplace/plugins and marketplace:// install refs.
	// Default: $REPO/plugins/marketplace when running from source,
	// ${HOME}/.opendray/marketplace when OPENDRAY_MARKETPLACE_DIR is unset
	// in a production install. A missing directory leaves the Hub
	// empty rather than failing boot. Env: OPENDRAY_MARKETPLACE_DIR.
	MarketplaceDir string `toml:"-"` // computed; not round-tripped in TOML
}

// Server holds HTTP listener configuration.
type Server struct {
	ListenAddr string `toml:"listen_addr"`
	LogLevel   string `toml:"log_level"`
}

// Auth holds JWT + admin bootstrap settings. Admin credentials themselves
// live in the `admin_auth` DB row — only the JWT signing key is stored
// in the config file.
type Auth struct {
	JWTSecret string `toml:"jwt_secret"`
	// AdminBootstrapUsername / AdminBootstrapPassword are only read when
	// the DB has no admin_auth row yet (first-boot bootstrap). Once the
	// user changes credentials from the UI, these are ignored.
	AdminBootstrapUsername string `toml:"admin_bootstrap_username,omitempty"`
	AdminBootstrapPassword string `toml:"admin_bootstrap_password,omitempty"`
}

// DB describes which Postgres OpenDray talks to.
type DB struct {
	Mode     string       `toml:"mode"` // "embedded" | "external"
	Embedded EmbeddedDB   `toml:"embedded"`
	External ExternalDB   `toml:"external"`
}

// EmbeddedDB configures the managed-by-OpenDray Postgres child process.
type EmbeddedDB struct {
	DataDir  string `toml:"data_dir"`
	CacheDir string `toml:"cache_dir"`
	Port     int    `toml:"port"`
	Version  string `toml:"version"`
	Password string `toml:"password"` // auto-generated on first boot
}

// ExternalDB describes a user-provided Postgres server.
type ExternalDB struct {
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	User     string `toml:"user"`
	Password string `toml:"password"`
	Name     string `toml:"name"`
	SSLMode  string `toml:"sslmode"`
}

// Plugins holds provider-runtime knobs.
type Plugins struct {
	Dir                  string `toml:"dir"`
	AutoResume           bool   `toml:"auto_resume"`
	IdleThresholdSeconds int    `toml:"idle_threshold_seconds"`
}

// Telegram bridge token — optional, only used when the plugin is enabled.
type Telegram struct {
	BotToken string `toml:"bot_token"`
}

// CFAccess holds client credentials for service-to-service Cloudflare
// Access in LAN deployments.
type CFAccess struct {
	ClientID     string `toml:"client_id"`
	ClientSecret string `toml:"client_secret"`
}

// PluginPlatform holds the M1 plugin-platform configuration knobs.
// These fields are intentionally separate from the legacy Plugins struct so
// they don't conflict with the provider-runtime settings.
type PluginPlatform struct {
	// DataDir is the root directory where installed plugin bundles are
	// written. Defaults to ${HOME}/.opendray/plugins. Override via
	// OPENDRAY_PLUGINS_DATA_DIR.
	DataDir string `toml:"data_dir"`

	// AllowLocalPlugins controls whether POST /api/plugins/install accepts
	// local-scheme sources ("local:<abs>" or bare absolute paths). Defaults
	// to false (deny). Override via OPENDRAY_ALLOW_LOCAL_PLUGINS.
	//
	// Truthy values (case-insensitive): "1", "true", "yes", "on"
	// Falsy / unset: everything else → false
	AllowLocalPlugins bool `toml:"allow_local_plugins"`
}

// Source describes where the effective config came from. Useful for log
// lines ("config loaded source=…") and the setup wizard.
type Source string

const (
	SourceFile   Source = "file"
	SourceEnv    Source = "env"
	SourceMixed  Source = "mixed"
	SourceNone   Source = "none"
)

// Defaults returns a config with the hard-coded defaults applied. Env
// and file layers merge on top of this.
func Defaults() Config {
	home, _ := os.UserHomeDir()
	defaultPluginsDataDir := filepath.Join(home, ".opendray", "plugins")
	// Marketplace catalog root. Prefer $REPO/plugins/marketplace during
	// development (picked up automatically when the working dir is the
	// repo root) and fall back to ~/.opendray/marketplace in prod.
	defaultMarketplaceDir := filepath.Join(home, ".opendray", "marketplace")
	if _, err := os.Stat("plugins/marketplace/catalog.json"); err == nil {
		if abs, aerr := filepath.Abs("plugins/marketplace"); aerr == nil {
			defaultMarketplaceDir = abs
		}
	}

	return Config{
		SchemaVersion: SchemaVersion,
		Server: Server{
			ListenAddr: "127.0.0.1:8640",
			LogLevel:   "info",
		},
		DB: DB{
			Mode: "embedded",
			Embedded: EmbeddedDB{
				DataDir:  expandHome("~/.opendray/pg"),
				CacheDir: expandHome("~/.opendray/pg-cache"),
				Port:     5433,
				Version:  "15.4.0",
			},
			External: ExternalDB{
				Port:    5432,
				SSLMode: "disable",
			},
		},
		Plugins: Plugins{
			Dir:                  "./plugins",
			IdleThresholdSeconds: 8,
		},
		PluginsDataDir:    defaultPluginsDataDir,
		AllowLocalPlugins: false,
		MarketplaceDir:    defaultMarketplaceDir,
	}
}

// Load reads the effective config:
//   - discovers the TOML file using [DefaultPaths]
//   - overlays env vars
//   - reports back which source(s) contributed so main.go can log it
//
// A missing file is NOT an error — the caller checks [Config.IsComplete]
// to decide whether to enter setup mode.
func Load() (Config, Source, error) {
	cfg := Defaults()
	source := SourceNone

	path, ok := findConfigFile()
	if ok {
		raw, err := os.ReadFile(path)
		if err != nil {
			return cfg, source, fmt.Errorf("config: read %s: %w", path, err)
		}
		if _, err := toml.Decode(string(raw), &cfg); err != nil {
			return cfg, source, fmt.Errorf("config: parse %s: %w", path, err)
		}
		if cfg.SchemaVersion > SchemaVersion {
			return cfg, source, fmt.Errorf("config: file %s is schema v%d, binary supports v%d",
				path, cfg.SchemaVersion, SchemaVersion)
		}
		source = SourceFile
	}

	changed := applyEnvOverrides(&cfg)
	if changed {
		if source == SourceFile {
			source = SourceMixed
		} else {
			source = SourceEnv
		}
	}

	// Normalise paths — reject silent breakage from a stray "~/".
	cfg.DB.Embedded.DataDir = expandHome(cfg.DB.Embedded.DataDir)
	cfg.DB.Embedded.CacheDir = expandHome(cfg.DB.Embedded.CacheDir)

	return cfg, source, nil
}

// Save writes cfg to the default config path (or OPENDRAY_CONFIG if set).
// Writes are atomic: tmp-file + rename, 0600 perms.
func Save(cfg Config) error {
	path := defaultWritePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("config: mkdir %s: %w", filepath.Dir(path), err)
	}
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("config: open tmp: %w", err)
	}
	if err := toml.NewEncoder(f).Encode(cfg); err != nil {
		f.Close()
		os.Remove(tmp)
		return fmt.Errorf("config: encode: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("config: close tmp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("config: rename: %w", err)
	}
	return nil
}

// DefaultPaths returns the candidate config locations, in precedence order.
// Earlier entries win.
func DefaultPaths() []string {
	paths := []string{}
	if p := os.Getenv("OPENDRAY_CONFIG"); p != "" {
		paths = append(paths, p)
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		paths = append(paths, filepath.Join(xdg, "opendray", "config.toml"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths,
			filepath.Join(home, ".config", "opendray", "config.toml"),
			filepath.Join(home, ".opendray", "config.toml"),
		)
	}
	paths = append(paths, "./config.toml")
	return paths
}

// defaultWritePath is where Save() writes. Follows the same precedence as
// Load() so edits land on the file that was actually loaded.
func defaultWritePath() string {
	if p := os.Getenv("OPENDRAY_CONFIG"); p != "" {
		return p
	}
	if existing, ok := findConfigFile(); ok {
		return existing
	}
	// No existing file — default to ~/.opendray/config.toml, the
	// friendliest single-user location.
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".opendray", "config.toml")
}

func findConfigFile() (string, bool) {
	for _, p := range DefaultPaths() {
		if _, err := os.Stat(p); err == nil {
			return p, true
		}
	}
	return "", false
}

// IsComplete reports whether the config has enough to boot into normal
// mode without user input. For the external-DB path we need real
// connection params; for embedded we just need the mode flag (the rest
// is auto-generated on first PG start).
func (c Config) IsComplete() bool {
	if c.Auth.JWTSecret == "" {
		return false
	}
	switch c.DB.Mode {
	case "embedded":
		return c.DB.Embedded.DataDir != ""
	case "external":
		return c.DB.External.Host != "" &&
			c.DB.External.User != "" &&
			c.DB.External.Name != ""
	default:
		return false
	}
}

// Validate catches structural problems that IsComplete doesn't — bad
// listen address, unknown log level, etc. Used by the setup wizard's
// finalize step.
func (c Config) Validate() error {
	if !strings.Contains(c.Server.ListenAddr, ":") {
		return errors.New("server.listen_addr must include a port")
	}
	switch c.Server.LogLevel {
	case "", "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("server.log_level %q not in debug|info|warn|error", c.Server.LogLevel)
	}
	switch c.DB.Mode {
	case "embedded":
		if c.DB.Embedded.Port <= 0 || c.DB.Embedded.Port > 65535 {
			return fmt.Errorf("db.embedded.port %d out of range", c.DB.Embedded.Port)
		}
	case "external":
		if c.DB.External.Host == "" {
			return errors.New("db.external.host required")
		}
	default:
		return fmt.Errorf("db.mode %q not in embedded|external", c.DB.Mode)
	}
	return nil
}

// GenerateJWTSecret produces a 48-byte base64url-encoded secret. Used by
// the wizard's "auto-generate" path and by the embedded-DB boot to mint
// a one-shot password.
func GenerateJWTSecret() (string, error) {
	b := make([]byte, 48)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// applyEnvOverrides overlays env vars on top of the loaded config. Returns
// true if at least one env var changed the effective value, so the caller
// can tag the Source as "env" or "mixed".
func applyEnvOverrides(cfg *Config) bool {
	changed := false

	setStr := func(dst *string, key string) {
		if v, ok := os.LookupEnv(key); ok && v != "" {
			*dst = v
			changed = true
		}
	}
	setInt := func(dst *int, key string) {
		if v, ok := os.LookupEnv(key); ok && v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				*dst = n
				changed = true
			}
		}
	}
	setBool := func(dst *bool, key string) {
		if v, ok := os.LookupEnv(key); ok {
			*dst = v == "true" || v == "1" || v == "yes"
			changed = true
		}
	}

	setStr(&cfg.Server.ListenAddr, "LISTEN_ADDR")
	setStr(&cfg.Server.LogLevel, "LOG_LEVEL")

	setStr(&cfg.Auth.JWTSecret, "JWT_SECRET")
	setStr(&cfg.Auth.AdminBootstrapUsername, "ADMIN_USERNAME")
	setStr(&cfg.Auth.AdminBootstrapPassword, "ADMIN_PASSWORD")

	// Any external-DB env var forces mode=external — this preserves the
	// current LXC deployment, which just sets DB_HOST et al. without
	// ever touching the wizard.
	if os.Getenv("DB_HOST") != "" {
		cfg.DB.Mode = "external"
		changed = true
	}
	setStr(&cfg.DB.External.Host, "DB_HOST")
	setInt(&cfg.DB.External.Port, "DB_PORT")
	setStr(&cfg.DB.External.User, "DB_USER")
	setStr(&cfg.DB.External.Password, "DB_PASSWORD")
	setStr(&cfg.DB.External.Name, "DB_NAME")
	setStr(&cfg.DB.External.SSLMode, "DB_SSLMODE")

	// Plugins
	setStr(&cfg.Plugins.Dir, "PLUGIN_DIR")
	setBool(&cfg.Plugins.AutoResume, "AUTO_RESUME")
	setInt(&cfg.Plugins.IdleThresholdSeconds, "IDLE_THRESHOLD_SECONDS")

	// Telegram
	setStr(&cfg.Telegram.BotToken, "OPENDRAY_TELEGRAM_BOT_TOKEN")

	// Plugin platform (M1/T25).
	//
	// OPENDRAY_PLUGINS_DATA_DIR overrides the root directory for installed
	// plugin bundles. An empty value is ignored — callers who want to clear
	// the default must use the TOML file.
	setStr(&cfg.PluginsDataDir, "OPENDRAY_PLUGINS_DATA_DIR")

	// OPENDRAY_ALLOW_LOCAL_PLUGINS gates local-scheme install sources.
	// Truthy (case-insensitive): "1", "true", "yes", "on" → true.
	// All other values, including unset or empty string → false.
	if v, ok := os.LookupEnv("OPENDRAY_ALLOW_LOCAL_PLUGINS"); ok {
		cfg.AllowLocalPlugins = isTruthy(v)
		changed = true
	}

	// OPENDRAY_MARKETPLACE_DIR overrides the on-disk catalog root. A
	// missing directory is tolerated — the server boots with an empty
	// Hub — so operators can point this at a not-yet-populated location
	// during initial rollout.
	setStr(&cfg.MarketplaceDir, "OPENDRAY_MARKETPLACE_DIR")

	return changed
}

// isTruthy returns true when s is one of the canonical truthy strings
// (case-insensitive): "1", "true", "yes", "on". All other values → false.
// This is the single source of truth for AllowLocalPlugins parsing.
func isTruthy(s string) bool {
	switch strings.ToLower(s) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// expandHome turns a leading "~/" into the user's home directory. No-op
// for paths that don't start with "~/".
func expandHome(p string) string {
	if !strings.HasPrefix(p, "~/") && p != "~" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	if p == "~" {
		return home
	}
	return filepath.Join(home, p[2:])
}

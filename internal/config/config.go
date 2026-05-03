// Package config loads opendray's TOML configuration with environment-variable
// overrides. The TOML file is the human-edited source of truth; env vars
// (prefix OPENDRAY_) override individual fields for 12-factor deploys.
package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Listen   string         `toml:"listen"`
	Database DatabaseConfig `toml:"database"`
	Admin    AdminConfig    `toml:"admin"`
	Log      LogConfig      `toml:"log"`
	Session  SessionConfig  `toml:"session"`
	Vault    VaultConfig    `toml:"vault"`
	MCP      MCPConfig      `toml:"mcp"`
}

// MCPConfig points at the MCP server registry directory and the
// secrets file used to substitute ${KEY} placeholders at spawn time.
//
// Defaults (when unset, see resolveMCPPaths in package app):
//
//	root         = <vault.root>/mcp
//	secrets_file = ~/.opendray/secrets.env  (intentionally OUTSIDE the
//	               vault so it never git-syncs along with notes/skills)
//
// Override either via env (OPENDRAY_MCP_ROOT, OPENDRAY_MCP_SECRETS_FILE)
// or by setting the field explicitly.
type MCPConfig struct {
	Root        string `toml:"root"`
	SecretsFile string `toml:"secrets_file"`
}

// VaultConfig points at the on-disk roots that hold notes + skills.
// The whole tree is meant to be a self-contained, git-versionable
// directory the user owns — no DB lock-in.
//
// Default layout when only `root` is set:
//
//	<root>/notes/           ← opendray-managed notes
//	<root>/skills/          ← agent skills (built-in overlays)
//
// When `notes` is set explicitly it OVERRIDES the `<root>/notes`
// computation, so users can point opendray at an existing Obsidian
// vault (or any flat folder of .md files) without restructuring.
// `skills` works the same way independently.
//
// `git_root` controls which directory the Vault Sync feature operates
// on. Defaults to whichever is the most natural git working tree:
//
//	if `notes` is set explicitly → git_root defaults to `notes`
//	otherwise                    → git_root defaults to `root`
type VaultConfig struct {
	Root    string `toml:"root"`     // e.g. "~/.opendray/vault"
	Notes   string `toml:"notes"`    // override notes root (default <root>/notes)
	Skills  string `toml:"skills"`   // override skills root (default <root>/skills)
	GitRoot string `toml:"git_root"` // override repo root for vault sync

	// Default prefixes for auto-derived note paths. Useful when the
	// user pulled an existing Obsidian vault with capital-first
	// folder names (Projects/, Personal/) instead of opendray's
	// default lowercase. Per-cwd overrides live in an in-vault JSON
	// file managed via the API; these are just the templates.
	PersonalPrefix string `toml:"personal_prefix"` // default "personal"
	ProjectsPrefix string `toml:"projects_prefix"` // default "projects"
}

type DatabaseConfig struct {
	URL string `toml:"url"`
}

type AdminConfig struct {
	User     string `toml:"user"`
	Password string `toml:"password"`
	TokenTTL string `toml:"token_ttl"` // e.g. "24h", "12h", "30m"
}

// Duration parses TokenTTL; returns 0 if unset.
func (a AdminConfig) Duration() time.Duration {
	d, _ := time.ParseDuration(a.TokenTTL)
	return d
}

type LogConfig struct {
	Level  string `toml:"level"`  // debug|info|warn|error
	Format string `toml:"format"` // json|text
}

// SessionConfig drives the session.Manager idle detector. Empty values
// use Manager defaults (30s threshold, 5s poll interval).
type SessionConfig struct {
	IdleThreshold string `toml:"idle_threshold"` // e.g. "30s", "2m"
	IdleInterval  string `toml:"idle_interval"`  // e.g. "5s"
}

// Threshold parses IdleThreshold; returns 0 if unset or invalid (caller
// should call Validate first to surface invalid values).
func (s SessionConfig) Threshold() time.Duration {
	d, _ := time.ParseDuration(s.IdleThreshold)
	return d
}

// Interval parses IdleInterval; returns 0 if unset.
func (s SessionConfig) Interval() time.Duration {
	d, _ := time.ParseDuration(s.IdleInterval)
	return d
}

func defaults() Config {
	return Config{
		Listen: "127.0.0.1:8770",
		Log:    LogConfig{Level: "info", Format: "text"},
	}
}

// Load reads the TOML file at path, then applies env overrides.
// An empty path skips file loading and uses defaults + env only.
func Load(path string) (Config, error) {
	cfg := defaults()
	if path != "" {
		if _, err := toml.DecodeFile(path, &cfg); err != nil {
			return cfg, fmt.Errorf("config: decode %s: %w", path, err)
		}
	}
	applyEnv(&cfg)
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("OPENDRAY_LISTEN"); v != "" {
		cfg.Listen = v
	}
	if v := os.Getenv("OPENDRAY_DATABASE_URL"); v != "" {
		cfg.Database.URL = v
	}
	if v := os.Getenv("OPENDRAY_ADMIN_USER"); v != "" {
		cfg.Admin.User = v
	}
	if v := os.Getenv("OPENDRAY_ADMIN_PASSWORD"); v != "" {
		cfg.Admin.Password = v
	}
	if v := os.Getenv("OPENDRAY_ADMIN_TOKEN_TTL"); v != "" {
		cfg.Admin.TokenTTL = v
	}
	if v := os.Getenv("OPENDRAY_LOG_LEVEL"); v != "" {
		cfg.Log.Level = v
	}
	if v := os.Getenv("OPENDRAY_LOG_FORMAT"); v != "" {
		cfg.Log.Format = v
	}
	if v := os.Getenv("OPENDRAY_SESSION_IDLE_THRESHOLD"); v != "" {
		cfg.Session.IdleThreshold = v
	}
	if v := os.Getenv("OPENDRAY_SESSION_IDLE_INTERVAL"); v != "" {
		cfg.Session.IdleInterval = v
	}
	if v := os.Getenv("OPENDRAY_VAULT_ROOT"); v != "" {
		cfg.Vault.Root = v
	}
	if v := os.Getenv("OPENDRAY_VAULT_NOTES"); v != "" {
		cfg.Vault.Notes = v
	}
	if v := os.Getenv("OPENDRAY_VAULT_SKILLS"); v != "" {
		cfg.Vault.Skills = v
	}
	if v := os.Getenv("OPENDRAY_VAULT_GIT_ROOT"); v != "" {
		cfg.Vault.GitRoot = v
	}
	if v := os.Getenv("OPENDRAY_MCP_ROOT"); v != "" {
		cfg.MCP.Root = v
	}
	if v := os.Getenv("OPENDRAY_MCP_SECRETS_FILE"); v != "" {
		cfg.MCP.SecretsFile = v
	}
}

func (c Config) Validate() error {
	if c.Listen == "" {
		return errors.New("config: listen address is empty")
	}
	if c.Database.URL == "" {
		return errors.New("config: database.url is empty (set OPENDRAY_DATABASE_URL or [database].url)")
	}
	if c.Session.IdleThreshold != "" {
		if _, err := time.ParseDuration(c.Session.IdleThreshold); err != nil {
			return fmt.Errorf("config: session.idle_threshold: %w", err)
		}
	}
	if c.Session.IdleInterval != "" {
		if _, err := time.ParseDuration(c.Session.IdleInterval); err != nil {
			return fmt.Errorf("config: session.idle_interval: %w", err)
		}
	}
	if c.Admin.TokenTTL != "" {
		if _, err := time.ParseDuration(c.Admin.TokenTTL); err != nil {
			return fmt.Errorf("config: admin.token_ttl: %w", err)
		}
	}
	return nil
}

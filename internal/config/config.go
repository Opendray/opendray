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
	Listen    string          `toml:"listen" json:"listen"`
	Database  DatabaseConfig  `toml:"database" json:"database"`
	Admin     AdminConfig     `toml:"admin" json:"admin"`
	Log       LogConfig       `toml:"log" json:"log"`
	Session   SessionConfig   `toml:"session" json:"session"`
	Vault     VaultConfig     `toml:"vault" json:"vault"`
	MCP       MCPConfig       `toml:"mcp" json:"mcp"`
	Providers ProvidersConfig `toml:"providers" json:"providers"`

	// FilePath is the path config.toml was loaded from. Set by Load
	// after a successful read so the runtime can find the same file
	// to write back through the Settings API. Empty when running in
	// env-only mode. Not serialised back to toml.
	FilePath string `toml:"-" json:"-"`
}

// ProvidersConfig groups the on-disk locations where each external
// CLI tool (Claude, Codex, Gemini) keeps its data. opendray reads
// these to build the per-session History panel and to pick sane
// defaults when creating new accounts.
//
// Every field is optional: leaving the section out (or any single
// field empty) falls back to the upstream CLI's standard layout
// under $HOME, so the zero-value config matches today's hardcoded
// behaviour exactly. Override only when the operator runs a CLI
// from a non-default location (e.g. CLAUDE_CONFIG_DIR set on the
// shell, or a vendored install under /opt).
type ProvidersConfig struct {
	Claude ClaudeProviderConfig `toml:"claude" json:"claude"`
	Codex  CodexProviderConfig  `toml:"codex" json:"codex"`
	Gemini GeminiProviderConfig `toml:"gemini" json:"gemini"`
}

// ClaudeProviderConfig points at where Claude Code CLI persists
// per-project transcripts and per-account credentials.
//
// Defaults (when fields are empty):
//
//	HistoryRoots: [~/.claude/projects, ~/.claude-accounts/*/projects]
//	             — both are scanned and deduped via EvalSymlinks
//	AccountsDir : ~/.claude-accounts
//	             — root used when creating a new account without an
//	               explicit ConfigDir
type ClaudeProviderConfig struct {
	HistoryRoots []string `toml:"history_roots" json:"history_roots"`
	AccountsDir  string   `toml:"accounts_dir" json:"accounts_dir"`
}

// CodexProviderConfig points at the OpenAI Codex CLI's session
// rollouts directory. Default: ~/.codex/sessions.
type CodexProviderConfig struct {
	SessionsRoot string `toml:"sessions_root" json:"sessions_root"`
}

// GeminiProviderConfig points at the Google Gemini CLI's per-project
// tmp directory and the projects.json mapping file.
//
// Defaults: ~/.gemini/tmp and ~/.gemini/projects.json.
type GeminiProviderConfig struct {
	TmpRoot      string `toml:"tmp_root" json:"tmp_root"`
	ProjectsFile string `toml:"projects_file" json:"projects_file"`
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
	Root        string `toml:"root" json:"root"`
	SecretsFile string `toml:"secrets_file" json:"secrets_file"`
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
	Root    string `toml:"root" json:"root"`         // e.g. "~/.opendray/vault"
	Notes   string `toml:"notes" json:"notes"`       // override notes root (default <root>/notes)
	Skills  string `toml:"skills" json:"skills"`     // override skills root (default <root>/skills)
	GitRoot string `toml:"git_root" json:"git_root"` // override repo root for vault sync

	// Default prefixes for auto-derived note paths. Useful when the
	// user pulled an existing Obsidian vault with capital-first
	// folder names (Projects/, Personal/) instead of opendray's
	// default lowercase. Per-cwd overrides live in an in-vault JSON
	// file managed via the API; these are just the templates.
	PersonalPrefix string `toml:"personal_prefix" json:"personal_prefix"` // default "personal"
	ProjectsPrefix string `toml:"projects_prefix" json:"projects_prefix"` // default "projects"
}

type DatabaseConfig struct {
	URL string `toml:"url" json:"url"`
}

type AdminConfig struct {
	User     string `toml:"user" json:"user"`
	Password string `toml:"password" json:"password"`
	TokenTTL string `toml:"token_ttl" json:"token_ttl"` // e.g. "24h", "12h", "30m"
}

// Duration parses TokenTTL; returns 0 if unset.
func (a AdminConfig) Duration() time.Duration {
	d, _ := time.ParseDuration(a.TokenTTL)
	return d
}

type LogConfig struct {
	Level  string `toml:"level" json:"level"`   // debug|info|warn|error
	Format string `toml:"format" json:"format"` // json|text
	// File is an optional path. When set, every log line is also
	// written there (in addition to stderr). The file rotates at
	// 10 MB and keeps the most recent 5 files. Empty = stderr only.
	File string `toml:"file" json:"file"`
}

// SessionConfig drives the session.Manager idle detector. Empty values
// use Manager defaults (30s threshold, 5s poll interval).
type SessionConfig struct {
	IdleThreshold string `toml:"idle_threshold" json:"idle_threshold"` // e.g. "30s", "2m"
	IdleInterval  string `toml:"idle_interval" json:"idle_interval"`   // e.g. "5s"
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
		cfg.FilePath = path
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

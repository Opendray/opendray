// Package config loads opendray's TOML configuration with environment-variable
// overrides. The TOML file is the human-edited source of truth; env vars
// (prefix OPENDRAY_) override individual fields for 12-factor deploys.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
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
	Memory    MemoryConfig    `toml:"memory" json:"memory"`
	Backup    BackupConfig    `toml:"backup" json:"backup"`

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
// MemoryConfig drives the optional opendray-native memory subsystem
// (the "remember things across sessions" RAG layer exposed as an
// in-process MCP server). Every field is optional; the zero-value
// config is the documented default — BM25 keyword retrieval over
// pgvector storage, project-scoped, no API keys.
//
// The architecture uses two replaceable subsystems:
//
//   - Embedder: turns text into vectors. Built-in BM25 (no model)
//     or HTTP (OpenAI-compatible: ollama / OpenAI / LocalAI / etc.).
//     Phase 2 will add a built-in ONNX model (bge-m3) — operators
//     who want it today can point an HTTP backend at ollama.
//   - Store: persists vectors. pgvector (default, reuses the
//     opendray PG) or chromem-go (single-file, opt-in for setups
//     without pgvector).
type MemoryConfig struct {
	// Backend selects the embedder. "auto" = BM25 with future ONNX
	// upgrade; "bm25" = forced keyword; "http" = OpenAI-compatible
	// HTTP endpoint (configured via [memory.http]).
	Backend string `toml:"backend" json:"backend"`

	// Store selects the vector store. "pgvector" (default; reuses
	// the existing PG) or "chromem" (single-file, no PG dependency).
	Store string `toml:"store" json:"store"`

	// DefaultTopK is the K returned by memory.search when callers
	// don't pass an explicit value. Empty/0 → 5.
	DefaultTopK int `toml:"default_top_k" json:"default_top_k"`

	// SimilarityThreshold (0..1) — minimum cosine similarity for a
	// candidate to count as a match (used both for retrieval cutoff
	// and dedupe-on-insert). Empty → 0.7.
	SimilarityThreshold float64 `toml:"similarity_threshold" json:"similarity_threshold"`

	// DedupThreshold (0..1) — M11. When memory_store finds an
	// existing memory in the same scope with cosine similarity ≥
	// this value, it merges into that row (re-embed + bump
	// metadata.deduped_count) instead of inserting a new one.
	// 0 disables dedup; sensible values: ~0.85 for dense embedders
	// (bge-m3 / ada-002), ~0.4 for BM25. Empty → 0 (off — preserves
	// historical behaviour for fresh installs).
	DedupThreshold float64 `toml:"dedup_threshold" json:"dedup_threshold"`

	// Gatekeeper (M12) — pre-write LLM judge.
	Gatekeeper MemoryGatekeeperConfig `toml:"gatekeeper" json:"gatekeeper"`

	// Cleaner (M13) — periodic LLM librarian that proposes
	// deletions / merges for existing memories.
	Cleaner MemoryCleanerConfig `toml:"cleaner" json:"cleaner"`

	// Local + HTTP backends. Only the active one matters.
	Local MemoryLocalConfig `toml:"local" json:"local"`
	HTTP  MemoryHTTPConfig  `toml:"http" json:"http"`

	// Scope rules for newly stored memories.
	Scope MemoryScopeConfig `toml:"scope" json:"scope"`

	// Chromem path. Only consulted when Store == "chromem". Empty
	// → ~/.opendray/memory/chromem.gob.
	ChromemPath string `toml:"chromem_path" json:"chromem_path"`
}

// MemoryLocalConfig points the LocalONNX embedder at on-disk
// model + tokenizer artifacts. Used when [memory.backend = "local"]
// AND the binary was built with `-tags local_onnx`. Standard
// builds get the stub embedder that always errors.
type MemoryLocalConfig struct {
	// Model is a friendly name for logs / UI. Has no effect on
	// inference — the actual weights come from ModelPath. Common
	// values: "bge-m3", "bge-small-en-v1.5", "nomic-embed-text".
	Model string `toml:"model" json:"model"`
	// LibraryPath is the directory holding libonnxruntime.dylib
	// (macOS) / libonnxruntime.so (Linux). Empty defaults to
	// /opt/homebrew/opt/onnxruntime/lib on Apple Silicon, or
	// /usr/lib on Linux.
	LibraryPath string `toml:"library_path" json:"library_path"`
	// ModelPath is the absolute path to the .onnx weights.
	ModelPath string `toml:"model_path" json:"model_path"`
	// TokenizerPath is the absolute path to tokenizer.json
	// (HuggingFace standard format).
	TokenizerPath string `toml:"tokenizer_path" json:"tokenizer_path"`
	// MaxSeqLen caps input length. Empty / 0 → 512 (bge-m3 default).
	MaxSeqLen int `toml:"max_seq_len" json:"max_seq_len"`
}

// MemoryHTTPConfig points at any OpenAI-compatible /v1/embeddings
// endpoint. Examples:
//
//	BaseURL  = "http://localhost:11434/v1"      Model = "nomic-embed-text"
//	BaseURL  = "https://api.openai.com/v1"      Model = "text-embedding-3-small"
//	BaseURL  = "http://localhost:8080/v1"       Model = "<your local model>"
type MemoryHTTPConfig struct {
	BaseURL    string `toml:"base_url" json:"base_url"`
	Model      string `toml:"model" json:"model"`
	APIKey     string `toml:"api_key" json:"api_key"`
	Dimensions int    `toml:"dimensions" json:"dimensions"`
}

// MemoryScopeConfig governs the visibility model for stored memories.
type MemoryScopeConfig struct {
	// Default scope for memory.store calls when the agent doesn't
	// pass one explicitly. "session" / "project" / "global".
	// Empty → "project".
	Default string `toml:"default" json:"default"`
}

// MemoryCleanerConfig (M13) — the periodic LLM librarian. Off by
// default; flip Enabled when an operator wants automated review.
// The HTTP endpoints (/memory/cleanup/*) work without the
// scheduler so operators can run one-off cleanups by hand even
// when auto-runs are off.
type MemoryCleanerConfig struct {
	// Enabled toggles the auto-run scheduler. The HTTP endpoints
	// (manual /memory/cleanup/run) work either way.
	Enabled bool `toml:"enabled" json:"enabled"`

	// SummarizerID pins which configured summarizer provider runs
	// the librarian judgement. Empty → registry default. Same field
	// shape as the gatekeeper.
	SummarizerID string `toml:"summarizer_id" json:"summarizer_id"`

	// IntervalSeconds between automatic sweeps. Empty / 0 → 86400
	// (24h). Set to a small value (e.g. 300 = 5 min) for testing.
	IntervalSeconds int `toml:"interval_seconds" json:"interval_seconds"`

	// InitialDelaySeconds before the first sweep. Empty / 0 → 300
	// (5 min) so the process is warmed up before judging anything.
	InitialDelaySeconds int `toml:"initial_delay_seconds" json:"initial_delay_seconds"`

	// BatchSize caps memories reviewed per LLM call. Empty / 0 → 30.
	BatchSize int `toml:"batch_size" json:"batch_size"`

	// MinAgeHours skips memories younger than this many hours so
	// the cleaner never reviews something the user just wrote.
	// Empty / 0 → 24.
	MinAgeHours int `toml:"min_age_hours" json:"min_age_hours"`

	// SkipIfDecidedWithinHours avoids re-proposing decisions for the
	// same memory_id within this window after an earlier verdict.
	// Empty / 0 → 168 (7 days).
	SkipIfDecidedWithinHours int `toml:"skip_if_decided_within_hours" json:"skip_if_decided_within_hours"`

	// CallTimeoutMs caps each LLM call. Reasoning models on local
	// LM Studio can take 10-30s for a 30-row batch. Empty / 0 →
	// 60000 (60s).
	CallTimeoutMs int `toml:"call_timeout_ms" json:"call_timeout_ms"`

	// IncludeGlobalScope sweeps the global memory scope in addition
	// to project scope. Default false — global memories are usually
	// operator-curated and a librarian sweep there feels invasive
	// until the operator has trust in the cleaner.
	IncludeGlobalScope bool `toml:"include_global_scope" json:"include_global_scope"`
}

// MemoryGatekeeperConfig (M12) — pre-write LLM judge that decides
// whether a memory_store call carries a durable fact or noise.
type MemoryGatekeeperConfig struct {
	// Enabled flips the feature. Default false — the gatekeeper
	// adds a per-store LLM round-trip (~200ms with LM Studio), and
	// noisy writes are a tolerable problem until operators see the
	// payoff in their memory list.
	Enabled bool `toml:"enabled" json:"enabled"`

	// SummarizerID picks which configured summarizer provider runs
	// the judgement. Empty → use the registry default (whatever
	// memory_summarizer_providers.is_default = true points at).
	SummarizerID string `toml:"summarizer_id" json:"summarizer_id"`

	// MaxLatencyMs caps the per-call timeout. Above this the
	// gatekeeper logs and degrades to "allow" — better to let
	// the write through than block on a slow LLM. Empty → 2000.
	MaxLatencyMs int `toml:"max_latency_ms" json:"max_latency_ms"`
}

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
//	HistoryRoots:   [~/.claude/projects, ~/.claude-accounts/*/projects]
//	               — both are scanned and deduped via EvalSymlinks
//	AccountsDir:    ~/.claude-accounts
//	               — root used when creating a new account without an
//	                 explicit ConfigDir
//	WatcherEnabled: true (zero value is "watch")
//	               — when true, an fsnotify-backed watcher under
//	                 AccountsDir auto-registers a new account row when
//	                 `<dir>/<name>/.credentials.json` appears (the
//	                 result of `CLAUDE_CONFIG_DIR=<dir> claude login`).
//	                 Set to false to disable the watcher; the UI's
//	                 "Import local" button still works.
type ClaudeProviderConfig struct {
	HistoryRoots        []string `toml:"history_roots" json:"history_roots"`
	AccountsDir         string   `toml:"accounts_dir" json:"accounts_dir"`
	WatcherEnabled      *bool    `toml:"watcher_enabled" json:"watcher_enabled"`
	AutoFailoverEnabled *bool    `toml:"auto_failover_enabled" json:"auto_failover_enabled"`
}

// AutoFailoverIsEnabled returns the effective state of the rate-limit-
// auto-failover feature (Phase 2 Tier A): nil pointer (omitted in config)
// or explicit false → disabled; explicit true → enabled. Disabled by
// default because the behavior — silently switching a live session to
// a different Anthropic identity when it hits a rate limit — needs an
// operator opt-in: it changes billing attribution and rate-limit pool
// without the user clicking anything.
func (c ClaudeProviderConfig) AutoFailoverIsEnabled() bool {
	if c.AutoFailoverEnabled == nil {
		return false
	}
	return *c.AutoFailoverEnabled
}

// WatcherIsEnabled returns the effective accounts-watcher state:
// nil pointer (omitted in config) → true; explicit false → disabled.
// Keeps the zero value of ClaudeProviderConfig in "watch" mode so a
// fresh install Just Works.
func (c ClaudeProviderConfig) WatcherIsEnabled() bool {
	if c.WatcherEnabled == nil {
		return true
	}
	return *c.WatcherEnabled
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
	// MaxConns caps the pgx connection pool. Empty / 0 → store defaults
	// (16). Tune up for high-fanout integration traffic; tune down on
	// shared PG instances where opendray must not crowd out peers.
	MaxConns int `toml:"max_conns" json:"max_conns"`
}

// BackupConfig drives the disaster-recovery backup + admin export
// feature (internal/backup). The master encryption passphrase is
// NOT in the toml; it lives only in env OPENDRAY_BACKUP_KEY so it
// can never be checked in or read off disk by accident.
//
// Defaults when fields are empty:
//
//	enabled       = false           — feature off by default
//	local_dir     = ~/.opendray/backups   — first writable target's root
//	export_dir    = ~/.opendray/exports   — staging dir for /export bundles
//	pg_dump_path  = ""              — resolved from PATH at startup
type BackupConfig struct {
	Enabled       bool   `toml:"enabled" json:"enabled"`
	LocalDir      string `toml:"local_dir" json:"local_dir"`
	ExportDir     string `toml:"export_dir" json:"export_dir"`
	PgDumpPath    string `toml:"pg_dump_path" json:"pg_dump_path"`
	PgRestorePath string `toml:"pg_restore_path" json:"pg_restore_path"`
}

type AdminConfig struct {
	User     string `toml:"user" json:"user"`
	Password string `toml:"password" json:"password"`
	TokenTTL string `toml:"token_ttl" json:"token_ttl"` // e.g. "24h", "12h", "30m"
	// MobileTokenTTL is the absolute lifetime of bearer tokens issued
	// to the mobile app via /api/v1/auth/mobile-login. Empty/0 falls
	// back to a 30-day default (vs 24h for browser tokens) so users
	// don't have to re-enter their password every day on devices that
	// already gate access behind biometrics + secure storage.
	MobileTokenTTL string `toml:"mobile_token_ttl" json:"mobile_token_ttl"`
}

// Duration parses TokenTTL; returns 0 if unset.
func (a AdminConfig) Duration() time.Duration {
	d, _ := time.ParseDuration(a.TokenTTL)
	return d
}

// MobileDuration parses MobileTokenTTL; returns 0 if unset.
func (a AdminConfig) MobileDuration() time.Duration {
	d, _ := time.ParseDuration(a.MobileTokenTTL)
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
	if v := os.Getenv("OPENDRAY_DATABASE_MAX_CONNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.Database.MaxConns = n
		}
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
	if v := os.Getenv("OPENDRAY_ADMIN_MOBILE_TOKEN_TTL"); v != "" {
		cfg.Admin.MobileTokenTTL = v
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
	if v := os.Getenv("OPENDRAY_BACKUP_ENABLED"); v == "1" || v == "true" {
		cfg.Backup.Enabled = true
	}
	if v := os.Getenv("OPENDRAY_BACKUP_LOCAL_DIR"); v != "" {
		cfg.Backup.LocalDir = v
	}
	if v := os.Getenv("OPENDRAY_BACKUP_EXPORT_DIR"); v != "" {
		cfg.Backup.ExportDir = v
	}
	if v := os.Getenv("OPENDRAY_BACKUP_PG_DUMP_PATH"); v != "" {
		cfg.Backup.PgDumpPath = v
	}
	if v := os.Getenv("OPENDRAY_BACKUP_PG_RESTORE_PATH"); v != "" {
		cfg.Backup.PgRestorePath = v
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
	if c.Admin.MobileTokenTTL != "" {
		if _, err := time.ParseDuration(c.Admin.MobileTokenTTL); err != nil {
			return fmt.Errorf("config: admin.mobile_token_ttl: %w", err)
		}
	}
	return nil
}

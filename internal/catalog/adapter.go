package catalog

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/opendray/opendray-v2/internal/agyacct"
	"github.com/opendray/opendray-v2/internal/cliacct"
	"github.com/opendray/opendray-v2/internal/mcp"
	"github.com/opendray/opendray-v2/internal/session"
	"github.com/opendray/opendray-v2/internal/skills"
)

// resolveExecutable expands magic tokens in a manifest's executable
// field. Currently:
//
//	$SHELL  → user's interactive shell from env, falling back to /bin/bash.
//
// Used so the bundled "shell" provider follows whatever the operator's
// account is configured for (zsh on modern macOS, bash on Linux, …)
// instead of always launching /bin/bash.
func resolveExecutable(raw string) string {
	switch raw {
	case "$SHELL":
		if s := os.Getenv("SHELL"); s != "" {
			return s
		}
		return "/bin/bash"
	default:
		return raw
	}
}

// SessionProvider adapts Catalog (and the optional Claude account
// service) to session.ProviderResolver. The session.Manager owns
// spawn-time scratch dirs; SessionProvider only supplies a Prepare
// callback that writes per-session MCP config into that scratch dir,
// reads the bound Claude account's OAuth token from disk, materialises
// enabled agent skills, and contributes the args/env the provider's
// CLI needs to pick the values up.
type SessionProvider struct {
	cat         *Catalog
	accounts    *cliacct.Service // optional; nil disables claude multi-account
	agyAccounts *agyacct.Service // optional; nil disables antigravity multi-account
	skills      *skills.Loader   // optional; nil disables skill injection
	mcps        *mcp.Loader      // optional; nil disables vault MCP injection
	secretsFile string           // dotenv file for ${KEY} substitution; empty = no substitution
	log         *slog.Logger

	// memory describes the auto-attached memory MCP server. Zero
	// value (Enabled=false) skips injection. Set via
	// WithMemoryAutoAttach when memory + an integration token are
	// available — otherwise we'd render an MCP server config the
	// agent can't authenticate.
	memory MemoryAutoAttach

	// dbtool describes the auto-attached Database-tool MCP server.
	// Same lifecycle as memory: zero value skips injection, set via
	// WithDbtoolAutoAttach when the feature is enabled and a key was
	// minted.
	dbtool DbtoolAutoAttach

	// memoryMirror, when set, is invoked from a background goroutine
	// right after the session's PrepareFunc returns — it pulls
	// Claude's local .claude/.../memory/*.md files into the shared
	// opendray pgvector store so cross-CLI search picks them up.
	// Nil → no mirroring (memory disabled, or mirror not wired).
	memoryMirror MemoryMirrorFunc

	// ambientInjector, when set, renders a markdown banner of
	// recent project memories into the system prompt at spawn
	// time. Backed by internal/memory/injector. Nil → no injection.
	ambientInjector AmbientInjector

	// projectDocInjector, when set, renders the cross-agent project
	// goal + plan + recent journal as a system-prompt banner. Backed
	// by internal/projectdoc.Service.RenderForSpawn. Nil → no
	// injection (e.g. older builds without memory layers 2-4).
	projectDocInjector ProjectDocInjector

	// knowledgeInjector, when set, prepends a compact "Project
	// knowledge" banner (skills + playbooks) at spawn. Backed by
	// internal/knowledge.Service. Nil → no injection ([knowledge] off).
	knowledgeInjector KnowledgeInjector

	// projectDocBudget caps the rendered banner size in bytes. 0
	// disables the cap (legacy behaviour). Operators dial this via
	// WithProjectDocBudget; the catalog adapter calls
	// RenderForSpawnWithBudget when non-zero. Default 4096 — about
	// 1k tokens, plenty for a one-screen banner.
	projectDocBudget int

	// projectScanner, when set, auto-detects tech stack + structure
	// at spawn time (only re-scans when the existing tech_stack doc
	// is older than scannerMaxAge). Nil → no auto-scan, operator
	// must POST /project-scan/run manually.
	projectScanner ProjectScanner

	// scannerMaxAge controls when a stale tech_stack doc triggers
	// a re-scan. Default 6h.
	scannerMaxAge time.Duration

	// gitActivity, when set, kicks off a background refresh of the
	// recent_activity doc when the cached one is older than
	// gitActivityMaxAge. Done async — we don't block spawn on a
	// 60-150s LLM call.
	gitActivity       GitActivityRefresher
	gitActivityMaxAge time.Duration
}

// AmbientInjector is the contract internal/memory/injector
// satisfies — kept here so catalog stays import-decoupled from the
// memory package.
type AmbientInjector interface {
	Render(ctx context.Context, sessionID, cwd string) (string, error)
}

// ProjectDocInjector is the contract internal/projectdoc.Service
// satisfies. Returns a rendered markdown banner combining the
// project goal, plan, tech_stack, and recent journal entries;
// empty string means "nothing to inject — skip silently".
//
// The catalog adapter calls RenderForSpawnWithBudget when an
// operator opts into a byte cap; otherwise it falls back to the
// legacy RenderForSpawn (no cap). Implementations must support
// both shapes to stay forward-compatible.
type ProjectDocInjector interface {
	RenderForSpawn(ctx context.Context, cwd string, recentLogs int) (string, error)
	RenderForSpawnWithBudget(ctx context.Context, cwd string, recentLogs, maxBytes int) (string, error)
}

// KnowledgeInjector is the contract internal/knowledge.Service satisfies —
// renders a compact "Project knowledge" banner (skills + playbooks) for the
// spawning agent. Empty string means nothing to inject.
type KnowledgeInjector interface {
	RenderForSpawn(ctx context.Context, cwd string, maxBytes int) (string, error)
}

// ProjectScanner is the contract internal/projectscan.Service
// satisfies. The catalog adapter calls Run at spawn time (when the
// stored tech_stack doc is older than maxAge) so a fresh agent sees
// the current tech stack + structure without re-indexing the repo.
// Errors are best-effort — failure to scan shouldn't block the
// spawn.
type ProjectScanner interface {
	Run(ctx context.Context, cwd string) error
	IsStale(ctx context.Context, cwd string, maxAge time.Duration) bool
}

// GitActivityRefresher is the contract internal/gitactivity.Service
// satisfies. Same shape as ProjectScanner: at spawn time, if the
// recent_activity doc is stale, the catalog kicks off a refresh in
// a background goroutine (not sync — git+LLM takes 60-150s and we
// don't want to block PTY allocation that long). The next spawn,
// or a polling UI, will see the refreshed doc.
type GitActivityRefresher interface {
	IsStale(ctx context.Context, cwd string, maxAge time.Duration) bool
	RefreshAsync(cwd string)
}

// MemoryMirrorFunc syncs Claude's local memory files for the given
// cwd into opendray's pgvector store. The catalog package keeps a
// function reference rather than the concrete *memory.Mirror so the
// import graph stays one-directional — internal/memory imports
// internal/catalog would create a cycle, since catalog already
// imports many other packages.
type MemoryMirrorFunc func(ctx context.Context, cwd string) (int, error)

// MemoryAutoAttach holds the runtime knobs the SessionProvider
// uses to inject opendray's memory MCP into every spawned session.
// All fields are required when Enabled is true; the catalog adapter
// errors out at spawn time if any are missing.
type MemoryAutoAttach struct {
	// Enabled toggles the whole feature. When false, no memory MCP
	// server is added to the rendered mcp.json.
	Enabled bool
	// BinaryPath is the absolute path to the opendray executable.
	// The MCP subprocess is launched as `<BinaryPath> mcp-memory`.
	// Resolved at startup via os.Executable so the agent doesn't
	// rely on $PATH.
	BinaryPath string
	// BaseURL is the gateway origin the MCP subprocess calls back
	// for /api/v1/admin/memory/*. Usually `http://127.0.0.1:<port>`.
	BaseURL string
	// APIKey is the bearer the subprocess uses to authenticate.
	// opendray mints this at startup as a dedicated integration key.
	APIKey string
	// Scope determines the visibility band ("session", "project",
	// "global"). Defaults to "project" when empty.
	Scope string
}

func NewSessionProvider(
	cat *Catalog,
	accounts *cliacct.Service,
	agyAccounts *agyacct.Service,
	skills *skills.Loader,
	mcps *mcp.Loader,
	secretsFile string,
	log *slog.Logger,
) *SessionProvider {
	if log == nil {
		log = slog.Default()
	}
	return &SessionProvider{
		cat:         cat,
		accounts:    accounts,
		agyAccounts: agyAccounts,
		skills:      skills,
		mcps:        mcps,
		secretsFile: secretsFile,
		log:         log.With("component", "catalog.session"),
	}
}

// WithMemoryAutoAttach enables auto-injection of opendray's memory
// MCP server into every spawned session's rendered mcp.json. Pass
// MemoryAutoAttach{Enabled: false} to turn the feature off (the
// default). Returns the receiver for fluent setup at app startup.
func (sp *SessionProvider) WithMemoryAutoAttach(cfg MemoryAutoAttach) *SessionProvider {
	sp.memory = cfg
	return sp
}

// DbtoolAutoAttach holds the runtime knobs for injecting the
// Database-tool MCP server (`opendray mcp-dbtool`) into spawned
// sessions. Field semantics mirror MemoryAutoAttach; the key is a
// dedicated `opendray-dbtool` integration carrying db:read/db:write —
// deliberately NOT the memory key, so neither key's blast radius grows.
type DbtoolAutoAttach struct {
	Enabled    bool
	BinaryPath string
	BaseURL    string
	// APIKey is the SIGNED key (db:signed) used by providers whose MCP
	// config is per-session; those spawns also get a per-cwd HMAC
	// signature the gateway verifies.
	APIKey string
	// AgyAPIKey is the honest-path key for antigravity, whose MCP config
	// is HOME-global and can't carry a per-session signature.
	AgyAPIKey string
	// SignSecret signs the per-session cwd binding (HMAC-SHA256). Shared
	// with the gateway's verifier; never injected into a session.
	SignSecret []byte
}

// WithDbtoolAutoAttach enables auto-injection of the Database-tool MCP
// server into every spawned session. Zero value = off.
func (sp *SessionProvider) WithDbtoolAutoAttach(cfg DbtoolAutoAttach) *SessionProvider {
	sp.dbtool = cfg
	return sp
}

// signDbtoolCwd is hex HMAC-SHA256(secret, cwd) — the per-session cwd
// proof injected for signed-key providers. It MUST stay byte-identical to
// the gateway's verifier (internal/dbtool.verifyCwdSig).
func signDbtoolCwd(secret []byte, cwd string) string {
	m := hmac.New(sha256.New, secret)
	m.Write([]byte(cwd))
	return hex.EncodeToString(m.Sum(nil))
}

// WithAmbientInjector installs the ambient-memory injector — used
// at spawn time to prepend a "Recent project memory" banner to the
// agent's system prompt. Returns the receiver for chained setup.
func (sp *SessionProvider) WithAmbientInjector(inj AmbientInjector) *SessionProvider {
	sp.ambientInjector = inj
	return sp
}

// WithProjectDocInjector installs the cross-agent project-doc
// injector — prepends a "Project context" banner (goal + plan +
// recent journal) to the agent's system prompt at spawn time.
func (sp *SessionProvider) WithProjectDocInjector(inj ProjectDocInjector) *SessionProvider {
	sp.projectDocInjector = inj
	return sp
}

// WithKnowledgeInjector installs the M-KG knowledge injector — prepends a
// compact "Project knowledge" (skills + playbooks) banner at spawn time.
func (sp *SessionProvider) WithKnowledgeInjector(inj KnowledgeInjector) *SessionProvider {
	sp.knowledgeInjector = inj
	return sp
}

// WithProjectDocBudget caps the rendered banner size in bytes.
// 0 disables the cap (legacy behaviour). Operators tune this when
// the unbudgeted banner pushes spawn prompts past model context
// limits; default 4096 (≈1k tokens) is a reasonable cap for
// typical projects.
func (sp *SessionProvider) WithProjectDocBudget(maxBytes int) *SessionProvider {
	sp.projectDocBudget = maxBytes
	return sp
}

// WithProjectScanner installs the project scanner. When the
// stored tech_stack doc for the spawning session's cwd is older
// than maxAge (or missing), Run is called synchronously so the
// freshly-scanned info ends up in the spawn-time banner. Set
// maxAge=0 to use the default 6h.
func (sp *SessionProvider) WithProjectScanner(scanner ProjectScanner, maxAge time.Duration) *SessionProvider {
	sp.projectScanner = scanner
	if maxAge <= 0 {
		maxAge = 6 * time.Hour
	}
	sp.scannerMaxAge = maxAge
	return sp
}

// WithGitActivityRefresher installs the git activity refresher.
// At spawn time we check if the recent_activity doc is stale and,
// if so, kick off the refresh asynchronously. The first spawn
// after a stale doc still sees the *previous* summary in its
// banner — the freshly generated one lands moments later for the
// next spawn or polling client. We trade banner freshness for not
// blocking the agent's PTY allocation behind a 60-150s LLM call.
// maxAge=0 uses the default 12h.
func (sp *SessionProvider) WithGitActivityRefresher(r GitActivityRefresher, maxAge time.Duration) *SessionProvider {
	sp.gitActivity = r
	if maxAge <= 0 {
		maxAge = 12 * time.Hour
	}
	sp.gitActivityMaxAge = maxAge
	return sp
}

// WithMemoryMirror installs a function that ingests Claude's local
// memory files into the shared store. Called from a goroutine on
// every session spawn so the agent's MCP search sees yesterday's
// notes without manual setup.
func (sp *SessionProvider) WithMemoryMirror(fn MemoryMirrorFunc) *SessionProvider {
	sp.memoryMirror = fn
	return sp
}

func (sp *SessionProvider) Resolve(ctx context.Context, id string) (session.ProviderInfo, error) {
	p, err := sp.cat.Get(ctx, id)
	if err != nil {
		return session.ProviderInfo{}, err
	}
	if !p.Enabled {
		return session.ProviderInfo{}, fmt.Errorf("%w: %s is disabled", session.ErrProviderUnavailable, id)
	}

	// User config "command" override always wins; otherwise resolve
	// magic tokens (e.g. "$SHELL") in the manifest's executable.
	exe := resolveExecutable(p.Manifest.Executable)
	if v, ok := p.Config["command"].(string); ok && v != "" {
		exe = v
	}
	args := append([]string(nil), p.Manifest.DefaultArgs...)
	// Translate ConfigSchema → CLI args/env (cliFlag, cliValue, envVar,
	// extraArgs). Stable iteration order: schema definition order in the
	// manifest, which keeps spawn args reproducible across restarts.
	configArgs, configEnv := applyConfigSchema(p.Manifest.ConfigSchema, p.Config)
	args = append(args, configArgs...)
	// Model → e.g. `--model X`. A per-session pin (Session.Model, threaded
	// via context) wins over the per-provider operator default.
	args = append(args, modelArgs(p.Manifest, p.Config, session.ModelFromContext(ctx))...)
	if p.Manifest.ID == "codex" {
		if session.PermissionModeFromContext(ctx) == "bypass" {
			// codex's full bypass flag (--dangerously-bypass-approvals-and-
			// sandbox, appended below) is a clap ArgGroup mutually exclusive
			// with --ask-for-approval (-a) and --sandbox (-s). Those land in
			// args from the provider's saved config schema, so strip them
			// here — otherwise codex rejects the combo and the session exits
			// within seconds of spawn. dropConflictingFlags can't catch this
			// because it only fires on USER-supplied trigger flags, but for
			// integration sessions the bypass flag is provider-side.
			args = stripFlagsWithValues(args, "--ask-for-approval", "-a", "--sandbox", "-s")
		} else if approval, ok := p.Config["approval"].(string); ok && approval != "" {
			// Non-bypass: apply the operator's approval policy.
			args = append(args, "-c", "approval_policy="+tomlString(approval))
		}
	}
	// Integration bypass: append the provider's unattended/auto-approve
	// flag when the creating integration's spawn profile set
	// permission_mode=bypass, so its sessions run without a human to
	// approve tool calls.
	if session.PermissionModeFromContext(ctx) == "bypass" {
		args = append(args, bypassArgsFor(p.Manifest.ID)...)
	}

	info := session.ProviderInfo{
		ID:         p.Manifest.ID,
		Executable: exe,
		Args:       args,
		Conflicts:  providerConflicts(p.Manifest.ID),
	}

	// Account selection. claude isolates accounts via CLAUDE_CONFIG_DIR;
	// antigravity isolates via HOME (agy keys all state off $HOME). Both
	// arrive through the same generic session.AccountID — the session
	// manager set it from the provider-appropriate field at spawn.
	selectedAccountID := session.AccountID(ctx)
	wantClaudeAccount := id == "claude" && selectedAccountID != "" && sp.accounts != nil
	wantAgyAccount := id == "antigravity" && selectedAccountID != "" && sp.agyAccounts != nil

	// Merge vault MCP registry (enabled-only) into the provider's
	// inline mcp_servers list. Vault entries are loaded eagerly here
	// (cheap — small JSON reads) so the spawn-decision branch below
	// can short-circuit when there are zero servers in either tier.
	//
	// Precedence on `name` collision: provider config wins. Lets users
	// override a vault entry per-provider without editing the registry.
	servers := mergeMCPServers(loadVaultMCPs(sp.mcps, sp.log), parseMCPServers(p.Config))
	// Integration-scoped MCP servers: declared on the integration that
	// created this session (provider-agnostic), merged after vault+config
	// so the integration's own tools attach to whatever provider the
	// operator pointed it at. Integration entries win on name collision.
	//
	// Antigravity exception: agy's only MCP surface is the HOME-global
	// mcp_config.json (see renderMCP), and a spawn without an account
	// binding uses the gateway user's REAL HOME — writing an
	// integration's servers (with their resolved credentials) there
	// would expose them to the operator's own agy sessions and to every
	// other integration sharing that HOME. So integration MCP reaches
	// antigravity only when the spawn is account-bound (dedicated HOME
	// = per-account isolation); otherwise the servers are dropped with
	// a warning and the operator should bind the integration to an
	// antigravity account (or point it at claude, whose MCP surface is
	// per-session).
	if raw := session.IntegrationMCPServersFromContext(ctx); raw != "" {
		accountBound := session.AccountID(ctx) != "" && sp.agyAccounts != nil
		if integrationMCPAllowed(id, accountBound) {
			servers = mergeMCPServers(servers, parseMCPServersJSON(raw))
		} else {
			sp.log.Warn("dropping integration MCP servers: antigravity spawn is not account-bound, refusing to write integration credentials into the shared HOME-global mcp_config.json",
				"provider", id)
		}
	}
	// Auto-attach opendray's memory MCP server when enabled at app
	// startup. This is what makes "the agent remembers things across
	// sessions" actually work without per-CLI manual setup.
	//
	// Exception: third-party integration sessions are isolated. They are
	// self-managed — they bring their own tools and own their data — so
	// we never auto-attach the cross-project memory MCP, which would let
	// the agent read other projects' facts/journal/docs. Operator and CLI
	// sessions are unaffected (project data is already cwd-scoped; the
	// shared global/KB layer stays available to them).
	isIntegration := session.OriginFromContext(ctx) == session.OriginIntegration
	// The agent calls memory_* as native MCP tools on every MCP-capable
	// provider, antigravity included: agy loads servers from its global
	// <$HOME>/.gemini/config/mcp_config.json, which renderMCP merges our
	// entry into. (An earlier release drove agy through an `opendray
	// memory` CLI shim because we believed agy had no MCP config surface
	// — that belief came from testing the wrong file; see the renderMCP
	// antigravity arm.)
	if sp.memory.Enabled && p.Manifest.Capabilities.SupportsMcp && !isIntegration {
		servers = append(servers, MCPServer{
			Name:    "opendray-memory",
			Command: sp.memory.BinaryPath,
			Args:    []string{"mcp-memory"},
			Env: map[string]string{
				"OPENDRAY_BASE_URL":     sp.memory.BaseURL,
				"OPENDRAY_API_KEY":      sp.memory.APIKey,
				"OPENDRAY_MEMORY_SCOPE": defaultStr(sp.memory.Scope, "project"),
				// Scope key is the cwd at spawn time — populated below
				// inside Prepare since we need access to the live
				// session.Cwd from context. (Except antigravity, whose
				// entry lands in a HOME-global file shared across
				// sessions: there the mcp-memory subprocess derives the
				// scope from its own cwd, which agy sets to the session
				// workspace.)
			},
		})
	}
	// Auto-attach the Database-tool MCP server under the same rules as
	// memory: MCP-capable provider, never for isolated integration
	// sessions (an integration must not reach the operator's registered
	// project databases through a spawned session's ambient tools).
	if sp.dbtool.Enabled && p.Manifest.Capabilities.SupportsMcp && !isIntegration {
		servers = append(servers, MCPServer{
			Name:    "opendray-dbtool",
			Command: sp.dbtool.BinaryPath,
			Args:    []string{"mcp-dbtool"},
			Env: map[string]string{
				"OPENDRAY_BASE_URL": sp.dbtool.BaseURL,
				"OPENDRAY_API_KEY":  sp.dbtool.APIKey,
				// Project key (the session cwd) is populated inside
				// Prepare, exactly like memory's scope key.
			},
		})
	}
	mcpEnabled := p.Manifest.Capabilities.SupportsMcp && len(servers) > 0

	// Skill injection: enabled by default for providers in the safe
	// list (claude, antigravity). Codex is opt-in via skills_enabled=true
	// because its only injection path is CODEX_HOME, which clobbers
	// ChatGPT-OAuth auth.
	skillsEnabled := sp.skills != nil && providerSupportsSkills(id)
	if v, ok := p.Config["skills_disabled"].(bool); ok && v {
		skillsEnabled = false
	}
	if v, ok := p.Config["skills_enabled"].(bool); ok && v && sp.skills != nil {
		skillsEnabled = true
	}

	// OpenCode reaches its local-endpoint provider config only through the
	// generated OPENCODE_CONFIG written in Prepare. Make sure we don't take
	// the no-Prepare fast path when the operator configured a local
	// endpoint but happened to disable skills and memory MCP.
	wantsOpenCodeConfig := wantsOpenCodeSessionConfig(id, p.Config)

	// An integration system prompt is injected inside the Prepare
	// closure (same surface as ambient memory), so it must not take the
	// no-Prepare fast path even when nothing else needs one.
	hasIntegrationPrompt := session.IntegrationSystemPromptFromContext(ctx) != ""

	if !wantClaudeAccount && !wantAgyAccount && !mcpEnabled && !skillsEnabled && len(configEnv) == 0 && !wantsOpenCodeConfig && !hasIntegrationPrompt {
		return info, nil
	}

	providerID := p.Manifest.ID
	info.Prepare = func(prepareCtx context.Context, _, baseDir string) (session.PrepareOutput, error) {
		out := session.PrepareOutput{Env: map[string]string{}}

		// Schema-derived env (e.g. ANTHROPIC_API_KEY when authType=custom)
		// applied first so later branches — multi-account claude, MCP —
		// can override deliberately.
		for k, v := range configEnv {
			out.Env[k] = v
		}

		// OpenCode: register the operator's local OpenAI-compatible
		// endpoint (if configured) as a provider in the generated
		// OPENCODE_CONFIG and default-select it. Runs before skills/MCP so
		// all three merge into the same per-session config file.
		if providerID == "opencode" {
			if err := injectOpenCodeLocalProvider(prepareCtx, baseDir, p.Config, &out); err != nil {
				return session.PrepareOutput{}, fmt.Errorf("inject opencode local provider: %w", err)
			}
		}

		// M21 — Pre-assign the agent-side session UUID so the M18
		// transcript reader can locate the *.jsonl file directly,
		// instead of falling back to "latest mtime in dir" which
		// picks up unrelated active conversations. Claude Code
		// accepts `--session-id <uuid>`; Codex does not,
		// so it stays on the cwd-based reader path. injectSessionIDFor
		// mutates out.Args + out.ClaudeSessionID directly, and the
		// session manager picks up the UUID for persistence. When this
		// is a reactivation carrying an existing agent UUID, it emits
		// `--resume <id>` instead so the prior transcript continues.
		injectSessionIDFor(prepareCtx, providerID, &out)

		if wantClaudeAccount {
			configDir, token, err := sp.accounts.ResolveSpawnCreds(prepareCtx, selectedAccountID)
			if err != nil {
				return session.PrepareOutput{}, fmt.Errorf("claude account %s: %w", selectedAccountID, err)
			}
			// Static token only for legacy token-file accounts; config-dir
			// accounts authenticate via CLAUDE_CONFIG_DIR/.credentials.json,
			// which Claude Code refreshes on its own.
			if token != "" {
				out.Env["CLAUDE_CODE_OAUTH_TOKEN"] = token
			}
			if configDir != "" {
				// Point Claude Code at the account's persistent config
				// dir directly. Earlier attempts to materialise skills
				// here via symlinks broke first-run / auth state because
				// Claude Code rewrites .claude.json atomically (replacing
				// any symlink with a fresh small file). We now keep the
				// account dir untouched and inject skills purely via the
				// --append-system-prompt CLI flag below.
				out.Env["CLAUDE_CONFIG_DIR"] = configDir
			}
		}

		if wantAgyAccount {
			// agy keys its entire credential + conversation state off
			// $HOME, so binding to an account = pointing HOME at the
			// account's dedicated dir (which holds its own
			// .gemini/antigravity-cli/antigravity-oauth-token).
			home, err := sp.agyAccounts.ResolveSpawnHome(prepareCtx, selectedAccountID)
			if err != nil {
				return session.PrepareOutput{}, fmt.Errorf("antigravity account %s: %w", selectedAccountID, err)
			}
			out.Env["HOME"] = home
			// Share the (large) Playwright browser cache across accounts
			// instead of re-downloading it per HOME. Best-effort: a
			// failure just means this account fetches its own copy.
			if err := ensureAgySharedCache(home); err != nil {
				sp.log.Warn("antigravity shared-cache symlink failed",
					"home", home, "err", err)
			}
		}

		// Tier 1 skill index injected per-provider. The agent sees a
		// short line per skill (~30 tokens) and pulls full SKILL.md
		// lazily via `opendray skill describe <id>` through its Bash
		// tool. Each CLI has a different injection surface — the
		// dispatch lives in injectSkillsFor below.
		if skillsEnabled {
			loaded, err := sp.skills.List()
			if err != nil {
				return session.PrepareOutput{}, fmt.Errorf("load skills: %w", err)
			}
			if len(loaded) > 0 {
				if err := injectSkillsFor(providerID, baseDir, loaded, &out); err != nil {
					return session.PrepareOutput{}, fmt.Errorf("inject skills: %w", err)
				}
			}
		}

		// Inject memory guidance into the agent's system prompt. Without
		// this nudge, the CLI tends to use its built-in markdown memory
		// instead of our shared store, defeating the cross-CLI value prop.
		// Done here (after skills, before MCP rendering) so message
		// ordering stays predictable.
		if sp.memory.Enabled && p.Manifest.Capabilities.SupportsMcp && !isIntegration {
			if err := injectMemoryGuidanceFor(providerID, baseDir, &out); err != nil {
				return session.PrepareOutput{}, fmt.Errorf("inject memory guidance: %w", err)
			}
		}

		// Ambient memory: pull a markdown banner of recent project
		// memories from the injector and prepend it to the system
		// prompt. Same per-CLI dispatch as memory guidance — claude
		// gets another --append-system-prompt arg, codex appends to
		// AGENTS.md, antigravity to its --add-dir'd AGENTS.md. Empty rendered text means
		// the operator's profile says "none" or there are no
		// memories yet; we silently skip.
		if sp.ambientInjector != nil {
			cwd := session.Cwd(prepareCtx)
			sessID := session.SessionIDFromContext(prepareCtx)
			text, err := sp.ambientInjector.Render(prepareCtx, sessID, cwd)
			if err != nil {
				sp.log.Warn("ambient memory render failed; skipping inject",
					"session_id", sessID, "cwd", cwd, "err", err)
			} else if text != "" {
				if err := injectAmbientMemoryFor(providerID, baseDir, text, &out); err != nil {
					return session.PrepareOutput{}, fmt.Errorf("inject ambient memory: %w", err)
				}
			}
		}

		// Cross-agent project context: goal + plan + recent journal
		// (memory layers 2-4) + tech stack (M16 scanner). Injected
		// through the same per-CLI channel as ambient memory so the
		// agent sees one composite system prompt. Failures here are
		// non-fatal — a missing banner is better than a failed spawn.
		if sp.projectDocInjector != nil {
			cwd := session.Cwd(prepareCtx)
			// Trigger a fresh project scan when the cached tech_stack
			// is stale. Runs synchronously so the renderer below
			// pulls in the latest info. Failure is logged, not
			// propagated — a stale or missing tech_stack section is
			// still better than blocking the spawn.
			if cwd != "" && sp.projectScanner != nil {
				if sp.projectScanner.IsStale(prepareCtx, cwd, sp.scannerMaxAge) {
					if err := sp.projectScanner.Run(prepareCtx, cwd); err != nil {
						sp.log.Warn("project scanner failed; spawn continues with stale tech_stack",
							"cwd", cwd, "err", err)
					}
				}
			}
			// Git activity refresher — async because the LLM step is
			// slow (60-150s). The current spawn sees the previous
			// summary in its banner; the freshly generated one lands
			// in time for the next spawn.
			if cwd != "" && sp.gitActivity != nil {
				if sp.gitActivity.IsStale(prepareCtx, cwd, sp.gitActivityMaxAge) {
					sp.gitActivity.RefreshAsync(cwd)
				}
			}
			if cwd != "" {
				// M-PB — when WithProjectDocBudget is set, route to the
				// budgeted renderer so the banner can't blow past the
				// configured byte cap. Zero budget keeps the legacy
				// unconstrained shape.
				var (
					text string
					err  error
				)
				if sp.projectDocBudget > 0 {
					text, err = sp.projectDocInjector.RenderForSpawnWithBudget(prepareCtx, cwd, 5, sp.projectDocBudget)
				} else {
					text, err = sp.projectDocInjector.RenderForSpawn(prepareCtx, cwd, 5)
				}
				if err != nil {
					sp.log.Warn("project doc render failed; skipping inject",
						"cwd", cwd, "err", err)
				} else if text != "" {
					if err := injectAmbientMemoryFor(providerID, baseDir, text, &out); err != nil {
						return session.PrepareOutput{}, fmt.Errorf("inject project docs: %w", err)
					}
				}
			}
		}

		// M-KG — prepend the project's distilled knowledge (skills +
		// playbooks). Best-effort; active only when the injector is wired
		// (i.e. [knowledge] enabled). Nil/empty → skip silently.
		if sp.knowledgeInjector != nil {
			if cwd := session.Cwd(prepareCtx); cwd != "" {
				if text, err := sp.knowledgeInjector.RenderForSpawn(prepareCtx, cwd, 4096); err != nil {
					sp.log.Warn("knowledge render failed; skipping inject", "cwd", cwd, "err", err)
				} else if text != "" {
					if err := injectAmbientMemoryFor(providerID, baseDir, text, &out); err != nil {
						return session.PrepareOutput{}, fmt.Errorf("inject knowledge: %w", err)
					}
				}
			}
		}

		// Carry-over context on account switch: when the operator opted
		// into "carry context", SwitchClaudeAccount put a recap of the
		// prior conversation on the context. Inject it through the same
		// --append-system-prompt channel as ambient memory so the fresh
		// session under the new account starts with continuity. One-shot
		// — only present on the switch respawn.
		if err := injectCarryoverFor(prepareCtx, providerID, baseDir, &out); err != nil {
			return session.PrepareOutput{}, fmt.Errorf("inject carryover context: %w", err)
		}

		// Integration spawn profile: inject the creating integration's boot
		// system prompt through the same per-provider surface as ambient
		// memory. No-op when the integration didn't set one.
		if err := injectIntegrationPromptFor(prepareCtx, providerID, baseDir, &out); err != nil {
			return session.PrepareOutput{}, err
		}

		// Background mirror: pull whatever Claude has already written
		// to <cwd>/.claude/projects/.../memory/*.md into the shared
		// store, so the agent's MCP search sees them. Fire-and-forget
		// so spawn isn't blocked on filesystem walks.
		if sp.memoryMirror != nil {
			cwd := session.Cwd(prepareCtx)
			if cwd != "" {
				go func() {
					if _, err := sp.memoryMirror(context.Background(), cwd); err != nil {
						sp.log.Debug("memory mirror sync", "cwd", cwd, "err", err)
					}
				}()
			}
		}

		// Antigravity's MCP surface is keyed off the session's effective
		// HOME — the account dir when the spawn is account-bound (set
		// above), else the gateway user's real HOME.
		sessionHome := out.Env["HOME"]
		if sessionHome == "" {
			sessionHome, _ = os.UserHomeDir()
		}

		if mcpEnabled {
			// Memory MCP needs the live cwd as scope_key. We attach
			// it here (rather than statically when servers is built)
			// because Prepare runs per spawn and the cwd is only on
			// the context at this point. Antigravity is the exception:
			// its entry lands in the HOME-global mcp_config.json shared
			// by every session under that HOME, so baking one session's
			// cwd in would leak it into the others — instead the entry
			// carries a static opt-in flag telling the mcp-memory
			// subprocess to derive the key from its own cwd (agy spawns
			// MCP servers from the session workspace; verified). The
			// flag is identical for every session, so the shared file
			// never churns.
			cwd := session.Cwd(prepareCtx)
			for i := range servers {
				if servers[i].Env == nil && (servers[i].Name == "opendray-memory" || servers[i].Name == "opendray-dbtool") {
					servers[i].Env = map[string]string{}
				}
				switch servers[i].Name {
				case "opendray-memory":
					if providerID == "antigravity" {
						servers[i].Env["OPENDRAY_MEMORY_SCOPE_FROM_CWD"] = "1"
					} else {
						servers[i].Env["OPENDRAY_MEMORY_SCOPE_KEY"] = cwd
					}
				case "opendray-dbtool":
					if providerID == "antigravity" {
						// HOME-global MCP config: the honest-path key
						// (no db:signed), no per-session signature — a
						// shared config can't carry a per-cwd one.
						servers[i].Env["OPENDRAY_API_KEY"] = sp.dbtool.AgyAPIKey
						servers[i].Env["OPENDRAY_DBTOOL_CWD_FROM_CWD"] = "1"
					} else {
						// Per-session MCP config: the signed key plus a
						// per-cwd HMAC proof the agent can't forge.
						servers[i].Env["OPENDRAY_DBTOOL_CWD"] = cwd
						if len(sp.dbtool.SignSecret) > 0 {
							servers[i].Env["OPENDRAY_DBTOOL_CWD_SIG"] =
								signDbtoolCwd(sp.dbtool.SignSecret, cwd)
						}
					}
				}
			}
			// Resolve ${KEY} placeholders against the secrets file at
			// spawn time so the rendered claude-mcp.json / codex
			// config.toml gets real values. The on-disk vault entries
			// keep the placeholder so they stay git-safe.
			resolved, missing := resolveMCPSecrets(servers, sp.secretsFile, sp.log)
			if len(missing) > 0 {
				sp.log.Warn("MCP servers reference unset secrets",
					"provider", providerID, "missing", missing)
			}
			extraArgs, mcpEnv, err := renderMCP(providerID, baseDir, cwd, sessionHome, resolved)
			if err != nil {
				return session.PrepareOutput{}, err
			}
			// Append, don't overwrite — earlier branches (skills) may
			// have already populated out.Args with --append-system-prompt
			// and similar.
			out.Args = append(out.Args, extraArgs...)
			for k, v := range mcpEnv {
				out.Env[k] = v
			}
		}

		// Antigravity (agy) renders MCP into <home>/.gemini/config/
		// mcp_config.json. Converge it to empty when MCP is off this
		// spawn so removed servers (and their credentials) don't linger,
		// and always purge the legacy <cwd>/.gemini/settings.json
		// entries a previous release wrote (agy never read that file).
		if providerID == "antigravity" {
			if !mcpEnabled {
				if err := syncAgyGlobalMCP(sessionHome, nil); err != nil {
					sp.log.Warn("agy global MCP cleanup failed",
						"provider", providerID, "home", sessionHome, "err", err)
				}
			}
			if cwd := session.Cwd(prepareCtx); cwd != "" {
				if err := syncGeminiWorkspaceMCP(cwd, nil); err != nil {
					sp.log.Warn("legacy workspace MCP cleanup failed",
						"provider", providerID, "cwd", cwd, "err", err)
				}
			}
		}

		// Grok renders MCP into <cwd>/.grok/config.toml (project-scoped
		// config). Mirror the antigravity cleanup so disabling MCP this
		// spawn prunes stale managed entries (and their credentials).
		if providerID == "grok" {
			cwd := session.Cwd(prepareCtx)
			if cwd != "" && !mcpEnabled {
				if err := syncGrokWorkspaceMCP(cwd, nil); err != nil {
					sp.log.Warn("workspace MCP cleanup failed",
						"provider", providerID, "cwd", cwd, "err", err)
				}
			}
		}

		if providerID == "codex" && out.Env["CODEX_HOME"] != "" {
			if err := ensureCodexScratchTrust(out.Env["CODEX_HOME"], session.Cwd(prepareCtx)); err != nil {
				return session.PrepareOutput{}, fmt.Errorf("prepare codex config: %w", err)
			}
		}

		if len(out.Env) == 0 {
			out.Env = nil
		}
		return out, nil
	}
	return info, nil
}

// integrationMCPAllowed reports whether an integration's declared MCP
// servers may be rendered for this provider. Everywhere except
// antigravity the answer is yes — those providers get per-session MCP
// config files. Antigravity's only surface is the HOME-global
// mcp_config.json: without an account binding the spawn shares the
// gateway user's real HOME, and writing the integration's servers
// (with their resolved credentials) there would expose them to the
// operator's own agy sessions and to every other integration under
// that HOME.
func integrationMCPAllowed(providerID string, accountBound bool) bool {
	return providerID != "antigravity" || accountBound
}

// providerSupportsSkills enumerates which CLI providers we have a
// safe skill-injection path for by default.
//
//	claude — `--append-system-prompt` flag, zero filesystem touch
//	antigravity — writes AGENTS.md inside the per-session scratch dir
//	         and adds it via --add-dir (agy's workspace-context
//	         convention; verified it reads AGENTS.md from added dirs)
//	         without touching the user's real ~/.gemini
//
// codex is intentionally NOT in the default list: it has no system-
// prompt CLI flag, so the only path is `<CODEX_HOME>/instructions.md`,
// which means overriding CODEX_HOME to a scratch dir — that wipes the
// user's ChatGPT-OAuth auth state stored under ~/.codex/. The codex
// arm of injectSkillsFor still exists for users who explicitly opt in
// (provider.config.skills_enabled=true), accepting the auth tradeoff.
//
// Adding a new provider here requires a matching arm in injectSkillsFor.
func providerSupportsSkills(id string) bool {
	switch id {
	case "claude", "antigravity", "opencode":
		return true
	default:
		return false
	}
}

// injectAgyContext appends content to <baseDir>/AGENTS.md and ensures
// `--add-dir <baseDir>` is set (idempotent). Antigravity (agy) reads
// AGENTS.md from directories added via --add-dir — verified live — so we
// inject workspace context without touching the user's real ~/.gemini
// state dir or the cwd AGENTS.md.
func injectAgyContext(baseDir, content string, out *session.PrepareOutput) error {
	path := filepath.Join(baseDir, "AGENTS.md")
	if err := appendToFile(path, content); err != nil {
		return err
	}
	if !hasArgPair(out.Args, "--add-dir", baseDir) {
		out.Args = append(out.Args, "--add-dir", baseDir)
	}
	return nil
}

// ensureAgySharedCache symlinks a per-account antigravity HOME's
// Playwright browser cache to the gateway user's real one, so each
// account doesn't re-download the (hundreds-of-MB) browser payload under
// its own HOME. No-op when home IS the real HOME, when the shared source
// doesn't exist yet, or when the account already has its own cache entry
// (real dir or prior symlink) in place. Best-effort — the caller logs
// and continues on error.
func ensureAgySharedCache(home string) error {
	realHome, err := os.UserHomeDir()
	if err != nil || realHome == "" || home == realHome {
		return nil
	}
	const rel = ".cache/ms-playwright-go"
	src := filepath.Join(realHome, rel)
	if st, err := os.Stat(src); err != nil || !st.IsDir() {
		return nil // nothing to share yet
	}
	dst := filepath.Join(home, rel)
	if _, err := os.Lstat(dst); err == nil {
		return nil // already present
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return fmt.Errorf("mkdir cache parent: %w", err)
	}
	if err := os.Symlink(src, dst); err != nil {
		return fmt.Errorf("symlink cache: %w", err)
	}
	return nil
}

// injectSkillsFor dispatches to the per-CLI Tier 1 index injection
// path. Each provider has its own convention for picking up extra
// system instructions:
//
//	claude: --append-system-prompt <text>          (CLI flag)
//	codex:  <CODEX_HOME>/instructions.md           (file in config dir)
//	antigravity: <baseDir>/AGENTS.md + --add-dir=<baseDir>
//
// The skills index itself is the same markdown across providers — only
// the delivery mechanism differs.
func injectSkillsFor(providerID, baseDir string, loaded []skills.Skill, out *session.PrepareOutput) error {
	index := skills.IndexPrompt(loaded)
	switch providerID {
	case "claude":
		out.Args = append(out.Args, "--append-system-prompt", index)
		return nil
	case "codex":
		// CODEX_HOME may already be set by the MCP renderer (which
		// writes config.toml into the same dir). If not, create a
		// fresh per-session home so we have somewhere to drop
		// instructions.md.
		home := out.Env["CODEX_HOME"]
		if home == "" {
			home = filepath.Join(baseDir, "codex-home")
			if err := os.MkdirAll(home, 0o700); err != nil {
				return fmt.Errorf("mkdir codex home: %w", err)
			}
			out.Env["CODEX_HOME"] = home
		}
		// Symlink the user's real codex home (auth.json, history,
		// cache, …) into our scratch so codex finds its OAuth state.
		// We skip config.toml (MCP renderer may want to write its own)
		// and instructions.md (we write our own below). Codex doesn't
		// atomic-rewrite auth.json the way Claude rewrites .claude.json,
		// so symlinks survive token refreshes.
		userHome := os.Getenv("CODEX_HOME")
		if userHome == "" {
			if h, err := os.UserHomeDir(); err == nil {
				userHome = filepath.Join(h, ".codex")
			}
		}
		if userHome != "" && userHome != home {
			if err := mirrorCodexHome(userHome, home); err != nil {
				return fmt.Errorf("mirror codex home: %w", err)
			}
		}
		// Codex reads AGENTS.md as global memory from CODEX_HOME.
		// Write the index there. If we already symlinked the user's
		// AGENTS.md from the mirror step, drop it and prepend the
		// user's content (if non-empty) so we don't lose it.
		agentsPath := filepath.Join(home, "AGENTS.md")
		var userAgents []byte
		if info, err := os.Lstat(agentsPath); err == nil {
			// Symlink → read the resolved content so we can preserve it.
			if info.Mode()&os.ModeSymlink != 0 {
				if data, rerr := os.ReadFile(agentsPath); rerr == nil {
					userAgents = data
				}
				_ = os.Remove(agentsPath)
			}
		}
		body := []byte(index)
		if len(userAgents) > 0 {
			body = append(body, "\n\n---\n\n"...)
			body = append(body, userAgents...)
		}
		if err := os.WriteFile(agentsPath, body, 0o600); err != nil {
			return fmt.Errorf("write %s: %w", agentsPath, err)
		}
		return nil
	case "antigravity":
		// agy reads AGENTS.md from --add-dir'd workspace dirs.
		return injectAgyContext(baseDir, index, out)
	case "opencode":
		// OpenCode loads the instruction files listed in its config. Write
		// the skill index to a per-session AGENTS.md and reference it from
		// the generated OPENCODE_CONFIG (instructions array).
		if err := os.WriteFile(openCodeAgentsPath(baseDir), []byte(index), 0o600); err != nil {
			return fmt.Errorf("write %s: %w", openCodeAgentsPath(baseDir), err)
		}
		return ensureOpenCodeInstructions(baseDir, out)
	default:
		return fmt.Errorf("no skill injection path for provider %s", providerID)
	}
}

// loadVaultMCPs reads the enabled servers from the registry and
// converts them into the catalog.MCPServer shape used by renderMCP.
// Returns nil when the loader is disabled or the registry is empty
// — the caller treats nil and empty equivalently.
func loadVaultMCPs(loader *mcp.Loader, log *slog.Logger) []MCPServer {
	if loader == nil {
		return nil
	}
	enabled, err := loader.ListEnabled()
	if err != nil {
		log.Warn("load vault MCPs failed", "err", err)
		return nil
	}
	out := make([]MCPServer, 0, len(enabled))
	for _, s := range enabled {
		out = append(out, MCPServer{
			Name:      s.Name,
			Transport: s.Transport,
			Command:   s.Command,
			Args:      s.Args,
			Env:       s.Env,
			URL:       s.URL,
			Headers:   s.Headers,
		})
	}
	return out
}

// mergeMCPServers concatenates two lists with provider-config (the
// `extra` slice) winning on name collision. Order: vault entries
// first so the rendered config keeps a stable, ID-sorted layout for
// the registry portion, with provider-specific overrides appended
// after.
func mergeMCPServers(vault, extra []MCPServer) []MCPServer {
	if len(vault) == 0 {
		return extra
	}
	if len(extra) == 0 {
		return vault
	}
	overrides := map[string]bool{}
	for _, s := range extra {
		overrides[s.Name] = true
	}
	out := make([]MCPServer, 0, len(vault)+len(extra))
	for _, s := range vault {
		if overrides[s.Name] {
			continue // provider config will replace this entry
		}
		out = append(out, s)
	}
	out = append(out, extra...)
	return out
}

// resolveMCPSecrets substitutes ${KEY} placeholders in env/headers/url
// /args of every server. The secrets file is reloaded each call so
// users can edit it (via the Plugins page or a shell) without
// restarting the gateway. Missing placeholders pass through literally
// so the agent surfaces a clear "credential not set" error.
func resolveMCPSecrets(servers []MCPServer, path string, log *slog.Logger) ([]MCPServer, []string) {
	if len(servers) == 0 || path == "" {
		return servers, nil
	}
	secrets, err := mcp.LoadSecrets(path)
	if err != nil {
		log.Warn("load MCP secrets failed", "path", path, "err", err)
		return servers, nil
	}
	out := make([]MCPServer, len(servers))
	seen := map[string]bool{}
	var missing []string
	for i, s := range servers {
		// MCPServer (catalog) maps 1:1 to mcp.Server for substitution
		// purposes. Build a temporary mcp.Server, run Resolve, copy
		// the resolved fields back.
		resolved, miss := secrets.Resolve(mcp.Server{
			Env:     s.Env,
			Headers: s.Headers,
			URL:     s.URL,
			Args:    s.Args,
		})
		out[i] = MCPServer{
			Name:      s.Name,
			Transport: s.Transport,
			Command:   s.Command,
			Args:      resolved.Args,
			Env:       resolved.Env,
			URL:       resolved.URL,
			Headers:   resolved.Headers,
		}
		for _, k := range miss {
			if !seen[k] {
				seen[k] = true
				missing = append(missing, k)
			}
		}
	}
	return out, missing
}

// mirrorCodexHome copies the minimal subset of the user's Codex home
// that a session scratch dir needs to start authenticated inside a
// sandbox.
//
// We intentionally do NOT symlink the whole ~/.codex tree anymore.
// Under codex's workspace-write sandbox, symlinked sqlite/log/cache
// files still resolve to paths outside the writable scratch dir, so
// startup fails with "attempt to write a readonly database". Keeping
// mutable runtime state local to dest avoids that while still
// preserving auth and user rules/plugins.
func mirrorCodexHome(src, dest string) error {
	if err := os.MkdirAll(dest, 0o700); err != nil {
		return err
	}
	allow := map[string]bool{
		"auth.json":                true,
		".codex-global-state.json": true,
		"installation_id":          true,
		"version.json":             true,
		"plugins":                  true,
		"rules":                    true,
		"skills":                   true,
	}
	for name := range allow {
		srcPath := filepath.Join(src, name)
		if _, err := os.Stat(srcPath); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		dstPath := filepath.Join(dest, name)
		if err := copyPath(srcPath, dstPath); err != nil {
			return fmt.Errorf("copy %s: %w", name, err)
		}
	}
	return nil
}

func copyPath(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			if li, lerr := os.Lstat(src); lerr == nil && li.Mode()&os.ModeSymlink != 0 {
				// Dangling symlink in user's codex home — skip rather than
				// fail the whole mirror. Stale skill links are common when
				// a source repo is moved or deleted.
				return nil
			}
		}
		return err
	}
	if info.IsDir() {
		if err := os.MkdirAll(dst, info.Mode().Perm()); err != nil {
			return err
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if err := copyPath(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
				return err
			}
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func ensureCodexScratchTrust(home, cwd string) error {
	if home == "" || cwd == "" {
		return nil
	}
	if err := os.MkdirAll(home, 0o700); err != nil {
		return err
	}
	path := filepath.Join(home, "config.toml")
	bodyBytes, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		bodyBytes = []byte(codexBaseConfigForScratch())
	}
	body := string(bodyBytes)
	projectHeader := "[projects." + tomlString(cwd) + "]"
	if strings.Contains(body, projectHeader) {
		return nil
	}
	if strings.TrimSpace(body) != "" {
		body = strings.TrimRight(body, "\n") + "\n\n"
	}
	body += projectHeader + "\ntrust_level = \"trusted\"\n"
	return os.WriteFile(path, []byte(body), 0o600)
}

// providerConflicts returns the static CLI argument-group rules for
// a provider. Each entry maps a "trigger" flag (one that may appear in
// user spawn args) to the set of provider-config flags that conflict
// with it and must be stripped before exec.
//
// Currently only codex needs this: its clap ArgGroup makes
// --dangerously-bypass-approvals-and-sandbox mutually exclusive with
// --ask-for-approval (-a) and --sandbox (-s). Without this stripping
// step the spawn fails because the saved provider config keeps emitting
// the default approval/sandbox flags.
func providerConflicts(providerID string) map[string][]string {
	switch providerID {
	case "codex":
		return map[string][]string{
			"--dangerously-bypass-approvals-and-sandbox": {
				"--ask-for-approval", "-a",
				"--sandbox", "-s",
			},
		}
	default:
		return nil
	}
}

// defaultStr returns def when s is empty after trimming, otherwise s.
func defaultStr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// memoryGuidanceText is appended to the agent's system prompt
// whenever the memory MCP server is auto-attached. Rewritten in
// PR-M1 to match Claude's auto-memory discipline — give the model
// concrete save/skip criteria, categories, and dedup rules rather
// than a generic "memory exists, use it" hint.
//
// The wording is verbose by const-string standards (~80 lines) but
// the discipline it encodes is what makes the difference between
// "agent never uses memory" and "agent records every durable
// project fact it discovers." Anything looser produces empty
// memory stores in real-world use.
//
// Kept inside a const so per-spawn cost is just a string copy.
// Optimisation knob if it grows further: move to a runtime-
// loaded markdown file under ~/.opendray/prompts/.
const memoryGuidanceText = `## Persistent cross-agent memory (opendray-memory)

This session has access to an MCP server named ` + "`opendray-memory`" + ` that
persists durable facts to a shared store. **Every Claude / Codex /
Antigravity session in the same project reads and writes the same
store.** What you save here is what the next session sees, no
matter which CLI it runs under.

### Use this, not your built-in file memory

` + "`opendray-memory`" + ` is the **only** memory you should write. Do
**not** use your CLI's own built-in memory feature (Claude's
` + "`# Memory` / `MEMORY.md`" + ` auto-memory files, or any local
per-project memory file you'd normally create). Those files are
**CLI-local** — a memory you write to a file is invisible to the
next Codex or Antigravity session in this project, which defeats the
entire point of a shared brain. opendray already imports any
pre-existing local memory files into this store for you, and
injects the relevant project memory into your context at startup,
so the file layer is redundant. Always route durable facts through
` + "`memory_store`" + ` (and project state through the ` + "`project_*`" + ` /
` + "`session_log_append`" + ` tools below).

### At session start

Call ` + "`opendray-memory.memory_load_context()`" + ` once for the project
context relevant to the user's first ask. The store may already
hold user preferences, project facts, past decisions you'd
otherwise re-discover. Skipping this is the most common reason
agents make mistakes that prior sessions already corrected.

### When to store (proactively, without being asked)

Save when you encounter ANY of these. Set ` + "`metadata.type`" + ` to the
matching category so future sessions can filter:

- **user_preference** — the user states or implies a durable
  preference ("I prefer Go", "use pnpm", "no emoji in commits").
- **project_fact** — non-obvious project information you discover
  while working: DB schema details, deployment topology, key file
  locations, environment quirks, external service URLs. Save what
  a future session would otherwise have to re-discover.
- **feedback** — the user corrects your approach. Save the
  correction + the **Why:** so you don't repeat the mistake. Often
  load-bearing on edge cases.
- **reference** — pointers to external systems: where bugs live
  (Linear / GitHub Issues), which Grafana dashboard tracks which
  metric, where ops runbooks are.

### When NOT to store

- Anything derivable from the current code: file paths, function
  names, types, struct fields. ` + "`grep`" + ` finds these next time.
- Ephemeral state: what's in progress, the last command you ran,
  the file currently open. The next session will look fresh.
- Anything already documented in CLAUDE.md / AGENTS.md —
  operator-curated docs are the source of truth there.

### How to store

Call ` + "`opendray-memory.memory_store`" + ` with:

- ` + "`text`" + `: the memory body. Lead with the fact itself in one
  sentence. For non-obvious items add a **Why:** line (the reason
  this matters) and a **How to apply:** line (when/where this
  guidance kicks in). Brief — one short paragraph per memory.
- ` + "`metadata.type`" + `: ` + "`user_preference`" + ` / ` + "`project_fact`" + ` / ` + "`feedback`" + ` /
  ` + "`reference`" + `.
- Scope defaults to ` + "`project`" + ` (visible only in this cwd). Pass
  ` + "`metadata.scope: global`" + ` for stable user-level facts you want
  visible everywhere. Use ` + "`global`" + ` sparingly — it's expensive
  context for every future spawn.

### Dedup discipline

Before storing, call ` + "`memory_search`" + ` with a query that would match
the fact you're about to write. If a near-match exists:

- If the existing entry is still correct → don't duplicate.
- If your version is more accurate or richer → update the
  existing entry's text rather than creating a sibling.

### Stale memory

If you find a memory that contradicts the current state of the
code or the user's latest direction, **surface it to the user**
and propose an update or delete. Don't silently work around it —
silent contradictions are how memory rot starts.

### Project state — keep it current (DO NOT skip)

memory_store is for DISCRETE FACTS. Project STATE — what we're
building, where we are, what just happened — belongs in three
other tools you also have on this MCP server:

- ` + "`project_goal_set`" + ` — the project's long-term intent.
  Update only when the goal genuinely changes; rare.
- ` + "`project_plan_set`" + ` — the current roadmap / WIP arc.
  **Update whenever the plan moves forward**: when you finish a
  phase, when a new phase appears, when scope shifts. Each call
  files a proposal that the operator approves, so it's safe to
  call often — the operator filters noise. A stale plan is the
  most common reason future sessions repeat work.
- ` + "`session_log_append`" + ` — append a journal entry. **Call this
  every time you complete a meaningful unit of work** in the
  current session: shipped a feature, fixed a bug, made a
  decision, hit a blocker, learned something the next session
  needs. Title = one-line summary, content = what you did + why.
  These accumulate into the project journal that every future
  session sees at spawn time.
- ` + "`decision_record`" + ` — ADR-style entry for choices that future
  sessions should not re-litigate ("we picked pgvector over
  Pinecone because…"). Use for genuine architectural locks-in,
  not every micro-decision.

DO NOT confuse the layers:

| layer        | what                                | when |
|--------------|-------------------------------------|------|
| memory_store | one-sentence fact, top-K retrieved  | rarely; only durable facts |
| session_log_append | what we just did               | often; every meaningful step |
| project_plan_set   | where we are vs where we're going | when the plan shifts |
| project_goal_set   | the project's North Star       | rarely; only when goal changes |

The single most common failure mode is agents writing "currently
working on M5" as a memory_store entry. That's WRONG — it's
ephemeral state that belongs in a session_log_append OR a
project_plan_set update. memory_store is for things future
sessions will still want to retrieve months from now.
`

// injectSessionIDFor pre-assigns the agent-side session UUID for
// providers that support `--session-id`. Returns true when an ID was
// injected, false when the provider doesn't support pre-assignment.
//
// Mutates out.Args (adds the flag pair) and out.ClaudeSessionID
// (so the session manager can persist the value onto the row).
// Codex has no equivalent flag — it generates its own UUID inside
// rollout-<ts>-<uuid>.jsonl, which the M18 reader still finds via
// the cwd-based fallback path.
//
// Idempotent against operator-supplied --session-id: if the user
// already passed a UUID via Session.Args we leave it untouched.
// (Args ordering puts provider-baseline → out.Args → sess.Args, so
// a user-supplied flag wins anyway, but we skip injection too to
// avoid emitting a duplicate flag.)
//
// Resume path: when ctx carries a resume UUID (set by the session
// manager on reactivation), claude continues that conversation via
// `--resume <id>` rather than starting a fresh `--session-id`. We
// preserve the original UUID on out.ClaudeSessionID so the row keeps
// pointing at the same transcript. Providers without a verified resume
// flag fall back to a fresh session-id (a new turn, history intact
// on disk) until that's confirmed.
func injectSessionIDFor(ctx context.Context, providerID string, out *session.PrepareOutput) bool {
	resumeID := session.ResumeClaudeSessionIDFromContext(ctx)
	switch providerID {
	case "claude":
		if resumeID != "" {
			out.Args = append(out.Args, "--resume", resumeID)
			out.ClaudeSessionID = resumeID
			return true
		}
		id := uuid.NewString()
		out.Args = append(out.Args, "--session-id", id)
		out.ClaudeSessionID = id
		return true
	case "antigravity":
		// agy auto-creates a conversation per cwd on a fresh spawn, so we
		// don't pre-assign an id. On restart / account switch the manager
		// sets the cwd's conversation id (the switch also copies the db
		// into the new account's HOME first), and we resume it.
		if convID := session.AntigravityResumeConversationFromContext(ctx); convID != "" {
			out.Args = append(out.Args, "--conversation", convID)
			return true
		}
		return false
	}
	return false
}

// injectMemoryGuidanceFor adds memoryGuidanceText to the provider's
// system-prompt surface — same per-CLI dispatch shape as
// injectSkillsFor, so both layers add into the same channel without
// stepping on each other.
//
//	claude → another --append-system-prompt arg (Claude concatenates
//	         every occurrence into the system prompt).
//	codex  → append to <CODEX_HOME>/AGENTS.md (created earlier by
//	         injectSkillsFor when skills are on; otherwise we lazily
//	         set up CODEX_HOME here).
//	antigravity → append to the --add-dir'd <baseDir>/AGENTS.md
//	         (idempotent — won't duplicate if injectSkillsFor already
//	         added it).
func injectMemoryGuidanceFor(providerID, baseDir string, out *session.PrepareOutput) error {
	switch providerID {
	case "claude":
		out.Args = append(out.Args, "--append-system-prompt", memoryGuidanceText)
		return nil
	case "codex":
		home := out.Env["CODEX_HOME"]
		if home == "" {
			home = filepath.Join(baseDir, "codex-home")
			if err := os.MkdirAll(home, 0o700); err != nil {
				return fmt.Errorf("mkdir codex home: %w", err)
			}
			out.Env["CODEX_HOME"] = home
		}
		path := filepath.Join(home, "AGENTS.md")
		return appendToFile(path, "\n\n---\n\n"+memoryGuidanceText)
	case "antigravity":
		return injectAgyContext(baseDir, "\n\n---\n\n"+memoryGuidanceText, out)
	case "opencode":
		if err := appendToFile(openCodeAgentsPath(baseDir), "\n\n---\n\n"+memoryGuidanceText); err != nil {
			return err
		}
		return ensureOpenCodeInstructions(baseDir, out)
	}
	// Other providers: silently skip — they don't have an MCP
	// surface yet so the memory MCP wouldn't be attached anyway.
	return nil
}

// injectAmbientMemoryFor injects the rendered "Recent project
// memory" banner into the agent's system prompt. Same per-CLI
// dispatch as injectMemoryGuidanceFor.
// injectCarryoverFor injects the account-switch conversation recap (set
// by Manager.SwitchClaudeAccount via session.WithCarryoverContext) into
// the spawn's system-prompt surface. No-op when the context carries no
// recap (the common path — every spawn except an opted-in switch).
// Reuses injectAmbientMemoryFor's per-provider dispatch; in practice the
// recap is only ever set on Claude switches, so this lands as a
// --append-system-prompt arg.
func injectCarryoverFor(ctx context.Context, providerID, baseDir string, out *session.PrepareOutput) error {
	text := session.CarryoverContextFromContext(ctx)
	if text == "" {
		return nil
	}
	return injectAmbientMemoryFor(providerID, baseDir, text, out)
}

// injectIntegrationPromptFor injects the integration's boot system
// prompt (set via the integration's spawn profile) through the same
// per-provider surface as ambient memory. No-op when unset.
func injectIntegrationPromptFor(ctx context.Context, providerID, baseDir string, out *session.PrepareOutput) error {
	text := session.IntegrationSystemPromptFromContext(ctx)
	if text == "" {
		return nil
	}
	return injectAmbientMemoryFor(providerID, baseDir, text, out)
}

// bypassArgsFor returns the per-provider flag that makes a CLI run
// unattended (no human to approve tool calls), used when an
// integration sets bypass_permissions. Empty for providers without one.
func bypassArgsFor(providerID string) []string {
	switch providerID {
	case "claude", "antigravity":
		return []string{"--dangerously-skip-permissions"}
	case "codex":
		return []string{"--dangerously-bypass-approvals-and-sandbox"}
	}
	return nil
}

// stripFlagsWithValues removes the named flags (and a following value
// token when the flag takes one) from args. Both "--flag value" and
// "--flag=value" forms are handled. Used to resolve codex's clap
// ArgGroup conflict between the saved approval/sandbox config and the
// integration bypass flag: both originate provider-side, so they escape
// the user-triggered dropConflictingFlags pass in the session manager.
func stripFlagsWithValues(args []string, flags ...string) []string {
	if len(args) == 0 || len(flags) == 0 {
		return args
	}
	drop := make(map[string]struct{}, len(flags))
	for _, f := range flags {
		drop[f] = struct{}{}
	}
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		tok := args[i]
		name := tok
		if eq := strings.IndexByte(tok, '='); eq >= 0 {
			name = tok[:eq]
		}
		if _, hit := drop[name]; !hit {
			out = append(out, tok)
			continue
		}
		// --flag=value is a single token — dropping it is enough.
		if strings.Contains(tok, "=") {
			continue
		}
		// --flag value: also skip the value token when present and not
		// itself another flag.
		if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
			i++
		}
	}
	return out
}

func injectAmbientMemoryFor(providerID, baseDir, text string, out *session.PrepareOutput) error {
	if text == "" {
		return nil
	}
	switch providerID {
	case "claude":
		out.Args = append(out.Args, "--append-system-prompt", text)
		return nil
	case "codex":
		home := out.Env["CODEX_HOME"]
		if home == "" {
			home = filepath.Join(baseDir, "codex-home")
			if err := os.MkdirAll(home, 0o700); err != nil {
				return fmt.Errorf("mkdir codex home: %w", err)
			}
			out.Env["CODEX_HOME"] = home
		}
		path := filepath.Join(home, "AGENTS.md")
		return appendToFile(path, "\n\n---\n\n"+text)
	case "antigravity":
		return injectAgyContext(baseDir, "\n\n---\n\n"+text, out)
	case "opencode":
		if err := appendToFile(openCodeAgentsPath(baseDir), "\n\n---\n\n"+text); err != nil {
			return err
		}
		return ensureOpenCodeInstructions(baseDir, out)
	}
	return nil
}

// appendToFile appends content to path, creating it (mode 0600)
// when missing. Used by both injectSkillsFor extensions and
// injectMemoryGuidanceFor so multiple system-prompt sources can
// coexist in the same file.
func appendToFile(path, content string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	if _, err := f.WriteString(content); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// hasArgPair reports whether args contains the consecutive pair
// flag+value (e.g. "--include-directories" then a path). Lets the
// memory-guidance injector skip adding a duplicate flag when
// injectSkillsFor already added one.
func hasArgPair(args []string, flag, value string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == flag && args[i+1] == value {
			return true
		}
	}
	return false
}

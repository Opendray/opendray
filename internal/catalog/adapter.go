package catalog

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
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
type ProjectDocInjector interface {
	RenderForSpawn(ctx context.Context, cwd string, recentLogs int) (string, error)
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
	if p.Manifest.ID == "codex" {
		if approval, ok := p.Config["approval"].(string); ok && approval != "" {
			args = append(args, "-c", "approval_policy="+tomlString(approval))
		}
	}

	info := session.ProviderInfo{
		ID:         p.Manifest.ID,
		Executable: exe,
		Args:       args,
	}

	// Account selection: only meaningful for the Claude provider —
	// other CLIs (codex / gemini) don't yet have a multi-account
	// abstraction in v2 because their auth model differs.
	claudeAccountID := session.AccountID(ctx)
	wantClaudeAccount := id == "claude" && claudeAccountID != "" && sp.accounts != nil

	// Merge vault MCP registry (enabled-only) into the provider's
	// inline mcp_servers list. Vault entries are loaded eagerly here
	// (cheap — small JSON reads) so the spawn-decision branch below
	// can short-circuit when there are zero servers in either tier.
	//
	// Precedence on `name` collision: provider config wins. Lets users
	// override a vault entry per-provider without editing the registry.
	servers := mergeMCPServers(loadVaultMCPs(sp.mcps, sp.log), parseMCPServers(p.Config))
	// Auto-attach opendray's memory MCP server when enabled at app
	// startup. This is what makes "the agent remembers things across
	// sessions" actually work without per-CLI manual setup.
	if sp.memory.Enabled && p.Manifest.Capabilities.SupportsMcp {
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
				// session.Cwd from context.
			},
		})
	}
	mcpEnabled := p.Manifest.Capabilities.SupportsMcp && len(servers) > 0

	// Skill injection: enabled by default for providers in the safe
	// list (claude, gemini). Codex is opt-in via skills_enabled=true
	// because its only injection path is CODEX_HOME, which clobbers
	// ChatGPT-OAuth auth.
	skillsEnabled := sp.skills != nil && providerSupportsSkills(id)
	if v, ok := p.Config["skills_disabled"].(bool); ok && v {
		skillsEnabled = false
	}
	if v, ok := p.Config["skills_enabled"].(bool); ok && v && sp.skills != nil {
		skillsEnabled = true
	}

	if !wantClaudeAccount && !mcpEnabled && !skillsEnabled && len(configEnv) == 0 {
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

		// M21 — Pre-assign the agent-side session UUID so the M18
		// transcript reader can locate the *.jsonl file directly,
		// instead of falling back to "latest mtime in dir" which
		// picks up unrelated active conversations. Claude Code and
		// Gemini both accept `--session-id <uuid>`; Codex does not,
		// so it stays on the cwd-based reader path. injectSessionIDFor
		// mutates out.Args + out.ClaudeSessionID directly, and the
		// session manager picks up the UUID for persistence.
		injectSessionIDFor(providerID, &out)

		if wantClaudeAccount {
			acct, token, err := sp.accounts.ReadToken(prepareCtx, claudeAccountID)
			if err != nil {
				return session.PrepareOutput{}, fmt.Errorf("claude account %s: %w", claudeAccountID, err)
			}
			out.Env["CLAUDE_CODE_OAUTH_TOKEN"] = token
			if acct.ConfigDir != "" {
				// Point Claude Code at the account's persistent config
				// dir directly. Earlier attempts to materialise skills
				// here via symlinks broke first-run / auth state because
				// Claude Code rewrites .claude.json atomically (replacing
				// any symlink with a fresh small file). We now keep the
				// account dir untouched and inject skills purely via the
				// --append-system-prompt CLI flag below.
				out.Env["CLAUDE_CONFIG_DIR"] = acct.ConfigDir
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

		// Inject memory-tool guidance into the agent's system prompt
		// when the memory MCP is being attached. Without this nudge,
		// Claude (and to a lesser extent Codex/Gemini) tends to use
		// its built-in markdown memory feature instead of our shared
		// MCP store, defeating the cross-CLI value prop. Done here
		// (after skills, before MCP rendering) so the message ordering
		// stays predictable.
		if sp.memory.Enabled && p.Manifest.Capabilities.SupportsMcp {
			if err := injectMemoryGuidanceFor(providerID, baseDir, &out); err != nil {
				return session.PrepareOutput{}, fmt.Errorf("inject memory guidance: %w", err)
			}
		}

		// Ambient memory: pull a markdown banner of recent project
		// memories from the injector and prepend it to the system
		// prompt. Same per-CLI dispatch as memory guidance — claude
		// gets another --append-system-prompt arg, codex appends to
		// AGENTS.md, gemini to GEMINI.md. Empty rendered text means
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
				text, err := sp.projectDocInjector.RenderForSpawn(prepareCtx, cwd, 5)
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

		if mcpEnabled {
			// Memory MCP needs the live cwd as scope_key. We attach
			// it here (rather than statically when servers is built)
			// because Prepare runs per spawn and the cwd is only on
			// the context at this point.
			cwd := session.Cwd(prepareCtx)
			for i := range servers {
				if servers[i].Name == "opendray-memory" {
					if servers[i].Env == nil {
						servers[i].Env = map[string]string{}
					}
					servers[i].Env["OPENDRAY_MEMORY_SCOPE_KEY"] = cwd
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
			extraArgs, mcpEnv, err := renderMCP(providerID, baseDir, resolved)
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

			// Gemini mirroring: if renderMCP set GEMINI_CONFIG_DIR,
			// symlink the user's real ~/.gemini into the scratch dir
			// so they stay logged in.
			if providerID == "gemini" && out.Env["GEMINI_CONFIG_DIR"] != "" {
				home := out.Env["GEMINI_CONFIG_DIR"]
				userHome := os.Getenv("GEMINI_CONFIG_DIR")
				if userHome == "" {
					if h, err := os.UserHomeDir(); err == nil {
						userHome = filepath.Join(h, ".gemini")
					}
				}
				if userHome != "" && userHome != home {
					if err := mirrorGeminiHome(userHome, home); err != nil {
						return session.PrepareOutput{}, fmt.Errorf("mirror gemini home: %w", err)
					}
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

// providerSupportsSkills enumerates which CLI providers we have a
// safe skill-injection path for by default.
//
//	claude — `--append-system-prompt` flag, zero filesystem touch
//	gemini — writes a GEMINI.md inside the per-session scratch dir
//	         and adds it to the workspace via --include-directories;
//	         does NOT override ~/.gemini, so auth is preserved
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
	case "claude", "gemini":
		return true
	default:
		return false
	}
}

// injectSkillsFor dispatches to the per-CLI Tier 1 index injection
// path. Each provider has its own convention for picking up extra
// system instructions:
//
//	claude: --append-system-prompt <text>          (CLI flag)
//	codex:  <CODEX_HOME>/instructions.md           (file in config dir)
//	gemini: <baseDir>/GEMINI.md + --include-directories=<baseDir>
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
	case "gemini":
		// Gemini reads GEMINI.md as memory from cwd, parent dirs, and
		// ~/.gemini/. We can't (and shouldn't) touch any of those, so
		// drop the index in baseDir and ask gemini to widen the
		// workspace via --include-directories.
		path := filepath.Join(baseDir, "GEMINI.md")
		if err := os.WriteFile(path, []byte(index), 0o600); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
		out.Args = append(out.Args, "--include-directories", baseDir)
		return nil
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

// mirrorCodexHome symlinks every entry of src into dest, skipping
// files we layer ourselves (config.toml, instructions.md). Lets the
// codex CLI see the user's auth.json + history.jsonl + cache while
// our skill index sits alongside as a real file.
func mirrorCodexHome(src, dest string) error {
	if err := os.MkdirAll(dest, 0o700); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	skip := map[string]bool{
		"config.toml":     true,
		"instructions.md": true,
	}
	for _, e := range entries {
		if skip[e.Name()] {
			continue
		}
		srcPath := filepath.Join(src, e.Name())
		dstPath := filepath.Join(dest, e.Name())
		if err := os.Symlink(srcPath, dstPath); err != nil {
			if os.IsExist(err) {
				continue
			}
			return fmt.Errorf("symlink %s: %w", e.Name(), err)
		}
	}
	return nil
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
Gemini session in the same project reads and writes the same
store.** What you save here is what the next session sees, no
matter which CLI it runs under.

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
- Anything already documented in CLAUDE.md / AGENTS.md /
  GEMINI.md — operator-curated docs are the source of truth there.

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
func injectSessionIDFor(providerID string, out *session.PrepareOutput) bool {
	switch providerID {
	case "claude", "gemini":
		id := uuid.NewString()
		out.Args = append(out.Args, "--session-id", id)
		out.ClaudeSessionID = id
		return true
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
//	gemini → append to <baseDir>/GEMINI.md and ensure
//	         --include-directories <baseDir> is set (idempotent — won't
//	         duplicate if injectSkillsFor already added it).
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
	case "gemini":
		path := filepath.Join(baseDir, "GEMINI.md")
		if err := appendToFile(path, "\n\n---\n\n"+memoryGuidanceText); err != nil {
			return err
		}
		if !hasArgPair(out.Args, "--include-directories", baseDir) {
			out.Args = append(out.Args, "--include-directories", baseDir)
		}
		return nil
	}
	// Other providers: silently skip — they don't have an MCP
	// surface yet so the memory MCP wouldn't be attached anyway.
	return nil
}

// injectAmbientMemoryFor injects the rendered "Recent project
// memory" banner into the agent's system prompt. Same per-CLI
// dispatch as injectMemoryGuidanceFor.
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
	case "gemini":
		path := filepath.Join(baseDir, "GEMINI.md")
		if err := appendToFile(path, "\n\n---\n\n"+text); err != nil {
			return err
		}
		if !hasArgPair(out.Args, "--include-directories", baseDir) {
			out.Args = append(out.Args, "--include-directories", baseDir)
		}
		return nil
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

func mirrorGeminiHome(src, dest string) error {
	if err := os.MkdirAll(dest, 0o700); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	skip := map[string]bool{
		"settings.json": true,
	}
	for _, e := range entries {
		if skip[e.Name()] {
			continue
		}
		srcPath := filepath.Join(src, e.Name())
		dstPath := filepath.Join(dest, e.Name())
		if err := os.Symlink(srcPath, dstPath); err != nil {
			if os.IsExist(err) {
				continue
			}
			return fmt.Errorf("symlink %s: %w", e.Name(), err)
		}
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

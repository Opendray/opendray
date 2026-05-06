package catalog

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

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
}

// AmbientInjector is the contract internal/memory/injector
// satisfies — kept here so catalog stays import-decoupled from the
// memory package.
type AmbientInjector interface {
	Render(ctx context.Context, sessionID, cwd string) (string, error)
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

// defaultStr returns def when s is empty after trimming, otherwise s.
func defaultStr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// memoryGuidanceText is appended to the agent's system prompt
// whenever the memory MCP server is auto-attached. The wording
// nudges the model to prefer the shared cross-CLI store over its
// own built-in markdown memory feature.
//
// Kept short: every byte costs context window. Anything longer
// goes in the MCP tool descriptions themselves.
const memoryGuidanceText = `## Persistent memory across CLIs (opendray-memory)

This session has access to an MCP server named ` + "`opendray-memory`" + ` that
persists facts to a shared store available to every Claude / Codex /
Gemini session running in the same project directory.

PREFER opendray-memory's tools over writing to local memory files:

- BEFORE answering anything that might benefit from past context,
  call ` + "`opendray-memory.memory_search(query)`" + `. Memories survive across
  sessions and across CLIs.
- WHEN the user states a durable fact (preference, identifier,
  decision, ongoing task), call ` + "`opendray-memory.memory_store(text)`" + `
  instead of writing local memory files. The shared store is the
  single source of truth.
- Memories are scoped to the current project (cwd) by default. Any
  future Claude / Codex / Gemini session in the same directory can
  read them through the same MCP server.
`

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

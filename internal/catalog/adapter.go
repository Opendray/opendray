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

	if !wantClaudeAccount && !mcpEnabled && !skillsEnabled {
		return info, nil
	}

	providerID := p.Manifest.ID
	info.Prepare = func(prepareCtx context.Context, _, baseDir string) (session.PrepareOutput, error) {
		out := session.PrepareOutput{Env: map[string]string{}}

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

		if mcpEnabled {
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
//   claude — `--append-system-prompt` flag, zero filesystem touch
//   gemini — writes a GEMINI.md inside the per-session scratch dir
//            and adds it to the workspace via --include-directories;
//            does NOT override ~/.gemini, so auth is preserved
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
//   claude: --append-system-prompt <text>          (CLI flag)
//   codex:  <CODEX_HOME>/instructions.md           (file in config dir)
//   gemini: <baseDir>/GEMINI.md + --include-directories=<baseDir>
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

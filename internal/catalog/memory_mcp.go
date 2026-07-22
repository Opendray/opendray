package catalog

import (
	"fmt"
	"os"
	"path/filepath"
)

// AttachMemoryMCP renders the opendray-memory MCP server for one spawn of
// providerID and returns the extra CLI args + env the CLI needs to pick it
// up. It reuses the exact per-provider renderMCP machinery the interactive
// session adapter uses, so every caller — interactive sessions, headless
// Round Table members, any future out-of-band agent — injects memory the
// same way. This is the single dynamic injection point: nothing is baked
// into a provider at startup; the config is rendered per call.
//
// baseDir is a per-call scratch dir the renderer may write config files into
// (claude-mcp.json, codex-home/, opencode config). runCwd is the working
// directory the CLI itself will run in — grok reads project-scoped MCP from
// <runCwd>/.grok, and antigravity derives the memory scope from the MCP
// subprocess cwd, so for agy runCwd MUST be the project dir. scopeKey is the
// memory scope (the project whose facts/journal to expose); it is baked into
// the server env for every provider except antigravity (which derives it
// from runCwd). home is the effective HOME — antigravity's mcp_config.json
// surface; pass the operator's real HOME when the spawn isn't account-bound.
//
// readOnly attaches the server in read-only mode (OPENDRAY_MEMORY_READONLY=1):
// only search/read tools, every write refused server-side. Callers that run
// the CLI with tool permissions fully open (Round Table members) MUST pass
// true — the server, not the CLI, is the boundary there.
//
// Returns (nil, nil, nil) when memory auto-attach is disabled or unconfigured,
// so callers can wire it unconditionally and degrade to a tool-less spawn.
func AttachMemoryMCP(mem MemoryAutoAttach, providerID, baseDir, runCwd, scopeKey, home string, readOnly bool) ([]string, map[string]string, error) {
	if !mem.Enabled || mem.BinaryPath == "" {
		return nil, nil, nil
	}

	env := map[string]string{
		"OPENDRAY_BASE_URL":     mem.BaseURL,
		"OPENDRAY_API_KEY":      mem.APIKey,
		"OPENDRAY_MEMORY_SCOPE": defaultStr(mem.Scope, "project"),
	}
	if readOnly {
		env["OPENDRAY_MEMORY_READONLY"] = "1"
	}
	// Scope key: antigravity's entry lands in a HOME-global mcp_config.json
	// shared by every session under that HOME, so baking a per-call key in
	// would leak it into (and clobber) concurrent sessions. Instead it opts
	// into deriving the key from the MCP subprocess cwd (agy spawns MCP from
	// its own workspace, which the caller sets to runCwd). Every other
	// provider gets the key baked into its per-call entry.
	//
	// CAVEAT (antigravity only): because that file is shared, the read-only
	// flag also lives in the shared entry. A concurrent NON-read-only agy
	// spawn under the same HOME (e.g. an interactive session) converges the
	// same entry and can momentarily flip it. Isolated providers (claude /
	// codex / grok / opencode) each write a per-call config, so their
	// read-only guarantee is airtight; only the default-account agy seat
	// shares. Account-bound agy already has an isolated HOME and is unaffected.
	if providerID == "antigravity" {
		env["OPENDRAY_MEMORY_SCOPE_FROM_CWD"] = "1"
	} else if scopeKey != "" {
		env["OPENDRAY_MEMORY_SCOPE_KEY"] = scopeKey
	}

	server := MCPServer{
		Name:    "opendray-memory",
		Command: mem.BinaryPath,
		Args:    []string{"mcp-memory"},
		Env:     env,
	}

	args, mcpEnv, err := renderMCP(providerID, baseDir, runCwd, home, []MCPServer{server})
	if err != nil {
		return nil, nil, err
	}

	// Codex reads MCP servers from <CODEX_HOME>/config.toml, so renderMCP
	// points CODEX_HOME at a scratch codex-home — but that home has no
	// auth.json, and codex would fail to authenticate. Mirror the minimal
	// authenticated subset of the user's real codex home into it (same as
	// the interactive adapter) and trust the run cwd so exec can start.
	if providerID == "codex" && mcpEnv["CODEX_HOME"] != "" {
		userHome := os.Getenv("CODEX_HOME")
		if userHome == "" {
			if h, herr := os.UserHomeDir(); herr == nil {
				userHome = filepath.Join(h, ".codex")
			}
		}
		if userHome != "" {
			if err := mirrorCodexHome(userHome, mcpEnv["CODEX_HOME"]); err != nil {
				return nil, nil, fmt.Errorf("mirror codex home for memory MCP: %w", err)
			}
		}
		if err := ensureCodexScratchTrust(mcpEnv["CODEX_HOME"], runCwd); err != nil {
			return nil, nil, fmt.Errorf("trust codex scratch cwd: %w", err)
		}
	}

	return args, mcpEnv, nil
}

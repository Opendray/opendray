package catalog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
)

// MCPServer is one entry from a provider's user config under the
// `mcp_servers` key. Default transport is stdio. URL/Headers are only
// meaningful for sse / http transports.
type MCPServer struct {
	Name      string            `json:"name"`
	Transport string            `json:"transport,omitempty"` // stdio | sse | http (default stdio)
	Command   string            `json:"command,omitempty"`
	Args      []string          `json:"args,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	URL       string            `json:"url,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
}

// parseMCPServers extracts the user-configured MCP server list from a
// provider's config. Returns nil (not error) on missing / malformed —
// MCP injection is best-effort.
func parseMCPServers(cfg map[string]any) []MCPServer {
	raw, ok := cfg["mcp_servers"]
	if !ok {
		return nil
	}
	body, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var out []MCPServer
	if err := json.Unmarshal(body, &out); err != nil {
		return nil
	}
	return out
}

// parseMCPServersJSON unmarshals a raw JSON array string (same shape as
// the provider config's mcp_servers) into []MCPServer. Returns nil (not
// error) on empty / malformed — integration MCP injection is best-effort,
// mirroring parseMCPServers.
func parseMCPServersJSON(raw string) []MCPServer {
	if raw == "" {
		return nil
	}
	var out []MCPServer
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}

// renderMCP writes per-provider MCP config files into baseDir and
// returns the extra CLI args / env required to make the provider
// pick them up. Provider IDs without a renderer return empty.
//
// cwd is the session's working directory (grok's injection surface lives
// under it). home is the session's effective HOME — the multi-account
// dir when the spawn is account-bound, else the gateway user's real HOME
// — which is antigravity's injection surface.
func renderMCP(providerID, baseDir, cwd, home string, servers []MCPServer) ([]string, map[string]string, error) {
	if len(servers) == 0 {
		return nil, nil, nil
	}
	// Every renderer keys its per-provider config on the server's display
	// name. Two servers sharing a name collide on that key: Claude's map
	// silently drops one, Codex emits a duplicate TOML table that fails to
	// parse and bricks the session before it can print a single byte. Reject
	// the collision here, once, so the failure is a clear operator error
	// rather than a provider-specific mystery — and before any renderer
	// writes a half-formed config file to disk.
	if dup, ok := duplicateServerName(servers); ok {
		return nil, nil, fmt.Errorf("two MCP servers share the name %q; names must be unique", dup)
	}
	switch providerID {
	case "claude":
		return renderClaudeMCP(baseDir, servers)
	case "codex":
		return renderCodexMCP(baseDir, servers)
	case "antigravity":
		// agy reads MCP servers from exactly one file (verified 2026-07-02
		// against agy 1.0.15, empirically with marker stdio servers):
		// <$HOME>/.gemini/config/mcp_config.json — the config surface shared
		// by the whole Antigravity suite (CLI + IDE). It does NOT read the
		// gemini-cli-style <cwd>/.gemini/settings.json (the surface a
		// previous fix targeted — entries there never appear in a session),
		// and it does NOT read a workspace-level .agents/mcp_config.json or
		// <cwd>/.gemini/config/mcp_config.json (both tested, never loaded,
		// git repo or not). `agy plugin` remains a separate, closed
		// import/marketplace system — irrelevant for injection now that the
		// documented mcp_config.json surface works.
		//
		// Because the file is keyed off $HOME, per-account spawns (which
		// point HOME at a dedicated dir) are naturally isolated, while
		// default spawns share the operator's real file — so the merge is
		// managed-entry converged and strictly non-destructive to
		// user-authored servers.
		return renderAgyMCP(home, servers)
	case "opencode":
		// OpenCode reads MCP from its JSON config's `mcp` block; we merge
		// into the per-session OPENCODE_CONFIG file opendray generates.
		return renderOpenCodeMCP(baseDir, servers)
	case "grok":
		// Grok (xAI Build CLI) reads project-scoped MCP from
		// <cwd>/.grok/config.toml (only its [mcp_servers] table) and
		// union-merges it with the user's ~/.grok/config.toml — global-only
		// servers stay untouched, so injecting here never disturbs the
		// operator's personal Grok setup.
		return renderGrokMCP(cwd, servers)
	default:
		// Provider declared supportsMcp=true but we have no renderer
		// for it; surface as a no-op rather than failing the spawn.
		return nil, nil, nil
	}
}

func renderClaudeMCP(baseDir string, servers []MCPServer) ([]string, map[string]string, error) {
	entries := map[string]map[string]any{}
	for _, s := range servers {
		spec := stdioMCPServerSpec(s)
		if spec == nil {
			continue
		}
		entries[s.Name] = spec
	}
	if len(entries) == 0 {
		return nil, nil, nil
	}

	payload := map[string]any{"mcpServers": entries}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("marshal claude mcp: %w", err)
	}

	path := filepath.Join(baseDir, "claude-mcp.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return nil, nil, fmt.Errorf("write claude mcp: %w", err)
	}
	return []string{"--mcp-config", path}, nil, nil
}

// geminiManagedFile records which mcpServers entries inside the
// workspace settings.json are opendray-managed, so subsequent renders
// can update / remove exactly those entries without ever touching
// user-authored ones.
const geminiManagedFile = ".opendray-managed.json"

// agyManagedFile records which mcpServers entries inside the agy global
// mcp_config.json are opendray-managed. Same converge contract as
// geminiManagedFile, different directory.
const agyManagedFile = ".opendray-managed.json"

// renderAgyMCP merges the session's MCP servers into
// <home>/.gemini/config/mcp_config.json — the only file agy loads MCP
// servers from (see the renderMCP antigravity arm). home is the
// session's effective HOME so account-bound spawns write their own
// isolated copy. Merge is non-destructive: user-authored entries are
// preserved verbatim, and only entries we managed before are ever
// updated or removed.
//
// KNOWN LIMITATION: the file is per-HOME, not per-session. Two
// concurrent agy sessions under the same HOME see the union of their
// managed entries, and a spawn converging the file can remove an entry
// a still-running session was using (agy watches the file). The
// opendray-memory entry is identical for every session (no per-session
// env — scope comes from the MCP subprocess cwd, which agy sets to the
// session workspace), so the common case never churns; only concurrent
// integration sessions with different MCP sets can fight.
func renderAgyMCP(home string, servers []MCPServer) ([]string, map[string]string, error) {
	if strings.TrimSpace(home) == "" {
		return nil, nil, nil
	}
	entries := map[string]map[string]any{}
	for _, s := range servers {
		spec := agyMCPServerSpec(s)
		if spec == nil {
			continue
		}
		entries[s.Name] = spec
	}
	if err := syncAgyGlobalMCP(home, entries); err != nil {
		return nil, nil, err
	}
	return nil, nil, nil
}

// agyMCPServerSpec converts one registry server into agy's
// mcp_config.json entry shape. stdio uses command/args/env; remote (sse /
// streamable HTTP) uses serverUrl + headers — agy rejects the legacy
// url/httpUrl field names, and there is no transport-type field (the
// endpoint is probed). Returns nil for entries missing their required
// field.
func agyMCPServerSpec(s MCPServer) map[string]any {
	switch s.Transport {
	case "sse", "http":
		if s.URL == "" {
			return nil
		}
		spec := map[string]any{"serverUrl": s.URL}
		if len(s.Headers) > 0 {
			spec["headers"] = s.Headers
		}
		return spec
	default: // stdio
		if s.Command == "" {
			return nil
		}
		spec := map[string]any{"command": s.Command}
		if len(s.Args) > 0 {
			spec["args"] = s.Args
		}
		if len(s.Env) > 0 {
			spec["env"] = s.Env
		}
		return spec
	}
}

// agyGlobalMCPMu serialises syncAgyGlobalMCP's read-modify-write. The
// workspace-scoped grok/gemini syncs get away without a lock because a
// cwd rarely has two concurrent spawns; the agy file is per-HOME —
// shared by every non-account-bound antigravity spawn across all
// projects — so concurrent Prepares are normal, not an edge case. One
// process-wide mutex is enough: the gateway is the only writer.
var agyGlobalMCPMu sync.Mutex

// syncAgyGlobalMCP applies the desired opendray-managed entries to
// <home>/.gemini/config/mcp_config.json. An empty desired map removes
// every previously managed entry and is a no-op when nothing was ever
// managed — so calling this on every agy spawn keeps the file converged
// with the registry without churning untouched setups.
func syncAgyGlobalMCP(home string, desired map[string]map[string]any) error {
	agyGlobalMCPMu.Lock()
	defer agyGlobalMCPMu.Unlock()

	dir := filepath.Join(home, ".gemini", "config")
	managedPath := filepath.Join(dir, agyManagedFile)
	prevManaged := readGeminiManaged(managedPath)
	if len(desired) == 0 && len(prevManaged) == 0 {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir agy config dir: %w", err)
	}

	configPath := filepath.Join(dir, "mcp_config.json")
	config := map[string]any{}
	if data, err := os.ReadFile(configPath); err == nil {
		// An empty file (agy's first-run/migration writes one) is not a
		// parse error — treat it as "no config yet" and let injection
		// proceed. Only genuinely malformed content is refused.
		if len(bytes.TrimSpace(data)) > 0 {
			if err := json.Unmarshal(data, &config); err != nil {
				// Never risk rewriting a user-authored file we can't parse.
				return fmt.Errorf("parse %s (fix or remove it to enable MCP injection): %w", configPath, err)
			}
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("read agy mcp_config: %w", err)
	}

	mcpServers, _ := config["mcpServers"].(map[string]any)
	if mcpServers == nil {
		mcpServers = map[string]any{}
	}
	// Drop entries we managed before that are no longer desired.
	// User-authored entries are never in the managed list.
	for _, name := range prevManaged {
		if _, still := desired[name]; !still {
			delete(mcpServers, name)
		}
	}
	managed := make([]string, 0, len(desired))
	for name, spec := range desired {
		mcpServers[name] = spec
		managed = append(managed, name)
	}
	sort.Strings(managed)
	config["mcpServers"] = mcpServers

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal agy mcp_config: %w", err)
	}
	// Atomic (temp + rename): agy watches this file, and a torn write
	// would not only feed it invalid JSON but also hard-fail every
	// subsequent sync via the parse guard above. 0600 on create:
	// managed entries may embed resolved credentials (e.g. the
	// opendray-memory gateway key); an existing user file keeps its mode.
	if err := writeFileAtomic(configPath, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write agy mcp_config: %w", err)
	}

	if len(managed) == 0 {
		_ = os.Remove(managedPath)
	} else {
		body, err := json.MarshalIndent(managed, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal agy managed list: %w", err)
		}
		if err := writeFileAtomic(managedPath, append(body, '\n'), 0o600); err != nil {
			return fmt.Errorf("write agy managed list: %w", err)
		}
	}
	return nil
}

// writeFileAtomic writes data to path via a same-directory temp file +
// rename, so concurrent readers (agy watches its mcp_config.json) never
// observe a torn write. An existing file keeps its permission bits;
// mode applies on create only — matching os.WriteFile's semantics.
func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	if fi, err := os.Stat(path); err == nil {
		mode = fi.Mode().Perm()
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op after successful rename
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// syncGeminiWorkspaceMCP applies the desired opendray-managed entries to
// <cwd>/.gemini/settings.json. An empty desired map removes every
// previously managed entry (server disabled / removed from the registry)
// and is a no-op when nothing was ever managed.
//
// LEGACY: a previous fix targeted this file believing agy read it (it
// does not — see the renderMCP antigravity arm). It is kept only so agy
// spawns can purge the stale managed entries (and their embedded
// credentials) that fix left behind in project workspaces; nothing
// renders new entries into it.
func syncGeminiWorkspaceMCP(cwd string, desired map[string]map[string]any) error {
	dir := filepath.Join(cwd, ".gemini")
	managedPath := filepath.Join(dir, geminiManagedFile)
	prevManaged := readGeminiManaged(managedPath)
	if len(desired) == 0 && len(prevManaged) == 0 {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir agy workspace dir: %w", err)
	}

	settingsPath := filepath.Join(dir, "settings.json")
	settings := map[string]any{}
	if data, err := os.ReadFile(settingsPath); err == nil {
		// An empty settings.json is "no config yet", not a parse error.
		if len(bytes.TrimSpace(data)) > 0 {
			if err := json.Unmarshal(data, &settings); err != nil {
				// Never risk rewriting a user-authored file we can't parse.
				return fmt.Errorf("parse %s (fix or remove it to enable MCP injection): %w", settingsPath, err)
			}
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("read agy settings: %w", err)
	}

	mcpServers, _ := settings["mcpServers"].(map[string]any)
	if mcpServers == nil {
		mcpServers = map[string]any{}
	}
	// Drop entries we managed before that are no longer desired.
	// User-authored entries are never in the managed list.
	for _, name := range prevManaged {
		if _, still := desired[name]; !still {
			delete(mcpServers, name)
		}
	}
	managed := make([]string, 0, len(desired))
	for name, spec := range desired {
		mcpServers[name] = spec
		managed = append(managed, name)
	}
	sort.Strings(managed)
	if len(mcpServers) == 0 {
		delete(settings, "mcpServers")
	} else {
		settings["mcpServers"] = mcpServers
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal agy settings: %w", err)
	}
	// 0600 on create: managed entries may embed resolved credentials
	// (e.g. the opendray-memory integration key). WriteFile keeps the
	// existing mode when the user's file already exists.
	if err := os.WriteFile(settingsPath, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write agy settings: %w", err)
	}

	if len(managed) == 0 {
		_ = os.Remove(managedPath)
	} else {
		body, err := json.MarshalIndent(managed, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal agy managed list: %w", err)
		}
		if err := os.WriteFile(managedPath, append(body, '\n'), 0o600); err != nil {
			return fmt.Errorf("write agy managed list: %w", err)
		}
	}
	writeGeminiGitignore(dir)
	return nil
}

// readGeminiManaged loads the managed-entry names from a prior render.
// Missing / malformed → nil (treated as "nothing managed yet").
func readGeminiManaged(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var names []string
	if err := json.Unmarshal(data, &names); err != nil {
		return nil
	}
	return names
}

// writeGeminiGitignore drops a .gitignore next to the workspace settings
// (mirroring how .opendray/ is treated) so the opendray-managed files —
// which can embed integration credentials — never land in version
// control. Only created when missing: a user-authored .gemini/.gitignore
// always wins.
func writeGeminiGitignore(dir string) {
	path := filepath.Join(dir, ".gitignore")
	if _, err := os.Lstat(path); err == nil || !os.IsNotExist(err) {
		return
	}
	content := "# Generated by opendray. settings.json carries opendray-managed MCP\n" +
		"# server entries (may embed integration credentials) — do not commit.\n" +
		"settings.json\n" +
		geminiManagedFile + "\n" +
		".gitignore\n"
	_ = os.WriteFile(path, []byte(content), 0o644)
}

// grokManagedFile records which [mcp_servers] entries inside the
// project config.toml are opendray-managed, so subsequent renders can
// update / remove exactly those without touching user-authored ones.
const grokManagedFile = ".opendray-managed.json"

// renderGrokMCP merges the session's MCP servers into
// <cwd>/.grok/config.toml — the project-scoped surface Grok honours.
// Grok reads ONLY [mcp_servers] from project config and union-merges it
// with the user's global ~/.grok/config.toml (same-named servers are
// replaced, global-only servers untouched), so this never disturbs the
// operator's global Grok setup. Merge is non-destructive: user-authored
// entries in an existing project file are preserved.
func renderGrokMCP(cwd string, servers []MCPServer) ([]string, map[string]string, error) {
	if strings.TrimSpace(cwd) == "" {
		return nil, nil, nil
	}
	entries := map[string]map[string]any{}
	for _, s := range servers {
		spec := grokMCPServerSpec(s)
		if spec == nil {
			continue
		}
		entries[s.Name] = spec
	}
	if err := syncGrokWorkspaceMCP(cwd, entries); err != nil {
		return nil, nil, err
	}
	return nil, nil, nil
}

// syncGrokWorkspaceMCP applies the desired opendray-managed [mcp_servers]
// entries to <cwd>/.grok/config.toml. An empty desired map removes every
// previously managed entry (server disabled / removed from the registry)
// and is a no-op when nothing was ever managed — so calling this on every
// grok spawn keeps the project config converged with the registry without
// churning untouched projects.
func syncGrokWorkspaceMCP(cwd string, desired map[string]map[string]any) error {
	dir := filepath.Join(cwd, ".grok")
	managedPath := filepath.Join(dir, grokManagedFile)
	prevManaged := readGrokManaged(managedPath)
	if len(desired) == 0 && len(prevManaged) == 0 {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir grok workspace dir: %w", err)
	}

	configPath := filepath.Join(dir, "config.toml")
	config := map[string]any{}
	if data, err := os.ReadFile(configPath); err == nil {
		if err := toml.Unmarshal(data, &config); err != nil {
			// Never risk rewriting a user-authored file we can't parse.
			return fmt.Errorf("parse %s (fix or remove it to enable MCP injection): %w", configPath, err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("read grok config: %w", err)
	}

	mcpServers, _ := config["mcp_servers"].(map[string]any)
	if mcpServers == nil {
		mcpServers = map[string]any{}
	}
	// Drop entries we managed before that are no longer desired.
	// User-authored entries are never in the managed list.
	for _, name := range prevManaged {
		if _, still := desired[name]; !still {
			delete(mcpServers, name)
		}
	}
	managed := make([]string, 0, len(desired))
	for name, spec := range desired {
		mcpServers[name] = spec
		managed = append(managed, name)
	}
	sort.Strings(managed)
	if len(mcpServers) == 0 {
		delete(config, "mcp_servers")
	} else {
		config["mcp_servers"] = mcpServers
	}

	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(config); err != nil {
		return fmt.Errorf("marshal grok config: %w", err)
	}
	// 0600: managed entries may embed resolved credentials (e.g. a Vault
	// token). WriteFile keeps the existing mode when the file already exists.
	if err := os.WriteFile(configPath, buf.Bytes(), 0o600); err != nil {
		return fmt.Errorf("write grok config: %w", err)
	}

	if len(managed) == 0 {
		_ = os.Remove(managedPath)
	} else {
		body, err := json.MarshalIndent(managed, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal grok managed list: %w", err)
		}
		if err := os.WriteFile(managedPath, append(body, '\n'), 0o600); err != nil {
			return fmt.Errorf("write grok managed list: %w", err)
		}
	}
	writeGrokGitignore(dir)
	return nil
}

// readGrokManaged loads the managed-entry names from a prior render.
// Missing / malformed → nil (treated as "nothing managed yet").
func readGrokManaged(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var names []string
	if err := json.Unmarshal(data, &names); err != nil {
		return nil
	}
	return names
}

// writeGrokGitignore drops a .gitignore next to the project config so the
// opendray-managed config.toml — which can embed resolved credentials —
// never lands in version control. Only created when missing: a
// user-authored .grok/.gitignore always wins.
func writeGrokGitignore(dir string) {
	path := filepath.Join(dir, ".gitignore")
	if _, err := os.Lstat(path); err == nil || !os.IsNotExist(err) {
		return
	}
	content := "# Generated by opendray. config.toml carries opendray-managed MCP\n" +
		"# server entries (may embed resolved credentials) — do not commit.\n" +
		"config.toml\n" +
		grokManagedFile + "\n" +
		".gitignore\n"
	_ = os.WriteFile(path, []byte(content), 0o644)
}

// grokMCPServerSpec converts one registry server into Grok's TOML
// [mcp_servers.<name>] table shape. stdio uses command/args/env; sse uses
// url + type="sse"; streamable HTTP uses url alone (Grok's default when no
// type is set). Returns nil for entries missing their required field.
func grokMCPServerSpec(s MCPServer) map[string]any {
	switch s.Transport {
	case "sse":
		if s.URL == "" {
			return nil
		}
		spec := map[string]any{"url": s.URL, "type": "sse"}
		if len(s.Headers) > 0 {
			spec["headers"] = s.Headers
		}
		return spec
	case "http":
		if s.URL == "" {
			return nil
		}
		spec := map[string]any{"url": s.URL}
		if len(s.Headers) > 0 {
			spec["headers"] = s.Headers
		}
		return spec
	default: // stdio
		if s.Command == "" {
			return nil
		}
		spec := map[string]any{"command": s.Command}
		if len(s.Args) > 0 {
			spec["args"] = s.Args
		}
		if len(s.Env) > 0 {
			spec["env"] = s.Env
		}
		return spec
	}
}

func stdioMCPServerSpec(s MCPServer) map[string]any {
	switch s.Transport {
	case "sse":
		if s.URL == "" {
			return nil
		}
		spec := map[string]any{"type": "sse", "url": s.URL}
		if len(s.Headers) > 0 {
			spec["headers"] = s.Headers
		}
		return spec
	case "http":
		if s.URL == "" {
			return nil
		}
		spec := map[string]any{"type": "http", "url": s.URL}
		if len(s.Headers) > 0 {
			spec["headers"] = s.Headers
		}
		return spec
	default: // stdio
		if s.Command == "" {
			return nil
		}
		spec := map[string]any{"command": s.Command}
		if len(s.Args) > 0 {
			spec["args"] = s.Args
		}
		if len(s.Env) > 0 {
			spec["env"] = s.Env
		}
		return spec
	}
}

func renderCodexMCP(baseDir string, servers []MCPServer) ([]string, map[string]string, error) {
	var blocks []string
	for _, s := range servers {
		// Codex stable supports stdio only.
		if s.Transport != "" && s.Transport != "stdio" {
			continue
		}
		if s.Command == "" {
			continue
		}
		blocks = append(blocks, codexServerBlock(s))
	}
	if len(blocks) == 0 {
		return nil, nil, nil
	}

	home := filepath.Join(baseDir, "codex-home")
	if err := os.MkdirAll(home, 0o700); err != nil {
		return nil, nil, fmt.Errorf("mkdir codex home: %w", err)
	}
	path := filepath.Join(home, "config.toml")
	body := codexBaseConfigForScratch()
	if strings.TrimSpace(body) != "" {
		body = strings.TrimRight(body, "\n") + "\n\n"
	}
	body += strings.Join(blocks, "\n\n") + "\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		return nil, nil, fmt.Errorf("write codex config: %w", err)
	}
	return nil, map[string]string{"CODEX_HOME": home}, nil
}

func codexBaseConfigForScratch() string {
	home := os.Getenv("CODEX_HOME")
	if home == "" {
		if h, err := os.UserHomeDir(); err == nil {
			home = filepath.Join(h, ".codex")
		}
	}
	if home == "" {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(home, "config.toml"))
	if err != nil {
		return ""
	}
	return string(data)
}

// duplicateServerName reports the first display name shared by two
// renderable servers (those that would actually be emitted — i.e. carry a
// command or URL). Servers dropped for being invalid can't collide, so
// they're ignored.
func duplicateServerName(servers []MCPServer) (string, bool) {
	seen := make(map[string]bool, len(servers))
	for _, s := range servers {
		if s.Command == "" && s.URL == "" {
			continue
		}
		if seen[s.Name] {
			return s.Name, true
		}
		seen[s.Name] = true
	}
	return "", false
}

func codexServerBlock(s MCPServer) string {
	var b strings.Builder
	fmt.Fprintf(&b, "[mcp_servers.%s]\n", tomlKey(s.Name))
	fmt.Fprintf(&b, "command = %s\n", tomlString(s.Command))
	if len(s.Args) > 0 {
		b.WriteString("args = [")
		for i, a := range s.Args {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(tomlString(a))
		}
		b.WriteString("]\n")
	}
	if len(s.Env) > 0 {
		keys := make([]string, 0, len(s.Env))
		for k := range s.Env {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		b.WriteString("env = { ")
		for i, k := range keys {
			if i > 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(&b, "%s = %s", tomlKey(k), tomlString(s.Env[k]))
		}
		b.WriteString(" }\n")
	}
	return b.String()
}

func tomlKey(k string) string {
	for _, r := range k {
		if r == '_' || r == '-' ||
			(r >= '0' && r <= '9') ||
			(r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') {
			continue
		}
		return tomlString(k)
	}
	if k == "" {
		return tomlString(k)
	}
	return k
}

func tomlString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

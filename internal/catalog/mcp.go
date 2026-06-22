package catalog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

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
// cwd is the session's working directory. Only antigravity needs it: agy
// reuses gemini-cli's config layout and ignores a scratch config dir, so
// the only injection surface is the workspace settings file at
// <cwd>/.gemini/settings.json.
func renderMCP(providerID, baseDir, cwd string, servers []MCPServer) ([]string, map[string]string, error) {
	if len(servers) == 0 {
		return nil, nil, nil
	}
	switch providerID {
	case "claude":
		return renderClaudeMCP(baseDir, servers)
	case "codex":
		return renderCodexMCP(baseDir, servers)
	case "antigravity":
		// Antigravity (agy) is gemini-cli's successor and reads the
		// workspace <cwd>/.gemini/settings.json mcpServers map (verified
		// live; it starts servers lazily, on first tool need — a fake
		// server + a non-forcing prompt won't trigger the spawn, which is
		// why an earlier probe looked negative). The non-destructive merge
		// attaches opendray-memory per-session, never globally.
		return renderGeminiMCP(cwd, servers)
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

// renderGeminiMCP merges the session's MCP servers into
// <cwd>/.gemini/settings.json — the workspace config surface Antigravity
// (agy) honours (it reuses gemini-cli's settings layout) besides the
// user-level ~/.gemini/settings.json (which holds auth + personal config
// and must not be clobbered). Merge is non-destructive: existing user
// keys and user mcpServers entries are preserved verbatim.
func renderGeminiMCP(cwd string, servers []MCPServer) ([]string, map[string]string, error) {
	if strings.TrimSpace(cwd) == "" {
		return nil, nil, nil
	}
	entries := map[string]map[string]any{}
	for _, s := range servers {
		spec := stdioMCPServerSpec(s)
		if spec == nil {
			continue
		}
		entries[s.Name] = spec
	}
	if err := syncGeminiWorkspaceMCP(cwd, entries); err != nil {
		return nil, nil, err
	}
	return nil, nil, nil
}

// syncGeminiWorkspaceMCP applies the desired opendray-managed entries to
// <cwd>/.gemini/settings.json. An empty desired map removes every
// previously managed entry (server disabled / removed from the registry)
// and is a no-op when nothing was ever managed — so calling this on
// every agy spawn keeps the workspace file converged with the
// registry without churning untouched projects.
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
		if err := json.Unmarshal(data, &settings); err != nil {
			// Never risk rewriting a user-authored file we can't parse.
			return fmt.Errorf("parse %s (fix or remove it to enable MCP injection): %w", settingsPath, err)
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

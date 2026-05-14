package catalog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

// renderMCP writes per-provider MCP config files into baseDir and
// returns the extra CLI args / env required to make the provider
// pick them up. Provider IDs without a renderer return empty.
func renderMCP(providerID, baseDir string, servers []MCPServer) ([]string, map[string]string, error) {
	if len(servers) == 0 {
		return nil, nil, nil
	}
	switch providerID {
	case "claude":
		return renderClaudeMCP(baseDir, servers)
	case "codex":
		return renderCodexMCP(baseDir, servers)
	case "gemini":
		return renderGeminiMCP(baseDir, servers)
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

func renderGeminiMCP(baseDir string, servers []MCPServer) ([]string, map[string]string, error) {
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

	home := filepath.Join(baseDir, "gemini-home")
	if err := os.MkdirAll(home, 0o700); err != nil {
		return nil, nil, fmt.Errorf("mkdir gemini home: %w", err)
	}

	payload := map[string]any{"mcpServers": entries}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("marshal gemini settings: %w", err)
	}

	path := filepath.Join(home, "settings.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return nil, nil, fmt.Errorf("write gemini settings: %w", err)
	}
	return nil, map[string]string{"GEMINI_CONFIG_DIR": home}, nil
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

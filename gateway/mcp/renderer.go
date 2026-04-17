package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/linivek/ntc/kernel/store"
)

// Injection is the output of rendering MCP servers for one session.
// It is merged onto the base ResolvedCLI the plugin runtime produced.
type Injection struct {
	Args []string
	Env  map[string]string
}

// renderer turns a filtered list of MCP servers into an Injection by
// writing a tool-specific config file inside baseDir. Each supported
// agent gets its own implementation.
type renderer interface {
	render(baseDir string, servers []store.MCPServer) (Injection, error)
}

// renderers is the registry of per-agent renderers. Agents not present
// here get no MCP injection — intentional, e.g. gemini / qwen-code.
var renderers = map[string]renderer{
	"claude": claudeRenderer{},
	"codex":  codexRenderer{},
}

// ── Claude ──────────────────────────────────────────────────────
// Claude Code CLI accepts --mcp-config <file> pointing at a JSON doc
// shaped like { "mcpServers": { <name>: <spec> } }.

type claudeRenderer struct{}

func (claudeRenderer) render(baseDir string, servers []store.MCPServer) (Injection, error) {
	if len(servers) == 0 {
		return Injection{}, nil
	}

	entries := map[string]map[string]any{}
	for _, s := range servers {
		spec := claudeServerSpec(s)
		if spec == nil {
			continue
		}
		entries[s.Name] = spec
	}
	if len(entries) == 0 {
		return Injection{}, nil
	}

	payload := map[string]any{"mcpServers": entries}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return Injection{}, fmt.Errorf("mcp: marshal claude config: %w", err)
	}

	path := filepath.Join(baseDir, "claude-mcp.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return Injection{}, fmt.Errorf("mcp: write claude config: %w", err)
	}

	return Injection{Args: []string{"--mcp-config", path}}, nil
}

func claudeServerSpec(s store.MCPServer) map[string]any {
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

// ── Codex ───────────────────────────────────────────────────────
// Codex CLI reads ~/.codex/config.toml. We redirect it to a scratch
// dir via CODEX_HOME, then write only the mcp_servers section.
// Stable Codex supports stdio transport only; sse/http are skipped.

type codexRenderer struct{}

func (codexRenderer) render(baseDir string, servers []store.MCPServer) (Injection, error) {
	if len(servers) == 0 {
		return Injection{}, nil
	}

	var blocks []string
	for _, s := range servers {
		if s.Transport != "stdio" || s.Command == "" {
			continue
		}
		blocks = append(blocks, codexServerBlock(s))
	}
	if len(blocks) == 0 {
		return Injection{}, nil
	}

	home := filepath.Join(baseDir, "codex-home")
	if err := os.MkdirAll(home, 0o700); err != nil {
		return Injection{}, fmt.Errorf("mcp: mkdir codex home: %w", err)
	}
	path := filepath.Join(home, "config.toml")
	body := strings.Join(blocks, "\n\n") + "\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		return Injection{}, fmt.Errorf("mcp: write codex config: %w", err)
	}

	return Injection{Env: map[string]string{"CODEX_HOME": home}}, nil
}

func codexServerBlock(s store.MCPServer) string {
	var b strings.Builder
	fmt.Fprintf(&b, "[mcp_servers.%s]\n", tomlKey(s.Name))
	fmt.Fprintf(&b, "command = %s\n", tomlString(s.Command))

	if len(s.Args) > 0 {
		fmt.Fprintf(&b, "args = [")
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

// tomlKey bare-quotes if the key contains anything outside [A-Za-z0-9_-].
func tomlKey(k string) string {
	safe := true
	for _, r := range k {
		if !(r == '_' || r == '-' ||
			(r >= '0' && r <= '9') ||
			(r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z')) {
			safe = false
			break
		}
	}
	if safe && k != "" {
		return k
	}
	return tomlString(k)
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

// supportsAgent returns true when we know how to inject for this agent.
func supportsAgent(name string) bool {
	_, ok := renderers[name]
	return ok
}

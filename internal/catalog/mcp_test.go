package catalog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderClaudeMCP_StdioServer(t *testing.T) {
	dir := t.TempDir()
	servers := []MCPServer{
		{Name: "fs", Command: "npx", Args: []string{"-y", "@modelcontextprotocol/server-filesystem", "/tmp"}},
	}
	args, env, err := renderMCP("claude", dir, servers)
	if err != nil {
		t.Fatal(err)
	}
	if len(env) != 0 {
		t.Errorf("env=%v, want none", env)
	}
	if len(args) != 2 || args[0] != "--mcp-config" {
		t.Fatalf("args=%v", args)
	}
	body, err := os.ReadFile(args[1])
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	servers0 := got["mcpServers"].(map[string]any)["fs"].(map[string]any)
	if servers0["command"].(string) != "npx" {
		t.Errorf("command=%v", servers0["command"])
	}
}

func TestRenderClaudeMCP_HTTPServer(t *testing.T) {
	dir := t.TempDir()
	servers := []MCPServer{
		{Name: "remote", Transport: "http", URL: "https://api.example.com/mcp",
			Headers: map[string]string{"Authorization": "Bearer xyz"}},
	}
	args, _, err := renderMCP("claude", dir, servers)
	if err != nil || len(args) != 2 {
		t.Fatalf("unexpected args=%v err=%v", args, err)
	}
	body, _ := os.ReadFile(args[1])
	if !strings.Contains(string(body), `"type": "http"`) ||
		!strings.Contains(string(body), `"https://api.example.com/mcp"`) {
		t.Errorf("missing http transport bits: %s", body)
	}
}

func TestRenderClaudeMCP_DropsInvalid(t *testing.T) {
	dir := t.TempDir()
	servers := []MCPServer{
		{Name: "no-command"},                                  // dropped (no command)
		{Name: "no-url", Transport: "sse"},                    // dropped (no url)
		{Name: "ok", Command: "node", Args: []string{"x.js"}}, // kept
	}
	args, _, err := renderMCP("claude", dir, servers)
	if err != nil || len(args) != 2 {
		t.Fatalf("args=%v err=%v", args, err)
	}
	body, _ := os.ReadFile(args[1])
	if !strings.Contains(string(body), `"ok"`) {
		t.Error("ok server missing")
	}
	if strings.Contains(string(body), "no-command") || strings.Contains(string(body), "no-url") {
		t.Error("invalid servers leaked into output")
	}
}

func TestRenderCodexMCP_TomlOutput(t *testing.T) {
	t.Setenv("CODEX_HOME", t.TempDir())
	dir := t.TempDir()
	servers := []MCPServer{
		{Name: "fs", Command: "npx", Args: []string{"-y", "server-fs"},
			Env: map[string]string{"DEBUG": "1", "PATH": "/usr/bin"}},
	}
	args, env, err := renderMCP("codex", dir, servers)
	if err != nil {
		t.Fatal(err)
	}
	if len(args) != 0 {
		t.Errorf("args=%v, want none", args)
	}
	home, ok := env["CODEX_HOME"]
	if !ok {
		t.Fatalf("CODEX_HOME missing from env=%v", env)
	}
	body, err := os.ReadFile(filepath.Join(home, "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	str := string(body)
	if !strings.Contains(str, "[mcp_servers.fs]") {
		t.Errorf("missing mcp_servers.fs section: %s", str)
	}
	if !strings.Contains(str, `command = "npx"`) {
		t.Errorf("missing command field: %s", str)
	}
	// env keys are sorted in the renderer
	if idx := strings.Index(str, "env = {"); idx == -1 ||
		!strings.Contains(str[idx:], `DEBUG = "1"`) ||
		!strings.Contains(str[idx:], `PATH = "/usr/bin"`) {
		t.Errorf("env block malformed: %s", str)
	}
}

func TestRenderCodexMCP_PreservesUserConfig(t *testing.T) {
	userHome := t.TempDir()
	t.Setenv("CODEX_HOME", userHome)
	if err := os.WriteFile(filepath.Join(userHome, "config.toml"), []byte(`model = "gpt-5.4"

[projects."/repo"]
trust_level = "trusted"
`), 0o600); err != nil {
		t.Fatal(err)
	}

	_, env, err := renderMCP("codex", t.TempDir(), []MCPServer{
		{Name: "fs", Command: "npx"},
	})
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(env["CODEX_HOME"], "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	str := string(body)
	if !strings.Contains(str, `model = "gpt-5.4"`) {
		t.Errorf("user model config missing: %s", str)
	}
	if !strings.Contains(str, `[projects."/repo"]`) {
		t.Errorf("user project trust missing: %s", str)
	}
	if !strings.Contains(str, `[mcp_servers.fs]`) {
		t.Errorf("mcp server missing: %s", str)
	}
}

func TestRenderCodexMCP_SkipsNonStdio(t *testing.T) {
	t.Setenv("CODEX_HOME", t.TempDir())
	dir := t.TempDir()
	servers := []MCPServer{
		{Name: "remote", Transport: "http", URL: "https://x"},
		{Name: "stdio", Command: "node"},
	}
	_, env, err := renderMCP("codex", dir, servers)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := os.ReadFile(filepath.Join(env["CODEX_HOME"], "config.toml"))
	if strings.Contains(string(body), "remote") {
		t.Errorf("non-stdio server leaked: %s", body)
	}
	if !strings.Contains(string(body), "stdio") {
		t.Errorf("stdio server missing: %s", body)
	}
}

func TestRenderGeminiMCP_SettingsOutput(t *testing.T) {
	dir := t.TempDir()
	servers := []MCPServer{
		{Name: "fs", Command: "npx", Args: []string{"-y", "server-fs"}},
	}
	args, env, err := renderMCP("gemini", dir, servers)
	if err != nil {
		t.Fatal(err)
	}
	if len(args) != 0 {
		t.Errorf("args=%v, want none", args)
	}
	home, ok := env["GEMINI_CONFIG_DIR"]
	if !ok {
		t.Fatalf("GEMINI_CONFIG_DIR missing from env=%v", env)
	}
	body, err := os.ReadFile(filepath.Join(home, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	servers0 := got["mcpServers"].(map[string]any)["fs"].(map[string]any)
	if servers0["command"].(string) != "npx" {
		t.Errorf("command=%v", servers0["command"])
	}
}

func TestRenderMCP_UnknownProviderNoOp(t *testing.T) {
	args, env, err := renderMCP("unknown-provider", t.TempDir(), []MCPServer{
		{Name: "x", Command: "y"},
	})
	if err != nil || args != nil || env != nil {
		t.Errorf("expected no-op for unknown-provider, got args=%v env=%v err=%v", args, env, err)
	}
}

func TestRenderMCP_EmptyServers(t *testing.T) {
	args, env, err := renderMCP("claude", t.TempDir(), nil)
	if err != nil || args != nil || env != nil {
		t.Errorf("expected no-op for empty servers, got args=%v env=%v err=%v", args, env, err)
	}
}

func TestParseMCPServers_FromMap(t *testing.T) {
	cfg := map[string]any{
		"mcp_servers": []any{
			map[string]any{"name": "fs", "command": "npx", "args": []any{"-y", "x"}},
			map[string]any{"name": "remote", "transport": "http", "url": "https://x"},
		},
	}
	got := parseMCPServers(cfg)
	if len(got) != 2 {
		t.Fatalf("got %d, want 2", len(got))
	}
	if got[0].Name != "fs" || got[0].Command != "npx" {
		t.Errorf("fs: %+v", got[0])
	}
	if got[1].Transport != "http" || got[1].URL != "https://x" {
		t.Errorf("remote: %+v", got[1])
	}
}

func TestParseMCPServers_Missing(t *testing.T) {
	if parseMCPServers(map[string]any{}) != nil {
		t.Error("expected nil for missing key")
	}
}

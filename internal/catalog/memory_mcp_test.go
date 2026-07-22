package catalog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// Disabled / unconfigured memory auto-attach is a no-op so callers can wire it
// unconditionally and degrade to a tool-less spawn.
func TestAttachMemoryMCP_DisabledIsNoop(t *testing.T) {
	args, env, err := AttachMemoryMCP(MemoryAutoAttach{}, "claude", t.TempDir(), "/cwd", "/cwd", "/home", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if args != nil || env != nil {
		t.Fatalf("disabled attach must return nil args/env, got args=%v env=%v", args, env)
	}
}

// For claude the helper writes claude-mcp.json (via --mcp-config) carrying the
// opendray-memory server with the read-only flag + the scope key baked in.
func TestAttachMemoryMCP_ClaudeReadOnly(t *testing.T) {
	base := t.TempDir()
	mem := MemoryAutoAttach{
		Enabled:    true,
		BinaryPath: "/usr/local/bin/opendray",
		BaseURL:    "http://127.0.0.1:8770",
		APIKey:     "secret-key",
		Scope:      "project",
	}
	args, _, err := AttachMemoryMCP(mem, "claude", base, base, "/proj/foo", "/home/op", true)
	if err != nil {
		t.Fatalf("attach: %v", err)
	}

	// --mcp-config <path> is returned and the file exists.
	var cfgPath string
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--mcp-config" {
			cfgPath = args[i+1]
		}
	}
	if cfgPath == "" {
		t.Fatalf("expected --mcp-config in args, got %v", args)
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read %s: %v", cfgPath, err)
	}

	var parsed struct {
		MCPServers map[string]struct {
			Command string            `json:"command"`
			Args    []string          `json:"args"`
			Env     map[string]string `json:"env"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("parse claude-mcp.json: %v", err)
	}
	srv, ok := parsed.MCPServers["opendray-memory"]
	if !ok {
		t.Fatalf("opendray-memory server missing: %s", data)
	}
	if srv.Env["OPENDRAY_MEMORY_READONLY"] != "1" {
		t.Errorf("read-only flag not set: %v", srv.Env)
	}
	if srv.Env["OPENDRAY_MEMORY_SCOPE_KEY"] != "/proj/foo" {
		t.Errorf("scope key = %q, want /proj/foo", srv.Env["OPENDRAY_MEMORY_SCOPE_KEY"])
	}
	if srv.Env["OPENDRAY_API_KEY"] != "secret-key" {
		t.Errorf("api key not propagated: %v", srv.Env)
	}
}

// Antigravity derives its scope from the subprocess cwd (its entry lives in a
// HOME-global file), so the helper opts into FROM_CWD instead of baking a key.
func TestAttachMemoryMCP_AntigravityUsesFromCwd(t *testing.T) {
	home := t.TempDir()
	mem := MemoryAutoAttach{
		Enabled: true, BinaryPath: "/bin/opendray",
		BaseURL: "http://127.0.0.1:8770", APIKey: "k", Scope: "project",
	}
	if _, _, err := AttachMemoryMCP(mem, "antigravity", t.TempDir(), "/proj/foo", "/proj/foo", home, true); err != nil {
		t.Fatalf("attach: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(home, ".gemini", "config", "mcp_config.json"))
	if err != nil {
		t.Fatalf("read agy mcp_config: %v", err)
	}
	var parsed struct {
		MCPServers map[string]struct {
			Env map[string]string `json:"env"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("parse: %v", err)
	}
	env := parsed.MCPServers["opendray-memory"].Env
	if env["OPENDRAY_MEMORY_SCOPE_FROM_CWD"] != "1" {
		t.Errorf("antigravity should derive scope from cwd, env=%v", env)
	}
	if _, baked := env["OPENDRAY_MEMORY_SCOPE_KEY"]; baked {
		t.Errorf("antigravity must NOT bake a scope key (leaks across shared HOME file): %v", env)
	}
	if env["OPENDRAY_MEMORY_READONLY"] != "1" {
		t.Errorf("read-only flag not set: %v", env)
	}
}

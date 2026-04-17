package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/opendray/opendray/kernel/store"
)

func TestClaudeRenderer_StdioAndSSE(t *testing.T) {
	dir := t.TempDir()
	servers := []store.MCPServer{
		{
			Name: "fs", Transport: "stdio",
			Command: "npx", Args: []string{"-y", "@modelcontextprotocol/server-filesystem", "/tmp"},
			Env: map[string]string{"API_KEY": "abc"},
		},
		{
			Name: "sentry", Transport: "sse",
			URL:     "https://mcp.sentry.dev/sse",
			Headers: map[string]string{"Authorization": "Bearer xyz"},
		},
		{
			Name: "disabled-missing-cmd", Transport: "stdio", // Command empty — skipped
		},
	}

	inj, err := claudeRenderer{}.render(dir, servers)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if len(inj.Args) != 2 || inj.Args[0] != "--mcp-config" {
		t.Fatalf("unexpected args: %v", inj.Args)
	}

	data, err := os.ReadFile(inj.Args[1])
	if err != nil {
		t.Fatalf("read rendered: %v", err)
	}
	var parsed struct {
		MCPServers map[string]map[string]any `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal rendered: %v", err)
	}
	if _, ok := parsed.MCPServers["fs"]; !ok {
		t.Fatalf("fs server missing")
	}
	if _, ok := parsed.MCPServers["sentry"]; !ok {
		t.Fatalf("sentry server missing")
	}
	if _, ok := parsed.MCPServers["disabled-missing-cmd"]; ok {
		t.Fatalf("server with empty Command should have been skipped")
	}
	if parsed.MCPServers["sentry"]["type"] != "sse" {
		t.Fatalf("sentry type should be sse")
	}
}

func TestClaudeRenderer_Empty(t *testing.T) {
	inj, err := claudeRenderer{}.render(t.TempDir(), nil)
	if err != nil {
		t.Fatalf("render empty: %v", err)
	}
	if len(inj.Args) != 0 || len(inj.Env) != 0 {
		t.Fatalf("expected zero injection, got %+v", inj)
	}
}

func TestCodexRenderer_SkipsNonStdio(t *testing.T) {
	dir := t.TempDir()
	servers := []store.MCPServer{
		{Name: "sse-only", Transport: "sse", URL: "https://x"}, // skipped
		{
			Name: "fs", Transport: "stdio",
			Command: "npx", Args: []string{"-y", "fs-mcp"},
			Env:     map[string]string{"TOKEN": `he said "hi"`},
		},
	}

	inj, err := codexRenderer{}.render(dir, servers)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	home, ok := inj.Env["CODEX_HOME"]
	if !ok || home == "" {
		t.Fatalf("CODEX_HOME not set: %v", inj.Env)
	}

	data, err := os.ReadFile(filepath.Join(home, "config.toml"))
	if err != nil {
		t.Fatalf("read config.toml: %v", err)
	}
	s := string(data)
	if !strings.Contains(s, "[mcp_servers.fs]") {
		t.Fatalf("fs block missing:\n%s", s)
	}
	if strings.Contains(s, "[mcp_servers.sse-only]") {
		t.Fatalf("sse server should be skipped:\n%s", s)
	}
	// Escaped quote inside env value.
	if !strings.Contains(s, `\"hi\"`) {
		t.Fatalf("env value quote not escaped:\n%s", s)
	}
}

func TestCodexRenderer_EmptyWhenAllNonStdio(t *testing.T) {
	inj, err := codexRenderer{}.render(t.TempDir(), []store.MCPServer{
		{Name: "a", Transport: "sse", URL: "https://x"},
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if len(inj.Args) != 0 || len(inj.Env) != 0 {
		t.Fatalf("expected zero injection, got %+v", inj)
	}
}

func TestFilterForAgent(t *testing.T) {
	servers := []store.MCPServer{
		{Name: "a", Enabled: true, AppliesTo: []string{"*"}},
		{Name: "b", Enabled: true, AppliesTo: []string{"claude"}},
		{Name: "c", Enabled: true, AppliesTo: []string{"codex"}},
		{Name: "d", Enabled: false, AppliesTo: []string{"*"}},   // disabled
		{Name: "e", Enabled: true, AppliesTo: []string{}},       // empty = all
	}

	got := filterForAgent(servers, "claude")
	names := make([]string, len(got))
	for i, s := range got {
		names[i] = s.Name
	}
	want := []string{"a", "b", "e"}
	if len(names) != len(want) {
		t.Fatalf("claude: got %v want %v", names, want)
	}
	for i, n := range want {
		if names[i] != n {
			t.Fatalf("claude: got %v want %v", names, want)
		}
	}

	got = filterForAgent(servers, "codex")
	if len(got) != 3 { // a, c, e
		t.Fatalf("codex filter: got %d want 3", len(got))
	}
}

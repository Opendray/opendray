package catalog

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/opendray/opendray-v2/internal/session"
)

func readOpenCodeConfig(t *testing.T, baseDir string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(openCodeConfigPath(baseDir))
	if err != nil {
		t.Fatalf("read generated config: %v", err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse generated config: %v", err)
	}
	return cfg
}

func TestInjectOpenCodeLocalProvider(t *testing.T) {
	tests := []struct {
		name        string
		cfg         map[string]any
		wantFile    bool
		wantModel   string // expected top-level default "model" ("" = unset)
		wantBaseURL string
	}{
		{
			name:     "no local endpoint is a no-op",
			cfg:      map[string]any{},
			wantFile: false,
		},
		{
			name:        "base url registers provider and default-selects model",
			cfg:         map[string]any{"localBaseUrl": "http://localhost:1234/v1", "localModel": "qwen3-coder"},
			wantFile:    true,
			wantModel:   "opendray-local/qwen3-coder",
			wantBaseURL: "http://localhost:1234/v1",
		},
		{
			name:        "explicit model wins over local default",
			cfg:         map[string]any{"localBaseUrl": "http://localhost:1234/v1", "localModel": "qwen3-coder", "model": "anthropic/claude-sonnet-4-6"},
			wantFile:    true,
			wantModel:   "", // top-level model stays unset; --model flag handles it
			wantBaseURL: "http://localhost:1234/v1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			out := &session.PrepareOutput{Env: map[string]string{}}
			if err := injectOpenCodeLocalProvider(dir, tt.cfg, out); err != nil {
				t.Fatalf("injectOpenCodeLocalProvider: %v", err)
			}
			if _, err := os.Stat(openCodeConfigPath(dir)); (err == nil) != tt.wantFile {
				t.Fatalf("config file present=%v, want %v", err == nil, tt.wantFile)
			}
			if !tt.wantFile {
				if out.Env["OPENCODE_CONFIG"] != "" {
					t.Errorf("OPENCODE_CONFIG set on no-op")
				}
				return
			}
			if out.Env["OPENCODE_CONFIG"] != openCodeConfigPath(dir) {
				t.Errorf("OPENCODE_CONFIG=%q, want %q", out.Env["OPENCODE_CONFIG"], openCodeConfigPath(dir))
			}
			cfg := readOpenCodeConfig(t, dir)
			provider, ok := cfg["provider"].(map[string]any)
			if !ok {
				t.Fatalf("provider block missing: %v", cfg)
			}
			local, ok := provider[openCodeLocalProvider].(map[string]any)
			if !ok {
				t.Fatalf("opendray-local provider missing: %v", provider)
			}
			opts, _ := local["options"].(map[string]any)
			if got, _ := opts["baseURL"].(string); got != tt.wantBaseURL {
				t.Errorf("baseURL=%q, want %q", got, tt.wantBaseURL)
			}
			if got, _ := cfg["model"].(string); got != tt.wantModel {
				t.Errorf("default model=%q, want %q", got, tt.wantModel)
			}
		})
	}
}

func TestWantsOpenCodeSessionConfig(t *testing.T) {
	tests := []struct {
		name string
		id   string
		cfg  map[string]any
		want bool
	}{
		{"opencode with local endpoint", "opencode", map[string]any{"localBaseUrl": "http://localhost:1234/v1"}, true},
		{"opencode whitespace-only endpoint", "opencode", map[string]any{"localBaseUrl": "   "}, false},
		{"opencode no endpoint", "opencode", map[string]any{}, false},
		{"non-opencode with endpoint key", "claude", map[string]any{"localBaseUrl": "http://localhost:1234/v1"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := wantsOpenCodeSessionConfig(tt.id, tt.cfg); got != tt.want {
				t.Errorf("wantsOpenCodeSessionConfig(%q)=%v, want %v", tt.id, got, tt.want)
			}
		})
	}
}

func TestRenderOpenCodeMCP(t *testing.T) {
	dir := t.TempDir()
	servers := []MCPServer{
		{Name: "opendray-memory", Command: "/bin/opendray", Args: []string{"mcp-memory"}, Env: map[string]string{"OPENDRAY_API_KEY": "k"}},
		{Name: "remote-thing", URL: "https://example.com/mcp", Headers: map[string]string{"Authorization": "Bearer x"}},
	}
	args, env, err := renderOpenCodeMCP(dir, servers)
	if err != nil {
		t.Fatalf("renderOpenCodeMCP: %v", err)
	}
	if args != nil {
		t.Errorf("expected no CLI args, got %v", args)
	}
	if env["OPENCODE_CONFIG"] != openCodeConfigPath(dir) {
		t.Errorf("OPENCODE_CONFIG=%q, want %q", env["OPENCODE_CONFIG"], openCodeConfigPath(dir))
	}
	cfg := readOpenCodeConfig(t, dir)
	mcp, ok := cfg["mcp"].(map[string]any)
	if !ok {
		t.Fatalf("mcp block missing: %v", cfg)
	}
	local, ok := mcp["opendray-memory"].(map[string]any)
	if !ok {
		t.Fatalf("opendray-memory entry missing")
	}
	if local["type"] != "local" {
		t.Errorf("type=%v, want local", local["type"])
	}
	cmd, _ := local["command"].([]any)
	if len(cmd) != 2 || cmd[0] != "/bin/opendray" || cmd[1] != "mcp-memory" {
		t.Errorf("command=%v, want [/bin/opendray mcp-memory]", cmd)
	}
	remote, ok := mcp["remote-thing"].(map[string]any)
	if !ok || remote["type"] != "remote" || remote["url"] != "https://example.com/mcp" {
		t.Errorf("remote entry wrong: %v", remote)
	}
}

func TestEnsureOpenCodeInstructionsDedup(t *testing.T) {
	dir := t.TempDir()
	out := &session.PrepareOutput{Env: map[string]string{}}
	for i := 0; i < 3; i++ {
		if err := ensureOpenCodeInstructions(dir, out); err != nil {
			t.Fatalf("ensureOpenCodeInstructions: %v", err)
		}
	}
	cfg := readOpenCodeConfig(t, dir)
	list, _ := cfg["instructions"].([]any)
	if len(list) != 1 {
		t.Fatalf("instructions=%v, want exactly one entry", list)
	}
	if list[0] != openCodeAgentsPath(dir) {
		t.Errorf("instructions[0]=%v, want %s", list[0], openCodeAgentsPath(dir))
	}
}

// TestOpenCodeConfigMergesAllContributors verifies provider + MCP +
// instructions all land in one file without clobbering each other, mirroring
// the spawn-prep call order.
func TestOpenCodeConfigMergesAllContributors(t *testing.T) {
	dir := t.TempDir()
	out := &session.PrepareOutput{Env: map[string]string{}}
	if err := injectOpenCodeLocalProvider(dir, map[string]any{"localBaseUrl": "http://localhost:1234/v1", "localModel": "m"}, out); err != nil {
		t.Fatal(err)
	}
	if err := ensureOpenCodeInstructions(dir, out); err != nil {
		t.Fatal(err)
	}
	if _, _, err := renderOpenCodeMCP(dir, []MCPServer{{Name: "opendray-memory", Command: "/bin/opendray", Args: []string{"mcp-memory"}}}); err != nil {
		t.Fatal(err)
	}
	cfg := readOpenCodeConfig(t, dir)
	if _, ok := cfg["provider"].(map[string]any); !ok {
		t.Error("provider block lost after later merges")
	}
	if _, ok := cfg["mcp"].(map[string]any); !ok {
		t.Error("mcp block missing")
	}
	if list, _ := cfg["instructions"].([]any); len(list) != 1 {
		t.Errorf("instructions=%v, want one entry", list)
	}
	if cfg["$schema"] != "https://opencode.ai/config.json" {
		t.Errorf("schema=%v", cfg["$schema"])
	}
}

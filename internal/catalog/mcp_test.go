package catalog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestRenderClaudeMCP_StdioServer(t *testing.T) {
	dir := t.TempDir()
	servers := []MCPServer{
		{Name: "fs", Command: "npx", Args: []string{"-y", "@modelcontextprotocol/server-filesystem", "/tmp"}},
	}
	args, env, err := renderMCP("claude", dir, "", "", servers)
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
	args, _, err := renderMCP("claude", dir, "", "", servers)
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
	args, _, err := renderMCP("claude", dir, "", "", servers)
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
	args, env, err := renderMCP("codex", dir, "", "", servers)
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

	_, env, err := renderMCP("codex", t.TempDir(), "", "", []MCPServer{
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
	_, env, err := renderMCP("codex", dir, "", "", servers)
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

func readAgyMCPConfig(t *testing.T, home string) map[string]any {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(home, ".gemini", "config", "mcp_config.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	return got
}

func TestRenderAgyMCP_WritesGlobalConfig(t *testing.T) {
	home := t.TempDir()
	servers := []MCPServer{
		{Name: "fs", Command: "npx", Args: []string{"-y", "server-fs"}},
		{Name: "gw", Transport: "http", URL: "http://127.0.0.1:8770/mcp",
			Headers: map[string]string{"Authorization": "Bearer k"}},
	}
	args, env, err := renderMCP("antigravity", t.TempDir(), t.TempDir(), home, servers)
	if err != nil {
		t.Fatal(err)
	}
	if len(args) != 0 || len(env) != 0 {
		t.Errorf("args=%v env=%v, want none (agy reads <home>/.gemini/config/mcp_config.json)", args, env)
	}
	mcps := readAgyMCPConfig(t, home)["mcpServers"].(map[string]any)
	fs := mcps["fs"].(map[string]any)
	if fs["command"].(string) != "npx" {
		t.Errorf("command=%v", fs["command"])
	}
	// Remote entries must use agy's serverUrl field — it rejects the
	// url/httpUrl shapes other CLIs use.
	gw := mcps["gw"].(map[string]any)
	if gw["serverUrl"].(string) != "http://127.0.0.1:8770/mcp" {
		t.Errorf("serverUrl=%v", gw["serverUrl"])
	}
	if _, ok := gw["url"]; ok {
		t.Errorf("legacy url field leaked: %v", gw)
	}
	// Managed-entry ledger must exist so the next render can prune
	// stale entries without touching user-authored ones.
	if _, err := os.Stat(filepath.Join(home, ".gemini", "config", agyManagedFile)); err != nil {
		t.Errorf("managed ledger missing: %v", err)
	}
}

func TestRenderAgyMCP_MergePreservesUserConfig(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".gemini", "config")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	user := `{
  "mcpServers": {
    "user-server": {"command": "uvx", "args": ["my-mcp"]},
    "user-remote": {"serverUrl": "https://example.com/mcp"}
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "mcp_config.json"), []byte(user), 0o644); err != nil {
		t.Fatal(err)
	}

	_, _, err := renderMCP("antigravity", t.TempDir(), t.TempDir(), home, []MCPServer{
		{Name: "fs", Command: "npx"},
	})
	if err != nil {
		t.Fatal(err)
	}
	mcps := readAgyMCPConfig(t, home)["mcpServers"].(map[string]any)
	if _, ok := mcps["user-server"]; !ok {
		t.Errorf("user stdio entry lost: %v", mcps)
	}
	if _, ok := mcps["user-remote"]; !ok {
		t.Errorf("user remote entry lost: %v", mcps)
	}
	if _, ok := mcps["fs"]; !ok {
		t.Errorf("managed entry missing: %v", mcps)
	}
}

func TestRenderAgyMCP_RemovesStaleManagedEntries(t *testing.T) {
	home := t.TempDir()
	if _, _, err := renderMCP("antigravity", t.TempDir(), t.TempDir(), home, []MCPServer{
		{Name: "fs", Command: "npx"},
		{Name: "old", Command: "node"},
	}); err != nil {
		t.Fatal(err)
	}
	// Second spawn: "old" disabled / removed from the registry.
	if _, _, err := renderMCP("antigravity", t.TempDir(), t.TempDir(), home, []MCPServer{
		{Name: "fs", Command: "npx"},
	}); err != nil {
		t.Fatal(err)
	}
	mcps := readAgyMCPConfig(t, home)["mcpServers"].(map[string]any)
	if _, ok := mcps["old"]; ok {
		t.Errorf("stale managed entry survived: %v", mcps)
	}
	if _, ok := mcps["fs"]; !ok {
		t.Errorf("live managed entry missing: %v", mcps)
	}

	// Full cleanup (no MCP servers at all this spawn).
	if err := syncAgyGlobalMCP(home, nil); err != nil {
		t.Fatal(err)
	}
	mcps = readAgyMCPConfig(t, home)["mcpServers"].(map[string]any)
	if len(mcps) != 0 {
		t.Errorf("managed entries not cleaned up: %v", mcps)
	}
	if _, err := os.Stat(filepath.Join(home, ".gemini", "config", agyManagedFile)); !os.IsNotExist(err) {
		t.Errorf("managed ledger should be removed after cleanup, err=%v", err)
	}
}

func TestRenderAgyMCP_MalformedUserConfigErrors(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".gemini", "config")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "mcp_config.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := renderMCP("antigravity", t.TempDir(), t.TempDir(), home, []MCPServer{
		{Name: "fs", Command: "npx"},
	}); err == nil {
		t.Fatal("expected error on malformed user mcp_config.json (must not clobber)")
	}
}

// The agy file is per-HOME — every non-account-bound antigravity spawn
// converges the same file, so concurrent syncs are normal. Run with
// -race: the mutex + atomic rename must keep the file parseable and the
// user's entry intact under interleaving.
func TestSyncAgyGlobalMCP_ConcurrentSpawns(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".gemini", "config")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	user := `{"mcpServers":{"user-server":{"command":"uvx"}}}`
	if err := os.WriteFile(filepath.Join(dir, "mcp_config.json"), []byte(user), 0o644); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			desired := map[string]map[string]any{
				"opendray-memory": {"command": "/bin/opendray", "args": []any{"mcp-memory"}},
			}
			if i%2 == 0 {
				desired[fmt.Sprintf("extra-%d", i)] = map[string]any{"command": "npx"}
			}
			if err := syncAgyGlobalMCP(home, desired); err != nil {
				t.Errorf("sync %d: %v", i, err)
			}
		}(i)
	}
	wg.Wait()

	mcps := readAgyMCPConfig(t, home)["mcpServers"].(map[string]any)
	if _, ok := mcps["user-server"]; !ok {
		t.Errorf("user entry lost under concurrency: %v", mcps)
	}
	if _, ok := mcps["opendray-memory"]; !ok {
		t.Errorf("memory entry missing after concurrent syncs: %v", mcps)
	}
	// And a final converge must still work (file parseable, ledger sane).
	if err := syncAgyGlobalMCP(home, nil); err != nil {
		t.Fatalf("final cleanup: %v", err)
	}
	mcps = readAgyMCPConfig(t, home)["mcpServers"].(map[string]any)
	if _, ok := mcps["user-server"]; !ok {
		t.Errorf("user entry lost on cleanup: %v", mcps)
	}
	if len(mcps) != 1 {
		t.Errorf("managed entries should all be gone, got: %v", mcps)
	}
}

// Integration-declared MCP servers must never land in antigravity's
// shared HOME-global file: only an account-bound spawn (dedicated HOME)
// may carry them. Every other provider renders per-session files, so
// they're always allowed.
func TestIntegrationMCPAllowed(t *testing.T) {
	cases := []struct {
		provider     string
		accountBound bool
		want         bool
	}{
		{"claude", false, true},
		{"codex", false, true},
		{"opencode", false, true},
		{"grok", false, true},
		{"antigravity", false, false},
		{"antigravity", true, true},
	}
	for _, c := range cases {
		if got := integrationMCPAllowed(c.provider, c.accountBound); got != c.want {
			t.Errorf("integrationMCPAllowed(%q, %v) = %v, want %v",
				c.provider, c.accountBound, got, c.want)
		}
	}
}

// The legacy <cwd>/.gemini/settings.json surface (a previous fix wrote
// MCP entries there; agy never read it) must still be purgeable so agy
// spawns can clean up what that release left behind.
func TestSyncGeminiWorkspaceMCP_LegacyCleanup(t *testing.T) {
	cwd := t.TempDir()
	if err := syncGeminiWorkspaceMCP(cwd, map[string]map[string]any{
		"stale": {"command": "npx"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := syncGeminiWorkspaceMCP(cwd, nil); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(cwd, ".gemini", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if _, ok := got["mcpServers"]; ok {
		t.Errorf("legacy managed entries not cleaned up: %v", got)
	}
	if _, err := os.Stat(filepath.Join(cwd, ".gemini", geminiManagedFile)); !os.IsNotExist(err) {
		t.Errorf("legacy managed ledger should be removed, err=%v", err)
	}
}

func readGrokConfig(t *testing.T, cwd string) map[string]any {
	t.Helper()
	var got map[string]any
	if _, err := toml.DecodeFile(filepath.Join(cwd, ".grok", "config.toml"), &got); err != nil {
		t.Fatalf("decode grok config.toml: %v", err)
	}
	return got
}

func TestRenderGrokMCP_WritesProjectConfig(t *testing.T) {
	cwd := t.TempDir()
	servers := []MCPServer{
		{Name: "vault", Command: "vault-mcp", Args: []string{"--addr", "https://v"}, Env: map[string]string{"VAULT_TOKEN": "s.xxx"}},
	}
	args, env, err := renderMCP("grok", t.TempDir(), cwd, "", servers)
	if err != nil {
		t.Fatal(err)
	}
	if len(args) != 0 || len(env) != 0 {
		t.Errorf("args=%v env=%v, want none (grok reads <cwd>/.grok/config.toml)", args, env)
	}
	mcps, ok := readGrokConfig(t, cwd)["mcp_servers"].(map[string]any)
	if !ok {
		t.Fatalf("mcp_servers table missing")
	}
	vault, ok := mcps["vault"].(map[string]any)
	if !ok {
		t.Fatalf("vault server missing: %v", mcps)
	}
	if vault["command"].(string) != "vault-mcp" {
		t.Errorf("command=%v", vault["command"])
	}
	if vault["env"].(map[string]any)["VAULT_TOKEN"].(string) != "s.xxx" {
		t.Errorf("env not rendered: %v", vault["env"])
	}
	// Managed ledger + gitignore so stale entries can be pruned and the
	// credential-bearing config never lands in version control.
	if _, err := os.Stat(filepath.Join(cwd, ".grok", grokManagedFile)); err != nil {
		t.Errorf("managed ledger missing: %v", err)
	}
	gi, err := os.ReadFile(filepath.Join(cwd, ".grok", ".gitignore"))
	if err != nil {
		t.Fatalf("gitignore missing: %v", err)
	}
	if !strings.Contains(string(gi), "config.toml") {
		t.Errorf("gitignore does not cover config.toml: %s", gi)
	}
}

func TestRenderGrokMCP_RemoteTransports(t *testing.T) {
	cwd := t.TempDir()
	_, _, err := renderMCP("grok", t.TempDir(), cwd, "", []MCPServer{
		{Name: "sse_srv", Transport: "sse", URL: "http://h/sse", Headers: map[string]string{"X-Key": "v"}},
		{Name: "http_srv", Transport: "http", URL: "http://h/mcp"},
	})
	if err != nil {
		t.Fatal(err)
	}
	mcps := readGrokConfig(t, cwd)["mcp_servers"].(map[string]any)
	sse := mcps["sse_srv"].(map[string]any)
	if sse["url"].(string) != "http://h/sse" || sse["type"].(string) != "sse" {
		t.Errorf("sse render wrong: %v", sse)
	}
	if sse["headers"].(map[string]any)["X-Key"].(string) != "v" {
		t.Errorf("sse headers wrong: %v", sse)
	}
	httpSrv := mcps["http_srv"].(map[string]any)
	if httpSrv["url"].(string) != "http://h/mcp" {
		t.Errorf("http url wrong: %v", httpSrv)
	}
	// Streamable HTTP is grok's default for a url with no type.
	if _, hasType := httpSrv["type"]; hasType {
		t.Errorf("http server should omit type (streamable default): %v", httpSrv)
	}
}

func TestRenderGrokMCP_MergePreservesUserConfig(t *testing.T) {
	cwd := t.TempDir()
	dir := filepath.Join(cwd, ".grok")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	user := "[mcp_servers.user-server]\ncommand = \"uvx\"\nargs = [\"my-mcp\"]\n"
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(user), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := renderMCP("grok", t.TempDir(), cwd, "", []MCPServer{
		{Name: "vault", Command: "vault-mcp"},
	}); err != nil {
		t.Fatal(err)
	}
	mcps := readGrokConfig(t, cwd)["mcp_servers"].(map[string]any)
	if _, ok := mcps["user-server"]; !ok {
		t.Errorf("user mcp entry lost: %v", mcps)
	}
	if _, ok := mcps["vault"]; !ok {
		t.Errorf("managed entry missing: %v", mcps)
	}
}

func TestRenderGrokMCP_RemovesStaleManagedEntries(t *testing.T) {
	cwd := t.TempDir()
	if _, _, err := renderMCP("grok", t.TempDir(), cwd, "", []MCPServer{
		{Name: "vault", Command: "vault-mcp"},
		{Name: "old", Command: "node"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := renderMCP("grok", t.TempDir(), cwd, "", []MCPServer{
		{Name: "vault", Command: "vault-mcp"},
	}); err != nil {
		t.Fatal(err)
	}
	mcps := readGrokConfig(t, cwd)["mcp_servers"].(map[string]any)
	if _, ok := mcps["old"]; ok {
		t.Errorf("stale managed entry survived: %v", mcps)
	}
	if _, ok := mcps["vault"]; !ok {
		t.Errorf("live managed entry missing: %v", mcps)
	}

	// Full cleanup (MCP off this spawn).
	if err := syncGrokWorkspaceMCP(cwd, nil); err != nil {
		t.Fatal(err)
	}
	if _, ok := readGrokConfig(t, cwd)["mcp_servers"]; ok {
		t.Errorf("managed entries not cleaned up")
	}
	if _, err := os.Stat(filepath.Join(cwd, ".grok", grokManagedFile)); !os.IsNotExist(err) {
		t.Errorf("managed ledger should be removed after cleanup, err=%v", err)
	}
}

func TestRenderGrokMCP_MalformedUserConfigErrors(t *testing.T) {
	cwd := t.TempDir()
	dir := filepath.Join(cwd, ".grok")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("[mcp_servers.x\nbroken"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := renderMCP("grok", t.TempDir(), cwd, "", []MCPServer{
		{Name: "vault", Command: "vault-mcp"},
	}); err == nil {
		t.Fatal("expected error on malformed user config.toml (must not clobber)")
	}
}

func TestRenderMCP_UnknownProviderNoOp(t *testing.T) {
	args, env, err := renderMCP("unknown-provider", t.TempDir(), "", "", []MCPServer{
		{Name: "x", Command: "y"},
	})
	if err != nil || args != nil || env != nil {
		t.Errorf("expected no-op for unknown-provider, got args=%v env=%v err=%v", args, env, err)
	}
}

func TestRenderMCP_EmptyServers(t *testing.T) {
	args, env, err := renderMCP("claude", t.TempDir(), "", "", nil)
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

package catalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/opendray/opendray-v2/internal/session"
)

// Antigravity reaches opendray-memory as native MCP tools (its entry is
// merged into <home>/.gemini/config/mcp_config.json — see renderAgyMCP),
// so it receives the standard MCP memory guidance through its AGENTS.md
// surface. An earlier release drove agy through an `opendray memory` CLI
// shim; that path is gone.
func TestInjectMemoryGuidance_Antigravity(t *testing.T) {
	base := t.TempDir()
	out := &session.PrepareOutput{Env: map[string]string{}}
	if err := injectMemoryGuidanceFor("antigravity", base, out); err != nil {
		t.Fatal(err)
	}
	// agy guidance is appended to the --add-dir'd AGENTS.md.
	body, err := os.ReadFile(filepath.Join(base, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	got := string(body)
	// Must steer the agent at the native MCP tools.
	for _, want := range []string{
		"opendray-memory",
		"memory_store",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("agy memory guidance missing %q; got:\n%s", want, got)
		}
	}
	// The retired CLI-shim wording must not resurface — the agent HAS
	// the MCP tools now.
	if strings.Contains(got, "opendray memory call") {
		t.Errorf("agy guidance still references the retired CLI shim:\n%s", got)
	}
	if !hasArgPair(out.Args, "--add-dir", base) {
		t.Errorf("missing --add-dir %s; args=%v", base, out.Args)
	}
}

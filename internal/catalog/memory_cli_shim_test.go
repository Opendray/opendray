package catalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/opendray/opendray-v2/internal/session"
)

func TestMemoryViaCLI(t *testing.T) {
	// Only antigravity routes memory through the CLI shim today; everyone
	// else loads the opendray-memory MCP server (or has no MCP surface).
	cliBucket := map[string]bool{"antigravity": true}
	for _, id := range []string{"claude", "codex", "opencode", "antigravity", "grok", "shell"} {
		if got, want := memoryViaCLI(id), cliBucket[id]; got != want {
			t.Errorf("memoryViaCLI(%q) = %v, want %v", id, got, want)
		}
	}
}

func TestInjectMemoryCLIGuidance_Antigravity(t *testing.T) {
	base := t.TempDir()
	out := &session.PrepareOutput{Env: map[string]string{}}
	if err := injectMemoryCLIGuidanceFor("antigravity", base, "/abs/path/opendray", out); err != nil {
		t.Fatal(err)
	}
	// agy guidance is appended to the --add-dir'd AGENTS.md.
	body, err := os.ReadFile(filepath.Join(base, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	got := string(body)
	// Must steer the agent at the CLI, not phantom MCP tools.
	for _, want := range []string{
		"opendray memory call",
		"opendray memory tools",
		"session_log_append",
		"memory_load_context",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("agy CLI guidance missing %q; got:\n%s", want, got)
		}
	}
	// It must NOT tell agy to call the MCP server (the old phantom bug).
	if strings.Contains(got, "access to an MCP server named") {
		t.Errorf("agy CLI guidance must not reference the MCP server it can't load")
	}
	// The absolute-path fallback (PATH-independent) must be present.
	for _, want := range []string{"OPENDRAY_BIN", "/abs/path/opendray"} {
		if !strings.Contains(got, want) {
			t.Errorf("agy CLI guidance missing PATH fallback %q", want)
		}
	}
}

func TestInjectMemoryCLIGuidance_NonShimProviderNoop(t *testing.T) {
	// Providers not in the CLI-shim bucket get nothing from this path
	// (they receive the MCP guidance elsewhere).
	for _, id := range []string{"claude", "codex", "opencode"} {
		base := t.TempDir()
		out := &session.PrepareOutput{Env: map[string]string{}}
		if err := injectMemoryCLIGuidanceFor(id, base, "/abs/path/opendray", out); err != nil {
			t.Errorf("%s: should not error: %v", id, err)
		}
		if len(out.Args) != 0 {
			t.Errorf("%s: should not mutate args; got %+v", id, out.Args)
		}
	}
}

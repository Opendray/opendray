package catalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/opendray/opendray-v2/internal/session"
)

// The global instruction (opendray-wide system-prompt doc) must reach
// EVERY provider's system prompt, including grok — the one provider with
// no existing per-CLI injector. Grok appends extra system-prompt text via
// --rules, which it permits at most once per invocation, so the global
// instruction is delivered as a single --rules arg.

func TestInjectGlobalInstruction_Grok(t *testing.T) {
	out := &session.PrepareOutput{}
	if err := injectGlobalInstructionFor("grok", t.TempDir(), "GLOBAL_DOC", out); err != nil {
		t.Fatal(err)
	}
	if len(out.Args) != 2 || out.Args[0] != "--rules" || out.Args[1] != "GLOBAL_DOC" {
		t.Errorf("grok: want [--rules GLOBAL_DOC], got %+v", out.Args)
	}
}

func TestInjectGlobalInstruction_Claude(t *testing.T) {
	out := &session.PrepareOutput{}
	if err := injectGlobalInstructionFor("claude", t.TempDir(), "GLOBAL_DOC", out); err != nil {
		t.Fatal(err)
	}
	if len(out.Args) != 2 || out.Args[0] != "--append-system-prompt" || out.Args[1] != "GLOBAL_DOC" {
		t.Errorf("claude: want [--append-system-prompt GLOBAL_DOC], got %+v", out.Args)
	}
}

func TestInjectGlobalInstruction_Codex(t *testing.T) {
	base := t.TempDir()
	out := &session.PrepareOutput{Env: map[string]string{}}
	if err := injectGlobalInstructionFor("codex", base, "GLOBAL_DOC", out); err != nil {
		t.Fatal(err)
	}
	home := out.Env["CODEX_HOME"]
	if home == "" {
		t.Fatalf("codex: CODEX_HOME not set")
	}
	body, err := os.ReadFile(filepath.Join(home, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	if !strings.Contains(string(body), "GLOBAL_DOC") {
		t.Errorf("codex: AGENTS.md missing global doc; got: %s", body)
	}
}

func TestInjectGlobalInstruction_EmptyTextNoop(t *testing.T) {
	for _, prov := range []string{"claude", "codex", "grok", "opencode", "antigravity"} {
		out := &session.PrepareOutput{Env: map[string]string{}}
		if err := injectGlobalInstructionFor(prov, t.TempDir(), "", out); err != nil {
			t.Errorf("%s: empty text should not error: %v", prov, err)
		}
		if len(out.Args) != 0 {
			t.Errorf("%s: empty text should not mutate args; got %+v", prov, out.Args)
		}
	}
}

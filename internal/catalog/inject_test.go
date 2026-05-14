package catalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/opendray/opendray-v2/internal/session"
)

// M19 — verify the spawn-time injection paths work for each CLI.
// We exercise injectAmbientMemoryFor directly rather than spinning
// up a real session, since these are pure functions of (provider,
// baseDir, text) → file/arg mutations.

func TestInjectAmbientMemory_Claude(t *testing.T) {
	out := &session.PrepareOutput{}
	if err := injectAmbientMemoryFor("claude", t.TempDir(), "PROJECT_BANNER", out); err != nil {
		t.Fatal(err)
	}
	if len(out.Args) != 2 ||
		out.Args[0] != "--append-system-prompt" ||
		out.Args[1] != "PROJECT_BANNER" {
		t.Errorf("claude: wrong args: %+v", out.Args)
	}
}

func TestInjectAmbientMemory_Codex(t *testing.T) {
	base := t.TempDir()
	out := &session.PrepareOutput{Env: map[string]string{}}
	if err := injectAmbientMemoryFor("codex", base, "PROJECT_BANNER", out); err != nil {
		t.Fatal(err)
	}
	codexHome := out.Env["CODEX_HOME"]
	if codexHome == "" {
		t.Fatalf("codex: CODEX_HOME not set")
	}
	if filepath.Dir(codexHome) != base {
		t.Errorf("codex: CODEX_HOME should be under baseDir; got %s vs base %s", codexHome, base)
	}
	agents := filepath.Join(codexHome, "AGENTS.md")
	body, err := os.ReadFile(agents)
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	if !strings.Contains(string(body), "PROJECT_BANNER") {
		t.Errorf("codex: AGENTS.md missing banner; got: %s", body)
	}
}

func TestInjectAmbientMemory_Gemini(t *testing.T) {
	base := t.TempDir()
	out := &session.PrepareOutput{}
	if err := injectAmbientMemoryFor("gemini", base, "PROJECT_BANNER", out); err != nil {
		t.Fatal(err)
	}
	geminiMd := filepath.Join(base, "GEMINI.md")
	body, err := os.ReadFile(geminiMd)
	if err != nil {
		t.Fatalf("read GEMINI.md: %v", err)
	}
	if !strings.Contains(string(body), "PROJECT_BANNER") {
		t.Errorf("gemini: GEMINI.md missing banner; got: %s", body)
	}
	// Should also add --include-directories=baseDir to args so
	// gemini picks up GEMINI.md as workspace memory.
	found := false
	for i := 0; i+1 < len(out.Args); i++ {
		if out.Args[i] == "--include-directories" && out.Args[i+1] == base {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("gemini: --include-directories <baseDir> arg missing; got %+v", out.Args)
	}
}

func TestInjectAmbientMemory_GeminiIdempotent(t *testing.T) {
	// If the caller (skills injection) already added
	// --include-directories baseDir, we should NOT add a duplicate.
	base := t.TempDir()
	out := &session.PrepareOutput{
		Args: []string{"--include-directories", base},
	}
	if err := injectAmbientMemoryFor("gemini", base, "BANNER", out); err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, a := range out.Args {
		if a == "--include-directories" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly one --include-directories flag; got %d. args: %+v", count, out.Args)
	}
}

func TestInjectAmbientMemory_EmptyTextNoop(t *testing.T) {
	for _, prov := range []string{"claude", "codex", "gemini", "shell"} {
		out := &session.PrepareOutput{Env: map[string]string{}}
		if err := injectAmbientMemoryFor(prov, t.TempDir(), "", out); err != nil {
			t.Errorf("%s: empty text should not error: %v", prov, err)
		}
		if len(out.Args) != 0 {
			t.Errorf("%s: empty text should not mutate args; got %+v", prov, out.Args)
		}
	}
}

func TestInjectAmbientMemory_UnknownProviderSilent(t *testing.T) {
	out := &session.PrepareOutput{}
	if err := injectAmbientMemoryFor("nonexistent", t.TempDir(), "X", out); err != nil {
		t.Errorf("unknown provider should not error: %v", err)
	}
}

// M21 — injectSessionIDFor pre-assigns the agent-side session UUID so
// the M18 transcript reader hits the correct *.jsonl directly. Bug
// caught in production: without this, sessions.claude_session_id is
// empty, the reader falls back to "latest mtime in dir", and picks up
// unrelated active conversations.

func TestInjectSessionID_Claude(t *testing.T) {
	out := &session.PrepareOutput{}
	if !injectSessionIDFor("claude", out) {
		t.Fatal("expected injection to fire for claude")
	}
	if out.ClaudeSessionID == "" {
		t.Errorf("claude: ClaudeSessionID empty after inject")
	}
	if len(out.Args) != 2 || out.Args[0] != "--session-id" || out.Args[1] != out.ClaudeSessionID {
		t.Errorf("claude: expected --session-id <id> arg pair, got %+v", out.Args)
	}
}

func TestInjectSessionID_Gemini(t *testing.T) {
	out := &session.PrepareOutput{}
	if !injectSessionIDFor("gemini", out) {
		t.Fatal("expected injection to fire for gemini")
	}
	if out.ClaudeSessionID == "" {
		t.Errorf("gemini: ClaudeSessionID empty after inject")
	}
	if len(out.Args) != 2 || out.Args[0] != "--session-id" || out.Args[1] != out.ClaudeSessionID {
		t.Errorf("gemini: expected --session-id <id> arg pair, got %+v", out.Args)
	}
}

func TestInjectSessionID_CodexSkipped(t *testing.T) {
	// Codex has no --session-id flag — must skip rather than emit a
	// bogus arg that codex would reject.
	out := &session.PrepareOutput{}
	if injectSessionIDFor("codex", out) {
		t.Errorf("codex: injection should not fire (no --session-id support)")
	}
	if len(out.Args) != 0 || out.ClaudeSessionID != "" {
		t.Errorf("codex: out should be untouched, got args=%v id=%q", out.Args, out.ClaudeSessionID)
	}
}

func TestInjectSessionID_FreshUUIDsAcrossSpawns(t *testing.T) {
	// Every spawn must get its own UUID; otherwise two concurrent
	// Claude sessions would race for the same *.jsonl.
	out1 := &session.PrepareOutput{}
	out2 := &session.PrepareOutput{}
	injectSessionIDFor("claude", out1)
	injectSessionIDFor("claude", out2)
	if out1.ClaudeSessionID == out2.ClaudeSessionID {
		t.Errorf("expected distinct UUIDs across spawns, got %q twice", out1.ClaudeSessionID)
	}
}

func TestEnsureCodexScratchTrust_AppendsCurrentCwd(t *testing.T) {
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(`model = "gpt-5.4"
`), 0o600); err != nil {
		t.Fatal(err)
	}

	cwd := "/Users/test/work with spaces"
	if err := ensureCodexScratchTrust(home, cwd); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(home, "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	str := string(body)
	if !strings.Contains(str, `model = "gpt-5.4"`) {
		t.Errorf("base config missing: %s", str)
	}
	if !strings.Contains(str, `[projects."/Users/test/work with spaces"]`) {
		t.Errorf("project trust header missing: %s", str)
	}
	if !strings.Contains(str, `trust_level = "trusted"`) {
		t.Errorf("trust level missing: %s", str)
	}
}

package catalog

import (
	"context"
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
	if !injectSessionIDFor(context.Background(), "claude", out) {
		t.Fatal("expected injection to fire for claude")
	}
	if out.ClaudeSessionID == "" {
		t.Errorf("claude: ClaudeSessionID empty after inject")
	}
	if len(out.Args) != 2 || out.Args[0] != "--session-id" || out.Args[1] != out.ClaudeSessionID {
		t.Errorf("claude: expected --session-id <id> arg pair, got %+v", out.Args)
	}
}

func TestInjectSessionID_ClaudeResume(t *testing.T) {
	// On reactivation the manager threads the existing agent UUID via
	// ctx; claude must continue it with --resume, NOT mint a fresh
	// --session-id (which would start a blank conversation and orphan
	// the original transcript). Regression for the 2026-05-23 incident.
	const prior = "d3cca926-5e55-4d6b-8920-392e990347a1"
	out := &session.PrepareOutput{}
	ctx := session.WithResumeClaudeSessionID(context.Background(), prior)
	if !injectSessionIDFor(ctx, "claude", out) {
		t.Fatal("expected injection to fire for claude resume")
	}
	if out.ClaudeSessionID != prior {
		t.Errorf("resume: ClaudeSessionID = %q, want preserved %q", out.ClaudeSessionID, prior)
	}
	if len(out.Args) != 2 || out.Args[0] != "--resume" || out.Args[1] != prior {
		t.Errorf("resume: expected --resume %s, got %+v", prior, out.Args)
	}
}

func TestInjectSessionID_Gemini(t *testing.T) {
	out := &session.PrepareOutput{}
	if !injectSessionIDFor(context.Background(), "gemini", out) {
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
	if injectSessionIDFor(context.Background(), "codex", out) {
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
	injectSessionIDFor(context.Background(), "claude", out1)
	injectSessionIDFor(context.Background(), "claude", out2)
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

func TestMirrorCodexHome_CopiesMinimalAuthSubset(t *testing.T) {
	src := t.TempDir()
	dest := t.TempDir()

	if err := os.WriteFile(filepath.Join(src, "auth.json"), []byte(`{"token":"x"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "state_5.sqlite"), []byte("sqlite"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(src, "rules"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "rules", "team.md"), []byte("rule"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(src, "plugins"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "plugins", "marketplace.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := mirrorCodexHome(src, dest); err != nil {
		t.Fatal(err)
	}

	authPath := filepath.Join(dest, "auth.json")
	info, err := os.Lstat(authPath)
	if err != nil {
		t.Fatalf("auth.json missing: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("auth.json should be copied, not symlinked")
	}
	body, err := os.ReadFile(authPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"token":"x"}` {
		t.Fatalf("auth.json content mismatch: %s", body)
	}

	if _, err := os.Stat(filepath.Join(dest, "rules", "team.md")); err != nil {
		t.Fatalf("rules directory should be copied: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "plugins", "marketplace.json")); err != nil {
		t.Fatalf("plugins directory should be copied: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "state_5.sqlite")); !os.IsNotExist(err) {
		t.Fatalf("sqlite runtime should not be mirrored, got err=%v", err)
	}
}

func TestMirrorCodexHome_SkipsDanglingSymlinks(t *testing.T) {
	src := t.TempDir()
	dest := t.TempDir()

	if err := os.WriteFile(filepath.Join(src, "auth.json"), []byte(`{"token":"x"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	skillsDir := filepath.Join(src, "skills")
	if err := os.MkdirAll(skillsDir, 0o700); err != nil {
		t.Fatal(err)
	}
	// Real skill alongside a dangling symlink — mirrors the user's actual
	// ~/.codex/skills layout when a skill source repo has been deleted.
	if err := os.MkdirAll(filepath.Join(skillsDir, "good"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillsDir, "good", "SKILL.md"), []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(src, "nonexistent-target"), filepath.Join(skillsDir, "broken")); err != nil {
		t.Fatal(err)
	}

	if err := mirrorCodexHome(src, dest); err != nil {
		t.Fatalf("mirror should tolerate dangling symlinks: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dest, "skills", "good", "SKILL.md")); err != nil {
		t.Fatalf("valid skill should be copied: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(dest, "skills", "broken")); !os.IsNotExist(err) {
		t.Fatalf("dangling symlink should be skipped, got err=%v", err)
	}
}

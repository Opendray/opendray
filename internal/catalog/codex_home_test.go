package catalog

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/opendray/opendray-v2/internal/session"
)

const codexThread = "019deded-aaaa-7bbb-8ccc-0123456789ab"

func writeCodexHome(t *testing.T, home, auth string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(home, "rules"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "auth.json"), []byte(auth), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "rules", "team.md"), []byte("rule"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func stageCodexRollout(t *testing.T, home, id string) {
	t.Helper()
	dir := filepath.Join(home, "sessions", "2026", "10", "07")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "rollout-2026-10-07T20-21-55-"+id+".jsonl"), []byte("x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestFinalizeCodexHome_AccountBoundForcesScratchWithAccountLogin(t *testing.T) {
	base, dflt, acct := t.TempDir(), t.TempDir(), t.TempDir()
	writeCodexHome(t, dflt, `{"who":"default"}`)
	writeCodexHome(t, acct, `{"who":"work"}`)
	out := session.PrepareOutput{Env: map[string]string{}}

	if err := finalizeCodexHome(base, acct, dflt, "", true, &out); err != nil {
		t.Fatal(err)
	}
	scratch := out.Env["CODEX_HOME"]
	if scratch != filepath.Join(base, "codex-home") {
		t.Fatalf("CODEX_HOME = %q, want scratch under base", scratch)
	}
	body, _ := os.ReadFile(filepath.Join(scratch, "auth.json"))
	if string(body) != `{"who":"work"}` {
		t.Errorf("scratch must authenticate as the bound account, got %s", body)
	}
	if _, err := os.Stat(filepath.Join(scratch, "rules", "team.md")); err != nil {
		t.Errorf("shared config should be mirrored from the default home: %v", err)
	}
	if out.LeadingArgs != nil {
		t.Errorf("fresh spawn must not resume, got %v", out.LeadingArgs)
	}
	if out.OnExit == nil {
		t.Error("scratch spawn must register the exit sync hook")
	}
}

func TestFinalizeCodexHome_ResumesSeededThread(t *testing.T) {
	base, dflt, acct := t.TempDir(), t.TempDir(), t.TempDir()
	writeCodexHome(t, dflt, `{}`)
	writeCodexHome(t, acct, `{}`)
	stageCodexRollout(t, acct, codexThread)
	out := session.PrepareOutput{Env: map[string]string{}}

	if err := finalizeCodexHome(base, acct, dflt, codexThread, true, &out); err != nil {
		t.Fatal(err)
	}
	if want := []string{"resume", codexThread}; !reflect.DeepEqual(out.LeadingArgs, want) {
		t.Errorf("LeadingArgs = %v, want %v", out.LeadingArgs, want)
	}
	if session.FindCodexRollout(out.Env["CODEX_HOME"], codexThread) == "" {
		t.Error("resumed rollout must be seeded into the scratch home")
	}
}

func TestFinalizeCodexHome_MissingRolloutStartsFresh(t *testing.T) {
	base, dflt, acct := t.TempDir(), t.TempDir(), t.TempDir()
	writeCodexHome(t, dflt, `{}`)
	writeCodexHome(t, acct, `{}`)
	out := session.PrepareOutput{Env: map[string]string{}}

	if err := finalizeCodexHome(base, acct, dflt, codexThread, true, &out); err != nil {
		t.Fatal(err)
	}
	if out.LeadingArgs != nil {
		t.Errorf("no rollout → must start fresh, got %v", out.LeadingArgs)
	}
}

func TestFinalizeCodexHome_UnboundDirectHomeOnlyResumes(t *testing.T) {
	dflt := t.TempDir()
	writeCodexHome(t, dflt, `{}`)
	stageCodexRollout(t, dflt, codexThread)
	out := session.PrepareOutput{Env: map[string]string{}}

	if err := finalizeCodexHome(t.TempDir(), dflt, dflt, codexThread, false, &out); err != nil {
		t.Fatal(err)
	}
	if out.Env["CODEX_HOME"] != "" {
		t.Errorf("unbound, nothing injected → codex runs on its own home; got CODEX_HOME=%q", out.Env["CODEX_HOME"])
	}
	if want := []string{"resume", codexThread}; !reflect.DeepEqual(out.LeadingArgs, want) {
		t.Errorf("LeadingArgs = %v, want %v", out.LeadingArgs, want)
	}
	if out.OnExit != nil {
		t.Error("direct home needs no exit sync")
	}
}

func TestFinalizeCodexHome_ExitHookSyncsBack(t *testing.T) {
	base, dflt, acct := t.TempDir(), t.TempDir(), t.TempDir()
	writeCodexHome(t, dflt, `{}`)
	writeCodexHome(t, acct, `{}`)
	out := session.PrepareOutput{Env: map[string]string{}}
	if err := finalizeCodexHome(base, acct, dflt, "", true, &out); err != nil {
		t.Fatal(err)
	}
	// codex writes a rollout into the scratch home during the run…
	stageCodexRollout(t, out.Env["CODEX_HOME"], codexThread)
	// …and the exit hook moves it into the account home + reports it.
	if got := out.OnExit(); got != codexThread {
		t.Errorf("OnExit thread = %q, want %q", got, codexThread)
	}
	if session.FindCodexRollout(acct, codexThread) == "" {
		t.Error("rollout should be durable in the account home after exit")
	}
}

func TestFinalizeCodexHome_RejectsSymlinkedAccountLogin(t *testing.T) {
	base, dflt, acct := t.TempDir(), t.TempDir(), t.TempDir()
	writeCodexHome(t, dflt, `{}`)
	secret := filepath.Join(t.TempDir(), "secret")
	_ = os.WriteFile(secret, []byte("SECRET"), 0o600)
	if err := os.Symlink(secret, filepath.Join(acct, "auth.json")); err != nil {
		t.Skip(err.Error())
	}
	out := session.PrepareOutput{Env: map[string]string{}}
	if err := finalizeCodexHome(base, acct, dflt, "", true, &out); err == nil {
		t.Fatal("a symlinked account auth.json must be refused")
	}
}

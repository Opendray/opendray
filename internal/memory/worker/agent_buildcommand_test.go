package worker

import (
	"testing"
)

func argsContain(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}

func TestBuildCommand_GeminiUsesPromptNotPrint(t *testing.T) {
	w := &AgentWorker{cfg: Config{ProviderID: "gemini"}}
	args, _, err := w.buildCommand(Request{UserInput: "hello world"}, "sid-1", t.TempDir())
	if err != nil {
		t.Fatalf("buildCommand: %v", err)
	}
	if argsContain(args, "--print") {
		t.Errorf("gemini must NOT use --print (Claude-only flag); got %v", args)
	}
	if !argsContain(args, "--prompt") {
		t.Errorf("gemini must use --prompt for headless mode; got %v", args)
	}
	if !argsContain(args, "hello world") {
		t.Errorf("gemini --prompt must carry the user input; got %v", args)
	}
}

func TestBuildCommand_ClaudeUsesPrint(t *testing.T) {
	w := &AgentWorker{cfg: Config{ProviderID: "claude"}}
	args, _, err := w.buildCommand(Request{UserInput: "x"}, "sid", t.TempDir())
	if err != nil {
		t.Fatalf("buildCommand: %v", err)
	}
	if !argsContain(args, "--print") {
		t.Errorf("claude uses --print; got %v", args)
	}
}

func TestBuildCommand_CodexUsesExecNotPrint(t *testing.T) {
	w := &AgentWorker{cfg: Config{ProviderID: "codex"}}
	args, _, err := w.buildCommand(Request{UserInput: "x"}, "sid", t.TempDir())
	if err != nil {
		t.Fatalf("buildCommand: %v", err)
	}
	if len(args) == 0 || args[0] != "exec" {
		t.Errorf("codex must use the `exec` subcommand; got %v", args)
	}
	if argsContain(args, "--print") {
		t.Errorf("codex must NOT use --print; got %v", args)
	}
}

func TestBuildCommand_OpencodeUnsupported(t *testing.T) {
	w := &AgentWorker{cfg: Config{ProviderID: "opencode"}}
	if _, _, err := w.buildCommand(Request{UserInput: "x"}, "sid", t.TempDir()); err == nil {
		t.Error("opencode has no headless worker path; buildCommand must return ErrAgentUnsupported")
	}
}

package worker

import (
	"strings"
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

func TestBuildCommand_AntigravityUsesPrint(t *testing.T) {
	w := &AgentWorker{cfg: Config{ProviderID: "antigravity"}}
	args, _, err := w.buildCommand(Request{UserInput: "hello world"}, "sid-1", t.TempDir())
	if err != nil {
		t.Fatalf("buildCommand: %v", err)
	}
	if !argsContain(args, "--print") {
		t.Errorf("antigravity must use --print for headless mode; got %v", args)
	}
	if argsContain(args, "--prompt") {
		t.Errorf("antigravity must NOT use --prompt; got %v", args)
	}
	// agy takes the prompt as the VALUE of --print (NOT stdin): the flag
	// must be immediately followed by the user input, and
	// --dangerously-skip-permissions must come FIRST so it isn't swallowed
	// as the prompt.
	if len(args) == 0 || args[0] != "--dangerously-skip-permissions" {
		t.Errorf("antigravity must pass --dangerously-skip-permissions first; got %v", args)
	}
	var printValue string
	for i, a := range args {
		if a == "--print" && i+1 < len(args) {
			printValue = args[i+1]
		}
	}
	if printValue != "hello world" {
		t.Errorf("antigravity must carry the prompt as the --print value; got %q in %v", printValue, args)
	}
}

func TestBuildCommand_AntigravityFoldsSystemAndSchema(t *testing.T) {
	w := &AgentWorker{cfg: Config{ProviderID: "antigravity"}}
	args, _, err := w.buildCommand(Request{
		SystemPrompt:             "you are a judge",
		UserInput:                "rate this",
		ResponseFormatJSONSchema: `{"type":"object"}`,
	}, "sid", t.TempDir())
	if err != nil {
		t.Fatalf("buildCommand: %v", err)
	}
	var printValue string
	for i, a := range args {
		if a == "--print" && i+1 < len(args) {
			printValue = args[i+1]
		}
	}
	for _, want := range []string{"you are a judge", "rate this", "JSON object"} {
		if !strings.Contains(printValue, want) {
			t.Errorf("antigravity --print value must fold %q; got %q", want, printValue)
		}
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

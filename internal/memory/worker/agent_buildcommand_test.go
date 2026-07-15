package worker

import (
	"context"
	"strings"
	"testing"
)

// fakeAgyAccounts is a stub AgyAccountReader that maps an account id to a HOME.
type fakeAgyAccounts struct{ home string }

func (f fakeAgyAccounts) ResolveSpawnHome(_ context.Context, _ string) (string, error) {
	return f.home, nil
}

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

func TestBuildCommand_AntigravityAccountSetsHome(t *testing.T) {
	// With an account pinned + a reader, agy binds to the account's HOME.
	w := &AgentWorker{
		cfg:         Config{ProviderID: "antigravity", AccountID: "agy1"},
		agyAccounts: fakeAgyAccounts{home: "/tmp/agy-home-1"},
	}
	_, env, err := w.buildCommand(Request{UserInput: "hi"}, "sid", t.TempDir())
	if err != nil {
		t.Fatalf("buildCommand: %v", err)
	}
	if !argsContain(env, "HOME=/tmp/agy-home-1") {
		t.Errorf("antigravity account must bind HOME; got env %v", env)
	}

	// No account → no HOME override (host-global login).
	w2 := &AgentWorker{cfg: Config{ProviderID: "antigravity"}}
	_, env2, err := w2.buildCommand(Request{UserInput: "hi"}, "sid", t.TempDir())
	if err != nil {
		t.Fatalf("buildCommand: %v", err)
	}
	for _, e := range env2 {
		if strings.HasPrefix(e, "HOME=") {
			t.Errorf("no account → no HOME override; got %q", e)
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

func TestBuildCommand_OpencodeUsesRun(t *testing.T) {
	w := &AgentWorker{cfg: Config{ProviderID: "opencode", Model: "anthropic/claude-sonnet-4-6"}}
	args, _, err := w.buildCommand(Request{SystemPrompt: "be terse", UserInput: "hello world"}, "sid", t.TempDir())
	if err != nil {
		t.Fatalf("buildCommand: %v", err)
	}
	if len(args) == 0 || args[0] != "run" {
		t.Errorf("opencode must use the `run` subcommand first; got %v", args)
	}
	if argsContain(args, "--print") || argsContain(args, "--prompt") {
		t.Errorf("opencode must NOT use --print/--prompt; got %v", args)
	}
	// The prompt is the positional message right after `run`, with the system
	// block folded in.
	if len(args) < 2 || !strings.Contains(args[1], "hello world") || !strings.Contains(args[1], "be terse") {
		t.Errorf("opencode must carry the folded prompt as the `run` message; got %v", args)
	}
	if !argsContain(args, "--model") || !argsContain(args, "anthropic/claude-sonnet-4-6") {
		t.Errorf("opencode must pass the pinned --model; got %v", args)
	}
}

func TestBuildCommand_GrokUsesSinglePrompt(t *testing.T) {
	w := &AgentWorker{cfg: Config{ProviderID: "grok", Model: "grok-build"}}
	args, _, err := w.buildCommand(Request{
		SystemPrompt:             "you are a judge",
		UserInput:                "rate this",
		ResponseFormatJSONSchema: `{"type":"object"}`,
	}, "sid", t.TempDir())
	if err != nil {
		t.Fatalf("buildCommand: %v", err)
	}
	// grok's headless flag is `-p <PROMPT>` with the prompt as the flag value.
	if !argsContain(args, "-p") {
		t.Errorf("grok must use -p for headless mode; got %v", args)
	}
	var pValue string
	for i, a := range args {
		if a == "-p" && i+1 < len(args) {
			pValue = args[i+1]
		}
	}
	// No system-prompt flag on grok, so system + schema fold into the value.
	for _, want := range []string{"you are a judge", "rate this", "JSON object"} {
		if !strings.Contains(pValue, want) {
			t.Errorf("grok -p value must fold %q; got %q", want, pValue)
		}
	}
	if !argsContain(args, "--output-format") || !argsContain(args, "plain") {
		t.Errorf("grok must request --output-format plain; got %v", args)
	}
	if !argsContain(args, "-m") || !argsContain(args, "grok-build") {
		t.Errorf("grok must pass the pinned -m model; got %v", args)
	}
}

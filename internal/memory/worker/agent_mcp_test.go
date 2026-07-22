package worker

import (
	"reflect"
	"testing"
)

// Each CLI enables headless MCP tool execution with a different flag; claude
// uses an allow-list (also its file-safety boundary), grok/opencode need a
// blanket approve, and antigravity/codex need nothing extra here.
func TestMCPToolApprovalArgs(t *testing.T) {
	cases := map[string][]string{
		"claude":      {"--allowedTools", "mcp__opendray-memory"},
		"grok":        {"--always-approve"},
		"opencode":    {"--dangerously-skip-permissions"},
		"antigravity": nil,
		"codex":       nil,
	}
	for provider, want := range cases {
		if got := mcpToolApprovalArgs(provider); !reflect.DeepEqual(got, want) {
			t.Errorf("mcpToolApprovalArgs(%q) = %v, want %v", provider, got, want)
		}
	}
}

// The claude allow-list must name the memory server exactly so file/bash tools
// stay unlisted (auto-denied in --print) — this is the read-only boundary for
// claude members.
func TestMCPToolApprovalArgs_ClaudeAllowlistIsScoped(t *testing.T) {
	got := mcpToolApprovalArgs("claude")
	if len(got) != 2 || got[0] != "--allowedTools" || got[1] != "mcp__opendray-memory" {
		t.Fatalf("claude allow-list drifted: %v", got)
	}
}

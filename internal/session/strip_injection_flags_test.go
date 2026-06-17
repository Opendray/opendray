package session

import (
	"reflect"
	"testing"
)

func TestStripInjectionFlags(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{"nil", nil, []string{}},
		{"no injection flags kept verbatim", []string{"--model", "opus", "--foo=bar"}, []string{"--model", "opus", "--foo=bar"}},
		{
			"claude --mcp-config <value> form drops flag + value",
			[]string{"--mcp-config", "/tmp/x.json", "--model", "opus"},
			[]string{"--model", "opus"},
		},
		{
			"claude --append-system-prompt=... form drops the single token",
			[]string{"--append-system-prompt=be brief", "--model", "opus"},
			[]string{"--model", "opus"},
		},
		{
			"standalone bypass flags dropped (claude/gemini/codex)",
			[]string{"--dangerously-skip-permissions", "--yolo", "--dangerously-bypass-approvals-and-sandbox", "keep"},
			[]string{"keep"},
		},
		{
			"value flag immediately followed by another flag does not eat the flag",
			[]string{"--mcp-config", "--model", "opus"},
			[]string{"--model", "opus"},
		},
		{
			"mixed: strips injection, keeps the rest",
			[]string{"--mcp-config", "/c.json", "--resume", "abc", "--dangerously-skip-permissions", "--append-system-prompt", "hi"},
			[]string{"--resume", "abc"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripInjectionFlags(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("stripInjectionFlags(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

package catalog

import (
	"reflect"
	"testing"
)

func TestStripFlagsWithValues(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		flags []string
		want  []string
	}{
		{
			name:  "codex bypass: strip approval+sandbox space-value forms",
			args:  []string{"--ask-for-approval", "never", "--model", "gpt-5-codex", "-s", "workspace-write"},
			flags: []string{"--ask-for-approval", "-a", "--sandbox", "-s"},
			want:  []string{"--model", "gpt-5-codex"},
		},
		{
			name:  "strip --flag=value single-token form",
			args:  []string{"--ask-for-approval=never", "--model", "gpt-5-codex"},
			flags: []string{"--ask-for-approval", "-a", "--sandbox", "-s"},
			want:  []string{"--model", "gpt-5-codex"},
		},
		{
			name:  "strip short-form -a with value",
			args:  []string{"-a", "never", "--model", "x"},
			flags: []string{"--ask-for-approval", "-a", "--sandbox", "-s"},
			want:  []string{"--model", "x"},
		},
		{
			name:  "flag at end with no following value",
			args:  []string{"--model", "x", "--sandbox"},
			flags: []string{"--sandbox", "-s"},
			want:  []string{"--model", "x"},
		},
		{
			name:  "value-less flag followed by another flag keeps that flag",
			args:  []string{"--sandbox", "--model", "x"},
			flags: []string{"--sandbox"},
			want:  []string{"--model", "x"},
		},
		{
			name:  "no matching flags is a passthrough",
			args:  []string{"--model", "x", "--dangerously-bypass-approvals-and-sandbox"},
			flags: []string{"--ask-for-approval", "-a", "--sandbox", "-s"},
			want:  []string{"--model", "x", "--dangerously-bypass-approvals-and-sandbox"},
		},
		{
			name:  "empty args",
			args:  nil,
			flags: []string{"--sandbox"},
			want:  []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripFlagsWithValues(tt.args, tt.flags...)
			// Normalise nil vs empty for comparison.
			if len(got) == 0 && len(tt.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("stripFlagsWithValues() = %v, want %v", got, tt.want)
			}
		})
	}
}

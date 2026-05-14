package session

import (
	"reflect"
	"testing"
)

func TestDropOverriddenFlags(t *testing.T) {
	tests := []struct {
		name     string
		provider []string
		user     []string
		want     []string
	}{
		{
			name:     "no user args returns provider untouched",
			provider: []string{"--ask-for-approval", "on-request", "-s", "workspace-write"},
			user:     nil,
			want:     []string{"--ask-for-approval", "on-request", "-s", "workspace-write"},
		},
		{
			name:     "user value-flag drops provider value-flag with value",
			provider: []string{"--ask-for-approval", "on-request", "-s", "workspace-write"},
			user:     []string{"--ask-for-approval", "never"},
			want:     []string{"-s", "workspace-write"},
		},
		{
			name:     "user bool-flag drops provider bool-flag (no value follows)",
			provider: []string{"--verbose", "--debug"},
			user:     []string{"--verbose"},
			want:     []string{"--debug"},
		},
		{
			name:     "user --key=value form is recognized by name",
			provider: []string{"--model", "gpt-4", "-s", "read-only"},
			user:     []string{"--model=gpt-5"},
			want:     []string{"-s", "read-only"},
		},
		{
			name:     "provider --key=value form is dropped wholesale",
			provider: []string{"--model=gpt-4", "-s", "read-only"},
			user:     []string{"--model", "gpt-5"},
			want:     []string{"-s", "read-only"},
		},
		{
			name:     "short flag with value (-s workspace-write) is dropped properly",
			provider: []string{"-s", "workspace-write", "--ask-for-approval", "on-request"},
			user:     []string{"-s", "danger-full-access"},
			want:     []string{"--ask-for-approval", "on-request"},
		},
		{
			name:     "non-overridden flags survive",
			provider: []string{"--ask-for-approval", "on-request", "-s", "workspace-write"},
			user:     []string{"-c", "model=gpt-5"},
			want:     []string{"--ask-for-approval", "on-request", "-s", "workspace-write"},
		},
		{
			name:     "user provides bare flag for provider value-flag drops both via peek",
			provider: []string{"--ask-for-approval", "on-request"},
			user:     []string{"--ask-for-approval"},
			want:     []string{},
		},
		{
			name:     "positional values in provider args are kept",
			provider: []string{"--ask-for-approval", "on-request", "prompt-here"},
			user:     []string{"--ask-for-approval", "never"},
			want:     []string{"prompt-here"},
		},
		{
			name:     "regression: codex bypass scenario from SpawnDialog",
			provider: []string{"--ask-for-approval", "on-request", "-s", "workspace-write"},
			user:     []string{"--ask-for-approval", "never", "-c", `approval_policy="never"`},
			want:     []string{"-s", "workspace-write"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := dropOverriddenFlags(tc.provider, tc.user)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("dropOverriddenFlags(%v, %v) = %v, want %v",
					tc.provider, tc.user, got, tc.want)
			}
		})
	}
}

func TestDropConflictingFlags(t *testing.T) {
	codexConflicts := map[string][]string{
		"--dangerously-bypass-approvals-and-sandbox": {
			"--ask-for-approval", "-a",
			"--sandbox", "-s",
		},
	}

	tests := []struct {
		name      string
		provider  []string
		user      []string
		conflicts map[string][]string
		want      []string
	}{
		{
			name:      "codex bypass strips approval and sandbox",
			provider:  []string{"--ask-for-approval", "on-request", "-s", "workspace-write"},
			user:      []string{"--dangerously-bypass-approvals-and-sandbox"},
			conflicts: codexConflicts,
			want:      []string{},
		},
		{
			name:      "codex bypass strips short-form approval too",
			provider:  []string{"-a", "on-request", "-s", "workspace-write"},
			user:      []string{"--dangerously-bypass-approvals-and-sandbox"},
			conflicts: codexConflicts,
			want:      []string{},
		},
		{
			name:      "codex bypass strips --key=value form",
			provider:  []string{"--ask-for-approval=on-request", "--sandbox=read-only"},
			user:      []string{"--dangerously-bypass-approvals-and-sandbox"},
			conflicts: codexConflicts,
			want:      []string{},
		},
		{
			name:      "no trigger flag keeps provider args intact",
			provider:  []string{"--ask-for-approval", "on-request", "-s", "workspace-write"},
			user:      []string{"--verbose"},
			conflicts: codexConflicts,
			want:      []string{"--ask-for-approval", "on-request", "-s", "workspace-write"},
		},
		{
			name:      "non-conflict provider flags survive",
			provider:  []string{"--ask-for-approval", "on-request", "-c", "model=gpt-5"},
			user:      []string{"--dangerously-bypass-approvals-and-sandbox"},
			conflicts: codexConflicts,
			want:      []string{"-c", "model=gpt-5"},
		},
		{
			name:      "nil conflicts map is a no-op",
			provider:  []string{"--ask-for-approval", "on-request"},
			user:      []string{"--dangerously-bypass-approvals-and-sandbox"},
			conflicts: nil,
			want:      []string{"--ask-for-approval", "on-request"},
		},
		{
			name:      "positional values preserved",
			provider:  []string{"--ask-for-approval", "on-request", "prompt-here"},
			user:      []string{"--dangerously-bypass-approvals-and-sandbox"},
			conflicts: codexConflicts,
			want:      []string{"prompt-here"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := dropConflictingFlags(tc.provider, tc.user, tc.conflicts)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("dropConflictingFlags(%v, %v, %v) = %v, want %v",
					tc.provider, tc.user, tc.conflicts, got, tc.want)
			}
		})
	}
}

func TestFlagName(t *testing.T) {
	tests := []struct {
		in   string
		name string
		ok   bool
	}{
		{"--ask-for-approval", "--ask-for-approval", true},
		{"--ask-for-approval=never", "--ask-for-approval", true},
		{"-s", "-s", true},
		{"-s=workspace-write", "-s", true},
		{"workspace-write", "", false},
		{"", "", false},
		{"-", "", false},
	}
	for _, tc := range tests {
		gotName, gotOk := flagName(tc.in)
		if gotName != tc.name || gotOk != tc.ok {
			t.Errorf("flagName(%q) = (%q, %v); want (%q, %v)",
				tc.in, gotName, gotOk, tc.name, tc.ok)
		}
	}
}

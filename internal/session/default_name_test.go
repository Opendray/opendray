package session

import "testing"

func TestDefaultSessionName(t *testing.T) {
	cases := []struct {
		name     string
		provider string
		cwd      string
		want     string
	}{
		{"cwd basename", "claude", "/home/navid/projects/opendray", "opendray"},
		{"trailing slash", "antigravity", "/home/navid/projects/opendray/", "opendray"},
		{"root falls back to provider", "claude", "/", "claude"},
		{"empty falls back to provider", "codex", "", "codex"},
		{"dot falls back to provider", "antigravity", ".", "antigravity"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := defaultSessionName(tc.provider, tc.cwd); got != tc.want {
				t.Errorf("defaultSessionName(%q, %q) = %q, want %q",
					tc.provider, tc.cwd, got, tc.want)
			}
		})
	}
}

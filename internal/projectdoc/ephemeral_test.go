package projectdoc

import "testing"

func TestIsEphemeralCwd(t *testing.T) {
	tests := []struct {
		cwd  string
		want bool
	}{
		{"", true},
		{"   ", true},
		{"/tmp", true},
		{"/tmp/odray-test-1234", true},
		{"/private/tmp/x", true},
		{"/var/folders/ab/T/session", true},
		{"/private/var/folders/zz/x", true},
		{"/Users/dev/proj/tmp.abc123", true},
		{"/home/dev/.cache/something", true},

		{"/Users/linivek/Documents/HomeLab/Claude_Workspace/opendray-v2", false},
		{"/home/dev/proj", false},
		{"/srv/app", false},
		// 'tmp' as a real path segment (not /tmp root, not tmp.) stays a project.
		{"/home/dev/tmpwork", false},
		{"/home/dev/templates", false},
	}
	for _, tt := range tests {
		if got := IsEphemeralCwd(tt.cwd); got != tt.want {
			t.Errorf("IsEphemeralCwd(%q) = %v, want %v", tt.cwd, got, tt.want)
		}
	}
}

func TestValidateWriteTargetRejectsEphemeral(t *testing.T) {
	s := &Service{} // validateWriteTarget short-circuits before any DB use
	if err := s.validateWriteTarget(nil, "/tmp/throwaway", KindGoal); err == nil {
		t.Fatalf("doc write under /tmp must be rejected")
	}
}

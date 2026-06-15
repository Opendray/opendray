package backup

import "testing"

func TestBackupKind_OrDefault(t *testing.T) {
	cases := []struct {
		in   BackupKind
		want BackupKind
	}{
		{"", KindDBOnly},
		{KindDBOnly, KindDBOnly},
		{KindFullInstance, KindFullInstance},
	}
	for _, tc := range cases {
		if got := tc.in.orDefault(); got != tc.want {
			t.Errorf("BackupKind(%q).orDefault() = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestParseBackupKind(t *testing.T) {
	tests := []struct {
		in      string
		want    BackupKind
		wantErr bool
	}{
		{"", KindDBOnly, false},
		{"db_only", KindDBOnly, false},
		{"full_instance", KindFullInstance, false},
		{"bogus", "", true},
		{"Full_Instance", "", true}, // case-sensitive by design
	}
	for _, tc := range tests {
		got, err := ParseBackupKind(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("ParseBackupKind(%q) expected error, got %q", tc.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseBackupKind(%q) unexpected error: %v", tc.in, err)
		}
		if got != tc.want {
			t.Errorf("ParseBackupKind(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

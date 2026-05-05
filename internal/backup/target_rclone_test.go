package backup

import (
	"errors"
	"os/exec"
	"strings"
	"testing"
)

func TestNewRcloneTarget_Validation(t *testing.T) {
	if _, err := exec.LookPath("rclone"); err != nil {
		t.Skip("rclone not on PATH")
	}
	cases := []struct {
		name string
		id   string
		cfg  RcloneConfig
		want string
	}{
		{"happy", "rc1", RcloneConfig{Remote: "gdrive"}, ""},
		{"no_id", "", RcloneConfig{Remote: "gdrive"}, "id required"},
		{"no_remote", "x", RcloneConfig{}, "remote name required"},
		{"colon_in_remote", "x", RcloneConfig{Remote: "bad:remote"}, "must not contain"},
		{"slash_in_remote", "x", RcloneConfig{Remote: "bad/remote"}, "must not contain"},
		{"space_in_remote", "x", RcloneConfig{Remote: "bad remote"}, "must not contain"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := NewRcloneTarget(c.id, c.cfg)
			if c.want == "" && err != nil {
				t.Fatalf("got err %v, want nil", err)
			}
			if c.want != "" && (err == nil || !strings.Contains(err.Error(), c.want)) {
				t.Fatalf("got err %v, want substring %q", err, c.want)
			}
		})
	}
}

func TestNewRcloneTarget_BinaryNotFound(t *testing.T) {
	_, err := NewRcloneTarget("x", RcloneConfig{
		Remote:     "gdrive",
		BinaryPath: "/definitely/not/a/real/path/rclone",
	})
	if err == nil || !strings.Contains(err.Error(), "unusable") {
		t.Fatalf("got %v, want 'unusable' error", err)
	}
}

func TestRcloneTarget_RemotePath(t *testing.T) {
	if _, err := exec.LookPath("rclone"); err != nil {
		t.Skip("rclone not on PATH")
	}
	tg, err := NewRcloneTarget("rc", RcloneConfig{
		Remote:     "gdrive",
		PathPrefix: "opendray/backups",
	})
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct{ in, want string }{
		{"a.bin", "gdrive:opendray/backups/a.bin"},
		{"sub/dir/file.bin", "gdrive:opendray/backups/sub/dir/file.bin"},
		{"/leading-slash.bin", "gdrive:opendray/backups/leading-slash.bin"},
	}
	for _, c := range cases {
		got, err := tg.remotePath(c.in)
		if err != nil {
			t.Errorf("remotePath(%q) err = %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("remotePath(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestRcloneTarget_RemotePath_Rejects(t *testing.T) {
	if _, err := exec.LookPath("rclone"); err != nil {
		t.Skip("rclone not on PATH")
	}
	tg, _ := NewRcloneTarget("rc", RcloneConfig{Remote: "gdrive"})
	bad := []string{"", "../up.bin", "a/../../up.bin", "x\x00y"}
	for _, in := range bad {
		_, err := tg.remotePath(in)
		if !errors.Is(err, ErrTargetRejectedPath) {
			t.Errorf("remotePath(%q) err = %v, want ErrTargetRejectedPath", in, err)
		}
	}
}

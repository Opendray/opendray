package backup

import (
	"errors"
	"strings"
	"testing"
)

func TestNewSMBTarget_Validation(t *testing.T) {
	cases := []struct {
		name string
		id   string
		cfg  SMBConfig
		want string // substring of error; "" means OK
	}{
		{"happy", "smb1", SMBConfig{Host: "h", Share: "s", User: "u", Password: "p"}, ""},
		{"empty_id", "", SMBConfig{Host: "h", Share: "s", User: "u"}, "id required"},
		{"no_host", "smb1", SMBConfig{Share: "s", User: "u"}, "host required"},
		{"no_share", "smb1", SMBConfig{Host: "h", User: "u"}, "share required"},
		{"no_user", "smb1", SMBConfig{Host: "h", Share: "s"}, "user required"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := NewSMBTarget(c.id, c.cfg)
			if c.want == "" && err != nil {
				t.Fatalf("got err %v, want nil", err)
			}
			if c.want != "" && (err == nil || !strings.Contains(err.Error(), c.want)) {
				t.Fatalf("got err %v, want substring %q", err, c.want)
			}
		})
	}
}

func TestSMBTarget_DefaultPort(t *testing.T) {
	tg, err := NewSMBTarget("smb", SMBConfig{Host: "h", Share: "s", User: "u"})
	if err != nil {
		t.Fatal(err)
	}
	if tg.cfg.Port != 445 {
		t.Errorf("Port = %d, want 445", tg.cfg.Port)
	}
}

func TestSMBTarget_Resolve(t *testing.T) {
	tg, _ := NewSMBTarget("smb", SMBConfig{Host: "h", Share: "s", User: "u", PathPrefix: "opendray/backups"})
	good := []struct{ in, want string }{
		{"a.bin", "opendray/backups/a.bin"},
		{"sub/dir/file.bin", "opendray/backups/sub/dir/file.bin"},
		{"/leading-slash.bin", "opendray/backups/leading-slash.bin"},
	}
	for _, c := range good {
		got, err := tg.resolve(c.in)
		if err != nil {
			t.Errorf("resolve(%q) err = %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("resolve(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSMBTarget_Resolve_Rejects(t *testing.T) {
	tg, _ := NewSMBTarget("smb", SMBConfig{Host: "h", Share: "s", User: "u", PathPrefix: "p"})
	bad := []string{"", "../up.bin", "a/../../up.bin", "x\x00y"}
	for _, in := range bad {
		_, err := tg.resolve(in)
		if !errors.Is(err, ErrTargetRejectedPath) {
			t.Errorf("resolve(%q) err = %v, want ErrTargetRejectedPath", in, err)
		}
	}
}

func TestSMBTarget_Resolve_NoPrefix(t *testing.T) {
	tg, _ := NewSMBTarget("smb", SMBConfig{Host: "h", Share: "s", User: "u"})
	got, err := tg.resolve("a.bin")
	if err != nil {
		t.Fatal(err)
	}
	if got != "a.bin" {
		t.Errorf("got %q, want a.bin", got)
	}
}

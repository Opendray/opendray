package backup

import (
	"errors"
	"strings"
	"testing"
)

func TestNewSFTPTarget_Validation(t *testing.T) {
	cases := []struct {
		name string
		id   string
		cfg  SFTPConfig
		want string
	}{
		{"happy_pw", "sftp1", SFTPConfig{Host: "h", User: "u", Password: "p"}, ""},
		{"happy_key", "sftp1", SFTPConfig{Host: "h", User: "u", PrivateKey: "fake-pem"}, ""},
		{"no_id", "", SFTPConfig{Host: "h", User: "u", Password: "p"}, "id required"},
		{"no_host", "x", SFTPConfig{User: "u", Password: "p"}, "host required"},
		{"no_user", "x", SFTPConfig{Host: "h", Password: "p"}, "user required"},
		{"no_auth", "x", SFTPConfig{Host: "h", User: "u"}, "password or private_key required"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := NewSFTPTarget(c.id, c.cfg)
			if c.want == "" && err != nil {
				t.Fatalf("got err %v, want nil", err)
			}
			if c.want != "" && (err == nil || !strings.Contains(err.Error(), c.want)) {
				t.Fatalf("got err %v, want substring %q", err, c.want)
			}
		})
	}
}

func TestSFTPTarget_DefaultPort(t *testing.T) {
	tg, err := NewSFTPTarget("sftp", SFTPConfig{Host: "h", User: "u", Password: "p"})
	if err != nil {
		t.Fatal(err)
	}
	if tg.cfg.Port != 22 {
		t.Errorf("Port = %d, want 22", tg.cfg.Port)
	}
}

func TestSFTPTarget_Resolve_AbsPrefix(t *testing.T) {
	tg, _ := NewSFTPTarget("sftp", SFTPConfig{
		Host: "h", User: "u", Password: "p", PathPrefix: "/var/backups/opendray",
	})
	got, err := tg.resolve("a.bin")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/var/backups/opendray/a.bin" {
		t.Errorf("got %q", got)
	}
}

func TestSFTPTarget_Resolve_RelativePrefix(t *testing.T) {
	tg, _ := NewSFTPTarget("sftp", SFTPConfig{
		Host: "h", User: "u", Password: "p", PathPrefix: "opendray/backups",
	})
	got, err := tg.resolve("a.bin")
	if err != nil {
		t.Fatal(err)
	}
	if got != "opendray/backups/a.bin" {
		t.Errorf("got %q", got)
	}
}

func TestSFTPTarget_Resolve_Rejects(t *testing.T) {
	tg, _ := NewSFTPTarget("sftp", SFTPConfig{Host: "h", User: "u", Password: "p"})
	bad := []string{"", "../up.bin", "a/../../up.bin", "x\x00y"}
	for _, in := range bad {
		_, err := tg.resolve(in)
		if !errors.Is(err, ErrTargetRejectedPath) {
			t.Errorf("resolve(%q) err = %v, want ErrTargetRejectedPath", in, err)
		}
	}
}

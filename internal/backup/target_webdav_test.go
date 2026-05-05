package backup

import (
	"errors"
	"strings"
	"testing"
)

func TestNewWebDAVTarget_Validation(t *testing.T) {
	cases := []struct {
		name string
		id   string
		cfg  WebDAVConfig
		want string
	}{
		{"happy_https", "wd1", WebDAVConfig{BaseURL: "https://cloud.example.com/dav/", User: "u", Password: "p"}, ""},
		{"happy_http", "wd1", WebDAVConfig{BaseURL: "http://nas.local/dav", User: "u"}, ""},
		{"no_id", "", WebDAVConfig{BaseURL: "https://x", User: "u"}, "id required"},
		{"no_url", "x", WebDAVConfig{User: "u"}, "base_url required"},
		{"bad_scheme", "x", WebDAVConfig{BaseURL: "ftp://x", User: "u"}, "must start with http"},
		{"no_user", "x", WebDAVConfig{BaseURL: "https://x"}, "user required"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := NewWebDAVTarget(c.id, c.cfg)
			if c.want == "" && err != nil {
				t.Fatalf("got err %v, want nil", err)
			}
			if c.want != "" && (err == nil || !strings.Contains(err.Error(), c.want)) {
				t.Fatalf("got err %v, want substring %q", err, c.want)
			}
		})
	}
}

func TestWebDAVTarget_Resolve(t *testing.T) {
	tg, _ := NewWebDAVTarget("wd", WebDAVConfig{
		BaseURL: "https://x", User: "u",
		PathPrefix: "opendray/backups",
	})
	cases := []struct{ in, want string }{
		{"a.bin", "/opendray/backups/a.bin"},
		{"sub/dir/file.bin", "/opendray/backups/sub/dir/file.bin"},
		{"/leading-slash.bin", "/opendray/backups/leading-slash.bin"},
	}
	for _, c := range cases {
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

func TestWebDAVTarget_Resolve_Rejects(t *testing.T) {
	tg, _ := NewWebDAVTarget("wd", WebDAVConfig{BaseURL: "https://x", User: "u"})
	bad := []string{"", "../up.bin", "a/../../up.bin", "x\x00y"}
	for _, in := range bad {
		_, err := tg.resolve(in)
		if !errors.Is(err, ErrTargetRejectedPath) {
			t.Errorf("resolve(%q) err = %v, want ErrTargetRejectedPath", in, err)
		}
	}
}

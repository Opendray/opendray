package backup

import (
	"errors"
	"strings"
	"testing"
)

func TestNewS3Target_Validation(t *testing.T) {
	cases := []struct {
		name string
		id   string
		cfg  S3Config
		want string
	}{
		{"happy", "s3-aws", S3Config{Endpoint: "s3.amazonaws.com", Bucket: "b", AccessKey: "k", SecretKey: "s"}, ""},
		{"no_id", "", S3Config{Endpoint: "x", Bucket: "b", AccessKey: "k", SecretKey: "s"}, "id required"},
		{"no_endpoint", "x", S3Config{Bucket: "b", AccessKey: "k", SecretKey: "s"}, "endpoint required"},
		{"no_bucket", "x", S3Config{Endpoint: "x", AccessKey: "k", SecretKey: "s"}, "bucket required"},
		{"no_access_key", "x", S3Config{Endpoint: "x", Bucket: "b", SecretKey: "s"}, "access_key required"},
		{"no_secret_key", "x", S3Config{Endpoint: "x", Bucket: "b", AccessKey: "k"}, "secret_key required"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := NewS3Target(c.id, c.cfg)
			if c.want == "" && err != nil {
				t.Fatalf("got err %v, want nil", err)
			}
			if c.want != "" && (err == nil || !strings.Contains(err.Error(), c.want)) {
				t.Fatalf("got err %v, want substring %q", err, c.want)
			}
		})
	}
}

func TestS3Target_Resolve(t *testing.T) {
	tg, _ := NewS3Target("s3", S3Config{
		Endpoint: "x", Bucket: "b", AccessKey: "k", SecretKey: "s",
		PathPrefix: "opendray/backups",
	})
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

func TestS3Target_Resolve_Rejects(t *testing.T) {
	tg, _ := NewS3Target("s3", S3Config{Endpoint: "x", Bucket: "b", AccessKey: "k", SecretKey: "s"})
	bad := []string{"", "../up.bin", "a/../../up.bin", "x\x00y"}
	for _, in := range bad {
		_, err := tg.resolve(in)
		if !errors.Is(err, ErrTargetRejectedPath) {
			t.Errorf("resolve(%q) err = %v, want ErrTargetRejectedPath", in, err)
		}
	}
}

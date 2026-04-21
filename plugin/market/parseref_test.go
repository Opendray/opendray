package market

import (
	"errors"
	"testing"
)

func TestParseRef_Accepts(t *testing.T) {
	cases := []struct {
		in   string
		want Ref
	}{
		{"foo", Ref{Name: "foo"}},
		{"foo@1.0.0", Ref{Name: "foo", Version: "1.0.0"}},
		{"marketplace://foo", Ref{Name: "foo"}},
		{"marketplace://foo@1.0.0", Ref{Name: "foo", Version: "1.0.0"}},
		{"acme/hello", Ref{Publisher: "acme", Name: "hello"}},
		{"acme/hello@2.1.0", Ref{Publisher: "acme", Name: "hello", Version: "2.1.0"}},
		{"marketplace://acme/hello@2.1.0", Ref{Publisher: "acme", Name: "hello", Version: "2.1.0"}},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, err := ParseRef(tc.in)
			if err != nil {
				t.Fatalf("ParseRef(%q) err: %v", tc.in, err)
			}
			if got != tc.want {
				t.Errorf("ParseRef(%q) = %#v, want %#v", tc.in, got, tc.want)
			}
		})
	}
}

func TestParseRef_Rejects(t *testing.T) {
	cases := []string{
		"",
		"   ",
		"marketplace://",
		"../etc/passwd",
		`\backslash`,
		"foo/bar/baz",   // too many slashes
		"/foo",          // empty publisher
		"foo/",          // empty name
	}
	for _, raw := range cases {
		t.Run(raw, func(t *testing.T) {
			if _, err := ParseRef(raw); !errors.Is(err, ErrBadRef) {
				t.Errorf("ParseRef(%q) = %v, want ErrBadRef", raw, err)
			}
		})
	}
}

func TestRefString(t *testing.T) {
	cases := []struct {
		ref  Ref
		want string
	}{
		{Ref{Name: "foo"}, "foo"},
		{Ref{Name: "foo", Version: "1.0.0"}, "foo@1.0.0"},
		{Ref{Publisher: "acme", Name: "hello"}, "acme/hello"},
		{Ref{Publisher: "acme", Name: "hello", Version: "2.1.0"}, "acme/hello@2.1.0"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			if got := tc.ref.String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}

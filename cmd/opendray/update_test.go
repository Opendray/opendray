package main

import "testing"

func TestNormaliseVersion(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"v2.0.0", "2.0.0"},
		{"2.0.0", "2.0.0"},
		{"  v2.0.0 ", "2.0.0"},
		{"", ""},
		{"v0.0.0-dev", "0.0.0-dev"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			if got := normaliseVersion(tc.in); got != tc.want {
				t.Errorf("normaliseVersion(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

package app

import "testing"

func TestScopesCover(t *testing.T) {
	cases := []struct {
		name string
		have []string
		want []string
		ok   bool
	}{
		{"all present", []string{"memory:read", "memory:write", "session:read"}, []string{"memory:read", "memory:write"}, true},
		{"missing write", []string{"memory:read", "session:read"}, []string{"memory:read", "memory:write"}, false},
		{"missing all", []string{"session:read"}, []string{"memory:read", "memory:write"}, false},
		{"empty have", nil, []string{"memory:read"}, false},
		{"empty want", []string{"session:read"}, nil, true},
		{"wildcard covers prefix", []string{"memory.*"}, []string{"memory.read"}, true},
		{"exact wildcard mismatch", []string{"memory.*"}, []string{"session.read"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := scopesCover(c.have, c.want); got != c.ok {
				t.Errorf("scopesCover(%v, %v) = %v, want %v", c.have, c.want, got, c.ok)
			}
		})
	}
}

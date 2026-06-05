package memory

import "testing"

func TestNormalizeScope(t *testing.T) {
	cases := []struct {
		in   Scope
		want Scope
	}{
		{ScopeProject, ScopeProject},
		{ScopeGlobal, ScopeGlobal},
		{legacyScopeSession, ScopeProject}, // retired session folds to project
		{"session", ScopeProject},
		{"", ""},           // empty passes through (caller defaults it)
		{"weird", "weird"}, // unknown passes through so Validate can reject
	}
	for _, c := range cases {
		t.Run(string(c.in), func(t *testing.T) {
			if got := normalizeScope(c.in); got != c.want {
				t.Errorf("normalizeScope(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestScopeValidate(t *testing.T) {
	cases := []struct {
		in    Scope
		valid bool
	}{
		{ScopeProject, true},
		{ScopeGlobal, true},
		{legacyScopeSession, false}, // session is no longer a valid scope on its own
		{"", false},
		{"nonsense", false},
	}
	for _, c := range cases {
		t.Run(string(c.in), func(t *testing.T) {
			err := c.in.Validate()
			if (err == nil) != c.valid {
				t.Errorf("Validate(%q) err=%v, want valid=%v", c.in, err, c.valid)
			}
		})
	}
}

package revocation

import (
	"strings"
	"testing"
)

// TestMatches — happy path + negative cases across all supported
// range shapes. Table-driven so regressions show the exact input
// that drifted.
func TestMatches(t *testing.T) {
	cases := []struct {
		name        string
		entryName   string
		versions    string
		pub, pname  string
		ver         string
		want        bool
		wantErr     bool
	}{
		{"exact match", "acme/evil", "1.2.3", "acme", "evil", "1.2.3", true, false},
		{"exact miss",  "acme/evil", "1.2.3", "acme", "evil", "1.2.4", false, false},
		{"leq hit",     "acme/evil", "<=1.2.3", "acme", "evil", "1.2.3", true, false},
		{"leq miss",    "acme/evil", "<=1.2.3", "acme", "evil", "1.2.4", false, false},
		{"lt hit",      "acme/evil", "<2.0.0", "acme", "evil", "1.9.9", true, false},
		{"geq hit",     "acme/evil", ">=2.0.0", "acme", "evil", "2.1.0", true, false},
		{"wildcard hit","acme/evil", "*", "acme", "evil", "999.0.0", true, false},
		{"empty range",  "acme/evil", "", "acme", "evil", "1.0.0", true, false},
		{"publisher mismatch", "acme/evil", "*", "rival", "evil", "1.0.0", false, false},
		{"name mismatch",      "acme/evil", "*", "acme", "other", "1.0.0", false, false},
		{"bare name defaults to opendray-examples",
			"fs-readme", "*", "opendray-examples", "fs-readme", "1.0.0", true, false},
		{"bare name mismatches acme publisher",
			"fs-readme", "*", "acme", "fs-readme", "1.0.0", false, false},
		{"installed publisher defaults to opendray-examples",
			"fs-readme", "*", "", "fs-readme", "1.0.0", true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := Entry{Name: tc.entryName, Versions: tc.versions, Action: "warn"}
			got, err := e.Matches(tc.pub, tc.pname, tc.ver)
			if (err != nil) != tc.wantErr {
				t.Errorf("err = %v, wantErr=%v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("Matches = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestMatches_NonSemverVersion — an installed version that isn't
// valid semver (e.g. "dev-build") falls through ANY constraint
// except "*".
func TestMatches_NonSemverVersion(t *testing.T) {
	e := Entry{Name: "acme/x", Versions: "<=1.0.0"}
	hit, err := e.Matches("acme", "x", "dev-build")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if hit {
		t.Errorf("non-semver version should miss <=1.0.0")
	}

	e.Versions = "*"
	hit, _ = e.Matches("acme", "x", "dev-build")
	if !hit {
		t.Errorf("non-semver version should still match *")
	}
}

// TestMatches_MalformedEntry — bad Name surfaces the error so the
// poller can log + skip.
func TestMatches_MalformedEntry(t *testing.T) {
	cases := []string{"", "a/b/c", "a/", "/a", "has space"}
	for _, raw := range cases {
		t.Run(raw, func(t *testing.T) {
			e := Entry{Name: raw, Versions: "*"}
			_, err := e.Matches("x", "y", "1.0.0")
			if err == nil {
				t.Errorf("want error on malformed Name %q", raw)
			}
		})
	}
}

// TestMatches_MalformedVersions — bad Versions range surfaces.
func TestMatches_MalformedVersions(t *testing.T) {
	e := Entry{Name: "acme/x", Versions: "not~a~range"}
	_, err := e.Matches("acme", "x", "1.0.0")
	if err == nil || !strings.Contains(err.Error(), "versions") {
		t.Errorf("err = %v; want versions error", err)
	}
}

// TestNormalisedAction — unknown actions map to warn.
func TestNormalisedAction(t *testing.T) {
	cases := map[string]string{
		ActionUninstall: ActionUninstall,
		ActionDisable:   ActionDisable,
		ActionWarn:      ActionWarn,
		"":              ActionWarn,
		"unknown":       ActionWarn,
		"deprecate":     ActionWarn,
	}
	for in, want := range cases {
		if got := (Entry{Action: in}).NormalisedAction(); got != want {
			t.Errorf("NormalisedAction(%q) = %q, want %q", in, got, want)
		}
	}
}

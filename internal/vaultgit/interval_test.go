package vaultgit

import (
	"errors"
	"testing"
)

// parseIntervalMs is what stands between the operator and a setting that
// silently does nothing: before it existed, an unparseable interval was
// swallowed and the previous value kept, so a typo returned 200 OK and
// changed nothing.
func TestParseIntervalMs(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    int64
		wantErr bool
	}{
		{"minutes", "10m", 600_000, false},
		{"hours", "2h", 7_200_000, false},
		{"seconds", "45s", 45_000, false},
		{"compound", "1h30m", 5_400_000, false},
		{"surrounding space is tolerated", "  15m  ", 900_000, false},

		// Below the loop's own tick floor: clamped, not rejected —
		// asking for "every 5s" is a reasonable thing to type and the
		// friendlier answer is the fastest we can actually do.
		{"clamped to the tick floor", "5s", 30_000, false},
		{"exactly the floor", "30s", 30_000, false},

		// Everything the old fallback swallowed.
		{"bare number", "10", 0, true},
		{"unit only", "m", 0, true},
		{"empty", "", 0, true},
		{"whitespace only", "   ", 0, true},
		{"prose", "ten minutes", 0, true},
		{"milliseconds as a plain int", "600000", 0, true},
		{"unknown unit", "10x", 0, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseIntervalMs("commit_interval", tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("parseIntervalMs(%q) = %d, want an error", tc.in, got)
				}
				if !errors.Is(err, ErrBadInterval) {
					t.Fatalf("error must wrap ErrBadInterval so the handler can answer 400; got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseIntervalMs(%q) unexpected error: %v", tc.in, err)
			}
			if got != tc.want {
				t.Fatalf("parseIntervalMs(%q) = %d ms, want %d ms", tc.in, got, tc.want)
			}
		})
	}
}

// The field name has to reach the operator — "invalid interval" alone
// doesn't say which of the two fields was rejected.
func TestParseIntervalMsErrorNamesTheField(t *testing.T) {
	_, err := parseIntervalMs("pull_interval", "nope")
	if err == nil {
		t.Fatal("want an error")
	}
	if got := err.Error(); !contains(got, "pull_interval") || !contains(got, `"nope"`) {
		t.Fatalf("error should name the field and echo the value, got %q", got)
	}
}

func contains(haystack, needle string) bool {
	return len(needle) == 0 || (len(haystack) >= len(needle) &&
		func() bool {
			for i := 0; i+len(needle) <= len(haystack); i++ {
				if haystack[i:i+len(needle)] == needle {
					return true
				}
			}
			return false
		}())
}

package telegram

import (
	"reflect"
	"testing"
)

func TestParseExtraClaudeDirs(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", nil},
		{"whitespace only", "   \n  \t ", nil},
		{"single path", "/home/me/.claude", []string{"/home/me/.claude"}},
		{
			"comma separated with spaces",
			"/a/.claude, /b/.claude ,  /c/.claude",
			[]string{"/a/.claude", "/b/.claude", "/c/.claude"},
		},
		{
			"newline separated",
			"/a/.claude\n/b/.claude\n/c/.claude",
			[]string{"/a/.claude", "/b/.claude", "/c/.claude"},
		},
		{
			"mixed with empty fields",
			"/a/.claude,,\n,/b/.claude,",
			[]string{"/a/.claude", "/b/.claude"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseExtraClaudeDirs(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseExtraClaudeDirs(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

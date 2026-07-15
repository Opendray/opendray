package roundtable

import (
	"reflect"
	"strings"
	"testing"
)

func TestNormalizeSeats(t *testing.T) {
	tests := []struct {
		name    string
		seats   []Seat
		wantErr bool
		check   func(t *testing.T, got []Seat)
	}{
		{
			name:    "no seats",
			seats:   nil,
			wantErr: true,
		},
		{
			name:    "unsupported provider",
			seats:   []Seat{{Provider: "gemini"}},
			wantErr: true,
		},
		{
			name:  "opencode seat allowed",
			seats: []Seat{{Provider: "opencode"}},
			check: func(t *testing.T, got []Seat) {
				if len(got) != 1 || got[0].Provider != "opencode" {
					t.Fatalf("want 1 opencode seat, got %v", got)
				}
			},
		},
		{
			name:  "grok seat allowed",
			seats: []Seat{{Provider: "grok"}},
			check: func(t *testing.T, got []Seat) {
				if len(got) != 1 || got[0].Provider != "grok" {
					t.Fatalf("want 1 grok seat, got %v", got)
				}
			},
		},
		{
			name:  "persona trimmed and kept",
			seats: []Seat{{Provider: "claude", Persona: "  you are the security reviewer  "}},
			check: func(t *testing.T, got []Seat) {
				if got[0].Persona != "you are the security reviewer" {
					t.Errorf("persona should be trimmed + kept, got %q", got[0].Persona)
				}
			},
		},
		{
			name:  "five-vendor table",
			seats: []Seat{{Provider: "claude"}, {Provider: "codex"}, {Provider: "antigravity"}, {Provider: "grok"}, {Provider: "opencode"}},
			check: func(t *testing.T, got []Seat) {
				if len(got) != 5 {
					t.Fatalf("want 5 seats, got %d", len(got))
				}
			},
		},
		{
			name:    "duplicate vendor rejected",
			seats:   []Seat{{Provider: "claude"}, {Provider: "claude"}},
			wantErr: true,
		},
		{
			name:  "single seat is allowed",
			seats: []Seat{{Provider: "claude"}},
			check: func(t *testing.T, got []Seat) {
				if len(got) != 1 {
					t.Fatalf("want 1 seat, got %d", len(got))
				}
			},
		},
		{
			name:  "three-vendor table",
			seats: []Seat{{Provider: "claude"}, {Provider: "codex"}, {Provider: "antigravity"}},
			check: func(t *testing.T, got []Seat) {
				if len(got) != 3 {
					t.Fatalf("want 3 seats, got %d", len(got))
				}
			},
		},
		{
			name: "account kept for claude + antigravity, cleared for others",
			seats: []Seat{
				{Provider: "claude", AccountID: "acct1"},
				{Provider: "antigravity", AccountID: "agy1"},
				{Provider: "codex", AccountID: "leak"},
				{Provider: "grok", AccountID: "leak"},
				{Provider: "opencode", AccountID: "leak"},
			},
			check: func(t *testing.T, got []Seat) {
				want := map[string]string{
					"claude":      "acct1",
					"antigravity": "agy1",
					"codex":       "",
					"grok":        "",
					"opencode":    "",
				}
				for _, s := range got {
					if s.AccountID != want[s.Provider] {
						t.Errorf("%s account = %q, want %q", s.Provider, s.AccountID, want[s.Provider])
					}
				}
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := normalizeSeats(tc.seats)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("want error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.check != nil {
				tc.check(t, got)
			}
		})
	}
}

func TestValidSeatProvider(t *testing.T) {
	for _, p := range []string{"claude", "codex", "antigravity", "grok", "opencode"} {
		if !validSeatProvider(p) {
			t.Errorf("%q should be valid", p)
		}
	}
	for _, p := range []string{"", "gemini", "shell"} {
		if validSeatProvider(p) {
			t.Errorf("%q should be invalid (no headless worker path)", p)
		}
	}
}

func TestParseMentions(t *testing.T) {
	seats := []Seat{{Provider: "claude"}, {Provider: "codex"}, {Provider: "antigravity"}}
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{name: "no mention", content: "just thinking out loud", want: nil},
		{name: "single", content: "@codex what do you think?", want: []string{"codex"}},
		{
			name:    "multiple in seat order regardless of text order",
			content: "@antigravity and @claude please weigh in",
			want:    []string{"claude", "antigravity"},
		},
		{
			name:    "all expands to every seat",
			content: "@all go",
			want:    []string{"claude", "codex", "antigravity"},
		},
		{
			name:    "case insensitive",
			content: "@CoDeX ?",
			want:    []string{"codex"},
		},
		{
			name:    "unknown mention ignored",
			content: "@gemini @claude",
			want:    []string{"claude"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseMentions(tc.content, seats)
			if len(got) == 0 && len(tc.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("parseMentions(%q) = %v, want %v", tc.content, got, tc.want)
			}
		})
	}
}

func TestDeriveTitle(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "@all let's discuss the game idea", want: "let's discuss the game idea"},
		{in: "just a plain message", want: "just a plain message"},
		{in: "first line\nsecond line", want: "first line"},
		{in: "@claude", want: "@claude"}, // all-mention line falls back to raw
		{in: "   ", want: ""},
	}
	for _, tc := range tests {
		if got := deriveTitle(tc.in); got != tc.want {
			t.Errorf("deriveTitle(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
	// Long input is truncated with an ellipsis.
	long := deriveTitle(strings.Repeat("x", 200))
	if r := []rune(long); len(r) != 81 || r[80] != '…' {
		t.Errorf("long title should truncate to 80 runes + ellipsis, got len %d", len([]rune(long)))
	}
}

func TestParseMentions_OnlySeatedProvidersAddressed(t *testing.T) {
	// @codex is not seated → not returned even though it's a valid provider.
	seats := []Seat{{Provider: "claude"}}
	got := parseMentions("@codex @claude", seats)
	if !reflect.DeepEqual(got, []string{"claude"}) {
		t.Errorf("only seated members should be addressable; got %v", got)
	}
}

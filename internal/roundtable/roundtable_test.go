package roundtable

import (
	"reflect"
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
			name:    "opencode has no headless path",
			seats:   []Seat{{Provider: "opencode"}},
			wantErr: true,
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
			name:  "non-claude account cleared",
			seats: []Seat{{Provider: "claude", AccountID: "acct1"}, {Provider: "codex", AccountID: "leak"}},
			check: func(t *testing.T, got []Seat) {
				for _, s := range got {
					if s.Provider == "codex" && s.AccountID != "" {
						t.Errorf("codex account should be cleared, got %q", s.AccountID)
					}
					if s.Provider == "claude" && s.AccountID != "acct1" {
						t.Errorf("claude account should be kept, got %q", s.AccountID)
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
	for _, p := range []string{"claude", "codex", "antigravity"} {
		if !validSeatProvider(p) {
			t.Errorf("%q should be valid", p)
		}
	}
	for _, p := range []string{"", "gemini", "opencode", "grok", "shell"} {
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

func TestParseMentions_OnlySeatedProvidersAddressed(t *testing.T) {
	// @codex is not seated → not returned even though it's a valid provider.
	seats := []Seat{{Provider: "claude"}}
	got := parseMentions("@codex @claude", seats)
	if !reflect.DeepEqual(got, []string{"claude"}) {
		t.Errorf("only seated members should be addressable; got %v", got)
	}
}

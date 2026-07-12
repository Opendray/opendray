package roundtable

import "testing"

func TestNormalizeSeats(t *testing.T) {
	tests := []struct {
		name    string
		seats   []Seat
		wantErr bool
		check   func(t *testing.T, got []Seat)
	}{
		{
			name:    "too few seats",
			seats:   []Seat{{Provider: "claude"}},
			wantErr: true,
		},
		{
			name:    "unsupported provider",
			seats:   []Seat{{Provider: "claude"}, {Provider: "gemini"}},
			wantErr: true,
		},
		{
			name:    "opencode has no headless path",
			seats:   []Seat{{Provider: "claude"}, {Provider: "opencode"}},
			wantErr: true,
		},
		{
			name:    "duplicate vendor rejected",
			seats:   []Seat{{Provider: "claude"}, {Provider: "claude"}},
			wantErr: true,
		},
		{
			name:  "valid three-vendor table",
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
	valid := []string{"claude", "codex", "antigravity"}
	for _, p := range valid {
		if !validSeatProvider(p) {
			t.Errorf("%q should be valid", p)
		}
	}
	invalid := []string{"", "gemini", "opencode", "grok", "shell"}
	for _, p := range invalid {
		if validSeatProvider(p) {
			t.Errorf("%q should be invalid (no headless worker path)", p)
		}
	}
}

func TestParseJSON(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
		want    string
	}{
		{name: "clean", raw: `{"summary":"ok"}`, want: "ok"},
		{name: "fenced", raw: "```json\n{\"summary\":\"ok\"}\n```", want: "ok"},
		{name: "preamble", raw: `Here is my proposal: {"summary":"ok"} hope it helps`, want: "ok"},
		{name: "garbage", raw: `not json at all`, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var p proposal
			err := parseJSON(tc.raw, &p)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("want error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if p.Summary != tc.want {
				t.Errorf("summary = %q, want %q", p.Summary, tc.want)
			}
		})
	}
}

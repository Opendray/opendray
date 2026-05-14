package projectdoc

import "testing"

func TestEmbedTextForLog(t *testing.T) {
	tests := []struct {
		name string
		in   LogEntry
		want string
	}{
		{"title only", LogEntry{Title: "M5 landed"}, "M5 landed"},
		{"content only", LogEntry{Content: "deploy.sh wired"}, "deploy.sh wired"},
		{"both joined", LogEntry{Title: "T", Content: "C"}, "T — C"},
		{"whitespace trimmed", LogEntry{Title: "  T  ", Content: "  C  "}, "T — C"},
		{"empty both", LogEntry{}, ""},
		{"whitespace only", LogEntry{Title: "  ", Content: "  "}, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := embedTextForLog(tc.in)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPgvecString(t *testing.T) {
	tests := []struct {
		name string
		in   []float32
		want string
	}{
		{"empty", nil, "[]"},
		{"single", []float32{0.5}, "[0.5]"},
		{"multiple", []float32{0.1, 0.2, 0.3}, "[0.1,0.2,0.3]"},
		{"negative", []float32{-1, 2.5}, "[-1,2.5]"},
		{"zero", []float32{0, 0}, "[0,0]"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := pgvecString(tc.in)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

package projectdoc

import "testing"

func TestMinedRemovals(t *testing.T) {
	prev := `# Infra
- opendray is the gateway
- foo is deprecated
- keep me

<!-- kb-sig:abc123 -->`

	tests := []struct {
		name string
		next string
		want []string // normalized forms expected removed
	}{
		{
			name: "one line deleted",
			next: "# Infra\n- opendray is the gateway\n- keep me\n",
			want: []string{"foo is deprecated"},
		},
		{
			name: "nothing deleted",
			next: prev,
			want: nil,
		},
		{
			// A line that only MOVED must not register as deleted.
			name: "reordered only",
			next: "# Infra\n- keep me\n- foo is deprecated\n- opendray is the gateway\n",
			want: nil,
		},
		{
			// Reformatting (bullet style, emphasis, case) is the same line.
			name: "reformatted line is not a removal",
			next: "# Infra\n* opendray is the gateway\n* **Foo** is deprecated\n* keep me\n",
			want: nil,
		},
		{
			// The sig marker vanishing (clients strip it) is machinery,
			// never an operator deletion.
			name: "sig marker stripped",
			next: "# Infra\n- opendray is the gateway\n- foo is deprecated\n- keep me\n",
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := minedRemovals(prev, tt.next)
			if len(got) != len(tt.want) {
				t.Fatalf("removed %d lines %v, want %d", len(got), got, len(tt.want))
			}
			for _, w := range tt.want {
				if _, ok := got[w]; !ok {
					t.Errorf("expected removal %q missing from %v", w, got)
				}
			}
		})
	}
}

func TestLocksDoc(t *testing.T) {
	for a, want := range map[Author]bool{
		AuthorOperator:   true,
		AuthorApproved:   true, // approval locks like a hand edit…
		AuthorAgent:      false,
		AuthorScanner:    false,
		AuthorSummarizer: false,
	} {
		if LocksDoc(a) != want {
			t.Errorf("LocksDoc(%q) = %v, want %v", a, !want, want)
		}
	}
}

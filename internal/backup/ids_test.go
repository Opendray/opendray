package backup

import (
	"strings"
	"testing"
)

func TestNewBackupID_FormatAndUniqueness(t *testing.T) {
	cases := []struct {
		name string
		fn   func() string
		want string // prefix
	}{
		{"backup", NewBackupID, "bk_"},
		{"export", NewExportID, "exp_"},
		{"schedule", NewScheduleID, "sch_"},
		{"target", NewTargetID, "tgt_"},
		{"import", NewImportID, "imp_"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			seen := make(map[string]struct{}, 1000)
			for i := 0; i < 1000; i++ {
				id := c.fn()
				if !strings.HasPrefix(id, c.want) {
					t.Fatalf("%s id %q missing prefix %q", c.name, id, c.want)
				}
				if got := id[len(c.want):]; len(got) != 22 {
					t.Errorf("%s id %q body len = %d, want 22", c.name, id, len(got))
				}
				if strings.ToLower(id) != id {
					t.Errorf("%s id %q is not lowercase", c.name, id)
				}
				if _, dup := seen[id]; dup {
					t.Fatalf("%s id %q collided after %d generations", c.name, id, i)
				}
				seen[id] = struct{}{}
			}
		})
	}
}

func TestNewDownloadToken_FormatAndUniqueness(t *testing.T) {
	seen := make(map[string]struct{}, 1000)
	for i := 0; i < 1000; i++ {
		tok := NewDownloadToken()
		if len(tok) != 22 {
			t.Fatalf("token %q len = %d, want 22", tok, len(tok))
		}
		if strings.ContainsAny(tok, "_") {
			t.Errorf("token %q should not contain _ (no prefix)", tok)
		}
		if _, dup := seen[tok]; dup {
			t.Fatalf("token %q collided after %d", tok, i)
		}
		seen[tok] = struct{}{}
	}
}

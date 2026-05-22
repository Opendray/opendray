package session

import (
	"slices"
	"testing"
)

func TestEnsureColorTerm(t *testing.T) {
	t.Run("injects defaults when absent", func(t *testing.T) {
		got := ensureColorTerm([]string{"PATH=/usr/bin", "HOME=/var/lib/opendray"})
		if !slices.Contains(got, "TERM=xterm-256color") {
			t.Errorf("TERM not injected: %v", got)
		}
		if !slices.Contains(got, "COLORTERM=truecolor") {
			t.Errorf("COLORTERM not injected: %v", got)
		}
	})

	t.Run("respects existing TERM and COLORTERM", func(t *testing.T) {
		in := []string{"TERM=screen-256color", "COLORTERM=24bit"}
		got := ensureColorTerm(slices.Clone(in))
		if !slices.Equal(got, in) {
			t.Errorf("should not override existing values: %v", got)
		}
	})

	t.Run("fills only the missing one", func(t *testing.T) {
		got := ensureColorTerm([]string{"TERM=vt100"})
		if slices.Contains(got, "TERM=xterm-256color") {
			t.Errorf("should not override existing TERM: %v", got)
		}
		if !slices.Contains(got, "COLORTERM=truecolor") {
			t.Errorf("COLORTERM should be injected: %v", got)
		}
	})
}

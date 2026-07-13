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

// A TUI picks a light or dark palette by asking the terminal. Two routes
// exist: the OSC 11 background query (xterm.js answers that already) and
// the COLORFGBG environment variable. opendray never set COLORFGBG, so a
// CLI that only reads the env (grok's `theme = "auto"`, vim, tmux, …) had
// no way to know the operator was in light mode and defaulted to dark.
func TestEnsureThemeEnv(t *testing.T) {
	base := []string{"PATH=/bin"}

	// COLORFGBG is "<fg>;<bg>" as colour indices; readers key off the
	// trailing background field (0 = dark, 15 = light).
	dark := ensureThemeEnv(base, "dark")
	if !slices.Contains(dark, "COLORFGBG=15;0") {
		t.Fatalf("dark theme should advertise a dark background, got %v", dark)
	}

	light := ensureThemeEnv(base, "light")
	if !slices.Contains(light, "COLORFGBG=0;15") {
		t.Fatalf("light theme should advertise a light background, got %v", light)
	}

	// Unknown/empty theme: say nothing, leave the CLI to its own default.
	for _, theme := range []string{"", "solarized"} {
		got := ensureThemeEnv(base, theme)
		if len(got) != len(base) {
			t.Fatalf("theme %q should not set COLORFGBG, got %v", theme, got)
		}
	}

	// An explicit COLORFGBG already in the environment wins.
	pinned := ensureThemeEnv([]string{"COLORFGBG=7;0"}, "light")
	if slices.Contains(pinned, "COLORFGBG=0;15") {
		t.Fatalf("an explicit COLORFGBG must not be overridden, got %v", pinned)
	}
}

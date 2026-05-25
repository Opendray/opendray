package session

import (
	"regexp"
	"strings"
)

// shell_chrome.go: light-touch cleanup of a *shell* (or other non-Claude)
// session's rendered screen snapshot.
//
// Claude Code has a busy TUI, so FilterClaudeChrome strips model bars,
// spinners, permission hints, and treats short / symbol-only lines as
// debris. None of that holds for a plain shell: there the terminal
// output *is* the payload. A command can legitimately print "42", "ok",
// or a "----" rule, and FilterClaudeChrome would delete all three.
//
// FilterShellChrome therefore removes almost nothing. It only:
//   - trims trailing whitespace per line (cosmetic),
//   - drops a trailing bare prompt the shell redraws while idle (it
//     carries no output), and
//   - collapses runs of blank lines so the snippet packs tightly.
//
// Used by recentResponseSnippet for every provider except claude, so a
// shell — and the codex / gemini screen-snapshot fallback — keeps its
// output intact instead of being run through Claude-tuned heuristics.

// barePromptLine matches a line that is nothing but a shell prompt
// sigil — "$", "#", "%", ">", "❯", or a 2-char arrow prompt like "❯❯".
// Capped at two chars so a longer run of symbols ("$$$", "==>") is
// treated as real output, not a prompt. A real command line always has
// text after the sigil, so this only catches the idle prompt left
// waiting for input.
var barePromptLine = regexp.MustCompile(`^[>$#%❯➜»]{1,2}$`)

// stripScreenChrome picks the right chrome filter for a provider's
// rendered terminal output: Claude's heavy TUI gets FilterClaudeChrome,
// everything else (shell, plus the codex / gemini snapshot fallback)
// gets the content-preserving FilterShellChrome.
func stripScreenChrome(provider, snapshot string) string {
	if provider == "claude" {
		return FilterClaudeChrome(snapshot)
	}
	return FilterShellChrome(snapshot)
}

// FilterShellChrome cleans a shell session's screen snapshot without
// discarding any actual output. See the file comment for the rationale.
func FilterShellChrome(s string) string {
	if s == "" {
		return ""
	}
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " \t")
	}

	// Drop trailing blank lines and a final bare prompt (the idle
	// "waiting for input" prompt the shell keeps redrawn at the bottom).
	for len(lines) > 0 {
		last := strings.TrimSpace(lines[len(lines)-1])
		if last == "" || barePromptLine.MatchString(last) {
			lines = lines[:len(lines)-1]
			continue
		}
		break
	}

	// Collapse runs of 2+ blank lines to a single blank.
	out := make([]string, 0, len(lines))
	prevBlank := false
	for _, l := range lines {
		if l == "" {
			if prevBlank {
				continue
			}
			prevBlank = true
		} else {
			prevBlank = false
		}
		out = append(out, l)
	}

	// Trim leading blank lines.
	for len(out) > 0 && out[0] == "" {
		out = out[1:]
	}
	return strings.Join(out, "\n")
}

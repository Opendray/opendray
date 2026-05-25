package session

import "testing"

func TestFilterShellChrome_PreservesShortAndSymbolOutput(t *testing.T) {
	// Lines the Claude filter would discard as "debris" (≤3 runes,
	// symbol-only) are legitimate shell output and must survive.
	in := "42\nok\n---\n$$$\n"
	got := FilterShellChrome(in)
	want := "42\nok\n---\n$$$"
	if got != want {
		t.Errorf("short/symbol output mangled\n got: %q\nwant: %q", got, want)
	}
}

func TestFilterShellChrome_DropsTrailingBarePrompt(t *testing.T) {
	cases := map[string]string{
		"build complete\n$":   "build complete",
		"done\n#\n":           "done",
		"output here\n❯ ":     "output here",
		"line\n%\n\n":         "line",
		"tail\nuser@host:~$ ": "tail\nuser@host:~$", // full prompt is NOT a bare sigil — kept
	}
	for in, want := range cases {
		if got := FilterShellChrome(in); got != want {
			t.Errorf("FilterShellChrome(%q)\n got: %q\nwant: %q", in, got, want)
		}
	}
}

func TestFilterShellChrome_CollapsesBlankRuns(t *testing.T) {
	in := "first\n\n\n\nsecond"
	if got, want := FilterShellChrome(in), "first\n\nsecond"; got != want {
		t.Errorf("blank-run collapse\n got: %q\nwant: %q", got, want)
	}
}

func TestFilterShellChrome_Empty(t *testing.T) {
	if got := FilterShellChrome(""); got != "" {
		t.Errorf("empty input should stay empty, got %q", got)
	}
}

func TestStripScreenChrome_RoutesByProvider(t *testing.T) {
	// A 2-char line: Claude treats it as debris and drops it; shell keeps it.
	snap := "ok"
	if got := stripScreenChrome("claude", snap); got != "" {
		t.Errorf("claude should strip the short debris line, got %q", got)
	}
	if got := stripScreenChrome("shell", snap); got != "ok" {
		t.Errorf("shell should preserve %q, got %q", snap, got)
	}
}

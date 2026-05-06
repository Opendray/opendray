package session

import (
	"strings"
	"testing"

	"github.com/hinshun/vt10x"
)

func TestStripANSI(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"no escapes", "hello world", "hello world"},
		{"SGR colour", "\x1b[31mred\x1b[0m text", "red text"},
		{"CSI cursor move", "before\x1b[2;3Hafter", "beforeafter"},
		{"OSC title", "\x1b]0;title\x07body", "body"},
		{"DCS terminated by ST", "x\x1bP1;2|abc\x1b\\y", "xy"},
		{"strip CR", "abc\rdef\n", "abcdef\n"},
		{"keep tab + LF", "a\tb\nc", "a\tb\nc"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := StripANSI(c.in); got != c.want {
				t.Errorf("StripANSI(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestStripBoxDrawing(t *testing.T) {
	in := "╭─[opendray]─╮\n│ hello   │\n╰──────────╯"
	out := stripBoxDrawing(in)
	if strings.ContainsAny(out, "╭╮╰╯─│") {
		t.Errorf("box-drawing residue: %q", out)
	}
	if !strings.Contains(out, "hello") {
		t.Errorf("content lost: %q", out)
	}
}

func TestTailLines(t *testing.T) {
	cases := []struct {
		in   string
		n    int
		want string
	}{
		{"one\ntwo\nthree\n", 2, "two\nthree"},
		{"one\ntwo", 5, "one\ntwo"},
		{"", 3, ""},
		{"single", 1, "single"},
		{"a\nb\nc\nd\ne", 3, "c\nd\ne"},
	}
	for _, c := range cases {
		if got := TailLines(c.in, c.n); got != c.want {
			t.Errorf("TailLines(%q, %d) = %q, want %q", c.in, c.n, got, c.want)
		}
	}
}

func TestScreenSnapshot_KeepsOnlyVisibleScreen(t *testing.T) {
	vt := vt10x.New(vt10x.WithSize(40, 6))
	// Simulate a TUI that clears + redraws several times so the raw
	// byte stream is full of garbage but the visible screen is clean.
	frames := []string{
		"first frame line 1\nfirst frame line 2\n",
		"\x1b[2J\x1b[H", // clear screen + home
		"\x1b[33mWaiting for input\x1b[0m\nReply with /resume\n",
		"\x1b[2J\x1b[H", // clear again
		"Final question:\nDo you approve? [y/N]\n",
	}
	for _, f := range frames {
		_, _ = vt.Write([]byte(f))
	}
	snap := ScreenSnapshot(vt)
	if !strings.Contains(snap, "Final question:") {
		t.Errorf("snapshot missing latest frame:\n%s", snap)
	}
	for _, garbage := range []string{"first frame", "Waiting for input", "Reply with /resume"} {
		if strings.Contains(snap, garbage) {
			t.Errorf("snapshot contains stale frame %q:\n%s", garbage, snap)
		}
	}
	if strings.Contains(snap, "\x1b") {
		t.Errorf("snapshot contains ANSI escapes:\n%q", snap)
	}
	// Trailing blank rows should be trimmed.
	if strings.HasSuffix(snap, "\n\n") {
		t.Errorf("snapshot has dangling blank rows: %q", snap)
	}
}

func TestScreenSnapshot_CollapsesBlankRuns(t *testing.T) {
	vt := vt10x.New(vt10x.WithSize(20, 10))
	_, _ = vt.Write([]byte("hello\n\n\n\n\n\nworld\n"))
	snap := ScreenSnapshot(vt)
	// Internal blank runs of 3+ should collapse to a single blank line.
	if strings.Contains(snap, "\n\n\n") {
		t.Errorf("snapshot kept >2 consecutive blank rows:\n%q", snap)
	}
	for _, want := range []string{"hello", "world"} {
		if !strings.Contains(snap, want) {
			t.Errorf("snapshot missing %q:\n%s", want, snap)
		}
	}
}

func TestCleanTerminalOutput_End2End(t *testing.T) {
	raw := "" +
		"\x1b[2J\x1b[H" + // clear screen + cursor home
		"╭──── Claude ────╮\n" +
		"│ hello world!   │\n" +
		"╰────────────────╯\n" +
		"\x1b[33m? Continue?\x1b[0m\n"
	out := CleanTerminalOutput(raw, 5)
	for _, want := range []string{"hello world!", "? Continue?"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "\x1b") || strings.Contains(out, "╭") {
		t.Errorf("residue in output:\n%s", out)
	}
}

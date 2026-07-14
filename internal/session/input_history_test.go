package session

import "testing"

// TestStripInputHistoryNoise pins that emulator back-channel sequences are
// removed from recorded input while real keystrokes — including arrow keys
// and bracketed paste — survive verbatim. The polluting shapes are the ones
// observed live in a real captured input_history.log (SGR mouse, focus, OSC).
func TestStripInputHistoryNoise(t *testing.T) {
	esc := "\x1b"
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain text untouched", "git status\n", "git status\n"},
		{"empty", "", ""},
		{"SGR mouse motion stripped", esc + "[<35;66;2M", ""},
		{"SGR mouse release stripped", esc + "[<0;12;7m", ""},
		{"focus in/out stripped", esc + "[I" + esc + "[O", ""},
		{"OSC color response (BEL) stripped", esc + "]11;rgb:0606/0707/0c0c\x07", ""},
		{"OSC color response (ST) stripped", esc + "]11;rgb:06/07/0c" + esc + "\\", ""},
		{"X10 mouse stripped", esc + "[Mabc", ""},           // ESC [ M + exactly 3 coord bytes "abc"
		{"X10 mouse then text", esc + "[Mabc" + "hi", "hi"}, // 3 coord bytes consumed, "hi" survives
		{
			name: "real keystrokes survive around mouse noise",
			in:   esc + "[<35;66;2M" + "ls -la\r" + esc + "[<35;65;2M",
			want: "ls -la\r",
		},
		{"arrow keys preserved", esc + "[A" + esc + "[B" + esc + "[C" + esc + "[D", esc + "[A" + esc + "[B" + esc + "[C" + esc + "[D"},
		{"Ctrl-C preserved", "\x03", "\x03"},
		{
			name: "bracketed paste markers + content preserved",
			in:   esc + "[200~pasted text" + esc + "[201~",
			want: esc + "[200~pasted text" + esc + "[201~",
		},
		{"unterminated OSC left intact", esc + "]11;rgb:incomplete", esc + "]11;rgb:incomplete"},
		{"truncated SGR mouse left intact", esc + "[<35;66", esc + "[<35;66"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := string(stripInputHistoryNoise([]byte(c.in)))
			if got != c.want {
				t.Errorf("stripInputHistoryNoise(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// TestStripInputHistoryNoiseRealSample feeds a burst shaped like the actual
// polluted capture (leading OSC + focus + a long run of SGR mouse motion,
// with one real command buried inside) and asserts only the command remains.
func TestStripInputHistoryNoiseRealSample(t *testing.T) {
	esc := "\x1b"
	sample := esc + "[O" +
		esc + "]11;rgb:0606/0707/0c0c\x07" +
		esc + "[<35;66;2M" + esc + "[<35;65;2M" + esc + "[<35;64;2M" +
		"make test\r" +
		esc + "[<35;61;2M" + esc + "[I"
	got := string(stripInputHistoryNoise([]byte(sample)))
	if got != "make test\r" {
		t.Errorf("real-sample filter = %q, want %q", got, "make test\r")
	}
}

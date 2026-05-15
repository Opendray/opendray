package session

import (
	"bytes"
	"testing"
)

// The headline regression: the Dart xterm fork emits "\x1b[?1;2c"
// in response to a Primary DA query (CSI c). That sequence used to
// reach Gemini's stdin, where its input parser ate the "\x1b[?"
// prefix but leaked the trailing "1;2c" into the visible prompt
// and entered a broken state that swallowed the next Enter.
func TestStripTerminalCapabilityResponses_PrimaryDA(t *testing.T) {
	in := []byte("\x1b[?1;2c")
	got := stripTerminalCapabilityResponses(in)
	if len(got) != 0 {
		t.Errorf("DA response not stripped; got %q (% x)", got, got)
	}
}

func TestStripTerminalCapabilityResponses_PrimaryDA_NoParams(t *testing.T) {
	// Parameter-less DA response — still matches the pattern.
	in := []byte("\x1b[?c")
	got := stripTerminalCapabilityResponses(in)
	if len(got) != 0 {
		t.Errorf("param-less DA not stripped; got %q", got)
	}
}

func TestStripTerminalCapabilityResponses_DA_EmbeddedInTypedInput(t *testing.T) {
	// xterm.js writes the DA response back to the PTY together
	// with any user input that arrived in the same WS frame. The
	// filter has to skip just the response, not the surrounding
	// text.
	in := []byte("hello\x1b[?1;2cworld\r")
	got := stripTerminalCapabilityResponses(in)
	want := []byte("helloworld\r")
	if !bytes.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestStripTerminalCapabilityResponses_CursorPositionReport(t *testing.T) {
	// CPR is the answer to "where is my cursor?" — same back-
	// channel concern as DA. xterm.js will send it whenever the
	// CLI asks. Without the '?' marker.
	in := []byte("\x1b[24;80R")
	got := stripTerminalCapabilityResponses(in)
	if len(got) != 0 {
		t.Errorf("CPR not stripped; got %q", got)
	}
}

func TestStripTerminalCapabilityResponses_StatusReport(t *testing.T) {
	// "Operating status ok"
	in := []byte("\x1b[0n")
	got := stripTerminalCapabilityResponses(in)
	if len(got) != 0 {
		t.Errorf("status report not stripped; got %q", got)
	}
}

func TestStripTerminalCapabilityResponses_PreservesUserInput(t *testing.T) {
	// Plain ASCII / UTF-8 — the fast path (no ESC) returns the
	// input slice as-is. Verify no allocation happens (same
	// backing slice).
	in := []byte("hello 世界\r")
	got := stripTerminalCapabilityResponses(in)
	if !bytes.Equal(got, in) {
		t.Errorf("plain input modified: got %q want %q", got, in)
	}
}

func TestStripTerminalCapabilityResponses_PreservesArrowKeys(t *testing.T) {
	// Arrow keys: ESC [ A/B/C/D. NOT capability responses — must
	// pass through so cursor navigation in interactive TUIs works.
	in := []byte("\x1b[A\x1b[B\x1b[C\x1b[D")
	got := stripTerminalCapabilityResponses(in)
	if !bytes.Equal(got, in) {
		t.Errorf("arrow keys stripped: got %q want %q", got, in)
	}
}

func TestStripTerminalCapabilityResponses_PreservesColorReset(t *testing.T) {
	// SGR reset — also CSI but ends in 'm', not c/R/n. Must keep.
	in := []byte("text\x1b[0m\r")
	got := stripTerminalCapabilityResponses(in)
	if !bytes.Equal(got, in) {
		t.Errorf("SGR stripped incorrectly: got %q want %q", got, in)
	}
}

func TestStripTerminalCapabilityResponses_PreservesLiteralBytes(t *testing.T) {
	// A user typing the four letters "1;2c" by hand (no leading
	// ESC) MUST survive — the filter only kicks in on well-formed
	// escape sequences.
	in := []byte("1;2c")
	got := stripTerminalCapabilityResponses(in)
	if !bytes.Equal(got, in) {
		t.Errorf("literal '1;2c' stripped: got %q", got)
	}
}

func TestStripTerminalCapabilityResponses_NoEscapeAtAll(t *testing.T) {
	// Sanity for the fast path.
	in := []byte("just a typed message\r")
	got := stripTerminalCapabilityResponses(in)
	if !bytes.Equal(got, in) {
		t.Errorf("fast path mutated input: got %q want %q", got, in)
	}
}

func TestStripTerminalCapabilityResponses_NilAndEmpty(t *testing.T) {
	if got := stripTerminalCapabilityResponses(nil); len(got) != 0 {
		t.Errorf("nil input: got %q", got)
	}
	if got := stripTerminalCapabilityResponses([]byte{}); len(got) != 0 {
		t.Errorf("empty input: got %q", got)
	}
}

func TestStripTerminalCapabilityResponses_MultipleResponses(t *testing.T) {
	// A burst can carry several responses in one frame — startup
	// queries often clump together. All of them must be removed.
	in := []byte("\x1b[?1;2c\x1b[24;80R\x1b[0n")
	got := stripTerminalCapabilityResponses(in)
	if len(got) != 0 {
		t.Errorf("multi-response burst not fully stripped; got %q", got)
	}
}

func TestStripTerminalCapabilityResponses_MalformedSequencePassesThrough(t *testing.T) {
	// A bare "ESC [" with no terminator could be a partial paste
	// or a malformed sequence — we leave it intact rather than
	// risk eating real user data. Likewise an ESC followed by a
	// non-'[' continuation byte.
	in := []byte("\x1b[\x1bX")
	got := stripTerminalCapabilityResponses(in)
	if !bytes.Equal(got, in) {
		t.Errorf("malformed bytes stripped: got %q want %q", got, in)
	}
}

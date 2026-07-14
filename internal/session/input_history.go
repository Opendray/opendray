package session

import "bytes"

// stripInputHistoryNoise removes machine-generated terminal back-channel
// sequences from a COPY of session input before it is recorded in the
// input-history ring used by context checkpoints. These bytes are valid
// input to a TUI — so they still reach the PTY untouched — but they are
// noise for a human or agent reading "what the operator actually did":
// mouse-motion reports dominate the stream and bury the real keystrokes.
//
// Stripped (all emulator-emitted, never typed by a human):
//
//	SGR mouse       ESC [ < <params> (M|m)
//	X10/UTF-8 mouse ESC [ M <3 coordinate bytes>
//	Focus in/out    ESC [ I  ,  ESC [ O
//	OSC response    ESC ] <...> (BEL | ESC \)      e.g. color-query answers
//
// Everything else is preserved verbatim: printable text, Enter / Ctrl keys,
// arrow-key CSI (ESC [ A..D), bracketed-paste markers and their content.
// Applied on top of stripTerminalCapabilityResponses, which already removed
// DA / CPR / DSR answers.
func stripInputHistoryNoise(data []byte) []byte {
	// Fast path: no ESC means nothing to strip; typed text is unaffected
	// and allocation-free.
	if !bytes.Contains(data, []byte{escByte}) {
		return data
	}
	out := make([]byte, 0, len(data))
	i := 0
	for i < len(data) {
		if data[i] == escByte {
			if end, ok := scanInputNoise(data, i); ok {
				i = end
				continue
			}
		}
		out = append(out, data[i])
		i++
	}
	return out
}

// scanInputNoise inspects a putative escape sequence at data[start] (where
// data[start]==ESC) and, when it matches one of the emulator back-channel
// shapes above, returns the index immediately after it and true. A partial
// or unrecognised sequence returns (start, false) so the caller emits the
// ESC normally — we never eat bytes we're unsure about.
func scanInputNoise(data []byte, start int) (int, bool) {
	if start+1 >= len(data) {
		return start, false
	}
	switch data[start+1] {
	case '[':
		i := start + 2
		if i >= len(data) {
			return start, false
		}
		switch data[i] {
		case '<':
			// SGR mouse: ESC [ < digits;digits;digits (M|m).
			j := i + 1
			for j < len(data) && ((data[j] >= '0' && data[j] <= '9') || data[j] == ';') {
				j++
			}
			if j < len(data) && (data[j] == 'M' || data[j] == 'm') {
				return j + 1, true
			}
			return start, false
		case 'M':
			// X10/UTF-8 mouse: ESC [ M followed by exactly 3 bytes. Only
			// strip when all 3 are present; a truncated tail is left intact.
			if i+3 < len(data) {
				return i + 4, true
			}
			return start, false
		case 'I', 'O':
			// Focus-in / focus-out. Distinct from arrow keys, which are
			// ESC [ A..D and fall through untouched.
			return i + 1, true
		}
		return start, false
	case ']':
		// OSC: ESC ] ... terminated by BEL (0x07) or ST (ESC \).
		j := start + 2
		for j < len(data) {
			if data[j] == 0x07 {
				return j + 1, true
			}
			if data[j] == escByte && j+1 < len(data) && data[j+1] == '\\' {
				return j + 2, true
			}
			j++
		}
		return start, false // unterminated: leave intact
	}
	return start, false
}

package session

import "bytes"

// stripTerminalCapabilityResponses removes well-known terminal-
// emulator capability answers from input destined for a CLI's
// stdin. These are auto-sent by xterm.js (and our Dart xterm fork)
// when the CLI emits a capability query — they're a protocol-level
// answer to a question the CLI asked, not user input.
//
// Patterns stripped:
//
//	Primary Device Attributes  ESC [ ? <digits;...> c     (response to "ESC [ c")
//	Cursor Position Report     ESC [ <row> ; <col> R      (response to "ESC [ 6 n")
//	Status Report              ESC [ <n> n                (response to "ESC [ 5 n")
//
// Why filter at the gateway instead of upstream-in-the-emulator:
//
//   - We have at least two terminal emulators (xterm.js for web,
//     our Dart fork for mobile). Patching the answer-emit logic in
//     both would still leave any future client unfixed; the PTY
//     boundary is a single chokepoint.
//   - These responses arrive as fast as the emulator can write
//     them — Gemini reliably enters a broken state because of one
//     extra burst of bytes at startup. Filtering at the Go layer
//     short-circuits the problem regardless of timing.
//
// Why this is safe:
//
//   - CLIs that genuinely want DA / CPR / Status data use these
//     queries as best-effort capability detection. When no answer
//     arrives within a short window they fall back to safe
//     defaults (xterm-compatible feature set). Claude and Codex
//     behave identically before and after this change.
//   - We never strip the user's literal bytes; only well-formed
//     answer patterns are removed. A user typing the four letters
//     "1;2c" by hand at the prompt is unaffected (no leading ESC).
func stripTerminalCapabilityResponses(data []byte) []byte {
	// Fast path: no ESC byte means there's no escape sequence to
	// strip. Avoids any allocation for typed input.
	if !bytes.Contains(data, []byte{escByte}) {
		return data
	}
	out := make([]byte, 0, len(data))
	i := 0
	for i < len(data) {
		if i+1 < len(data) && data[i] == escByte && data[i+1] == '[' {
			if end, ok := scanCapabilityResponse(data, i); ok {
				// Drop the whole sequence [i, end).
				i = end
				continue
			}
		}
		out = append(out, data[i])
		i++
	}
	return out
}

// scanCapabilityResponse inspects a putative CSI sequence starting
// at `data[start]` (where data[start]==ESC and data[start+1]=='[”)
// and returns the index of the byte immediately AFTER the sequence
// when it matches one of the recognised capability-answer shapes.
// Returns (end, true) on match, (start, false) otherwise — caller
// then emits the ESC byte normally and continues parsing.
//
// We only match when the sequence is well-formed (no embedded
// unexpected bytes); a malformed-looking sequence is left intact
// to avoid eating partial user input that happens to start with
// ESC [.
func scanCapabilityResponse(data []byte, start int) (int, bool) {
	// start indexes ESC. data[start+1]='['. Cursor: first byte
	// after the '['.
	i := start + 2
	if i >= len(data) {
		return start, false
	}
	// Optional '?' marks a DA response (private-mode CSI form).
	private := false
	if data[i] == '?' {
		private = true
		i++
	}
	// Parameter bytes: digits and semicolons.
	for i < len(data) {
		b := data[i]
		if (b >= '0' && b <= '9') || b == ';' {
			i++
			continue
		}
		break
	}
	if i >= len(data) {
		return start, false
	}
	terminator := data[i]
	switch {
	case private && terminator == 'c':
		// Primary Device Attributes response.
		return i + 1, true
	case !private && terminator == 'R':
		// Cursor Position Report. Without the '?' marker.
		return i + 1, true
	case !private && terminator == 'n':
		// Status Report (e.g. "ESC [ 0 n" = "operating status ok").
		return i + 1, true
	}
	return start, false
}

// escByte is the ASCII Escape control character. Named so callers
// don't have to remember 0x1B is what introduces a CSI sequence.
const escByte = 0x1B

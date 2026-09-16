package session

import "testing"

// The PTY must be created with a nonzero window size. Some CLIs gate
// their first render on terminal dimensions (grok's --minimal mode comes
// up blank at 0x0), so spawn() uses these as an initial floor before the
// client sends its real size. Guard against a regression to 0.
func TestDefaultPTYSizeNonZero(t *testing.T) {
	if defaultPTYCols == 0 || defaultPTYRows == 0 {
		t.Fatalf("default PTY size must be nonzero, got %dx%d", defaultPTYCols, defaultPTYRows)
	}
	// Sanity floor: a classic terminal is at least 80x24; anything
	// smaller risks TUIs mis-rendering their first frame.
	if defaultPTYCols < 80 || defaultPTYRows < 24 {
		t.Errorf("default PTY size %dx%d below the 80x24 floor", defaultPTYCols, defaultPTYRows)
	}
}

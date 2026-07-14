package session

import (
	"bytes"
	"testing"
)

// TestCheckpointContextInputRing verifies the manager surfaces a live
// session's cwd and the accumulated input-history tail (the data a context
// checkpoint records), and returns ok=false for unknown sessions.
func TestCheckpointContextInputRing(t *testing.T) {
	m := newBareManager()
	const id = "ses_cp"
	rs := &runningSession{
		sess:      Session{ID: id, State: StateRunning, Cwd: "/work/dir"},
		inputRing: NewRing(inputRingSize),
	}
	// Simulate operator input flowing through Manager.Input's ring write.
	_, _ = rs.inputRing.Write([]byte("git status\n"))
	_, _ = rs.inputRing.Write([]byte("make test\n"))
	m.sessions[id] = rs

	cwd, input, ok := m.CheckpointContext(id)
	if !ok {
		t.Fatal("CheckpointContext(live) ok=false, want true")
	}
	if cwd != "/work/dir" {
		t.Errorf("cwd = %q, want /work/dir", cwd)
	}
	if want := []byte("git status\nmake test\n"); !bytes.Equal(input, want) {
		t.Errorf("input history = %q, want %q", input, want)
	}

	if _, _, ok := m.CheckpointContext("ses_missing"); ok {
		t.Error("CheckpointContext(unknown) ok=true, want false")
	}
}

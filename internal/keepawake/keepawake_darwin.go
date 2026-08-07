//go:build darwin

package keepawake

import (
	"context"
	"os"
	"os/exec"
	"strconv"
)

// caffeinatePath is macOS's built-in power-assertion helper. Referenced
// by absolute path rather than via $PATH so a session-inherited PATH
// (the LaunchAgent copies the operator's) can't shadow it.
const caffeinatePath = "/usr/bin/caffeinate"

// newInhibitCmd builds the caffeinate invocation for a mode.
//
// The flags are load-bearing:
//
//	-s  assert against system sleep, but only while on wall power.
//	    Exactly the ModeAC contract — a desk-bound Mac never idles out,
//	    a laptop on battery is left alone.
//	-i  assert against idle sleep on any power source (ModeAlways).
//	-w  wait on a pid: caffeinate exits — releasing the assertion — as
//	    soon as that process is gone. Passing our own pid means a
//	    SIGKILLed gateway cannot strand the host in a permanently awake
//	    state, which is the failure mode that would make this feature
//	    worse than the bug it fixes.
//
// Neither flag blocks a deliberate sleep (lid close, Apple menu →
// Sleep), so the operator keeps the last word on their own machine.
func newInhibitCmd(ctx context.Context, mode Mode) *exec.Cmd {
	assert := "-s"
	if mode == ModeAlways {
		assert = "-i"
	}
	return exec.CommandContext(ctx, caffeinatePath,
		assert, "-w", strconv.Itoa(os.Getpid()))
}

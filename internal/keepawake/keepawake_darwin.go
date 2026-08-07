//go:build darwin

package keepawake

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
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
//	-i  assert against idle sleep on any power source (ModeAlways, and
//	    ModeOnDemand's holds — those exist only while a request or turn
//	    is live, and active use should finish regardless of the plug).
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
	if mode == ModeAlways || mode == ModeOnDemand {
		assert = "-i"
	}
	return exec.CommandContext(ctx, caffeinatePath,
		assert, "-w", strconv.Itoa(os.Getpid()))
}

// wakePreflight checks the one host setting on-demand mode cannot work
// without: "Wake for network access" (womp). With it off, a sleeping
// Mac never wakes for incoming traffic, so the gateway would simply be
// unreachable between activity bursts — silently worse than ModeAC. We
// can't fix it ourselves (pmset needs root), so warn loudly once.
var wakePreflight = func(ctx context.Context, log *slog.Logger) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "/usr/bin/pmset", "-g", "custom").Output()
	if err != nil {
		log.Debug("could not read pmset settings to verify wake-on-network", "err", err)
		return
	}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "womp" && fields[1] == "0" {
			log.Warn("Wake for network access is off, so a sleeping host cannot be " +
				"woken by phone or web traffic — on_demand mode will look like an " +
				"unreachable gateway. Enable it with: sudo pmset -a womp 1")
			return
		}
	}
}

// Package keepawake stops the host machine from idle-sleeping while the
// gateway is serving.
//
// opendray exists so an operator can reach their AI CLI sessions from a
// phone or another machine. That promise breaks the moment the host —
// typically a Mac sitting on a desk — decides it has been idle long
// enough to sleep: the network interface goes down, a remote Postgres
// becomes unreachable, and every phone/web request times out. Incoming
// traffic can dark-wake the machine, but the wake window is short and a
// user-session LaunchAgent is not reliably scheduled inside it, so the
// request has usually timed out before the listener runs again. The
// symptom reads as "the gateway is flaky", which sends people reading
// gateway code instead of `pmset -g log`.
//
// The fix is for the gateway to hold a power assertion for as long as it
// serves. On macOS that is /usr/bin/caffeinate, spawned as a child
// process rather than linked through IOKit: release binaries are
// cross-compiled from a Linux runner with CGO disabled, so an
// IOPMAssertionCreateWithName cgo call is not available to us.
// caffeinate ships with the base system, and its -w flag ties the
// assertion's lifetime to our pid — if opendray is SIGKILLed and never
// reaches its shutdown path, the assertion is still released.
//
// Every other platform is a no-op. A Linux or BSD gateway is a server
// that does not idle-suspend under any default install, and reaching
// for logind inhibitors there would be solving a problem nobody has.
package keepawake

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Mode selects how aggressively the gateway holds the host awake.
type Mode string

const (
	// ModeAC holds the host awake only while it runs on wall power, so a
	// laptop on battery still sleeps normally. The default: it fixes the
	// desk-bound gateway without quietly draining someone's MacBook.
	ModeAC Mode = "ac"

	// ModeAlways holds the host awake on any power source. For an
	// operator who genuinely wants their laptop reachable on battery.
	ModeAlways Mode = "always"

	// ModeOff never touches host power state. The gateway then stops
	// answering whenever the host sleeps — correct only if something
	// else already keeps the machine up.
	ModeOff Mode = "off"
)

// ParseMode normalises a configured value. Empty means "unset", which
// resolves to ModeAC.
func ParseMode(s string) (Mode, error) {
	switch Mode(strings.ToLower(strings.TrimSpace(s))) {
	case "", ModeAC:
		return ModeAC, nil
	case ModeAlways:
		return ModeAlways, nil
	case ModeOff:
		return ModeOff, nil
	default:
		return "", fmt.Errorf("unknown mode %q (want %q, %q or %q)",
			s, ModeAC, ModeAlways, ModeOff)
	}
}

const (
	// baseBackoff is the pause before re-spawning a helper that died.
	baseBackoff = 5 * time.Second
	// maxBackoff caps the exponential growth: even in a hard crash loop
	// we keep retrying, because the alternative is a gateway that
	// silently stops being reachable.
	maxBackoff = 5 * time.Minute
	// healthyRun is how long a helper must survive before its next exit
	// counts as a fresh failure rather than a continuing crash loop.
	healthyRun = time.Minute
)

// Keeper supervises the platform helper that holds the power assertion.
type Keeper struct {
	log  *slog.Logger
	mode Mode

	// newCmd builds the inhibitor process. It is nil on platforms with
	// no implementation, and is swapped out in tests.
	newCmd func(ctx context.Context, mode Mode) *exec.Cmd

	// Re-spawn pacing, fields rather than constants so tests can drive
	// the supervisor loop without sleeping for real seconds.
	baseBackoff time.Duration
	maxBackoff  time.Duration
}

// New returns a Keeper for the given mode. It starts nothing; call Run.
func New(log *slog.Logger, mode Mode) *Keeper {
	if log == nil {
		log = slog.Default()
	}
	return &Keeper{
		log:         log.With("component", "keepawake"),
		mode:        mode,
		newCmd:      newInhibitCmd,
		baseBackoff: baseBackoff,
		maxBackoff:  maxBackoff,
	}
}

// Run holds the assertion until ctx is cancelled, re-spawning the helper
// if it dies. It returns as soon as there is nothing to supervise, so
// callers can treat it as a plain background goroutine.
func (k *Keeper) Run(ctx context.Context) {
	if k.mode == ModeOff {
		k.log.Info("host idle-sleep inhibition disabled by config; " +
			"the gateway will stop answering whenever the host sleeps")
		return
	}
	if k.newCmd == nil {
		k.log.Debug("host idle-sleep inhibition is not implemented on this platform",
			"mode", string(k.mode))
		return
	}

	backoff := k.baseBackoff
	for ctx.Err() == nil {
		started := time.Now()
		err := k.runOnce(ctx)
		if ctx.Err() != nil {
			// Expected: shutdown killed the helper. The assertion is
			// released with it.
			k.log.Debug("stopped holding the host awake")
			return
		}
		// A missing helper is not going to appear on retry.
		if errors.Is(err, exec.ErrNotFound) || errors.Is(err, fs.ErrNotExist) {
			k.log.Warn("cannot hold the host awake: power-assertion helper not found. "+
				"The host may sleep out from under the gateway, making it unreachable "+
				"from phone and web until someone wakes it.", "err", err)
			return
		}
		lived := time.Since(started)
		if lived >= healthyRun {
			backoff = k.baseBackoff
		}
		k.log.Warn("power-assertion helper exited; restarting",
			"lived", lived.Round(time.Second), "retry_in", backoff, "err", err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		backoff = min(backoff*2, k.maxBackoff)
	}
}

// runOnce spawns the helper and blocks until it exits.
func (k *Keeper) runOnce(ctx context.Context) error {
	cmd := k.newCmd(ctx, k.mode)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", cmd.Path, err)
	}
	k.log.Info("holding the host awake while the gateway runs",
		"mode", string(k.mode), "helper_pid", cmd.Process.Pid)
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("%s exited: %w", filepath.Base(cmd.Path), err)
	}
	return nil
}

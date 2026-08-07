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
// The fix is for the gateway to hold a power assertion while it serves —
// either continuously (ModeAC / ModeAlways), or only while there is
// something to serve (ModeOnDemand). On-demand relies on macOS waking
// the machine when traffic arrives (Wake for network access programs the
// NIC to wake on TCP data to listening ports); the gateway's job is then
// to grab an assertion the moment a request or running session shows up,
// so the short dark-wake window is extended long enough to actually
// serve, and to let go once things quiet down so the host can sleep
// again. Sleep itself is never fought — only slept-through requests are.
//
// On macOS the assertion is /usr/bin/caffeinate, spawned as a child
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

	// ModeOnDemand lets the host sleep whenever the gateway is quiet and
	// holds it awake only while there is activity — in-flight requests,
	// attached live streams, or a session mid-turn. Reachability while
	// asleep depends on the host waking for incoming traffic ("Wake for
	// network access" on a wired interface, or a Bonjour sleep proxy);
	// the on-demand keeper then extends that wake for as long as serving
	// takes, plus a short linger.
	ModeOnDemand Mode = "on_demand"

	// ModeOff never touches host power state. The gateway then stops
	// answering whenever the host sleeps — correct only if something
	// else already keeps the machine up.
	ModeOff Mode = "off"
)

// ParseMode normalises a configured value. Empty means "unset", which
// resolves to ModeAC.
func ParseMode(s string) (Mode, error) {
	norm := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(s)), "-", "_")
	switch Mode(norm) {
	case "", ModeAC:
		return ModeAC, nil
	case ModeAlways:
		return ModeAlways, nil
	case ModeOnDemand:
		return ModeOnDemand, nil
	case ModeOff:
		return ModeOff, nil
	default:
		return "", fmt.Errorf("unknown mode %q (want %q, %q, %q or %q)",
			s, ModeAC, ModeAlways, ModeOnDemand, ModeOff)
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

	// pollInterval is how often on-demand mode re-samples the activity
	// signal.
	pollInterval = 15 * time.Second
	// lingerAfterIdle is how long on-demand mode keeps holding the
	// assertion after activity last read true. Long enough that a user
	// tapping around a phone UI doesn't race the host back to sleep
	// between screens; short enough that a quiet gateway lets the host
	// sleep within a few minutes.
	lingerAfterIdle = 2 * time.Minute
)

// Keeper supervises the platform helper that holds the power assertion.
type Keeper struct {
	log  *slog.Logger
	mode Mode

	// newCmd builds the inhibitor process. It is nil on platforms with
	// no implementation, and is swapped out in tests.
	newCmd func(ctx context.Context, mode Mode) *exec.Cmd

	// active reports whether the gateway is currently doing something a
	// sleeping host would break. Only consulted by ModeOnDemand.
	active func() bool

	// Pacing knobs, fields rather than constants so tests can drive
	// the loops without sleeping for real seconds.
	baseBackoff time.Duration
	maxBackoff  time.Duration
	poll        time.Duration
	linger      time.Duration
}

// Option customises a Keeper.
type Option func(*Keeper)

// WithActivity supplies the activity signal ModeOnDemand holds the host
// awake for. The func must be cheap and safe to call from another
// goroutine; it is sampled every poll interval.
func WithActivity(f func() bool) Option {
	return func(k *Keeper) { k.active = f }
}

// New returns a Keeper for the given mode. It starts nothing; call Run.
func New(log *slog.Logger, mode Mode, opts ...Option) *Keeper {
	if log == nil {
		log = slog.Default()
	}
	k := &Keeper{
		log:         log.With("component", "keepawake"),
		mode:        mode,
		newCmd:      newInhibitCmd,
		baseBackoff: baseBackoff,
		maxBackoff:  maxBackoff,
		poll:        pollInterval,
		linger:      lingerAfterIdle,
	}
	for _, opt := range opts {
		opt(k)
	}
	return k
}

// Run enforces the configured mode until ctx is cancelled. It returns as
// soon as there is nothing to supervise, so callers can treat it as a
// plain background goroutine.
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
	if k.mode == ModeOnDemand {
		if k.active == nil {
			// No activity signal means on-demand can never assert, which
			// would silently be ModeOff. Fail towards reachability.
			k.log.Warn("on_demand mode has no activity signal; " +
				"holding the host awake continuously instead")
			k.supervise(ctx)
			return
		}
		k.runOnDemand(ctx)
		return
	}
	k.supervise(ctx)
}

// runOnDemand lets the host sleep while the gateway is quiet and holds
// the assertion only across bursts of activity. The host waking at all
// when traffic hits a sleeping machine is the OS's job (Wake for network
// access); ours is to keep it up long enough to serve.
func (k *Keeper) runOnDemand(ctx context.Context) {
	if wakePreflight != nil {
		wakePreflight(ctx, k.log)
	}
	k.log.Info("holding the host awake on demand: it may sleep while the gateway is quiet",
		"poll", k.poll, "linger", k.linger)
	for ctx.Err() == nil {
		if !k.waitForActivity(ctx) {
			return
		}
		holdCtx, cancel := context.WithCancel(ctx)
		superviseDone := make(chan bool, 1)
		go func() {
			superviseDone <- k.supervise(holdCtx)
			// A supervisor that gave up permanently must also end the
			// hold below, or Run would block on a signal that stays
			// active with nothing left to supervise.
			cancel()
		}()
		k.holdWhileActive(holdCtx)
		cancel()
		if gaveUp := <-superviseDone; gaveUp {
			return
		}
	}
}

// waitForActivity blocks until the activity signal reads true. It
// returns false when ctx ended first.
func (k *Keeper) waitForActivity(ctx context.Context) bool {
	if k.active() {
		return true
	}
	ticker := time.NewTicker(k.poll)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			if k.active() {
				return true
			}
		}
	}
}

// holdWhileActive blocks while the activity signal stays warm, returning
// once it has read false continuously for the linger window (or ctx
// ended).
func (k *Keeper) holdWhileActive(ctx context.Context) {
	var idleSince time.Time
	ticker := time.NewTicker(k.poll)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if k.active() {
				idleSince = time.Time{}
				continue
			}
			if idleSince.IsZero() {
				idleSince = time.Now()
				continue
			}
			if time.Since(idleSince) >= k.linger {
				return
			}
		}
	}
}

// supervise holds the assertion until ctx is cancelled, re-spawning the
// helper if it dies. It reports true when the helper can never run
// (missing binary) so on-demand mode stops re-attempting it forever.
func (k *Keeper) supervise(ctx context.Context) (gaveUp bool) {
	backoff := k.baseBackoff
	for ctx.Err() == nil {
		started := time.Now()
		err := k.runOnce(ctx)
		if ctx.Err() != nil {
			// Expected: shutdown (or the end of an on-demand hold)
			// killed the helper. The assertion is released with it.
			k.log.Debug("stopped holding the host awake")
			return false
		}
		// A missing helper is not going to appear on retry.
		if errors.Is(err, exec.ErrNotFound) || errors.Is(err, fs.ErrNotExist) {
			k.log.Warn("cannot hold the host awake: power-assertion helper not found. "+
				"The host may sleep out from under the gateway, making it unreachable "+
				"from phone and web until someone wakes it.", "err", err)
			return true
		}
		lived := time.Since(started)
		if lived >= healthyRun {
			backoff = k.baseBackoff
		}
		k.log.Warn("power-assertion helper exited; restarting",
			"lived", lived.Round(time.Second), "retry_in", backoff, "err", err)
		select {
		case <-ctx.Done():
			return false
		case <-time.After(backoff):
		}
		backoff = min(backoff*2, k.maxBackoff)
	}
	return false
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

package keepawake

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

func TestRun_OnDemandHoldsOnlyWhileActive(t *testing.T) {
	var active atomic.Bool
	var spawns atomic.Int32
	pidFile := filepath.Join(t.TempDir(), "helper.pid")

	k := New(quietLogger(), ModeOnDemand, WithActivity(active.Load))
	k.baseBackoff = time.Millisecond
	k.maxBackoff = 5 * time.Millisecond
	k.poll = 2 * time.Millisecond
	k.linger = 10 * time.Millisecond
	k.newCmd = testHelperCmd(pidFile, &spawns)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		k.Run(ctx)
		close(done)
	}()

	// Quiet gateway: no helper may be spawned.
	time.Sleep(50 * time.Millisecond)
	if got := spawns.Load(); got != 0 {
		t.Fatalf("helper spawned %d times while inactive, want 0", got)
	}

	// Activity shows up: the helper must start and stay up.
	active.Store(true)
	pid := waitForHelperPid(t, pidFile)
	if err := syscall.Kill(pid, 0); err != nil {
		t.Fatalf("helper pid %d not alive while active: %v", pid, err)
	}

	// Activity ends: after the linger window the helper must be gone,
	// releasing the assertion so the host can sleep again.
	active.Store(false)
	waitForProcessGone(t, pid)

	// Activity returns: a fresh hold must start a fresh helper.
	if err := os.Remove(pidFile); err != nil {
		t.Fatalf("remove pid file: %v", err)
	}
	active.Store(true)
	waitForHelperPid(t, pidFile)
	if got := spawns.Load(); got < 2 {
		t.Fatalf("helper spawned %d times after re-activation, want >= 2", got)
	}

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after cancel")
	}
}

// A cancel that lands while on-demand mode is idling (no helper up) must
// end Run promptly, not leave a goroutine polling forever.
func TestRun_OnDemandReturnsOnCancelWhileQuiet(t *testing.T) {
	var active atomic.Bool // stays false
	var spawns atomic.Int32
	k := New(quietLogger(), ModeOnDemand, WithActivity(active.Load))
	k.poll = 2 * time.Millisecond
	k.linger = 10 * time.Millisecond
	k.newCmd = testHelperCmd(filepath.Join(t.TempDir(), "helper.pid"), &spawns)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		k.Run(ctx)
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after cancel while quiet")
	}
	if got := spawns.Load(); got != 0 {
		t.Fatalf("helper spawned %d times, want 0", got)
	}
}

// A missing helper binary is permanent; on-demand mode must stop trying
// rather than re-attempt it on every activity burst forever.
func TestRun_OnDemandGivesUpWhenHelperIsMissing(t *testing.T) {
	var active atomic.Bool
	active.Store(true)
	var spawns atomic.Int32
	k := New(quietLogger(), ModeOnDemand, WithActivity(active.Load))
	k.baseBackoff = time.Millisecond
	k.maxBackoff = 5 * time.Millisecond
	k.poll = 2 * time.Millisecond
	k.linger = 10 * time.Millisecond
	k.newCmd = func(ctx context.Context, _ Mode) *exec.Cmd {
		spawns.Add(1)
		return exec.CommandContext(ctx, "/nonexistent/opendray-keepawake-helper")
	}

	done := make(chan struct{})
	go func() {
		k.Run(context.Background())
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run kept retrying a helper that does not exist")
	}
	if got := spawns.Load(); got != 1 {
		t.Fatalf("attempted the missing helper %d times, want exactly 1", got)
	}
}

// On-demand without an activity signal must fall back to holding the
// host awake continuously — failing towards reachability, not ModeOff.
func TestRun_OnDemandWithoutSignalHoldsContinuously(t *testing.T) {
	var spawns atomic.Int32
	k := newTestKeeper(t, ModeOnDemand, "sleep 30", &spawns)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		k.Run(ctx)
		close(done)
	}()
	deadline := time.After(2 * time.Second)
	for spawns.Load() == 0 {
		select {
		case <-deadline:
			t.Fatal("helper never spawned without an activity signal")
		case <-time.After(time.Millisecond):
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after cancel")
	}
}

func TestTracker_ActiveDuringAndShortlyAfterRequests(t *testing.T) {
	tr := NewTracker()
	tr.recent = 80 * time.Millisecond
	if tr.Active() {
		t.Fatal("a tracker that has never served must be inactive")
	}

	release := make(chan struct{})
	handled := make(chan struct{})
	h := tr.Wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		close(handled)
		<-release
	}))
	go h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	<-handled
	if !tr.Active() {
		t.Fatal("tracker must be active while a request is in flight")
	}

	close(release)
	// Immediately after completion the recent window keeps it warm.
	if !tr.Active() {
		t.Fatal("tracker must stay active inside the recent window")
	}
	deadline := time.After(2 * time.Second)
	for tr.Active() {
		select {
		case <-deadline:
			t.Fatal("tracker never went inactive after the recent window")
		case <-time.After(5 * time.Millisecond):
		}
	}
}

// testHelperCmd builds a newCmd whose process records its pid, so tests
// can verify the assertion-holder's actual lifetime.
func testHelperCmd(pidFile string, spawns *atomic.Int32) func(context.Context, Mode) *exec.Cmd {
	script := "echo $$ > " + pidFile + " && sleep 30"
	return func(ctx context.Context, _ Mode) *exec.Cmd {
		spawns.Add(1)
		return exec.CommandContext(ctx, "/bin/sh", "-c", script)
	}
}

func waitForHelperPid(t *testing.T, pidFile string) int {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		if raw, err := os.ReadFile(pidFile); err == nil {
			if pid, err := strconv.Atoi(strings.TrimSpace(string(raw))); err == nil && pid > 0 {
				return pid
			}
		}
		select {
		case <-deadline:
			t.Fatal("helper never wrote its pid file")
		case <-time.After(time.Millisecond):
		}
	}
}

func waitForProcessGone(t *testing.T, pid int) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for syscall.Kill(pid, 0) == nil {
		select {
		case <-deadline:
			t.Fatalf("helper pid %d still alive after activity ended", pid)
		case <-time.After(time.Millisecond):
		}
	}
}

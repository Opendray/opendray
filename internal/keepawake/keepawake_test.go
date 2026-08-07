package keepawake

import (
	"context"
	"io"
	"log/slog"
	"os/exec"
	"sync/atomic"
	"testing"
	"time"
)

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// newTestKeeper builds a Keeper whose helper is a plain /bin/sh command,
// so the supervisor loop can be exercised on any POSIX host — CI runs
// Linux, where the real newInhibitCmd is nil by design.
func newTestKeeper(t *testing.T, mode Mode, script string, spawns *atomic.Int32) *Keeper {
	t.Helper()
	k := New(quietLogger(), mode)
	k.baseBackoff = time.Millisecond
	k.maxBackoff = 5 * time.Millisecond
	k.newCmd = func(ctx context.Context, _ Mode) *exec.Cmd {
		if spawns != nil {
			spawns.Add(1)
		}
		return exec.CommandContext(ctx, "/bin/sh", "-c", script)
	}
	return k
}

func TestParseMode(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    Mode
		wantErr bool
	}{
		{"empty defaults to ac", "", ModeAC, false},
		{"explicit ac", "ac", ModeAC, false},
		{"always", "always", ModeAlways, false},
		{"off", "off", ModeOff, false},
		{"case and space insensitive", "  Always ", ModeAlways, false},
		{"unknown value", "sometimes", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseMode(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseMode(%q) err = %v, wantErr = %v", tt.in, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("ParseMode(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// Run must be safe to call as a bare goroutine on every configuration
// that has nothing to supervise — it returns instead of spinning.
func TestRun_ReturnsWhenThereIsNothingToSupervise(t *testing.T) {
	tests := []struct {
		name   string
		keeper func() *Keeper
	}{
		{
			name: "mode off",
			keeper: func() *Keeper {
				return newTestKeeper(t, ModeOff, "sleep 30", nil)
			},
		},
		{
			name: "platform without an implementation",
			keeper: func() *Keeper {
				k := New(quietLogger(), ModeAC)
				k.newCmd = nil
				return k
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			done := make(chan struct{})
			go func() {
				tt.keeper().Run(context.Background())
				close(done)
			}()
			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Fatal("Run did not return")
			}
		})
	}
}

func TestRun_RestartsHelperThatExitsEarly(t *testing.T) {
	var spawns atomic.Int32
	k := newTestKeeper(t, ModeAC, "exit 1", &spawns)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		k.Run(ctx)
		close(done)
	}()

	deadline := time.After(5 * time.Second)
	for spawns.Load() < 3 {
		select {
		case <-deadline:
			t.Fatalf("helper respawned only %d times, want >= 3", spawns.Load())
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

func TestRun_StopsAndReleasesOnCancel(t *testing.T) {
	var spawns atomic.Int32
	k := newTestKeeper(t, ModeAC, "sleep 30", &spawns)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		k.Run(ctx)
		close(done)
	}()

	// Wait for the helper to actually be running before cancelling, so
	// this exercises the kill path rather than a pre-start return.
	deadline := time.After(2 * time.Second)
	for spawns.Load() == 0 {
		select {
		case <-deadline:
			t.Fatal("helper never spawned")
		case <-time.After(time.Millisecond):
		}
	}
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after cancel")
	}
	if got := spawns.Load(); got != 1 {
		t.Fatalf("helper spawned %d times, want exactly 1 (no respawn after cancel)", got)
	}
}

// A helper binary that isn't there will never appear, so Run must give
// up rather than retry forever.
func TestRun_GivesUpWhenHelperIsMissing(t *testing.T) {
	var spawns atomic.Int32
	k := New(quietLogger(), ModeAC)
	k.baseBackoff = time.Millisecond
	k.maxBackoff = 5 * time.Millisecond
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

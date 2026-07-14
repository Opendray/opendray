package session

import (
	"errors"
	"testing"
)

// TestNextMatrix is the frozen State x Event -> State transition matrix
// from docs/design/session-state-machine.md. Every legal cell and every
// rejected illegal cell is pinned here; new lifecycle code MUST route
// through Next and keep this matrix green. `-` means illegal (Next returns
// ErrIllegalTransition and leaves the State unchanged).
func TestNextMatrix(t *testing.T) {
	const illegal = State("-")

	events := []Event{
		EventStart, EventIdle, EventResume,
		EventUserStop, EventExit, EventGatewayShutdown,
	}

	// want[from][event]; illegal marks a rejected transition.
	want := map[State]map[Event]State{
		// Zero State: only a spawn is legal.
		"": {
			EventStart: StateRunning, EventIdle: illegal, EventResume: illegal,
			EventUserStop: illegal, EventExit: illegal, EventGatewayShutdown: illegal,
		},
		StatePending: {
			EventStart: illegal, EventIdle: illegal, EventResume: illegal,
			EventUserStop: StateStopped, EventExit: StateEnded, EventGatewayShutdown: StateInterrupted,
		},
		StateRunning: {
			EventStart: illegal, EventIdle: StateIdle, EventResume: StateRunning,
			EventUserStop: StateStopped, EventExit: StateEnded, EventGatewayShutdown: StateInterrupted,
		},
		StateIdle: {
			EventStart: illegal, EventIdle: StateIdle, EventResume: StateRunning,
			EventUserStop: StateStopped, EventExit: StateEnded, EventGatewayShutdown: StateInterrupted,
		},
		// Terminal States: only a restart re-enters running; the three
		// termination events are idempotent no-ops preserving the State.
		StateStopped: {
			EventStart: StateRunning, EventIdle: illegal, EventResume: illegal,
			EventUserStop: StateStopped, EventExit: StateStopped, EventGatewayShutdown: StateStopped,
		},
		StateEnded: {
			EventStart: StateRunning, EventIdle: illegal, EventResume: illegal,
			EventUserStop: StateEnded, EventExit: StateEnded, EventGatewayShutdown: StateEnded,
		},
		StateInterrupted: {
			EventStart: StateRunning, EventIdle: illegal, EventResume: illegal,
			EventUserStop: StateInterrupted, EventExit: StateInterrupted, EventGatewayShutdown: StateInterrupted,
		},
	}

	for from, row := range want {
		for _, ev := range events {
			expected := row[ev]
			got, err := Next(from, ev)
			if expected == illegal {
				if !errors.Is(err, ErrIllegalTransition) {
					t.Errorf("Next(%q, %q): want ErrIllegalTransition, got (%q, %v)", from, ev, got, err)
				}
				// Illegal transitions must not change the State.
				if got != from {
					t.Errorf("Next(%q, %q): illegal transition must return unchanged State, got %q", from, ev, got)
				}
				if CanTransition(from, ev) {
					t.Errorf("CanTransition(%q, %q) = true, want false", from, ev)
				}
				continue
			}
			if err != nil {
				t.Errorf("Next(%q, %q): unexpected error %v (want -> %q)", from, ev, err, expected)
				continue
			}
			if got != expected {
				t.Errorf("Next(%q, %q) = %q, want %q", from, ev, got, expected)
			}
			if !CanTransition(from, ev) {
				t.Errorf("CanTransition(%q, %q) = false, want true", from, ev)
			}
		}
	}
}

// TestNextIdempotent covers the highest-risk operational hazards the
// design calls out: a double Stop, repeat exit-detector wakeups, and a
// gateway shutdown racing a session that already exited. Each must be a
// harmless no-op, never a state flip.
func TestNextIdempotent(t *testing.T) {
	cases := []struct {
		name  string
		from  State
		event Event
	}{
		{"double user stop", StateStopped, EventUserStop},
		{"repeat exit wakeup", StateEnded, EventExit},
		{"shutdown after user stop keeps stopped", StateStopped, EventGatewayShutdown},
		{"exit after user stop keeps stopped", StateStopped, EventExit},
		{"shutdown after natural end keeps ended", StateEnded, EventGatewayShutdown},
		{"resume while running is a no-op", StateRunning, EventResume},
		{"idle while idle is a no-op", StateIdle, EventIdle},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Next(c.from, c.event)
			if err != nil {
				t.Fatalf("Next(%q, %q) errored: %v", c.from, c.event, err)
			}
			if got != c.from {
				t.Errorf("Next(%q, %q) = %q, want unchanged %q", c.from, c.event, got, c.from)
			}
		})
	}
}

// TestNextRejectsRestartOfLiveSession pins the guard that a session which
// is already running/idle cannot be spawned again — the SSOT form of
// ErrAlreadyRunning, and the guard that stops two resumes racing for the
// same cwd.
func TestNextRejectsRestartOfLiveSession(t *testing.T) {
	for _, from := range []State{StateRunning, StateIdle} {
		if _, err := Next(from, EventStart); !errors.Is(err, ErrIllegalTransition) {
			t.Errorf("Next(%q, EventStart): want ErrIllegalTransition, got %v", from, err)
		}
	}
}

// TestStartLegalIffTerminal pins the invariant that makes routing
// Manager.Start's guard through CanTransition behaviour-identical to the
// historical `!state.IsTerminal()` check: EventStart is legal from exactly
// the terminal States (restart/resume), and rejected from every live
// State (pending/running/idle) — the SSOT form of ErrAlreadyRunning and
// the guard against two resumes racing the same cwd.
func TestStartLegalIffTerminal(t *testing.T) {
	for _, s := range []State{StatePending, StateRunning, StateIdle, StateStopped, StateEnded, StateInterrupted} {
		if got, want := CanTransition(s, EventStart), s.IsTerminal(); got != want {
			t.Errorf("CanTransition(%q, EventStart) = %v, want IsTerminal() = %v", s, got, want)
		}
	}
}

// TestTerminationEventPrecedence pins that TerminationEvent reproduces the
// historical classifyExitState precedence, and that feeding the resulting
// Event through Next from a running session reaches the same terminal
// State classifyExitState would have chosen. This keeps the new SSOT and
// the legacy classifier in lock-step during incremental adoption.
func TestTerminationEventPrecedence(t *testing.T) {
	cases := []struct {
		name          string
		stop, closing bool
		wantEvent     Event
		wantState     State
	}{
		{"user stop beats shutdown", true, true, EventUserStop, StateStopped},
		{"user stop", true, false, EventUserStop, StateStopped},
		{"shutdown -> interrupted", false, true, EventGatewayShutdown, StateInterrupted},
		{"spontaneous exit -> ended", false, false, EventExit, StateEnded},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ev := TerminationEvent(c.stop, c.closing)
			if ev != c.wantEvent {
				t.Fatalf("TerminationEvent(%v,%v) = %q, want %q", c.stop, c.closing, ev, c.wantEvent)
			}
			got, err := Next(StateRunning, ev)
			if err != nil {
				t.Fatalf("Next(running, %q) errored: %v", ev, err)
			}
			if got != c.wantState {
				t.Errorf("Next(running, %q) = %q, want %q", ev, got, c.wantState)
			}
			// Must agree with the legacy classifier still in session.go.
			if legacy := classifyExitState(c.stop, c.closing); legacy != c.wantState {
				t.Errorf("classifyExitState(%v,%v) = %q, diverged from matrix %q", c.stop, c.closing, legacy, c.wantState)
			}
		})
	}
}

// TestClassifyInterrupt freezes the interrupted sub-state classifier: the
// three recovery classes are derived from observed facts (process
// liveness, gateway restart), not from stale DB intent.
func TestClassifyInterrupt(t *testing.T) {
	cases := []struct {
		name             string
		procAlive        bool
		gatewayRestarted bool
		wantReason       InterruptReason
		wantRecovery     RecoveryStrategy
	}{
		{"WS drop, process healthy", true, false, InterruptDisconnected, RecoveryWaitReattach},
		{"gateway restart, orphan alive", true, true, InterruptOrphaned, RecoveryAdoptPID},
		{"process dead", false, false, InterruptCrashed, RecoveryRollback},
		{"process dead beats gateway restart", false, true, InterruptCrashed, RecoveryRollback},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotReason := ClassifyInterrupt(c.procAlive, c.gatewayRestarted)
			if gotReason != c.wantReason {
				t.Errorf("ClassifyInterrupt(%v,%v) = %q, want %q", c.procAlive, c.gatewayRestarted, gotReason, c.wantReason)
			}
			if gotRecovery := gotReason.Recovery(); gotRecovery != c.wantRecovery {
				t.Errorf("%q.Recovery() = %q, want %q", gotReason, gotRecovery, c.wantRecovery)
			}
		})
	}
}

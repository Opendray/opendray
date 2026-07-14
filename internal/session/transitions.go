package session

import "errors"

// This file is the single source of truth for the session lifecycle
// state machine. It encodes the transition matrix agreed in the
// 2026-07 state-machine hardening design (docs/design/session-state-machine.md):
// every (State, Event) pair either maps to a well-defined next State or
// is rejected as an illegal transition.
//
// It is intentionally PURE and side-effect free — no DB, no PTY, no
// locks — so it can be exhaustively table-tested and adopted incrementally
// by the Manager/store as the authoritative guard. Persisted states are
// unchanged (session.go still owns the State enum and the sessions.state
// column); the three interrupted sub-states are modelled here as a
// refinement layer (InterruptReason), not as new persisted enum values.

// Event enumerates the lifecycle signals that drive a session between
// States. A transition is legal iff Next(state, event) returns a nil
// error. See docs/design/session-state-machine.md for the full matrix.
type Event string

const (
	// EventStart — a PTY is (re)spawned for a row that is not currently
	// running: initial Create, operator Restart, or startup auto-resume.
	// Legal from the zero State ("", fresh) and any terminal State;
	// rejected from running/idle (that is ErrAlreadyRunning).
	EventStart Event = "start"
	// EventIdle — the idle detector observed no activity past threshold.
	EventIdle Event = "idle"
	// EventResume — input/output activity resumed a previously idle
	// session (a no-op while already running).
	EventResume Event = "resume"
	// EventUserStop — the operator explicitly stopped the session
	// (Manager.Stop / DELETE-as-stop). Highest precedence terminator.
	EventUserStop Event = "user_stop"
	// EventExit — the CLI process exited on its own (clean exit or a
	// spontaneous crash of the child).
	EventExit Event = "exit"
	// EventGatewayShutdown — the gateway process itself is exiting, so
	// the PTY dies with it. Distinct from EventExit so reconciliation can
	// tell "the daemon killed this" apart from "the agent exited" and
	// auto-resume it.
	EventGatewayShutdown Event = "gateway_shutdown"
)

// ErrIllegalTransition is returned by Next when the (State, Event) pair
// is not a permitted transition (e.g. ended -> running via EventIdle, or
// EventStart on an already-running session).
var ErrIllegalTransition = errors.New("illegal session state transition")

// terminationEvents are the three ways a live session can leave the
// running set. Applied to an already-terminal State they are idempotent
// no-ops (the State is preserved), mirroring the store's
// `state NOT IN ('stopped','ended')` guard: a user stop is never
// overwritten by a later gateway shutdown, etc.
var terminationEvents = map[Event]State{
	EventUserStop:        StateStopped,
	EventExit:            StateEnded,
	EventGatewayShutdown: StateInterrupted,
}

// liveTransitions is the transition table for the non-terminal States
// (plus the zero State, which represents a not-yet-persisted session).
// A missing (State, Event) entry is an illegal transition, UNLESS the
// State is terminatable (see terminatableStates), in which case the
// termination events fold in from terminationEvents so the precedence
// lives in one place.
var liveTransitions = map[State]map[Event]State{
	// Zero State: a fresh session that has not been spawned yet. It is
	// NOT terminatable — you cannot stop/exit/interrupt a session that
	// never started — so only a spawn is legal here.
	"": {
		EventStart: StateRunning,
	},
	// StatePending: a spawn is already in flight. It is terminatable
	// (a mid-spawn session can still be stopped / die / be interrupted)
	// but NOT startable — a second start would race the first spawn for
	// the same cwd. Only running/terminal rows are restartable.
	StatePending: {},
	StateRunning: {
		EventIdle:   StateIdle,
		EventResume: StateRunning, // no-op: already running
	},
	StateIdle: {
		EventIdle:   StateIdle, // no-op: already idle
		EventResume: StateRunning,
	},
}

// terminatableStates are the live States that hold a real PTY and can
// therefore receive a termination event (user stop / exit / gateway
// shutdown). The zero State is excluded on purpose.
var terminatableStates = map[State]bool{
	StatePending: true,
	StateRunning: true,
	StateIdle:    true,
}

// Next applies event to from and returns the resulting State. It is the
// guarded, idempotent transition function every lifecycle mutation should
// route through:
//
//   - Illegal transitions return (from, ErrIllegalTransition) and MUST be
//     rejected by the caller (no state change, surface an error).
//   - Idempotent no-ops (re-applying the event that produced the current
//     State) return (from, nil) so repeat exit-detector wakeups or a
//     double Stop are harmless.
//
// Next never mutates anything; it only computes the target State.
func Next(from State, event Event) (State, error) {
	if from.IsTerminal() {
		// From any terminal State only a (re)start is legal; the three
		// termination events are idempotent no-ops that preserve the
		// existing terminal State (a stopped session stays stopped even
		// if the process is later observed to have exited).
		switch {
		case event == EventStart:
			return StateRunning, nil
		case isTerminationEvent(event):
			return from, nil
		default:
			return from, ErrIllegalTransition
		}
	}

	if next, ok := liveTransitions[from][event]; ok {
		return next, nil
	}
	if next, ok := terminationEvents[event]; ok && terminatableStates[from] {
		return next, nil
	}
	return from, ErrIllegalTransition
}

// CanTransition reports whether event is a legal transition from state
// (including idempotent no-ops). Convenience wrapper over Next for guard
// checks that do not need the resulting State.
func CanTransition(from State, event Event) bool {
	_, err := Next(from, event)
	return err == nil
}

func isTerminationEvent(event Event) bool {
	_, ok := terminationEvents[event]
	return ok
}

// TerminationEvent maps the two booleans the exit detector already tracks
// (an explicit user stop; the gateway closing) to the lifecycle Event
// that drives the terminal transition. Precedence matches the historical
// classifyExitState: a user stop wins over a gateway shutdown, which wins
// over a spontaneous exit. Provided so callers can converge on Next as the
// single transition point instead of duplicating this precedence.
func TerminationEvent(stopRequested, closing bool) Event {
	switch {
	case stopRequested:
		return EventUserStop
	case closing:
		return EventGatewayShutdown
	default:
		return EventExit
	}
}

// InterruptReason refines StateInterrupted into the three recovery classes
// frozen in the 2026-07 design. The persisted state stays "interrupted";
// this is the "why", derived from OBSERVED facts (process liveness, gateway
// restart) rather than guessed — the "reconcile = truth" contract, where
// the DB row is only intent and the OS process table / WS / event stream
// are the reality written back.
type InterruptReason string

const (
	// InterruptDisconnected — the WS transport dropped but the CLI process
	// is healthy. Recovery: silently buffer output and wait for re-attach;
	// do NOT respawn (that would abandon a live, working process).
	InterruptDisconnected InterruptReason = "disconnected"
	// InterruptOrphaned — the gateway restarted and left the CLI running
	// as an orphan. Recovery: reconcile probes the OS for the recorded PID
	// and adopts it if still alive, rather than blindly respawning.
	InterruptOrphaned InterruptReason = "orphaned"
	// InterruptCrashed — the process is gone. Recovery: treat as failed and
	// roll back to the last checkpoint (or respawn with --resume where the
	// provider supports it).
	InterruptCrashed InterruptReason = "crashed"
)

// ClassifyInterrupt derives the InterruptReason from observed facts, not
// from the DB's stale intent:
//
//   - procAlive: is the OS process for the recorded PID still running?
//   - gatewayRestarted: did the gateway process itself restart (vs. only
//     the WS client disconnecting)?
//
// Precedence: a dead process is always "crashed" regardless of why we are
// looking; a live process after a gateway restart is "orphaned" (adoptable);
// a live process with only the transport gone is "disconnected".
func ClassifyInterrupt(procAlive, gatewayRestarted bool) InterruptReason {
	switch {
	case !procAlive:
		return InterruptCrashed
	case gatewayRestarted:
		return InterruptOrphaned
	default:
		return InterruptDisconnected
	}
}

// RecoveryStrategy names the action reconciliation should take for a given
// InterruptReason. Kept separate from the reason so the "what happened"
// (classification) and the "what to do" (policy) can evolve independently.
type RecoveryStrategy string

const (
	// RecoveryWaitReattach — buffer incremental output and wait for the
	// client to re-attach; the process is healthy (InterruptDisconnected).
	RecoveryWaitReattach RecoveryStrategy = "wait_reattach"
	// RecoveryAdoptPID — verify the recorded PID and adopt the live orphan
	// instead of respawning (InterruptOrphaned).
	RecoveryAdoptPID RecoveryStrategy = "adopt_pid"
	// RecoveryRollback — the process is gone; fail and restore the last
	// checkpoint / respawn with resume (InterruptCrashed).
	RecoveryRollback RecoveryStrategy = "rollback"
)

// Recovery returns the recovery strategy for this interrupt reason.
func (r InterruptReason) Recovery() RecoveryStrategy {
	switch r {
	case InterruptDisconnected:
		return RecoveryWaitReattach
	case InterruptOrphaned:
		return RecoveryAdoptPID
	default: // InterruptCrashed and any unknown reason: safest is rollback
		return RecoveryRollback
	}
}

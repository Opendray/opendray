# Session State Machine Hardening

Status: **Phase 2 — SSOT wired; concurrency-safe resume + interrupt cause persisted**
Owner: opendray gateway
Last updated: 2026-07-14

This is the load-bearing first step of the "harden the foundation before
orchestration" plan, and — after evaluation — the part that was kept.

Priority order as originally discussed, with outcomes:

1. **Session state-machine hardening** ← this document. **DONE & kept.**
2. ~~Context-level backup / restore (cwd snapshot + uncommitted diff + input
   history)~~ — **built then removed.** For a commit-everything-via-PR git
   workflow the working tree is almost always clean, so a checkpoint had
   nothing to snapshot; and the important "context survives a restart"
   property is already delivered here by auto-resume (`--resume`), not by a
   file checkpoint. Reverted (see 0079 drop migration).
3. ~~Audit panel (cost / quota stats + multimodal Artifacts tracking)~~ —
   **dropped.** Sessions don't drop except on a deliberate restart (already
   covered), and subscription cloud agents have no per-call cost to meter;
   the scoped API-call audit already exists (web Activity page).
4. _(parked)_ Multi-model orchestration (fan-out one task to N models, merge).

---

## 1. States

Persisted in `sessions.state` (TEXT). Unchanged by this phase — the enum
values below already exist in `internal/session/session.go`.

| State         | Terminal | Meaning |
|---------------|:--------:|---------|
| `pending`     | no  | Row created, PTY not yet live (transient). |
| `running`     | no  | PTY live, agent active. |
| `idle`        | no  | PTY live, no activity past the idle threshold. |
| `stopped`     | yes | Operator explicitly stopped it (`Manager.Stop` / DELETE-as-stop). Restartable. |
| `ended`       | yes | Process exited on its own (clean exit or child crash). Restartable. |
| `interrupted` | yes | Was live when the **gateway** process exited; PTY died with the daemon. Auto-resumed on next startup. |

`interrupted` is the ambiguous one. This phase **refines** it into three
recovery classes (below) without adding new persisted enum values.

### Zero State

`""` is the pre-persistence State of a freshly-constructed session. Only
`start` is legal from it; you cannot stop / exit / interrupt a session
that never spawned. Note `pending` (a *persisted* row whose spawn is in
flight) is distinct: it is terminatable but **not** startable — a second
`start` would race the first spawn for the same cwd.

---

## 2. `interrupted` sub-states (frozen) — and the architectural reality

The single `interrupted` state conflated three situations that were
*expected* to need different recovery. Naming per antigravity, endorsed by
all three reviewers. Modelled as `InterruptReason` refining
`StateInterrupted` — **not** as new persisted enum values.

| Reason         | Observed reality | Intended recovery |
|----------------|------------------|-------------------|
| `disconnected` | WS transport dropped, **process healthy** | `wait_reattach`: buffer output, wait for re-attach. Do NOT respawn. |
| `orphaned`     | Gateway restarted, CLI left as an **orphan (still alive)** | `adopt_pid`: probe the recorded PID and adopt it if alive. |
| `crashed`      | Process is **gone** | `rollback`: fail, roll back / respawn with `--resume`. |

```
ClassifyInterrupt(procAlive, gatewayRestarted):
    !procAlive               -> crashed
    procAlive && gwRestarted -> orphaned
    procAlive && !gwRestarted-> disconnected
```

### ⚠️ What the code investigation actually showed (2026-07-14)

Two of these three collapse once you look at the real process model
(`spawn` uses `pty.Start`, so the gateway holds the PTY **master fd**):

- **`disconnected` is already correct with no state change.** A WS client
  drop only calls `unsub()` (`handler.go`) → removes the subscriber channel
  (`manager.go`). It does **not** touch the process: the session stays
  `running`, the ring buffer keeps filling, and re-attach replays it. There
  is nothing to persist or recover — the design's `wait_reattach` *is* the
  current behaviour. `disconnected` is therefore a runtime transport state,
  never a persisted one.

- **`orphaned` / `adopt_pid` is infeasible for PTY-backed children.** When
  the gateway dies, the kernel destroys the PTY pair as its master fd
  closes; the orphan gets EIO/SIGHUP and is I/O-dead even if momentarily
  alive, and a new gateway **cannot re-acquire the master fd**. Blindly
  killing the recorded PID is also unsafe (PID reuse). So a gateway restart
  collapses to a single achievable outcome: **process gone → respawn with
  `--resume`** — which is what reconciliation already does.

**Consequence:** at *startup* reconcile the only recovery-relevant class is
effectively `crashed`. The genuinely useful, achievable hardening is
therefore (a) making resume **concurrency-safe** so two resumes can't race
the same cwd, and (b) recording the interruption **cause** for audit — not
building an un-attachable "adopt" path.

### Persisted cause vs. runtime recovery reason

`InterruptReason` above (disconnected/orphaned/crashed) is the *runtime
recovery* view. Persisted separately is the *audit cause* — why the row
became interrupted, observable at the interruption point
(`InterruptCause`, column `sessions.interrupt_reason`, migration 0077):

| Cause              | Set by | Meaning |
|--------------------|--------|---------|
| `gateway_shutdown` | `waitExit` when `isClosing` (`pump.go`) | Graceful daemon exit (self-update / restart). |
| `gateway_crash`    | `MarkRunningAsInterrupted` at next startup (`store.go`) | Daemon died hard; row was still live at boot. |

Nullable and additive; cleared to NULL on resume (`Reactivate`). No
backfill — historical rows have no observable cause.

**Reconcile = truth-check, not guess** still holds: the DB row is intent;
the OS process table / WS liveness are reality. The cause column records the
reality we *can* observe; we do not guess an "orphan is adoptable" story the
architecture cannot deliver.

---

## 3. Transition matrix (frozen)

Encoded as the single source of truth in
`internal/session/transitions.go` (`Next(state, event) (State, error)`)
and pinned cell-by-cell in `transitions_test.go`. `-` = **illegal**
(`Next` returns `ErrIllegalTransition`, State unchanged). Cells equal to
the row State are **idempotent no-ops**.

| from \ event   | `start`     | `idle`  | `resume`  | `user_stop`   | `exit`        | `gateway_shutdown` |
|----------------|-------------|---------|-----------|---------------|---------------|--------------------|
| `""` (zero)    | running     | –       | –         | –             | –             | –                  |
| `pending`      | – (spawning)| –       | –         | stopped       | ended         | interrupted        |
| `running`      | – (already) | idle    | running·  | stopped       | ended         | interrupted        |
| `idle`         | – (already) | idle·   | running   | stopped       | ended         | interrupted        |
| `stopped`      | running     | –       | –         | stopped·      | stopped·      | stopped·           |
| `ended`        | running     | –       | –         | ended·        | ended·        | ended·             |
| `interrupted`  | running     | –       | –         | interrupted·  | interrupted·  | interrupted·       |

`·` marks an idempotent no-op.

### Guard + idempotency guarantees

- **Guarded**: illegal jumps are rejected (`ended -> running` via `idle`;
  `start` on an already-running session, i.e. `ErrAlreadyRunning`; any
  `idle`/`resume` on a terminal row).
- **Idempotent**: repeat exit-detector wakeups, a double `Stop`, or a
  `gateway_shutdown` racing a session that already exited are all no-ops.
  The termination precedence (user stop ▷ gateway shutdown ▷ spontaneous
  exit) means a user stop is **never** overwritten by a later shutdown —
  this reproduces the historical `classifyExitState` precedence, which
  `TerminationEvent(stopRequested, closing)` now maps into the matrix.
- **No two resumes race the same cwd**: `start` is illegal from
  `running`/`idle`, so a second resume attempt is rejected rather than
  spawning a duplicate process against the same working directory.

---

## 4. The `disconnected -> resuming -> running` reconnect chain (design)

`resuming` is a **transient in-memory phase**, not a persisted state: a
terminal (`interrupted`) row is being brought back live via `start`. The
alignment focus is **incremental log buffering & replay** so a re-attach
does not lose or duplicate output.

```
running ──WS drop──▶ interrupted(disconnected)
                         │  (process still alive; ring buffer keeps filling)
                         ▼
                     [resuming]  ← client re-attaches
                         │  1. replay buffered tail from the ring buffer
                         │  2. queue new input until replay drains (input guard)
                         │  3. release the input queue
                         ▼
                     running
```

Rules:
- **Buffer, don't respawn** for `disconnected`: the PTY is healthy; only
  the transport was lost. Respawning would kill live work.
- **Input guard during `resuming`**: client input is queued, not sent to
  the PTY, until buffered output has been replayed — so the operator never
  types into a half-replayed screen.
- **Incremental replay** from the existing ring buffer (`ringbuf.go`);
  the open question is how much tail to replay vs. a full snapshot (see §6).

Note (per §2): `disconnected` already behaves exactly like this today —
the process is never touched on a WS drop and re-attach replays the ring.
The `orphaned` "adopt probe" is **not pursued** (PTY master fd dies with the
gateway; see §2). For a gateway restart the chain is simply: interrupted
(`gateway_shutdown` or `gateway_crash`) → concurrency-safe `start` → replay.

---

## 5. Highest-risk transitions (tests first)

Written test-first in `transitions_test.go`; these are the cells most
likely to corrupt state under failure:

1. **Gateway restart** (`running/idle/pending -> interrupted`) then
   `start -> running` (auto-resume).
2. **Disconnect** (`running -> interrupted(disconnected)`), process stays
   alive — must NOT respawn.
3. **Abnormal child exit** (`running -> ended`) vs. gateway shutdown —
   precedence must not misclassify a crash as an interruption.
4. **Resume failure**: `start` from `interrupted` fails → row must remain
   `interrupted` (recoverable next boot), never a phantom `running`.
5. **Double stop / repeat exit wakeup**: idempotent no-ops.

---

## 6. Open questions

- ~~**Persisting the interrupt reason**~~ — **DONE** (migration 0077,
  `InterruptCause` = `gateway_shutdown` / `gateway_crash`). See §2.
- ~~**Concurrency-safe resume**~~ — **DONE** (`Manager.tryReserveStart`
  reservation closes the check-then-spawn TOCTOU; see §7).
- **Replay fidelity vs. cost**: antigravity wants context pruned /
  compressed on resume to cut token cost; that is in tension with the
  "full-context replay" goal. Reconcile the two — likely full **terminal**
  replay (ring buffer) but pruned **model-context** replay.
- ~~**Backup scope / checkpoint format**~~ and ~~**audit / Artifacts
  tracking**~~ — resolved: both were evaluated and are out of scope for this
  deployment (see the outcomes in the priority list above).

---

## 7. Adoption plan (incremental, non-destructive) — status

`transitions.go` is a **pure SSOT** with no side effects. It was landed
first, fully tested, so the lifecycle mutation points migrate onto it one at
a time without a big-bang rewrite:

1. ✅ **Done (PR #450)** — `Manager.Start`'s "already running" guard routes
   through `CanTransition(state, EventStart)`, pinned equal to
   `!IsTerminal()` by `TestStartLegalIffTerminal`.
2. ✅ **Done (PR #450)** — `waitExit`'s terminal classification goes through
   `TerminationEvent` + `Next` (`classifyExitState` kept as a thin shim,
   pinned equal by `TestTerminationEventPrecedence`).
3. ✅ **Done (Phase 2)** — **concurrency-safe resume**:
   `Manager.tryReserveStart` reserves the id under `mu` across the state
   check *and* the spawn, closing the TOCTOU where two resumes could both
   pass the guard and race the same cwd (`TestTryReserveStart*`).
4. ✅ **Done (Phase 2)** — persist the interruption **cause**
   (`gateway_shutdown` / `gateway_crash`) for audit (migration 0077).
   Replaces the old plan item "reconcile PID-liveness probing → adopt",
   which §2 showed is infeasible for PTY-backed children.
5. ⬜ **Out of scope** — context backup/checkpoint and the audit/cost panel
   were both evaluated and removed/dropped (see the priority list at the top).
   Orchestration remains parked.

Each step is guarded by the matrix + reservation tests, so behaviour cannot
silently drift.

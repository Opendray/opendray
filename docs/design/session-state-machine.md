# Session State Machine Hardening

Status: **Phase 1 — design frozen, SSOT landed (guard not yet wired)**
Owner: opendray gateway
Last updated: 2026-07-14

This is the load-bearing first step of the "harden the foundation before
orchestration" plan. Context-level backup/restore, the audit/cost panel,
and any future multi-model orchestration all depend on a **trustworthy**
session state, so the state machine is hardened first.

Priority order for the whole plan (orchestration is explicitly deferred):

1. **Session state-machine hardening** ← this document
2. Context-level backup / restore (cwd snapshot + uncommitted diff + input history)
3. Audit panel (cost / quota stats + multimodal Artifacts tracking)
4. _(parked)_ Multi-model orchestration (fan-out one task to N models, merge)

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

## 2. `interrupted` sub-states (frozen)

The single `interrupted` state conflated three genuinely different
situations that need different recovery. Naming per antigravity, endorsed
by all three reviewers. Modelled as `InterruptReason` refining
`StateInterrupted` — **not** as new persisted enum values (so no migration
is required to freeze the taxonomy; persisting the reason is a later,
additive column).

| Reason         | Observed reality | Recovery strategy |
|----------------|------------------|-------------------|
| `disconnected` | WS transport dropped, **process healthy** | `wait_reattach`: silently buffer incremental output, wait for re-attach. **Do NOT respawn** — that abandons a live, working process. |
| `orphaned`     | Gateway restarted, CLI left as an **orphan (still alive)** | `adopt_pid`: reconcile probes the OS for the recorded PID and **adopts** it if alive, rather than blindly respawning. |
| `crashed`      | Process is **gone** | `rollback`: treat as failed, roll back to the last checkpoint (or respawn with `--resume` where the provider supports it). |

### Classification is truth, not guess

```
ClassifyInterrupt(procAlive, gatewayRestarted):
    !procAlive               -> crashed      (a dead process is always crashed)
    procAlive && gwRestarted -> orphaned     (adoptable)
    procAlive && !gwRestarted-> disconnected (transport only)
```

**Reconcile = truth-check, not guess.** The DB row records *intent*; the
OS process table / WS liveness / event stream are *reality*. Reconcile
must observe the facts and write the truth back — never infer state from
the stale DB value alone.

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

For `orphaned`, the chain first runs an **adopt** probe: verify the
recorded PID (and that it is our child / matches the expected argv) before
deciding adopt-vs-respawn. For `crashed`, skip straight to rollback +
resume.

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

## 6. Open questions (carry into Phase 2)

- **Persisting the interrupt reason**: an additive nullable
  `interrupt_reason` column vs. keeping it purely runtime. Needed before
  the audit panel can show *why* a session dropped.
- **Replay fidelity vs. cost**: antigravity wants context pruned /
  compressed on resume to cut token cost; that is in tension with the
  "full-context replay" goal. Reconcile the two — likely full **terminal**
  replay (ring buffer) but pruned **model-context** replay.
- **Checkpoint definition** for `crashed` rollback: what exactly is a
  checkpoint, when is it taken, where stored.
- **Backup scope** (Phase 2): cwd key-file snapshot + uncommitted diff +
  input history — depends on this state machine being solid first.
- **Audit multimodal / Artifacts diff tracking**: Phase 3 vs. later.

---

## 7. Adoption plan (incremental, non-destructive)

`transitions.go` is a **pure SSOT** with no side effects; nothing in the
running gateway calls it yet. It is landed first, fully tested, so the
lifecycle mutation points can be migrated onto it one at a time without a
big-bang rewrite:

1. Route `Manager.Start`'s "already running" guard through
   `CanTransition(state, EventStart)`.
2. Route `waitExit`'s terminal classification through
   `TerminationEvent` + `Next` (keeping `classifyExitState` as a thin
   shim, already pinned equal by `TestTerminationEventPrecedence`).
3. Add reconcile PID-liveness probing → `ClassifyInterrupt` →
   `RecoveryStrategy` (this is where `orphaned`/`disconnected`/`crashed`
   start driving behaviour).
4. Only then: persist `interrupt_reason`, then build backup/checkpoint on
   top.

Each step is guarded by the existing matrix tests, so behaviour cannot
silently drift.

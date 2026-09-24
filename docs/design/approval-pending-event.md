# `session.approval_pending` Event

Status: **Draft — RFC for review (no implementation yet)**
Owner: opendray gateway
Last updated: 2026-08-22

A new event bus topic that fires when a running session is paused waiting
for the operator to approve a tool call. Small, enabling change: it is the
shared dependency under the voice RFC (#524, voice approvals) and the
relay/push RFC (#525, AFK notifications), and it improves the web UI on its
own.

---

## 1. Problem

Today opendray publishes session lifecycle events —
`session.started` / `idle` / `turn_completed` / `ended` / `output`
(`internal/session/pump.go`, `manager.go`) — but nothing that says *"the
agent stopped because it is waiting for a yes/no."* Every consumer that
wants the AFK loop (voice approvals, mobile push, a web "approve" toast)
would otherwise re-detect that state independently, each scraping the PTY
in its own way.

Land the detection once, as an event, and every surface subscribes.

## 2. Why now

- **Voice (#524)** — flow 3 (voice approvals) is gated on it.
- **Relay + push (#525)** — the highest-value notification ("approve on your
  phone") needs it.
- **Web UI** — an explicit "approval pending" affordance instead of the user
  reading the TUI.

Recommended in both prior RFCs as a standalone change to build first.

## 3. Detection

opendray already parses agent transcripts into `text` / `thinking` /
`tool_use` / `tool_result` blocks (`internal/session/claude_jsonl.go`) and
already runs an idle watcher (`pump.go idleWatcher`). Approval-pending is the
intersection of those two signals:

- **Primary (transcript-based, providers with a JSONL transcript).** The
  newest block is a `tool_use` with **no matching `tool_result`**, and the
  session has gone idle (agent emitted nothing for `idle_threshold`). That is
  the agent paused on a permission gate. Carries the tool name + input
  summary from the `tool_use` block.
- **Fallback (PTY pattern, providers without a structured transcript).** A
  small per-provider table of approval-prompt markers (e.g. the CLI's
  "Do you want to proceed? (y/n)" line) matched against the recent PTY tail.
  Table-driven data, isolated to this detector — the same shape as the voice
  keymap.

Detection lives beside the idle watcher so it reuses the existing
active→idle edge; it never polls the agent.

## 4. Event shape

```go
// Topic: "session.approval_pending"
type ApprovalPending struct {
    SessionID string `json:"session_id"`
    Provider  string `json:"provider"`
    Tool      string `json:"tool,omitempty"`     // e.g. "Bash", "Write"
    Summary   string `json:"summary,omitempty"`  // short human-readable action
    Cwd       string `json:"cwd,omitempty"`
    Detected  string `json:"detected"`           // "transcript" | "pty"
}
```

A matching `session.approval_resolved` (approved / denied / timed out) closes
the pair so subscribers can clear a pending notification. Resolution itself
(writing the accept/deny keystroke) is **out of scope** here — it belongs to
the consumer (voice C6, a web button), which writes to `POST
/sessions/{id}/input`.

## 5. Consumers (not built here)

- Voice (#524) — asks "approve?", writes the keystroke on the spoken answer.
- Push (#525) — sends an actionable notification to registered devices.
- Web UI — an inline "approve / deny" affordance.

This RFC ships only the **producer** (detection + the two events).

## 6. Fundamentals compliance

| Ref | Fundamental       | Verdict                                                  |
|-----|-------------------|----------------------------------------------------------|
| F1  | Cross-CLI parity  | Neutral — transcript path is provider-agnostic; PTY fallback is a small data table, verified per CLI |
| F5  | Multi-user        | Event scoped to the session; subscribers apply principal scoping |

No new dependency, no per-provider branching in logic (only a marker table
for the fallback).

## 7. Phasing

1. **P1 — Producer.** Detection beside the idle watcher + the two events
   (`session.approval_pending` / `session.approval_resolved`), transcript
   path first (claude / codex — JSONL providers), then the PTY-fallback table
   for the rest. Ships with a web-UI consumer as the first proof.

That is the whole RFC — it is deliberately one phase, so voice and push can
depend on a stable event.

## 8. Open decisions

- **D1 — False positives on the transcript path.** A `tool_use` with no
  `tool_result` can also mean the tool is *running*, not awaiting approval.
  **Recommend** gating on the idle edge (only fire after `idle_threshold`
  with no output) and letting `session.approval_resolved` / new output cancel
  a pending state, rather than trying to distinguish perfectly up front.
- **D2 — PTY fallback scope.** Ship the marker table for the non-JSONL
  providers now, or defer them to "best-effort, transcript-only first"?
  **Recommend** transcript-first; add PTY markers per provider as verified.

## 9. Review asks (linivek)

1. Event shape + the `pending` / `resolved` pair — right granularity?
2. D1 — is idle-gated detection (accepting occasional lag over false
   positives) the right trade?
3. Ship this standalone ahead of voice (#524) and push (#525), with a
   web-UI consumer as the first user?

## 10. Non-goals

- Answering the approval (keystroke injection) — that is the consumer's job.
- A generic structured "agent state" stream — this is one targeted event, not
  a rework of the transcript pipeline.

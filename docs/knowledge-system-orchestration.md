# Knowledge System Orchestration — Memory · Notes · Knowledge

> Blueprint for unifying opendray's three knowledge surfaces into one
> AI-driven, coordinated pipeline. Supersedes the "three independent
> sweeps" state. Status: PROPOSAL (awaiting operator sign-off).
> All work stays local on `feat/knowledge-graph` until approved.

## 1. Problem

Three surfaces exist but work as silos:

- **Memory** (`internal/memory`) — pgvector episodic facts, auto-captured, AI-only.
- **Notes** (`internal/projectdoc`) — goal / plan / tech_stack / recent_activity docs + journal + proposals. The project's official documentation.
- **Knowledge** (`internal/knowledge` + KB docs) — entity/fact/playbook/skill graph + curated KB pages (infra/conventions/lessons/handbook).

Pain points:

1. **Overlap.** Notes' goal/plan/tech_stack duplicate facts also sitting in Memory; the Knowledge graph's `fact` nodes are a 1:1 copy of Memory rows.
2. **Notes has no AI drive.** goal/plan update only when an agent explicitly calls the MCP tool; in practice they're forgotten and go stale — yet Notes is the developer's (and a team's) primary project-doc view.
3. **No coordination.** Each layer has its own sweep; nothing routes one consolidation pass into the right destinations.
4. **No time-decay.** Infrastructure changes (RCC → opendray) and dead/paused projects accumulate as stale noise; nothing supersedes or archives.

## 2. The model — three layers by time-horizon × audience

| Layer | Horizon | Scope | Audience | Role | AI drive (today → target) |
|-------|---------|-------|----------|------|---------------------------|
| **Memory** | short / episodic | project | AI | Raw working-memory stream: progress, decisions, "don't drift". Feedstock. | auto-capture ✅ (keep) |
| **Notes** | medium / current-state | project | human + AI | The project's **living official doc**: goal, plan, architecture, tech stack, handbook. Developer's primary view; team-critical. | manual ❌ → **AI-proposed from Memory+Journal, human-approved** |
| **Knowledge** | long / accumulated | **global / cross-project** | AI + human | Transferable expertise: infrastructure, conventions, **reusable past-project features**, distilled skills/MCP, lessons. | AI-drafted ✅ (keep + extend) |

Mnemonic: **Memory = what just happened. Notes = where this project is. Knowledge = what we know across all projects.**

## 3. Ownership rules (kill the overlap)

Each piece of information has exactly one canonical home:

- **Current project state** (goal / plan / architecture / tech stack) → **Notes only.** Memory may capture it episodically, but Notes is canonical and the only surface shown/injected for it. Memory stops being a goal/plan source.
- **Cross-project reference** (infra / conventions / reusable features / skills / lessons) → **Knowledge only.**
- **Raw episodic facts** → **Memory only** (feedstock; not a primary display surface).
- **Knowledge graph `fact` nodes** are a Memory duplicate → **demote.** Keep `entity` (cross-project linking) + `playbook`/`skill` (the distilled value); the fact-node layer becomes an internal index, not a user surface (it may be retired once entities can be derived without it).

Result: no datum is authored in two places. Memory → (consolidation) → Notes/Knowledge is a one-way derivation, not a copy kept in sync by hand.

## 4. The consolidation pipeline (one engine, not three sweeps)

```
        work happens
            │
            ├── Memory.capture (episodic facts)        [auto, exists]
            └── Journal.append (session work-traces)   [auto, exists]
            │
   ┌────────▼─────────  CONSOLIDATION ENGINE  ─────────────────┐
   │  reads: this project's Memory + Journal (+ recent diff)    │
   │                                                            │
   │  ├─ per-project  → propose Notes updates                   │
   │  │     goal / plan / architecture / handbook               │
   │  │     (propose→approve; dirty-checked; only on drift)     │
   │  │                                                          │
   │  └─ cross-project → draft Knowledge updates                │
   │        infrastructure / conventions / lessons /            │
   │        reusable-features / skills   (global, lock-aware)    │
   └────────────────────────────────────────────────────────────┘
            │
   new session spawn ◄── inject: Notes(this project) + Knowledge(global)
```

Principles:

- **One scheduler, staged work.** Replace the separate anchor/reflect/KB/notes sweeps with one consolidation loop that, per cycle, (a) refreshes per-project Notes for projects with new Memory/Journal, (b) refreshes global Knowledge. Dirty-checked (skip unchanged) + lock-aware (never overwrite human edits) so steady-state cost ≈ 0 LLM.
- **Notes is the new consumer.** Today consolidation only feeds Knowledge; it must also feed Notes (the missing AI drive).
- **Propose vs write.** Notes (human's doc) uses propose→approve. Knowledge (AI reference) is drafted directly as `agent`, locked on human edit. Both already supported by projectdoc.

## 5. AI drive per layer (close the gaps)

- **Memory** — already automatic. No change.
- **Notes** — NEW: consolidation proposes goal/plan/architecture/handbook from Memory+Journal when it detects drift (the plan-drift detector already exists for plans; generalize it). Operator approves in the Inbox. This is the headline fix.
- **Knowledge** — already AI-drafted; extend with a **reusable-features** page (past project capabilities worth lifting) and keep skills/lessons.

## 6. Time-decay

- **Project lifecycle.** Add a per-project status: `active` / `paused` / `archived`.
  - `archived` (or `paused`): Notes frozen, excluded from spawn injection and from cross-project Knowledge distillation. Surfaced read-only.
  - Driven by the operator (a button) and/or inferred (no activity for N days → suggest archive).
- **Supersession / canonicalization.** Knowledge is re-drafted from *current* Memory each cycle, so it naturally tracks the latest (RCC → opendray). Harden with: (a) a small alias/supersedes map so deprecated terms collapse to the current canonical, (b) a "prefer recent, mark superseded" instruction in the drafter, (c) entity `supersedes` edges already in the ontology.
- **Stale Notes.** Because Notes is now AI-kept-current (§5), drift self-corrects; the existing stale-journal pruning stays for the raw journal.

## 7. Spawn injection (who feeds the prompt)

A spawning session, for its cwd, receives:

1. **Notes** of THIS project (goal / plan / architecture / handbook) — current state.
2. **Knowledge** global (infrastructure / conventions / lessons / reusable-features / skills) — transferable expertise.
3. A bounded slice of recent **Journal** (what just happened).

Memory facts are NOT injected wholesale (they're feedstock); the agent pulls from Memory on demand via the MCP. Budgeted; archived projects contribute nothing.

## 8. Graph demotion

The entity/fact graph stays as an internal index for now (operator: "暂留观望"):

- `fact` nodes — hidden from the UI (Memory is the fact surface); kept only as the anchor for entity extraction. Candidate for retirement once entities derive directly from Memory.
- `entity` nodes — retained: they power cross-project linking ("everything about PostgreSQL").
- `playbook` / `skill` — retained: the distilled value, surfaced via Knowledge (Lessons / Skills).

The Knowledge web page already leads with the KB; the graph tab is the demoted view.

## 9. Migration & rollout (additive, reversible)

The same discipline as the M-KB work: additive schema, feature-flagged, one-way deps, local-until-tested.

- **P-A — De-dup + ownership.** Stop surfacing/injecting goal/plan from Memory; Notes becomes canonical. (No schema change; injection + capture rules only.)
- **P-B — Notes AI drive.** Consolidation proposes goal/plan/architecture/handbook from Memory+Journal (generalize the plan-drift detector). Propose→approve. Highest user value.
- **P-C — Unified scheduler.** Fold anchor/reflect/KB/notes sweeps into one consolidation loop; shared dirty-check.
- **P-D — Project lifecycle.** `status` column + archive UI + injection/distillation skip.
- **P-E — Supersession.** Alias map + drafter "prefer-recent/mark-superseded" + entity supersedes edges.
- **P-F — Knowledge: reusable-features page.**
- **P-G — Graph demotion / fact-node retirement.**

Each phase: build → local rebuild + restart → operator validates → next. Push only after the whole arc is accepted.

## 10. Decisions (SIGNED OFF 2026-06-08)

1. ✅ **Notes canonical.** goal / plan / architecture live ONLY in Notes; Memory stops being a goal/plan source.
2. ✅ **Notes AI-drive = auto-propose on drift.** Every consolidation cycle, on detected drift, auto-file a Notes proposal; operator approves in the Inbox.
3. ✅ **Archive = operator button + auto-suggest after N idle days.**
4. ✅ **Fact nodes fully retired.** Entities are extracted straight from Memory; no `fact` node layer. (Memory is the fact store; Knowledge keeps entity + playbook + skill only.)
5. ✅ **Reusable-features page = AI-drafted, human-lockable** (same model as the other KB pages).

Implementation proceeds P-A → P-G (§9), each: build → local rebuild + restart → operator validates → next. Push only after the whole arc is accepted.

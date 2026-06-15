# The Experience Flywheel — unifying Memory · Notes · Knowledge

> **SUPERSEDED by [`cortex-architecture.md`](cortex-architecture.md).**
> The flywheel MODEL this doc defined (three rungs, one loop, two-nature
> Knowledge, AI-leads-human-supervises) still holds — Cortex is its
> completion: the three rungs fused into one named module, the fixed doc
> kinds replaced by per-project blueprints, third-party capture
> quarantined, the curation-conversation channel added, and the UI
> re-expressed behind a single nav entry. Do not base new decisions on
> the per-step status below; it is historical.

> Status: MODEL ACCEPTED — implementation in progress. The operator
> confirmed the flywheel model + the two-nature Knowledge refinement, and
> chose "Notes = projectdoc" (§7). Backend edges + the web overhaul are
> implemented, green, and committed LOCALLY (not pushed, not yet built —
> the operator has live opendray sessions; the unified rebuild waits for
> their go-ahead). Mobile follows after the web is validated. Supersedes
> `knowledge-system-orchestration.md` and the M-KG/M-KB patch series.
> See §10 for per-step progress.

## 0. Why a re-architecture (not another patch)

Memory, Notes, and Knowledge were built — and presented in the UI — as
**three independent stores with three nav tabs**. In reality they are
**three rungs of one ladder inside one closed loop**. Treating them as
silos produced the exact problems the operator flagged:

- **Redundancy.** The same datum (a goal, a fact, a project's shape) is
  copied into several stores and re-rendered into several pages
  (e.g. a per-project `kb_handbook` restates the project's Notes; the
  "Project" screen mixes Notes *and* Memory; two parallel "notes"
  systems exist — `projectdoc` and the markdown vault).
- **No visible loop.** The compounding cycle — *use prior knowledge to
  bootstrap a project → the work crystallises into notes + memory →
  distil that into knowledge → bootstrap the next project faster* — is
  hidden behind disconnected tabs.
- **No real iteration.** Knowledge does not supersede itself when the
  world changes (DOS → Windows → Linux). Supersession is a prompt hint,
  not a mechanism.

The fix is one coherent model with **one home per datum** and the loop
made first-class. The guiding invariant:

> **Promotion between rungs is *transformation* (raw → consolidated →
> generalised), never *copy*.** If a datum is copied, that is the bug.

## 1. The model — a maturity ladder closed into a loop

```
        ┌──────────── inject / bootstrap a new project ────────────┐
        │   (start from prior experience, not from scratch)        │
        ▼                                                          │
  ① MEMORY  ──crystallise──►  ② NOTES  ──distil──►  ③ KNOWLEDGE ───┘
   raw episodic facts          this project's          cross-project,
   (recall by relevance)       official doc            iterable assets
        ▲                                                   │
        └──────── work continuously emits memory + notes ◄──┘
```

| Rung | One job (in the loop) | Data | Provenance | Audience |
|------|----------------------|------|------------|----------|
| **① Memory** | Capture what happens; recall by relevance. The raw input. | episodic facts | AI auto-capture | mostly AI |
| **② Notes** | Crystallise memory + work into *where this project is*. | goal · plan · architecture · decisions · journal (one set per project) | AI-proposed → human-approved | human + AI |
| **③ Knowledge** | Generalise across projects into reusable, **iterable** expertise; inject it back to bootstrap. | see §2 (two natures) | see §2 | human + AI |

Mnemonic: **Memory = what just happened. Notes = where this project is.
Knowledge = what we know across all projects.** Each rung is a distinct
*maturity × generality*; nothing is authored twice.

## 2. Knowledge has two natures (the key refinement)

Knowledge is not one thing. Conflating the two below is a second root
cause of the mess.

### 2a. Foundational — declared ground truth + binding rules

The environment and the rules for using it. **Not derived from any
project**; it *governs* every project.

- **Examples.** Shared dev database (`192.168.3.88:5432`), the Proxmox
  VMs (`kv01`/`kv02`), the LXC ID range (`8601–8699`), credential store
  (Vaultwarden), the opendray gateway; plus the **rules of use**:
  "never connect as the `linivek` superuser", "create a per-project DB
  user", "store secrets in Vaultwarden, never in shell".
- **Shape.** Each item = **fact(s) + rules-of-use**. A foundational
  entry is not just reference; it carries **constraints that must be
  obeyed**.
- **Provenance.** Human-declared or environment-captured; AI keeps it
  current; **human locks** it.
- **Direction.** Flows **down** into every project as a **binding
  constraint** injected at spawn (higher priority than guidance).
- **Why it matters.** Foundational rules are the **guardrails** that let
  the AI lead autonomously without causing "serious accidents" — exactly
  the operator's stated concern. The human declares + locks the
  guardrails; the AI runs inside them.

### 2b. Emergent — distilled experience

What bubbles **up** from doing the work: lessons, playbooks, skills,
frameworks, reusable structures/components.

- **Provenance.** AI-distilled from many projects' Notes + Memory;
  human reviews.
- **Direction.** Up (distilled), then down (borrowed as help — *not*
  binding).
- **Authority.** Guidance, not law.

> Both natures are cross-project and iterable and injected. They differ
> in **provenance** (declared vs distilled), **authority** (binding vs
> guidance), and **default flow** (governs-down vs emerges-up).

## 3. The loop edges (and their current status)

| Edge | What it does | AI role | Human role | Status |
|------|--------------|---------|-----------|--------------|
| **Capture** work → Memory | record facts/events as work happens | auto | prune rarely | ✅ exists |
| **Crystallise** Memory → Notes | keep the project doc current from work | propose on drift | approve / edit | ✅ P-B (goal + plan drift on session-end); continuous-from-loop deferred |
| **Distil** Notes+Memory → Knowledge | lift transferable expertise | draft | review / lock | ✅ reflect + KB drafter + consolidation engine (P-C) |
| **Iterate** new evidence → supersede old Knowledge | keep knowledge current (DOS→Win→Linux) | propose update on divergence | review/ratify | ✅ **B3** — a locked page whose feedstock diverged is re-drafted as a *proposal* (not overwritten); unlocked pages auto-redraft |
| **Inject** Knowledge → new project | bootstrap from prior experience; **enforce foundational rules** | assemble spawn banner | — | ✅ **B2** — foundational injected FIRST as binding "RULES you MUST follow"; emergent as reference |

**The revolution was to close + strengthen the weak edges, not add stores.**
Done: Iterate is now a real propose-on-divergence mechanism (B3, via the
KB drafter's dirty-check + projectdoc proposals — simpler and broader than
the originally-planned `memory_conflicts` wiring); Inject now distinguishes
**binding foundational rules** from **emergent guidance** (B2). Crystallise
runs on session-end (P-B); making it also fire from the consolidation loop
is a deferred enhancement.

## 4. AI leads, human supervises (everywhere)

One uniform control model across all three rungs:

- **AI drives by default** — capture, crystallise, distil, iterate all
  run automatically.
- **Human is the supervisor** — every AI change to a human-visible
  surface is a **proposal** the operator can review/approve/edit, and any
  page can be **locked** (a human edit freezes it from AI overwrite).
- **Foundational knowledge is human-authority** — declared and locked by
  the human; injected as binding rules. This is the safety boundary.

## 5. De-duplication map (one home per datum)

| Datum / artifact | Currently scattered in | Target home | Action |
|------------------|------------------------|-------------|--------|
| goal / plan / architecture | Memory facts + Notes docs | **Notes** | Memory stops being a source ✅ (locked) |
| per-project handbook (`kb_handbook`) | Knowledge (added recently) | **Notes** | **DELETE** — per-project lives only in Notes |
| project memory hygiene (health / conflicts / archived) | mixed into the "Project" screen | **Memory** | **DECONFLATE** — move back under Memory |
| infrastructure / conventions | Knowledge KB pages + global CLAUDE.md | **Knowledge › Foundational** | keep; model as fact+rules, mark binding |
| lessons / skills / reusable | Knowledge | **Knowledge › Emergent** | keep |
| graph `fact` nodes (1:1 Memory mirror) | Knowledge graph | — | ✅ retired (P-G) |
| **two "notes" systems** (`projectdoc` vs markdown vault) | both claim "the project doc" | **one** | **OPEN DECISION** — see §7 |

## 6. What this means for the code (one coherent change)

Reuse the working loop-plumbing; remove the redundant parts; strengthen
the weak edges. Concretely:

- ✅ **Keep** (already correct loop edges): auto-capture; consolidation
  engine (P-C); reflect + KB distillation; reusable-features as Emergent
  (P-F); fact-node retirement (P-G); per-project lifecycle; Notes
  self-description (badges/purpose, P-B drift banner).
- ✅ **Remove**: the `kb_handbook` per-project page + its Notes tab + its
  backend drafting + spawn injection (B1); the project/vault toggle that
  jammed the conflated screen into Notes (superseded by the variant split).
- ✅ **Deconflate**: ProjectScreen gained a `variant` — `notes`
  (goal/plan/tech/activity/journal/inbox) at `/notes`; `memory`
  (health/conflicts/archived) at `/memory/project` (R1).
- ✅ **Build**: Foundational binding injection (B2); the Iterate edge as
  propose-on-divergence (B3). Continuous Crystallise deferred.
- ✅ **Re-express the UI** (web): Notes deconflated + self-describing;
  Knowledge redesigned into Foundational/Emergent with binding markers +
  iteration-proposal review (F3); vault demoted to `/vault`. Mobile
  pending (after web validation).

## 7. DECISION (made) — the two "notes" systems

There were two stores both claiming "the project's document": `projectdoc`
(structured goal/plan/journal, DB-backed, AI-driven, `.opendray/*.md`
mirror) and the **markdown vault** (files, git-synced, freeform).

> ✅ **Resolved: Notes = `projectdoc`.** The markdown vault is demoted out
> of the core triad to its own `/vault` route + nav item (a freeform /
> Obsidian-sync utility) — no longer a peer of Memory/Knowledge, no longer
> competing for the "Notes" name. Implemented in R1.

## 8. UI — express the flywheel, not the silos (sketch)

- **Project view** (per cwd): shows the project's position in the loop —
  what Knowledge bootstrapped it (the foundational rules in force +
  borrowed skills), its current Notes (goal/plan/journal), and what it
  has contributed back. Memory hygiene is reachable but not mixed in.
- **Knowledge view** (global, the compounding asset): two clearly
  separated sections — **Foundational** (infra/conventions/policies, with
  lock + "binding" markers) and **Emergent** (lessons/skills/frameworks/
  reusable). This is the asset that grows across projects.
- **Memory view**: raw facts + hygiene (health/conflicts/archived).

Final UI is designed after the model is signed off; this is direction,
not pixels.

## 9. Rollout

One coherent change, feature-flagged, local-until-approved, reversible.
Each step builds green; nothing is pushed and nothing is rebuilt/restarted
until the operator says so (they run live opendray sessions on the current
binary).

## 10. Progress

All local on `feat/knowledge-graph`, green (`go build`/`test`/`vet`/
`gofmt`, web `tsc`, i18n parity en/zh/es), **unpushed + not yet built**.

- ✅ `c2eb79e` — this architecture doc.
- ✅ **R1** `81229fc` — deconflate Notes/Memory (ProjectScreen `variant`),
  `/notes` = project doc, `/memory/project` = memory hygiene, vault → `/vault`.
- ✅ **B1** `72b7ed6` — remove per-project handbook (drafting + injection +
  migration 0042); move frozen-skip to the Reflector.
- ✅ **B2** `cde63de` — Foundational knowledge injected as binding rules,
  first + un-truncated; drafter prompts emit an explicit "## Rules" section.
- ✅ **B3** `3784737` — Iterate: locked pages get update *proposals* on
  feedstock divergence (not overwrites).
- ✅ **F3** `c8a87a6` — Knowledge page redesigned (Foundational/Emergent,
  binding/lock markers, inline proposal review); subtitle refreshed.
- ✅ **Mobile** `d272c2e` — full parity: ProjectScreen `variant`
  (notes|memory), handbook removed, vault → More, Knowledge two-nature +
  proposal review. Dead handbook i18n/types swept (web + shared too).

**Pending (gated on the operator):**
- The single unified rebuild + restart (`opendray-v2-update-local.sh
  --restart`) so the operator can validate web (+ a Flutter rebuild for
  mobile) — held until they pause their live opendray sessions.
- Optional later: continuous Crystallise from the consolidation loop; a
  loop-overview affordance (§8); explicit per-rule structured storage for
  Foundational items.

# Self-Evolving Knowledge Brain — Redesign Blueprint (arc: M-KG)

> Status: **DRAFT / planning**. Branch `feat/knowledge-graph`.
> **Architecture: #3 — memory-rooted evolution.** We do NOT rebuild the
> memory system and we do NOT build a second parallel store. We **grow a
> structured, self-evolving knowledge layer on top of the just-shipped M-U
> memory system**; the old `notes` file vault is retired into it. Memory stays
> the protagonist. Phased, additive, reversible — mirrors the discipline of
> [`memory-redesign.md`](./memory-redesign.md).

---

## 1. Why this redesign exists

opendray has two knowledge systems and both fail the operator:

- **`notes`** (a manual markdown file vault) only writes when an agent
  is **explicitly asked** → low trigger → forgotten → dead. Manual trigger is
  the killer.
- **`memory`** (pgvector facts, the M-U arc) is automatic and powerful but
  **fragmented, AI-only, cwd-locked** → no cross-project transfer, no
  compounding, can't self-grow.

Net: the knowledge-base value a hand-kept vault used to give is
**lost** — memory cannibalised the habit without giving the value back.

**North star: a knowledge layer whose purpose is AI self-evolution** —
structured, cross-project-transferable, *compounding* knowledge distilled
**automatically** from episodic memory. The human reads and corrects; the
human does **not** author or maintain.

## 2. The architecture decision (#3) and why

Three architectures were weighed:

| | what it is | verdict |
|---|---|---|
| 1 · Fuse & rebuild | one brain, memory rebuilt into it | highest ceiling **but** disrupts the just-shipped system; high risk |
| 2 · Independent peers | new KG reads memory, both kept separate | safest **but** permanent redundancy + a seam that caps capability + revives the "which system?" ambiguity |
| **3 · Memory-rooted evolution** ✅ | grow a structured layer **on top of** memory (referencing fragments, not copying) | best balance — see below |

**Key insight:** architectures 1 and 3 reach the **same destination** (one
unified knowledge brain). The difference is the *path*: 1 rebuilds, 3 evolves.
Because M-U just shipped, **evolution strictly dominates rebuild** — same
endpoint, less waste, lower risk, memory stays central throughout. Architecture
2 is a *different, worse* destination (redundancy + seam + ambiguity).

Decision drivers (owner-fixed): M-U is freshly stabilised (minimise
disruption), long-term capability ceiling matters more than short-term
convenience, and redundancy is to be avoided. #3 wins on all three.

**This reframes the task:** not "rebuild notes", but **"evolve memory into a
structured, self-evolving knowledge brain; retire notes into it."** Memory is
the star; structure is the capability it grows.

## 3. Design constraints (owner-fixed)

1. **Capture is an automatic byproduct of working — never an explicit request.**
   *"保留知识库，杀死维护"* (keep the knowledge base, kill the maintenance).
2. **Entity/concept-centric, NOT project-centric.** A project is one node type
   + one scope level, not a partition wall.
3. **The system self-maintains** (consolidate / merge / promote / archive).
   Humans only steer and veto.
4. **Everything is trustworthy or refutable**: provenance + confidence on every
   node; one-click override.
5. **Anti-entropy is first-class**: the graph must **compress, not accumulate**.
6. **Memory stays the core and stays stable** (see §7 Decoupling). The
   structured layer is a *consumer* of memory, never a rewrite of it.

Non-goals: a human note-taking app; a folder hierarchy; a clone of any external
note app, or any external-tool compatibility goal — the system is **DB-native and
self-contained**; replacing git/Issues as the truth for code and tasks.

## 4. Target architecture — the memory-rooted knowledge brain

### 4.1 Two tiers, one brain

```
            ┌──────────────────── the knowledge brain ─────────────────────┐
   work ──▶ │  EPISODIC TIER (today's memory — UNCHANGED)                   │
            │    fragments · journal · goal/plan · auto-capture · pgvector  │
            │                          │ consolidation (reflection)         │
            │                          ▼                                    │
            │  STRUCTURED TIER (NEW — internal/knowledge)                   │
            │    entity · fact · playbook · skill   (typed graph)           │
            │    references fragments, does NOT copy them                   │
            └──────────────────────────────────────────────────────────────┘
                       ▲ read/correct (human)        ▼ inject/apply (agents)
```

The episodic tier is **exactly today's memory** — we do not touch its core. The
structured tier is the new layer that **distils** episodic experience into
durable, transferable knowledge. They are two tiers of **one** brain, not two
products.

### 4.2 The structured layer — four node kinds (declarative / procedural / skill)

| kind | knowledge type | answers | renders to |
|---|---|---|---|
| `entity` | declarative · **backbone** | what exists & how it connects | graph hub / auto index page |
| `fact` | declarative · **assertion** | what's true about it | **a reference to one or more episodic memory rows**, anchored to an entity |
| `playbook` | **procedural** | how to do this class of work; pitfalls | an SOP doc |
| `skill` | **skill** | an invocable capability | a `SKILL.md` the agent auto-loads |

Crucially, a `fact` node is **not a copy** of a memory row — it is an
*anchoring* of existing episodic fragment(s) to an entity in the graph. This is
the anti-redundancy mechanism: one store, the structured layer adds *structure
over* the fragments, not a second set of them.

(Compact schemas grounded in the real homelab are in Appendix A at the bottom.)

### 4.3 DB-native, with an optional owned-file export

The system is **DB-native**: structured nodes live in pgvector alongside memory
— the single source of truth for reasoning (graph traversal + semantic search +
cross-project). There is no file-based store.

For local-first **data ownership**, Phase 6 adds an *optional* renderer that
exports nodes to portable plain-text markdown you can keep and git-sync. This is
an export for ownership/portability only — **not** a store and **not** a
compatibility target for any external tool. It honours opendray's "data never
leaves your network / Apache-2.0 / self-hosted" ethos without tying the design
to any note app.

### 4.4 The graph — closed edge vocabulary, no orphans

Typed edges from a **closed vocabulary** — `runs_on`, `uses`, `about`,
`part_of`, `depends_on`, `supersedes`, `derived_from`, `used_by`. No free-form
relation types (untyped free-association is what rots large note graphs). Entities are
**hubs**; facts/playbooks/skills anchor to them. **No orphans** — every node
joins the graph at creation.

### 4.5 The maturity axis = self-evolution

```
fact ──(recurrence / confirmation)──▶ playbook ──(reliable application)──▶ skill
"knows that"        reflection rolls up        "knows how"    proven & packaged   "can do"
```

Promotion is driven by `evidence` (observed / applied / success_rate) +
`provenance`. Every promotion is **reversible and operator-vetoable**. This
axis *is* the literal mechanism of "the AI gives itself new capability."

### 4.6 Scope = transfer radius (subsumes today's project isolation)

`project` (cwd) / `domain` (e.g. `go+proxmox`) / `global`. Isolation becomes a
**per-node property**: project-private facts stay isolated (today's no-leak
guarantee preserved); generalizable playbooks/skills live at `domain|global`
and transfer. The new model strictly subsumes the old all-or-nothing isolation.

## 5. Anti-entropy — the "won't it become an unreadable mess?" guarantee

Three causes of knowledge-system death, all removed by design:

- **Accumulation → consolidation.** Reflection merges / promotes / archives;
  node count grows **sublinearly and can shrink** (100 raw observations → 10
  facts → 2 playbooks). Expertise is compression, not a growing pile.
- **Weak structure → ontology replaces folders.** Closed node kinds + entity
  types + edge vocabulary = stronger regularity than any folder tree.
  Navigation = queries / hubs / scope / maturity — never a flat scroll.
- **Manual maintenance → AI maintenance.** The death-by-neglect failure mode
  is removed; the system maintains itself.

Three coherence guards (this is "maintenance", done by the system):
1. **Entity canonicalization** — `Postgres` / `pg` / `192.168.3.88` → **one**
   node. The hardest, most critical task (see §9.1).
2. **No-orphan** — every fact/playbook anchors to ≥1 entity.
3. **Closed edge vocabulary** — relation types cannot proliferate.

**Reading = generated views, not browsing.** The operator reads a synthesised
"project brain" / "what I know about X", rendered on demand. The graph is the
source; readable docs are projections.

## 6. The consolidation / graduation engine (the heart)

The engine is both the self-evolution mechanism **and** the anti-entropy guard.

- **Triggers**: `session.ended` (existing event) + a periodic idle sweep.
- **Pipeline**: pull recent episodic fragments → **entity-resolve & anchor**
  (canonicalization) → cluster recurring facts → **promote to `playbook`** when
  a repeatable how-to emerges → track application outcomes → **promote reliable
  playbooks to `skill`** → archive stale/unused.
- **Workers**: reuse memory's pluggable worker layer (`internal/memory/worker/`,
  summarizer vs agent) — no new worker plumbing.
- **Governance**: every write carries provenance; thresholds tunable; operator
  vetoes/corrects any promotion; nothing auto-promotes below an evidence floor.

## 7. Decoupling — how we protect the just-shipped memory system

This is the architectural guarantee that #3 does **not** destabilise M-U.

1. **New package `internal/knowledge`** holds the entire structured tier.
   Nothing of M-U moves into it.
2. **Strict one-way dependency.** `internal/knowledge` *imports/reads*
   `internal/memory` + `internal/projectdoc`. **`internal/memory` never imports
   `internal/knowledge`.** Memory stays a self-contained island; if knowledge
   is buggy or disabled, memory runs **byte-identically to today**.
3. **Consume via public surface, not internals.** Knowledge subscribes to the
   event bus (`session.ended`, memory-written) and reads memory's public API —
   it never reaches into memory's private state. Clean seam.
4. **Additive schema only.** `knowledge_nodes` / `knowledge_edges` reference
   `memories.id` by FK; the `memories` table is **untouched**. Migrations
   auto-apply on startup, fail-closed (M-U pattern).
5. **Shared infra by injection, not entanglement.** The embedder, worker
   registry, and pgvector pool are *passed in* (DI), not imported across the
   package boundary as logic.
6. **Feature-flagged.** `[knowledge] enabled = false` by default. Off ⇒ pure
   M-U behaviour. This is the rollback switch and the canary gate.

Net effect: **memory gains a consumer, not a rewrite.** The freshly-stabilised
system is never at risk.

## 8. Implementation steps & methods (Phase 0–7)

Each phase: additive, behind the flag, reversible. Memory online throughout.

### Phase 0 — Structured-tier substrate
- **Goal**: tables + ontology, no intelligence yet.
- **Method**: migrations for `knowledge_nodes` (id, kind, entity_type, title,
  body, scope, maturity, confidence, provenance jsonb, embedding vector,
  archived_at) and `knowledge_edges` (src, edge_type, dst); enforce closed
  `kind` / `entity_type` / `edge_type` enums in code. New `internal/knowledge`
  package skeleton + handler mounted under the existing dual-auth group.
- **Reuses**: pgvector, auto-migration, dual-auth.
- **Disruption**: zero — pure new tables, flag off.
- **Exit**: can CRUD nodes/edges via API; nothing reads memory yet.

### Phase 1 — Capture → entity anchoring (canonicalization core)
- **Goal**: episodic fragments auto-anchor to canonical entities (no-orphan).
- **Method**: on `memory-written` / idle sweep, extract entity mentions
  (rules + embedding-NN + LLM adjudication, see §9.1), resolve to existing
  entity or create one, write a `fact` node referencing the memory row + an
  `about` edge. Human merge/split UI for corrections.
- **Reuses**: existing embedder; worker layer for LLM adjudication.
- **Disruption**: read-only consumption of memory; additive writes to knowledge.
- **Exit**: every new fragment is reachable from an entity hub.

### Phase 2 — Retrieval + generated views
- **Goal**: query the graph; render "project brain" / "entity" pages.
- **Method**: graph traversal (recursive CTE, bounded depth) + semantic search
  over nodes (shared embedding space with fragments → directly comparable);
  a `knowledge_search` that ranks across tiers; a view-renderer that assembles
  a page from a node's neighbourhood. Extend, do not replace, memory retrieval.
- **Reuses**: M-U ranker (`ranking.go`), injector.
- **Exit**: ask "what do I know about X" and get a synthesised page.

### Phase 3 — Graduation engine: `fact` → `playbook`
- **Goal**: distil recurring facts/experiences into procedural playbooks.
- **Method**: a `reflect` worker task; clusters recurring facts around an
  entity/activity, drafts a playbook (steps + pitfalls), files it with
  provenance + evidence. Operator can veto.
- **Reuses**: pluggable worker layer (summarizer/agent).
- **Exit**: ≥1 playbook auto-distilled from real session history.

### Phase 4 — Graduation engine: `playbook` → `skill`
- **Goal**: promote reliable playbooks into invocable skills.
- **Method**: track playbook application outcomes; above a success-rate floor,
  render the playbook to a `SKILL.md` and register it with `internal/skills`
  so future agents auto-load it.
- **Reuses**: existing `internal/skills` loader.
- **Exit**: an AI-generated skill is auto-loaded and used in a later session.

### Phase 5 — Cross-project transfer (scope = domain)
- **Goal**: knowledge learned in project A reaches project B.
- **Method**: add `domain` to the scope enum; promotion may lift a node from
  `project` to `domain|global` when it recurs across cwds; injection/search
  include in-scope domain/global nodes for the active project.
- **Reuses**: M-U scope model + injector.
- **Exit**: a new cwd opens with relevant domain playbooks already present.

### Phase 6 — Owned-file face + anti-entropy maintenance loop
- **Goal**: portable markdown files + self-cleaning.
- **Method**: bidirectional node↔file sync (render to vault; ingest external
  edits); the consolidation loop (merge duplicates, archive stale/unused,
  recompute confidence); coherence dashboard.
- **Reuses**: M-U cleaner patterns, soft-archive lifecycle.
- **Exit**: node count plateaus/shrinks under load; files are git-syncable.

### Phase 7 — Retire `notes`; UI cut-over
- **Goal**: one system; remove the old vault.
- **Method**: import the notes vault as seed nodes; move the web/mobile Notes UI
  onto the knowledge brain; deprecate `internal/notes` + `notes-keeper`.
  Web + mobile + i18n updated together (slang regen, dead-key removal).
- **Exit**: `internal/notes` removed; the brain is the only knowledge surface.

## 9. Problems we will face & how we solve / decouple them

### 9.1 Entity canonicalization (the make-or-break problem)
*Why hard*: the same thing appears as `Postgres` / `pg` / `192.168.3.88`; if it
fragments into 3 nodes the graph shatters. *Solution*: a layered resolver —
(1) exact/alias match, (2) embedding nearest-neighbour above a high threshold,
(3) LLM adjudication for the ambiguous middle, (4) operator merge/split UI as
the backstop; every merge keeps a reversible `merged_from` trail (M-U already
does this for facts). *Decouple*: canonicalization is its own worker task,
swappable and independently tunable.

### 9.2 Memory's cwd-scoping vs a cross-project graph
*Why hard*: memory was built project-isolated; the graph wants to cross
projects. *Solution*: scope is a **per-node** property; project-scoped nodes
keep today's isolation, only `domain|global` nodes cross. The graph layer sits
*above* cwd; memory's isolation guarantees are untouched.

### 9.3 LLM cost & latency of reflection at scale
*Solution*: reflection runs on `session.ended` + batched idle sweeps, never on
the hot path; per-cwd LLM time budgets (M-U conflict-detector already does
this); the pluggable worker lets you pick a cheap local model for bulk and a
frontier model only for hard promotions.

### 9.4 Trust / hallucination in auto-distilled knowledge
*Why hard*: auto-generated expertise that's wrong is worse than none.
*Solution*: provenance on every node (which sessions/fragments it came from);
confidence scores; `candidate → confirmed` states; nothing auto-promotes below
an evidence floor; one-click operator veto; the dual-face file lets you read &
correct. Trust is a first-class UI surface, not an afterthought.

### 9.5 Migrating a live, just-shipped system
*Solution*: additive migrations only (no `memories` schema change); auto-apply
on startup, fail-closed (M-U pattern); everything behind the flag so a bad
migration can be neutralised by disabling the feature.

### 9.6 Graph traversal performance in Postgres
*Solution*: bounded-depth recursive CTEs; indexes on `(src, edge_type)`;
materialised "entity neighbourhood" views for hot pages; cap fan-out.

### 9.7 Preserving local-first data ownership when the vault retires
*Solution*: the optional export renderer (§4.3) writes structured nodes to owned,
portable plain-text markdown you can git-sync — local-first ownership is
preserved with no dependency on, or compatibility target of, any external tool.

### 9.8 Rollback safety
*Solution*: the one-way dependency (§7) + feature flag mean disabling knowledge
returns the system to **exact** M-U behaviour with no data migration needed.

## 10. Production rollout (how it actually ships)

opendray ships as a **single Go binary**; users upgrade with `opendray update`;
migrations **auto-apply on startup** (fail-closed). The rollout is therefore:

1. **Non-destructive upgrade.** New additive tables; existing memory data
   untouched; existing users upgrade losslessly with no manual SQL.
2. **Default-off flag.** `[knowledge] enabled = false` ships first — zero
   behavioural change for everyone. We dogfood it **on opendray's own repo**
   (the gateway driving its own development is the perfect test corpus).
3. **Background backfill.** When enabled, a low-priority goroutine distils
   existing `memories` + imports the `notes` vault into the graph (idle when
   caught up) — the first self-distillation pass; no big-bang.
4. **Canary → on.** Flip on for the operator's own instance, validate value
   (a new project opens with cross-project playbooks; an AI-generated skill is
   used), then make it the default in a later release.
5. **Observability.** `opendray doctor` gains a knowledge-health check
   (node/edge counts, orphan count = 0, canonicalization collision rate,
   compression ratio); a Health tab surfaces "is the brain working?".
6. **Surfaces.** Web + mobile read-views + the integration API expose the brain
   so third-party callers (opendray's gateway value) get the same knowledge.

## 11. Value & irreplaceability (why this is a moat)

This is not a feature; it is what makes opendray **uncopyable**:

- **Cross-CLI brain.** opendray sits *above* Claude Code / Codex / Gemini /
  shell. Its knowledge brain is the **only** one that learns across *all* your
  agents — what Claude learned, Codex applies. No single CLI can structurally
  do this.
- **Gateway vantage = the only cross-project corpus.** opendray sees **every**
  project across **every** CLI. Only it can distil **cross-project expertise**;
  a single-session CLI sees one repo and can never generalise across your work.
- **Local-first self-evolution.** Your expertise, your data, your models — a
  compounding brain that never leaves your network. No SaaS can match this
  without holding your knowledge hostage; opendray's Apache-2.0 / self-hosted
  stance makes private, owned self-evolution its native ground.
- **Compounding switching cost.** The more you use opendray, the more your
  instance knows *your* stack, *your* habits, *your* pitfalls. Your opendray
  becomes *yours* — irreplaceable because it has accumulated your engineering
  expertise. This is the deepest moat: not features, but accumulated, owned,
  compounding capability.
- **Category shift.** It turns opendray from "a remote terminal for AI CLIs"
  into "an engineering brain that gets measurably better the more you work" —
  the thing that drives work, not just relays it.

## 12. Migration / transition

- **Seed, don't start empty**: backfill `entity`/`fact` nodes from existing
  `memories` (entity extraction) and import the current notes vault as nodes.
  Existing experience becomes the first distillation pass — nothing thrown away.
- **Coexist while building**: stand the structured tier up beside memory behind
  the flag; cut the notes UI over only at parity; memory runs throughout.

## 13. Success criteria for the arc

- A **new** project opens with relevant **cross-project** playbooks already
  present (transfer works).
- Node count **plateaus or shrinks** as understanding matures (compression is
  measurable, not theoretical).
- The operator can read a **one-page synthesised brain** for any project/entity
  without ever browsing raw nodes.
- The AI **self-generates ≥1 skill** from accumulated playbooks and uses it in
  a later session (self-evolution demonstrated end-to-end).
- **Zero manual note-writing** is required for the knowledge base to grow.
- Disabling the flag returns the system to **exact** M-U behaviour (decoupling
  proven).

## 14. Open decisions (remaining)

- **D-a** `domain` scope taxonomy: free-form tags vs a curated set (e.g.
  `go+proxmox`, `flutter`, `homelab-ops`).
- **D-b** Optional export format for the owned-file face — opendray's own schema (no external-tool compatibility target).
- **D-c** First canary corpus: opendray-v2 repo only, or a couple of the
  operator's active projects.
- **D-d** Branch/arc name (`feat/knowledge-graph` / `M-KG`) — keep or rename
  now that the framing is "evolve memory", not "rebuild notes".

---

## Appendix A — node schemas (grounded in the homelab)

```yaml
# entity — a graph hub
id: svc-postgres-dev
kind: entity
entity_type: service        # service|host|project|tool|decision|tech|person
name: PostgreSQL (Dev)
aliases: [dev-db, 192.168.3.88:5432]
relations: { runs_on: [host-192.168.3.88], used_by: [proj-materialscout, proj-flashmind] }
scope: global
provenance: { source: human }
```
```yaml
# fact — references episodic memory rows, anchored (NOT a copy)
id: fact-pg-per-project-user
kind: fact
about: [svc-postgres-dev]
sources: [mem_8821, mem_9012]      # FK into memories — no duplication
assertion: "Each project gets its own CRUD user (<abbr>_user, 16+ alnum); never connect business code as superuser linivek"
scope: global
confidence: 0.95
evidence: { observed: 4 }
provenance: { source: reflection }
```
```yaml
# playbook — procedural know-how
id: pb-deploy-go-proxmox
kind: playbook
title: Deploy a Go service to Proxmox LXC
applies_when: "shipping a Go API backend to the homelab"
uses: [svc-postgres-dev, host-kv01, fact-lxc-id-range]
scope: domain:go+proxmox
maturity: playbook
evidence: { applied: 6, ok: 5, fail: 1 }
# body: ordered steps + anti-patterns (TCC denial → Full Disk Access; never
#       pass special-char passwords via shell — write SQL into the container)
```
```yaml
# skill — promoted, invocable, renders to a SKILL.md
id: skill-provision-lxc
kind: skill
name: provision-lxc
when_to_use: "user asks to create a new homelab container/service"
params: [project, role, node]
derived_from: [pb-deploy-go-proxmox]
maturity: skill
evidence: { invoked: 12, success_rate: 0.92 }
```

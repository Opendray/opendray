# Cortex — the unified Memory · Notes · Knowledge module

> Status: IMPLEMENTED (backend + web), LOCAL on `feat/knowledge-graph`,
> unpushed and **not yet built/restarted** — the operator runs live
> opendray sessions on the current binary; the unified rebuild waits
> for their go-ahead. Mobile parity (Phase 6) is gated on the operator
> validating the web UI first. Supersedes
> `experience-flywheel-architecture.md` (the flywheel MODEL it defined
> still holds; Cortex is its completion).

## 0. Why Cortex (not another patch)

The Experience Flywheel closed the loop on paper, but four operator
complaints showed the system still fell short:

1. **Notes were caged.** Every project — mobile app, service, CLI —
   carried the same hardcoded goal/plan/tech/activity tabs (`Kind` was
   a closed enum + DB CHECK). There was no way to actively ask the AI
   to update a doc; drift detection fired passively on session-end and
   covered only goal/plan, always as proposals that piled up — so
   goal/plan/tech-stack never actually tracked the work.
2. **Memory was polluted.** opendray is a third-party-consumable
   platform, but sessions created via scoped API keys were
   indistinguishable from the operator's own: their temp sessions fed
   durable memory, and frozen/abandoned projects kept feeding the
   Anchor and KB consolidation stages (only the Reflector honoured
   lifecycle).
3. **Foundational knowledge had no governance channel.** The
   infrastructure/conventions pages were either AI-drafted or
   human-locked — a binary switch with no way to *discuss and re-draft
   policy with the AI*.
4. **The triad was presented as three silos.** Memory, Notes and
   Knowledge are one organism — three rungs of one loop — yet the UI
   gave them three disconnected nav tabs.

**The decision: one module, named Cortex (心智中枢), governing all
three rungs.** AI leads, human supervises; promotion between rungs is
transformation, never copy — both invariants inherited from the
flywheel doc and now enforced by mechanism rather than convention.

## 1. Architecture

`internal/cortex` is a **facade/orchestration package**, not a physical
merge: `memory`, `projectdoc`, `knowledge` keep owning their plumbing
(capture, consolidation, proposals, mirroring, embeddings). Cortex owns
only what is genuinely cross-layer, and depends one-way on the three:

- the unified HTTP namespace `/api/v1/cortex/*` — the three layer
  handler sets are **dual-mounted** under it (legacy mounts stay
  untouched for integrations + the not-yet-migrated mobile app)
- flywheel status aggregation (`GET /cortex/status`)
- the quarantine review queue (§3)
- the blueprint proposer (§2)
- curation conversations (§4)

## 2. Notes: the doc blueprint system (Phase 3)

`project_docs.kind` is reinterpreted as a per-project **section slug**
(migration 0046, zero row rewrites). A new `doc_blueprint_sections`
table declares each project's section set:

- `(cwd, slug, title, description, position, maintainer_mode
  ai|human|scanner, prompt_hint, pinned, inject)`
- `overview` is the reserved, pinned, undeletable front page
  (inject=false — it restates the others, so it stays out of spawn).
- kb_* slugs remain fixed global pages under `__global__`, governed by
  the Knowledge layer — never part of a project blueprint.
- Defaults seed 1:1 from the legacy kinds (per existing cwd in the
  migration; lazily for new cwds), so pre-blueprint behaviour is the
  starting point, not a regression.
- The AI **blueprint proposer** (worker task `blueprint`, operator-
  triggered via `POST /cortex/blueprint/propose`) classifies the
  project from scanner signals and proposes a tailored section set —
  applied only on operator accept (`PUT /cortex/blueprint`).
- Deleting a section hides it without deleting its doc row; re-adding
  the slug resurrects the content.

**Docs now track the work.** Drift detection walks the blueprint after
each session: an **ai-maintained, unlocked** section is auto-updated
(PutDoc as agent); a human edit (updated_by=operator) or human
maintainer mode keeps the proposal gate. Scanner sections are rebuilt
mechanically and respect blueprint removal. The spawn banner renders
in blueprint order (foundational rules still first and untruncatable);
the `.opendray/` mirror covers every section as `<SLUG>.md`.

**Latent bug fixed en route:** `project_doc_proposals.kind` still
carried 0025's `CHECK (kind IN ('goal','plan'))` — every KB-page /
overview update proposal (the B3 Iterate edge) had been failing at
INSERT. Expect a burst of KB proposals after the first consolidation
cycle on the new binary.

## 3. Memory: provenance + quarantine (Phase 2)

- `sessions.origin` (operator|integration|cli) + `integration_id`,
  derived from the authenticated principal at create time — never
  client-supplied (0044).
- `integrations.memory_policy` (none|quarantine|full), default
  quarantine; **system integrations backfill to full** (the
  auto-registered opendray-memory MCP serves the operator's own
  agents — quarantining it would break their memory pipeline).
- `memories.tier` (durable|quarantine) + TTL (0045). The filter lives
  in the store layer, so injector / anchorer / KB drafter / conflict
  detector / list+search APIs all see durable-only via one change.
  Quarantine writes skip dedup-merge so quarantined text can never
  launder into a durable row. The cleaner purges expired quarantine.
- Capture routing: policy none → the session produces no memory at
  all; quarantine → tier + 30d TTL + integration attribution in
  metadata; full/operator → durable. Direct `/memory/store` calls from
  integration keys route identically.
- Review queue: `GET /cortex/memory/quarantine`, promote / discard;
  count surfaces on `/cortex/status` and the web Memory page.
- **Flywheel leak fixed:** the Anchor stage and the KB drafter now
  honour project lifecycle — paused/archived projects stop feeding the
  graph and the cross-project pages.
- **Ephemeral cwds are not projects.** Sessions spawned in temp dirs
  (`/tmp`, `/var/folders`, `.cache`, … — third-party consumers and
  tests do this constantly) leave NO project footprint:
  `projectdoc.IsEphemeralCwd` (the predicate the knowledge anchorer
  always used, now canonical) gates the journaler, drift, capture
  (no memory at all, not even quarantine), both scanners, blueprint
  seeding, doc writes, and `ListProjects`. Migration 0049 purges the
  legacy residue (docs/journal/blueprint/lifecycle deleted; their
  project-scope memories soft-archived, restorable in the grace
  window).

## 4. The curation channel (Phase 4)

`cortex_conversations` + messages (0048): a thread bound to one target
— a project doc section, a global knowledge page, or a blueprint.
Turns dispatch through `worker.Registry` (task `curation`, strict JSON
schema). An AI reply may carry a structured revision:

- target ai-maintained + unlocked → **applied directly** (the
  conversation IS operator intent)
- human-locked or human-mode → **automatically downgraded to a
  proposal** — a conversation can never silently overwrite something a
  human claimed.

For the Foundational pages this is the governance loop the operator
asked for (重新制定方针): discuss infrastructure/conventions with the
AI, who re-drafts keeping the binding "## Rules (MUST follow)" shape.

Escalation (`POST /conversations/{id}/escalate`) spawns a real claude
session in the project cwd (gateway workspace for global pages),
seeded with the transcript + instructions to ground changes in the
codebase and file them through the standard proposal tools.

Replies land async; `cortex.conversation.reply` on the eventbus drives
live UI refresh (web also polls while a turn is in flight).

## 5. UI (web — Phase 5)

One nav entry **Cortex** replaces Notes/Memory/Knowledge:

- `/cortex` — the flywheel home: per-rung cards with live counts
  (pending proposals split project-vs-global, quarantine badge),
  active-project jump list.
- `/cortex/project` — the project workspace: tabs ARE the blueprint;
  per-section curation chat; blueprint editor (+ AI propose); journal;
  inbox; memory hygiene folded in as a tab.
- `/cortex/knowledge` — Foundational/Emergent with the governance
  chat on every page; proposal review inline.
- `/cortex/memory` — cross-project browser + quarantine / archived /
  workers.
- `/vault` unchanged (the freeform markdown/Obsidian utility).
- Session inspector Notes tab: the personal scratchpad lane stays; the
  vault "Project docs" lane is replaced by a read-only view of the
  Cortex official doc deep-linking into the workspace — resolving the
  "two notes systems" confusion for good.
- Legacy routes redirect; i18n parity en/es/zh.

## 6. Compatibility & rollout

- Old API routes (`/project-docs*`, `/memory*`, `/knowledge/*`) stay
  mounted — integrations and the mobile app keep working unchanged.
  Mobile migrates to `/cortex/*` in Phase 6 (gated on the operator
  validating web); the legacy mounts retire some release after that.
- Migrations 0044–0048 run on next binary start; committing them is
  safe — nothing reaches the live service until the operator triggers
  the unified rebuild + restart.
- Every phase committed green: `go build/test -race/vet`, `gofmt`,
  web `tsc` + vite build, i18n parity.

## 7. Post-restart watchlist

1. The proposals-CHECK fix may release a backlog of KB update
   proposals on the first consolidation cycle — watch the Inbox.
2. Existing sessions backfilled origin=operator (correct: they predate
   scoped-key session creation). Verify a fresh `odk_live_*` session
   records origin=integration and its facts land in quarantine.
3. Garbage cwds in project_docs each received 5 seeded blueprint
   sections — cosmetic; archive or reset those projects as desired.

## 8. End-to-end validation script (operator, post-rebuild)

1. Scoped-key session → memory lands in quarantine → promote works,
   discard works, count on /cortex shows live.
2. New project → blueprint lazily seeds → "AI propose" reshapes it →
   spawn banner reflects blueprint order + inject flags.
3. Curation chat on an ai-maintained section → direct update; on a
   locked Foundational page → proposal, never overwrite.
4. Escalate button spawns a session seeded with the conversation.
5. Old bookmarks (/notes, /memory, /knowledge) redirect; mobile (still
   on legacy routes) unaffected.

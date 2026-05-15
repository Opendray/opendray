# Project memory — overview

Project memory is the **per-cwd, structured layer** that sits on
top of the discrete-fact memory store covered in section 11.
Each Claude / Codex / Gemini session you spawn through opendray
boots up reading the same five-layer banner derived from the
current project's state — so a new agent walks in with the same
context the previous one was working with.

If section 11 is "what does the agent remember as discrete
facts?", this section is "what does the agent know about THIS
PROJECT specifically?".

## The five layers

Every spawn injects a ~4-5 KB markdown banner composed of:

| Layer | Source | Who edits | UI surface |
|---|---|---|---|
| **Tech stack & structure** | Auto-scanned marker files (`go.mod`, `package.json`, `pubspec.yaml`, …) + git HEAD + top-level dirs | Scanner only — refreshed when stale ≥ 6h on next spawn | Project → **Tech** tab (read-only) |
| **Project goal** | Operator-written paragraph: "what we are building" | You (agents can _propose_ via MCP, you approve) | Project → **Goal** tab |
| **Project plan** | Operator-written paragraph: "what we are doing now / next" | Same as goal | Project → **Plan** tab |
| **Recent activity** | LLM-summarised `git log --since 7d --stat` + hot-path file list | Scanner only — refreshed every 24h | Project → **Activity** tab (read-only) |
| **Recent journal** | Last 5 session-end summaries (auto-written when sessions stop) | Auto — one per session_end event | Project → **Journal** tab |

## Why this exists separately from L5 facts

Section 11's memory store treats every entry as a **discrete
fact** ranked by top-K similarity. That shape works for "user
prefers pnpm" but not for "here's the multi-paragraph plan we
agreed on Monday". Project memory uses:

- **Replace-in-place** for goal / plan (you edit the whole doc)
- **Append-only** for journal (one row per session_end)
- **Overwrite-on-scan** for tech_stack + recent_activity

…all stored in different tables (`project_docs`,
`project_doc_proposals`, `session_logs`) so each can be queried
and managed independently. The Project page surfaces them as
tabs; the spawn injector composes them into one banner.

## When to use each layer

You're trying to record… | Put it in… | Why
---|---|---
**A long-term project intention** ("ship cross-CLI memory") | Goal | One paragraph, replaces, shows in spawn banner verbatim
**Current sprint / what's next** ("Phase 2: M6 spawn injection") | Plan | Same as goal; updated as work progresses
**A discrete preference / fact** ("we use pnpm; bcrypt cost=12") | Memory store (section 11) | Top-K retrieval; embeds into ambient banner only when relevant
**A decision made this session** ("chose bcrypt over argon2 because…") | `decision_record` MCP tool → journal as `kind=decision` | Permanent audit trail per session
**Just what happened in this session** | Auto-journal (no action needed) | Session-end event triggers it

## Cross-CLI: same project, any agent

Claude / Codex / Gemini all read the same Project memory through
their respective native injection channels (`--append-system-prompt`,
`$CODEX_HOME/AGENTS.md`, `GEMINI.md`). Tell **Claude** the project
goal once via the Project → Goal tab; the next **Codex** spawn in
the same cwd quotes it back to you on demand. This is the whole
point of unified memory — no manual re-explanation per CLI switch.

## Tabs you'll see on the Project page

Beyond the 5 layers above, the Project page (`/memory/project`)
shows three operator-facing inboxes that landed in Phase A-D:

- **Health** *(Phase A)* — first tab. Aggregate signals across
  both memory subsystems (new facts / journal entries this week,
  capture engine fires, plan/goal age, pending proposals, top-hit
  fact, zero-hit stale-fact count).
- **Inbox** — agent-proposed goal/plan edits awaiting your
  approval (file via the `project_goal_set` / `project_plan_set`
  MCP tools).
- **Conflicts** *(Phase C/D)* — cross-layer contradiction detector
  inbox. Each row shows two conflicting claims + LLM evidence +
  Accept / Dismiss buttons. Phase D added per-side "Delete A / B"
  quick-actions with a preview-and-confirm dialog so you see the
  full fact text before pulling the trigger.
- **Cleanup** — LLM librarian's keep / stale / duplicate queue
  for layer-5 facts (covered in section 11).

## Where to next

- **02 — Day-to-day workflow** walks you through setting goal/plan,
  reviewing agent-proposed edits, and reading journal entries.
- **03 — Scanner & cleaner** covers the auto-managed L1 / L4 / L5
  maintenance (tech_stack scanner, git activity summariser, LLM
  cleanup librarian).
- **04 — Reset & troubleshooting** covers the Reset action,
  orphan scope_keys, and the M22 isolation guarantees.

# Project memory — day-to-day workflow

## Opening Project memory

Three entry points, all land on the same screen scoped to one cwd:

- **Web**: sidebar → **Memory** → **Project** button (or
  `Cmd-K → Project memory`)
- **Web from a running session**: right-hand Inspector → **Memory**
  tab → "Open project memory" button
- **Mobile**: bottom nav → **Memory** → top **Project** button (or
  session detail → 🏁 icon → jumps straight to the right cwd)

If you arrive without a cwd in the URL, you'll see a picker
listing every project that has stored memory. Truncated old data
shows up as `orphan` and is visually muted — see section 04 for
how to clean those up.

## Goal — the long-term intention

The **Goal** tab is a single markdown editor. One paragraph is
ideal:

> Ship the v2 backup format with WAL-aware incremental snapshots.

The agent reads this verbatim on every spawn into this cwd, so
keep it crisp. Avoid:

- ❌ "TODO: figure out backup" → too vague to act on
- ❌ A 5-paragraph essay → bloats the spawn banner
- ✅ "Ship the v2 backup format with WAL-aware incremental
  snapshots." → concrete, one sentence

Updated by: usually you. Last-edit timestamp + `updated_by` show
right above the editor. If an agent calls the `project_goal_set`
MCP tool, you see a **proposal** in the Inbox tab (see below) —
no silent overwrite.

## Plan — current state and what's next

The **Plan** tab is the same shape but meant to change more
often. Format suggestion (not enforced):

```
Phase 1: <currently working on this>
Phase 2: <next>
Phase 3: <after that>

Blocker: <if any>
```

When you finish Phase 1 you edit this doc directly to bump the
phases forward. The agent on the next spawn sees the updated
plan without you having to repeat anything in chat.

## Inbox — agent-proposed edits

The MCP tools `project_goal_set` and `project_plan_set` deliberately
don't update the live doc directly — they file a **proposal** that
appears in the **Inbox** tab. Each proposal shows:

- Strong red warning: "Approve will REPLACE the current X
  entirely. This isn't a merge."
- Side-by-side diff: current content vs proposed content
- The agent's stated reason for the edit
- "Approve" (with a confirm dialog) and "Reject" buttons

The diff matters: agents are eager and may try to rewrite your
hand-crafted goal with their interpretation. Reject when you
disagree, approve when their version is genuinely better.

## Journal — what happened, automatically

The **Journal** tab lists every session_end event for this cwd.
Each entry has:

- Session metadata: id (last 8 chars), provider, duration, exit
  code
- Recent operator inputs (last 5 user-typed messages of that
  session)
- _Agent activity summary_ — 1-3 paragraphs of LLM-summarised
  "what the agent actually did" (files edited, decisions made,
  problems debugged, blockers hit). **Only present when the
  session had substantive work** — a 30-second "hi" exchange is
  correctly skipped per the "too sparse" guard in the prompt.

You don't write to this tab; opendray writes it on every
`session.stopped` / `session.ended` event.

If you want to record something manually (e.g. an architectural
decision you made out-of-band), use the MCP `decision_record`
tool from any agent session — it writes a `kind=decision` row
that appears here.

## Practical rhythm

Most operators settle into this loop:

1. Start a project: write a 1-sentence **Goal** and 3-phase **Plan**.
2. Spawn an agent (Claude / Codex / Gemini) → it boots already
   knowing what the project is about.
3. Work. The agent's session-end automatically appends a
   journal entry.
4. Plan changes mid-work → edit **Plan**. Next spawn sees the
   update.
5. Agent proposes a new plan → review in **Inbox**, approve or
   reject.
6. Days later, a new agent (or you on a new machine) spawns →
   reads all 5 layers, picks up where the last one left off.

Section 03 covers the scanner and cleaner that maintain the
auto-managed layers without your direct involvement.

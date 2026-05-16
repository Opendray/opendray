# Inspector panel

The Inspector is the collapsible right-side panel on the Sessions
page. It carries metadata, file browsing, history, and tooling
that doesn't fit inline in the terminal.

![Inspector panel tabs](/tutorial/sessions-inspector.png)

Toggle the panel with the inspector-toggle icon in the top-right
of the WorkbenchHeader (next to the session controls). The open
state persists per-user across reloads.

## Sub-tabs

The Inspector exposes seven tabs in a 3-row grid (4 + 2 + 1):

| Row | Tabs |
|---|---|
| 1 | Files · Git · Search · Tasks |
| 2 | History · Notes |
| 3 | Memory |

All tabs scope to the **session's `cwd`** — there's nothing
showing data outside that working directory.

### Files

A scrollable tree of the session's working directory. Helpful
when you've forgotten what files are in the project, or when
the agent references a file you want to peek at without leaving
the terminal.

Click a file → opens a read-only viewer in the Inspector.

The tree is collapsed by default to 3 levels deep so
`node_modules` or `.venv` doesn't blow it up; expand specific
subtrees on demand.

### Git

A full Git workbench for the cwd: branch switching, staging,
commits, push, and the in-panel pull-request command center.
Only shown when the cwd is inside a git repo. See
[Git workflow](#02-sessions-07-git-workflow) for the full tour.

### Search

Substring + regex search across the cwd. Powered by the same
ripgrep wrapper the FS handler uses on the backend, so it
respects `.gitignore` and skips `node_modules`. Match results
link to the file viewer.

### Tasks

The session-scoped Custom Tasks runner. Every task you've saved
under Plugins → Custom Tasks shows up here as a one-click
launcher; the run spawns a **new** session in the same cwd
under the parent session, so `g s` shows them grouped.

### History

**Project-level** input log: every prompt the operator has sent
in this cwd, pooled across every session ever spawned there.

Source of truth varies by provider:

| Provider | Reads from |
|---|---|
| Claude | `~/.claude/projects/<encoded-cwd>/*.jsonl` and `~/.claude-accounts/*/projects/...` |
| Codex | `~/.codex/sessions/**/*.jsonl` filtered by `session_meta.cwd` |
| Gemini | `~/.gemini/tmp/<dir>/logs.json` (resolved via `projects.json` short-name) |

These paths are configurable in **Settings → Server → Storage
paths** (one section per provider). For non-supported providers
(shell, etc.) the panel shows a friendly empty state.

Each row carries:

- Timestamp (relative — "2 hours ago")
- The prompt text (wrapped, never truncated)
- The CLI session id (8-char prefix; full id on hover)

Per-row actions appear on hover:

- **📋 Copy** — prompt text to clipboard.
- **➤ Resend** — re-injects the prompt into the **currently
  active** session via the same `/input` endpoint the terminal
  uses. Uses `\r` (raw-mode Enter), not `\n`, so Claude doesn't
  see a literal newline in the prompt.

Newest entries first. Filter box at the top is case-insensitive
substring match against prompt text. Polls every 10 s so newly
typed prompts show up automatically.

### Notes

The session's linked Obsidian note. Each session gets one note
at `<vault-root>/sessions/<session-id>.md` automatically. The
Inspector embeds the same Markdown editor you'd see on the
Notes page:

- **Source / Preview** tabs at the top of the note pane — pick
  whichever matches the moment.
- **Wiki-link suggestions** trigger when you type `[[`.
- **Backlinks** show on the right (other notes that link to
  this one).
- **Auto-save** debounced 1 s after the last keystroke.

This is the right place for the operator's running scratchpad:
"things to ask Claude about", "pending decisions", "TODO before
ending the session". Survives session restart because it's
file-based, not in-memory.

## What's NOT in the Inspector

- **Lifecycle event timeline** — the per-session "what happened
  when" view used to live here as an "Activity" tab. It was
  removed in favour of History because raw event streams turned
  out to be low-signal for vibe-coding. The system-wide event
  bus is still available on the **Activity** page (top nav).
- **Outline** of the latest assistant message — also previously
  here, since dropped. Use Search instead, or the agent's own
  scrollback.

## Sizing and hiding

The panel anchors to the right edge of the workbench and is
**user-resizable**:

- **Drag the left edge** to resize. A 6-pixel column on the
  inspector's left edge highlights on hover; press and drag to
  any width between **320 px** (default) and **900 px**.
- **Double-click** the same edge to snap back to the default
  width.
- The width is persisted per-user (zustand `opendray.layout` in
  `localStorage`), so it survives reloads and re-spawned
  sessions.

To hide the panel entirely, click the inspector-toggle icon in
the WorkbenchHeader. The open/closed state is persisted the same
way.

# Spawning a session

Click **New session** in the top-right of the Sessions page. The
spawn dialog has every field you need to launch a CLI under
opendray's lifecycle management.

![Spawn dialog](/tutorial/spawn-dialog.png)

## Required fields

### Provider

Pick the CLI to launch. The dropdown lists every provider
configured under [Providers](#providers-overview). Bundled
defaults: Claude, Codex, Gemini. The provider's icon and display
name show up here so you don't accidentally launch the wrong one
on a multi-CLI host.

### Working directory

The session's `pwd`. Two ways to fill it:

- **Type the absolute path** directly into the input.
- **Click the 📁 button** to open a file browser rooted at your
  home directory. Navigate, select a folder, click *Use this
  directory*.

opendray rejects paths that don't exist or aren't readable —
`stat()` is called pre-flight so you don't get an opaque
"fork/exec: no such file or directory" later.

## Optional fields

### Name

Friendly tab label. Default = the directory's basename (e.g.
`/Users/me/projects/foo` → `foo`). Override when you want a
custom label (e.g. `foo (refactor)`).

### Args

Extra CLI flags appended to the provider's defaults. Examples:

- Claude: `--continue` to resume the latest conversation in this
  cwd
- Codex: `--model gpt-5` to override the default
- Plain shell: leave blank

The provider's bundled args are not editable from here — they
live in the provider manifest (Providers page).

### Claude account

**Only shown when provider = `claude`.** Drops down all accounts
registered in **Providers → Claude accounts**. Picking one binds
the session to that credential — Claude Code reads
`~/.claude-accounts/<name>/` instead of the default
`~/.claude/`, so you can run `personal` and `work` accounts in
parallel without re-login dance.

The account binding is stored on the session row, so a Restart
of an ended session reuses the same account.

**Picker behaviour by account count:**

- **0 accounts registered** — picker hides entirely; the session
  spawns against the system `ANTHROPIC_API_KEY`.
- **1 account registered** — picker shows `Default (env / system)`
  plus that one account. Either is a sensible pick.
- **2+ accounts registered** — `Default` disappears. Once you've
  registered multiple accounts the "fall back to env" choice almost
  always isn't what you want for the next spawn, so opendray
  forces an explicit account pick. The first enabled account
  pre-selects automatically.

### Bypass / autonomy toggle

**Shown for `claude`, `codex`, `gemini`.** A per-session opt-in
to the provider's "skip the safety prompts" mode. Off by default;
flip it on when you trust the session to run unattended.

The flag is provider-specific:

| Provider | Toggle label | Appended flag(s) |
|---|---|---|
| Claude | `Bypass permission prompts` | `--dangerously-skip-permissions` |
| Codex | `Bypass approvals & sandbox` | `--dangerously-bypass-approvals-and-sandbox` |
| Gemini | `YOLO mode (--yolo)` | `--yolo` |

The toggle is **additive** — it doesn't disable a provider-wide
bypass set in **Providers → Claude / Codex / Gemini → Bypass /
Approval / YOLO**. If the provider config already has bypass on,
every session bypasses regardless of this toggle. Leave the
provider default off if you want per-session control.

Per-session bypass flags are appended *before* anything you type
into **Args** above. When the same flag exists in the provider's
saved config (e.g. codex's saved `--ask-for-approval on-request`),
the session-level value wins — opendray drops the provider's copy
to avoid duplicates that codex's parser would reject.

For codex specifically: `--ask-for-approval never` only
auto-approves shell exec commands, **not** MCP tool calls. The
session toggle uses `--dangerously-bypass-approvals-and-sandbox`
instead, which is codex's "skip everything" switch (also disables
the workspace sandbox).

The toggle resets to OFF on provider change + after each spawn, so
the next session is always a deliberate opt-in.

### Parent session

Drop-down of recently-stopped sessions. Selecting one fills the
spawn dialog with that session's provider + cwd + args + claude
account, ready to relaunch under a new id.

Useful patterns:

- A Claude session crashed mid-task → fork from it, same context
  comes back.
- Compare two model variants on the same task → spawn from the
  same parent twice with different `--model` args.

opendray persists `parent_session_id` so the family tree is
queryable later.

## What happens when you hit Spawn

1. opendray validates the cwd, generates a session id, inserts
   the DB row in state `STARTING`.
2. Forks a PTY, runs the provider's executable with the merged
   args + env.
3. Hooks up the stdout pump (writes to ring buffer + fans out to
   any subscribed WebSocket clients).
4. Marks state `RUNNING` once the first byte arrives, or after
   500ms if the process is silent on launch.
5. Switches the active tab to the new session.

If the launch fails (binary not found, cwd unreadable, exit-on-
start), the dialog stays open with a red banner showing the
specific error from `cmd.Start`.

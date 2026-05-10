# Claude accounts

opendray supports running multiple Claude accounts side-by-side
on the same gateway — for example a personal account and a work
account, or two different subscription tiers. Each session can be
bound to a specific account; switching between them is a one-click
operation that doesn't disturb anything else on the gateway.

This page explains the architecture (what is shared across
accounts, what is per-account), the canonical way to set a new
account up, and why opendray does not offer a "paste token" form.

## Architecture: what's shared, what isn't

Every account boils down to one filesystem directory under
`~/.claude-accounts/<name>/`. Claude Code reads its OAuth
credentials, model defaults, and recent-files cache from that
directory when the gateway sets `CLAUDE_CONFIG_DIR=<that-path>`
on the spawned process.

What that means in practice:

| Surface           | Per account?                       | Shared across all accounts? |
|-------------------|------------------------------------|-----------------------------|
| OAuth credentials | yes — `<dir>/.claude/.credentials.json` | no                      |
| Model defaults    | yes — Claude Code stores per-`CLAUDE_CONFIG_DIR` | no                |
| Anthropic billing | yes (each token is an account)     | no                          |
| Session list & state | no                              | **yes** — sessions table   |
| Memory (pgvector) | no                                 | **yes** — global / project / session scopes |
| Notes vault       | no                                 | **yes** — single vault on gateway disk |
| Channels (Slack / Feishu / …) | no                     | **yes** — channels are gateway-level |
| Integrations (third-party API callers) | no            | **yes** — integrations are gateway-level |
| Backups / schedules / targets | no                      | **yes** — gateway-level |

So switching a session from `personal` to `work` only swaps which
Anthropic identity executes the next API call. Everything else —
the conversation history, the memory the session has built up, the
notes it has written, the channels that get notified when it goes
idle — stays exactly where it was.

This is the design point of opendray's multi-account model: the
account is **just an authentication identity**, not a sandbox.

### Worked example

Imagine three sessions running against a single shared notes vault
and the same memory store:

```
session-A   provider=claude   account=personal
session-B   provider=claude   account=work
session-C   provider=codex                       (no account binding)
```

Memory written by session-A under the `project:my-app` scope is
visible to session-B's next memory.search call, even though they
are different Anthropic identities. Notes written by session-C
appear in the inspector of all three. The "account" boundary
intentionally doesn't exist in opendray's data model; it lives
purely at the OAuth layer.

If you ever do want hard isolation between two accounts (separate
notes, separate memory), run two opendray gateways on different
ports, each with its own database.

## Setting up a new Claude account

Account setup is a single host-shell command. The gateway watches
`~/.claude-accounts/` for new directories and registers a
`claude_accounts` row when one appears, so there is no separate
"create row" step in the UI — the row materialises from the
filesystem.

### Step 1 — Run `claude login` under a per-account config dir

On the gateway host (over SSH, `docker exec`, or however you
normally reach it):

```bash
# Pick a short slug for the account.
NAME=work

# Create the dir and run the official Claude OAuth flow under it.
# Claude Code writes its credentials.json relative to
# $CLAUDE_CONFIG_DIR, so the file lands in the right place.
mkdir -p "$HOME/.claude-accounts/$NAME"
CLAUDE_CONFIG_DIR="$HOME/.claude-accounts/$NAME" claude login
# … walk through the browser OAuth …
```

The token file Claude Code wrote is a self-managing credentials
blob — Claude Code's own refresh logic handles expiry, so the
account stays usable indefinitely without further attention.

Repeat for each account:

```bash
for n in personal work labs ; do
  mkdir -p "$HOME/.claude-accounts/$n"
  CLAUDE_CONFIG_DIR="$HOME/.claude-accounts/$n" claude login
done
```

### Step 2 — Make opendray see the new directory

The gateway's filesystem watcher picks up the new directory on its
next sweep, but you can force a synchronous scan with the
**Import local** button in the web panel. That triggers
`POST /api/v1/claude-accounts/import-local`, which scans
`~/.claude-accounts/` on the gateway's own host filesystem and
registers every directory it finds that doesn't already have a
matching row.

| Works for          | Doesn't work for                                                     |
|--------------------|----------------------------------------------------------------------|
| Bare metal gateway | Docker container without `$HOME` bind-mounted in                     |
| LXC where the operator's home is reachable | Remote-managed gateway you don't have shell on |
| Dev environments  | Mobile (the import button is intentionally web-only — there is nothing on a phone for the gateway to import from) |

If `Import local` reports "Nothing to import — accounts already in
sync" but you don't see the row, the gateway's `$HOME` from inside
its runtime probably looks empty. Confirm with `docker exec` or
`pct exec` and adjust your volume mount.

### Why is there no "Add account" form?

Earlier versions of the panel had **+ Add account** alongside
**Import local**. It was removed because pasting an OAuth token
into a web form produces an account that opendray cannot refresh
(the public Anthropic API surface doesn't include a refresh
endpoint), so the account died within the hour. Forcing the
host-shell `claude login` flow keeps the affordance honest: every
account in the panel is one Claude Code itself is managing.

If you have a real reason to seed a row programmatically (for
example a one-off short-lived access token in a CI pipeline), the
underlying API endpoints are still there:

- `POST /api/v1/claude-accounts` — create the row
- `PUT /api/v1/claude-accounts/{id}/token` — write the token file

They are intentionally not surfaced as UI affordances.

## Binding a session to an account

In the spawn dialog, the **Claude account** dropdown only appears
when provider = `claude`. Pick the account; opendray sets
`CLAUDE_CONFIG_DIR` for the spawned process so Claude Code reads
from the right directory.

The binding is persisted on the session row (`claude_account_id`)
so a Restart of an ended session reuses the same account. There
is no UI affordance for "binding a session to two accounts" —
sessions are 1:1 with accounts at any given moment.

## Switching mid-session

Sessions page → terminal pane → **Account switcher** dropdown
(top-right of the terminal). Picking a different account:

1. Sends SIGTERM to the running process.
2. Waits for clean exit.
3. Re-spawns the same provider + args + cwd, but with the new
   `CLAUDE_CONFIG_DIR`.
4. The session id stays the same — same tab, same Inspector
   linked note, same memory scope key.

The terminal contents reset (new process, new TUI). The session
retains every shared piece (memory, notes, history, channels);
only the OAuth identity changes.

## Limitations

- Only Claude has this binding. Codex / Gemini / Shell use env
  vars set per-process at spawn time. If you need multi-account
  Codex, set `OPENAI_API_KEY` differently in the spawn dialog's
  *Args*, or wrap the binary in a custom provider manifest with
  its own per-account env logic.
- Account names cannot contain `/`, `..`, or non-printable
  characters. The directory lookup is sandboxed strictly to
  `~/.claude-accounts/<name>`.
- Deleting an account directory while a bound session is running
  breaks the next API call from that session. opendray doesn't
  guard against this — be careful when cleaning up host
  filesystem state.
- The account row's `enabled` flag is independent of token
  validity. A row with `enabled=true` but a missing/expired
  credentials file will fail at spawn time, not at toggle time.

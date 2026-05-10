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

The gateway creates one row in its `claude_accounts` table per
account, and reads the OAuth credentials from a directory on the
host filesystem at session-spawn time. So setup is a two-piece
move: get a `<name>` directory populated with a working OAuth
session, and get a row in the table that points at it.

The canonical flow does both pieces in order with one shell
command.

### Method 1 — `claude login` under a per-account config dir (recommended)

Run this on the gateway host (or any machine whose `~` is the
gateway's `~`):

```bash
# Pick a short slug for the account.
NAME=work

# Create the dir and run the official Claude OAuth flow under it.
# Claude Code writes ~/.claude/.credentials.json relative to
# $CLAUDE_CONFIG_DIR, so the file lands in the right place.
mkdir -p "$HOME/.claude-accounts/$NAME"
CLAUDE_CONFIG_DIR="$HOME/.claude-accounts/$NAME" claude login
# … walk through the browser OAuth …
```

opendray watches `~/.claude-accounts/` and registers a new
`claude_accounts` row for `$NAME` automatically the next time the
panel refreshes. The token file Claude Code wrote is a
self-managing credentials blob — Claude Code's own refresh logic
handles expiry, so the account stays usable indefinitely.

| What you get            | Why this method                                          |
|-------------------------|----------------------------------------------------------|
| Long-lived credentials  | Claude Code refreshes the OAuth token internally         |
| Standard format         | The same `.credentials.json` Anthropic's tooling expects |
| No UI typing of secrets | The browser flow handles secret material                 |
| Works for any deployment shape | As long as you can SSH (or `docker exec`) into the gateway |

Repeat for each account:

```bash
for n in personal work labs ; do
  mkdir -p "$HOME/.claude-accounts/$n"
  CLAUDE_CONFIG_DIR="$HOME/.claude-accounts/$n" claude login
done
```

### Method 2 — Reserve the slot, populate later

If you want to register the account in opendray *before* you have
the OAuth credentials in hand (for example you're staging a config
change ahead of a hand-off), use the **Add account** form on the
Providers page. Enter a name and optional display name; leave it
in the "no token yet" state. The row is created with its
`config_dir` and `token_path` set to the standard locations so a
later `claude login` (Method 1) will populate it.

The mobile app does not expose a create form because the OAuth
flow itself can't reasonably run on the phone (Claude Code's
browser flow exits to a localhost URL that the gateway needs to
hear). Use the web panel from a desktop, or shell into the
gateway.

### Method 3 — Import existing credentials directories

Useful when you've already done Method 1 a few times — possibly
before you connected this gateway — and want opendray to pick up
all the existing `~/.claude-accounts/*` dirs in one click instead
of waiting for the filesystem watcher.

Click **Import local** in the web panel. The gateway scans the
configured accounts root on its own host filesystem and registers
every directory that has a working credentials file but no
matching DB row.

Caveats:

- Works only when the gateway has filesystem access to a populated
  `~/.claude-accounts/` (so: bare metal, LXC with the operator's
  home mounted, or a dev container with `$HOME` bind-mounted in).
  In a stock Docker container with no volume mount, the directory
  is empty and the button does nothing useful.
- The mobile app does not show an Import button — there's nothing
  on a phone for the gateway to import from.

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

## Why isn't there a "Paste token" form?

Earlier versions of the panel exposed a **OAuth token (optional)**
field on the Add form. That field was removed because the workflow
behind it doesn't actually produce a long-lived account:

- Claude Code authenticates via the `CLAUDE_CODE_OAUTH_TOKEN` env
  var, which expects the bare access-token string only — not the
  full credentials JSON. Operators who pasted the JSON object
  directly hit silent 401s on the next API call.
- An access token without its sibling refresh token expires
  within hours. opendray does not run an OAuth refresh loop
  against Anthropic (the refresh endpoint is not part of any
  documented public API surface), so even a correctly-pasted
  access token would die quickly.
- The standard `claude login` flow already produces a
  refresh-managed credentials file — Method 1 above. Going
  through a paste form was strictly worse.

If you have a one-off short-lived access token and explicitly
want to test it, you can still write to the underlying
endpoint (`PUT /api/v1/claude-accounts/{id}/token`) via the API.
It's intentionally not a UI affordance.

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

# Claude accounts

Claude Code looks for credentials at `~/.claude/`. To run multiple
Claude accounts on the same gateway (e.g. `personal` and `work`)
without losing your mind, opendray supports per-session account
binding — each session points at `~/.claude-accounts/<name>/`
instead of the default location.

The Claude accounts panel is on the **Providers** page (web) and
inside the **Claude provider config page** (mobile, More →
Providers → Claude Code).

## When to use this

- You have a personal Claude account and a work account, both
  on the same Anthropic billing.
- You're testing two different API tier subscriptions.
- You want to run two parallel Claude sessions in the same cwd
  with different model defaults.

If you only ever use one Claude account, you can ignore this — the
default `~/.claude/` path works fine.

## What is "the token"?

Each account row holds an OAuth credential blob that Anthropic's
official `claude` CLI normally writes to
`~/.claude/.credentials.json` (or the per-account variant under
`~/.claude-accounts/<name>/.claude/.credentials.json` when you
override `CLAUDE_CONFIG_DIR`). opendray reads that blob and stamps
it into the spawned process's environment so Claude authenticates
as the right account.

The blob looks roughly like:

```json
{
  "access_token": "sk-ant-…",
  "refresh_token": "…",
  "expires_at": 1234567890,
  "scope": "user:profile user:inference",
  "subscription_type": "pro"
}
```

You don't construct it by hand. You either let `claude login` write
it, or copy an already-written file from another machine.

## Three ways to register an account

opendray exposes three flows so each deployment shape has at least
one that works. They produce the same end state — a row in the
`claude_accounts` table with a usable token — but they differ on
who has filesystem access where, and how much typing is involved.

### Method 1 — CLI login on the gateway host (auto-detect)

Best when you have shell access on the gateway and don't mind
running an interactive `claude login`.

```bash
# On the gateway machine (the box opendray runs on):
CLAUDE_CONFIG_DIR="$HOME/.claude-accounts/personal" claude login
# … walk through the OAuth flow in the browser …
```

opendray watches `~/.claude-accounts/` for new directories and
shows them in the panel automatically. No "Add account" click
needed.

| Works for           | Doesn't work for                              |
|---------------------|-----------------------------------------------|
| Bare metal gateway  | Docker container without a mounted home dir  |
| LXC where the operator can SSH in | Remote gateway you only reach over the network |
| Local development   | Mobile app                                    |

### Method 2 — Paste token in the Add-account form

Universal. Works regardless of how / where the gateway runs.

1. On any machine where you've already done `claude login` (your
   laptop is fine), open the credentials file. By default it's
   at `~/.claude/.credentials.json`. If you used the per-account
   pattern from Method 1, look under
   `~/.claude-accounts/<name>/.claude/.credentials.json`.
2. Copy the entire JSON object.
3. In opendray, click **Add account**, fill in a `name` slug
   (and optionally a display name), paste the JSON into the
   **OAuth token** field, click Add.

This is the only flow available on **mobile** because the phone
has no `~/.claude-accounts/` of its own and no shell to run
`claude login` in.

| Works for                                | Caveat                                                  |
|------------------------------------------|---------------------------------------------------------|
| Every gateway shape, including remote   | Operator needs to manually copy a file once             |
| Mobile app                               | Refresh-on-expiry happens server-side; no mobile prompt |
| Sharing a token across two gateways      | Each gateway stores its own copy; rotate together       |

### Method 3 — Import local (web only, gateway-side scan)

Best when you've already run Method 1 a few times and want
opendray to pick up *all* the directories under
`~/.claude-accounts/` in one click instead of waiting for the
filesystem watcher.

The **Import local** button (next to **Add account** in the
panel) calls `POST /api/v1/claude-accounts/import-local`. The
gateway scans `~/.claude-accounts/` on its own host filesystem,
inspects every subdirectory for a `.claude/.credentials.json`,
and registers the ones it doesn't already have a row for.

| Works for          | Doesn't work for                                                |
|--------------------|-----------------------------------------------------------------|
| Local development | Docker container without a writable / mounted `$HOME`           |
| Bare metal        | Remote gateway where you don't control the filesystem layout    |
|                    | Mobile app (deliberately omitted — the phone has nothing to import) |

If you click **Import local** and nothing happens, the gateway's
home directory probably looks empty from inside the container.
Method 2 is your fallback.

## Which one should I use?

```
Local laptop / dev container?              → Method 1 or 3
Production gateway you SSH into?           → Method 1 (or 2 if it's read-only)
Production gateway behind Cloudflare,
  no shell access?                          → Method 2
Mobile?                                     → Method 2 (only option)
You already have ~/.claude-accounts/<n>/
  populated and just want opendray to see
  them?                                     → Method 3
```

## Binding a session to an account

In the [Spawn dialog](#sessions-spawning), the **Claude account**
dropdown only appears when provider = `claude`. Pick the account;
opendray sets `CLAUDE_CONFIG_DIR` for the spawned process so
Claude reads from the right directory.

The binding is persisted on the session row (`claude_account_id`)
so a Restart of an ended session reuses the same account.

## Switching mid-session

Live account switching: Sessions page → terminal pane → in the
top-right of the terminal there's an **Account switcher**
dropdown. Picking a different account:

1. Sends SIGTERM to the running process
2. Waits for clean exit
3. Re-spawns the same provider + args + cwd, but with the new
   `CLAUDE_CONFIG_DIR`
4. The session id stays the same — same tab, same Inspector linked
   note

The terminal contents reset (new process, new TUI). Treat it as
"the same session, different credential" rather than "a fresh
start".

## Limitations

- Only Claude has this binding. Codex / Gemini use env vars set
  per-process at spawn time; if you need multi-account Codex,
  set `OPENAI_API_KEY` differently in the spawn dialog's *Args*
  (or wrap in a custom provider manifest).
- Account names cannot contain `/`, `..`, or non-printable
  characters — the directory lookup is sandboxed to
  `~/.claude-accounts/<name>` strictly.
- Deleting an account directory while a bound session is running
  breaks the next Claude API call. opendray doesn't prevent the
  directory deletion — be careful.
- Tokens expire. opendray refreshes them server-side using the
  `refresh_token`, but if both tokens go stale (long idle) the
  next API call fails with 401 and you'll need to repeat Method
  1 or 2 to seed a fresh blob.

## Why no "Sign in with Anthropic" button?

A gateway-mediated OAuth flow ("mobile opens a webview, Anthropic
hosts the login, gateway swaps the auth code for tokens") would
make adding accounts genuinely one-click on every device. We
don't ship that today because Anthropic's OAuth client-id
allowlist is currently scoped to their own first-party CLIs and
desktop apps; there is no public registration path for a
third-party gateway. If that changes, we'll add Method 4 here.

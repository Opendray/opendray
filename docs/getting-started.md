# Getting started

Zero-to-first-session walkthrough. Plan for ~15 minutes if Postgres
is already on the host; 25 minutes if you need to install one too.

This guide is intentionally end-to-end — it covers the things that
sit *around* opendray (installing the CLIs it wraps, bootstrapping
Postgres) on top of the deploy paths in the [README](../README.md#install).
If you've used opendray before and just want to redeploy, the
condensed paths in README Production deploy are a better fit.

> **Already know "is opendray for me?"**
> If not, read the [What is opendray?](../README.md#what-is-opendray)
> section in the README first. The bullets there will save you
> 15 minutes if your use case isn't a match.

---

## Step 0 — what you'll need

| Tool | Why | Note |
|---|---|---|
| At least one of: Claude Code / Codex CLI / Gemini CLI | opendray is a **wrapper**, not a model — it spawns a CLI on your host | Step 1 below |
| PostgreSQL 15 / 16 / 17 + **pgvector** extension | State, sessions, memory vectors | Step 2 below |
| `go` 1.25+ and `pnpm` 10+ — *only* if you build from source | Skip if you grab a release binary | [Releases page](https://github.com/Opendray/opendray_v2/releases) |
| A reachable network port (default `:8770`) for the web admin | UI + API + WebSockets | Bind to `127.0.0.1` unless behind a reverse proxy |

---

## Step 1 — install at least one AI CLI

opendray spawns these CLIs against your local accounts. You install
them just like you would for terminal use; opendray finds them on
`$PATH`.

### Claude Code (recommended starting point)

```sh
npm install -g @anthropic-ai/claude-code
claude login        # browser-based OAuth
```

After login, credentials sit at `~/.claude/credentials.json`.
opendray reads them automatically when you select the **claude**
provider.

### Codex CLI (OpenAI)

```sh
# Follow https://github.com/openai/codex
# The exact npm package or pip target varies by release; whatever
# you end up with should put `codex` on $PATH.
codex --version     # sanity check
```

### Gemini CLI (Google)

```sh
npm install -g @google/gemini-cli
gemini auth login
```

### Verify at least one is reachable

```sh
which claude codex gemini      # at least one line should resolve
```

> You can run opendray with just **one** CLI installed and add the
> others later. The provider list is dynamic — opendray probes the
> binary at spawn time, missing ones just show as "command not
> found" in the Sessions error panel.

---

## Step 2 — install Postgres + pgvector

opendray requires PostgreSQL **15, 16, or 17** with the
[`pgvector`](https://github.com/pgvector/pgvector) extension. Pick
the install method matching your host.

### macOS (Homebrew)

```sh
brew install postgresql@17 pgvector
brew services start postgresql@17
```

### Ubuntu / Debian

```sh
sudo apt install postgresql-17 postgresql-17-pgvector
sudo systemctl enable --now postgresql
```

### Other Linux

Use your distro's PG packages, then either install pgvector via
package or build [from source](https://github.com/pgvector/pgvector#installation).

### Bootstrap the opendray database (one-time)

In `psql` connected as a superuser:

```sql
-- Locally (Homebrew default): `psql postgres`
-- Remote: `psql -h <host> -U postgres -d postgres`

CREATE DATABASE opendray;
CREATE USER opendray_user WITH ENCRYPTED PASSWORD '<pick a strong password>';
GRANT ALL PRIVILEGES ON DATABASE opendray TO opendray_user;

\c opendray
CREATE EXTENSION IF NOT EXISTS vector;
GRANT ALL ON SCHEMA public TO opendray_user;
```

> `CREATE EXTENSION vector` needs **superuser**. After it lands,
> `opendray_user` only needs the CRUD privileges granted above —
> opendray never reconnects as superuser at runtime.

Test the credentials from the host you'll run opendray on:

```sh
PGPASSWORD='<password>' psql -h <pg-host> -U opendray_user -d opendray -c "SELECT 'ok' AS check;"
```

You should see `check: ok` and no errors.

---

## Step 3 — pick a deploy path and install opendray

**Decision question first**: are you here for the session spawn
feature (drive Claude / Codex / Gemini from the web Sessions page)?

### If YES — you need a "Full" path

| Your host | Path | README section |
|---|---|---|
| macOS as 24/7 home server | macOS LaunchDaemon | [Option D](../README.md#option-d--macos-launchd-mac-mini--studio-as-home-server) |
| Linux box / VPS / LXC | systemd | [Option B](../README.md#option-b--systemd-bare-metal--vm--lxc) |
| Just testing in foreground | `go run` from source | [Quickstart](../README.md#quickstart-5-minute-dev-path) |
| Hand-rolled supervisor (s6 / runit / launchd Agent) | Direct binary | [Option C](../README.md#option-c--direct-binary--your-own-process-supervisor) |

> Skip Docker. The image is distroless (no Node, no AI CLIs, no
> `pg_dump`), so the Sessions tab will error on every spawn click.
> See the §A callout for the architectural reason.

### If NO — you only need channels / integrations / notes / API

| Your host | Path | README section |
|---|---|---|
|

You can still receive messages on Telegram / Slack / etc., write
notes, hit the integration API, and view the web admin. You just
can't spawn local AI CLI sessions from this deployment.

All paths converge on:

```sh
# Bootstrap a config the gateway will read
cp config.example.toml config.toml
$EDITOR config.toml            # set [database].url, [admin].password

# One-shot: create the schema (idempotent on re-run)
opendray migrate -config config.toml

# Run the gateway
opendray serve -config config.toml
```

The minimum two fields in `config.toml`:

```toml
[database]
url = "postgres://opendray_user:<password>@<host>:5432/opendray?sslmode=disable"

[admin]
password = "<initial-bootstrap-password>"
```

Everything else has sensible defaults — see `config.example.toml`
inline comments for the full surface.

---

## Step 4 — first login + change admin password

Open `http://localhost:8770/admin/` (or whatever host:port opendray
is bound to via `listen` in `config.toml`).

1. Log in as `admin` + the password you put in `[admin].password`.
2. **Immediately** go to Settings → Admin → Change password.

Why immediately: after the first password change, opendray writes a
bcrypt-hashed keyfile at `$HOME/.opendray/secrets/admin.key` and the
plaintext `[admin].password` in `config.toml` becomes inert (the
keyfile takes precedence). Until you change it, your only protection
is filesystem permissions on `config.toml`.

The full credential precedence chain is in
[operator-guide §admin](operator-guide.md#admin).

---

## Step 5 — configure a Provider

Providers → click the provider you installed in Step 1 → fill in:

- **Command path** — absolute path to the CLI binary
  (`which claude` finds it; on Apple Silicon Homebrew installs land
  at `/opt/homebrew/bin/claude`).
- **Accounts dir** (Claude only, optional) — a directory of named
  Claude credential sets if you want to switch identities per
  session. Leave blank to use the default `~/.claude`.

Save. opendray runs a one-off `<cli> --version` to probe; the
provider card turns green when the binary is reachable.

---

## Step 6 — spawn your first session

Sessions → New session → pick the provider → pick a working
directory (any project on your machine) → Spawn.

A browser-side terminal opens. Type prompts as you would in a real
terminal. Close the tab and the session keeps running on the host;
come back, the scrollback is intact.

---

## Step 7 (optional) — add a Telegram channel

This is the feature that makes opendray different from
`tmux` + `ssh`. With a channel wired up, opendray pushes
notifications when a session goes idle (CLI is waiting on input),
and your reply on Telegram flows back as the next stdin write.

### One-time Telegram setup

1. Telegram → search **@BotFather** → start chat.
2. `/newbot` → BotFather walks you through name + username → issues a token like
   `123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11`.
3. Find your chat ID:
   - DM your bot once (any text).
   - Open `https://api.telegram.org/bot<token>/getUpdates` — your
     numeric `chat.id` is in the JSON response.

### In opendray

Channels → New channel → kind **Telegram**:

- **Bot token**: from BotFather
- **Default chat ID**: the `chat.id` from `getUpdates`
- **Notify on**: tick `session.idle` (or all three topics)

Save → click **Test** on the channel card. Within seconds you'll
see a test message in Telegram.

Now leave a session idle for 30 seconds (the default idle threshold
— configurable via `[session].idle_threshold`). Telegram pings you
with the last bit of CLI output. Reply, and the text flows back
into the session's stdin.

---

## What next?

- **More channels**: Slack / Discord / Feishu (飞书) / DingTalk
  (钉钉) / WeCom (企业微信) — each has its own setup in the in-app
  Tutorial at `/admin/tutorial/`.
- **API integrations**: [docs/integration-guide.md](integration-guide.md)
  — scoped API keys, reverse-proxy mount, events WebSocket.
- **Memory subsystem**: enable local-first embeddings via
  `[memory.backend] = "local"` or wire up Ollama / LM Studio — see
  in-app Tutorial → Memory section.
- **Encrypted backups**: configure `[backup]` to push DB dumps to
  S3 / R2 / B2 / SFTP / rclone — see
  [operator-guide §backup](operator-guide.md#backup).

## Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| `relation "providers" does not exist` on migrate | Pre-v2.0.0 binary (issue #162) | Pull the latest binary — fix is in v2.0.0 |
| `type "vector" does not exist` on migrate | pgvector extension not enabled in the opendray database | Run `CREATE EXTENSION vector;` as superuser in `opendray` |
| `Spawn session failed: executable file not found in $PATH` | The wrapped CLI isn't installed on the opendray host, or the Command Path in the Provider config is wrong | Step 1 above; verify with `which claude` (or whichever CLI) |
| Telegram bot doesn't respond to replies | Bot privacy mode is on by default (bot only sees commands) | BotFather → `/setprivacy` → Disable |
| `Bad gateway` through a reverse proxy | Proxy isn't forwarding WebSocket upgrade headers | See [operator-guide §Topology](operator-guide.md#topology) for nginx / Caddy snippets |
| Sessions tab is empty but Channels work | Likely the binary can spawn but no Provider configured | Step 5 |

---

## See also

- [README](../README.md) — install table, deploy paths, project status
- [README.zh.md](../README.zh.md) — Simplified Chinese version
- [docs/quickstart.md](quickstart.md) — 5-minute dev environment (more focused than this guide)
- [docs/operator-guide.md](operator-guide.md) — operator reference: topology, auth, backup, logging
- [docs/integration-guide.md](integration-guide.md) — third-party API surface
- [VERSIONING.md](../VERSIONING.md) — major-as-generation versioning
- [CHANGELOG.md](../CHANGELOG.md) — release history

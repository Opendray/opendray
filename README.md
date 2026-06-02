<p align="center">
  <a href="https://opendray.dev"><img src="docs/assets/logo.png" alt="opendray" width="180"></a>
</p>

<h1 align="center">opendray</h1>

<p align="center">
  <strong>Self-hosted gateway for Claude Code · Codex · Gemini · shell — with one shared local-first memory layer across them.</strong>
  <br/>
  <sub>Run sessions on your own infra. Drive from web, mobile, or chat. Open REST + WebSocket API for integrations.</sub>
</p>

<p align="center">
  <strong><a href="https://opendray.dev">🌐 opendray.dev</a></strong>
</p>

<p align="center">
  <a href="https://opendray.dev"><img alt="Website" src="https://img.shields.io/badge/website-opendray.dev-F43F5E"></a>
  <a href="https://github.com/Opendray/opendray/releases/latest"><img alt="Latest release" src="https://img.shields.io/github/v/release/Opendray/opendray?label=release&color=4f46e5"></a>
  <a href="LICENSE"><img alt="License Apache 2.0" src="https://img.shields.io/github/license/Opendray/opendray?color=blue"></a>
  <a href="https://github.com/Opendray/opendray/actions/workflows/ci.yml"><img alt="CI" src="https://img.shields.io/github/actions/workflow/status/Opendray/opendray/ci.yml?branch=main&label=CI"></a>
  <a href="https://github.com/Opendray/opendray/discussions"><img alt="Discussions" src="https://img.shields.io/github/discussions/Opendray/opendray?color=ec4899"></a>
  <br/>
  <img alt="Go" src="https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go&logoColor=white">
  <img alt="React" src="https://img.shields.io/badge/React-19-61DAFB?logo=react&logoColor=black">
  <img alt="Flutter" src="https://img.shields.io/badge/Flutter-mobile-02569B?logo=flutter&logoColor=white">
  <img alt="Postgres" src="https://img.shields.io/badge/PostgreSQL-15%2F16%2F17-336791?logo=postgresql&logoColor=white">
</p>

<p align="center">
  🌐 <strong>English</strong> · <a href="README.zh.md">简体中文</a> · <a href="README.fa.md">فارسی</a> · <a href="README.es.md">Español</a> · <a href="README.pt-BR.md">Português</a> · <a href="README.ja.md">日本語</a> · <a href="README.ko.md">한국어</a> · <a href="README.fr.md">Français</a> · <a href="README.de.md">Deutsch</a> · <a href="README.ru.md">Русский</a>
</p>

---

## Why opendray exists

Three frictions in day-to-day work with AI coding CLIs that opendray is built to fix.

**Sessions die when your laptop sleeps.** Running Claude Code or Codex over SSH means the agent dies the moment your machine closes the lid or drops Wi-Fi. Context, in-flight tool calls, the partial diff you were about to review. Gone. opendray runs the agent on a host that doesn't sleep (a Mac mini under your desk, a NAS, a VPS) and lets you reattach from a web admin, a Flutter mobile app, or a chat message. The session keeps executing whether or not anyone's connected.

**Hitting a rate limit shouldn't kill what you were doing.** If you have multiple Anthropic accounts (work + personal, family plan + Pro), opendray treats them as a pool. It surfaces tier, quota and active-session count per account, balances new sessions across them, and lets you swap a live session to a different account without losing the conversation. The transcript moves with you. Same model for Codex and Gemini accounts.

**Memory is a first-class layer, not an afterthought.** Most AI CLIs re-index project context from scratch every session, burning tokens on repeated retrieval. opendray ships a local-first vector store (ONNX / Ollama / LM Studio embeddings) with three-domain retrieval across user, project, and session, plus drift detection across layers. Every byte stays on your network.

---

## What is opendray?

**opendray** wraps the AI coding CLIs you already use — Claude Code, Codex, Gemini, plus any shell — and turns them into something you can drive from anywhere. Run sessions on your home server / NAS / VPS, get notified on Telegram when one goes idle, reply from your phone to feed the next prompt back in, all over a self-hosted gateway you control end to end.

- 🛰 **One backend, three surfaces** — single Go binary serving a React web admin and a Flutter mobile app, with every action also exposed over a REST + WebSocket API for third-party integrations.
- 💬 **Six bidirectional channels, no walled gardens** — Telegram, Slack, Discord, Feishu (飞书), DingTalk (钉钉), WeCom (企业微信), plus a Bridge adapter for anything custom. Replies on any channel get routed back into the right session.
- 🧠 **Local-first memory** — ONNX / Ollama / LM Studio embeddings with three-scope retrieval (user · project · session), smart ranking, and cross-layer conflict detection. No vector data leaves your network.
- 🔌 **Integration-grade API** — scoped API keys, per-call audit log, reverse-proxy mounts. Treat opendray as the gateway behind your own product or just as a personal command centre.
- 🔑 **Multi-Claude-account fleet** — drop multiple `claude login` accounts into the gateway; the panel auto-discovers them via a filesystem watcher, balances new sessions across enabled accounts, and lets you switch a live session between accounts **without losing the conversation** (transcript is migrated under the hood). Each account row shows live capacity (subscription tier, rate-limit tier, active sessions, last-used, current Anthropic email) so you can pick the right one at a glance.
- 🔒 **Self-hosted, license-clear** — Apache 2.0, one static binary, cosign-signed releases with SPDX SBOM. No telemetry, no cloud account, no subscription.

## Status

**v2.7.0** (latest) — the v2 generation continues to iterate. See
[`VERSIONING.md`](VERSIONING.md) for the major-as-generation policy
(major = product generation, not strict SemVer "breaking change") and
[`CHANGELOG.md`](CHANGELOG.md) for the full release history.

This generation ships:

- **One-line installer + uninstaller wizards** (Linux + macOS;
  Windows funnels through WSL2). Walks the operator through Postgres
  bootstrap, AI-CLI install, admin credentials, listen address,
  binary install, schema migration, and service registration.
- **Self-managing binary** — `opendray update / start / stop /
  restart / status / providers list / providers update` so operators
  don't touch `systemctl` / `launchctl` for routine ops.
- **Goreleaser release pipeline** — cross-compiled binaries
  (linux/darwin × amd64/arm64), cosign keyless signing (Sigstore),
  SPDX SBOM, atomically verified self-update.

## Install

### One-line installer

**Linux / macOS / WSL2**

```sh
curl -fsSL https://raw.githubusercontent.com/Opendray/opendray/main/scripts/install.sh | bash
```

**Windows** — sets up WSL2 first, then runs the Linux installer inside it. [details →](scripts/README.md#windows)

```powershell
irm https://raw.githubusercontent.com/Opendray/opendray/main/scripts/install-windows.ps1 | iex
```

Walks through Postgres setup, AI-CLI install, admin credentials, and service registration — landing a running gateway in ~5–10 minutes. See [**`scripts/README.md`**](scripts/README.md) for what the wizard does, the file layout it creates, options, and troubleshooting.

> **Want the manual walkthrough?** Read [**docs/getting-started.md**](docs/getting-started.md) — a 15-minute end-to-end guide that mirrors what the wizard does so you can verify each step yourself.

### npm / npx (Node ≥ 18)

Install globally and put `opendray` on `PATH`:

```sh
npm install -g opendray
```

Or run on demand without installing:

```sh
npx opendray
```

This installs **just the binary** — no wizard, no service, no Postgres. The package pulls the matching `opendray-{linux,darwin}-{x64,arm64}` platform binary via `optionalDependencies` (the esbuild / Biome pattern — no `postinstall`, no network call at install time). Good for scripted environments, ephemeral runners, or when you already run your own Postgres and process supervisor.

You still bring a database and start the gateway yourself:

```sh
# 1. PostgreSQL 15+ with pgvector — point a DSN at it, set an admin password.
export OPENDRAY_DATABASE_URL="postgres://opendray:pw@127.0.0.1:5432/opendray?sslmode=disable"
export OPENDRAY_ADMIN_PASSWORD="$(openssl rand -base64 24)"
# 2. Apply the schema, then run (foreground).
opendray migrate
opendray serve        # → http://127.0.0.1:8770/admin/
```

Full walkthrough — pgvector setup, `config.toml`, running as a systemd / launchd service, and updating — in [**docs/install-binary.md**](docs/install-binary.md).

### Uninstall (Linux / macOS)

**Default** — stops the gateway and removes the binary, but **keeps** your `config.toml`, data directory (bcrypt keyfile, sessions, notes, vault), logs, and the PostgreSQL database so a re-install resumes where you left off:

```sh
curl -fsSL https://raw.githubusercontent.com/Opendray/opendray/main/scripts/uninstall.sh | bash
```

**Full purge** — also drops the PG database + role, deletes config / data / logs, removes the service user. Includes a post-delete verification step that bails loudly if anything survived:

```sh
curl -fsSL https://raw.githubusercontent.com/Opendray/opendray/main/scripts/uninstall.sh | OPENDRAY_PURGE=1 bash
```

### Day-to-day commands

After install, the `opendray` binary handles its own lifecycle — no need to remember `systemctl` / `launchctl` incantations:

```sh
sudo opendray update --restart   # download latest release, verify SHA, atomic replace + restart
```

```sh
sudo opendray providers update   # bump installed AI CLIs (claude / codex / gemini) to npm-latest
```

```sh
opendray providers list          # see which AI CLIs are installed + their versions
```

```sh
sudo opendray start              # start | stop | restart | status — wraps systemd / launchd
```

`opendray --help` lists the full subcommand set.

### Deploy path picker

Every supported path includes session spawn, AI-CLI access, encrypted backups, and the full integration API. opendray is a host-resident gateway — it spawns AI CLIs via PTYs and shares process state (`~/.claude`, ssh-agent, project files) with them. That model is incompatible with the container isolation that production Docker would impose, so Docker is not a supported deployment path for v2.x.

| Path | Best for | Jump to |
|---|---|---|
| 📦 **Pre-built binary** | "Just run it" — Linux / macOS, any supervisor | [Releases page](https://github.com/Opendray/opendray/releases) → see [Production deploy](#production-deploy) |
| 🐧 **systemd unit** | Bare-metal / VM / LXC Linux box | [Production deploy §A](#option-a--systemd-bare-metal--vm--lxc) |
| 🍎 **macOS LaunchDaemon** | Mac mini / Mac Studio as home server | [Production deploy §C](#option-c--macos-launchd-mac-mini--studio-as-home-server) |
| 🛠 **Build from source** | Dev / contributing / custom builds | [Quickstart](#quickstart-5-minute-dev-path) below |

## Quickstart (5-minute dev path)

For a full walkthrough with prereqs and troubleshooting, see [`docs/quickstart.md`](docs/quickstart.md). The condensed dev path:

```bash
# 1. Have a Postgres 15+ running on 127.0.0.1:5432 with pgvector enabled
#    (apt install postgresql-16 postgresql-16-pgvector / brew install postgresql@16 pgvector).
#    Point [database].url at any other DSN if you'd rather use a remote PG.

# 2. Local config — already gitignored.
cp config.example.toml config.toml
$EDITOR config.toml          # set [database].url, [admin].password

# 3. Build the web bundle into the embed tree.
cd app/web && pnpm install && pnpm build && cd ../..

# 4. Apply schema.
go run ./cmd/opendray migrate -config config.toml

# 5. Run.
go run ./cmd/opendray serve -config config.toml
# → REST + WS:  http://127.0.0.1:8770/api/v1/...
# → Web admin:  http://127.0.0.1:8770/admin/
```

This runs OpenDray in the foreground — Ctrl-C kills it. For a long-running
daemon, see **Production deploy** below.

## Production deploy

Four supported deploy paths, pick whichever fits your environment.
Each one gives you auto-restart on crash, persistent state, and
separation of secrets from config.

### Option A — systemd (bare-metal / VM / LXC)

The recommended Linux deploy path. Ships a hardened unit at
[`deploy/systemd/opendray.service`](deploy/systemd/opendray.service)
with sandboxing (`ProtectSystem=strict`, `NoNewPrivileges`,
`MemoryDenyWriteExecute`, capability scrub), `migrate`-then-`serve`
boot, and a 20s graceful-stop window.

**Get a binary first.** Either grab a pre-built archive from the
[Releases page](https://github.com/Opendray/opendray/releases)
(`opendray_*_linux_<arch>.tar.gz` — unpacks to a single `opendray`
binary), or build from source via the [Quickstart](#quickstart-5-minute-dev-path)
above (`go build ./cmd/opendray`).

```bash
# 1. Install the binary you just grabbed (or built).
sudo install -m 0755 /path/to/opendray /usr/local/bin/opendray

# 2. Create the service user + state dir.
sudo useradd -r -s /usr/sbin/nologin -d /var/lib/opendray opendray
sudo install -d -o opendray -g opendray -m 0700 /var/lib/opendray

# 3. Drop config + secrets (root-owned; mode 0640).
sudo install -D -m 0640 config.example.toml /etc/opendray/config.toml
sudo $EDITOR /etc/opendray/config.toml             # set [database].url etc.
sudo install -D -m 0640 -o root -g opendray /dev/null /etc/opendray/env.d/secrets
sudo $EDITOR /etc/opendray/env.d/secrets           # OPENDRAY_ADMIN_PASSWORD=…

# 4. Install + enable the unit.
sudo cp deploy/systemd/opendray.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now opendray

# 5. Verify.
sudo systemctl status opendray
sudo journalctl -u opendray -f --no-pager
```

The unit runs `opendray migrate` as `ExecStartPre`, so the first boot
applies all migrations before `serve` ever starts. Restarts are
`on-failure` with a 5s back-off and a 5-burst limit per minute.

### Option B — Direct binary + your own process supervisor

For LXC without systemd, FreeBSD `rc.d`, OpenRC, or anything else.
Build once, run with whatever supervisor you already use:

```bash
# Cross-compile a release archive locally:
goreleaser release --clean --snapshot
ls dist/                  # opendray_*_linux_amd64.tar.gz etc.

# Or grab a published release artefact:
# https://github.com/Opendray/opendray/releases
```

Then point your supervisor (s6, runit, supervisord, runwhen) at:

```
/usr/local/bin/opendray serve -config /etc/opendray/config.toml
```

Pre-flight: run `opendray migrate -config /etc/opendray/config.toml`
once before the first `serve`, or as a pre-start hook in your
supervisor of choice.

### Option C — macOS launchd (Mac mini / Studio as home server)

For Apple Silicon Mac mini / Mac Studio running 24/7. Ships a
LaunchDaemon at
[`deploy/launchd/com.opendray.opendray.plist`](deploy/launchd/com.opendray.opendray.plist)
that starts at boot before any user login, restarts on crash with
a 5s throttle, and logs to `/usr/local/var/log/opendray/`.

```bash
# 1. Install the darwin binary + config + state dirs.
sudo install -m 0755 ./opendray /usr/local/bin/opendray
sudo install -d -m 0755 \
  /usr/local/etc/opendray \
  /usr/local/var/lib/opendray \
  /usr/local/var/log/opendray
sudo install -m 0640 config.example.toml /usr/local/etc/opendray/config.toml
sudo $EDITOR /usr/local/etc/opendray/config.toml    # set [database].url etc.

# 2. Apply migrations once.
sudo /usr/local/bin/opendray migrate \
  -config /usr/local/etc/opendray/config.toml

# 3. Install + load the LaunchDaemon.
sudo cp deploy/launchd/com.opendray.opendray.plist /Library/LaunchDaemons/
sudo chown root:wheel /Library/LaunchDaemons/com.opendray.opendray.plist
sudo chmod 0644 /Library/LaunchDaemons/com.opendray.opendray.plist
sudo launchctl bootstrap system /Library/LaunchDaemons/com.opendray.opendray.plist

# 4. Verify.
sudo launchctl print system/com.opendray.opendray
tail -f /usr/local/var/log/opendray/opendray.log
```

Restart with `sudo launchctl kickstart -k system/com.opendray.opendray`;
unload entirely with `sudo launchctl bootout system/com.opendray.opendray`.

Postgres on macOS — install via Homebrew (`brew install postgresql@17 && brew services start postgresql@17`) and point `[database].url` at
`postgres://$USER@127.0.0.1:5432/opendray`. Add `pgvector` with
`brew install pgvector` and `CREATE EXTENSION vector` inside the
opendray database.

---

For Proxmox-specific LXC notes (PTY in unprivileged containers,
networking, cgroup tweaks), see [`deploy/lxc/proxmox-pty-notes.md`](deploy/lxc/proxmox-pty-notes.md).

For reverse-proxy / TLS termination (nginx, Caddy, Traefik, Cloudflare
Tunnel), see [`docs/operator-guide.md`](docs/operator-guide.md) §Topology.

### Optional: enable encrypted DB backups + data exports

```bash
# Master passphrase (env-only — never write into config.toml).
export OPENDRAY_BACKUP_KEY="$(openssl rand -base64 32)"
export OPENDRAY_BACKUP_ENABLED=1

# pg_dump / pg_restore must match the server's major version. On
# Apple Silicon dev machines pointing at a PG17 server:
export OPENDRAY_BACKUP_PG_DUMP_PATH=/opt/homebrew/opt/postgresql@17/bin/pg_dump
export OPENDRAY_BACKUP_PG_RESTORE_PATH=/opt/homebrew/opt/postgresql@17/bin/pg_restore
```

Restart opendray; the sidebar grows a Backups page (`/backups`)
for encrypted PostgreSQL dumps + restore, and `/export` for
zip-bundle data exports + import. See [`docs/operator-guide.md`](docs/operator-guide.md) §Backup for the full lifecycle.

A single Go binary carries the whole web bundle — no Node runtime
required at runtime, no separate static-file server, no Caddy/nginx
needed. Cloudflare Tunnel terminates TLS in front of `:8770`.

## Layout

```
cmd/opendray/        binary entry point (≤100 LOC per design §14)
internal/
├── app/             composition root (wires every subsystem)
├── audit/           subscribes to bus topics, persists to audit_log
├── auth/            admin bearer tokens (M2.5)
├── backup/          encrypted DB dumps + admin export/import├── catalog/         CLI provider manifests + per-id user config (M2)
├── channel/         channel hub + telegram impl (M4)
├── config/          TOML loader with OPENDRAY_* env overrides
├── eventbus/        in-process pub/sub
├── gateway/         chi HTTP router + middleware + slog
├── integration/     external-app registry + reverse proxy + events WS (M3)
├── memory/          cross-CLI persistent memory├── session/         PTY lifecycle + ring buffer + WS stream (M1)
├── store/           pgx pool + hand-rolled migration runner (M0)
├── version/         build-time identification
└── web/             go:embed of the web bundle (W5)

app/web/             React 19 + TypeScript + Vite SPA (Phase 2 W0-W5)
app/mobile/          Flutter app (iOS + Android), feature parity with web
docs/
├── design.md        SSOT north-star
└── adr/             architecture decisions, dated
```

## Web frontend

`app/web/` builds a single SPA into `internal/web/dist/`, which the Go
binary embeds and serves at `/admin/*`. The Vite dev server at `:5173`
proxies `/api` to `:8770` for HMR-driven development.

```bash
# dev (hot reload on the React side, separate Go server for the API)
cd app/web && pnpm dev               # http://localhost:5173
go run ./cmd/opendray serve -config ../../config.toml   # other terminal

# prod (one binary delivers everything)
cd app/web && pnpm build              # writes ../../internal/web/dist
cd ../..
go build ./cmd/opendray               # bakes dist into the binary
./opendray serve -config config.toml
```

See [`app/web/README.md`](app/web/README.md) for the frontend stack
(React + Vite + Tailwind v4 + shadcn/ui + TanStack Router/Query +
Zustand + xterm.js) and per-W milestone notes.

## Documentation

- [`docs/getting-started.md`](docs/getting-started.md) — **start here** if you're new: zero to first session in 15 minutes, including installing the wrapped CLIs and bootstrapping Postgres
- [`docs/install-binary.md`](docs/install-binary.md) — install from the npm package or a release binary (bring your own Postgres) and run it as a systemd / launchd service
- [`docs/quickstart.md`](docs/quickstart.md) — 5-minute dev environment (assumes you already know the moving parts)
- [`docs/operator-guide.md`](docs/operator-guide.md) — deploy + ops reference for production-ish setups
- [`docs/integration-guide.md`](docs/integration-guide.md) — how to write an external integration in any language
- [`VERSIONING.md`](VERSIONING.md) — versioning strategy (major-as-generation)
- [`CHANGELOG.md`](CHANGELOG.md) — release history

## Tests

```bash
go test -race ./...        # backend
cd app/web && pnpm build   # web (TS strict + vite production build)
```

End-to-end smoke flows are tracked in commit messages per milestone.
A Playwright harness is a planned follow-up.

## Relationship to v1

v1 (`Opendray/opendray`) is the legacy codebase, now archived. v2 is
the current and active generation — feature-complete and the only
branch receiving development. Of the 16 v1 builtins, four migrated
into the v2 backend; the rest became client-side features, channel
adapters, or integration-API consumers.

## License

Apache 2.0 — see [`LICENSE`](LICENSE). (v1 was MIT; v2 is licensed
independently.)

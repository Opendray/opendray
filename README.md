<p align="center">
  <img src="docs/assets/logo.png" alt="opendray" width="180">
</p>

<h1 align="center">opendray</h1>

<p align="center">
  <strong>Self-hosted multi-CLI control gateway for AI coding agents.</strong>
  <br/>
  <sub>Remote-control Claude Code · Codex · Gemini · shell from web, mobile, or your favourite messaging app.</sub>
</p>

<p align="center">
  <a href="https://github.com/Opendray/opendray_v2/releases/latest"><img alt="Latest release" src="https://img.shields.io/github/v/release/Opendray/opendray_v2?label=release&color=4f46e5"></a>
  <a href="LICENSE"><img alt="License Apache 2.0" src="https://img.shields.io/github/license/Opendray/opendray_v2?color=blue"></a>
  <a href="https://github.com/Opendray/opendray_v2/actions/workflows/ci.yml"><img alt="CI" src="https://img.shields.io/github/actions/workflow/status/Opendray/opendray_v2/ci.yml?branch=main&label=CI"></a>
  <a href="https://github.com/Opendray/opendray_v2/pkgs/container/opendray"><img alt="GHCR" src="https://img.shields.io/badge/ghcr.io-opendray%2Fopendray-2496ED?logo=docker&logoColor=white"></a>
  <a href="https://github.com/Opendray/opendray_v2/discussions"><img alt="Discussions" src="https://img.shields.io/github/discussions/Opendray/opendray_v2?color=ec4899"></a>
  <br/>
  <img alt="Go" src="https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go&logoColor=white">
  <img alt="React" src="https://img.shields.io/badge/React-19-61DAFB?logo=react&logoColor=black">
  <img alt="Flutter" src="https://img.shields.io/badge/Flutter-mobile-02569B?logo=flutter&logoColor=white">
  <img alt="Postgres" src="https://img.shields.io/badge/PostgreSQL-15%2F16%2F17-336791?logo=postgresql&logoColor=white">
</p>

<p align="center">
  🌐 <strong>English</strong> · <a href="README.zh.md">简体中文</a>
</p>

---

## What is opendray?

**opendray** wraps the AI coding CLIs you already use — Claude Code, Codex, Gemini, plus any shell — and turns them into something you can drive from anywhere. Run sessions on your home server / NAS / VPS, get notified on Telegram when one goes idle, reply from your phone to feed the next prompt back in, all over a self-hosted gateway you control end to end.

- 🛰 **One backend, three surfaces** — single Go binary serving a React web admin and a Flutter mobile app, with every action also exposed over a REST + WebSocket API for third-party integrations.
- 💬 **Six bidirectional channels, no walled gardens** — Telegram, Slack, Discord, Feishu (飞书), DingTalk (钉钉), WeCom (企业微信), plus a Bridge adapter for anything custom. Replies on any channel get routed back into the right session.
- 🧠 **Local-first memory** — ONNX / Ollama / LM Studio embeddings with three-scope retrieval (user · project · session), smart ranking, and cross-layer conflict detection. No vector data leaves your network.
- 🔌 **Integration-grade API** — scoped API keys, per-call audit log, reverse-proxy mounts. Treat opendray as the gateway behind your own product or just as a personal command centre.
- 🔒 **Self-hosted, license-clear** — Apache 2.0, one static binary, distroless container, cosign-signed releases with SPDX SBOM. No telemetry, no cloud account, no subscription.

## Status

**v2.0.0** — first release of the opendray v2 generation (2026-05-17).
See [`VERSIONING.md`](VERSIONING.md) for the major-as-generation policy
(major = product generation, not strict SemVer "breaking change").

What's in this generation:

- **Backend (Go)** — sessions, channels, providers, memory, backup,
  integration API. Single static binary with the React SPA embedded
  via `go:embed`.
- **Web admin** (React 19 + Vite + Tailwind v4 + shadcn/ui + TanStack
  Router/Query + Zustand + xterm.js)
- **Mobile app** (Flutter, iOS + Android, in `app/mobile/`) — parity
  with web on session control, channel management, memory, backups,
  notes, and the integration API
- **Six bidirectional channels** — Telegram · Slack · Discord ·
  Feishu (飞书) · DingTalk (钉钉) · WeCom (企业微信) — plus
  **Bridge** for custom WebSocket-bound platforms
- **Local-first memory** — ONNX / Ollama / LM Studio embedding;
  cross-layer retrieval (user · project · session) with smart ranking
  and conflict detection; no data leaves your network
- **Automated release pipeline** — goreleaser cross-compile
  (linux/darwin × amd64/arm64), cosign keyless signing (Sigstore),
  SPDX SBOM, GHCR multi-arch image

See [`CHANGELOG.md`](CHANGELOG.md) for the v2.0.0 entry and the
rolling Unreleased section for what's landing next.

## Install

### One-line installer

**Linux / macOS / WSL2**

```sh
curl -fsSL https://raw.githubusercontent.com/Opendray/opendray_v2/main/scripts/install.sh | bash
```

**Windows** — sets up WSL2 first, then runs the Linux installer inside it. [details →](scripts/README.md#windows)

```powershell
irm https://raw.githubusercontent.com/Opendray/opendray_v2/main/scripts/install-windows.ps1 | iex
```

Walks through Postgres setup, AI-CLI install, admin credentials, and service registration — landing a running gateway in ~5–10 minutes. See [**`scripts/README.md`**](scripts/README.md) for what the wizard does, the file layout it creates, options, and troubleshooting.

**Uninstall** (Linux / macOS) — keeps DB + data by default; add `--purge` to drop everything:

```sh
curl -fsSL https://raw.githubusercontent.com/Opendray/opendray_v2/main/scripts/uninstall.sh | bash
# or, full purge (DB, role, config, data, logs):
curl -fsSL https://raw.githubusercontent.com/Opendray/opendray_v2/main/scripts/uninstall.sh | bash -s -- --purge
```

> **Want the manual walkthrough?** Read [**docs/getting-started.md**](docs/getting-started.md) — a 15-minute end-to-end guide that mirrors what the wizard does so you can verify each step yourself.

### Deploy path picker

**👉 Before picking: do you want to spawn Claude / Codex / Gemini sessions from the web admin?**
- **Yes** → choose a 🟢 **Full** path below (binary / systemd / launchd / source). **Skip Docker.**
- **No, just channels + integrations + notes + API** → 🐳 Docker Compose is fine.

Why: the Docker image is distroless (no Node, no AI CLIs, no `pg_dump`) and opendray's PTY can't reach a host binary from inside a container. See the §A callout for the full breakdown.

| Path | Best for | Features | Jump to |
|---|---|---|---|
| 📦 **Pre-built binary** | "Just run it" — Linux / macOS, any supervisor | 🟢 Full | [Releases page](https://github.com/Opendray/opendray_v2/releases) → see [Production deploy](#production-deploy) |
| 🐧 **systemd unit** | Bare-metal / VM / LXC Linux box | 🟢 Full | [Production deploy §B](#option-b--systemd-bare-metal--vm--lxc) |
| 🍎 **macOS LaunchDaemon** | Mac mini / Mac Studio as home server | 🟢 Full | [Production deploy §D](#option-d--macos-launchd-mac-mini--studio-as-home-server) |
| 🛠 **Build from source** | Dev / contributing / custom builds | 🟢 Full | [Quickstart](#quickstart-5-minute-dev-path) below |
| 🐳 **Docker Compose** | Gateway / channels / integrations / notes / API on a Docker host | 🟡 **Gateway-only** — ❌ no session spawn, ❌ no backups | [Production deploy §A](#option-a--docker-compose-gateway-only-no-session-spawn) |

## Quickstart (5-minute dev path)

For a full walkthrough with prereqs and troubleshooting, see [`docs/quickstart.md`](docs/quickstart.md). The condensed dev path:

```bash
# 1. Start a Postgres for local dev (or point [database].url at your own).
docker compose -f docker-compose.test.yml up -d   # 127.0.0.1:5432

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

### Option A — Docker Compose (gateway-only, no session spawn)

> ## ⚠️ Read this before you pick Docker
>
> This path **does not support session spawning**. If you want to
> spawn Claude / Codex / Gemini / shell sessions from the Sessions
> page, **stop and use [Option B](#option-b--systemd-bare-metal--vm--lxc)
> (systemd) or [Option D](#option-d--macos-launchd-mac-mini--studio-as-home-server)
> (macOS launchd) instead** — opendray and the AI CLIs share auth,
> project state, and tool definitions on the same host, so they
> cohabit naturally there. Trying to make Docker work for session
> spawn means either (a) maintaining a custom fat image with Node +
> every CLI baked in, or (b) accepting that the Sessions tab will
> error out for every spawn click.
>
> **What works** inside the container — channels (Telegram / Slack /
> Discord / Feishu / DingTalk / WeCom), integrations API + reverse
> proxy + events WebSocket, notes vault + git sync, memory
> subsystem, web admin, mobile-app backend.
>
> **What doesn't** — spawning Claude / Codex / Gemini / shell
> sessions (no Node, no AI CLIs, no host PATH bridging), and
> encrypted backups via `pg_dump` (image is distroless, no
> `pg_dump`).
>
> Pick Docker only when this LXC / VPS / NAS is a dedicated channel
> + integration gateway and you run AI sessions somewhere else.

For deployments that want opendray as a long-running gateway on a
home server, NAS, VPS, or LXC with Docker:

```bash
# 1. Set passwords (file is gitignored).
cp .env.example .env
$EDITOR .env                # set POSTGRES_PASSWORD, OPENDRAY_ADMIN_PASSWORD

# 2. (Optional) Drop your own config.toml at ./config.toml.
#    Compose bind-mounts it read-only. Skip for pure env-mode.

# 3. Start everything (opendray + postgres) as a long-running daemon.
docker compose up -d

# 4. Tail logs until "listening on …".
docker compose logs -f opendray
```

OpenDray is reachable at `http://127.0.0.1:8770/admin/`. Both services
auto-restart on crash or host reboot (`restart: unless-stopped`).
Database migrations apply automatically before opendray starts — a
one-shot `opendray-migrate` service runs first, and the main
`opendray` service waits for it via `service_completed_successfully`,
so a fresh install just works.

Postgres uses [`pgvector/pgvector:pg17`](https://hub.docker.com/r/pgvector/pgvector) — opendray's memory subsystem requires the
pgvector extension, and this image preinstalls and auto-enables it
on first init. Postgres data lives in the named volume
`opendray-postgres-data`; OpenDray state (admin keyfile, vault)
lives in `opendray-state`.

**Pin to a release image** in production by commenting out `build: .`
and uncommenting `image: ghcr.io/opendray/opendray:v2.0.0` in
`docker-compose.yml` — see the file header for details.

**Add Cloudflare Tunnel** for internet-facing access without opening
firewall ports — set `CLOUDFLARED_TOKEN` in `.env`, then:

```bash
docker compose --profile tunnel up -d
```

Stop / restart / fully reset:

```bash
docker compose down                # stop, keep data
docker compose restart opendray    # restart just opendray
docker compose down -v             # nuke everything including DB
```

### Option B — systemd (bare-metal / VM / LXC)

For when OpenDray is the only service on a Linux box and you don't
want Docker in the mix. Ships a hardened unit at
[`deploy/systemd/opendray.service`](deploy/systemd/opendray.service)
with sandboxing (`ProtectSystem=strict`, `NoNewPrivileges`,
`MemoryDenyWriteExecute`, capability scrub), `migrate`-then-`serve`
boot, and a 20s graceful-stop window.

**Get a binary first.** Either grab a pre-built archive from the
[Releases page](https://github.com/Opendray/opendray_v2/releases)
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

### Option C — Direct binary + your own process supervisor

For LXC without systemd, FreeBSD `rc.d`, OpenRC, or anything else.
Build once, run with whatever supervisor you already use:

```bash
# Cross-compile a release archive locally:
goreleaser release --clean --snapshot
ls dist/                  # opendray_*_linux_amd64.tar.gz etc.

# Or grab a published release artefact (after v2.0.0 ships):
# https://github.com/Opendray/opendray_v2/releases
```

Then point your supervisor (s6, runit, supervisord, runwhen) at:

```
/usr/local/bin/opendray serve -config /etc/opendray/config.toml
```

Pre-flight: run `opendray migrate -config /etc/opendray/config.toml`
once before the first `serve`, or as a pre-start hook in your
supervisor of choice.

### Option D — macOS launchd (Mac mini / Studio as home server)

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
`postgres://$USER@127.0.0.1:5432/opendray`. Or just run a Postgres
container alongside (`docker run -d --restart unless-stopped …`).

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
zip-bundle data exports + import. ADR 0012 + the in-app
Tutorial → Backups section have the full lifecycle.

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
├── backup/          encrypted DB dumps + admin export/import (ADR 0012)
├── catalog/         CLI provider manifests + per-id user config (M2)
├── channel/         channel hub + telegram impl (M4)
├── config/          TOML loader with OPENDRAY_* env overrides
├── eventbus/        in-process pub/sub
├── gateway/         chi HTTP router + middleware + slog
├── integration/     external-app registry + reverse proxy + events WS (M3)
├── memory/          cross-CLI persistent memory (ADR 0014)
├── session/         PTY lifecycle + ring buffer + WS stream (M1)
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
- [`docs/quickstart.md`](docs/quickstart.md) — 5-minute dev environment (assumes you already know the moving parts)
- [`docs/design.md`](docs/design.md) — mission, architecture, subsystems,
  API, data model, roadmap
- [`docs/adr/`](docs/adr/) — every binding architecture decision, dated
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
branch receiving development. ADR 0001 documents the greenfield
decision; ADR 0004 explains which v1 builtins migrated (only 4 of
16) and which became client-side / channel / integration work in v2.

## License

Apache 2.0 — see [`LICENSE`](LICENSE). (v1 was MIT; v2 is licensed
independently.)

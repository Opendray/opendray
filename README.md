# opendray v2

> 🌐 **Languages**: English · [简体中文](README.zh.md)

> Multiplexer + integration gateway for AI agent CLIs.
> Web + mobile remote control of Claude Code, Codex, Gemini, shell sessions.
> One shared Claude Pro subscription serves the whole personal app
> ecosystem instead of per-token API billing.

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

Pick the path that fits how you want to run it:

| Path | Best for | Features | Jump to |
|---|---|---|---|
| 📦 **Pre-built binary** | "Just run it" — Linux / macOS, any supervisor | ✨ Full | [Releases page](https://github.com/Opendray/opendray_v2/releases) → see [Production deploy](#production-deploy) |
| 🐳 **Docker Compose** | Gateway / channels / integrations / notes / API on a Docker host | ⚠️ No session spawn, no backups (see §A) | [Production deploy §A](#option-a--docker-compose-gateway-use-cases) |
| 🐧 **systemd unit** | Bare-metal / VM / LXC Linux box | ✨ Full | [Production deploy §B](#option-b--systemd-bare-metal--vm--lxc) |
| 🍎 **macOS LaunchDaemon** | Mac mini / Mac Studio as home server | ✨ Full | [Production deploy §D](#option-d--macos-launchd-mac-mini--studio-as-home-server) |
| 🛠 **Build from source** | Dev / contributing / custom builds | ✨ Full | [Quickstart](#quickstart-5-minute-dev-path) below |

> **Full vs gateway-only**: "Full" means everything including
> spawning Claude / Codex / Gemini / shell sessions from the Sessions
> page, and encrypted backups via `pg_dump`. Docker Compose ships a
> minimal distroless image that bundles only the opendray binary —
> no Node runtime, no AI CLIs, no `pg_dump` — so session spawn and
> backup require deploying opendray directly on a host with those
> tools installed (systemd / launchd / direct binary). The Docker
> path is the right choice when you want opendray as a network
> gateway for channels + integrations + notes + memory + API
> consumers, without local CLI sessions.

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

### Option A — Docker Compose (gateway use cases)

> **What works inside the container** — channels (Telegram / Slack /
> Discord / Feishu / DingTalk / WeCom), integrations API + reverse
> proxy + events WebSocket, notes vault + git sync, memory subsystem,
> web admin, mobile-app backend.
>
> **What doesn't** — spawning Claude / Codex / Gemini / shell
> sessions, and encrypted backups via `pg_dump`. The bundled image
> is distroless (no Node runtime, no AI CLIs, no `pg_dump`), and
> opendray's PTY can't reach a host binary from inside a container.
> If you need those, deploy on the host via Option B (systemd) or
> Option D (macOS launchd) instead — opendray and the AI CLIs share
> auth, project state, and tool definitions on the same host, so
> they cohabit naturally there.

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

- [`docs/quickstart.md`](docs/quickstart.md) — full quickstart with prereqs, troubleshooting, and the docker-compose dev DB
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

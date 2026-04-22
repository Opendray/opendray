# OpenDray on Docker

The **full** image bundles OpenDray plus four AI coding CLIs — Claude
Code, Codex, Gemini CLI, OpenCode — so a single `docker pull` gives you
a working multi-agent launcher. A companion `docker-compose.yml`
orchestrates the server, Postgres, and persistent volumes; a wrapper
script (`scripts/opendray-docker`) hides the compose ceremony behind
verbs like `up`, `login`, `doctor`, `update`.

## TL;DR

```bash
# 1. Clone and drop into the repo
git clone https://github.com/Opendray/opendray.git
cd opendray

# 2. Bootstrap the env file (edit DB_PASSWORD at minimum)
cp .env.docker.example .env
$EDITOR .env

# 3. Launch
./scripts/opendray-docker up

# 4. Open http://localhost:8640 and finish the web setup wizard
```

First-time agent login (per-CLI OAuth cached into a persistent volume):

```bash
./scripts/opendray-docker login claude
./scripts/opendray-docker login codex
./scripts/opendray-docker login gemini
./scripts/opendray-docker login opencode
```

## What ships in the image

`Dockerfile.full` is a three-stage build:

| Stage | Image | Purpose |
|------|-------|---------|
| 1 | `ghcr.io/cirruslabs/flutter:beta` | Compile the Flutter web app |
| 2 | `golang:1.24-bookworm` | Build the Go binary with web assets embedded |
| 3 | `node:22-bookworm-slim` | Runtime — adds the agent CLIs |

Bundled CLIs (all on PATH as `claude`, `codex`, `gemini`, `opencode`):

| Agent | Package | Pin via |
|-------|---------|--------|
| Claude Code | `@anthropic-ai/claude-code` | `--build-arg CLAUDE_CODE_VERSION=x.y.z` |
| Codex | `@openai/codex` | `--build-arg CODEX_VERSION=x.y.z` |
| Gemini CLI | `@google/gemini-cli` | `--build-arg GEMINI_CLI_VERSION=x.y.z` |
| OpenCode | `opencode-ai` | `--build-arg OPENCODE_VERSION=x.y.z` |

Runtime also includes `git`, `curl`, `openssh-client`, and `dumb-init`
for PID-1 signal forwarding. The container runs as uid `1000` (user
`opendray`) by default; override with `--build-arg OPENDRAY_UID=...` if
your host uid differs and bind-mount permissions matter.

## Command reference

```
opendray-docker up                 start the stack
opendray-docker down               stop the stack (volumes preserved)
opendray-docker restart            restart opendray only
opendray-docker destroy            stop + WIPE all volumes (prompts)

opendray-docker status | ps        list containers
opendray-docker logs [service]     tail logs (default: opendray)
opendray-docker doctor             run health + connectivity checks
opendray-docker versions           print bundled CLI versions

opendray-docker shell [service]    drop into bash inside a container
opendray-docker login <agent>      interactive OAuth for an agent CLI
opendray-docker pull               pull newer images (no recreate)
opendray-docker update             pull + recreate (one-shot upgrade)
opendray-docker backup [file]      pg_dump → gzipped SQL
opendray-docker build              rebuild image from Dockerfile.full
```

Every verb is also exposed via Make for muscle-memory parity:
`make docker-up`, `make docker-logs`, `make docker-doctor`, etc.

## Data locations

| What | Where (in container) | Backed by |
|------|----------------------|-----------|
| Postgres data | `/var/lib/postgresql/data` | named volume `pgdata` |
| OpenDray state | `/home/opendray/.opendray` | named volume `opendray-home` |
| Claude OAuth | `/home/opendray/.claude` | named volume `claude-auth` |
| Codex OAuth | `/home/opendray/.codex` | named volume `codex-auth` |
| Gemini OAuth | `/home/opendray/.gemini` | named volume `gemini-auth` |
| OpenCode config | `/home/opendray/.config/opencode` | named volume `opencode-auth` |
| User workspace | `/workspace` | **bind mount** (`OPENDRAY_WORKSPACE`) |
| Custom plugins | `/home/opendray/plugins-custom` | bind mount `./plugins-custom` (ro) |

Upgrading the image preserves every named volume — your agent logins
survive a `pull + recreate`. The workspace is a bind mount so your
project code lives on the host, not inside Docker.

## Choosing an image

The compose file reads `OPENDRAY_IMAGE` to pick the tag:

```env
# track main
OPENDRAY_IMAGE=ghcr.io/opendray/opendray:latest-full

# pin a release
OPENDRAY_IMAGE=ghcr.io/opendray/opendray:v0.6.0-full

# use a locally-built image
OPENDRAY_IMAGE=opendray/opendray:local-full
```

Build locally when you need to change CLI versions or bake in a patch:

```bash
./scripts/opendray-docker build \
  --build-arg CLAUDE_CODE_VERSION=2.1.117 \
  --build-arg GEMINI_CLI_VERSION=0.38.2
```

## Upgrading

```bash
./scripts/opendray-docker update
```

This pulls newer images and recreates containers. Named volumes are
reattached. If the release notes mention a DB migration, OpenDray runs
it on startup — `opendray-docker logs` will show the migration log.

## Troubleshooting

**`./scripts/opendray-docker up` exits without starting anything** — the
script auto-creates `.env` from the template on first run and exits so
you can set `DB_PASSWORD`. Edit `.env` then re-run.

**`opendray` container restarts in a loop** — almost always a DB or
JWT misconfig. `opendray-docker logs` will name the offending env var.

**`login <agent>` can't find the binary** — you're running the slim
image (`Dockerfile`, not `Dockerfile.full`). Set `OPENDRAY_IMAGE` to a
`*-full` tag and `opendray-docker update`.

**Agent CLIs don't see my project files** — `OPENDRAY_WORKSPACE` points
to a directory that doesn't exist on the host, or you started the
stack without it set. Fix `.env`, then `opendray-docker up`.

**Port 8640 is taken** — set `OPENDRAY_HTTP_PORT=9000` in `.env` and
`opendray-docker up` — the compose file maps the host port
dynamically.

**Want a full reset** — `opendray-docker destroy` removes all
containers and volumes (prompts for confirmation). `up` again
re-initializes from scratch.

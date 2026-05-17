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

## Quickstart

For a full walkthrough with prereqs and troubleshooting, see [`docs/quickstart.md`](docs/quickstart.md). The condensed path:

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

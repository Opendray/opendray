# opendray v2

> Multiplexer + integration gateway for AI agent CLIs.
> Web remote control of Claude Code, Codex, Gemini, shell sessions.
> One shared Claude Pro subscription serves the whole personal app
> ecosystem instead of per-token API billing.

## Status

v2 v1.0-rc — Phase 1 (backend) + Phase 2 (web frontend) feature-complete
on `main`. Mobile + Slack + deploy automation deferred per the
post-v1.0 roadmap.

## Quickstart

```bash
# 1. Postgres on 192.168.3.88 (per CLAUDE.md home-lab conventions).
#    Credentials live in Vaultwarden item: homelab-db-opendray-v2.

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

- [`docs/design.md`](docs/design.md) — mission, architecture, subsystems,
  API, data model, roadmap
- [`docs/adr/`](docs/adr/) — every binding architecture decision, dated
- `docs/api.md` — REST + WS reference (generated, lands post-v1.0)
- `docs/integration-guide.md` — how to write an integration
- `docs/operator-guide.md` — deploy + ops

## Tests

```bash
go test -race ./...        # backend
cd app/web && pnpm build   # web (TS strict + vite production build)
```

End-to-end smoke flows are tracked in commit messages per milestone.
Playwright e2e harness lands post-v1.0.

## Relationship to v1

v1 (`../opendray`) keeps running in production. v2 reaches feature
parity before the user-facing switchover. After v2 v1.0 release, v1 is
archived for one quarter, then retired. ADR 0001 documents the
greenfield decision; ADR 0004 documents which v1 builtins migrate
(only 4 of 16) and which become client-side / channel / integration
work in v2.

## License

Apache 2.0 — see [`LICENSE`](LICENSE). v2 is licensed independently of v1
(which is MIT).

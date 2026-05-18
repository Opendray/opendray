# Quickstart

Get a local opendray-v2 instance running in about five minutes.

## Prerequisites

| Tool | Version | Why |
|---|---|---|
| Go | 1.25+ | Backend + embedded web bundle |
| pnpm | 10+ | Web SPA build |
| Node.js | 22+ | pnpm runtime (build only — not needed at deploy time) |
| Docker | recent | Local Postgres for dev / tests (optional if you bring your own DB) |

Verify:

```bash
go version              # go1.25.x
pnpm --version          # 10.x
node --version          # v22.x
docker --version        # 24.x or newer
```

## 5-minute path

```bash
git clone https://github.com/Opendray/opendray_v2.git
cd opendray_v2

# 1. Start the bundled Postgres (or skip and use your own — see below).
apt install -y postgresql-16 postgresql-16-pgvector  # or brew install postgresql pgvector
systemctl is-active postgresql  # should print active

# 2. Local config — gitignored, safe to edit.
cp config.example.toml config.toml
# Defaults already point at the docker-compose Postgres on 127.0.0.1:5432.
# Only thing you must change: [admin].password. Use anything for local dev.

# 3. Build the web bundle so the Go binary embeds it.
cd app/web
pnpm install --frozen-lockfile
pnpm build
cd ../..

# 4. Apply the schema (one-shot — re-running is a no-op).
go run ./cmd/opendray migrate -config config.toml

# 5. Run the gateway.
go run ./cmd/opendray serve -config config.toml
```

You should now have:

| URL | What |
|---|---|
| `http://127.0.0.1:8770/admin/` | Web admin SPA — log in with `admin` + the password you set |
| `http://127.0.0.1:8770/api/v1/...` | REST + WebSocket API |

Stop the server with `Ctrl-C`. Stop the Postgres with `systemctl stop postgresql       # or brew services stop postgresql

## Frontend hot-reload (optional)

For iterating on the React SPA without rebuilding the Go binary:

```bash
# Terminal 1 — Vite dev server with HMR
cd app/web
pnpm dev                # http://localhost:5173

# Terminal 2 — Go gateway
go run ./cmd/opendray serve -config config.toml
```

Vite proxies `/api` calls (REST + WebSocket) to `:8770`, so the dev experience is identical to production.

## Using your own Postgres

If you already have a Postgres 15+ instance, skip step 1 and edit `config.toml` instead:

```toml
[database]
url = "postgres://your_user:your_pass@your_host:5432/your_db?sslmode=disable"
```

### Required: pgvector extension

Opendray's memory subsystem needs the [`pgvector`](https://github.com/pgvector/pgvector)
extension. A locally-installed Postgres uses the
`pgvector/pgvector:pg17` image which preinstalls and auto-enables
it, so no manual step there. For a BYO Postgres, install pgvector
once with a superuser before running `opendray migrate`:

```sh
# 1. Install the OS package (only needed once per host).
#    Ubuntu / Debian:
sudo apt install postgresql-17-pgvector

#    macOS (Homebrew):
brew install pgvector

#    Other OSes / source build: see https://github.com/pgvector/pgvector#installation

# 2. Enable the extension in the opendray database (one-off, requires superuser).
psql "postgres://postgres@your_host/opendray" -c 'CREATE EXTENSION IF NOT EXISTS vector;'
```

After that, opendray's regular CRUD-only role can run migrations
without needing further superuser access — migration `0011_memory`
just creates the `memories` table (the extension is already
present).

### Recommendations

- Create a project-scoped role with only the CRUD privileges opendray needs — never use `postgres` / superuser at runtime.
- Rotate credentials out of band; don't commit them.
- Connection pool size is configurable via `[database].max_conns` (default `16`).
- Supported Postgres versions: **15, 16, 17**. The encrypted-backup subsystem additionally requires `pg_dump` / `pg_restore` matching the server's major version (see operator-guide.md `[backup]`).

## Running the test suite

```bash
# All packages, race detector on. DB-touching tests skip cleanly when env unset.
go test -race ./...

# Run the DB-backed integration tests (memory/summarizer + githost + store + app)
# against your locally-installed Postgres:
export OPENDRAY_DEV_DB_URL='postgres://opendray:opendray@127.0.0.1:5432/opendray?sslmode=disable'
go test -race ./...

# Linting (matches CI):
golangci-lint run --config .golangci.yml ./...

# Frontend type-check + production build:
cd app/web && pnpm build
```

## Optional: encrypted DB backups + data exports

```bash
# Master passphrase — must be in env, never in config.toml.
export OPENDRAY_BACKUP_KEY="$(openssl rand -base64 32)"
export OPENDRAY_BACKUP_ENABLED=1

# pg_dump / pg_restore must match the Postgres server's major version.
# Example for an Apple Silicon dev machine pointing at PG17:
export OPENDRAY_BACKUP_PG_DUMP_PATH=/opt/homebrew/opt/postgresql@17/bin/pg_dump
export OPENDRAY_BACKUP_PG_RESTORE_PATH=/opt/homebrew/opt/postgresql@17/bin/pg_restore
```

Restart `opendray serve`; a Backups page appears in the admin sidebar (`/backups`) along with `/export` for zip-bundle data exports and imports. See `docs/operator-guide.md` §Backup for the full lifecycle.

## Building a single distributable binary

The repo ships a `goreleaser` config for cross-platform release builds:

```bash
# Snapshot release — builds linux/darwin × amd64/arm64 archives in dist/
goreleaser release --clean --snapshot

# Container image (requires Docker daemon)
docker build -t opendray:dev .
docker run --rm -p 8770:8770 \
  -e OPENDRAY_DATABASE_URL="postgres://opendray:opendray@host.docker.internal:5432/opendray?sslmode=disable" \
  -e OPENDRAY_ADMIN_PASSWORD="$(openssl rand -base64 24)" \
  opendray:dev
```

For tagged releases, `goreleaser release` produces a draft GitHub release; a maintainer reviews the notes and publishes.

## Troubleshooting

### `pnpm build` fails / `internal/web/dist` is empty

The Go binary uses `//go:embed all:dist` and refuses to start with an empty `dist/`. Run `cd app/web && pnpm build` and confirm the directory is populated. If you only need the API and want to skip the web build, run the Vite dev server (`pnpm dev`) instead.

### `go run ./cmd/opendray migrate` reports `connection refused`

Postgres is not running, or the DSN in `config.toml` is wrong. Check `systemctl is-active postgresql  # should print active

### `go test ./internal/memory/summarizer/...` says `--- SKIP`

Expected behaviour. Those tests need a real Postgres; export `OPENDRAY_DEV_DB_URL` (see *Running the test suite* above) and re-run.

### Port 5432 / 8770 already in use

```bash
lsof -i :5432    # find the process bound to 5432
lsof -i :8770
```

Kill the conflicting process or change the port:
- Postgres: edit your PG config (port in postgresql.conf) and update config.toml accordingly.
- Gateway: edit `listen = "127.0.0.1:8770"` in `config.toml` (or set `OPENDRAY_LISTEN`).

## Next steps

- [`README.md`](../README.md) — high-level project overview
- [`CONTRIBUTING.md`](../CONTRIBUTING.md) — how to send a PR
- [`SECURITY.md`](../SECURITY.md) — vulnerability disclosure

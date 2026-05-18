# Contributing to OpenDray v2

Thanks for your interest. v2 is in `v1.0-rc`; the cutover from
[Opendray/opendray](https://github.com/Opendray/opendray) (v1) happens after
this release ships.

## Prerequisites

| Tool | Version | Purpose |
|---|---|---|
| Go | 1.25+ | Backend + embedded web bundle |
| pnpm | 10+ | Web SPA build (`app/web/`) |
| Node.js | 22+ | pnpm runtime (build only — not needed at deploy) |
| PostgreSQL | 15+ | Required at runtime; v2 has no bundled mode |

## Development Setup

> For a full walkthrough including troubleshooting, see [`docs/quickstart.md`](docs/quickstart.md). The condensed path is below.


```bash
git clone https://github.com/Opendray/opendray_v2.git
cd opendray_v2

# 1. Postgres reachable; credentials in your secret manager.
cp config.example.toml config.toml
$EDITOR config.toml         # set [database].url, [admin].password

# 2. Build the web bundle so the Go binary has something to embed.
cd app/web && pnpm install && pnpm build && cd ../..

# 3. Apply the schema.
go run ./cmd/opendray migrate -config config.toml

# 4. Run.
go run ./cmd/opendray serve -config config.toml
# REST + WS:  http://127.0.0.1:8770/api/v1/...
# Web admin:  http://127.0.0.1:8770/admin/
```

For HMR-driven frontend work, run the Vite dev server alongside:

```bash
cd app/web && pnpm dev          # http://localhost:5173 (proxies /api to :8770)
go run ./cmd/opendray serve -config ../../config.toml   # in another terminal
```

## Project Structure

See [`README.md`](README.md) for the layout. Subsystem responsibilities
are documented inline in each `internal/<subsystem>/` package's
`doc.go` file and in the operator / integration guides under `docs/`.

## Tests

```bash
go test -race ./...                       # backend
cd app/web && pnpm build                  # web (TS strict + Vite prod build)
```

A subset of tests in `internal/memory/summarizer/` exercises a real
Postgres. They `t.Skip` when `OPENDRAY_DEV_DB_URL` is unset, so the
default `go test ./...` is always green even without a database.

To run them locally, point `OPENDRAY_DEV_DB_URL` at any Postgres 15+
instance with `pgvector` available (e.g. an apt / brew install, or a
remote PG you control):

```bash
export OPENDRAY_DEV_DB_URL='postgres://opendray:opendray@127.0.0.1:5432/opendray?sslmode=disable'
go test -race ./internal/memory/summarizer/...
```

Never hard-code DSNs in source — they will leak via git history.

## Pull Request Process

1. Fork or branch from `main`.
2. Make focused changes — one PR, one concern.
3. Run checks locally:
   ```bash
   go vet ./...
   go test -race ./...
   ```
4. Open a pull request against `main`. Describe the change, link any related
   ADR, and include a test plan.
5. CI must pass before merge.

### Commit messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):
`feat:`, `fix:`, `refactor:`, `docs:`, `test:`, `ci:`, `chore:`. Optional
scope in parens: `feat(memory):`, `fix(backup):`.

### Architecture changes

Anything that crosses subsystem boundaries, changes a public API, or
introduces a new external dependency needs a clear write-up in the
PR description: context, decision, consequences. Reference any
relevant operator / integration guide updates from the same PR.

## Code Style

- **Go:** Standard library style. `gofmt` is enforced. Wrap errors with
  context using `fmt.Errorf("op: %w", err)`. Return new structs; never
  mutate in place.
- **TypeScript:** TS strict mode + the rules already enforced by the Vite +
  ESLint setup in `app/web/`.

## Reporting a security vulnerability

See [`SECURITY.md`](SECURITY.md). Don't open a public issue.

## License

By contributing, you agree that your contributions are licensed under the
Apache License 2.0 ([`LICENSE`](LICENSE)).

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

See [`README.md`](README.md) for the layout. Subsystem responsibilities are
described in [`docs/design.md`](docs/design.md); every binding decision lives
in [`docs/adr/`](docs/adr/).

## Tests

```bash
go test -race ./...                       # backend
cd app/web && pnpm build                  # web (TS strict + Vite prod build)
```

Some tests reach a real Postgres at `192.168.3.88` (home-lab convention).
Set `OPENDRAY_DEV_DB_URL` to point at your own DB or let them skip.

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
introduces a new external dependency needs an ADR in `docs/adr/`. Use the
existing files as a template — short context, decision, consequences,
optional code references.

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

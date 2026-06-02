# Contributing to OpenDray v2

Thanks for your interest. v2 is in `v1.0-rc`; the cutover from
[Opendray/opendray](https://github.com/Opendray/opendray) (v1) happens after
this release ships.

## Finding something to work on

If you're looking for an easy way in:

- **Good first issues:** [github.com/Opendray/opendray/labels/good first issue](https://github.com/Opendray/opendray/labels/good%20first%20issue). Each one has a clear acceptance criteria and a pointer to where to start.
- **Help wanted:** [github.com/Opendray/opendray/labels/help wanted](https://github.com/Opendray/opendray/labels/help%20wanted). Larger pieces of work where outside contribution is especially welcome.
- **Discussion #300 (v2.8 roadmap):** [the "What should we ship in v2.8?" thread](https://github.com/Opendray/opendray/discussions/300) lists the ideas currently on the radar. Comment there before starting work on anything bigger than a contained fix, so we can align on scope.

If you'd rather just open a PR for something not in the issue tracker (a small fix, a typo, a missing log line), go ahead. For anything that crosses subsystem boundaries or introduces a new external dependency, please open an issue or discussion first.

## Prerequisites

| Tool | Version | Purpose |
|---|---|---|
| Go | 1.25+ | Backend + embedded web bundle |
| pnpm | 10+ | Web SPA build (`app/web/`) |
| Node.js | 22+ | pnpm runtime (build only, not needed at deploy) |
| PostgreSQL | 15+ | Required at runtime; v2 has no bundled mode |

## Development Setup

> For a full walkthrough including troubleshooting, see [`docs/quickstart.md`](docs/quickstart.md). The condensed path is below.


```bash
git clone https://github.com/Opendray/opendray.git
cd opendray

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

Never hard-code DSNs in source. They will leak via git history.

## Pull Request Process

1. Fork or branch from `main`.
2. Make focused changes. One PR, one concern.
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

## Translating opendray

Translation contributions land in two places.

**README translations.** Ten languages live alongside `README.md` at the repo root (`README.zh.md`, `README.fa.md`, `README.es.md`, `README.pt-BR.md`, `README.ja.md`, `README.ko.md`, `README.fr.md`, `README.de.md`, `README.ru.md`). The convention is "Finglish" style: technical product names and OSS terms stay in Latin script (Claude Code, Codex, gateway, session, host, transcript, embedding) while the connective tissue is native. Each language uses its own register:

- ES, PT-BR, FR: informal (tú / você / tu).
- DE: informal "Du".
- JA: です・ます polite-neutral.
- KO: 합니다 register.
- RU: lowercase impersonal "вы" (Habr-style), with the project-wide override of no em-dash тире.
- FA: heavy Latin-English borrowing, RTL-wrapped body.

**Style rule that applies to every language:** no em-dashes or prose en-dashes anywhere in user-facing text (the `U+2014` and `U+2013` Unicode characters used as prose punctuation). Use periods, commas, parentheses, or colons instead. Compound-noun hyphens in code identifiers (`local-first`, `--from-source`, `AI-Coding-CLIs`) are the regular hyphen-minus `U+002D` and stay.

**Admin UI translations.** The Flutter / React app reads from `app/i18n/<lang>.json`. Currently shipped: `en.json` (canonical, ~4 k keys) and `zh.json`. Adding a new language means copying `en.json` to `<lang>.json`, translating the values, and wiring it into the i18next config. See the open good-first-issue for the canonical walkthrough.

**Stale translation policy.** If your PR changes English-language docs or strings, ideally update the other-language equivalents in the same PR. If that's too much work, flag the affected translations as stale in the PR description and someone will pick them up.

## Asking questions

- **General questions, build logs, philosophy:** [Discussions → General](https://github.com/Opendray/opendray/discussions/categories/general).
- **Ideas for new features:** [Discussions → Ideas](https://github.com/Opendray/opendray/discussions/categories/ideas).
- **Stuck on setup / config / a specific error:** [Discussions → Q&A](https://github.com/Opendray/opendray/discussions/categories/q-a).
- **Bug reports:** [Issues → New issue](https://github.com/Opendray/opendray/issues/new/choose).

## Reporting a security vulnerability

See [`SECURITY.md`](SECURITY.md). Don't open a public issue.

## License

By contributing, you agree that your contributions are licensed under the
Apache License 2.0 ([`LICENSE`](LICENSE)).

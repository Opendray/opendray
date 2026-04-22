# Contributing to OpenDray

Thank you for your interest in contributing to OpenDray. This document covers
everything you need to get started.

## Prerequisites

| Tool | Version | Notes |
|------|---------|-------|
| Go | 1.25+ | Backend + embedded frontend |
| Flutter | 3.41+ | Frontend (web + mobile) |
| PostgreSQL | 15+ | Session and plugin metadata |
| GNU Make | any | Build orchestration |

## Development Setup

```bash
# Clone the repository
git clone https://github.com/opendray/opendray.git
cd opendray

# Enable repo-local git hooks (README sync check, etc.)
git config core.hooksPath .githooks

# Copy environment template and fill in DB credentials
cp .env.example .env
# Edit .env: set DB_HOST, DB_PASSWORD at minimum

# Start backend + Flutter web dev server
make dev
```

The backend listens on `127.0.0.1:8640` by default. The Flutter dev server
proxies API calls to it.

The `core.hooksPath` step wires up the pre-commit hooks under `.githooks/`.
Today that's just a README translation-sync check — if you edit `README.md`
without touching `README.zh-CN.md` (or the reverse), the commit is blocked
until you update the other file. CI runs the same check on push/PR, so
bypassing the local hook with `--no-verify` just moves the failure into
the pipeline.

## Project Structure

```
cmd/opendray/     # Entrypoint — setup, service, uninstall, plugin subcommands
gateway/          # HTTP/WebSocket API (chi router)
kernel/           # Core domain: auth, hub (session manager), store (DB), pg, config
plugin/           # Plugin runtime + manifest loader + marketplace
plugins/
  builtin/        # Built-in plugins (agents + panels, embedded in binary)
app/              # Flutter frontend (iOS, Android, Web)
```

## Adding a New Agent Plugin

1. Create a directory under `plugins/builtin/<your-agent>/`.
2. Copy an existing `manifest.json` (e.g., `plugins/builtin/terminal/manifest.json`)
   as a starting point.
3. Fill in the required fields:

```json
{
  "name": "my-agent",
  "displayName": "My Agent",
  "description": "Short description of what this agent does",
  "icon": "...",
  "version": "1.0.0",
  "type": "cli",
  "cli": {
    "command": "my-agent-binary",
    "defaultArgs": [],
    "detectCmd": "which my-agent-binary"
  },
  "capabilities": {
    "models": [
      { "id": "default", "name": "Default", "description": "Default model" }
    ],
    "supportsResume": false,
    "supportsStream": true,
    "supportsImages": false,
    "supportsMcp": false,
    "dynamicModels": false
  },
  "configSchema": []
}
```

Key fields:
- **`name`** -- Unique identifier, lowercase, no spaces.
- **`type`** -- `cli` for interactive terminal agents, `panel` for UI panels.
- **`cli.command`** -- The binary that OpenDray will spawn in a PTY.
- **`cli.detectCmd`** -- Used by the health check to verify the binary exists.
- **`configSchema`** -- Drives the settings form in the UI. See `plugin/manifest.go`
  for the `ConfigField` struct definition.

4. Restart the backend. The plugin scanner will pick it up automatically.

## Adding a New Panel Plugin

Panel plugins follow the same manifest format but use `"type": "panel"` and a
`"category"` field (`docs`, `files`, `database`, `logs`, `tasks`, `git`, or `custom`).
No `cli` block is needed. Configuration is passed via `configSchema` fields that
the gateway reads at request time.

## Pull Request Process

1. Fork the repository and create a feature branch from `main`.
2. Make your changes. Write tests where applicable.
3. Run checks locally:
   ```bash
   make vet
   make test
   cd app && flutter analyze --no-fatal-infos
   ```
4. Open a pull request against `main`.
5. Describe the change, link any related issues, and include a test plan.

## Code Style

- **Go:** Standard library style. Run `gofmt` (enforced by CI). Wrap errors
  with context: `fmt.Errorf("createSession: %w", err)`.
- **Dart/Flutter:** Follow `flutter analyze` recommendations. No warnings in CI.
- **Commits:** Use [Conventional Commits](https://www.conventionalcommits.org/):
  `feat:`, `fix:`, `refactor:`, `docs:`, `test:`, `chore:`.

## License

By contributing, you agree that your contributions will be licensed under the
same license as the project (see [LICENSE](LICENSE)).

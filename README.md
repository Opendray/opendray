# NTC — Terminal-Centric Development Cockpit

A single Flutter app (iOS / Android / Web) that pilots any number of
terminal-based AI coding agents (Claude Code, OpenCode, Codex, Gemini,
Qwen Code, …) running on your own server. Every feature beyond the
terminal engine itself is a plugin.

NTC started as a remote runner for Claude Code; the current design
generalises it into a **micro-kernel**: a PTY-based session hub at the
centre, with AI agents and data panels loaded from `plugins/` at
startup.

> **Status**: personal project, actively developed. Stable enough for
> daily use on a single host; not yet battle-tested for multi-user.

## Why

- Mobile-first control plane for long-running coding agents — launch
  a Claude / OpenCode session on the server, close the phone, come
  back an hour later and keep typing.
- One binary, one database, one plugin directory. Add a new AI CLI
  by dropping in a `manifest.json`; NTC auto-discovers it.
- Local and free models are first-class through the LLM Providers
  panel (Ollama, LM Studio, Groq, Gemini — anything OpenAI-compatible).
  Per-session routing: the same OpenCode binary can run on Qwen3 in
  one tab and Groq Llama 3.3 in another.
- Telegram bridge — leave the LAN and reply to your agent's idle
  prompt from a phone, including structured multi-select keyboards.
- MCP servers centrally managed, injected per session without
  polluting `~/.claude.json` or `~/.codex/config.toml`.

## Screenshots

The Flutter client runs on phone, tablet, and the web. Main surfaces:

- **Dashboard** — session list, "new session" sheet.
- **Session** — xterm.js inside a WebView, with a mobile-friendly
  QuickKeys bar (Ctrl / Esc / Tab / arrows / custom chords).
- **Browser** — launcher grid for every enabled panel plugin. Cards
  appear dynamically based on what's installed and enabled.
- **Settings** — server URL, Claude accounts, plugins, LLM providers,
  MCP servers, language.

## Architecture

```
┌───────────────────────────────────────────────────┐
│  Flutter app (iOS / Android / Web)                │
│  Sessions │ Browser (Docs/Files/DB/LLM/…) │ Settings │
└────────────────────┬──────────────────────────────┘
                     │ REST + WebSocket
┌────────────────────┴──────────────────────────────┐
│  Go backend — single binary (embeds web build)    │
│                                                   │
│  kernel/                                          │
│   ├ terminal/   PTY engine, ring-buffer, idle     │
│   ├ hub/        multi-session manager + injectors │
│   ├ auth/       JWT                               │
│   └ store/      PostgreSQL + migrations           │
│                                                   │
│  plugin/        manifest scan, runtime, hooks     │
│                                                   │
│  gateway/       HTTP + WebSocket routing          │
│   ├ docs/       Git-forge markdown reader         │
│   ├ files/      sandboxed file browser            │
│   ├ database/   read-only PG browsing + SQL       │
│   ├ logs/       tail-follow + regex grep          │
│   ├ tasks/      Makefile / npm / shell runner     │
│   ├ git/        per-repo status + per-session diff│
│   ├ telegram/   Bot bridge (long-poll, commands,  │
│   │             bidirectional session binding)    │
│   ├ mcp/        MCP config renderer + injector    │
│   ├ llm_providers  OpenAI-compat endpoint book    │
│   └ llm_proxy/  Anthropic↔OpenAI translator       │
│                 (reserved for future use)         │
└───────────────────────────────────────────────────┘
```

## Plugins

Manifests live at `plugins/{agents,panels}/<name>/manifest.json`
and are scanned recursively on startup.

### Agents (6)

| Plugin        | Icon | Capabilities                                                     |
|---------------|------|------------------------------------------------------------------|
| Claude Code   | 🟣   | bypass-permissions, resume, MCP, images, multi-account (OAuth)   |
| Codex CLI     | 🤖   | approval modes, MCP                                              |
| Gemini CLI    | ✨   | yolo mode, sandbox, multimodal                                   |
| Qwen Code     | 🐉   | Qwen3-Coder / DashScope / ModelScope / OpenRouter                |
| **OpenCode**  | 🤖   | provider-agnostic, resume, MCP. Routes through LLM Providers     |
|               |      | to any OpenAI-compatible endpoint (local Ollama/LM Studio, Groq, |
|               |      | Gemini free, …).                                                 |
| Terminal      | ⬛   | system login shell                                               |

### Panels (10)

| Plugin             | Icon | Category      | What it does                                                      |
|--------------------|------|---------------|-------------------------------------------------------------------|
| Obsidian Reader    | 📄   | docs          | Browse an Obsidian vault stored on Gitea/GitHub/GitLab            |
| File Browser       | 📁   | files         | Sandboxed server file browsing with syntax highlighting           |
| PostgreSQL Browser | 🐘   | database      | Read-only SELECT + schema browser + query history                 |
| Web Preview        | 🌐   | preview       | In-app browser with device-viewport simulation                    |
| Simulator Preview  | 📱   | simulator     | Live iOS Simulator / Android Emulator screenshot stream           |
| Log Viewer         | 📋   | logs          | Tail-follow + regex grep + severity highlighting                  |
| Task Runner        | 🔧   | tasks         | Discover Makefile / npm / shell scripts and run them with live stream |
| Telegram Bridge    | ✈️   | messaging     | Bot bridge for remote session control, including structured prompts |
| MCP Servers        | 🔌   | mcp           | Central MCP server registry, per-session config injection         |
| **LLM Providers**  | 🛰️   | endpoints     | Address book of OpenAI-compatible model endpoints                 |

Each plugin declares:

- **Capabilities** (`supportsResume`, `supportsStream`, `supportsImages`, `supportsMcp`, `dynamicModels`, …)
- **ConfigSchema** — typed fields (string / secret / select / number / boolean / args) that drive the Flutter form
- **CLISpec** (agents) — command, default args, install detection
- **Category** (panels) — decides which launcher card the plugin appears under

## Quick start

### Prerequisites

- Go 1.25+
- Flutter 3.41+ (for the client)
- PostgreSQL 14+ (a dedicated user with CRUD rights on its own database)
- The CLIs you want to wrap — `claude`, `opencode`, `codex`, etc. —
  on the server's `PATH`. NTC never ships model weights; it just
  drives the CLIs.

### Local dev

```bash
# 1. Copy env template
cp .env.example .env
# edit .env with real DB credentials

# 2. Create the database
psql -h <host> -U postgres -c "CREATE DATABASE ntc;"
psql -h <host> -U postgres -c "CREATE USER ntc_user WITH PASSWORD '...';"
psql -h <host> -U postgres -c "GRANT ALL PRIVILEGES ON DATABASE ntc TO ntc_user;"

# 3. Run backend + Flutter in parallel
make dev

# Backend alone:
make dev-backend     # go run ./cmd/ntc (migrations run automatically)

# Client alone:
cd app && flutter run -d chrome    # or your device
```

The backend reads `.env` via `include .env` in the Makefile. If you
prefer running without Make, `set -a; source .env; set +a; go run ./cmd/ntc`.

### Production (LXC / Linux server)

```bash
make release-linux        # cross-compile Linux amd64 binary (with web embed)
make package              # tar up binary + plugins + deploy scripts
make deploy               # scp + ssh restart (requires NTC_DEPLOY_HOST)
```

`make deploy` expects `NTC_DEPLOY_HOST=root@<server>` and an SSH key at
`~/.ssh/ntc_deploy_key`; override either via env. The `.gitea/workflows/deploy.yml`
workflow performs the equivalent on push to `deploy-production`.

### First-run setup from the mobile app

1. **Settings → Server URL**: point at the backend (`http://<host>:8640` or your public HTTPS).
2. **Settings → Plugins**: enable the agents and panels you want.
   Each agent has an install-detection indicator (`which <cli>`); if
   the binary is missing the card shows a warning.
3. **Settings → Claude accounts** (optional, for Anthropic OAuth).
4. **Browser → LLM Providers** (optional, for OpenCode + local/free
   models): add an endpoint (provider type, base URL, optional
   `apiKeyEnv` name). "Detect models" probes `/v1/models`.
5. **Browser → MCP Servers** (optional): register stdio/sse/http MCP
   definitions; NTC injects them into Claude/Codex sessions at spawn.

## Tech stack

- **Backend**: Go 1.25+, `chi` router, `gorilla/websocket`, `creack/pty`,
  `jackc/pgx/v5`
- **Database**: PostgreSQL (pooled connections, 8 migrations)
- **Client**: Flutter 3.41 (Dart 3), `webview_flutter`, `xterm.js`
  inside WebView, `provider` for state, `dio` for HTTP, `go_router`
- **Auth**: JWT (7-day TTL, optional). Cloudflare Access service-token
  headers supported by the Flutter client.
- **Packaging**: single binary with the Flutter web build embedded
  via `go:embed`.

## Configuration

All backend configuration is via environment variables — see
[`.env.example`](.env.example) for the full list. Per-plugin
configuration lives in PostgreSQL and is edited from the mobile app;
the filesystem `manifest.json` is only read on first boot to seed the
DB and is the source of truth for the CLI shape, not the runtime
config.

## Security notes

- The PTY API is `root`-equivalent on the host — always run behind
  authentication (JWT or a reverse proxy + Access).
- PostgreSQL browser plugin is defence-in-depth read-only: regex gate,
  keyword blacklist, comment stripping, single-statement rule,
  `BeginTx(ReadOnly)`, row/time caps, password scrubbing in error
  messages.
- File / logs / tasks plugins sandbox all paths against a configurable
  allow-list and resolve symlinks before prefix-checking.
- API keys for LLM providers are never stored in the database — only
  the env-var **name** is. The gateway reads the value from the host
  environment at spawn time.

## Contributing

This is a personal project, but the plugin system is the main extension
point: a new agent or panel is a directory under `plugins/` with a
`manifest.json` — no backend code changes required for most additions.


## License

MIT — see [`LICENSE`](LICENSE).

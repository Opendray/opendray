# OpenDray

> A self-hosted terminal cockpit for piloting AI coding agents from your phone.

[![Go](https://github.com/opendray/opendray/actions/workflows/ci.yml/badge.svg)](https://github.com/opendray/opendray/actions)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![GitHub Release](https://img.shields.io/github/v/release/opendray/opendray)](https://github.com/opendray/opendray/releases)

Start a Claude Code session on your server from the train. Close the app. Come back an hour later. The session kept running. Review the diff.

---

## Why OpenDray?

- **Mobile-first** — Start a coding agent from your phone, close the app, come back later. The session keeps running on your server.
- **Multi-agent** — Claude Code, Codex, Gemini, OpenCode, Qwen side-by-side with shared MCP tools.
- **Plugin architecture** — Add any new AI CLI by dropping a `manifest.json`. No code changes, no rebuilds.
- **Telegram bridge** — Full bidirectional session control. No app required.
- **Self-hosted** — Single binary + PostgreSQL. Your code, your servers, your data.
- **LLM routing** — Same agent, different models. Route OpenCode to Qwen3 in one tab and Groq Llama in another.

## Quick Start

```bash
# Prerequisites: Go 1.25+, Flutter 3.41+, PostgreSQL 14+

# 1. Clone and configure
git clone https://github.com/opendray/opendray.git
cd opendray
cp .env.example .env
# Edit .env with your PostgreSQL credentials

# 2. Create the database
psql -U postgres -c "CREATE DATABASE opendray;"
psql -U postgres -c "CREATE USER opendray WITH PASSWORD 'your-password';"
psql -U postgres -c "GRANT ALL PRIVILEGES ON DATABASE opendray TO opendray;"

# 3. Run (backend + Flutter web in parallel)
make dev
```

Or build and run the production binary:

```bash
make release-linux    # Cross-compile with embedded web frontend
./bin/opendray-linux-amd64
```

## Architecture

```
┌────��──────────────────────────────────────────────┐
│  Flutter app (iOS / Android / Web)                │
│  Sessions │ Browser (Docs/Files/DB/LLM/…) │ Settings │
└────────────────────┬──────���───────────────────────┘
                     │ REST + WebSocket
┌───────��────────────┴───────────��──────────────────┐
│  Go backend — single binary (embeds web build)    │
│                                                   │
│  kernel/                                          │
│   ├ terminal/   PTY engine, ring-buffer, idle     │
│   ├ hub/        multi-session manager + injectors │
│   ├ auth/       JWT (required on public bind)     │
│   └ store/      PostgreSQL + migrations           │
│                                                   │
│  plugin/        manifest scan, runtime, hooks     │
│                                                   │
│  gateway/       HTTP + WebSocket routing          │
│   ├ telegram/   Bot bridge (bidirectional)        ��
│   ├ mcp/        MCP config renderer + injector    │
│   ├ llm_proxy/  Provider translation layer        │
│   ├ files/      Sandboxed file browser            │
│   ├ database/   Read-only PostgreSQL browser      │
│   ├ git/        Per-repo status + per-session diff│
│   ├ logs/       Tail-follow + regex grep          │
│   ├ tasks/      Makefile / npm / shell runner     │
│   └ docs/       Git-forge markdown reader         │
└───────────────────────────────────���───────────────┘
```

## Supported Agents

| Agent | Status | Type | Notes |
|---|---|---|---|
| Claude Code | Stable | Core | Full session management, resume, MCP, images, multi-account OAuth |
| Codex CLI | Stable | Core | Approval modes, MCP |
| Gemini CLI | Stable | Core | Yolo mode, sandbox, multimodal |
| OpenCode | Stable | Core | Provider-agnostic — routes through LLM Providers to any OpenAI-compatible endpoint |
| Qwen Code | Beta | Plugin | Qwen3-Coder / DashScope / ModelScope / OpenRouter |
| Terminal | Stable | Core | System login shell |

Add a new agent by creating `plugins/agents/<name>/manifest.json`:

```json
{
  "name": "my-agent",
  "kind": "agent",
  "icon": "🤖",
  "cliSpec": {
    "command": "my-agent-cli",
    "defaultArgs": ["--no-color"],
    "installDetect": "which my-agent-cli"
  },
  "capabilities": {
    "supportsResume": false,
    "supportsStream": true
  }
}
```

Restart OpenDray. The new agent appears in the session launcher.

## Panel Plugins

| Plugin | Category | What it does |
|---|---|---|
| Obsidian Reader | docs | Browse an Obsidian vault stored on Gitea/GitHub/GitLab |
| File Browser | files | Sandboxed server file browsing with syntax highlighting |
| PostgreSQL Browser | database | Read-only SELECT, schema browser, query history |
| Log Viewer | logs | Tail-follow + regex grep + severity highlighting |
| Task Runner | tasks | Discover and run Makefile / npm / shell scripts |
| Git | git | Per-repo status, per-session diff |
| Telegram Bridge | messaging | Bot bridge for remote session control |
| MCP Servers | mcp | Central MCP server registry, per-session injection |
| LLM Providers | endpoints | Address book of OpenAI-compatible model endpoints |
| Web Preview | preview | In-app browser with device-viewport simulation |

## Telegram Bridge

Control your agent sessions from Telegram without opening the app:

| Command | Description |
|---|---|
| `/status` | List running sessions |
| `/tail <id> [n]` | Last N lines of a session |
| `/link <id>` | Bind chat to session (two-way) |
| `/unlink` | Remove binding |
| `/stop <id>` | Stop a running session |
| `/send <id> <text>` | One-shot send without binding |

Linked chats relay messages bidirectionally — type in Telegram, it goes to the agent. Agent output streams back. Reply to any idle notification to route directly to that session.

## LLM Provider Routing

The LLM Providers panel lets you register any OpenAI-compatible endpoint:

- **Local models**: Ollama, LM Studio, llama.cpp
- **Cloud APIs**: Groq, Gemini (free tier), OpenRouter, Together AI
- **Custom endpoints**: Any server implementing `/v1/chat/completions`

When creating a session with OpenCode, pick which provider and model to use. The same CLI binary, different brain.

## MCP Server Management

Register MCP servers centrally in OpenDray. When an agent session starts, OpenDray injects the MCP configuration automatically — no need to edit `~/.claude.json` or `~/.codex/config.toml` on the host.

Supports stdio, SSE, and HTTP transports.

## Security

- The PTY API is root-equivalent on the host. Always run behind authentication.
- JWT authentication is **required** when binding to non-loopback addresses. The server refuses to start without it.
- Default bind address is `127.0.0.1:8640` (loopback only).
- Rate limiting on session creation and mutation endpoints.
- PostgreSQL browser plugin is defense-in-depth read-only.
- API keys for LLM providers are never stored in the database.

See [SECURITY.md](SECURITY.md) for the full threat model and deployment checklist.

## Configuration

All backend configuration is via environment variables. See [`.env.example`](.env.example) for the complete reference.

Key variables:

| Variable | Default | Description |
|---|---|---|
| `LISTEN_ADDR` | `127.0.0.1:8640` | HTTP listen address |
| `DB_HOST` | (required) | PostgreSQL host |
| `DB_PASSWORD` | (required) | PostgreSQL password |
| `JWT_SECRET` | (empty = dev mode) | Required for non-loopback bind |
| `PLUGIN_DIR` | `./plugins` | Plugin manifest directory |
| `OPENDRAY_TELEGRAM_BOT_TOKEN` | (empty) | Telegram bridge bot token |

## Tech Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.25+, chi router, gorilla/websocket, creack/pty, pgx/v5 |
| Frontend | Flutter 3.41+ (Dart 3), xterm.js via WebView, go_router |
| Database | PostgreSQL 14+ (8 auto-applied migrations) |
| Auth | JWT (7-day TTL), optional Cloudflare Access support |
| Packaging | Single binary with Flutter web build embedded via `go:embed` |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, plugin authoring guides, and PR process.

The fastest way to contribute is adding support for a new AI CLI — it's a `manifest.json` and a restart.

## License

MIT — see [LICENSE](LICENSE).

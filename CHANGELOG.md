# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.3](https://github.com/Opendray/opendray/compare/v0.3.2...v0.3.3) (2026-04-23)


### Bug Fixes

* **files,docs:** restore ".." up-navigation inside empty directories ([e194aee](https://github.com/Opendray/opendray/commit/e194aee0c8730f2e55dbe8a6854f17966939b2b0))
* **telegram:** stop idle-notification spam and cross-session reply routing ([fd921d2](https://github.com/Opendray/opendray/commit/fd921d2e426b912709e4d7c6395ad4141d0ba23c))

## [0.1.0] - 2026-04-17

### Added

- Go backend with PTY session management, WebSocket streaming, and PostgreSQL state
- Flutter client for iOS, Android, and Web with xterm.js terminal
- Plugin architecture: 6 agent plugins (Claude Code, Codex, Gemini, OpenCode, Qwen, Terminal)
- Plugin architecture: 10 panel plugins (Files, Database, Logs, Tasks, Git, MCP, Telegram, Docs, Preview, Simulator)
- Telegram bridge with bidirectional session control, structured prompts, and idle/exit notifications
- MCP server registry with per-session config injection
- LLM provider routing with OpenAI-compatible endpoint management
- Claude account management with multi-account OAuth support
- JWT authentication with refuse-to-start gate on non-loopback addresses
- Rate limiting on session operations
- Request body size caps on mutation endpoints
- Single binary deployment with embedded Flutter web build via go:embed
- Makefile-based build, release, and deploy pipeline
- Gitea Actions CI/CD workflow

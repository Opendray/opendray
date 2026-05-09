# Changelog

All notable changes to OpenDray v2 are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- LICENSE file (Apache 2.0) — previously declared in README only.
- SECURITY.md — threat model, default posture, deployment checklist, report channel.
- CONTRIBUTING.md — dev setup, test commands, PR + commit conventions.
- CHANGELOG.md — this file.

### Changed
- `internal/backup/cipher.go`: 6-line comment on `kdfSalt` flagging it
  as a frozen v1 protocol constant and pointing at ADR 0016. No code
  behaviour change.
- Renumbered ADR `0011-memory-subsystem.md` → `0014-memory-subsystem.md` to
  resolve the duplicate-0011 collision with `0011-channel-rich-content-and-bridge.md`.
  Updated cross-references in README, ADR 0013, and the embed-onnx stub.

## [v1.0.0] — 2026-05-09

First stable release. Tagged at commit `fe96fd8` on `main`. Web frontend
+ backend feature-complete; mobile + Slack inbound + automated release
workflow deferred to v1.x per the post-v1.0 roadmap. v1
(`Opendray/opendray`) keeps running in production through this quarter
per ADR 0001.

The feature inventory below was originally captured under
`[v1.0-rc] — 2026-05-05`; section was promoted to `[v1.0.0]` on tag.

### Added (since the greenfield start)

- **M0 — composition root:** `internal/app/`, config loader (`internal/config/`),
  pgx pool + hand-rolled migration runner (`internal/store/`), event bus
  (`internal/eventbus/`), structured logging via slog.
- **M1 — sessions:** PTY lifecycle, ring-buffer streaming, WS handler,
  resume-via-reconnect (per ADR 0003).
- **M2 — CLI catalog:** provider manifests + per-id user config
  (`internal/catalog/`).
- **M2.5 — admin auth:** bearer tokens with constant-time password compare
  and 24h TTL (`internal/auth/`).
- **M3 — integrations:** external-app registry, `/api/v1/proxy/{prefix}/*`
  reverse proxy, integration call log (`internal/integration/`, ADR 0006,
  ADR 0010).
- **M4 — channels:** channel hub + Telegram, Slack, Discord, DingTalk,
  Feishu, WeChat, WeCom (`internal/channel/`, ADR 0005, ADR 0011-channel).
- **Memory:** built-in pgvector cross-CLI memory layer
  (`internal/memory/`, ADR 0014). Three-CLI mirror keeps Claude / Codex /
  Gemini transcripts aligned. ONNX local-embedding optional via
  `-tags local_onnx`.
- **Ambient memory:** auto-capture from active sessions + auto-injection
  on session start (ADR 0013).
- **Backup + export:** AES-256-GCM encrypted PostgreSQL dumps,
  S3/WebDAV/SFTP/rclone targets, admin export/import bundles
  (`internal/backup/`, ADR 0012).
- **Web admin (W0–W5):** React 19 + Vite + Tailwind v4 + shadcn/ui +
  TanStack Router/Query + Zustand + xterm.js. Single SPA bundled into
  the Go binary via `go:embed` (ADR 0007, ADR 0008).
- **Events stream:** admin-bearer-authed `/api/v1/integrations/_events`
  WebSocket (ADR 0009).

### Deferred to post-v1.0

- Mobile (Flutter) client — replaced by responsive web in v2 phase 2.
- Slack inbound (M5+).
- Deploy automation (release toolchain — goreleaser, Dockerfile,
  systemd unit) lands in a follow-up PR.
- e2e Playwright harness.

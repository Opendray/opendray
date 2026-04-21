## Summary for Kev (≤300 words)

**Locked**
- Three-layer split: Flutter Workbench (renders contributions only) · Go Plugin Host (lifecycle + bridge gate) · Plugins in three forms (declarative / webview / host sidecar).
- Single Go binary deploy stays; no new infra.
- Manifest v1 is a strict superset of today's `plugin.Provider`; every current manifest loads unchanged via compat-mode.
- Bridge API surface (`opendray.workbench/fs/exec/http/session/storage/secret/events/commands/tasks/ui/clipboard/llm/git/telegram/logger`) frozen as the TS typings in 04-bridge-api.md.
- Capability-declaration + install-time consent + runtime gate + audit log.
- Marketplace = Git-repo registry (`opendray/marketplace`), sha256-pinned artifacts, optional ed25519 signatures, PR-based approval, poll-based revocation.
- JSON-RPC 2.0 stdio with LSP framing for sidecars.
- Host plugins disabled on iOS (App Store §2.5.2 safety).
- v1 contract freeze date: **2026-10-01**.

**Decisions (locked 2026-04-19)**
1. Bridge transport: dedicated `/api/plugins/{name}/ws` per plugin (not shared session WS). Blocks M2 → resolved.
2. Phone gestures: two-finger + edge-swipe; three-finger dropped (VoiceOver conflict). Blocks M5 → resolved.
3. iOS baked-in plugins: ship all 11 current `plugins/panels/*` pinned by hash; winnow to 5-7 post-launch via telemetry.
4. `opendray-dev` portable host: funded in M6 alongside the SDK — DX is load-bearing for ecosystem.
5. Cross-plugin command execute: v1 allow-all-with-audit-log; v2 introduces `exported: true` opt-in.
6. Manifest `extends` field: deferred to v2 only if real use case surfaces (YAGNI).

**Top 3 risks**
1. iOS review: WebView + host-plugin disablement is defensible, but a reviewer unfamiliar with the model could reject. Mitigation: pre-submission call with App Review, reviewer notes template in 10-security.md.
2. Supervisor sandbox on macOS/Windows is best-effort; a hostile host plugin could still exfiltrate within user's permission set. Mitigation: default-deny `http`/`fs`/`exec`, loud consent copy.
3. Schema lock-in too early: v1 may calcify around the current `Provider` shape. Mitigation: `v2Reserved` field + strict unknown-field-warn (not error) policy.

**M1 first-step tasks for the planner**
- Add DB migrations: `plugin_consents`, `plugin_kv`, `plugin_secret`, `plugin_audit`.
- Create `plugin/install/` (download + sha256 + extract + consent token).
- Extend `plugin/manifest.go` with v1 fields (form, publisher, engines, contributes, permissions) as optional; keep `Provider` loading path untouched.
- Scaffold `gateway/plugins_install.go` with `POST /api/plugins/install` + `/confirm`.
- Publish initial `@opendray/plugin-sdk` npm package with the manifest JSON schema and a declarative scaffold template.

## Relevant file paths

- `/home/linivek/workspace/opendray/plugin/manifest.go`
- `/home/linivek/workspace/opendray/plugin/runtime.go`
- `/home/linivek/workspace/opendray/plugin/hooks.go`
- `/home/linivek/workspace/opendray/gateway/server.go`
- `/home/linivek/workspace/opendray/gateway/api.go`
- `/home/linivek/workspace/opendray/kernel/store/db.go`
- `/home/linivek/workspace/opendray/kernel/store/queries.go`
- `/home/linivek/workspace/opendray/plugins/agents/claude/manifest.json` (reference for compat-mode)
- `/home/linivek/workspace/opendray/plugins/panels/git/manifest.json` (reference for compat-mode)
- `/home/linivek/workspace/opendray/plugins/panels/telegram/manifest.json` (Telegram bridge surface)

All 13 design docs above should be materialised by the parent orchestrator under `/home/linivek/workspace/opendray/docs/plugin-platform/` using the exact filenames called out at each section header.

# 12 — Roadmap

## Milestones

Each milestone ships independently and unlocks a specific developer / user experience. A milestone is "done" only when its acceptance criteria pass CI.

### M1 — Foundations (install + declarative)
**Unlocks:** third parties can ship declarative plugins.
**Scope:**
- `plugin/install/` package: download, sha256 verify, extract, consent.
- Manifest v1 parser (superset of current `Provider`).
- DB migrations: `plugin_consents`, `plugin_kv`, `plugin_secret`, `plugin_audit`.
- Bridge gateway skeleton + capability gate.
- `contributes.commands`, `contributes.statusBar`, `contributes.keybindings`, `contributes.menus` implemented end-to-end.
- SDK scaffold (`opendray plugin scaffold --form declarative`).
- Flutter shell renders status bar + command palette entries from registered contributions.

**Acceptance:**
- A hand-built `time-ninja` plugin installs from `marketplace://…`, survives a restart, and its command shows in the palette.
- Existing `plugins/agents/*` and `plugins/panels/*` load unchanged.
- Uninstall removes all traces.

### M2 — Webview runtime
**Unlocks:** rich UI plugins.
**Scope:**
- `gateway/plugins_assets.go` with `plugin://` scheme handler.
- WebView preload with `window.opendray` and wire protocol.
- Bridge WebSocket (`/api/plugins/{name}/bridge/ws`).
- `contributes.activityBar`, `contributes.views`, `contributes.panels`.
- `opendray.workbench.*`, `opendray.storage.*`, `opendray.events.*`.
- CSP enforcement + per-plugin WebView isolation where platform allows.

**Acceptance:**
- `kanban` example plugin runs on Android, iOS, and desktop.
- Revoking `storage` at runtime causes the next `storage.set` to fail within 200 ms.

### M3 — Host sidecar runtime
**Unlocks:** LSPs, heavy background plugins.
**Scope:**
- `plugin/host/supervisor.go` with backoff, idle-shutdown.
- JSON-RPC 2.0 stdio with LSP framing.
- `opendray.fs.*`, `opendray.exec.*`, `opendray.http.*` with full capability enforcement.
- `contributes.languageServers` + LSP proxy.
- Node and Deno runtime support.

**Acceptance:**
- `rust-analyzer-od` plugin provides completion in a Rust file.
- Killing the sidecar mid-request returns a clean `EUNAVAIL`; supervisor restarts it.

### M4 — Marketplace client + publisher
**Unlocks:** plugin ecosystem.
**Scope:**
- `plugin/market/` fetches `index.json`, resolves versions.
- Settings → Marketplace browse + install UI.
- `opendray plugin publish` CLI (fork + PR + signing).
- Revocation list polling.
- Signature verification.

**Acceptance:**
- End-to-end: publish a plugin from SDK to staging marketplace, install from a different device in < 5 minutes.
- Kill-switch entry uninstalls a malicious test plugin within 10 minutes of merge.

### M5 — Contract freeze (v1 GA)
**Unlocks:** third parties can rely on the API.
**Scope:**
- All MVP-tagged contribution points shipped.
- All `opendray.*` namespaces listed as MVP in 04-bridge-api.md implemented.
- Docs published at docs.opendray.dev.
- Example plugins repo live.
- iOS build tested through App Store review (phased rollout).

**Acceptance:**
- `opendray plugin validate` passes for every example plugin.
- No P0/P1 bugs open against bridge API or manifest schema.
- Contract freeze date: **2026-10-01** (post M5 ship). After this date, manifest schema and bridge-API signature changes require a major version.

### M6 — DX polish
**Unlocks:** plugin authors become productive in hours, not days.
**Scope:**
- Hot reload for all forms.
- Portable `opendray-dev` host for offline SDK usage.
- Bridge trace tooling.
- Localization pipeline.
- Generated doc site from manifest.

**Acceptance:**
- A new plugin author can ship their first plugin in under 30 minutes following the README (measured against 3 external devs).

### M7 — Post-v1 extensions
**Non-blocking exploration.** None of these block v1:
- `contributes.debuggers` (DAP).
- `contributes.languages` (syntax highlighting + snippets).
- `contributes.taskRunners` native pluggable runner (current `plugins/panels/task-runner` stays).
- Plugin-to-plugin command export permissions.
- Private marketplaces first-class.
- Multi-view split layout.
- Paid plugins and billing.

## Deprecation plan for current plugins

Current `plugins/agents/*` and `plugins/panels/*` are supported unchanged through v1 via compat-mode (see [07-lifecycle.md](07-lifecycle.md)).

| Period | State |
|--------|-------|
| M1 → M5 | Compat mode. New features land only on v1 manifests. Docs point to v1 for new plugins. |
| M5 → v1.5 | Deprecation banners in the SDK validator when it detects a legacy manifest. |
| v2 | Compat mode removed. Any remaining legacy manifests are auto-migrated at boot by the host. |

Builtins (the 6 agents + 11 panels) are migrated in-tree to native v1 manifests during M5, PR by PR, without breaking user config. Config rows in `plugin_kv` move unchanged because the field keys are preserved.

## Contract freeze policy

After **2026-10-01**:
- Adding new optional manifest fields: minor version bump.
- Adding new optional bridge methods / namespaces: minor version bump.
- Renaming, removing, or changing behavior of existing fields or methods: major version bump; support period of at least 12 months for the old contract.

Every breaking change must include:
- An SDK `lint` rule that flags the old usage.
- A compat shim on the host.
- A migration note in CHANGELOG.md.

## Tracking

Milestones are tracked in `Obsidiannote/Projects/OpenDray/plugin-platform/roadmap.md`. Each milestone has a Linear/issue list in the main repo.

> **Locked:** v1 contract freeze date is 2026-10-01. Moving it right slips third-party ecosystem; moving it left risks half-baked APIs. Treat as non-negotiable unless M1-M4 slip by more than one calendar month combined.

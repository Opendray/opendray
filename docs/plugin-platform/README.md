# OpenDray Plugin Platform v1

OpenDray is **mobile VS Code for vibe coding**. The Plugin Platform lets any developer ship a plugin that hot-installs into a running OpenDray instance without rebuilding the Flutter app or the Go backend.

This directory is the **v1 design contract**. Every schema, API, and behaviour below is what plugin authors may rely on. Breaking changes require a major version bump and a deprecation window.

## The three-layer picture

```
+--------------------------------------------------------+
| Workbench Shell   Flutter app, fixed UI                |  <-- layer 1
|  - renders contribution points (slots)                 |
|  - owns navigation, theming, gestures, a11y            |
+----------------------+---------------------------------+
                       |  HTTPS / WSS (localhost or LAN)
+----------------------v---------------------------------+
| Plugin Host       Go backend, fixed                    |  <-- layer 2
|  - install / lifecycle / capability gate               |
|  - bridge server (opendray.* API)                      |
|  - HookBus / events / task + session ownership         |
+----------------------+---------------------------------+
                       |  stdio JSON-RPC  /  JS bridge
+----------------------v---------------------------------+
| Plugins           downloaded bundles                   |  <-- layer 3
|  declarative  |  webview  |  host (sidecar)            |
+--------------------------------------------------------+
```

## Documents

| # | Doc | Purpose |
|---|-----|---------|
| 01 | [architecture.md](01-architecture.md) | Layer responsibilities, process model, data flow diagrams |
| 02 | [manifest.md](02-manifest.md) | Complete `manifest.json` JSON Schema v1 |
| 03 | [contribution-points.md](03-contribution-points.md) | Every UI and extension slot |
| 04 | [bridge-api.md](04-bridge-api.md) | `opendray.*` API reference |
| 05 | [capabilities.md](05-capabilities.md) | Permission taxonomy and consent model |
| 06 | [plugin-formats.md](06-plugin-formats.md) | Declarative / Webview / Host — pick one |
| 07 | [lifecycle.md](07-lifecycle.md) | Install, activate, update, uninstall |
| 08 | [workbench-slots.md](08-workbench-slots.md) | UI slot catalogue with wireframes |
| 09 | [marketplace.md](09-marketplace.md) | Registry, publishing, kill-switch |
| 10 | [security.md](10-security.md) | Threat model, iOS story, trust levels |
| 11 | [developer-experience.md](11-developer-experience.md) | SDK, scaffold, hot reload |
| 12 | [roadmap.md](12-roadmap.md) | M1..M7 milestones and contract freeze |

## Hello world — one per form

### Declarative (pure manifest)
```json
{
  "$schema": "https://opendray.dev/schemas/plugin-manifest-v1.json",
  "name": "hello-decl", "version": "0.1.0", "publisher": "acme",
  "form": "declarative",
  "engines": { "opendray": "^1.0.0" },
  "contributes": {
    "commands": [{ "id": "hello.say", "title": "Say Hello",
                   "run": { "kind": "notify", "message": "Hi from plugin!" } }],
    "statusBar": [{ "id": "hello.bar", "text": "hello", "command": "hello.say" }]
  }
}
```

### Webview (manifest + static `ui/` bundle)
```json
{
  "name": "hello-webview", "version": "0.1.0", "publisher": "acme",
  "form": "webview",
  "engines": { "opendray": "^1.0.0" },
  "ui": { "entry": "ui/index.html" },
  "contributes": {
    "views": [{ "id": "hello.view", "title": "Hello",
                "container": "activityBar", "icon": "icons/hello.svg" }]
  },
  "permissions": { "storage": true }
}
```
`ui/index.html` (inside the bundle) calls `window.opendray.workbench.showMessage('hi')`.

### Host (manifest + sidecar)
```json
{
  "name": "hello-lsp", "version": "0.1.0", "publisher": "acme",
  "form": "host",
  "engines": { "opendray": "^1.0.0" },
  "host": {
    "entry": "bin/hello-lsp",
    "platforms": { "linux-x64": "bin/linux-x64/hello-lsp",
                   "darwin-arm64": "bin/darwin-arm64/hello-lsp" },
    "protocol": "jsonrpc-stdio"
  },
  "activation": ["onLanguage:hello"],
  "contributes": {
    "languageServers": [{ "id": "hello.lsp", "languages": ["hello"] }]
  },
  "permissions": { "exec": false, "fs": { "read": ["${workspace}/**"] } }
}
```

## Decision legend

- `> **Locked:**` — frozen for v1. Changing it is a breaking change requiring v2.
- `> **Open:**` — unresolved, Kev must pick. Parent orchestrator should track these.
- `post-v1` — explicitly deferred. Do not schematise yet.

## Compatibility promise

Every manifest that loads today (the six agents and eleven panels under `plugins/agents/*` and `plugins/panels/*`) continues to load unchanged. v1 is a **strict superset** of the current `plugin.Provider` shape. See [lifecycle.md](07-lifecycle.md) §Compat for the upgrade path.

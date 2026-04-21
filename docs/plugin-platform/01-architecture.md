# 01 — Architecture

## 1. Layer responsibilities

### Layer 1 — Workbench Shell (Flutter app, fixed)
**Owns:** navigation shell, activity bar, side view container, bottom panel container, editor/terminal area, status bar, notifications, command palette, theming, phone vs. tablet layout adaptation, gestures, keyboard shortcuts, accessibility.

**Never owns:** plugin business logic. The shell is a deterministic renderer of contribution points declared in manifests and of view payloads returned by plugins.

**Go package reference:** N/A (Flutter). Lives under `app/lib/workbench/` (new module — see [roadmap.md](12-roadmap.md) M2).

### Layer 2 — Plugin Host (Go backend, fixed)
**Owns:**
- Plugin lifecycle — download, verify, install, enable, activate, deactivate, uninstall.
- Manifest parsing and capability enforcement.
- Bridge server — every `opendray.*` call is routed through a single gateway that checks capability, rate-limits, audits.
- Sidecar supervision for Host-form plugins (fork/exec, stdio JSON-RPC, restart with backoff).
- Event fan-out (`HookBus` today).
- Session, task, git, file, log, telegram, MCP subsystems (existing `gateway/*` packages).

**Go packages:**
- `plugin/` — runtime, manifest, hooks (existing) plus new `plugin/install`, `plugin/bridge`, `plugin/host` sub-packages (M1).
- `gateway/` — HTTP routes for install / bridge.
- `kernel/store/` — DB-backed plugin state.

### Layer 3 — Plugins (downloaded bundles)
Three forms, exactly one per plugin:
1. **Declarative** — manifest only. No custom UI, no custom process. Pure data: commands, statusbar items, themes, keybindings, snippets, task templates.
2. **Webview** — manifest + `ui/` folder containing a static web bundle (HTML/CSS/JS/WASM). Rendered in a WebView slot. Talks to host via `window.opendray.*`.
3. **Host** — manifest + sidecar executable (or a `runtime: "node" | "deno"` script). Supervised by the Plugin Host over stdio JSON-RPC 2.0. For LSP, DAP, heavy background work, language tooling.

A plugin **must not** mix forms in v1. A multi-surface feature (e.g. a language pack with both UI and LSP) ships as two plugins with a shared namespace.

## 2. Process model

```
+-------------------+            localhost HTTPS/WSS
| Flutter Workbench |<----------------------------+
+-------------------+                             |
                                                  |
                         +------------------------v----------------+
                         | opendray single Go binary (PID 1 of app)|
                         |                                         |
                         |  +-----------+  +---------------------+ |
                         |  | Gateway   |  | Plugin Host         | |
                         |  | (chi)     |  |  - Runtime          | |
                         |  +-----^-----+  |  - Bridge gateway   | |
                         |        |        |  - Capability gate  | |
                         |        +------->+  - HookBus          | |
                         |                 +----------^----------+ |
                         |                            | stdio      |
                         |     +----------+   +-------+---------+  |
                         |     | Webview  |   | Host sidecar(s) |  |
                         |     | (inline  |   | (one exec each) |  |
                         |     |  assets) |   +-----------------+  |
                         |     +----------+                        |
                         +-----------------------------------------+
```

> **Locked:** Single Go binary. No external services required (no Redis, no Kafka). Sidecars are children of the opendray process, supervised with exponential backoff. This protects the existing LXC/Docker/single-binary deploy story.

## 3. Communication channels

| Link | Protocol | Who speaks it | Framing |
|------|----------|---------------|---------|
| Workbench ↔ Host | HTTPS + WSS | Flutter → Go | REST JSON + WebSocket text frames |
| Webview ↔ Host | `postMessage` over WebView JS bridge | JS `window.opendray` → Go | JSON envelope, see [bridge-api.md §Wire](04-bridge-api.md) |
| Sidecar ↔ Host | stdio | Go ↔ sidecar | LSP-style JSON-RPC 2.0 (Content-Length framed) |
| Host ↔ Host subsystem | in-process function calls | Go | native |

> **Locked:** JSON-RPC 2.0 with LSP-style Content-Length framing for sidecars. This matches existing LSP ecosystem tooling so an LSP server can be wrapped as an OpenDray Host plugin with zero rewrite.

## 4. Request flow — plugin install

```
Workbench                 Gateway            Plugin Host          Filesystem       Marketplace
   |   POST /api/plugins/install  |                |                  |                |
   |  { src: "marketplace://acme/hello@0.1.0" }    |                  |                |
   |----------------------------->|                |                  |                |
   |                              | Install(src)   |                  |                |
   |                              |--------------->| resolve(src)     |                |
   |                              |                |----------------->|  GET index.json|
   |                              |                |                  |--------------->|
   |                              |                |                  |<-- meta --------|
   |                              |                |<-- artifact URL+sha256 ------------|
   |                              |                | download()       |                |
   |                              |                |--------------------------- GET zip |
   |                              |                |<------------------------- bytes    |
   |                              |                | verifySha256()   |                |
   |                              |                | verifySignature(optional)         |
   |                              |                | extract -> plugins/.installed/<name>/<ver>/
   |                              |                | parseManifest()  |                |
   |                              |                | capabilityDiff() |                |
   |   202 { consentRequired: [...perms] }         |                  |                |
   |<-----------------------------|                |                  |                |
   |   user taps "Install"        |                |                  |                |
   |   POST /api/plugins/install/confirm {token}   |                  |                |
   |----------------------------->|                |                  |                |
   |                              | confirm()      |                  |                |
   |                              |--------------->| persistConsent() |                |
   |                              |                | Runtime.Register()                |
   |                              |                | activate(if onStartup)            |
   |   200 { installed, enabled } |                |                  |                |
   |<-----------------------------|                |                  |                |
```

## 5. Request flow — webview view render

```
User taps activity-bar icon "Hello"
  -> Workbench looks up contributes.views where id=hello.view
  -> Workbench opens a WebView, src = plugin://hello-webview/ui/index.html
  -> WebView loads; content is served from embedded handler that streams bytes from plugins/.installed/hello-webview/0.1.0/ui/
  -> ui/index.html loads ui/main.js which calls window.opendray.workbench.ready()
  -> ui/main.js calls window.opendray.storage.get('hits', 0) -> bridge call -> Go -> return 42
  -> user interacts; any state is persisted back via opendray.storage.set(...)
```

## 6. Request flow — bridge call

```
webview code:   await opendray.fs.readFile('/etc/hosts')
  -> JS SDK posts {id:42, ns:'fs', method:'readFile', args:['/etc/hosts']}
  -> Workbench WebView bridge forwards to Go over the session WebSocket
  -> Gateway /api/plugins/<name>/bridge handler
  -> Plugin Host looks up capability required ('fs.read:/etc/hosts')
  -> Deny: path not in allowed roots -> error {id:42, error:{code:'EPERM',...}}
  -> Allow: execute + audit log + return {id:42, result: "<file bytes base64>"}
```

## 7. Data flow — event dispatch (existing HookBus evolved)

```
session pty output --> Hub.DispatchOutput
                          |
                          v
                     HookBus.Dispatch(HookEvent)
                          |
          +---------------+----------------+
          |                                |
          v                                v
   LocalListener (Go)                HTTP subscriber
   - Telegram bridge                 - Host sidecar         <- wrapped as
   - Bridge gateway emits              via POST to its         opendray.events.subscribe
     opendray.events.*                  stdio /events notify
     to all webviews
```

## 8. Where to find current code

| Concern | Today | v1 package |
|---------|-------|------------|
| Manifest struct | `plugin/manifest.go` | `plugin/manifest.go` (extended) |
| Runtime | `plugin/runtime.go` | `plugin/runtime.go` |
| Events | `plugin/hooks.go` | `plugin/hooks.go` (source of `opendray.events.*`) |
| DB | `kernel/store/queries.go` (`Plugin` struct) | same, plus `plugin_consents` table |
| Install routes | — | `gateway/plugins_install.go` (new) |
| Bridge routes | — | `gateway/plugins_bridge.go` (new) |
| Webview asset serving | — | `gateway/plugins_assets.go` (new) |
| Sidecar supervisor | — | `plugin/host/supervisor.go` (new) |

> **Locked (2026-04-19):** Dedicated `/api/plugins/{name}/ws` per plugin. Cleaner origin checks, plugin uptime decoupled from session lifetime, independent reconnect. Rejected: shared per-session WS (coupling + multiplexing overhead).

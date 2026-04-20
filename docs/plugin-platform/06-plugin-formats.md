# 06 — Plugin Formats

Every plugin picks exactly one of three forms. Pick the least powerful form that can do the job.

## Decision tree

```
Need background CPU, long-lived process, or LSP/DAP?
│
├── yes ──► host
│
└── no  ──► Need custom UI that the native `ui.*` tree can't express (canvas, rich editors, web libs)?
            │
            ├── yes ──► webview
            │
            └── no  ──► declarative
```

## A. Declarative

**Use cases:** status bar clocks, keybindings, themes, menus, command shortcuts, Telegram command wiring. Also the registration form used by every built-in OpenDray panel (git-viewer, git-forge, pg-browser, file-browser, task-runner, log-viewer, obsidian-reader, simulator-preview, web-browser, mcp, telegram) — those bundles ship only a `manifest.json`; the rendering code lives in the OpenDray gateway + Flutter app.

**Cannot do:** custom UI rendering beyond `contributes.*`, subscribe to events with custom handlers (events can only trigger declarative `run.kind`), heavy logic.

**Sandbox guarantees:**
- Runs entirely inside the Go host.
- No plugin code executes at all — everything is data.
- Impossible to spawn processes, read files, call network (unless via a declarative `run.kind: "exec"` that triggers the `exec` capability, which the manifest must declare).

**Quotas:** n/a (host memory only).

### Declarative = a registration form, not a third-party form

Important semantic boundary: `form: "declarative"` is how OpenDray's **own core panels** surface in the Hub so they share the install / Configure / consent flow with real plugins. It is **not** a form a third party can use to add a new full-featured panel, because the panel's rendering + HTTP logic has to live somewhere — and for declarative plugins, that somewhere is the OpenDray binaries, which the third party cannot modify.

Concretely:

| Aspect | Declarative built-in | Webview | Host |
|---|---|---|---|
| Bundle ships code? | ❌ manifest only | ✅ HTML/JS bundle | ✅ sidecar process |
| Publisher | `opendray-builtin` | any | any |
| Upgrade path | with OpenDray release | Hub re-install | Hub re-install |
| Can a third party replace it? | ❌ (must fork OpenDray) | ✅ | ✅ |
| Runs isolated from host? | ❌ (in-process Go + Dart) | ✅ sandboxed WebView | ✅ separate OS process |

The Hub surfaces this with a **BUILT-IN** badge on any marketplace entry where `publisher == "opendray-builtin"` and `form == "declarative"`. Plugin listings in the app use the same badge. The intent is: users see at a glance that the plugin is part of OpenDray itself, not a third-party extension they can replace or uninstall without functional loss (they can disable the UI entry but the Go/Dart code stays in the binary regardless).

**Third parties writing a new panel must use `webview` or `host`.** The declarative form has two legitimate third-party shapes only:
1. A plugin that contributes purely declarative surfaces (status bar, keybindings, command shortcuts, menus) — no panel, no custom UI.
2. A plugin that wraps a declarative `run.kind` (e.g. `exec`) behind a command contribution.

If a third party needs a data panel with custom rendering, the answer is webview. If they need a long-lived background process, the answer is host. Declarative-plus-panel-with-logic is reserved for OpenDray itself, because only OpenDray's binary can provide the logic.

## B. Webview

**Use cases:** kanban boards, dashboards, docs/preview plugins, data visualisations, rich editors.

**Files in bundle:**
```
my-plugin/
  manifest.json
  ui/
    index.html
    main.js
    style.css
    assets/
```

**Sandbox guarantees:**
- Rendered in a dedicated `WebView` (Android: `WebView`; iOS: `WKWebView`; desktop: `flutter_inappwebview` equivalent).
- Served from `plugin://<name>/` — custom scheme handled by the Flutter shell, bytes streamed from the installed bundle dir. No `file://` access.
- Default CSP (non-overridable):
  ```
  default-src 'self';
  script-src  'self' 'wasm-unsafe-eval';
  style-src   'self' 'unsafe-inline';
  img-src     'self' data: blob:;
  connect-src 'self' https://* wss://*;  // narrowed by http capability
  frame-ancestors 'none';
  ```
- The WebView is isolated per-plugin (separate cookie jar, localStorage, IndexedDB) — v1 uses a per-plugin WebView process where platform supports it.
- DOM APIs not exposed: `navigator.serviceWorker`, `navigator.geolocation`, `navigator.mediaDevices.*` unless a post-v1 `device` capability is added.
- `window.opendray` injected by preload; no other host surface.

**Quotas:**
- Bundle ≤ 20 MB.
- WebView RAM soft cap 128 MB per plugin (warning); hard cap 256 MB (WebView killed, user notified).
- No code download after install (CSP blocks it).

## C. Host (sidecar)

**Use cases:** LSP, DAP, indexers, file watchers, sync daemons, anything compute-heavy or needing native libs.

**Layout:**
```
my-plugin/
  manifest.json
  bin/
    linux-x64/<entry>
    darwin-arm64/<entry>
    windows-x64/<entry>.exe
  ui/                  # optional — host plugins MAY also contribute webview views
    ...
```
> **Locked:** A host plugin may additionally ship a webview UI, because some language plugins (e.g. Copilot-style) need both. The sidecar owns the method surface; the webview talks to the sidecar via `opendray.commands.execute('<plugin>.<method>', ...)` which the host proxies as a JSON-RPC call.

**Runtime kinds:**
- `binary` — native executable selected by `platforms.<os>-<arch>`.
- `node` — `runtime: "node"`, entry is `.js`; host invokes `node <entry>`. Plugin must declare `engines.node`.
- `deno` — `runtime: "deno"`, entry is `.ts` or `.js`; host invokes `deno run --allow-none <entry>`; capability-gated permissions are layered on top.

**Sandbox guarantees (best-effort, OS-dependent):**
- Linux: process starts under a dedicated `setuid`-less user context when available, `prlimit` caps (RLIMIT_AS 512 MB default, RLIMIT_NPROC 32), and (when available) a seccomp bpf profile denying `ptrace`, `mount`, `reboot`, `kexec_*`.
- macOS: sandbox-exec profile denying outbound network except via the host bridge when `http` capability is not granted.
- Windows: `JOB_OBJECT` with `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE` so the sidecar dies with the host.
- iOS: Host plugins are **disabled** on iOS (no second-process execution under App Store review — see [security.md §iOS](10-security.md#ios-app-store-strategy)). iOS users get an "unavailable on this device" banner.

**Quotas:**
- Default RSS 512 MB, CPU quota 80% of one core; raisable via `host.limits.memMb` / `host.limits.cpuPct` subject to user confirm at install.
- Startup deadline 5 s to emit a JSON-RPC `initialize` response; else supervisor kills and retries with backoff (2s, 4s, 8s, 16s, then disabled).

## Host ↔ Sidecar protocol (JSON-RPC 2.0)

Transport: stdio, LSP-style framing (`Content-Length: N\r\n\r\n<JSON>`).

### Required sidecar methods

All sidecars must implement:

- `initialize(params: InitializeParams) -> InitializeResult`
- `shutdown() -> void`  (host waits up to 2s, then SIGKILL)
- `exit() -> void` (notification)

### Lifecycle notifications host → sidecar

- `activate` — sent once after `initialize` acknowledgement.
- `deactivate` — sent before `shutdown`.
- `permissions/update` — sent when user revokes a capability.
- `config/update` — sent when user updates `contributes.settings` values.

### Event notifications host → sidecar

When plugin subscribes via manifest or `events.subscribe`:
- `event/<name>` with `params: { name, data, ts }`.

### Calls sidecar → host

Sidecar uses the same JSON-RPC channel to call `opendray.*`. Method mapping: `fs/readFile`, `exec/run`, `http/request`, `events/publish`, `ui/render`, `storage/get`, etc. Response contract identical to the webview wire protocol (§04).

### Streaming

Long-running responses use JSON-RPC notifications with a `streamId` correlator:
```
// request
{ "jsonrpc":"2.0", "id":7, "method":"exec/spawn", "params":{...} }
// responses
{ "jsonrpc":"2.0", "id":7, "result":{ "streamId":"s-42" } }
{ "jsonrpc":"2.0", "method":"$/stream", "params":{ "streamId":"s-42", "kind":"stdout", "data":"..." } }
{ "jsonrpc":"2.0", "method":"$/stream", "params":{ "streamId":"s-42", "kind":"end", "exitCode":0 } }
```

### Method namespacing

Plugin-contributed methods (commands, debuggers, LSP) are prefixed by the plugin name:
`myplugin/<method>`. Host routes `opendray.commands.execute('myplugin.refresh')` to `myplugin/refresh` on the sidecar.

### Errors

Use JSON-RPC error codes. OpenDray reserves `-32000..-32099` (see §04 error codes mapped 1:1).

## Quick comparison

| | Declarative | Webview | Host |
|--|--|--|--|
| Plugin code executed? | no | JS in WebView | yes (native or scripted) |
| Works on iOS? | yes | yes | no (v1) |
| Max binary size | 2 MB | 20 MB | 200 MB |
| Startup cost | zero | 100-500 ms WebView | 50 ms - 2 s |
| Can speak LSP? | no | no | yes |
| Can add themes / keybindings? | yes | yes | yes |
| Debugging story | logs only | DevTools | attach to PID |

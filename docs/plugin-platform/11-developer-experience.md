# 11 — Developer Experience

## `@opendray/plugin-sdk` npm package

Shipping layout:
```
@opendray/plugin-sdk/
  package.json
  dist/
    index.js              # runtime stub used in dev/test (no-ops bridge calls)
    index.d.ts            # TS types — the file defined verbatim in 04-bridge-api.md
    bridge.js             # minimal wire protocol impl for webview bundles
  schemas/
    manifest-v1.json      # the schema from 02-manifest.md
  cli.js                  # invoked via `opendray plugin <cmd>`
```

Install:
```
npm i -D @opendray/plugin-sdk
```

Bin:
```
"bin": { "opendray": "./cli.js" }
```

### SDK shape (public)

```ts
// Webview entry — grab a typed handle for your plugin
import { getOpenDray } from '@opendray/plugin-sdk';
const od = getOpenDray();
await od.workbench.showMessage('ready');

// Host sidecar entry — grab a typed JSON-RPC server
import { createHost } from '@opendray/plugin-sdk/host';
const host = createHost({
  async initialize(params) { return { capabilities: {} }; },
  async 'myext/refresh'(_args) { return { ok: true }; },
});
host.listen();
```

## CLI `opendray plugin <cmd>`

| Command | Purpose |
|---------|---------|
| `opendray plugin scaffold <name>` | Interactive; pick form + vertical template |
| `opendray plugin validate [dir]` | Validate manifest.json + structure |
| `opendray plugin dev [dir]` | Watch + hot reload against a running Opendray host |
| `opendray plugin build [dir]` | Produce a distributable zip |
| `opendray plugin publish [dir]` | Build + sign + open PR against marketplace |
| `opendray plugin lint [dir]` | Static checks (forbidden APIs, capability drift) |
| `opendray plugin sign --key <path>` | Sign bundle with ed25519/minisign |
| `opendray plugin doctor` | Diagnose local environment |

## Scaffold templates

```
opendray plugin scaffold --form declarative --template statusbar
opendray plugin scaffold --form webview     --template view-react
opendray plugin scaffold --form webview     --template view-svelte
opendray plugin scaffold --form host        --template lsp
opendray plugin scaffold --form host        --template db-panel
opendray plugin scaffold --form host        --template task-runner
opendray plugin scaffold --form host        --template agent-provider
opendray plugin scaffold --form declarative --template theme
opendray plugin scaffold --form declarative --template snippets
opendray plugin scaffold --form declarative --template telegram-command
```

Each template produces a runnable project with:
- `manifest.json` pre-filled with required fields, minimal `permissions`, one working contribution.
- `README.md` with 3 commands to get started.
- `package.json` with `opendray dev`, `opendray build`, `opendray publish`.
- TypeScript setup if webview/host.
- GitHub Actions workflow for CI validation.

## Dev-mode hot reload

`opendray plugin dev`:
1. Reads `manifest.json`.
2. Calls the host at `POST /api/plugins/dev/register` with the local path.
3. Host mounts `local:/path/to/dir/` with a special `sideloaded+dev` trust level and auto-activates.
4. Watches the dir; on any change:
   - Declarative: re-registers manifest.
   - Webview: tells Flutter shell to `reload` the view's WebView.
   - Host: sends `deactivate`, relaunches sidecar, sends `activate`.
5. Errors print to CLI in real time and to the host's Settings → Plugins → Logs.

> **Locked:** Dev mode is gated by `OPENDRAY_ALLOW_LOCAL_PLUGINS=1` on the host, off in production builds.

## Debugging

### Log drawer
- Every `opendray.logger.*` call lands in Settings → Plugins → <name> → Logs (tail + search).
- Filterable by level.
- Structured logs (`{ msg, meta }`) rendered as expandable rows.

### Breakpoints
- **Webview:** Chrome DevTools via Android USB / Safari Web Inspector via iOS USB. On desktop, a "Open DevTools" button appears in Settings → Plugins → <name> when running a debug build of OpenDray.
- **Host (node/deno):** `--inspect` flag wired up by `opendray plugin dev --inspect`; connect from VS Code with the launch config scaffolded by the template.
- **Host (binary):** attach your preferred debugger (`lldb`, `gdb`, Delve) to the sidecar PID shown in the Settings → Plugins → <name> → Diagnostics page.

### Bridge trace
`opendray plugin dev --trace-bridge` streams every bridge call + response to stdout:
```
> fs.readFile ["/etc/hosts"]  [plugin=myext]  denied:EPERM:path not in fs.read
< 11ms
```

## Testing harness

`@opendray/plugin-sdk/test` exports a mock host:
```ts
import { mockHost } from '@opendray/plugin-sdk/test';
const host = mockHost();
host.fs.withFile('/tmp/x', 'hi');
const od = host.asOpenDray({ plugin: { name: 'myext', version: '0.1.0', publisher: 'me' }});
expect(await od.fs.readFile('/tmp/x')).toBe('hi');
expect(host.auditLog.filter(a => a.ns === 'fs')).toHaveLength(1);
```

Ships `vitest` config template for webview plugins, `node:test` template for Node host plugins.

## Localization

- Manifest `displayName`, `description`, and every `title` field support `%keys%`.
- Per-locale JSON files under `i18n/`:
  ```
  i18n/
    en.json
    zh-CN.json
  ```
  ```json
  { "displayName": "Kanban", "view.board.title": "Board" }
  ```
- Host picks the locale matching the app's current locale, falls back to `en`.

## Debug output conventions

- Error messages must include plugin name prefix: `[myext] failed to load: ...`.
- Structured fields preferred over string interpolation.
- No `console.log` from webview in a shipped plugin — lint rule in the SDK.

## Documentation generation

`opendray plugin docs` generates a Markdown page from:
- Manifest fields (identity, capabilities, contributions).
- JSDoc on sidecar JSON-RPC method handlers (Node template).
- README.md merged in.

Publishers are encouraged to commit the generated `DOC.md` to their repo.

## Reference projects

A separate repo `opendray/plugin-examples` contains one working plugin per template. Each example is independently installable via `marketplace://opendray-examples/*`.

> **Locked (2026-04-19):** `opendray-dev` portable host ships with the SDK in M6. Plugin authors must be able to develop offline without a full OpenDray instance — this is the `code --extensionDevelopmentPath` equivalent and is load-bearing for ecosystem DX.

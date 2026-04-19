# 04 — Bridge API (`opendray.*`)

The Bridge API is the single surface through which plugin code (webview JS or sidecar JSON-RPC) talks to the OpenDray host. Every call is capability-gated, rate-limited, and audited.

The TypeScript declarations below are authoritative — they ship verbatim as `@opendray/plugin-sdk/index.d.ts`.

## Wire protocol (webview only)

Webview plugins get `window.opendray` injected by a preload script. Under the hood every call is a `postMessage` between the WebView and Flutter, which forwards over the plugin WebSocket.

### Request envelope
```json
{ "v": 1, "id": "42", "ns": "fs", "method": "readFile",
  "args": ["/tmp/x.txt"], "token": "<session-scoped bridge token>" }
```
### Response envelope
```json
{ "v": 1, "id": "42", "result": "...base64..." }
```
### Error envelope
```json
{ "v": 1, "id": "42", "error": { "code": "EPERM", "message": "fs.read not granted for /etc" } }
```
### Streaming envelope
```json
{ "v": 1, "id": "42", "stream": "chunk", "data": "..." }
{ "v": 1, "id": "42", "stream": "end" }
```

### Error codes (stable set)
`EPERM` capability denied · `EINVAL` bad args · `ENOENT` not found · `ETIMEOUT` rate-limited · `EUNAVAIL` host method offline · `EINTERNAL` host bug. Plugins must tolerate unknown codes (treat as fatal).

## Rate limits (default, per plugin, per minute)

| Namespace | Limit | Notes |
|-----------|-------|-------|
| `fs.*` read | 600 | |
| `fs.*` write | 120 | |
| `exec.*` | 60 | each spawn counts once |
| `http.request` | 300 | |
| `storage.*` | 600 | |
| `secret.*` | 60 | |
| `llm.*` | 60 | |
| `events.publish` | 300 | |
| all other | 300 | |

429 → `ETIMEOUT` with `Retry-After` header-style field in error data.

## TypeScript surface

```ts
// @opendray/plugin-sdk — public entry
declare global {
  interface Window { opendray: OpenDray; }
}

export interface OpenDray {
  version: string;                           // host version
  plugin:   PluginContext;                   // this plugin's identity
  workbench:ReturnType<typeof mkWorkbench>;
  fs:       FsApi;
  exec:     ExecApi;
  http:     HttpApi;
  session:  SessionApi;
  storage:  StorageApi;
  secret:   SecretApi;
  events:   EventsApi;
  commands: CommandsApi;
  tasks:    TasksApi;
  ui:       UiApi;
  clipboard:ClipboardApi;
  llm:      LlmApi;
  git:      GitApi;
  telegram: TelegramApi;
  logger:   LoggerApi;
}

export interface PluginContext {
  name: string; version: string; publisher: string;
  dataDir: string;            // per-plugin read/write dir
  workspaceRoot?: string;     // current session cwd if any
}
```

### `workbench`
```ts
export interface WorkbenchApi {
  /** Toast / banner at top. Returns when dismissed. */
  showMessage(msg: string, opts?: { kind?: 'info'|'warn'|'error'; durationMs?: number }): Promise<void>;
  /** Modal confirm. */
  confirm(msg: string, opts?: { okLabel?: string; cancelLabel?: string }): Promise<boolean>;
  /** Input prompt. */
  prompt(msg: string, opts?: { placeholder?: string; password?: boolean; default?: string }): Promise<string|null>;
  /** Focus a contributed view. */
  openView(viewId: string): Promise<void>;
  /** Update status-bar item. Only items contributed by this plugin. */
  updateStatusBar(id: string, patch: { text?: string; tooltip?: string; command?: string }): Promise<void>;
  /** Open the command palette pre-filtered. */
  runCommand(commandId: string, ...args: unknown[]): Promise<unknown>;
  /** Current theme (id + kind). */
  theme(): Promise<{ id: string; kind: 'light'|'dark'|'high-contrast' }>;
  /** Observe theme changes. */
  onThemeChange(cb: (t: { id: string; kind: string }) => void): Disposable;
}
```
Capability: none.
Usage:
```ts
await opendray.workbench.showMessage('Deploy finished', { kind: 'info' });
```

### `fs`
```ts
export interface FsApi {
  readFile(path: string, opts?: { encoding?: 'utf8'|'base64' }): Promise<string>;
  writeFile(path: string, data: string, opts?: { encoding?: 'utf8'|'base64'; mode?: number }): Promise<void>;
  exists(path: string): Promise<boolean>;
  stat(path: string): Promise<{ size: number; mtime: number; isDir: boolean }>;
  readDir(path: string): Promise<Array<{ name: string; isDir: boolean }>>;
  mkdir(path: string, opts?: { recursive?: boolean }): Promise<void>;
  remove(path: string, opts?: { recursive?: boolean }): Promise<void>;
  watch(glob: string, cb: (ev: { kind: 'create'|'modify'|'delete'; path: string }) => void): Disposable;
}
```
Capability: `fs.read` / `fs.write` with path match against `permissions.fs.{read|write}`. Plugins **must** use `${workspace}`, `${home}`, `${dataDir}` variables in declared paths; raw absolute paths outside those trees are rejected at install time.
Usage:
```ts
const txt = await opendray.fs.readFile(`${opendray.plugin.workspaceRoot}/README.md`);
```

### `exec`
```ts
export interface ExecApi {
  /** One-shot. Captures stdout/stderr. */
  run(cmd: string, args: string[], opts?: ExecOpts): Promise<ExecResult>;
  /** Spawn + stream. */
  spawn(cmd: string, args: string[], opts?: ExecOpts): ExecHandle;
}
export interface ExecOpts { cwd?: string; env?: Record<string,string>; timeoutMs?: number; input?: string; }
export interface ExecResult { exitCode: number; stdout: string; stderr: string; timedOut: boolean; }
export interface ExecHandle {
  readonly pid: number;
  stdout: AsyncIterable<string>;
  stderr: AsyncIterable<string>;
  write(input: string): Promise<void>;
  kill(signal?: 'SIGTERM'|'SIGKILL'): Promise<void>;
  wait(): Promise<{ exitCode: number }>;
}
```
Capability: `exec` — either `true` (any) or string-glob list like `["git *", "npm run *"]`. Host matches `cmd + " " + args.join(' ')` against each glob at call time.
Usage:
```ts
const r = await opendray.exec.run('git', ['status','--short']);
```

### `http`
```ts
export interface HttpApi {
  request(req: HttpRequest): Promise<HttpResponse>;
  /** SSE / chunked helper. */
  stream(req: HttpRequest): AsyncIterable<Uint8Array>;
}
export interface HttpRequest {
  url: string; method?: 'GET'|'POST'|'PUT'|'PATCH'|'DELETE'|'HEAD';
  headers?: Record<string,string>; body?: string | Uint8Array;
  timeoutMs?: number;
}
export interface HttpResponse { status: number; headers: Record<string,string>; body: Uint8Array; }
```
Capability: `http` — boolean or URL-pattern list (`["https://api.example.com/*"]`).
> **Locked:** Host denies connections to RFC1918 ranges and link-local unless explicitly pattern-listed. Prevents trivial SSRF from a compromised plugin.

### `session`
```ts
export interface SessionApi {
  list(): Promise<SessionInfo[]>;
  get(id: string): Promise<SessionInfo>;
  create(req: CreateSession): Promise<SessionInfo>;
  start(id: string): Promise<void>;
  stop(id: string): Promise<void>;
  sendInput(id: string, data: string): Promise<void>;        // write
  readBuffer(id: string, lines?: number): Promise<string>;   // read
  onEvent(cb: (ev: SessionEvent) => void): Disposable;       // requires events: ['session.*']
}
```
Capability: `session: "read"` or `"write"`. `"write"` implies read.

### `storage` — per-plugin KV
```ts
export interface StorageApi {
  get<T=unknown>(key: string, fallback?: T): Promise<T>;
  set(key: string, value: unknown): Promise<void>;
  delete(key: string): Promise<void>;
  list(prefix?: string): Promise<string[]>;
}
```
Capability: `storage: true`. Stored in `kernel/store` under `plugin_kv` (new table, M1). 1 MB per key, 100 MB per plugin hard cap.

### `secret` — encrypted per-plugin secrets
```ts
export interface SecretApi {
  get(key: string): Promise<string|null>;
  set(key: string, value: string): Promise<void>;
  delete(key: string): Promise<void>;
}
```
Capability: `secret: true`. Backed by the existing `auth.CredentialStore` encryption key. Never exposed to webview chrome; values returned only to the owning plugin.

### `events`
```ts
export interface EventsApi {
  /** Subscribe to a host event name. Namespaces: session.*, task.*, git.*, fs.*, workbench.*, plugin.*. */
  subscribe(name: string, cb: (ev: Event) => void): Disposable;
  /** Publish a plugin-scoped event. Name is auto-prefixed with plugin.<name>.  */
  publish(name: string, payload: unknown): Promise<void>;
}
```
Capability: `events: [patterns]` e.g. `["session.*","git.status"]`. Host denies `subscribe` for names not matching any pattern. Publishing under own prefix always allowed.

The existing `plugin.HookBus` becomes the wire for this API; `HookOnOutput`/`HookOnIdle`/`HookOnSessionStart`/`HookOnSessionStop` map to `session.output`, `session.idle`, `session.start`, `session.stop`.

### `commands`
```ts
export interface CommandsApi {
  register(id: string, fn: (...args: unknown[]) => unknown | Promise<unknown>): Disposable;
  execute(id: string, ...args: unknown[]): Promise<unknown>;
  list(): Promise<Array<{ id: string; title: string; pluginName: string }>>;
}
```
Capability: registration is always allowed for commands declared in own manifest; executing another plugin's command requires that plugin to mark the command as `exported: true` (post-v1; v1 allows cross-plugin execute freely but logs it).

### `tasks`
```ts
export interface TasksApi {
  list(): Promise<TaskDef[]>;
  run(id: string, args?: Record<string,string>): Promise<TaskRunHandle>;
  onRunChange(cb: (run: TaskRun) => void): Disposable;
}
```
Capability: `exec` equivalent; tasks always run with the user's env.

### `ui` — declarative tree for non-webview views
```ts
export interface UiApi {
  render(viewId: string, tree: UiNode): Promise<void>;
  patch(viewId: string, path: string[], patch: Partial<UiNode>): Promise<void>;
  onAction(viewId: string, cb: (action: { id: string; payload: unknown }) => void): Disposable;
}
export type UiNode =
  | { type: 'column'|'row'; children: UiNode[]; pad?: number; gap?: number }
  | { type: 'text'; text: string; style?: 'title'|'body'|'caption' }
  | { type: 'button'; id: string; label: string; icon?: string }
  | { type: 'list'; items: Array<{ id: string; title: string; subtitle?: string; icon?: string }> }
  | { type: 'input'; id: string; label?: string; placeholder?: string; value?: string }
  | { type: 'switch'; id: string; label: string; value: boolean }
  | { type: 'divider' };
```
Renders natively — no WebView. Use this for fast, theme-consistent views.

### `clipboard`
```ts
export interface ClipboardApi { readText(): Promise<string>; writeText(s: string): Promise<void>; }
```
Capability: `clipboard: "read" | "write" | "readwrite"`.

### `llm` — AI helper bound to the current session's provider
```ts
export interface LlmApi {
  complete(req: LlmRequest): Promise<LlmResponse>;
  stream(req: LlmRequest): AsyncIterable<LlmChunk>;
  listModels(providerId?: string): Promise<string[]>;
}
export interface LlmRequest { prompt: string; model?: string; system?: string; temperature?: number; maxTokens?: number; providerId?: string; }
```
Capability: `llm: true`. Host routes through the existing `gateway/llm_proxy` and counts tokens against the provider's quota.

### `git`
```ts
export interface GitApi {
  status(repoPath?: string): Promise<GitStatus>;
  diff(opts?: { staged?: boolean; path?: string }): Promise<string>;
  log(opts?: { limit?: number }): Promise<GitCommit[]>;
  stage(paths: string[]): Promise<void>;   // requires write
  commit(msg: string): Promise<string>;    // requires write
}
```
Capability: `git: "read" | "write"`. Backed by `gateway/git`.

### `telegram`
```ts
export interface TelegramApi {
  sendMessage(opts: { chatId?: string; text: string; parseMode?: 'MarkdownV2'|'HTML' }): Promise<void>;
  onCommand(command: string, cb: (ctx: TelegramCtx) => void): Disposable; // only for contributed telegramCommands
}
```
Capability: `telegram: true`. The plugin may only send to `notifyChatId` or allowed chats configured in the `telegram` plugin.

### `logger`
```ts
export interface LoggerApi {
  debug(msg: string, meta?: object): void;
  info (msg: string, meta?: object): void;
  warn (msg: string, meta?: object): void;
  error(msg: string, meta?: object): void;
}
```
Capability: none. Lines surface in Settings → Plugins → Logs (tail).

### `Disposable`
```ts
export interface Disposable { dispose(): void; }
```

## Usage snippets (5-line form)

**Read file:**
```ts
const md = await opendray.fs.readFile(`${opendray.plugin.workspaceRoot}/README.md`);
await opendray.workbench.showMessage(`README is ${md.length} bytes`);
```

**Spawn git:**
```ts
const r = await opendray.exec.run('git', ['rev-parse','HEAD'], { cwd: opendray.plugin.workspaceRoot });
opendray.logger.info('head', { sha: r.stdout.trim() });
```

**Subscribe session idle:**
```ts
opendray.events.subscribe('session.idle', e =>
  opendray.telegram.sendMessage({ text: `Session ${e.sessionId} idle` })
);
```

**Render declarative view:**
```ts
await opendray.ui.render('myext.view', {
  type: 'column', pad: 16, gap: 8,
  children: [{ type: 'text', text: 'Hello', style: 'title' },
             { type: 'button', id: 'refresh', label: 'Refresh' }]
});
opendray.ui.onAction('myext.view', a => a.id === 'refresh' && refresh());
```

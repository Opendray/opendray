# 04 — 桥接 API (`opendray.*`)

桥接 API 是插件代码（Web 视图 JS 或边车 JSON-RPC）与 OpenDray 宿主通信的唯一接口。每次调用都受到能力门控、速率限制并经过审计。

下方的 TypeScript 声明具有权威性 —— 它们原样作为 `@opendray/plugin-sdk/index.d.ts` 发布。

## 传输协议 (仅限 Web 视图)

Web 视图插件通过预加载脚本注入 `window.opendray`。在底层，每次调用都是 Web 视图与 Flutter 之间的 `postMessage`，Flutter 随后通过插件 WebSocket 进行转发。

### 请求信封 (Request envelope)
```json
{ "v": 1, "id": "42", "ns": "fs", "method": "readFile",
  "args": ["/tmp/x.txt"], "token": "<会话范围的桥接令牌>" }
```
### 响应信封 (Response envelope)
```json
{ "v": 1, "id": "42", "result": "...base64..." }
```
### 错误信封 (Error envelope)
```json
{ "v": 1, "id": "42", "error": { "code": "EPERM", "message": "未授予对 /etc 的 fs.read 权限" } }
```
### 流式传输信封 (Streaming envelope)
```json
{ "v": 1, "id": "42", "stream": "chunk", "data": "..." }
{ "v": 1, "id": "42", "stream": "end" }
```

### 错误代码 (稳定集合)
`EPERM` 能力被拒绝 · `EINVAL` 参数错误 · `ENOENT` 未找到 · `ETIMEOUT` 速率限制 · `EUNAVAIL` 宿主方法离线 · `EINTERNAL` 宿主错误。插件必须容忍未知的错误代码（将其视为致命错误）。

## 速率限制 (默认，每个插件，每分钟)

| 命名空间 | 限制 | 备注 |
|-----------|-------|-------|
| `fs.*` 读取 | 600 | |
| `fs.*` 写入 | 120 | |
| `exec.*` | 60 | 每次 spawn 计为一次 |
| `http.request` | 300 | |
| `storage.*` | 600 | |
| `secret.*` | 60 | |
| `llm.*` | 60 | |
| `events.publish` | 300 | |
| 所有其他 | 300 | |

429 → 返回 `ETIMEOUT`，错误数据中带有类似 `Retry-After` 标头的字段。

## TypeScript 接口表面

```ts
// @opendray/plugin-sdk — 公共入口
declare global {
  interface Window { opendray: OpenDray; }
}

export interface OpenDray {
  version: string;                           // 宿主版本
  plugin:   PluginContext;                   // 此插件的身份
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
  dataDir: string;            // 每个插件的读写目录
  workspaceRoot?: string;     // 当前会话的工作区根目录（如果有）
}
```

### `workbench` (工作台)
```ts
export interface WorkbenchApi {
  /** 顶部的气泡提示 / 横幅。关闭时返回。 */
  showMessage(msg: string, opts?: { kind?: 'info'|'warn'|'error'; durationMs?: number }): Promise<void>;
  /** 模态确认框。 */
  confirm(msg: string, opts?: { okLabel?: string; cancelLabel?: string }): Promise<boolean>;
  /** 输入提示框。 */
  prompt(msg: string, opts?: { placeholder?: string; password?: boolean; default?: string }): Promise<string|null>;
  /** 聚焦一个贡献的视图。 */
  openView(viewId: string): Promise<void>;
  /** 更新状态栏项目。仅限此插件贡献的项目。 */
  updateStatusBar(id: string, patch: { text?: string; tooltip?: string; command?: string }): Promise<void>;
  /** 打开预过滤的命令面板。 */
  runCommand(commandId: string, ...args: unknown[]): Promise<unknown>;
  /** 当前主题 (id + kind)。 */
  theme(): Promise<{ id: string; kind: 'light'|'dark'|'high-contrast' }>;
  /** 监听主题变化。 */
  onThemeChange(cb: (t: { id: string; kind: string }) => void): Disposable;
}
```
能力：无。
用法：
```ts
await opendray.workbench.showMessage('部署完成', { kind: 'info' });
```

### `fs` (文件系统)
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
能力：`fs.read` / `fs.write`，路径需与 `permissions.fs.{read|write}` 中的通配符匹配。插件**必须**在声明的路径中使用 `${workspace}`, `${home}`, `${dataDir}` 变量；在安装时会拒绝这些树结构之外的原始绝对路径。
用法：
```ts
const txt = await opendray.fs.readFile(`${opendray.plugin.workspaceRoot}/README.md`);
```

### `exec` (执行)
```ts
export interface ExecApi {
  /** 单次执行。捕获标准输出/标准错误。 */
  run(cmd: string, args: string[], opts?: ExecOpts): Promise<ExecResult>;
  /** 启动进程并流式传输。 */
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
能力：`exec` —— 可以是 `true` (任何) 或类似 `["git *", "npm run *"]` 的字符串通配符列表。宿主在调用时会根据每个通配符匹配 `cmd + " " + args.join(' ')`。
用法：
```ts
const r = await opendray.exec.run('git', ['status','--short']);
```

### `http`
```ts
export interface HttpApi {
  request(req: HttpRequest): Promise<HttpResponse>;
  /** SSE / 分块助手。 */
  stream(req: HttpRequest): AsyncIterable<Uint8Array>;
}
export interface HttpRequest {
  url: string; method?: 'GET'|'POST'|'PUT'|'PATCH'|'DELETE'|'HEAD';
  headers?: Record<string,string>; body?: string | Uint8Array;
  timeoutMs?: number;
}
export interface HttpResponse { status: number; headers: Record<string,string>; body: Uint8Array; }
```
能力：`http` —— 布尔值或 URL 模式列表 (`["https://api.example.com/*"]`)。
> **已锁定：** 宿主拒绝连接到 RFC1918 范围和链路本地地址，除非明确列出。这可以防止受损插件进行简单的 SSRF。

### `session` (会话)
```ts
export interface SessionApi {
  list(): Promise<SessionInfo[]>;
  get(id: string): Promise<SessionInfo>;
  create(req: CreateSession): Promise<SessionInfo>;
  start(id: string): Promise<void>;
  stop(id: string): Promise<void>;
  sendInput(id: string, data: string): Promise<void>;        // 写入
  readBuffer(id: string, lines?: number): Promise<string>;   // 读取
  onEvent(cb: (ev: SessionEvent) => void): Disposable;       // 需要 events: ['session.*']
}
```
能力：`session: "read"` 或 `"write"`。`"write"` 包含读取。

### `storage` — 每个插件的键值对存储 (KV)
```ts
export interface StorageApi {
  get<T=unknown>(key: string, fallback?: T): Promise<T>;
  set(key: string, value: unknown): Promise<void>;
  delete(key: string): Promise<void>;
  list(prefix?: string): Promise<string[]>;
}
```
能力：`storage: true`。存储在 `kernel/store` 的 `plugin_kv` 表中（M1 新增表）。每个键 1 MB，每个插件 100 MB 硬限制。

### `secret` — 加密后的插件专用机密
```ts
export interface SecretApi {
  get(key: string): Promise<string|null>;
  set(key: string, value: string): Promise<void>;
  delete(key: string): Promise<void>;
}
```
能力：`secret: true`。由现有的 `auth.CredentialStore` 加密密钥支持。绝不会暴露给 Web 视图的 Chrome 控制台；值仅返回给所属插件。

### `events` (事件)
```ts
export interface EventsApi {
  /** 订阅宿主事件。命名空间：session.*, task.*, git.*, fs.*, workbench.*, plugin.*。 */
  subscribe(name: string, cb: (ev: Event) => void): Disposable;
  /** 发布插件范围的事件。名称会自动添加 plugin.<name>. 前缀。 */
  publish(name: string, payload: unknown): Promise<void>;
}
```
能力：`events: [patterns]` 例如 `["session.*","git.status"]`。宿主会拒绝订阅不匹配任何模式的名称。始终允许在自己的前缀下发布。

现有的 `plugin.HookBus` 成为该 API 的传输线路；`HookOnOutput`/`HookOnIdle`/`HookOnSessionStart`/`HookOnSessionStop` 映射到 `session.output`, `session.idle`, `session.start`, `session.stop`。

### `commands` (命令)
```ts
export interface CommandsApi {
  register(id: string, fn: (...args: unknown[]) => unknown | Promise<unknown>): Disposable;
  execute(id: string, ...args: unknown[]): Promise<unknown>;
  list(): Promise<Array<{ id: string; title: string; pluginName: string }>>;
}
```
能力：始终允许为自己清单中声明的命令进行注册；执行另一个插件的命令需要该插件将命令标记为 `exported: true`（post-v1；v1 允许自由地跨插件执行，但会记录日志）。

### `tasks` (任务)
```ts
export interface TasksApi {
  list(): Promise<TaskDef[]>;
  run(id: string, args?: Record<string,string>): Promise<TaskRunHandle>;
  onRunChange(cb: (run: TaskRun) => void): Disposable;
}
```
能力：等同于 `exec`；任务始终以用户的环境运行。

### `ui` — 非 Web 视图视图的声明式树
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
原生渲染 —— 无 WebView。适用于快速、且与主题一致的视图。

### `clipboard` (剪贴板)
```ts
export interface ClipboardApi { readText(): Promise<string>; writeText(s: string): Promise<void>; }
```
能力：`clipboard: "read" | "write" | "readwrite"`。

### `llm` — 绑定到当前会话提供商的 AI 助手
```ts
export interface LlmApi {
  complete(req: LlmRequest): Promise<LlmResponse>;
  stream(req: LlmRequest): AsyncIterable<LlmChunk>;
  listModels(providerId?: string): Promise<string[]>;
}
export interface LlmRequest { prompt: string; model?: string; system?: string; temperature?: number; maxTokens?: number; providerId?: string; }
```
能力：`llm: true`。宿主通过现有的 `gateway/llm_proxy` 进行路由，并根据提供商的配额计算令牌（Tokens）。

### `git`
```ts
export interface GitApi {
  status(repoPath?: string): Promise<GitStatus>;
  diff(opts?: { staged?: boolean; path?: string }): Promise<string>;
  log(opts?: { limit?: number }): Promise<GitCommit[]>;
  stage(paths: string[]): Promise<void>;   // 需要写入权限
  commit(msg: string): Promise<string>;    // 需要写入权限
}
```
能力：`git: "read" | "write"`。由 `gateway/git` 支持。

### `telegram`
```ts
export interface TelegramApi {
  sendMessage(opts: { chatId?: string; text: string; parseMode?: 'MarkdownV2'|'HTML' }): Promise<void>;
  onCommand(command: string, cb: (ctx: TelegramCtx) => void): Disposable; // 仅限贡献的 telegramCommands
}
```
能力：`telegram: true`。插件只能发送到 `notifyChatId` 或 `telegram` 插件中配置的可允许聊天中。

### `logger` (日志记录器)
```ts
export interface LoggerApi {
  debug(msg: string, meta?: object): void;
  info (msg: string, meta?: object): void;
  warn (msg: string, meta?: object): void;
  error(msg: string, meta?: object): void;
}
```
能力：无。日志显示在“设置 → 插件 → 日志”中（实时查看）。

### `Disposable` (可释放对象)
```ts
export interface Disposable { dispose(): void; }
```

## 用法代码段 (5 行形式)

**读取文件：**
```ts
const md = await opendray.fs.readFile(`${opendray.plugin.workspaceRoot}/README.md`);
await opendray.workbench.showMessage(`README 大小为 ${md.length} 字节`);
```

**运行 git：**
```ts
const r = await opendray.exec.run('git', ['rev-parse','HEAD'], { cwd: opendray.plugin.workspaceRoot });
opendray.logger.info('head', { sha: r.stdout.trim() });
```

**订阅会话空闲事件：**
```ts
opendray.events.subscribe('session.idle', e =>
  opendray.telegram.sendMessage({ text: `会话 ${e.sessionId} 已空闲` })
);
```

**渲染声明式视图：**
```ts
await opendray.ui.render('myext.view', {
  type: 'column', pad: 16, gap: 8,
  children: [{ type: 'text', text: '你好', style: 'title' },
             { type: 'button', id: 'refresh', label: '刷新' }]
});
opendray.ui.onAction('myext.view', a => a.id === 'refresh' && refresh());
```

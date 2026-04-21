# 06 — 插件形式 (Plugin Formats)

每个插件都必须在三种形式中选择一种。请选择能完成任务的权限最小的形式。

## 决策树

```
是否需要后台 CPU、常驻进程或 LSP/DAP？
│
├── 是 ──► 宿主 (Host)
│
└── 否 ──► 是否需要原生 `ui.*` 树无法表达的自定义 UI（Canvas、富文本编辑器、Web 库）？
            │
            ├── 是 ──► Web 视图 (Webview)
            │
            └── 否 ──► 声明式 (Declarative)
```

## A. 声明式 (Declarative)

**用例：** 状态栏时钟、按键绑定、主题、菜单、命令快捷方式、Telegram 命令绑定。

**无法做到：** 除了 `contributes.*` 以外的自定义 UI 渲染，订阅带有自定义处理程序的事件（事件只能触发声明式的 `run.kind`），复杂逻辑。

**沙箱保证：**
- 完全在 Go 宿主内部运行。
- 根本不执行任何插件代码 —— 一切皆数据。
- 无法启动进程、读取文件、调用网络（除非通过触发 `exec` 能力的声明式 `run.kind: "exec"`，且清单中必须声明该能力）。

**配额：** 不适用（仅占用宿主内存）。

## B. Web 视图 (Webview)

**用例：** 看板、仪表盘、文档/预览插件、数据可视化、富文本编辑器。

**插件包中的文件：**
```
my-plugin/
  manifest.json
  ui/
    index.html
    main.js
    style.css
    assets/
```

**沙箱保证：**
- 在专用的 `WebView` 中渲染（Android: `WebView`; iOS: `WKWebView`; 桌面端: 等效的 `flutter_inappwebview`）。
- 通过 `plugin://<name>/` 协议提供服务 —— 该自定义协议由 Flutter 外壳处理，字节流从已安装的插件包目录中读取。无法访问 `file://`。
- 默认 CSP（不可覆盖）：
  ```
  default-src 'self';
  script-src  'self' 'wasm-unsafe-eval';
  style-src   'self' 'unsafe-inline';
  img-src     'self' data: blob:;
  connect-src 'self' https://* wss://*;  // 受 http 能力限制
  frame-ancestors 'none';
  ```
- 每个插件的 Web 视图都是隔离的（独立的 Cookie 存储、localStorage、IndexedDB）—— v1 在平台支持的情况下为每个插件使用独立的 Web 视图进程。
- 不开放的 DOM API：`navigator.serviceWorker`、`navigator.geolocation`、`navigator.mediaDevices.*`（除非在 post-v1 中添加 `device` 能力）。
- `window.opendray` 通过预加载脚本注入；无其他宿主接口。

**配额：**
- 插件包大小 ≤ 20 MB。
- Web 视图内存软限制 128 MB（警告）；硬限制 256 MB（Web 视图将被终止并通知用户）。
- 安装后禁止下载代码（受 CSP 限制）。

## C. 宿主 / 边车 (Host/Sidecar)

**用例：** LSP、DAP、索引器、文件监听器、同步守护进程，以及任何计算密集型或需要原生库的任务。

**布局：**
```
my-plugin/
  manifest.json
  bin/
    linux-x64/<entry>
    darwin-arm64/<entry>
    windows-x64/<entry>.exe
  ui/                  # 可选 —— 宿主插件也可以贡献 Web 视图
    ...
```
> **已锁定：** 宿主插件可以额外携带 Web 视图 UI，因为某些语言插件（例如 Copilot 风格的）两者都需要。边车进程拥有方法接口；Web 视图通过 `opendray.commands.execute('<plugin>.<method>', ...)` 与边车通信，宿主将其代理为 JSON-RPC 调用。

**运行时类型：**
- `binary` — 根据 `platforms.<os>-<arch>` 选择的原生可执行文件。
- `node` — `runtime: "node"`，入口为 `.js`；宿主调用 `node <entry>`。插件必须声明 `engines.node`。
- `deno` — `runtime: "deno"`，入口为 `.ts` 或 `.js`；宿主调用 `deno run --allow-none <entry>`；受能力门控的权限叠加在上方。

**沙箱保证（尽力而为，取决于操作系统）：**
- Linux：在可用的情况下，进程在专用的无 `setuid` 用户上下文中启动，受到 `prlimit` 限制（默认 RLIMIT_AS 512 MB，RLIMIT_NPROC 32），并且（在可用时）使用 seccomp bpf 配置文件拒绝 `ptrace`、`mount`、`reboot`、`kexec_*`。
- macOS：sandbox-exec 配置文件，在未授予 `http` 能力时拒绝除宿主桥接以外的出站网络请求。
- Windows：使用 `JOB_OBJECT` 并设置 `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`，使得边车进程随宿主一同退出。
- iOS：**禁用**宿主插件（在 App Store 审核下不允许执行第二个进程 —— 参见 [security.md §iOS](10-security.md#ios-app-store-strategy)）。iOS 用户会看到“此设备不可用”的横幅。

**配额：**
- 默认 RSS 512 MB，CPU 配额为单核的 80%；可通过 `host.limits.memMb` / `host.limits.cpuPct` 提高限制，但需在安装时经用户确认。
- 启动截止时间为 5 秒，需发出 JSON-RPC `initialize` 响应；否则管理器将终止进程并以指数退避策略重试（2s, 4s, 8s, 16s，之后禁用）。

## 宿主 ↔ 边车协议 (JSON-RPC 2.0)

传输方式：标准输入输出 (stdio)，LSP 风格的分帧 (`Content-Length: N\r\n\r\n<JSON>`)。

### 必需的边车方法

所有边车必须实现：

- `initialize(params: InitializeParams) -> InitializeResult`
- `shutdown() -> void`  (宿主最多等待 2 秒，然后发送 SIGKILL)
- `exit() -> void` (通知)

### 生命周期通知 (宿主 → 边车)

- `activate` — 在 `initialize` 确认后发送一次。
- `deactivate` — 在 `shutdown` 前发送。
- `permissions/update` — 当用户撤销某项能力时发送。
- `config/update` — 当用户更新 `contributes.settings` 值时发送。

### 事件通知 (宿主 → 边车)

当插件通过清单或 `events.subscribe` 订阅事件时：
- `event/<name>` 带有 `params: { name, data, ts }`。

### 调用 (边车 → 宿主)

边车使用相同的 JSON-RPC 通道调用 `opendray.*`。方法映射：`fs/readFile`, `exec/run`, `http/request`, `events/publish`, `ui/render`, `storage/get` 等。响应契约与 Web 视图传输协议 (§04) 一致。

### 流式传输 (Streaming)

长时间运行的响应使用带有 `streamId` 关联因子的 JSON-RPC 通知：
```
// 请求
{ "jsonrpc":"2.0", "id":7, "method":"exec/spawn", "params":{...} }
// 响应
{ "jsonrpc":"2.0", "id":7, "result":{ "streamId":"s-42" } }
{ "jsonrpc":"2.0", "method":"$/stream", "params":{ "streamId":"s-42", "kind":"stdout", "data":"..." } }
{ "jsonrpc":"2.0", "method":"$/stream", "params":{ "streamId":"s-42", "kind":"end", "exitCode":0 } }
```

### 方法命名空间

插件贡献的方法（命令、调试器、LSP）以插件名称为前缀：
`myplugin/<method>`。宿主将 `opendray.commands.execute('myplugin.refresh')` 路由到边车上的 `myplugin/refresh`。

### 错误

使用 JSON-RPC 错误代码。OpenDray 保留 `-32000..-32099` 范围（参见 §04 中 1:1 映射的错误代码）。

## 快速对比

| | 声明式 | Web 视图 | 宿主 |
|--|--|--|--|
| 是否执行插件代码？ | 否 | Web 视图中的 JS | 是 (原生或脚本) |
| 是否支持 iOS？ | 是 | 是 | 否 (v1) |
| 最大二进制大小 | 2 MB | 20 MB | 200 MB |
| 启动成本 | 零 | 100-500 ms (Web 视图) | 50 ms - 2 s |
| 是否支持 LSP？ | 否 | 否 | 是 |
| 是否支持添加主题 / 按键绑定？ | 是 | 是 | 是 |
| 调试方案 | 仅日志 | DevTools | 附加到进程 (PID) |

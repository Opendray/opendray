# 01 — 架构

## 1. 分层职责

### 第 1 层 — 工作台外壳 (Workbench Shell) (Flutter 应用, 固定)
**拥有：** 导航外壳、活动条 (activity bar)、侧视图容器、底部面板容器、编辑器/终端区域、状态栏、通知、命令面板、主题、手机与平板布局适配、手势、键盘快捷键、无障碍。

**从不拥有：** 插件业务逻辑。外壳是清单 (manifest) 中声明的贡献点以及插件返回的视图负载的确定性渲染器。

**Go 包参考：** 不适用 (Flutter)。位于 `app/lib/workbench/` 下 (新模块 — 参见 [roadmap.md](12-roadmap.md) M2)。

### 第 2 层 — 插件宿主 (Plugin Host) (Go 后端, 固定)
**拥有：**
- 插件生命周期 — 下载、验证、安装、启用、激活、停用、卸载。
- 清单解析和能力强制执行。
- 桥接服务器 — 每一个 `opendray.*` 调用都通过一个检查能力、速率限制、审计的网关进行路由。
- 宿主形式 (Host-form) 插件的侧车 (sidecar) 监督 (fork/exec, stdio JSON-RPC, 带退避算法的重启)。
- 事件扇出 (目前的 `HookBus`)。
- 会话、任务、git、文件、日志、telegram、MCP 子系统 (现有的 `gateway/*` 包)。

**Go 包：**
- `plugin/` — 运行时、清单、钩子 (现有) 以及新的 `plugin/install`, `plugin/bridge`, `plugin/host` 子包 (M1)。
- `gateway/` — 用于安装 / 桥接的 HTTP 路由。
- `kernel/store/` — 数据库支持的插件状态。

### 第 3 层 — 插件 (Plugins) (下载的包)
三种形式，每个插件仅限一种：
1. **声明式 (Declarative)** — 仅清单。无自定义 UI，无自定义进程。纯数据：命令、状态栏项目、主题、快捷键绑定、代码片段、任务模板。
2. **WebView** — 清单 + 包含静态 Web 包 (HTML/CSS/JS/WASM) 的 `ui/` 文件夹。在 WebView 插槽中渲染。通过 `window.opendray.*` 与宿主通信。
3. **宿主 (Host)** — 清单 + 侧车可执行文件 (或 `runtime: "node" | "deno"` 脚本)。由插件宿主通过 stdio JSON-RPC 2.0 进行监督。用于 LSP、DAP、繁重的后台工作、语言工具。

插件在 v1 中**不得**混合使用多种形式。具有多个表面的功能 (例如同时具有 UI 和 LSP 的语言包) 应作为具有共享命名空间的两个插件发布。

## 2. 进程模型

```
+-------------------+            localhost HTTPS/WSS
| Flutter 工作台    |<----------------------------+
+-------------------+                             |
                                                  |
                         +------------------------v----------------+
                         | opendray 单一 Go 二进制文件 (应用的 PID 1) |
                         |                                         |
                         |  +-----------+  +---------------------+ |
                         |  | 网关      |  | 插件宿主            | |
                         |  | (chi)     |  |  - 运行时           | |
                         |  +-----^-----+  |  - 桥接网关         | |
                         |        |        |  - 能力门控         | |
                         |        +------->+  - HookBus          | |
                         |                 +----------^----------+ |
                         |                            | stdio      |
                         |     +----------+   +-------+---------+  |
                         |     | WebView  |   | 宿主侧车 (Host) |  |
                         |     | (内联    |   | (每个插件一个)  |  |
                         |     |  资源)   |   +-----------------+  |
                         |     +----------+                        |
                         +-----------------------------------------+
```

> **Locked:** 单一 Go 二进制文件。不需要外部服务 (无需 Redis, 无需 Kafka)。侧车是 opendray 进程的子进程，使用指数退避算法进行监督。这保护了现有的 LXC/Docker/单一二进制文件部署方案。

## 3. 通信通道

| 链接 | 协议 | 发起方 | 帧格式 |
|------|----------|---------------|---------|
| 工作台 ↔ 宿主 | HTTPS + WSS | Flutter → Go | REST JSON + WebSocket 文本帧 |
| WebView ↔ 宿主 | WebView JS 桥接上的 `postMessage` | JS `window.opendray` → Go | JSON 信封，参见 [bridge-api.md §Wire](04-bridge-api.md) |
| 侧车 ↔ 宿主 | stdio | Go ↔ 侧车 | LSP 风格的 JSON-RPC 2.0 (Content-Length 分帧) |
| 宿主 ↔ 宿主子系统 | 进程内函数调用 | Go | 原生 |

> **Locked:** 侧车使用带有 LSP 风格 Content-Length 分帧的 JSON-RPC 2.0。这与现有的 LSP 生态系统工具相匹配，因此 LSP 服务器可以作为 OpenDray 宿主插件进行封装，无需任何重写。

## 4. 请求流 — 插件安装

```
工作台                     网关                插件宿主             文件系统          市场
   |   POST /api/plugins/install  |                |                  |                |
   |  { src: "marketplace://acme/hello@0.1.0" }    |                  |                |
   |----------------------------->|                |                  |                |
   |                              | Install(src)   |                  |                |
   |                              |--------------->| resolve(src)     |                |
   |                              |                |----------------->|  GET index.json|
   |                              |                |                  |--------------->|
   |                              |                |                  |<-- meta --------|
   |                              |                |<-- 构件 URL+sha256 ----------------|
   |                              |                | download()       |                |
   |                              |                |--------------------------- GET zip |
   |                              |                |<------------------------- 字节     |
   |                              |                | verifySha256()   |                |
   |                              |                | verifySignature(可选)             |
   |                              |                | 解压 -> plugins/.installed/<name>/<ver>/
   |                              |                | parseManifest()  |                |
   |                              |                | capabilityDiff() |                |
   |   202 { consentRequired: [...perms] }         |                  |                |
   |<-----------------------------|                |                  |                |
   |   用户点击 "安装"             |                |                  |                |
   |   POST /api/plugins/install/confirm {token}   |                  |                |
   |----------------------------->|                |                  |                |
   |                              | confirm()      |                  |                |
   |                              |--------------->| persistConsent() |                |
   |                              |                | Runtime.Register()                |
   |                              |                | activate(如果是 onStartup)        |
   |   200 { installed, enabled } |                |                  |                |
   |<-----------------------------|                |                  |                |
```

## 5. 请求流 — WebView 视图渲染

```
用户点击活动条图标 "Hello"
  -> 工作台查找 contributes.views，其中 id=hello.view
  -> 工作台打开一个 WebView，src = plugin://hello-webview/ui/index.html
  -> WebView 加载；内容由内嵌的处理器提供，该处理器从 plugins/.installed/hello-webview/0.1.0/ui/ 流式传输字节
  -> ui/index.html 加载 ui/main.js，后者调用 window.opendray.workbench.ready()
  -> ui/main.js 调用 window.opendray.storage.get('hits', 0) -> 桥接调用 -> Go -> 返回 42
  -> 用户交互；任何状态都通过 opendray.storage.set(...) 持久化回来
```

## 6. 请求流 — 桥接调用

```
webview 代码:   await opendray.fs.readFile('/etc/hosts')
  -> JS SDK 发布 {id:42, ns:'fs', method:'readFile', args:['/etc/hosts']}
  -> 工作台 WebView 桥接通过会话 WebSocket 转发给 Go
  -> 网关 /api/plugins/<name>/bridge 处理器
  -> 插件宿主查找所需的能力 ('fs.read:/etc/hosts')
  -> 拒绝: 路径不在允许的根目录下 -> 错误 {id:42, error:{code:'EPERM',...}}
  -> 允许: 执行 + 审计日志 + 返回 {id:42, result: "<文件字节 base64>"}
```

## 7. 数据流 — 事件派发 (现有 HookBus 的演进)

```
会话 pty 输出 --> Hub.DispatchOutput
                          |
                          v
                     HookBus.Dispatch(HookEvent)
                          |
          +---------------+----------------+
          |                                |
          v                                v
   本地监听器 (Go)                    HTTP 订阅者
   - Telegram 桥接                   - 宿主侧车 (Host sidecar)  <- 封装为
   - 桥接网关向所有 WebView            通过其 stdio /events      opendray.events.subscribe
     发出 opendray.events.*            通知发送 POST
```

## 8. 在何处查找 current 代码

| 关注点 | 现状 | v1 包 |
|---------|-------|------------|
| 清单结构 | `plugin/manifest.go` | `plugin/manifest.go` (扩展) |
| 运行时 | `plugin/runtime.go` | `plugin/runtime.go` |
| 事件 | `plugin/hooks.go` | `plugin/hooks.go` (`opendray.events.*` 的来源) |
| 数据库 | `kernel/store/queries.go` (`Plugin` 结构体) | 相同，外加 `plugin_consents` 表 |
| 安装路由 | — | `gateway/plugins_install.go` (新) |
| 桥接路由 | — | `gateway/plugins_bridge.go` (新) |
| WebView 资源服务 | — | `gateway/plugins_assets.go` (新) |
| 侧车监督器 | — | `plugin/host/supervisor.go` (新) |

> **Locked (2026-04-19):** 每个插件拥有专用的 `/api/plugins/{name}/ws`。更清晰的源检查、插件运行时间与会话生命周期解耦、独立重连。已拒绝：共享的每个会话 WS (存在耦合 + 多路复用开销)。

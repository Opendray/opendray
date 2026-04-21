# 实施计划：OpenDray 插件平台 M3 — 宿主 Sidecar 运行时

> 输出文件：`/home/linivek/workspace/opendray/docs/plugin-platform/M3-PLAN.md`
> 设计合约：`/home/linivek/workspace/opendray/docs/plugin-platform/12-roadmap.md` §M3
> 前置任务：M2 已在 `kevlab` 分支发布 — 23/26 个任务已完成（见 `docs/plugin-platform/M2-RELEASE.md`）；T16b 桌面端 WebView、T23 E2E 看板、T25 CSP 测试作为遗留任务转入 M3。
> 北极星指标：`rust-analyzer-od`（或最小化的 `fs-readme` 参考插件）为 `opendray.fs.read` 提供的 Rust 文件提供 LSP 补全；在请求中途杀掉 sidecar 会返回干净的 `EUNAVAIL`，且管理器在其退避窗口内重启它。每次特权调用都通过声明的路径/命令/URL/命名空间匹配通过能力门控 — 无路径穿越，无 SSRF，无 fork 炸弹。

---

## 1. 范围边界

**包含（M3 合约）：**

- **`opendray.fs.*`** — `readFile`, `writeFile`, `exists`, `stat`, `readDir`, `mkdir`, `remove`, `watch`。所有 I/O 均在宿主侧执行；插件永远不持有文件描述符。路径在规范化后，与 `permissions.fs.{read|write}` 中声明的 glob 允许列表进行匹配，并支持 `${workspace}`, `${home}`, `${dataDir}`, `${tmp}` 基础变量扩展（与 05-capabilities.md §路径模式一致）。软限制：每次读取 10 MiB，每次写入 10 MiB，每次 `readDir` 4096 个条目。`watch` 通过 T11 使用的相同 chunk/end 信封返回订阅流 ID。
- **`opendray.exec.*`** — `run`（单次），`spawn`（流式），`kill`, `wait`。命令行允许列表通过现有的 `bridge.MatchExecGlobs` 进行匹配。默认每次 spawn 超时 10 秒（可通过 `opts.timeoutMs` 覆盖，最高 5 分钟）。合并 stdout+stderr 流块信封，每个块限制 16 KiB（镜像现有的 `commands.Dispatcher.truncate` 常量，但采用增量流式处理而非截断）。Linux：在挂载 `cgroup v2` 时为每个进程提供 cgroup（CPU + 内存软限制）— macOS 和 Windows 上的文档差异。所有平台：`Setpgid=true`，以便管理器在撤销/断开连接时可以杀死整个进程组。
- **`opendray.http.*`** — `request`, `stream`。URL 模式允许列表通过现有的 `bridge.MatchHTTPURL`（RFC1918 / 回环 / 链路本地拒绝策略保持不变）。请求体限制 4 MiB，响应体限制 16 MiB，响应流使用 `{stream:"chunk"}` 信封。重定向限制为 5 次，且仅跳转至仍匹配插件允许列表的 URL。TLS：使用 Go 默认的 `crypto/tls` 验证。
- **`opendray.secret.*`** — `get`, `set`, `delete`。值在现有的 `plugin_secret` 表中以 AES-GCM 加密存储。密钥加密密钥 (KEK) 通过 HKDF-SHA256 从宿主凭据存储主密钥派生（与目前保护管理员 bcrypt 行的材料相同 — 见 `kernel/auth/credentials.go`）；数据加密密钥 (DEK) 是 32 字节随机字节，由 KEK 包装并存储在每个插件的新 `plugin_secret_kek` 行中。严禁记录明文。严禁返回其他插件的密钥。
- **宿主 sidecar 管理器** — 新增 `plugin/host/supervisor.go`：根据需求为每个 `form:"host"` 插件启动一个 sidecar 子进程，通过管道传输标准输入输出，使用 LSP 的 `Content-Length: N\r\n\r\n<json>` 报文头封装 JSON-RPC 2.0，在崩溃时以指数退避重启（200 ms → 5 s，封顶），并在无请求 `IdleShutdownMinutes`（默认 10 分钟）后关闭 sidecar。每个插件一个 sidecar。管理器在 iOS 上通过特性标志关闭 — `form:"host"` 插件拒绝在 iOS 构建版本上安装（根据 10-security.md iOS 策略）。
- **`form:"host"` 清单接线** — 在 `Provider` 上激活已保留的 `HostV1` 结构（`entry`, `runtime ∈ {binary,node,deno,python3,bun,custom}`, `platforms`, `protocol ∈ {jsonrpc-stdio}`, `restart`, `env`, `cwd`）。验证器拒绝 iOS 上的宿主插件；安装器拒绝缺失或不可执行 `entry` 的包。
- **`contributes.languageServers`** — 新的贡献点。网关暴露 `GET /api/plugins/lsp/{plugin}/{language}/proxy` (WebSocket)，将编辑器的 LSP 流量通过隧道传输到 sidecar 的 JSON-RPC 流。服务端去重：每个语言一个 LSP，按安装顺序尝试插件（第一个响应 `initialize` 的获胜）。
- **能力门控增加** — 已实现的匹配器（`MatchExecGlobs`, `MatchHTTPURL`, `MatchFSPath`）获得一个薄封装，在匹配前扩展基础变量。新的匹配器 `MatchSecretNamespace(plugin, key string)` 强制要求 `key` 不包含 `/` 或 `..`，并通过 SQL 行所有权隐式限定在插件范围内。
- **许可 UI 增加** — Flutter 设置 → 插件页面（已在 M2 T21 上线）中的每个能力粒度开关。新开关：每个文件系统路径 glob（每个声明的条目开/关）、每个执行模式、每个 HTTP URL glob。切换开关会通过新端点 `PATCH /api/plugins/{name}/consents` 翻转 `perms_json`，该端点接受部分 `PermissionsV1` 补丁；`bridgeMgr.InvalidateConsent` 会在每个受影响的能力上触发，以便活动流在 M2 建立的 200 ms SLO 内终止。
- **DB 迁移 014–017** — 见 §7。
- **参考插件** — `plugins/examples/fs-readme/`（最小化的宿主形态插件：读取 `${workspace}/README.md`，通过一个微型 sidecar Node 脚本进行总结，通过 `opendray.workbench.showMessage` 返回预览）。练习 fs.read + exec (sidecar spawn) + host-form manifest。`rust-analyzer-od` 是一个挑战目标 — 验收门槛是较小的 `fs-readme` 插件。

**排除 — 为了保持 M3 预算而枚举推迟的内容：**

| 诱人的功能蔓延 | 推迟至 |
|---|---|
| 市场客户端, `plugin/market/`, `opendray plugin publish`, 签名验证, 撤销列表轮询 | **M4** |
| 用于 `render:"declarative"` 视图的声明式组件树渲染器 | **M5** |
| Node.js / Deno / Python 运行时安装程序 — M3 假设运行时二进制文件已在 PATH 中，如果不在则干净地失败（返回带有带有人类可读消息的 `EUNAVAIL`） | **M6 / v1 后** |
| 插件作者的热重载 (`opendray plugin dev`), bridge 追踪工具, 便携式 `opendray-dev` | **M6** |
| 多进程插件（每个插件超过一个 sidecar） | **v1 后** |
| 完整的 seccomp 过滤器集 — M3 交付最小可行方案：在 `CAP_SYS_ADMIN` 可用时（通常不可用 — 文档差异）使用 Linux `unshare(CLONE_NEWNET)` 网络命名空间隔离，以及用于干净终止的 `Setpgid=true`。任何更高级的功能都在 v1 之后。 | **v1 后** |
| `opendray.session.*`, `opendray.commands.execute` 跨插件, `opendray.tasks.*`, `opendray.clipboard.*`, `opendray.llm.*`, `opendray.git.*`, `opendray.telegram.*`, `opendray.logger.*` | **M5** (受现有 HTTP API 门控；接线仅涉及接线) |
| Sidecar 发起的 bridge 调用 (sidecar → host `workbench.showMessage`) | **M3 子范围** — 由 T12 中的双向 JSON-RPC 覆盖，但明确排除 sidecar 的 `fs.*`/`exec.*`/`http.*`/`secret.*` 调用（sidecar 已经拥有这些资源的宿主级访问权限；它们不需要 bridge 门控）。 |
| 用于 bridge WS 源的 `WKProcessPool` / Android data-dir 后缀加固（已在 M2 交付；M3 保持相同策略） | — |
| 有线格式升级 — M3 保持在 `V=1` | — |
| CSP 黄金文件测试（M2 T25 遗留任务） — 在 T27 中交付 | — |
| 桌面端内联 WebView（M2 T16b 遗留任务） | **M6** |

---

## 2. 任务图

> **约定：** 每个任务都有 ID T#，依赖列表，要创建/修改的文件（绝对路径），核心类型/签名，验收标准，测试，复杂度 S/M/L，风险。除非另有说明，文件路径均在 `/home/linivek/workspace/opendray/` 下。

### T1 — 在 `Provider` 和验证器上激活 `HostV1`
- **依赖：** 无
- **修改：** `/home/linivek/workspace/opendray/plugin/manifest.go` — 向 `Provider` 添加 `Host *HostV1 \`json:"host,omitempty"\``；添加镜像 `02-manifest.md` §host 模式的 `HostV1` 结构：`Entry string`, `Runtime string`（枚举 `binary|node|deno|python3|bun|custom`，默认 `binary`），`Platforms map[string]string`（键 `^(linux|darwin|windows)-(x64|arm64)$`），`Protocol string`（枚举 `jsonrpc-stdio`），`Restart string`（枚举 `on-failure|always|never`，默认 `on-failure`），`Env map[string]string`, `Cwd string`, `IdleShutdownMinutes int`（默认 10）。`/home/linivek/workspace/opendray/plugin/manifest_validate.go` — 添加由 `validateContributes` 在 `p.EffectiveForm() == FormHost` 时调用的 `validateHostV1(p Provider)`：entry 必填且无 `..`，runtime 在枚举中，protocol 必须为 `jsonrpc-stdio`，restart 在枚举中，platform 键匹配正则，env 键正则为 `^[A-Z_][A-Z0-9_]*$`。iOS 构建标签：`validateHostV1` 在 iOS 上无条件返回错误（见 T2 的构建标签接线）。
- **验收：** `go build ./...` 干净通过。每个现有清单继续解析。手动创建的带有 `runtime:"node"`, `entry:"sidecar/index.js"` 的宿主形态清单在桌面端解析 + 验证通过。相同的清单在 `//go:build ios` 构建上验证失败，并显示错误 `contributes.host: host-form plugins are not supported on iOS`。
- **测试：** 扩展 `/home/linivek/workspace/opendray/plugin/manifest_v1_test.go` → `TestLoadManifest_V1Host` 黄金文件；扩展 `/home/linivek/workspace/opendray/plugin/manifest_validate_test.go` → 8 个无效案例 + 2 个有效案例。
- **复杂度：** S
- **风险/缓解：** 低。增量式 + 验证器。风险：遗忘 iOS 构建标签 — 缓解：显式的 `ios_test.go` 构建标签门控测试。

### T2 — iOS 宿主形态拒绝构建标签
- **依赖：** T1
- **创建：** `/home/linivek/workspace/opendray/plugin/host_os_ios.go` (`//go:build ios`) — 导出常量 `HostFormAllowed = false`；`/home/linivek/workspace/opendray/plugin/host_os_other.go` (`//go:build !ios`) — `HostFormAllowed = true`。
- **修改：** `/home/linivek/workspace/opendray/plugin/manifest_validate.go` — `validateHostV1` 在 `!HostFormAllowed` 时直接报错。`/home/linivek/workspace/opendray/plugin/install/install.go` — `Installer.Stage` 在 `!HostFormAllowed` 时拒绝 `form:"host"` 的包，并返回一个新的哨兵错误 `ErrHostFormNotSupported`。
- **验收：** 使用 `GOOS=ios`（或现有的 iOS 构建路径）进行构建会使任何宿主形态安装失败并显示哨兵错误。桌面/Android 保持正常工作。
- **测试：** `/home/linivek/workspace/opendray/plugin/install/install_ios_test.go`（构建标签 `//go:build ios`）断言 Stage 拒绝；`/home/linivek/workspace/opendray/plugin/install/install_desktop_test.go`（构建标签 `//go:build !ios`) 断言 Stage 接受。
- **复杂度：** S
- **风险/缓解：** 低。隔离在清单 + 安装器路径。

### T3 — 能力门控基础变量扩展
- **依赖：** 无（与 T1/T2 并行）
- **修改：** `/home/linivek/workspace/opendray/plugin/bridge/capabilities.go` — 添加一个新的纯函数 `ExpandPathVars(pattern string, ctx PathVarCtx) string`，其中 `PathVarCtx = struct{ Workspace, Home, DataDir, Tmp string }`。在 `MatchFSPath` 运行之前，在声明的模式中替换 `${workspace}`, `${home}`, `${dataDir}`, `${tmp}`。`Gate.Check` 获得 `WithPathVars(ctx PathVarCtx) *Gate` 函数选项（或新的 `CheckExpanded(ctx, plugin, need, vars PathVarCtx)` 方法 — 根据 preferences.md 选择后者以避免改变不可变的 Gate）。
- **验收：** 当 `perms.fs.read = ["${workspace}/**"]` 时，`Gate.CheckExpanded(ctx, "fs-readme", Need{Cap:"fs.read", Target:"/home/kev/proj/README.md"}, PathVarCtx{Workspace:"/home/kev/proj", Home:"/home/kev"})` 返回 `nil`。使用 `Target:"/etc/passwd"` 进行相同的调用返回 `PermError`。路径穿越技巧（例如 `${workspace}/../../etc/passwd`）失败：`MatchFSPath` 已通过 `filepath.Clean` 进行规范化；T3 增加了 `TestExpandPathVars_TraversalBlocked` 回归测试。
- **测试：** `/home/linivek/workspace/opendray/plugin/bridge/capabilities_expand_test.go` — 12 个案例（干净、穿越、未知变量、空变量、嵌套 `${}`）。
- **复杂度：** S
- **风险/缓解：** 中 — 未知的 `${var}` 目前具有二义性。决定：`ExpandPathVars` 保持未知变量字面量（`${unknown}` 保持原样，然后 `MatchFSPath` 匹配失败 — 安全默认值）。引用 05-capabilities.md §路径模式。

### T4 — DB 迁移 014: plugin_secret_kek
- **依赖：** 无
- **创建：** `/home/linivek/workspace/opendray/kernel/store/migrations/014_plugin_secret_kek.sql`, `/home/linivek/workspace/opendray/kernel/store/migrations/014_plugin_secret_kek_down.sql`
- **模式 (014 up):**
  ```sql
  CREATE TABLE IF NOT EXISTS plugin_secret_kek (
      plugin_name  TEXT PRIMARY KEY REFERENCES plugins(name) ON DELETE CASCADE,
      wrapped_dek  BYTEA NOT NULL,           -- 在宿主 KEK 下加密的 32 字节 DEK
      kek_kid      TEXT NOT NULL,            -- 用于轮换的 KEK 密钥 ID
      created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
      updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
  );
  ```
- **验收：** `embedded-postgres` 测试套件干净地应用迁移。回滚迁移删除该表。
- **测试：** `/home/linivek/workspace/opendray/kernel/store/migrations_test.go` 在现有的迁移往返测试中获得一个额外案例。
- **复杂度：** S
- **风险/缓解：** 低。

### T5 — DB 迁移 015: plugin_host_state
- **依赖：** 无（与 T4 并行）
- **创建：** `/home/linivek/workspace/opendray/kernel/store/migrations/015_plugin_host_state.sql`, `/home/linivek/workspace/opendray/kernel/store/migrations/015_plugin_host_state_down.sql`
- **模式：**
  ```sql
  CREATE TABLE IF NOT EXISTS plugin_host_state (
      plugin_name     TEXT PRIMARY KEY REFERENCES plugins(name) ON DELETE CASCADE,
      last_started_at TIMESTAMPTZ,
      last_exit_code  INTEGER,
      restart_count   INTEGER NOT NULL DEFAULT 0,
      last_error      TEXT
  );
  ```
- **验收：** 迁移往返。管理器在这里持久化生命周期统计数据，以便设置 UI 可以显示“最近一小时内重启 3 次”。
- **复杂度：** S

### T6 — DB 迁移 016: plugin_secret 加密升级
- **依赖：** T4
- **修改：** 无（现有的 `012_plugin_secret.sql` 已具有 `ciphertext BYTEA`；M3 只是开始正确使用它）。`/home/linivek/workspace/opendray/kernel/store/migrations/016_plugin_secret_nonce.sql` — ALTER TABLE 添加 `nonce BYTEA NOT NULL DEFAULT ''::bytea` 列（AES-GCM nonce，12 字节）。回滚迁移删除该列。
- **验收：** 旧的密钥行（目前没有 — M1 脚手架，M2 未使用）在第一次读取时通过 `kernel/store/plugin_secret.go` (T15) 中的单次迁移器重新加密。空的默认 `nonce` 识别 M3 之前的行。
- **复杂度：** S

### T7 — 密钥加密原语：KEK / DEK 辅助工具
- **依赖：** T4
- **创建：** `/home/linivek/workspace/opendray/kernel/auth/secret_kek.go`, `/home/linivek/workspace/opendray/kernel/auth/secret_kek_test.go`
- **核心类型/函数：**
  ```go
  // KEKProvider 在每次调用时派生宿主 KEK。保持基于派生
  // 意味着 KEK 永远不会持久化 — 只有派生它的 bcrypt 管理员哈希会被持久化，
  // 该哈希已经存在于 admin_auth 中。
  type KEKProvider interface {
      DeriveKEK(ctx context.Context, kid string) ([]byte, error) // 32 字节
  }

  // NewKEKProviderFromAdminAuth 连线一个通过 HKDF-SHA256 
  // 配合 info="opendray-plugin-kek/<kid>" 从 admin_auth 行派生 32 字节 KEK 的提供者。
  func NewKEKProviderFromAdminAuth(store *CredentialStore) KEKProvider

  // WrapDEK 使用 AES-256-GCM 在 kek 下加密 32 字节 DEK。返回
  // 连接后的 nonce||ciphertext (nonce 为 12 字节)。如果 dek 不是 
  // 32 字节或 kek 不是 32 字节则抛出 panic。
  func WrapDEK(kek, dek []byte) (wrapped []byte, err error)

  // UnwrapDEK 是逆过程。返回 32 字节 DEK 或错误。
  func UnwrapDEK(kek, wrapped []byte) (dek []byte, err error)
  ```
- **验收：** 1000 个随机 DEK 的 `Wrap`/`Unwrap` 往返均成功。篡改任何密文字节都会导致 GCM 标签验证失败。KEK 轮换：使用旧 `kid` 解包，使用新 `kid` 重新包装 — 专用测试 `TestKEKRotation_RewrapSucceeds` 断言此点。
- **测试：** 表格驱动，包括错误大小密钥的负面测试 + 包装格式的模糊测试语料库。
- **复杂度：** M
- **风险/缓解：** **高** — 密钥泄露是不可恢复的事件。缓解措施：(a) KEK 材料永远不离开 `crypto/subtle` 可比较的字节切片；(b) `secret_kek.go` 中没有包含任何密钥材料的日志行；(c) `gosec ./...` 在 CI 中为此包运行，零 HIGH 发现作为合并门槛；(d) 在 10-security.md 中记录“轮换管理员密码会轮换 KEK — 宿主在登录时遍历每个 `plugin_secret_kek` 行并使用新 KEK 重新包装”。

### T8 — 为流增加 Bridge 命名空间签名
- **依赖：** 无（并行）
- **修改：** `/home/linivek/workspace/opendray/gateway/plugins_bridge.go` — 对 `Namespace` 接口本身没有更改（自 T20b 以来它已接受 `envID` + `*bridge.Conn`）。文档化流帧合约：命名空间通过 `conn.WriteEnvelope(bridge.NewStreamChunk(envID, data))` 和 `conn.WriteEnvelope(bridge.NewStreamEnd(envID))` 发送 chunk/end。在 `/home/linivek/workspace/opendray/plugin/bridge/protocol.go` 中为流内终止错误案例（执行中途非零退出、HTTP 流截断、fs.watch 错误）添加辅助函数 `bridge.NewStreamChunkErr(id string, we *WireError)`。
- **验收：** 现有测试继续通过；新辅助函数通过 JSON 往返。
- **测试：** 使用 `TestNewStreamChunkErr_EnvelopeShape` 扩展 `/home/linivek/workspace/opendray/plugin/bridge/protocol_test.go`。
- **复杂度：** S

### T9 — `opendray.fs.*` 命名空间 (读取路径)
- **依赖：** T3, T8
- **创建：** `/home/linivek/workspace/opendray/plugin/bridge/api_fs.go`, `/home/linivek/workspace/opendray/plugin/bridge/api_fs_test.go`
- **核心类型/函数：**
  ```go
  // FSConfig 连接 FS 命名空间所需的依赖项：用于
  // 能力检查的 Gate，用于每次调用会话上下文的 PathVarResolver
  // (工作区根目录等)，以及用于测试可注入 watch 去抖动的 Clock。
  type FSConfig struct {
      Gate     *Gate
      Resolver PathVarResolver
      Log      *slog.Logger
  }

  // PathVarResolver 在调用时返回给定插件的当前 {workspace, home, dataDir, tmp}。
  // 由网关实现 — 从活动会话解析 workspace，从 ${PluginsDataDir}/<name>/<version>/data/ 解析 dataDir。
  type PathVarResolver interface {
      Resolve(ctx context.Context, plugin string) (PathVarCtx, error)
  }

  type FSAPI struct { /* 未导出 */ }
  func NewFSAPI(cfg FSConfig) *FSAPI
  // Dispatch 实现 gateway.Namespace。
  func (a *FSAPI) Dispatch(ctx context.Context, plugin, method string, args json.RawMessage, envID string, conn *Conn) (any, error)
  // 此处实现的方法 (仅限读取路径):
  //   readFile(path, opts?{encoding}) → string (utf8) | base64 string
  //   exists(path) → bool
  //   stat(path) → {size, mtime, isDir}
  //   readDir(path) → [{name, isDir}]
  ```
- **能力强制执行：** 每个方法解析路径变量，清理路径，然后调用 `Gate.Check(ctx, plugin, Need{Cap:"fs.read", Target: cleanedAbsolute})`。读取容量 = 10 MiB 硬限制；`readDir` 容量 = 4096 个条目。
- **验收：** 具有 `permissions.fs.read=["${workspace}/**"]` 的插件可以 `readFile("${workspace}/README.md")`（服务器在匹配前扩展变量）。读取 `/etc/passwd` → `EPERM`。读取大于 10 MiB 的文件 → `EINVAL {message:"fs.readFile: file exceeds 10 MiB cap"}`。
- **测试：** 表格驱动，包括来自 M2 T8 攻击列表的穿越尝试以及 unicode 规范化尝试 (`README\u202e.md`)。使用 `t.TempDir()` 作为合成工作区。
- **复杂度：** M
- **风险/缓解：** **高 — TOCTOU**。缓解措施：每次调用在最终 `Open` 之前通过 `filepath.EvalSymlinks` 重新规范化路径；生成的已解析路径会针对授权 glob 重新检查。逃离工作区的符号链接将无法通过第二次检查。

### T10 — `opendray.fs.*` 命名空间 (写入路径 + watch)
- **依赖：** T9
- **修改：** `/home/linivek/workspace/opendray/plugin/bridge/api_fs.go` — 添加 `writeFile`, `mkdir`, `remove`, `watch` 方法。`watch` 具备流式能力：启动一个 `fsnotify.Watcher`，通过 `conn.Subscribe(envID, "fs.watch")` 注册订阅，推送 `{create|modify|delete, path}` 块。通过 `fs.unwatch{subId}` 或撤销来取消订阅。
- **依赖：** 将 `github.com/fsnotify/fsnotify v1.8.0` 添加到 `go.mod`（固定；如果任务运行器插件存在，则已经在模块缓存中 — 请验证）。
- **能力：** `fs.write`；`watch` 需要 `fs.read`（订阅会看到它本可以读取的文件的事件）。
- **验收：** writeFile 以 0644 模式创建文件（可通过 `opts.mode` 覆盖）；`remove{recursive:true}` 拒绝授权之外的路径；`watch` 在 `t.TempDir` 中外部文件更改后的 500 ms 内交付 3 个事件。
- **测试：** 增加 `TestFSWatch_DisposesOnRevoke`（M2 T11 模式）。
- **复杂度：** L
- **风险/缓解：** 高 — fsnotify 具有平台差异（Linux inotify 限制）。缓解措施：每个插件最多 16 个活动监听器；超过则返回 `EINVAL`。在达到 inotify 限制时快速失败，并提供清晰的消息指向 `sysctl fs.inotify.max_user_watches`。

### T11 — `opendray.exec.*` 命名空间
- **依赖：** T3, T8
- **创建：** `/home/linivek/workspace/opendray/plugin/bridge/api_exec.go`, `/home/linivek/workspace/opendray/plugin/bridge/api_exec_test.go`
- **核心类型/函数：**
  ```go
  type ExecConfig struct {
      Gate      *Gate
      Resolver  PathVarResolver
      Log       *slog.Logger
      MaxTimeout time.Duration // 默认 5 分钟; 每次调用的 opts.timeoutMs 会被夹持
  }
  type ExecAPI struct { /* 未导出 */ }
  func NewExecAPI(cfg ExecConfig) *ExecAPI
  // 方法: run(cmd, args, opts?) → {exitCode, stdout, stderr, timedOut}
  //          spawn(cmd, args, opts?) → {pid, subId}  + 流式块
  //          write(subId, input)     → null
  //          kill(subId, signal?)    → null
  //          wait(subId)             → {exitCode}
  ```
- **能力强制执行：** 在每次 `run`/`spawn` 时，计算 `cmdline = cmd + " " + strings.Join(args, " ")`，调用 `Gate.Check(ctx, plugin, Need{Cap:"exec", Target: cmdline})`。如果 `opts.cwd` 在插件声明的 fs.read/fs.write 授权之外，则拒绝（通过 `ExecConfig.AllowCwdOutsideFS` 配置，默认 false）。
- **进程属性：** `Setpgid=true`。Linux 额外调用 `syscall.Unshare(syscall.CLONE_NEWNET)`，**仅当** `ExecConfig.IsolateNetNS=true` 且 opendray 持有 `cap_sys_admin` 时 — 否则在管理器启动时记录一次警告并跳过。在 10-security.md 中记录此差异。
- **流式容量：** 每个块 ≤16 KiB；输出截断遵循现有的 `commands.Dispatcher` 语义，但会将 `{error:{code:"EINVAL",message:"output truncated"}}` 信封作为软终止符暴露在流中途，而非静默截断。
- **验收：** 具有 `permissions.exec=["git *"]` 的插件执行 `run("git",["status","--short"])` 返回带有退出代码和捕获的 stdout 的结果。使用相同授权执行 `run("rm",["-rf","/"])` 返回 `EPERM`。具有 1 MiB stdout 的 `spawn` 以 ≤64 个块进行流式传输，没有缓冲爆炸。
- **测试：** 14 个案例，包括零退出成功、非零退出成功（通过 `Result.Exit` 冒泡）、超时终止、kill(SIGTERM) → 5 秒后升级为 SIGKILL、cwd 在授权之外。
- **复杂度：** L
- **风险/缓解：** **高 — fork 炸弹 / 资源耗尽。** 缓解措施：每个插件硬性限制 4 个并发 spawn（超出部分进入队列 → 带有 retryAfter 的 `ETIMEOUT`）；如果可写，Linux cgroup v2 `memory.max`=512 MiB, `cpu.max`=50%（如果不可写，每次宿主引导记录一次日志）。记录 macOS/Windows 没有 cgroup 等效项，依赖 OS ulimit。

### T12 — `opendray.http.*` 命名空间
- **依赖：** T3, T8
- **创建：** `/home/linivek/workspace/opendray/plugin/bridge/api_http.go`, `/home/linivek/workspace/opendray/plugin/bridge/api_http_test.go`
- **核心类型/函数：**
  ```go
  type HTTPConfig struct {
      Gate            *Gate
      Log             *slog.Logger
      MaxRequestBody  int64 // 默认 4 MiB
      MaxResponseBody int64 // 默认 16 MiB
      MaxRedirects    int   // 默认 5
      DialTimeout     time.Duration // 默认 10 s
      TotalTimeout    time.Duration // 默认 60 s
      TLSMinVersion   uint16 // 默认 tls.VersionTLS12
  }
  type HTTPAPI struct { /* 未导出 */ }
  func NewHTTPAPI(cfg HTTPConfig) *HTTPAPI
  // 方法: request(req) → {status, headers, body(base64)}
  //          stream(req)  → base64 正文切片的块信封; 关闭时结束
  ```
- **能力强制执行：** `Gate.Check(ctx, plugin, Need{Cap:"http", Target: req.URL})`。现有的 `MatchHTTPURL` 已拒绝 RFC1918 / 回环 / 链路本地。重定向跳转：检查每一跳（不仅是初始 URL）— 文件中新增辅助函数 `followRedirectsWithGate`；跳转失败 `Gate.Check` 会中断链条，返回 `EPERM` 并附带命名跳转 URL 的消息。
- **正文限制：** 如果 `len(body) > MaxRequestBody` 则拒绝请求体。响应体在 `MaxResponseBody` 处截断，流路径上带有结尾的 `{error:{code:"EINVAL",message:"response body truncated"}}` 信封（非流路径返回截断的正文并在 `result.truncated=true` 中带上相同代码）。
- **验收：** 具有 `permissions.http=["https://api.github.com/*"]` 的插件可以执行 `request({url:"https://api.github.com/repos/opendray/opendray"})`。对 `https://169.254.169.254` (AWS IMDS) 的请求即使某种程度上模式匹配也会被拒绝（匹配器已拦截）。从 `api.github.com` 到 `malicious.com` 的重定向在跳转时被 EPERM 拦截。
- **测试：** 包括一个返回重定向并断言门控在链条中途拦截的 `httptest.Server` 链。
- **复杂度：** L
- **风险/缓解：** **高 — 通过 DNS 重绑定的 SSRF。** 缓解措施：dial 使用自定义 `net.Dialer.Control`，在 `connect()` 之前再次通过 `isPrivateHost` 检查**已解析的 IP** — 拦截 DNS 重绑定以及通过指定解析为私有的 IP 字面量进行的绕过。专用测试 `TestHTTP_SSRF_DNSRebind` 使用在第二次查找时返回 `10.0.0.1` 的自定义解析器。

### T13 — `opendray.secret.*` 命名空间
- **依赖：** T6, T7
- **创建：** `/home/linivek/workspace/opendray/plugin/bridge/api_secret.go`, `/home/linivek/workspace/opendray/plugin/bridge/api_secret_test.go`; `/home/linivek/workspace/opendray/kernel/store/plugin_secret.go`, `/home/linivek/workspace/opendray/kernel/store/plugin_secret_test.go`
- **核心类型/函数：**
  ```go
  // 存储层.
  func (d *DB) SecretGet(ctx, plugin, key string) (string, bool, error)
  func (d *DB) SecretSet(ctx, plugin, key, value string, wrap func([]byte) ([]byte, []byte, error)) error
  // wrap 返回 (ciphertext, nonce, err); DB 层保持密钥不可知.
  func (d *DB) SecretDelete(ctx, plugin, key string) error
  func (d *DB) SecretList(ctx, plugin string) ([]string, error) // 仅键 — 绝不包含值

  // DEK 管理.
  func (d *DB) EnsureKEKRow(ctx, plugin string, wrappedDEK []byte, kid string) error
  func (d *DB) GetWrappedDEK(ctx, plugin string) (wrapped []byte, kid string, err error)

  // Bridge 层.
  type SecretAPI struct{ /* 未导出 */ }
  type SecretConfig struct {
      DB  *store.DB
      Gate *Gate
      KEK  auth.KEKProvider
  }
  func NewSecretAPI(cfg SecretConfig) *SecretAPI
  ```
- **生命周期：** 插件的第一次 `SecretSet` 为一个插件生成一个随机 32 字节 DEK，通过 `KEKProvider.DeriveKEK` → `WrapDEK` 进行包装，并调用 `EnsureKEKRow`。后续的 set 复用相同的 DEK。卸载时通过 FK 级联删除 → `plugin_secret_kek` 和 `plugin_secret` 行均被删除。
- **能力：** `secret:true`。键验证：`MatchSecretNamespace(key)` — 正则 `^[a-zA-Z0-9._-]{1,128}$`，无 `/`，无 `..`。
- **验收：** 1 KiB 密钥的往返。对不存在的键执行 `get` 返回 `null`（结果），而非错误。另一个插件对相同键执行 `get` 返回 `null`（密钥按插件行命名空间隔离 — 一个插件看不到另一个插件的密钥）。
- **测试：** 10-security.md 中的日志脱敏规则通过 `TestSecret_NeverLogged` 强制执行，该测试捕获 set/get 期间的 slog 输出并断言密钥值不存在。
- **复杂度：** L
- **风险/缓解：** **关键。** 见 §6 威胁表。所有触及加密的代码均通过 `gosec ./...` 运行，零 HIGH 发现作为合并门槛。

### T14 — 宿主 sidecar 管理器骨架
- **依赖：** T1, T2, T5
- **创建：** `/home/linivek/workspace/opendray/plugin/host/supervisor.go`, `/home/linivek/workspace/opendray/plugin/host/supervisor_test.go`, `/home/linivek/workspace/opendray/plugin/host/process.go`, `/home/linivek/workspace/opendray/plugin/host/process_unix.go` (`//go:build unix`), `/home/linivek/workspace/opendray/plugin/host/process_windows.go` (`//go:build windows`)
- **核心类型/函数：**
  ```go
  // Supervisor 拥有每个宿主形态插件 sidecar 的生命周期。
  type Supervisor struct { /* 未导出 */ }
  type Config struct {
      DataDir           string              // ${PluginsDataDir}
      Runtime           *plugin.Runtime     // 用于解析 Host 规范
      State             *store.DB           // plugin_host_state 写入器
      Log               *slog.Logger
      IdleShutdown      time.Duration       // 默认 10 min
      MaxRestartBackoff time.Duration       // 默认 5 s
      InitialBackoff    time.Duration       // 默认 200 ms
  }
  func NewSupervisor(cfg Config) *Supervisor
  // Ensure 如果 sidecar 未运行则启动，否则更新 lastUsedAt。
  // 返回 *Sidecar 句柄。并发调用者共享每个插件一个 sidecar。
  func (s *Supervisor) Ensure(ctx context.Context, plugin string) (*Sidecar, error)
  // Kill 终止 sidecar + 取消任何进行中的请求。
  func (s *Supervisor) Kill(plugin string, reason string) error
  // Stop 停止所有 sidecar (优雅停止)。在宿主关闭时调用。
  func (s *Supervisor) Stop(ctx context.Context) error

  // Sidecar 暴露 JSON-RPC 写入 + 读取通道。
  type Sidecar struct { /* 未导出 */ }
  func (s *Sidecar) Call(ctx context.Context, method string, params any) (json.RawMessage, error)
  func (s *Sidecar) Notify(method string, params any) error
  // Subscribe 返回服务器推送通知的通道 (LSP $/progress 等)
  func (s *Sidecar) Subscribe() <-chan Notification
  ```
- **进程启动：** 查找 `Provider.Host` (T1)；对于 `runtime="node"` 执行 `node <entry>`；对于 `binary` 使用 `Host.Platforms[runtime.GOOS+"-"+runtime.GOARCH]`；`Setpgid=true`；管道传输 stdin/stdout/stderr。Stderr 排入一个环形缓冲区（最后 64 KiB），将来通过设置 → 日志 UI 暴露。
- **退避：** 在崩溃时，等待 `min(initial*2^crashes, max)`；在 5 分钟稳定运行时间后重置计数器。`restart:"never"` 禁用重启；`restart:"always"` 忽略退出代码。
- **空闲关闭：** 计时器检查 `time.Since(lastUsedAt)`；如果超过配置，发送 `shutdown` JSON-RPC 通知，等待 2 秒，然后发送 `SIGTERM`，5 秒后再发送 `SIGKILL`。
- **验收：** 在一个固定的 node 脚本上执行 `Ensure` 会启动进程，在 < 200 ms 内响应 `ping` JSON-RPC 为 `pong`，在配置的超时后进入空闲，在退避窗口内从模拟崩溃中恢复。
- **测试：** 固定脚本是一个 40 行的 `sidecar.js`，在 stdin 上循环，对 `{method:"ping"}` 进行回复。Windows 路径使用通过 `exec.LookPath` 的 `node.exe` 检测。
- **复杂度：** L
- **风险/缓解：** **高 — 僵尸进程，stdin/stdout 关闭时的死锁。** 缓解措施：每个 goroutine 都绑定到一个 `context.Context`；sidecar 关闭通过该 context 使用 `sync.WaitGroup` 合并和 10 秒硬性期限进行扇出。`-race` 测试 `TestSupervisor_KillUnderLoad` 启动 50 个并发 `Ensure` 调用 + 循环中的 `Kill`，进行 100 次迭代。

### T15 — JSON-RPC LSP 报文编解码器
- **依赖：** 无（与 T14 并行）
- **创建：** `/home/linivek/workspace/opendray/plugin/host/jsonrpc.go`, `/home/linivek/workspace/opendray/plugin/host/jsonrpc_test.go`
- **核心类型/函数：**
  ```go
  // FramedReader 每次调用读取一条 LSP 报文封装的 JSON-RPC 消息。封装格式为:
  //   Content-Length: N\r\n
  //   (可选的其他 header, 忽略)
  //   \r\n
  //   <N 字节的 JSON>
  type FramedReader struct { /* 未导出 */ }
  func NewFramedReader(r io.Reader) *FramedReader
  func (r *FramedReader) Read() (json.RawMessage, error)

  type FramedWriter struct { /* 未导出 */ }
  func NewFramedWriter(w io.Writer) *FramedWriter
  // Write 将 msg 编码为 JSON, 写入 Content-Length header, 然后写入 payload。
  // 通过内部 sync.Mutex 保证并发调用安全。
  func (w *FramedWriter) Write(msg any) error

  // RPC 是最小的 JSON-RPC 2.0 有线形状。
  type RPC struct {
      JSONRPC string          `json:"jsonrpc"` // 始终为 "2.0"
      ID      json.RawMessage `json:"id,omitempty"`
      Method  string          `json:"method,omitempty"`
      Params  json.RawMessage `json:"params,omitempty"`
      Result  json.RawMessage `json:"result,omitempty"`
      Error   *RPCError       `json:"error,omitempty"`
  }
  type RPCError struct {
      Code    int             `json:"code"`
      Message string          `json:"message"`
      Data    json.RawMessage `json:"data,omitempty"`
  }
  ```
- **验收：** LSP 3.17 规范 §基础协议中每个示例的往返编码/解码。格式错误的 Content-Length 报文头（负数、> 16 MiB、缺失 \r\n\r\n 终止符）返回命名错误；读取器通过跳到下一个有效报文头进行恢复。
- **测试：** 对 `FramedReader` 使用 10k 个随机字节序列进行模糊测试；无 panic，无死循环（报文头限制 8 KiB + 正文限制 16 MiB）。
- **复杂度：** M
- **风险/缓解：** 中 — 来自崩溃的 sidecar 的格式错误帧不得导致宿主拒绝服务。缓解措施：硬性限制正文大小为 16 MiB，溢出时返回 EINVAL。

### T16 — Sidecar 双向调用多路复用器
- **依赖：** T14, T15
- **创建：** `/home/linivek/workspace/opendray/plugin/host/mux.go`, `/home/linivek/workspace/opendray/plugin/host/mux_test.go`
- **核心类型/函数：**
  ```go
  // Mux 拥有单个 sidecar 的 JSON-RPC 流上的入站/出站去复用。
  // 出站请求获得一个 ID, 入站响应解析匹配的待处理调用。
  // 入站请求 (sidecar → host) 被交付给处理程序。
  // 入站通知进入 Subscribe()。
  type Mux struct { /* 未导出 */ }
  func NewMux(r io.Reader, w io.Writer, handler RPCHandler, log *slog.Logger) *Mux
  func (m *Mux) Start(ctx context.Context)
  func (m *Mux) Call(ctx context.Context, method string, params any) (json.RawMessage, error)
  func (m *Mux) Notify(method string, params any) error
  func (m *Mux) Notifications() <-chan Notification

  // RPCHandler 解析 sidecar → host 的 RPC 调用。
  // 对于 M3, 这被限制为几个众所周知的方法 (workbench/showMessage, fs/readFile);
  // 其他任何方法都返回 MethodNotFound。
  type RPCHandler interface {
      Handle(ctx context.Context, method string, params json.RawMessage) (any, error)
  }
  ```
- **ID 生成：** 从 1 开始的单调递增 int64；在 1 req/s 的速度下 5 年后在 math.MaxInt32 处翻转 — 实际上永远不会发生。
- **验收：** 在 `-race` 下，在将 `params` 回显为 `result` 的 sidecar 回环上进行的 1000 个并发 `Call` 均能正确解析。sidecar → host 的 `workbench.showMessage` 调用路由通过 `RPCHandler.Handle` 且响应返回给 sidecar。
- **测试：** 与在测试进程内部运行的存根 sidecar 配对 (`io.Pipe`)。
- **复杂度：** L
- **风险/缓解：** 高 — 翻转后的 ID 重用会破坏调用路由。缓解措施：检测映射冲突（ID 已在待处理中）并使旧调用失败，返回 `EINTERNAL`。

### T17 — 管理器 ↔ 命名空间接线 (Sidecar 后端能力)
- **依赖：** T14, T16; 设计上与 T9/T11/T12 解耦
- **创建：** `/home/linivek/workspace/opendray/plugin/host/host_rpc_handler.go`, `/home/linivek/workspace/opendray/plugin/host/host_rpc_handler_test.go`
- **核心类型/函数：**
  ```go
  // HostRPCHandler 是连接到每个 Mux 的 RPCHandler。它将 sidecar → host
  // 的调用通过 WebView 插件使用的相同 bridge.Namespace 界面进行路由,
  // 从而使能力强制执行成为单一代码路径。
  type HostRPCHandler struct { /* 未导出 */ }
  type HostRPCConfig struct {
      Namespaces map[string]bridge.Dispatcher // "fs", "exec", "http", "secret", "workbench", "storage", "events"
      Gate       *bridge.Gate
      Log        *slog.Logger
      Plugin     string // 所属插件名称; 每个 Mux 不可变
  }
  func NewHostRPCHandler(cfg HostRPCConfig) *HostRPCHandler
  // Handle 实现 RPCHandler; 将 "<ns>/<method>" 字符串映射到正确的
  // 调度器, 从构造函数中传递 Plugin, 以便 sidecar 无法伪装成其他插件。
  func (h *HostRPCHandler) Handle(ctx context.Context, method string, params json.RawMessage) (any, error)
  ```
- **合约：** sidecar 调用形式为 `{"method":"fs/readFile","params":["/abs/path"]}`。`method` 中的 `/` 将命名空间与方法分开；两者都通过 WebView 插件使用的相同 `Dispatcher.Dispatch`。能力检查运行方式相同。
- **验收：** 发送已授权路径的 `fs/readFile` 的宿主形态固定 sidecar 获得字节；发送未授权路径的 `fs/readFile` 获得一个 JSON-RPC 错误，`code=-32001` 且消息为 "EPERM: ..."。
- **测试：** 跨每个命名空间存根进行表格驱动测试。
- **复杂度：** M
- **风险/缓解：** 中 — 方法名注入。缓解措施：拒绝包含超过一个 `/` 或任何 `..`, `\x00` 的 `method`。测试 `TestHostRPC_MethodInjectionRejected`。

### T18 — `contributes.languageServers` 贡献 + 扁平化
- **依赖：** T1
- **修改：** `/home/linivek/workspace/opendray/plugin/manifest.go` — 向 `ContributesV1` 添加 `LanguageServers []LanguageServerV1 \`json:"languageServers,omitempty"\``；添加 `LanguageServerV1 struct { ID string; Languages []string; Transport string; InitializationOptions json.RawMessage }`。`/home/linivek/workspace/opendray/plugin/manifest_validate.go` — 验证 `transport="stdio"` (枚举), languages 不能为空, id 正则。`/home/linivek/workspace/opendray/plugin/contributions/registry.go` — 使用 `OwnedLanguageServer` 扩展 `FlatContributions`；添加 `func (r *Registry) LookupLanguageServer(language string) (OwnedLanguageServer, bool)` (按安装顺序第一个匹配)。
- **验收：** 注册一个带有 `contributes.languageServers=[{id:"rust",languages:["rust"],transport:"stdio"}]` 的插件使 `LookupLanguageServer("rust")` 返回它。第二个声明 `languages:["rust"]` 的插件被 Lookup 跳过。
- **测试：** 扩展具有安装顺序确定性的注册表测试。
- **复杂度：** S

### T19 — LSP 代理网关路由
- **依赖：** T14, T16, T17, T18
- **创建：** `/home/linivek/workspace/opendray/gateway/plugins_lsp.go`, `/home/linivek/workspace/opendray/gateway/plugins_lsp_test.go`
- **修改：** `/home/linivek/workspace/opendray/gateway/server.go` — 在受保护组中注册 `r.Get("/api/plugins/lsp/{language}/proxy", s.pluginsLSPProxy)`。
- **核心类型/函数：**
  ```go
  // pluginsLSPProxy 处理 WebSocket 升级, 其中编辑器隧道传输原始
  // LSP JSON-RPC 流量。代理:
  //   1. 通过 contributions.Registry.LookupLanguageServer 查找哪个插件拥有该语言。
  //   2. 如果 sidecar 未运行, 则请求 Supervisor.Ensure 启动。
  //   3. 在双向复制帧, 直到任意一方关闭。
  // 能力检查: 此路由需要一个新的内置能力 "lsp", 隐式授予任何
  // 带有 contributes.languageServers 的插件。除了安装授权屏幕外无需用户许可
  // (符合 contributes.commands 的逻辑)。
  func (s *Server) pluginsLSPProxy(w http.ResponseWriter, r *http.Request)
  ```
- **帧转换：** WS → LSP：网关读取 WS 二进制消息，期望它们已预先封装（编辑器按原样发送 Content-Length 报文头）。 LSP → WS：读取 FramedReader 输出并将每条消息包装为一条 WS 消息。最大消息大小匹配 bridge WS：每帧 1 MiB；超大触发代码为 1009 的优雅关闭。
- **验收：** `wscat` 连接，发送 `initialize` 请求，从 sidecar 获得 `initialize` 响应，然后 `textDocument/didOpen` 通知在往返中幸存。
- **测试：** 集成套件启动一个实现 `initialize` → 空能力响应的存根 sidecar。
- **复杂度：** L
- **风险/缓解：** 中 — 代理不得缓冲整个 LSP 会话。缓解措施：通过两个 goroutine + `io.Copy` 实现真实流式处理；有界块读取；显式关闭传播。

### T20 — 许可补丁端点 + 粒度 UI 接线
- **依赖：** M2 T12 (`UpdateConsentPerms` 已存在)
- **创建：** `/home/linivek/workspace/opendray/gateway/plugins_consents_patch.go`, `/home/linivek/workspace/opendray/gateway/plugins_consents_patch_test.go`
- **修改：** `/home/linivek/workspace/opendray/gateway/plugins_consents.go` — 添加路由 `PATCH /api/plugins/{name}/consents`，接受部分 `PermissionsV1` 形状的正文；合并到存储的 `perms_json` 中；按**差异化**能力调用 `bridgeMgr.InvalidateConsent`（而非按顶级键 — 从 `fs.read` 中删除一个 glob 的切换必须使 `fs.read` 订阅失效，即使顶级 `fs` 键保持不变）。
- **核心类型/函数：**
  ```go
  // ConsentPatch 是部分权限合并有效负载。Nil 字段保持不变; 
  // 零长度切片是显式清除。
  type ConsentPatch struct {
      Fs     *FSPermsPatch   `json:"fs,omitempty"`
      Exec   *[]string       `json:"exec,omitempty"`  // 切片指针: nil = 未变, &[] = 清除
      HTTP   *[]string       `json:"http,omitempty"`
      Secret *bool           `json:"secret,omitempty"`
      // Storage, Events, Session, Clipboard, Git, Telegram, LLM 补丁类似。
  }
  type FSPermsPatch struct {
      Read  *[]string `json:"read,omitempty"`
      Write *[]string `json:"write,omitempty"`
  }
  ```
- **验收：** `PATCH {fs:{read:["${workspace}/**"]}}` 更新存储的权限；后续带有匹配路径的 `fs.readFile` 调用成功；新 glob 之外的调用在 200 ms 内被拒绝（M2 SLO 仍然有效）。
- **测试：** SLO 回归测试镜像 M2 的 `TestRevoke_StorageWithin200ms` 但目标是 fs-glob 删除。
- **复杂度：** M

### T21 — Flutter 设置 UI：粒度能力
- **依赖：** T20
- **修改：** `/home/linivek/workspace/opendray/app/lib/features/settings/plugin_consents_page.dart` — 在每个能力开关下添加可展开部分：fs 显示每行一个声明的 glob（可切换），exec 显示每行一个模式，http 显示每行一个 URL glob。`/home/linivek/workspace/opendray/app/lib/core/api/api_client.dart` — 添加 `patchPluginConsents(name, ConsentPatch)`。
- **验收：** 切换关闭单个 fs.read glob（例如 `${home}/.ssh/**`）会更新权限，下一次针对该 glob 的插件调用失败，并通过 SnackBar 显示 EPERM。UI 不崩溃；开关可以切回开启状态。
- **测试：** Widget 测试 `test/features/settings/plugin_consents_granular_test.dart` 断言每种能力类型的 开关 → API 调用映射。
- **复杂度：** M

### T22 — Flutter bridge SDK: fs/exec/http/secret TS 类型
- **依赖：** T9–T13
- **修改：** `/home/linivek/workspace/opendray/app/lib/features/workbench/plugin_bridge_channel.dart` — 为新命名空间添加 Dart 端回退路由（除了信封路由外无需新代码；通道已转发未知 ns 字符串）。确保 WebView 宿主捆绑的预加载 shim 字符串包含 `fs`, `exec`, `http`, `secret` 代理子集。
- **修改：** `/home/linivek/workspace/opendray/gateway/plugins_assets.go` 内的嵌入式 Go 资产 `gateway.OpenDrayShimJS`（验证位置；否则创建 `plugins_shim.go`）— 扩展 `nsProxy` 调用以涵盖新命名空间 + `fs.watch` / `exec.spawn` 流式回调模式。
- **验收：** WebView 插件可以 `await opendray.fs.readFile("${workspace}/README.md")` 并获取内容。Shim 仍符合 60 行预算（M2 为 40 行；M3 允许使用流式辅助函数增加到 60 行）。
- **测试：** 使用通过每个新命名空间的两个模拟往返扩展 `/home/linivek/workspace/opendray/app/test/features/workbench/plugin_bridge_channel_test.dart`。
- **复杂度：** M

### T23 — 主接线
- **依赖：** T7, T9, T10, T11, T12, T13, T14, T15, T16, T17, T19, T20
- **修改：** `/home/linivek/workspace/opendray/cmd/opendray/main.go` — 在 365 行附近的现有 bridge 接线之后：
  1. 构造 `hostSupervisor := host.NewSupervisor(host.Config{DataDir: cfg.PluginsDataDir, Runtime: providerRuntime, State: db, Log: logger})`。
  2. 在关闭钩子链中注册 `hostSupervisor.Stop`（与 `installer.Stop` 生命周期相同）。
  3. 构建命名空间：`fsAPI := bridge.NewFSAPI(...)`, `execAPI := bridge.NewExecAPI(...)`, `httpAPI := bridge.NewHTTPAPI(...)`, `secretAPI := bridge.NewSecretAPI(...)`。
  4. `gw.RegisterNamespace("fs", fsAPI)`，对 exec/http/secret 执行相同操作。
  5. 连接 `hostSupervisor.SetRPCHandlerFactory(func(pluginName string) host.RPCHandler { return host.NewHostRPCHandler(host.HostRPCConfig{Namespaces: {...}, Plugin: pluginName, ...}) })`。
  6. 通过 `auth.NewKEKProviderFromAdminAuth(adminCreds)` 构造 KEK 提供者。
- **验收：** `./opendray` 引导成功。`GET /api/plugins/kanban/bridge/ws` 发送 `{v:1,id:"1",ns:"fs",method:"readFile",args:["..."]}` → EPERM（看板没有 fs 授权）。当没有插件声明 rust 时 `GET /api/plugins/lsp/rust/proxy` 返回 404；安装测试宿主插件使其返回 101 升级。
- **测试：** 使用 `TestBridge_M3NamespacesRegistered` 扩展 `/home/linivek/workspace/opendray/gateway/plugins_bridge_test.go`。
- **复杂度：** M

### T24 — PathVarResolver 实现 (网关)
- **依赖：** T9 (定义接口)
- **创建：** `/home/linivek/workspace/opendray/gateway/path_vars.go`, `/home/linivek/workspace/opendray/gateway/path_vars_test.go`
- **核心类型/函数：**
  ```go
  // pathVarResolver 为活动网关实现 bridge.PathVarResolver。
  type pathVarResolver struct {
      plugins *plugin.Runtime
      dataDir string
      sessions sessionLookup     // 用于工作区根目录的 sessionHub 最小接口
  }
  // sessionLookup 返回用户活动会话的 cwd, 如果没有会话则返回 ""。
  type sessionLookup interface {
      ActiveWorkspace(userID string) string
  }
  func (r *pathVarResolver) Resolve(ctx context.Context, plugin string) (bridge.PathVarCtx, error)
  ```
- **语义：** `workspace = sessions.ActiveWorkspace(defaultUserID)`（可能为空 — 插件获得空扩展，这会导致匹配失败，作为安全默认值）。`home = os.UserHomeDir()`。`dataDir = filepath.Join(cfg.PluginsDataDir, plugin, "<version>", "data")` 且该目录在第一次解析时以模式 0700 创建。`tmp = os.TempDir()`。
- **验收：** 插件上下文解析在一次会话中的 bridge 调用中保持一致。工作区更改在下一次 bridge 调用时生效。
- **测试：** 表格驱动。
- **复杂度：** S

### T25 — 参考插件 `plugins/examples/fs-readme/`
- **依赖：** T1, T2, T9, T11, T14, T22
- **创建：**
  - `/home/linivek/workspace/opendray/plugins/examples/fs-readme/manifest.json` — 宿主形态, runtime `node`, entry `sidecar.js`, 贡献 `commands` (一个命令 `fs-readme.summarise`), 权限 `fs.read:["${workspace}/**"]` + `exec:["node *"]`。
  - `/home/linivek/workspace/opendray/plugins/examples/fs-readme/sidecar.js` — 80 行 Node 脚本，在 stdin 上使用 `readline`，对方法 `summarise` 回复带有 `{result:<README 的前 200 字节>}` 的 JSON-RPC。演示 sidecar → host `fs/readFile` 调用，通过能力门控获取 README（宿主执行 I/O）。
  - `/home/linivek/workspace/opendray/plugins/examples/fs-readme/README.md` — 用法 + 证明内容。
- **验收：** 在 PATH 中有 Node 的开发宿主上执行 `opendray plugin install ./plugins/examples/fs-readme --yes`。调用该命令通过命令调用 HTTP 端点返回工作区 README 的前 200 字节。在空闲时杀死 sidecar 会在 500 ms 内重启；在请求期间杀死它会向调用者返回 `EUNAVAIL`。
- **测试：** 由 T26 E2E 覆盖。
- **复杂度：** M

### T26 — E2E 测试: fs-readme 全生命周期
- **依赖：** T25, T23
- **创建：** 使用 `TestE2E_FSReadmeFullLifecycle` (构建标签 `//go:build e2e`) 扩展 `/home/linivek/workspace/opendray/plugin/e2e_test.go`。
- **场景：**
  1. 通过本地源安装 fs-readme。
  2. 断言 `contributes.commands` 列出了 summarise 命令。
  3. 调用 `fs-readme.summarise` → 返回测试固定 README 的前 200 字节。
  4. `DELETE /api/plugins/fs-readme/consents/fs.read` — 下一次调用在 200 ms 内返回 EPERM。
  5. 通过 PATCH 重新授权；下一次调用正常。
  6. 通过 `SIGTERM` 杀死其 PID 以终止 sidecar — `Supervisor` 在 2 秒内重启；下一次调用正常。
  7. 尝试在强制 iOS 构建上安装（通过翻转 `HostFormAllowed` 的环境变量） — 安装被拒绝。
  8. 卸载 — plugin_secret_kek 行级联删除，plugin_host_state 行级联删除。
- **验收：** `go test -race -tags=e2e -timeout=10m ./plugin/...` 绿色通过。SLO 步骤的时间预算严格限制。
- **复杂度：** L
- **风险/缓解：** CI 上 sidecar 启动不稳定 (Node 不在 PATH)。缓解措施：当 `exec.LookPath("node")` 报错时通过 `t.Skip("node not on PATH")` 跳过测试 — 在签收检查清单中记录 CI 机器必须提供 Node 20。

### T27 — 遗留任务: CSP 测试 + 桌面端 webview 存根 + 看板 E2E
- **依赖：** —
- **创建：** `/home/linivek/workspace/opendray/gateway/plugins_assets_csp_test.go` (M2 T25 推迟的测试 — 黄金文件精确标头匹配)，使用 `TestE2E_KanbanFullLifecycle` (M2 T23 推迟的测试 — 来自 M2-PLAN §T23 规范) 扩展 `/home/linivek/workspace/opendray/plugin/e2e_test.go`。
- **修改：** `/home/linivek/workspace/opendray/app/lib/features/workbench/webview_host_desktop.dart` — 落地 M2 T16b 回退 widget (通过 `desktop_webview_window` 0.2.3 在模态桌面窗口中打开插件视图)。在 10-security.md §网络策略中记录软隔离限制。
- **验收：** M2 签收检查清单项 T16b/T23/T25 从 🟡 变为 ✅。
- **复杂度：** M
- **风险/缓解：** 低 — 这些是追赶任务，范围已冻结。

### T28 — 文档
- **依赖：** 全部
- **修改：** `/home/linivek/workspace/opendray/docs/plugin-platform/10-security.md` — 从下文 §6 增加威胁/缓解行 (TOCTOU fs, fork-bomb exec, SSRF http, KEK leak)；澄清 KEK 派生 + 轮换策略；记录仅限 Linux 的 cgroup 限制以及 macOS/Windows 的差异。 `/home/linivek/workspace/opendray/docs/plugin-platform/11-developer-experience.md` — 增加“宿主形态插件开发：15 分钟教程”，使用 fs-readme 作为演练。 `/home/linivek/workspace/opendray/docs/plugin-platform/SUMMARY.md` — 将 M3 行标记为绿色，使用新的 PathVarResolver 和 Supervisor 接口更新目录。 `/home/linivek/workspace/opendray/docs/plugin-platform/04-bridge-api.md` — 无签名更改 (M3 填充已声明的方法)。 `/home/linivek/workspace/opendray/docs/plugin-platform/M3-RELEASE.md` — 镜像 M2-RELEASE.md 风格的新文件：任务表、发布内容、排除项、手动冒烟测试、签收检查清单。
- **验收：** 新插件作者可以遵循宿主形态教程并在 ≤60 分钟内发布 fs-readme-lite。
- **复杂度：** S

### T29 — 第一个 PR 接缝 (可选)
- **依赖：** 无
- **捆绑：** 将 T1 + T3 + T4 + T5 + T8 + T15 放入一个 PR 中。网关无功能变化；随后按分支落地管理器/命名空间。净效果：新类型 + 迁移 + 帧编解码器，均可独立测试。
- **验收：** `go test -race ./...` 绿色通过；行为上二进制文件与当前的 `kevlab` HEAD 一致。
- **复杂度：** S

---

## 3. 任务依赖图

```
T1 ──────┬──▶ T2 ────────────────────────────────┐
         ├──▶ T18 ──▶ T19 ──────────────▶ T23 ───┤
         └──▶ T14 ──▶ T16 ──▶ T17 ─────┐         │
                │                       │         │
T3 ──────┬──────│──▶ T9 ──▶ T10 ───────┤         │
         ├──────│──▶ T11 ──────────────┤         │
         └──────│──▶ T12 ──────────────┤         │
                │                       │         │
T4 ──▶ T7 ──▶ T13 ────────────────────┤         │
T6 ──────────┘                          │         │
                                        │         │
T5 ──▶ T14 (依赖已在上方计算)            │         │
T15 ──▶ T16 ──▶ T17 ─────────────────── ┤         │
T8  ──▶ T9/T10/T11/T12 (流式辅助工具)    │         │
                                        ▼         │
                                      T23 ────────┘
                                        │
                                        ▼
T24 ──▶ (解析器在调用时被 T9/T11/T12 使用, 在 T23 接线)
T20 ──▶ T21                             │
                                        ▼
                                      T22 ──▶ T25 ──▶ T26 ──▶ T28
                                                    T27 (并行)
```

关键路径：`T1 → T14 → T16 → T17 → T23 → T25 → T26` (7 个跳步)。其余任务可在第一个 PR 接缝 (§5) 后并行运行。

---

## 4. 测试矩阵

### 单元测试 (`go test -race`, 触及包的目标覆盖率 ≥80%)
- 扩展 `/home/linivek/workspace/opendray/plugin/manifest_v1_test.go` (T1)
- 扩展 `/home/linivek/workspace/opendray/plugin/manifest_validate_test.go` (T1, T18)
- `/home/linivek/workspace/opendray/plugin/install/install_ios_test.go` + `install_desktop_test.go` (T2)
- `/home/linivek/workspace/opendray/plugin/bridge/capabilities_expand_test.go` (T3)
- 扩展 `/home/linivek/workspace/opendray/kernel/store/migrations_test.go` (T4, T5, T6)
- `/home/linivek/workspace/opendray/kernel/auth/secret_kek_test.go` (T7)
- 扩展 `/home/linivek/workspace/opendray/plugin/bridge/protocol_test.go` (T8)
- `/home/linivek/workspace/opendray/plugin/bridge/api_fs_test.go` (T9, T10)
- `/home/linivek/workspace/opendray/plugin/bridge/api_exec_test.go` (T11)
- `/home/linivek/workspace/opendray/plugin/bridge/api_http_test.go` (T12)
- `/home/linivek/workspace/opendray/plugin/bridge/api_secret_test.go` + `kernel/store/plugin_secret_test.go` (T13)
- `/home/linivek/workspace/opendray/plugin/host/supervisor_test.go` (T14)
- `/home/linivek/workspace/opendray/plugin/host/jsonrpc_test.go` (T15; 包括模糊测试)
- `/home/linivek/workspace/opendray/plugin/host/mux_test.go` (T16)
- `/home/linivek/workspace/opendray/plugin/host/host_rpc_handler_test.go` (T17)
- 扩展 `/home/linivek/workspace/opendray/plugin/contributions/registry_test.go` (T18)
- `/home/linivek/workspace/opendray/gateway/path_vars_test.go` (T24)

### 集成测试
- `/home/linivek/workspace/opendray/gateway/plugins_lsp_test.go` (T19) — `httptest` + 对接 `initialize` 的存根 sidecar。
- `/home/linivek/workspace/opendray/gateway/plugins_consents_patch_test.go` (T20) — 包括 SLO 回归。
- 扩展 `/home/linivek/workspace/opendray/gateway/plugins_bridge_test.go` (T23) — 断言已注册 fs/exec/http/secret 命名空间。
- `/home/linivek/workspace/opendray/gateway/plugins_assets_csp_test.go` (T27)。

### 端到端测试 (`//go:build e2e`)
- 使用 `TestE2E_FSReadmeFullLifecycle` (T26) 和 `TestE2E_KanbanFullLifecycle` (T27) 扩展 `/home/linivek/workspace/opendray/plugin/e2e_test.go`。

### Flutter widget 测试
- 扩展 `/home/linivek/workspace/opendray/app/test/features/workbench/plugin_bridge_channel_test.dart` (T22)。
- `/home/linivek/workspace/opendray/app/test/features/settings/plugin_consents_granular_test.dart` (T21)。

### 覆盖率门槛
CI 针对每个触及的包运行 `go test -race -cover ./plugin/... ./gateway/... ./kernel/...`，行覆盖率达到 80%。针对每个 PR 运行 `gosec ./plugin/bridge/... ./plugin/host/... ./kernel/auth/...`；任何新的 HIGH 发现都会阻止合并。沿用 M2 的目标 (80%+)。

---

## 5. 发布顺序

### 推荐线性顺序 (单名工程师, 顺序执行)

```
T29 (接缝) → T1 → T2 → T3 → T4 → T5 → T6 → T7 → T8 → T15 → T18 → T24 → T14 → T16 → T17 → T9 → T10 → T11 → T12 → T13 → T19 → T20 → T21 → T22 → T23 → T25 → T26 → T27 → T28
```

### 并行路径 (T29 接缝合并后)

三名 Agent 可并行工作：

- **路径 A — 管理器轨道:** T14 → T15 (顺序可交换) → T16 → T17 → T19。完全拥有 `/home/linivek/workspace/opendray/plugin/host/*`；与其他路径无文件重叠。
- **路径 B — 能力命名空间:** T3 → T7 → T8 (并行) → T9 → T10 (顺序 — 共享 `api_fs.go`) → T11 → T12 → T13。拥有 `/home/linivek/workspace/opendray/plugin/bridge/api_*.go` 和 `/home/linivek/workspace/opendray/kernel/store/plugin_secret.go` + `plugin_secret_kek.go` + `secret_kek.go`。
- **路径 C — 清单 + UI + 文档:** T1 → T2 → T18 → T20 → T21 → T22 → T27 → T28。拥有 manifest.go, contributions/registry.go (仅限注册表扩展), settings/flutter UI, 文档。

**合并点 (预计无冲突):**
- T23 (主接线) 必须最后执行 — 消耗所有内容。
- T25 (fs-readme 插件) 与路径 B 同步开发；在路径 A + 路径 B + T23 合并后由 T26 测试。
- T24 (路径变量解析器) 在路径 C 中开发但在 T23 中连接。

**文件隔离分析：**
- 路径 A：`plugin/host/` 下的 5 个新文件，加上 T23 中对 `cmd/opendray/main.go` 的添加。
- 路径 B：`plugin/bridge/` + `kernel/store/` + `kernel/auth/` 下的 8 个新文件，加上 T23 中对 `cmd/opendray/main.go` 的添加。
- 路径 C：`plugin/` 中的 2 个文件，`plugin/contributions/` 中的 2 个文件，`app/lib/features/settings/` 和 `app/lib/features/workbench/` 下的 3 个 Dart 文件。

只有 `cmd/opendray/main.go` 会看到多路径编辑；一旦路径 A 和 B 报告绿色，通过 T23 在单一顺序下暂存三个 main.go 添加项（hostSupervisor, 新命名空间, 路径变量解析器）。

---

## 6. 安全模型附录

| 威胁 | 受影响能力 | 缓解措施 | 验证方式 |
|---|---|---|---|
| `fs.readFile` 上的 **TOCTOU** — 插件提供的路径在 `stat` 和 `open` 之间变为指向 `/etc/passwd` 的符号链接。 | `fs.read`, `fs.write` | 在 `os.Open` 之前立即运行 `filepath.EvalSymlinks`, 然后针对授权 glob 重新匹配已解析的路径。两次检查之间的任何更改都会导致失败并关闭。 | T9 `TestFS_TOCTOUSymlinkEscape`, T10 `TestFS_WriteTOCTOU`。 |
| 通过 `exec.spawn` 的 **Fork 炸弹** — 插件循环生成自身, 耗尽宿主资源。 | `exec` | 每个插件硬性限制 4 个并发 spawn; 超过则排队并拒绝, 返回 `ETIMEOUT`。Linux cgroup v2 如果可写则 `pids.max=32` (否则记录一次警告)。在撤销许可时, 管理器杀死整个进程组。 | T11 `TestExec_ForkBombCapped` 生成 100 个短寿命 `sh` 进程; 断言 ≤4 个同时存活。 |
| 通过 DNS 重绑定在 `http.request` 上进行的 **SSRF 绕过** — 插件允许的主机在第二次查找时解析为 `169.254.169.254`。 | `http` | 自定义 `net.Dialer.Control` 回调在 `connect(2)` 之前立即运行, 并通过 `isPrivateHost` 重新检查**已解析的 IP**。拦截与 DNS 无关的重绑定和 IP 字面量攻击。 | T12 `TestHTTP_SSRF_DNSRebind` 带有恶意解析器存根。 |
| **KEK 泄露** — 宿主内存转储或 `/proc/<pid>/mem` 暴露派生的 KEK。 | `secret` | KEK 在每次使用时派生 (从不持久化); 存在于 `crypto/subtle` 安全的字节切片中; 在每次 wrap/unwrap 后的延迟块中通过 `explicit_bzero` 等效项 ( `runtime.KeepAlive` + 手动填充) 清零。没有 KEK, `plugin_secret_kek.wrapped_dek` 就毫无用处。密码轮换会轮换 KEK (遍历 + 重新包装)。 `kernel/auth/secret_kek.go` 上的 `gosec` 合并门槛。 | T7 `TestKEK_ZeroedAfterUse`, 宿主开发机上的手动 `strings /proc/<pid>/mem` 冒烟测试 (记录在 M3-RELEASE.md 中)。 |
| 通过精心设计的键名的 **秘密跨插件泄露**。 | `secret` | `MatchSecretNamespace` 正则 `^[a-zA-Z0-9._-]{1,128}$`; 行级 DB 范围限定 (PK 为 `(plugin_name, key)`)。API 从不接受 `plugin_name` 参数 — 它从 bridge Conn 的插件中隐式获取。 | T13 `TestSecret_KeyInjectionRejected`。 |
| **路径变量注入** — 插件声明 `permissions.fs.read=["${workspace}/**"]` 但工作区是一个指向 `/` 的符号链接。 | `fs.read`, `fs.write` | `PathVarResolver.Resolve` 在返回前对工作区调用 `filepath.EvalSymlinks` on the workspace before returning; if the symlink points outside `$HOME`, the resolver returns an error and the gate fails closed. | T24 `TestPathVar_WorkspaceSymlinkEscape`。 |
| **Sidecar 伪装** — sidecar 发送 JSON-RPC 调用, 声称自己是另一个插件。 | 所有 sidecar 可达能力 | `HostRPCHandler` 每个 sidecar 构造一次, 绑定 Supervisor 的 plugin-name 字段中的 `Plugin`; sidecar 无法覆盖。方法从处理程序接收 `plugin`, 绝不从 RPC 有效负载接收。 | T17 `TestHostRPC_MethodInjectionRejected`。 |
| 回环 bridge 上的 MITM 导致 **LSP 流量被篡改**。 | `contributes.languageServers` | LSP 代理仅接受来自 Flutter 宿主的经过身份验证的 WS 升级 (与 `/api/plugins/{name}/bridge/ws` 相同的 JWT 中间件)。根据 M2 源策略, 在移动端构建中仅限回环。 | T19 `TestLSP_UnauthenticatedRejected`。 |

**硬性保证扩展 (追加至 10-security.md §硬性保证):**
- 无论能力授权如何, sidecar 都无法读取或写入另一个插件的 `plugin_secret` 行 — 通过构造函数注入 `HostRPCConfig.Plugin` 强制执行。
- 没有 `exec` 能力的插件无法通过 `exec.run` 生成 sidecar 子进程; sidecar 本身由 `form:"host"` 管理, 仅通过管理器启动 (插件无法通过 bridge 调用管理器)。

---

## 7. 从 M2 迁移

### DB 迁移

下一个空闲 ID 是 014 (已确认: `kernel/store/migrations/013_plugin_audit.sql` 是最高的 M1 迁移; M2 未增加)。M3 引入三个：

| 迁移 | 文件 | 目的 |
|---|---|---|
| 014 | `kernel/store/migrations/014_plugin_secret_kek.sql` | 每个插件的已包装 DEK 行 (T4) |
| 015 | `kernel/store/migrations/015_plugin_host_state.sql` | 管理器持久化 (T5) |
| 016 | `kernel/store/migrations/016_plugin_secret_nonce.sql` | 添加 AES-GCM nonce 列 (T6) |

(017 预留用于潜在的 T26 修复; 不要预先分配。)

每个迁移都配有匹配的 `*_down.sql`。所有迁移都是增量的 — 不回填现有行 (M2 中密钥表为空)。

### 有线格式兼容性

M3 保持在 `V=1` 信封。每个 M3 方法都符合现有的信封形状 (请求/响应 + `stream:"chunk"|"end"`)。没有新的帧类型。Flutter M2 构建版本继续工作 — 它们只是不通过 shim 暴露新命名空间 (shim 受网关提供的 `opendray-shim.js` 内容的版本控制; 更新服务器会自动更新 shim, 无需重新构建 Flutter)。

### M2 合约保留

- 每个 M2 HTTP 端点 + WS 端点保持不变。
- `ContributesV1` JSON 是增量的 (新的 `languageServers` 字段为 `omitempty`)。
- `PermissionsV1` JSON 是增量的 — 无字段重命名; 现有字段保留其语义。
- `plugin_kv` + `plugin_consents` + `plugin_audit` 模式保持原样。
- M2 `TestE2E_KanbanFullLifecycle` (作为遗留任务在 T27 落地) 继续无改动通过。

### 回滚

向后兼容: 设置 `OPENDRAY_DISABLE_HOST_PLUGINS=1` 使 `hostSupervisor.Ensure` 返回 `EUNAVAIL` 而不杀死其他功能 (WebView 插件, 来自 WebView 插件的 fs/exec/http/secret, LSP 代理返回 503)。模式回滚: 反向运行 016 → 015 → 014 `_down.sql`; 因为 M2 下 `plugin_secret` 行为空, 所以无数据丢失。

---

## 8. “M3 完成”验收标准

- [ ] `fs-readme` 参考插件通过 `opendray plugin install ./plugins/examples/fs-readme --yes` 在 Linux + macOS 桌面构建版本上完成安装 (iOS/Android 由 `!HostFormAllowed` 排除)。
- [ ] 调用 `fs-readme.summarise` 在 500 ms 内返回工作区 `README.md` 的前 200 字节。
- [ ] 在请求中途杀死 sidecar 向调用者返回 `EUNAVAIL` (T14 重启 + T16 mux 集成); 后续调用在 2 秒内成功 (管理器已重新生成)。
- [ ] `rust-analyzer-od` (挑战目标) 为通过 LSP 代理打开的 Rust 文件提供补全 — 如果 CI PATH 上没有 Node/rust-analyzer, 则通过 `t.Skip` 跳过并在 M3-RELEASE.md 中注明; 仍需进行桌面开发演练。
- [ ] 所有四个特权命名空间在能力强制执行下正确响应其 M3 方法; 未经授权的调用返回 EPERM; SLO: 撤销 → 下一次调用 EPERM ≤ 200 ms (匹配 M2 SLO, 用于 T20 中落地的新能力差异路径)。
- [ ] AES-GCM 秘密往返在宿主重启后存活; 密码轮换在下一次登录时通过单次重新包装遍历来轮换 KEK (T7 + T13)。
- [ ] M3 触及的每个包上的 `go test -race -cover ./...` ≥ 80%。
- [ ] `go vet ./...` 干净; 触及包上的 `staticcheck ./...` 干净; `gosec ./plugin/bridge/... ./plugin/host/... ./kernel/auth/...` 不引入新的 HIGH 发现。
- [ ] `flutter test` 通过: 每个 M1 + M2 widget 测试仍为绿色 + 新的 T21/T22 widget 测试为绿色。
- [ ] 所有 17 个捆绑的旧版清单 + 看板 + time-ninja + fs-readme 加载后逐字节保持不变 (扩展黄金文件断言)。
- [ ] M2 的 `TestE2E_KanbanFullLifecycle` (现已通过 T27 落地) 继续无改动通过。
- [ ] iOS 归档构建成功。任何在 iOS 上安装 `form:"host"` 插件的尝试在 Stage 时均失败并返回 `ErrHostFormNotSupported` — 在 `TestE2E_FSReadmeFullLifecycle` 第 7 步中验证。
- [ ] LSP 代理接受来自 `wscat` 客户端的 `initialize` 请求并获得来自测试固定 sidecar 的响应。
- [ ] SSRF 测试 `TestHTTP_SSRF_DNSRebind` + TOCTOU 测试 `TestFS_TOCTOUSymlinkEscape` + fork-炸弹测试 `TestExec_ForkBombCapped` 均绿色通过。
- [ ] 文档: `11-developer-experience.md` 包含宿主形态插件开发教程; `10-security.md` 逐字包含 §6 威胁/缓解行; `M3-RELEASE.md` 发布并包含任务表 + 冒烟测试演练 + 签收检查清单。

### 冒烟测试演练 (手动)

在 PATH 中包含 Node 20 和 PostgreSQL (通过 embedded-postgres) 的 Linux 桌面端运行。

```bash
# 1. 清理数据目录
export OPENDRAY_DATA_DIR="${HOME}/.opendray-test-m3"
rm -rf "$OPENDRAY_DATA_DIR"; mkdir -p "$OPENDRAY_DATA_DIR/plugins/.installed"

# 2. 构建 + 启动
cd /home/linivek/workspace/opendray
go build -o opendray ./cmd/opendray
OPENDRAY_ALLOW_LOCAL_PLUGINS=1 OPENDRAY_DATA_DIR="$OPENDRAY_DATA_DIR" ./opendray &
GATEWAY_PID=$!; sleep 3

# 3. 安装 fs-readme
./opendray plugin install ./plugins/examples/fs-readme --yes
# 预期: "Installing fs-readme@1.0.0 with capabilities: fs.read, exec"

# 4. 调用命令
TOKEN="<device-code flow token>"
curl -X POST "http://localhost:8080/api/plugins/fs-readme/commands/fs-readme.summarise/invoke" \
     -H "Authorization: Bearer $TOKEN"
# 预期: 工作区 README.md 的前 200 字节

# 5. 撤销 fs.read; 重试; 预期 EPERM
curl -X PATCH "http://localhost:8080/api/plugins/fs-readme/consents" \
     -H "Authorization: Bearer $TOKEN" \
     -d '{"fs":{"read":[]}}'
curl -X POST "http://localhost:8080/api/plugins/fs-readme/commands/fs-readme.summarise/invoke" \
     -H "Authorization: Bearer $TOKEN"
# 预期: {error:{code:"EPERM"}}

# 6. 杀死 sidecar; 管理器重启
pkill -f "node.*fs-readme/sidecar.js"
sleep 2
curl -X POST "http://localhost:8080/api/plugins/fs-readme/commands/fs-readme.summarise/invoke" \
     -H "Authorization: Bearer $TOKEN"
# 预期: 成功 (在重新授权后); sidecar PID 与之前不同

# 7. 卸载; 验证级联删除
curl -X DELETE "http://localhost:8080/api/plugins/fs-readme" -H "Authorization: Bearer $TOKEN"
psql $OPENDRAY_DATABASE_URL -c "SELECT count(*) FROM plugin_secret_kek WHERE plugin_name='fs-readme';"
# 预期: 0

kill $GATEWAY_PID
```

### SLO 目标

| 指标 | 目标 | 测量方式 |
|---|---|---|
| `fs.readFile` (p95, 100 KiB 文件, 热缓存) | ≤ 30 ms | T9 基准测试 |
| `exec.run` (`git --version`, 冷启动 sidecar) | ≤ 150 ms | T11 基准测试 |
| `http.request` (GET api.github.com, LAN) | ≤ 300 ms | T12 基准测试 |
| `secret.get` + `secret.set` 往返 | ≤ 20 ms | T13 基准测试 |
| Sidecar 冷启动 (Node, fs-readme 固定程序) | ≤ 800 ms | T14 基准测试 |
| 许可撤销 → 下一次 EPERM (沿用 M2) | ≤ 200 ms | T20 SLO 测试 |
| 通过代理的 LSP `initialize` 往返 | ≤ 500 ms | T19 集成测试 |

---

## 开放性问题

1. **KEK 来源。** 建议: 通过 HKDF 从 bcrypt 管理员密码哈希 (而非密码本身 — 存储的哈希) 派生。 缺点: 轮换管理员密码需要遍历每一个 `plugin_secret` 行并重新包装。 替代方案: 在新的 `host_kek` 表中引入专用的宿主 KEK 行, 在首次引导时从 `crypto/rand` 生成种子, 并由 OS 密钥链条目 (macOS Keychain, Linux `libsecret`) 包装。 优点: 轮换是本地的。 缺点: 跨平台密钥链集成是 M6 的深坑。 **建议 M3 采用选项 A (从管理员哈希 HKDF); 将选项 B 标记为 M6 跟进任务。** Kev 待签收。

2. **Sidecar 的 gRPC 与 JSON-RPC 之争。** M3 建议在带有 LSP 帧封装的 stdio 上使用 JSON-RPC 2.0, 因为 (a) LSP sidecar 已经使用它, (b) Node/Python/Deno 实现非常简单, (c) 符合锁定 `protocol: "jsonrpc-stdio"` 的 `02-manifest.md` §host。 gRPC 对于结构化 RPC 更好, 但强制插件作者使用代码生成工具链。 **建议保持 JSON-RPC。 无需更改。** 标记以备完整。

3. **iOS 宿主形态。** M3 硬性拒绝 iOS 构建上的 `form:"host"` (App Store §2.5.2 风险)。 这意味着 rust-analyzer-od 永远不会在 iOS 上发布, 这没关系, 但它也意味着 **桌面端 Linux, 桌面端 macOS 以及 Android** 是受支持的宿主形态平台。 特别是 Android 可能需要额外审查 (Google Play §4.4 — 外部代码)。 **建议: M3 仅在 Linux + macOS 桌面端发布宿主形态; Android 在完成 Google Play 审查 (M4) 之前通过构建标志关闭。** Kev 待签收。

4. **Linux 上的 cgroup v2。** 没有 `CAP_SYS_ADMIN`, opendray 无法强制执行 cgroup 限制。 在登录会话下运行的用户模式 opendray 通常确实拥有自己的 cgroup (`user.slice/user-<uid>.slice/user@<uid>.service`), 可用于非特权操作, 但可写控制器的集合取决于发行版。 **建议: 在 sidecar 启动时尝试写入 `memory.max` + `pids.max`; 如果被拒绝, 每次宿主引导记录一次警告; 不导致 sidecar 启动失败。** 在 10-security.md 中记录此差异。

5. **fs.watch inotify 限制。** Linux inotify 具有每个用户的限制 (`/proc/sys/fs/inotify/max_user_watches`, 在大多数发行版中默认为 8192, 在某些容器中为 128)。 触及限制的插件作者会看到令人困惑的错误。 **建议: 每个插件限制 16 个活动监听器, 并暴露为清晰的 EINVAL。 在开发教程中记录 sysctl 调节旋钮。**

6. **http.request 上的重定向匹配。** 匹配 **不同** 插件声明的 URL 模式的重定向链跳转是否仍应被允许, 还是所有跳转都必须保持在单个初始模式内? 保守的选择是“根据完整授权列表重新匹配每一跳” (任意匹配)。 宽松的选择是“仅最终 URL 必须匹配”。 **建议: 在授权列表中进行任意匹配 (所有跳转均根据整个允许列表重新检查, 而不仅仅是初始模式)。 与浏览器对待 `connect-src` 的方式保持一致。** Kev 待签收。

7. **LSP 代理身份验证模型。** M3 在代理路由上使用现有的 JWT 中间件, 与 bridge WS 相同。 但声明了 `contributes.languageServers` 的插件隐式获得了 LSP 访问权限 — 没有单独的 `permissions.lsp` 能力。 这是否可以接受, 还是 LSP 流量应该由一个新的 `permissions.languageServer` 键控? **建议: 通过安装许可 + 现有的 `contributes.languageServers` 贡献隐式授权 — 符合 VS Code 的模型。 无需新能力。** Kev 待签收。

8. **Python 运行时支持。** `02-manifest.md` §host 锁定了 `runtime: {binary, node, deno}`。 提示词增加了 `python3, bun, custom` — 这是一个清单模式更改。 **建议: 在 T1 中扩展枚举并在 T28 中修补 `02-manifest.md`。 在 PR 描述中明确提出, 以便模式漂移可见。** Kev 在 T1 落地前确认枚举扩展是否可以。

---

## 相关文件路径 (全部为绝对路径)

### 设计合约
- `/home/linivek/workspace/opendray/docs/plugin-platform/12-roadmap.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/02-manifest.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/03-contribution-points.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/04-bridge-api.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/05-capabilities.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/07-lifecycle.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/10-security.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/11-developer-experience.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/M1-PLAN.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/M2-PLAN.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/M2-RELEASE.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/SUMMARY.md`

### 现有代码锚点
- `/home/linivek/workspace/opendray/plugin/manifest.go`
- `/home/linivek/workspace/opendray/plugin/manifest_validate.go`
- `/home/linivek/workspace/opendray/plugin/bridge/capabilities.go`
- `/home/linivek/workspace/opendray/plugin/bridge/manager.go`
- `/home/linivek/workspace/opendray/plugin/bridge/protocol.go`
- `/home/linivek/workspace/opendray/plugin/bridge/api_workbench.go`
- `/home/linivek/workspace/opendray/plugin/bridge/api_storage.go`
- `/home/linivek/workspace/opendray/plugin/bridge/api_events.go`
- `/home/linivek/workspace/opendray/plugin/commands/dispatcher.go`
- `/home/linivek/workspace/opendray/plugin/contributions/registry.go`
- `/home/linivek/workspace/opendray/plugin/compat/synthesize.go`
- `/home/linivek/workspace/opendray/plugin/install/install.go`
- `/home/linivek/workspace/opendray/plugin/runtime.go`
- `/home/linivek/workspace/opendray/plugin/e2e_test.go`
- `/home/linivek/workspace/opendray/kernel/store/plugin_consents.go`
- `/home/linivek/workspace/opendray/kernel/store/migrations/012_plugin_secret.sql`
- `/home/linivek/workspace/opendray/kernel/store/migrations/013_plugin_audit.sql`
- `/home/linivek/workspace/opendray/kernel/auth/credentials.go`
- `/home/linivek/workspace/opendray/gateway/server.go`
- `/home/linivek/workspace/opendray/gateway/plugins_bridge.go`
- `/home/linivek/workspace/opendray/gateway/plugins_consents.go`
- `/home/linivek/workspace/opendray/gateway/workbench_stream.go`
- `/home/linivek/workspace/opendray/cmd/opendray/main.go`
- `/home/linivek/workspace/opendray/app/lib/features/settings/plugin_consents_page.dart`
- `/home/linivek/workspace/opendray/app/lib/features/workbench/plugin_bridge_channel.dart`
- `/home/linivek/workspace/opendray/app/lib/core/api/api_client.dart`

### 待创建的文件 (M3)
- `/home/linivek/workspace/opendray/plugin/host_os_ios.go`
- `/home/linivek/workspace/opendray/plugin/host_os_other.go`
- `/home/linivek/workspace/opendray/plugin/bridge/api_fs.go`
- `/home/linivek/workspace/opendray/plugin/bridge/api_fs_test.go`
- `/home/linivek/workspace/opendray/plugin/bridge/api_exec.go`
- `/home/linivek/workspace/opendray/plugin/bridge/api_exec_test.go`
- `/home/linivek/workspace/opendray/plugin/bridge/api_http.go`
- `/home/linivek/workspace/opendray/plugin/bridge/api_http_test.go`
- `/home/linivek/workspace/opendray/plugin/bridge/api_secret.go`
- `/home/linivek/workspace/opendray/plugin/bridge/api_secret_test.go`
- `/home/linivek/workspace/opendray/plugin/bridge/capabilities_expand_test.go`
- `/home/linivek/workspace/opendray/plugin/host/supervisor.go`
- `/home/linivek/workspace/opendray/plugin/host/supervisor_test.go`
- `/home/linivek/workspace/opendray/plugin/host/process.go`
- `/home/linivek/workspace/opendray/plugin/host/process_unix.go`
- `/home/linivek/workspace/opendray/plugin/host/process_windows.go`
- `/home/linivek/workspace/opendray/plugin/host/jsonrpc.go`
- `/home/linivek/workspace/opendray/plugin/host/jsonrpc_test.go`
- `/home/linivek/workspace/opendray/plugin/host/mux.go`
- `/home/linivek/workspace/opendray/plugin/host/mux_test.go`
- `/home/linivek/workspace/opendray/plugin/host/host_rpc_handler.go`
- `/home/linivek/workspace/opendray/plugin/host/host_rpc_handler_test.go`
- `/home/linivek/workspace/opendray/plugin/install/install_ios_test.go`
- `/home/linivek/workspace/opendray/plugin/install/install_desktop_test.go`
- `/home/linivek/workspace/opendray/kernel/auth/secret_kek.go`
- `/home/linivek/workspace/opendray/kernel/auth/secret_kek_test.go`
- `/home/linivek/workspace/opendray/kernel/store/plugin_secret.go`
- `/home/linivek/workspace/opendray/kernel/store/plugin_secret_test.go`
- `/home/linivek/workspace/opendray/kernel/store/migrations/014_plugin_secret_kek.sql`
- `/home/linivek/workspace/opendray/kernel/store/migrations/014_plugin_secret_kek_down.sql`
- `/home/linivek/workspace/opendray/kernel/store/migrations/015_plugin_host_state.sql`
- `/home/linivek/workspace/opendray/kernel/store/migrations/015_plugin_host_state_down.sql`
- `/home/linivek/workspace/opendray/kernel/store/migrations/016_plugin_secret_nonce.sql`
- `/home/linivek/workspace/opendray/kernel/store/migrations/016_plugin_secret_nonce_down.sql`
- `/home/linivek/workspace/opendray/gateway/plugins_lsp.go`
- `/home/linivek/workspace/opendray/gateway/plugins_lsp_test.go`
- `/home/linivek/workspace/opendray/gateway/plugins_consents_patch.go`
- `/home/linivek/workspace/opendray/gateway/plugins_consents_patch_test.go`
- `/home/linivek/workspace/opendray/gateway/plugins_assets_csp_test.go`
- `/home/linivek/workspace/opendray/gateway/path_vars.go`
- `/home/linivek/workspace/opendray/gateway/path_vars_test.go`
- `/home/linivek/workspace/opendray/plugins/examples/fs-readme/manifest.json`
- `/home/linivek/workspace/opendray/plugins/examples/fs-readme/sidecar.js`
- `/home/linivek/workspace/opendray/plugins/examples/fs-readme/README.md`
- `/home/linivek/workspace/opendray/app/lib/features/workbench/webview_host_desktop.dart` (M2 T16b 遗留任务)
- `/home/linivek/workspace/opendray/app/test/features/settings/plugin_consents_granular_test.dart`
- `/home/linivek/workspace/opendray/docs/plugin-platform/M3-RELEASE.md`

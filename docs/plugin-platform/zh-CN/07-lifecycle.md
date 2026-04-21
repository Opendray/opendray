# 07 — 插件生命周期 (Plugin Lifecycle)

## 状态机

```
                    请求安装
                       │
          未安装 ──────►│
                       ▼
                    下载中 ──────── 失败 ────── 未安装
                       │
                   验证 + 许可
                       │
                       ▼
                    已安装 ◄─────── 禁用 ─────── 已启用
                       │
                       ▼                          ▲
                    已启用 ──────────────────────┘
                       │
                  触发激活事件
                       │
                       ▼
                    激活中 ──────── 失败 ────── 已启用 (错误标志)
                       │
                       ▼
                    活跃中  ────── 空闲超时 (宿主形式) ──┐
                       │                               │
                  触发停用事件                          │
                       │                               │
                       ▼                               │
                    停用中 ◄──────────────────────────┘
                       │
                       ▼
                    已启用
                       │
                      卸载
                       │
                       ▼
                    卸载中
                       │
                       ▼
                    未安装
```

## 状态语义 (State semantics)

| 状态 | 含义 | UI 显示 |
|-------|---------|---------------|
| `uninstalled` | 不在磁盘上 | 市场 (Marketplace) |
| `downloading` | 正在获取字节流 | “正在安装”行的进度条 |
| `installed` | 插件包已在磁盘，清单已验证，但用户尚未许可 | “待审核” |
| `enabled` | 已许可，已在数据库注册，符合激活条件 | 设置 |
| `activating` | 宿主正在启动边车 / 加载 Web 视图资源 | 加载动画 |
| `active` | 插件正在运行；接受桥接调用 | 绿点 |
| `deactivating` | 已发送 `shutdown`，最多等待 2 秒 | |
| `disabled` | 已注册但被屏蔽 —— 不分发激活事件 | 开关关闭 |
| `uninstalling` | 正在移除文件和数据库行 | |

## 激活事件 (Activation events)

当清单中 `activation` 列出的任何事件触发时，插件就会激活。`onStartup` 在宿主引导时激活。

| 事件 | 触发时机 |
|-------|-----------|
| `onStartup` | 宿主引导启动 |
| `onCommand:<id>` | 任何调用者运行该命令 |
| `onView:<id>` | 用户打开该视图 |
| `onSession:start` / `stop` / `idle` / `output` | 来自 `HookBus` 的会话事件 |
| `onLanguage:<id>` | 编辑器打开该语言的文件 |
| `onFile:<glob>` | 编辑器打开匹配通配符的文件 |
| `onSchedule:cron:<expr>` | 定时触发；使用 5 字段语法的 cron 解析 |

插件最多可以列出 32 个事件。如果列表为空，插件永远不会自动激活；用户必须手动激活。

## 空闲关闭 (仅限宿主形式)

如果边车进程在 `host.idleTimeoutSec`（默认 300 秒）内未收到请求，且没有活跃的订阅，宿主将发送 `shutdown`，插件返回 `enabled` 状态。下一个激活事件将重新启动它。

> **已锁定：** iOS 永远不会激活宿主形式的插件。在 iOS 上，针对宿主插件的 `onLanguage:*` 和 `onSession:output` 会静默执行无操作（no-op），并显示一次性横幅说明原因。

## 安装流程 (Install flow)

1. 调用 `POST /api/plugins/install`，正文为 `{ src }`。
   `src` 可以是：
   - `marketplace://<publisher>/<name>@<version>`
   - `https://...path/to/bundle.zip`
   - `local:/abs/path/to/bundle/` (仅限开发模式，受 `OPENDRAY_ALLOW_LOCAL_PLUGINS=1` 限制)
2. 宿主下载插件，进行 sha256 验证（以及适用的签名验证）。
3. 宿主根据 JSON Schema v1 验证清单。
4. 宿主计算与已安装版本（如果有）的能力差异。
5. 宿主返回 `202`，附带许可令牌和能力列表。
6. UI 显示许可界面；用户确认。
7. 调用 `POST /api/plugins/install/confirm {token}`。
8. 宿主将插件包解压到 `plugins/.installed/<name>/<version>/`。
9. 宿主写入许可行并固定清单哈希。
10. 宿主调用 `Runtime.Register()`（现有方法）并初始化数据库。
11. 如果清单列出了 `onStartup`，宿主立即激活。

## 更新流程 (Update flow)

- 使用相同的流程下载新版本。
- 如果能力差异为**相同或更窄**，则静默应用更新。
- 如果能力差异为**更宽**，则在激活新版本前重新提示许可。
- 旧版本保留在 `plugins/.installed/<name>/<oldVersion>/` 直到下次垃圾回收（24 小时），以便低成本回滚。
- 清单中的 `engines.opendray` 必须满足当前宿主版本，否则安装失败并返回 `EINCOMPAT`。

## 清单模式版本升级时的迁移

- 清单具有隐式的 `schemaVersion = 1`。v2 可能会增加必填字段；v1 插件将继续通过兼容层加载，并为缺失字段提供默认值。
- 插件可以提升 `engines.opendray` 以适配新版宿主；旧版宿主将拒绝安装。
- 宿主永远不会重写磁盘上的插件包。迁移仅在读取侧进行。

## 崩溃恢复 (Crash recovery)

### Web 视图崩溃
- Flutter 检测到 Web 视图进程死亡 → 外壳显示“插件已崩溃 —— 是否重新加载？”的操作。
- 每个插件每天有 `crashes` 计数器。崩溃 3 次以上 → 插件自动禁用并发出通知。

### 边车崩溃
- 管理器以指数退避策略重启（2s, 4s, 8s, 16s, 32s）。如果在 10 分钟内发生 5 次失败，插件将自动禁用。
- 崩溃转储（标准错误尾部 ≤ 64 KB）存储在 `plugins/.crash/<name>-<ts>.log` 中；显示在“设置 → 插件 → 日志”中。

### 宿主崩溃 (整个 Go 进程死亡)
- 整个应用重启。所有插件进入 `enabled` 状态 → 根据清单事件重新激活。插件无法阻止系统引导启动。

## 卸载流程 (Uninstall flow)

1. 调用 `DELETE /api/plugins/<name>`。
2. 宿主调用边车的 `deactivate` + `shutdown`，等待 2 秒，然后发送 SIGKILL。
3. 卸载 Web 视图。
4. 调用 `Runtime.Remove()` —— 现有方法。
5. 删除 `plugin_kv` 行（除非用户选择“保留我的数据”）、`plugin_consent`、`plugin_audit`。
6. 移除解压后的插件包目录。
7. 发出事件 `plugin.uninstalled`。

## 针对当前清单的兼容模式

`plugins/agents/*` 和 `plugins/panels/*` 下的当前清单**没有**以下字段：
- `form`, `publisher`, `engines`, `contributes`, `permissions`。

在加载时，宿主会合成一个兼容清单：
```
form      = 如果 type 为 ("cli","local","shell") 则为 "host"，否则为 "declarative"
publisher = "opendray-builtin"
engines   = { "opendray": ">=0" }
contributes.agentProviders = [原样保留 Provider]   // 针对智能体清单
contributes.views = [{ id: <name>, title: displayName, container: "activityBar", render: "webview" }]  // 针对面板清单
permissions = {} // 宿主信任内置插件
```
这些合成的清单仅保存在内存中；磁盘上的文件永远不会被重写。内置插件被视为“受信任的发布者”，并跳过许可界面。

> **已锁定：** 兼容模式对 v1 是永久性的。当 v2 发布时，内置插件将逐一迁移到原生的 v1 清单，且不造成破坏。

## 负责的包

- `plugin/install/` (新增) — 下载、验证、解压、许可。
- `plugin/runtime.go` (现有) — 注册 + 数据库。
- `plugin/host/supervisor.go` (新增) — 边车生命周期。
- `plugin/lifecycle/activation.go` (新增) — 挂接到 `HookBus` 的激活事件调度器。

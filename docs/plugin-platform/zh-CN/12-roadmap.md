# 12 — 路线图 (Roadmap)

## 里程碑

每个里程碑都独立发布，并解锁特定的开发者/用户体验。只有当其验收标准通过 CI 时，里程碑才算“完成”。

### M1 — 基础 (安装 + 声明式)
**解锁：** 第三方可以发布声明式插件。
**范围：**
- `plugin/install/` 包：下载、sha256 验证、解压、授权。
- 清单 v1 解析器（当前 `Provider` 的超集）。
- 数据库迁移：`plugin_consents`、`plugin_kv`、`plugin_secret`、`plugin_audit`。
- 桥接网关骨架 + 能力门控。
- `contributes.commands`、`contributes.statusBar`、`contributes.keybindings`、`contributes.menus` 端到端实现。
- SDK 脚手架 (`opendray plugin scaffold --form declarative`)。
- Flutter 壳根据注册的贡献渲染状态栏 + 命令面板条目。

**验收：**
- 一个手动构建的 `time-ninja` 插件从 `marketplace://…` 安装，重启后依然存在，且其命令显示在面板中。
- 现有的 `plugins/agents/*` 和 `plugins/panels/*` 无需修改即可加载。
- 卸载会删除所有痕迹。

### M2 — Webview 运行时
**解锁：** 丰富的 UI 插件。
**范围：**
- 带有 `plugin://` 方案处理程序的 `gateway/plugins_assets.go`。
- 带有 `window.opendray` 和有线协议的 WebView 预加载。
- 桥接 WebSocket (`/api/plugins/{name}/bridge/ws`)。
- `contributes.activityBar`、`contributes.views`、`contributes.panels`。
- `opendray.workbench.*`、`opendray.storage.*`、`opendray.events.*`。
- CSP 强制执行 + 平台允许的情况下实现每个插件的 WebView 隔离。

**验收：**
- `kanban` 示例插件在 Android、iOS 和桌面上运行。
- 在运行时撤销 `storage` 权限会导致下一次 `storage.set` 在 200 毫秒内失败。

### M3 — 宿主边车运行时
**解锁：** LSP、重型后台插件。
**范围：**
- 带有退避和空闲关闭功能的 `plugin/host/supervisor.go`。
- 带有 LSP 帧的 JSON-RPC 2.0 标准输入输出。
- 带有完整能力强制执行的 `opendray.fs.*`、`opendray.exec.*`、`opendray.http.*`。
- `contributes.languageServers` + LSP 代理。
- 支持 Node 和 Deno 运行时。

**验收：**
- `rust-analyzer-od` 插件在 Rust 文件中提供补全。
- 在请求中途杀掉边车会返回干净的 `EUNAVAIL`；管理器（supervisor）将其重新启动。

### M4 — 市场客户端 + 发布者
**解锁：** 插件生态系统。
**范围：**
- `plugin/market/` 获取 `index.json`，解析版本。
- 设置 → 市场浏览 + 安装 UI。
- `opendray plugin publish` CLI（fork + PR + 签名）。
- 撤销列表轮询。
- 签名验证。

**验收：**
- 端到端：将插件从 SDK 发布到暂存市场，在不到 5 分钟内从另一台设备安装。
- 紧急开关条目在合并后 10 分钟内卸载恶意测试插件。

### M5 — 合约冻结 (v1 正式版)
**解锁：** 第三方可以依赖 API。
**范围：**
- 发布所有标记为 MVP 的贡献点。
- 实现 04-bridge-api.md 中列为 MVP 的所有 `opendray.*` 命名空间。
- 在 docs.opendray.dev 发布文档。
- 示例插件仓库上线。
- 通过 App Store 审核测试 iOS 构建（分阶段推出）。

**验收：**
- 每个示例插件都通过 `opendray plugin validate`。
- 桥接 API 或清单架构没有打开的 P0/P1 错误。
- 合约冻结日期：**2026-10-01**（M5 发布后）。在此日期之后，清单架构和桥接 API 签名更改需要大版本更新。

### M6 — 开发者体验 (DX) 完善
**解锁：** 插件作者在几小时而非几天内变得富有成效。
**范围：**
- 所有表单的热重载。
- 用于离线 SDK 使用的便携式 `opendray-dev` 宿主。
- 桥接追踪工具。
- 本地化流水线。
- 根据清单生成文档网站。

**验收：**
- 新插件作者可以在 30 分钟内按照 README 发布他们的第一个插件（针对 3 位外部开发者进行测试）。

### M7 — v1 后的扩展
**非阻塞性探索。** 这些都不影响 v1：
- `contributes.debuggers` (DAP)。
- `contributes.languages`（语法高亮 + 代码片段）。
- `contributes.taskRunners` 原生可插拔运行器（保留当前的 `plugins/panels/task-runner`）。
- 插件对插件的命令导出权限。
- 一流的私有市场支持。
- 多视图分割布局。
- 付费插件和计费。

## 当前插件的弃用计划

目前的 `plugins/agents/*` 和 `plugins/panels/*` 通过兼容模式（参见 [07-lifecycle.md](07-lifecycle.md)）在 v1 中不受影响地获得支持。

| 期间 | 状态 |
|--------|-------|
| M1 → M5 | 兼容模式。新功能仅在 v1 清单上落地。文档指向新插件的 v1。 |
| M5 → v1.5 | 当 SDK 验证器检测到旧版清单时显示弃用横幅。 |
| v2 | 移除兼容模式。任何剩余的旧版清单在宿主启动时自动迁移。 |

内置插件（6 个代理 + 11 个面板）在 M5 期间逐个 PR 迁移到仓库内的原生 v1 清单，且不破坏用户配置。`plugin_kv` 中的配置行保持不变，因为字段键得到了保留。

## 合约冻结政策

**2026-10-01** 之后：
- 添加新的可选清单字段：小版本更新。
- 添加新的可选桥接方法/命名空间：小版本更新。
- 重命名、移除或更改现有字段或方法的行为：大版本更新；旧合约至少支持 12 个月。

每次破坏性更改必须包括：
- 标记旧用法的 SDK `lint` 规则。
- 宿主上的兼容性适配。
- CHANGELOG.md 中的迁移说明。

## 追踪

里程碑在 `Obsidiannote/Projects/OpenDray/plugin-platform/roadmap.md` 中进行追踪。每个里程碑在主仓库中都有一个 Linear/任务列表。

> **已锁定：** v1 合约冻结日期为 2026-10-01。向后推迟会延误第三方生态系统；向前提前则面临 API 不成熟的风险。除非 M1-M4 累计推迟超过一个日历月，否则将其视为不可协商。

## Kev 摘要 (≤300 字)

**已锁定 (Locked)**
- 三层拆分：Flutter 工作台 (仅负责渲染贡献点) · Go 插件宿主 (生命周期 + 桥接网关) · 三种形式的插件 (声明式 / WebView / 宿主侧车)。
- 保持单一 Go 二进制文件部署；无新基础设施。
- Manifest v1 是目前 `plugin.Provider` 的严格超集；所有当前的 manifest 都可以通过兼容模式保持不变地加载。
- 桥接 API 表面 (`opendray.workbench/fs/exec/http/session/storage/secret/events/commands/tasks/ui/clipboard/llm/git/telegram/logger`) 已冻结，如 04-bridge-api.md 中的 TS 类型定义。
- 能力声明 + 安装时授权 + 运行时拦截 + 审计日志。
- 市场 = Git 仓库注册表 (`opendray/marketplace`)，sha256 固定构件，可选 ed25519 签名，基于 PR 的审批，基于轮询的撤销。
- 针对侧车的 JSON-RPC 2.0 stdio (配合 LSP 分帧)。
- iOS 上禁用宿主插件 (App Store §2.5.2 安全性要求)。
- v1 合约冻结日期：**2026-10-01**。

**决策 (2026-04-19 已锁定)**
1. 桥接传输：每个插件拥有专用的 `/api/plugins/{name}/ws` (而非共享的会话 WS)。阻塞 M2 → 已解决。
2. 手机手势：双指 + 边缘滑动；移除三指手势 (与 VoiceOver 冲突)。阻塞 M5 → 已解决。
3. iOS 内置插件：随附当前所有 11 个按哈希固定的 `plugins/panels/*`；发布后根据遥测结果精简至 5-7 个。
4. `opendray-dev` 便携式宿主：在 M6 中与 SDK 一起获得资金支持 —— 开发者体验 (DX) 对生态系统至关重要。
5. 跨插件命令执行：v1 允许所有执行并记录审计日志；v2 引入 `exported: true` 选项。
6. Manifest `extends` 字段：推迟到 v2，仅在确实出现使用案例时引入 (YAGNI)。

**前 3 大风险**
1. iOS 审核：WebView + 禁用宿主插件在逻辑上是站得住脚的，但如果不熟悉该模型的审核员可能会拒绝。缓解措施：在提交前与 App Review 进行沟通，10-security.md 中提供审核员说明模板。
2. macOS/Windows 上的 Supervisor 沙箱是尽力而为的；恶意宿主插件仍可能在用户权限范围内窃取信息。缓解措施：默认拒绝 `http`/`fs`/`exec`，并提供明显的授权文案。
3. 模式锁定过早：v1 可能会僵化在当前的 `Provider` 形状周围。缓解措施：`v2Reserved` 字段 + 严格的未知字段警告 (而非错误) 策略。

**规划器的 M1 第一步任务**
- 添加数据库迁移：`plugin_consents`, `plugin_kv`, `plugin_secret`, `plugin_audit`。
- 创建 `plugin/install/` (下载 + sha256 + 解压 + 授权令牌)。
- 扩展 `plugin/manifest.go`，将 v1 字段 (form, publisher, engines, contributes, permissions) 作为可选字段；保持 `Provider` 加载路径不变。
- 为 `gateway/plugins_install.go` 提供脚手架，包含 `POST /api/plugins/install` + `/confirm`。
- 发布初始的 `@opendray/plugin-sdk` npm 包，包含 manifest JSON schema 和一个声明式脚手架模板。

## 相关文件路径

- `/home/linivek/workspace/opendray/plugin/manifest.go`
- `/home/linivek/workspace/opendray/plugin/runtime.go`
- `/home/linivek/workspace/opendray/plugin/hooks.go`
- `/home/linivek/workspace/opendray/gateway/server.go`
- `/home/linivek/workspace/opendray/gateway/api.go`
- `/home/linivek/workspace/opendray/kernel/store/db.go`
- `/home/linivek/workspace/opendray/kernel/store/queries.go`
- `/home/linivek/workspace/opendray/plugins/agents/claude/manifest.json` (兼容模式参考)
- `/home/linivek/workspace/opendray/plugins/panels/git/manifest.json` (兼容模式参考)
- `/home/linivek/workspace/opendray/plugins/panels/telegram/manifest.json` (Telegram 桥接表面)

上述所有 13 份设计文档应由父编排器在 `/home/linivek/workspace/opendray/docs/plugin-platform/` 下生成，并使用每个章节标题中指定的准确文件名。

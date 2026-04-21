# OpenDray 插件平台 M4 — 市场 (消费侧)

**状态：** ✅ M4.1 已关闭 — 详情请参阅 `M4-RELEASE.md`。
**依赖项：** M3 关闭于 commit `208bcb3` (`kevlab`)
**范围：** 仅限 M4.1 — 消费侧市场。第三方发布者 CLI (M4.2) 推迟至 v1 发布之后；在此期间 Kev 是唯一的发布者并手动编辑市场仓库。
**合约冻结日期 (不可移动)：** 2026-10-01 — M4.1 + M5 必须在剩余窗口内完成。

## 1. 范围界限

### 范围内 (M4.1 — 本里程碑)

- 真实的远程市场：`plugin/market/` 包从公共注册表 URL 获取 `index.json` + 每版本 JSON，验证 SHA-256，验证 Ed25519 签名，并缓存至本地。
- 连接 `HTTPSSource.Fetch` — `marketplace://` 引用解析为针对远程注册表（M3 发布了本地目录桩；M4 将其替换为真实的）。
- 撤销列表轮询（默认 6 小时 + 启动时）。操作：`uninstall` (卸载) / `disable` (禁用) / `warn` (警告)。
- Flutter：中心 (Hub) 从读取本地目录迁移至调用由远程获取后端支持的新 `/api/marketplace/registry/*` 接口；显示信任徽章 + 自动更新指示器。
- 市场仓库模板 (`github.com/Opendray/opendray-marketplace`)，带有 CI 工作流，在每次合并至 `main` 时重新生成 `index.json`。即使在唯一发布者模式下也需要此模板，以便 Kev 手动编辑的 PR 得到一致的验证和发布。

### 推迟至 M4.2 (发布后)

- **发布者 CLI `opendray plugin {scaffold,validate,build,publish}`**。第三方开发者尚无法自行发布；在 M4.2 落地之前，他们要么手动提交针对市场仓库的 PR，要么通过 Kev 进行。理由：应用发布日期紧迫，发布者工作流是在相同有线格式之上的非阻塞人体工程学层。记录在项目记忆 `m4_2_publisher_cli.md` 中。

### 范围外 (推迟至 M5 或更晚)

- 旧版插件向 v1 的迁移 — 按照 12-roadmap §Deprecation 规定在 M5 进行。
- Webview + 侧车混合形式 — M5，已在 M3-RELEASE §3 中标注。
- iOS App Store 提交 — M5。
- 付费插件 / 计费 — M7。
- 一等公民级别的私有市场支持 — M7。
- 市场内评分 / 评论 — v1 发布之后。

## 2. 已解决的设计决策

2026-04-20 已解决全部五个悬而未决的问题。记录于此以便 T1 从一组固定的约束开始；未来的歧义通过阅读本节解决。

### 2.1 注册表 URL — CDN 前置的自定义域名

**主域名：** `https://marketplace.opendray.dev/` (Cloudflare CDN)。
**备用镜像 1：** `https://raw.githubusercontent.com/opendray/marketplace/main/`
**预留镜像位：** 社区运行的镜像（可配置，无默认值）。

为什么不直接使用 GitHub：未认证的速率限制为每 IP 每小时 60 次请求。在生态系统规模下，每次应用启动都会拉取 `index.json` + 任何插件安装都会拉取每版本 JSON — 使用共享 NAT 的一栋办公楼会在几分钟内触发限制。CDN 可以吸收这些请求并提供全球低延迟。自定义域名在同意书中也显得更正规（“marketplace.opendray.dev 表示此插件……”）。

基础设施：Cloudflare DNS CNAME → GitHub Pages 或 R2。Kev 已在运行 CF 隧道；增加的表面积 ≈ 零。

### 2.2 签名密钥 — 三层查找

发布者 CLI + 服务端的签名验证器按此优先级解析私钥：

1. `OPENDRAY_SIGNING_KEY` 环境变量 — 用于 CI / GitHub Actions。
2. 操作系统钥匙串 (macOS Keychain / Linux Secret Service / Windows Credential Manager) — 开发者默认使用。服务名称为 `opendray`，账号为 `<publisher>`。使用 `github.com/zalando/go-keyring` 实现跨平台。
3. `~/.config/opendray/keys/<publisher>.age` — 针对边缘情况（共享工作站、无 Secret Service 的无头 Linux）的密码加密文件。

镜像了每个插件作者都已经熟悉的 `age` / `sops` / `gpg` / `vsce` 模式。

### 2.3 市场 CI — GitHub Actions

具有第三方 PR 贡献者的公共仓库。自托管的 Gitea 需要贡献者在 `tea.linivek.online` 注册 — 这会产生足够的摩擦力从而扼杀生态系统。

预算：公共仓库在标准运行器上获得无限制的 GitHub Actions 分钟数。

### 2.4 撤销轮询频率 — 可通过环境变量覆盖

默认 6 小时（按照规范）。通过 `OPENDRAY_REVOCATION_POLL_HOURS` 暴露，下限 1 / 上限 168。

- `1 h` — 高安全性（金融、医疗）。
- `24 h` — 移动端电池敏感。
- `168 h` (1 周) — 带有本地镜像的企业集群。

上限 168 防止设置为“从不” — 撤销是安全基础设施，必须最终进行轮询。

### 2.5 中心 (Hub) vs 设置 → 市场 — 两者兼有

- **中心 (Hub)** (自 M3 起存在)：消费侧界面。浏览卡片、安装按钮、同意对话框、配置对话框。
- **设置 → 市场** (M4 新增子页面)：管理侧界面。注册表 URL + 镜像、自动更新开关、信任级别过滤（“仅显示已验证+”）、“立即刷新缓存”、撤销日志。

两种角色，两个页面。与 App Store “浏览” vs iOS 设置 → “App Store” 偏好设置的划分相同。

## 3. 额外的生态系统级决策

### 3.1 信任级别 — 语义

| 级别 | 含义 | 授予方式 |
|-------|-------|-----------|
| `official` | OpenDray 核心团队维护或审计 | 手动编辑 `publishers/opendray.json` |
| `verified` | 发布者通过了 DNS TXT + 身份检查，Ed25519 密钥已注册 | CI 验证 DNS TXT 解析 + 匹配令牌；人工在合并时设置 `trust: verified` |
| `community` | 任何已合并 PR 的作者 | 默认值 |
| `sideloaded` | 当安装源不是市场时客户端附加的标签 | 客户端在 `local:` / 裸路径上自动附加 |

### 3.2 发布者验证流程 (供参考；在 M4.2 中实现)

1. 第三方开发者 fork `github.com/Opendray/opendray-marketplace`。
2. 添加 `publishers/<name>.json`，包含 `keys: [ed25519 pubkey]` + `domainVerification.record: "opendray-verify=<token>"`。
3. 在声明的域名下添加 DNS TXT `opendray-verify=<token>`。
4. 开启 PR。
5. CI (T24) 验证 TXT 记录解析并匹配令牌。
6. 人工维护者评审 → 合并 → 信任级别从 `community` 开始。升级到 `verified` 是身份检查后的独立手动编辑。

在市场仓库的 README 中记录。M4.2 中的发布者 CLI 会自动执行步骤 1-2 和 4。

### 3.3 版本锁定 — 不支持 "latest"

安装引用必须携带特定版本：`marketplace://<publisher>/<name>@<version>`。裸 `<name>` 引用在浏览时客户端解析为最新版本，但安装调用携带锁定版本。防止 PR 合并时发生静默升级。自动更新是独立的按插件选择项。

### 3.4 命名空间 — 从第一天起使用 `publisher/name`

安装引用使用 `publisher/name` 格式。预留一个与第三方生态系统兼容的形状，可以避免在 M4.2 落地时发生破坏性变更。M3 的裸 `name` 引用（如 `marketplace://fs-readme`）在 T1 重构期间向后兼容地解析为 `opendray-examples/fs-readme`。

## 3. 任务图 (29 个任务)

### 后端 — 市场客户端 (T1–T12)

| ID | 标题 | 依赖于 | 工作量 |
|----|-------|------------|--------|
| T1 | 重构 `plugin/marketplace` → `plugin/market/local` + 引入 `plugin/market/remote` 骨架 | — | S |
| T2 | `market/remote.FetchIndex` — HTTP GET index.json + 模式验证 | T1 | S |
| T3 | `market/remote.FetchVersion` — 每版本 JSON + SHA256 检查 | T2 | S |
| T4 | `install.HTTPSSource.Fetch` — 下载 ZIP，验证 SHA256，解压至临时区 | T3 + T1 | M |
| T5 | Ed25519 签名验证 + 发布者密钥解析 | T3 | M |
| T6 | 镜像回退 — 在 5xx/超时时进行轮询重试 | T2, T3, T4 | S |
| T7 | 注册表缓存 — 文件系统持久化，stale-while-revalidate | T2 | S |
| T8 | `market/revocations.go` — 轮询循环 (6 h + 启动时), `plugin_revocation_seen` 表 | T2 | M |
| T9 | 撤销操作调度器 — 通过安装器 + Provider 运行时连接卸载 / 禁用 / 警告操作 | T8 | M |
| T10 | 信任级别传播 — Entry.Trust 从发布者记录流转至 Hub | T5 | S |
| T11 | 网关 `GET /api/marketplace/registry` (索引 + 每版本) + `POST /refresh` | T7, T10 | S |
| T12 | 设置 → 市场管理子页面后端 — 配置中的注册表 URL + 镜像 + 用户偏好中的自动更新开关 | T11 | S |

### 发布者 CLI (T13–T17) — **推迟至 M4.2**

`cmd/opendray-plugin/` 是一个独立于 `cmd/opendray` 的新二进制文件，因此插件作者不需要完整的网关。**不在 M4.1 范围内** — 在 M4.2 落地之前，唯一的发布者 (Kev) 手动编辑市场仓库 + 依靠 CI (T22–T24) 进行验证。

在记忆 `m4_2_publisher_cli.md` 中记录以免被遗忘。

| ID | 标题 | 依赖于 | 工作量 |
|----|-------|------------|--------|
| T13 | `cmd/opendray-plugin/` 骨架 + cobra 风格的子命令调度 | — | S |
| T14 | `scaffold` — 交互式清单向导 (表单 / 权限 / configSchema) | T13 | M |
| T15 | `validate` — 针对本地清单运行 `ValidateV1` + 捆绑包 Lint 规则 (zip 大小, setuid 位, 符号链接逃逸) | T13 | S |
| T16 | `build` — zip + SHA256 + 可选的 Ed25519 签名 | T15, T5 | M |
| T17 | `publish` — fork 市场仓库，创建分支，写入每版本 JSON，使用模板正文开启 PR (+ 针对发布者入驻的 DNS TXT 验证助手) | T16, T11 | M |

### Flutter (T18–T21)

| ID | 标题 | 依赖于 | 工作量 |
|----|-------|------------|--------|
| T18 | 中心 (Hub) 从 `/api/marketplace/registry` (而非本地目录) 获取数据 + 缓存感知的刷新 | T11 | M |
| T19 | 中心卡片上的信任徽章 (official / verified / community / sideloaded) + 图例 | T18, T10 | S |
| T20 | 自动更新 — 插件页面列表显示“有更新可用”芯片；设置中针对能力扩大的开关 | T18, T11 | M |
| T21 | 针对 `uninstall` / `disable` 操作的撤销横幅 + 系统对话框 | T9 | S |

### 市场仓库基础设施 (T22–T24)

这些落地于 `github.com/Opendray/opendray-marketplace`，而非主仓库。

| ID | 标题 | 依赖于 | 工作量 |
|----|-------|------------|--------|
| T22 | 模板仓库布局 (plugins/ / publishers/ / CODEOWNERS / revocations.json) | — | ✅ `opendray-marketplace@5d10d36` |
| T23 | GitHub Actions：在 push 到 main 时重新生成 index.json + 上传至 CDN 镜像 | T22 | ✅ `opendray-marketplace@5d10d36` |
| T24 | CI 验证：清单模式 + SHA 匹配产物 URL + 沙箱扫描 (禁用文件类型) + PR 上的能力差异评论 | T23 | ✅ `opendray-marketplace@5d10d36` (与 T22/T23 一同发布) |

### 测试 (T25–T27)

| ID | 标题 | 依赖于 | 工作量 |
|----|-------|------------|--------|
| T25 | 集成测试：网关对接 `file://` 测试注册表桩 | T4, T5 | M |
| T26 | 签名验证单元测试 + 集成测试 | T5 | S |
| T27 | 撤销 E2E：伪造注册表标记插件为已撤销 → 客户端轮询 → 动作触发 | T8, T9 | M |

### 文档 (T28–T29)

| ID | 标题 | 依赖于 | 工作量 |
|----|-------|------------|--------|
| T28 | 使用发布者工作流 + 新 CLI 更新 `11-developer-experience.md` | T17 | S |
| T29 | M4-RELEASE.md — 状态表 + 冒烟测试 + 提交历史 | 全部 | S |

**总计 (M4.1):** 24 个任务 · 8 M / 16 S。预计持续时间：按 M3 的速度约为 3 周。
**总计 (M4.2):** 5 个任务 · 3 M / 2 S。在 v1 发布后运行。

## 4. 依赖链

```
T1 (重构)
 └▶ T2 (获取索引) ──┬▶ T3 (获取版本) ──┬▶ T4 (HTTPS 源) ─┐
                     │                       │                     │
                     │                       ├▶ T5 (签名) ──┤
                     │                       │                     │
                     │                       └▶ T6 (镜像) ─────┤
                     │                                              │
                     ├▶ T7 (缓存) ────────┬▶ T11 (注册表 API) ─┼▶ T18 (中心) ─┬▶ T19 (徽章)
                     │                     │                      │             ├▶ T20 (更新)
                     │                     └▶ T12 (设置) ─────┘             │
                     │                                                           │
                     └▶ T8 (撤销) ──▶ T9 (动作) ────────────────────── └▶ T21 (横幅)

T13–T17 (发布者 CLI) — **推迟至 M4.2**，v1 发布后解锁。

T22 (仓库模板) ──▶ T23 (CI 生成) ──▶ T24 (PR 检查)

测试/文档依赖于上述所有内容；滚动更新顺序见下文。
```

## 5. 滚动更新顺序 (M4.1)

1. **T22 → T23** — 创建市场仓库 + 重新生成 `index.json` 的 CI。即使是唯一发布者模式也需要在 T2 有内容可获取之前运行此项。
2. **T1 → T2 → T7 → T3 → T11** — 最薄的可用切片：网关可以与远程注册表通信并返回索引。Flutter 仍显示 M3 本地目录回退；尚无用户可见的更改。
3. **T4 → T6** — HTTPS 安装路径连接。此时 `marketplace://opendray-examples/fs-readme@1.0.0` 可以从真实注册表工作（在 T5 之前使用 Ed25519 开发者绕过）。
4. **T5 → T10** — 签名 + 信任传播。
5. **T8 → T9 → T21** — 甚至在 UI 迁移之前就发布撤销基础设施，以便首先保护现有用户。
6. **T18 → T19 → T20** — Flutter 迁移。此时中心 (Hub) 对接真实注册表上线。
7. **T12** — 设置 → 市场管理子页面。
8. **T24** — 市场仓库 PR 验证任务 (SHA 检查, 模式验证, 能力差异)。
9. **T25 → T27** — 随着每个切片落地补充测试；T27 (撤销 E2E) 是最高价值的验收测试。
10. **T28 → T29** — 文档 + 发布说明。

**M4.2 (v1 发布后):** 按顺序执行 T13 → T17。

## 6. "M4.1 已完成" 的验收标准

- 端到端：Kev 手动编辑 `github.com/Opendray/opendray-marketplace` 上的 PR，添加带有真实签名产物 URL 的 `plugins/opendray-examples/fs-readme/1.0.0.json` → CI (T23/T24) 在合并时验证并重新生成 `index.json` → 第二台设备上的中心 (Hub) 在 10 分钟内显示该插件，并可通过一次点击安装，且 SHA-256 + Ed25519 签名验证通过。
- 撤销：在 `revocations.json` 中将已安装插件标记为 `uninstall` → 在轮询窗口内，客户端自动卸载并显示横幅。
- 信任徽章在每个中心卡片上渲染。从 `local:` 或裸绝对路径安装时显示 `sideloaded` 标签。
- 设置 → 市场显示注册表 URL、镜像列表、自动更新开关、刷新按钮和撤销日志。
- 网关 `/api/marketplace/registry` 响应携带 M3 本地目录所使用的相同 Entry 形状（Flutter 在线路上保持向后兼容）。
- 市场有线格式无 P0/P1 bug。

## 6.2 "M4.2 已完成" 的验收标准 (未来)

- `opendray plugin scaffold` 生成可用的清单。
- `opendray plugin validate` 在 `plugins/examples/*` 中的每个示例上退出码为 0。
- 第三方开发机执行 `opendray plugin publish` → fork 仓库 + 开启 PR → CI 通过 → 维护者合并 → 中心 (Hub) 显示该插件。
- DNS TXT 验证流程已记录并测试。

## 7. 安全说明 (承袭自 10-security.md §6)

- 对于 `trust: verified` 或 `official`，**必须进行签名验证**。这些信任级别若缺失或签名无效，将导致安装失败并显示清晰的消息。`community` 和 `sideloaded` 不强制要求签名。
- **发布者密钥轮换** — 发布者记录列出多个带有 `addedAt` / `expiresAt` 的 Ed25519 密钥。如果签名匹配任何未过期的密钥，则通过验证。
- **不会对注册表 URL 尝试 TLS 固定 (Pinning)** — 可配置的镜像使得固定变得脆弱，撤销列表检查 + PR 审计链才是真正的防御手段。
- **SSRF 保持拦截状态** — HTTPSSource.Fetch 使用与 `opendray.http.*` 相同的 `net.Dialer.Control` RFC1918 拒绝路径。注册表 + 产物 URL 必须解析为公网 IP（对测试有轻微不便 — 桩件使用 `file://` 而非 HTTP）。
- **撤销是建议性的。** 离线安装不会自动执行，但在下次网络接触时会显示警告。无强制拨号行为。

## 8. 非目标

- **无厂商运行的审批工作流。** 市场是一个通过 PR 评审的 Git 仓库。合并即发布。
- **无应用内发布 UI。** 发布是开发者工作流；应用用于安装。
- **无付费计费。** 不在 v1 范围内；属于 M7。
- **无一等公民级别的私有注册表支持。** 你已经可以将 `OPENDRAY_MARKETPLACE_REGISTRY_URL` 指向任何 URL；一等公民级别的私有市场特性属于 M7。

## 9. 相关文件路径

- `plugin/market/` — 新包 (部分复用了 M3 的 `plugin/marketplace` 作为 `local` 后端)。
- `plugin/install/source.go` — `HTTPSSource.Fetch` 的 ENotImpl → 已实现。
- `cmd/opendray-plugin/` — 新二进制文件。
- `gateway/plugins_market.go` — 新增 (最终取代 M3 的 `gateway/marketplace.go`；在迁移期间保持并行)。
- `app/lib/features/hub/hub_page.dart` — 重新连接至 `/api/marketplace/registry`。
- `docs/plugin-platform/09-marketplace.md` — 已经是权威规范（无需重写；在代码落地后对具体文件路径进行小幅更新）。

## 10. 开放风险 — 合约冻结

v1 合约冻结日期为 **2026-10-01**。M4 + M5 + iOS 评审必须在此窗口内完成。按 M3 的进度，M4 约需 4 周；迁移 17 个遗留插件 (即使每个 2-3 天) 加上 iOS 评审需 8-10 周。这导致没有任何缓冲。

**若进度滑坡的缓解措施：**

- 在不包含 T13-T17 (发布者 CLI) 的情况下发布 M4 — 所有插件在冻结后之前保持为第一方。用户仍可从市场仓库安装；仅第三方插件发布被推迟。
- 如果签名基础设施耗时过长，将 T8 + T9 (撤销) 与纯手动撤销 (Kev 在 LXC 客户端侧切换) 结合，而非远程轮询。

每月评审冻结日期。按照路线图 §Locked，仅当 "M1–M4 累计滑动超过一个日历月" 时才可延期。M3 按时发布，因此我们仍有完整的 M4 预算。

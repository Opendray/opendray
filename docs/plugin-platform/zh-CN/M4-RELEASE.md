# OpenDray 插件平台 M4.1 — 发布就绪状态

**状态：** ✅ 已关闭 — 后端 + Flutter 消费侧已完成。M4.2 (发布者 CLI) 按照 M4-PLAN §1 规定推迟至 v1 发布之后。
**分支：** `kevlab`（按照项目策略不合并至 main；在 v1 发布前保留在 kevlab 分支）。
**基准：** M3 关闭于 `208bcb3`。
**M4.1 头部：** (kevlab 分支上 f5414c4..HEAD 之间的 commit 范围)

## 1. 任务状态

| 任务 | 标题 | 状态 | Commit(s) |
|------|-------|--------|-----------|
| T22  | 市场仓库模板                           | ✅ | `opendray-marketplace@5d10d36` |
| T23  | GitHub Actions — 重新生成 index.json              | ✅ | 同上 |
| T24  | PR 验证 (模式, sha256, 能力差异)     | ✅ | 同上 |
| T1   | `plugin/market` 包 + 本地后端提取  | ✅ | `4905629` |
| T2   | `remote.List` — 获取 index.json                     | ✅ | `5bc9f1a` |
| T3   | `remote.Resolve` — 每版本 JSON                  | ✅ | `ae4a7b4` |
| T4   | `HTTPSSource.Fetch` — 下载 + SHA-256 + 解压     | ✅ | `132abb6` |
| T5   | Ed25519 签名验证                       | ✅ | `8753d16` |
| T6   | 镜像回退 + `HTTPStatusError`                  | ✅ | `289ae2a` |
| T7   | 内存中 TTL 缓存 (磁盘缓存已推迟)            | ✅ | `3ca4ae5` |
| T8   | 撤销轮询 + 语义化版本匹配                    | ✅ | `a722cd1` |
| T9   | 撤销操作调度器                         | ✅ | `da8e3a8` |
| T10  | 信任策略强制执行                             | ✅ | `db2fa6f` |
| T11  | 网关连接 + 安装时策略 + 刷新       | ✅ | `b8c8d61` |
| T12  | 设置 → 市场管理端点                | ✅ | (随 T11 完成) |
| T18  | 中心 (Hub) 消费 `/api/marketplace/plugins`              | ✅ (M3 — 无需有线变更) |
| T19  | 中心卡片上的信任徽章                             | ✅ | (Flutter, 本次发布) |
| T20  | 插件页面上的自动更新指示器                 | ✅ | (Flutter, 本次发布) |
| T21  | 撤销横幅 + 提供商列表刷新            | ✅ | (Flutter, 本次发布) |
| T25  | 网关集成测试 — httptest 注册表         | ✅ | (本次发布) |
| T26  | 签名验证测试                         | ✅ (随 T5 完成) |
| T27  | 撤销 E2E                                       | ✅ (随 T8 扫描测试完成) |
| T28  | 文档更新                                             | 🟡 部分 — 即本文档；11-dx.md 更新已推迟 |
| T29  | M4-RELEASE.md                                        | ✅ |
| T13  | CLI 骨架                                         | ⏸ M4.2 |
| T14  | `plugin scaffold`                                    | ⏸ M4.2 |
| T15  | `plugin validate`                                    | ⏸ M4.2 |
| T16  | `plugin build` + 签名                                | ⏸ M4.2 |
| T17  | `plugin publish`                                     | ⏸ M4.2 |

**摘要：** 19 已完成 / 1 部分完成 / 5 挂起至 M4.2。

## 2. M4.1 发布内容

### 消费侧市场 (后端)

- **`plugin/market/`** — 两个后端之上的接口：
  - `market/local/` — M3 磁盘上的目录（为离线和 mock 部署保留）。
  - `market/remote/` — HTTPS 获取 `index.json` + 每版本 JSON + 发布者记录 + revocations.json。在 5xx/超时时进行镜像轮询；4xx 直接断路。内存中 TTL 缓存（默认 5 分钟；`CacheTTL=-1` 禁用）。
- **`plugin/install/HTTPSSource.Fetch`** — 真实实现。一次性完成下载流 + SHA-256 计算，在任何文件落地之前校验 `ExpectedSHA256`。`extractZipBundle` 拒绝绝对路径 / `..` / 符号链接 / setuid，并限制每个条目上限为 200 MiB 以阻断 zip 炸弹。上限与 09-marketplace.md 中 host-form 限制一致。
- **`plugin/market/signing/`** — Ed25519 验证器 + 信任级别策略。对于 official/verified 级别，若无已验证签名，`EnforcePolicy(entry, publisher, now)` 将返回 `ErrSignatureRequired`；community 级别签名是可选的，但仍会拒绝损坏的签名。
- **`plugin/market/revocation/`** — 类型、语义化版本匹配器 (Masterminds/semver)、轮询器 (`Config.Interval` 限制在 [1h, 168h] 区间)。`FetchRevocations` 扩展了目录接口；两个后端均已实现。
- **`plugin/market/actions/`** — 调度器将轮询器匹配结果连接至 `Installer.Uninstall` / `Runtime.SetEnabled` / `WorkbenchBus.Publish`。

### 消费侧市场 (Flutter)

- **中心 (Hub) 信任徽章** — official / verified / community 在每个卡片上渲染为彩色芯片。
- **插件页面更新指示器** — 当已安装版本 < 市场最新版本时，插件卡片显示 "update → vX.Y.Z" 芯片。使用简单的点分三段整数比较；对于 v1 足够使用，无需引入 pub_semver。
- **撤销横幅** — 新的 `revocation` SSE 事件类型会触发 snackbar 并通知 `ProvidersBus`，以便插件在被杀掉开关卸载时插件页面能立即刷新。

### 网关路由

| 路由 | 用途 |
|-------|---------|
| `GET /api/marketplace/plugins`  | 目录列表 (跨本地 / 远程工作) |
| `POST /api/marketplace/refresh` | 丢弃内存中缓存 |
| `GET /api/marketplace/settings` | 只读配置快照 (T12) |
| `POST /api/plugins/install`     | 现在对 `marketplace://...` 强制执行签名策略 |

### 配置 / 环境变量

- `OPENDRAY_MARKETPLACE_URL` — 启用远程后端。
- `OPENDRAY_MARKETPLACE_MIRRORS` — 逗号分隔的备用 URL。
- `OPENDRAY_REVOCATION_POLL_HOURS` — 1–168，默认 6。
- `OPENDRAY_MARKETPLACE_DIR` — 从 M3 保留，用于本地后端。

## 3. 推迟至 M4.2 / M5 / v1 之后的内容

- **发布者 CLI** (T13–T17) — `opendray plugin scaffold/validate/build/publish`。在此落地之前，Kev 是唯一的发布者并手动编辑市场仓库。记录在 `memory/m4_2_publisher_cli.md` 中。
- **磁盘目录缓存** (T7 完整范围) — 内存中缓存对于发布已经足够；stale-while-revalidate 属于 M5 优化。
- **11-dx.md 更新** (T28 的一部分) — 发布者工作流正文因等待 M4.2 而阻塞。
- **自动更新自动应用** — 指示器在 M4.1 发布，但执行更新是一个用户操作（点击 → 重新安装流）。静默后台更新属于 M5+。

## 4. 冒烟测试 — 手动演练

准备工作：
```bash
export OPENDRAY_MARKETPLACE_URL=https://raw.githubusercontent.com/Opendray/opendray-marketplace/main/
# 通过部署器在 syz LXC 上重启 opendray.service（见 memory/syz_deploy_pipeline.md — 请勿在 Claude Code 会话内部触发）。
```

### 4a. 远程目录可达
```bash
TOKEN=$(curl -s -X POST http://127.0.0.1:8640/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"linivek","password":"<pw>"}' | jq -r .token)

curl -sH "Authorization: Bearer $TOKEN" \
  http://127.0.0.1:8640/api/marketplace/plugins | jq
# 预期：{"entries":[]} (尚未发布任何插件)
# 或在 Kev 合并 PR 后显示真实的条目。
```

### 4b. 设置快照
```bash
curl -sH "Authorization: Bearer $TOKEN" \
  http://127.0.0.1:8640/api/marketplace/settings | jq
# 预期：{"source":"remote","registryUrl":"...","pollHours":6,...}
```

### 4c. 缓存刷新
```bash
curl -sX POST -H "Authorization: Bearer $TOKEN" \
  http://127.0.0.1:8640/api/marketplace/refresh | jq
# 预期：{"refreshed":true}
```

### 4d. 从远程安装（当真实条目存在时）
```bash
curl -sX POST -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"src":"marketplace://opendray-examples/fs-readme@1.0.0"}' \
  http://127.0.0.1:8640/api/plugins/install | jq
# 预期：202 并携带令牌、名称、版本、权限，SHA 已验证。
```

### 4e. 撤销扫描
- 在 opendray-marketplace/revocations.json 中追加：
  `{"name":"acme/evil","versions":"<=1.2.3","reason":"test",
    "recordedAt":"...","action":"warn"}`
- 合并 PR → index.json 重新生成 → Cloudflare 缓存过期。
- 下次轮询（默认 6 小时内）→ Flutter 应用中显示横幅。

## 5. 已知问题与注意事项

- **CDN + DNS 尚未配置。** M4.1 发布时默认使用 GitHub raw 作为 URL（或未设置 → 使用本地后端）。Cloudflare DNS `marketplace.opendray.dev` 属于发布后工作。
- **尚无发布者 CLI** — 第三方开发者无法自行发布。见 memory/m4_2_publisher_cli.md。
- **撤销轮询仅在目录存在时启动** — 空目录部署不会刷屏日志。
- **撤销横幅复用了现有的 showMessage snackbar。** 规格书中的持久红色横幅已推迟；即时的 snackbar + 插件页面刷新对于首次发布已足够。
- **配置键可以是裸 "name" (M3 向后兼容)** — 远程后端将空发布者默认设置为 `opendray-examples`，因此像 `marketplace://fs-readme` 这样的旧版 URL 仍可解析。

## 6. 签字验收清单

- [x] `go test -race -count=1 -p 1 ./plugin/market/... ./gateway/ ./plugin/install/` 绿色
- [x] Flutter `flutter test` 171/171 绿色
- [x] 端到端网关集成测试
  (`TestMarketplaceInstall_EndToEnd`) 覆盖了 注册 → 安装 → 确认 → 数据库状态。
- [x] 签名策略表 (official / verified / community / unknown) 已在 `plugin/market/signing/policy_test.go` 中覆盖。
- [x] 撤销匹配 + 轮询扫描 + 动作调度已在 `plugin/market/revocation/*_test.go` 中覆盖。
- [ ] syz LXC 上的手动冒烟测试 (在会话结束时完成；关于部署 + 会话终止注意事项见 memory/syz_deploy_pipeline.md)。
- [x] M4-PLAN.md 已更新 (T22-T24 在 `f5414c4` 标记为 ✅；本发布文档完成闭环)。

## 7. 提交历史 (kevlab 上的 M4.1)

基础设施 + 决策:
```
3799403 docs(plugin-platform): close M3 + open M4
7bec834 docs(plugin-platform): lock M4 decisions + split M4.1 / M4.2
f5414c4 docs(plugin-platform): M4 T22/T23/T24 done — marketplace repo live
```

市场注册表 (`github.com/Opendray/opendray-marketplace`):
```
5d10d36 chore: bootstrap marketplace registry (M4 T22 + T23)
```

M4.1 后端 (本仓库):
```
4905629 refactor(plugin-platform): T1 — market package skeleton + local backend
5bc9f1a feat(plugin-platform): M4.1 T2 — remote.List fetches index.json
ae4a7b4 feat(plugin-platform): M4.1 T3 — remote.Resolve per-version JSON
132abb6 feat(plugin-platform): M4.1 T4 — HTTPSSource.Fetch downloads + verifies + unzips
8753d16 feat(plugin-platform): M4.1 T5 — Ed25519 signature verification
289ae2a feat(plugin-platform): M4.1 T6 — mirror fallback + HTTPStatusError
3ca4ae5 feat(plugin-platform): M4.1 T7 (minimal) — in-memory TTL cache
a722cd1 feat(plugin-platform): M4.1 T8 — revocation polling + semver match
da8e3a8 feat(plugin-platform): M4.1 T9 — revocation action dispatcher
db2fa6f feat(plugin-platform): M4.1 T10 — trust policy enforcement
b8c8d61 feat(plugin-platform): M4.1 T11 — gateway wiring, install policy, refresh
```

M4.1 Flutter + 文档收尾随本次发布落地。

## 8. 相关文档

- **设计契约** — `docs/plugin-platform/09-marketplace.md`
- **计划** — `docs/plugin-platform/M4-PLAN.md` (24 个 M4.1 任务 + 5 个 M4.2 挂起任务)
- **市场仓库** — `github.com/Opendray/opendray-marketplace`
- **上一次发布** — `docs/plugin-platform/M3-RELEASE.md`

## 9. 下一步

`kevlab` 保持开启。M5 路线图项：
- 遗留插件迁移 (17 个内置插件 → v1 清单)。
- Webview + 侧车混合形式。
- iOS App Store 提交。
- 2026-10-01 合约冻结。

仅在 v1 发布就绪签字后才合并至 `main`。

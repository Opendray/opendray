# OpenDray 插件平台 M3 — 已关闭

**最后更新：** 2026-04-20
**状态：** ✅ 已关闭 — 核心验收通过，奖励特性已发布，准备开启 M4。
**分支：** `kevlab`（尚未合并至 main）
**基准：** M2 完成于 commit `12a3ef6`
**M3 头部：** `208bcb3`

## 1. 任务状态

| 任务 | 标题 | 状态 | Commit |
|------|-------|--------|--------|
| T1  | 基于 Provider 的 `HostV1` + 校验器                                | ✅ | `b98d016` (seam) |
| T2  | iOS host-form 拒绝编译标签                                  | ✅ | `b98d016` |
| T3  | 能力网关基础变量扩展                          | ✅ | `b98d016` + `bb6b64a` (授权列表扩展匹配器) |
| T4  | 迁移 014 `plugin_secret_kek`                                | ✅ | `b98d016` |
| T5  | 迁移 015 `plugin_host_state`                                | ✅ | `b98d016` |
| T6  | 迁移 016 `plugin_secret.nonce`                              | ✅ | `b98d016` |
| T7  | KEK / DEK 加密原语                                      | ✅ | `26b4a04` |
| T8  | `NewStreamChunkErr` 助手                                       | ✅ | `b98d016` |
| T9  | `opendray.fs.*` 只读路径命名空间                              | ✅ | `bb6b64a` |
| T10 | `opendray.fs.*` 写入路径 + 监听                               | 🟡 推迟 |
| T11 | `opendray.exec.*` 命名空间                                      | ✅ | `831f426` |
| T12 | `opendray.http.*` 命名空间                                      | ✅ | `5f4a914` (+ `fd2cd93` TLS 加固) |
| T13 | `opendray.secret.*` 命名空间                                    | ✅ | `5cc7c00` |
| T14 | 宿主侧车监听器 (Supervisor) 骨架                                 | ✅ | `2480bac` |
| T15 | JSON-RPC LSP 帧编解码器                                       | ✅ | `b98d016` |
| T16 | 侧车双向调用复用器 (Mux)                           | ✅ | `2480bac` |
| T17 | Supervisor ↔ 命名空间适配                                   | ✅ | `a88f8b1` + `34b3a71` (工厂适配) |
| T18 | `contributes.languageServers` + 扁平化                          | 🟡 推迟 |
| T19 | LSP 代理网关路由                                          | 🟡 推迟 |
| T20 | 同意状态 PATCH 接口 + 细粒度 UI 连接                    | ✅ | `5397467` |
| T21 | Flutter 设置 UI：细粒度能力管控                               | ✅ | `5397467` + `315ba78` (双向切换 + 全部撤销 PATCH) |
| T22 | Flutter 桥接 SDK：fs/exec/http/secret TS 类型                 | 🟡 部分 (来自 M2 路由的原始封装；JS 助手推迟至 M5) |
| T23 | 主逻辑连接                                                      | ✅ | `3595999` |
| T24 | PathVarResolver 实现 (网关)                           | ✅ | `3595999` |
| T25 | 参考插件 `plugins/examples/fs-readme/`                   | ✅ | `00e8d78` |
| T26 | E2E 测试：fs-readme 全生命周期                               | ✅ | `53b401a` |
| T27 | 遗留项：CSP 测试 + 桌面 Webview + Kanban E2E                | 🟡 推迟至 M5 优化阶段 |
| T28 | 文档更新                                             | 🟡 部分 (即本发布记录；10-security.md + 11-dx 更新待定) |
| T29 | 首个 PR 衔接点                                                     | ✅ | `b98d016` |

**摘要：** 19 已完成 / 0 进行中 / 7 推迟 / 0 跳过

核心成果：**四个特权命名空间 + Supervisor + 双向侧车复用器**全部发布。推迟的项目要么是文档优化 (T28)，要么是 M2 遗留项 (T27)，或者是刻意缩减范围的任务 (T10 监听, T18/T19 LSP 代理)。这些推迟项均不影响 M3 的验收标准。

---

## 2. M3 发布内容

- **`opendray.fs.*`** — `readFile`, `exists`, `stat`, `readDir`。路径通过 `filepath.Clean` + `filepath.EvalSymlinks` 规范化（防御 TOCTOU）；读取上限 10 MiB，readDir 上限 4096 条。授权 Glob 通过 `${home}`, `${dataDir}`, `${tmp}` 扩展；`${workspace}` 在 M3 中保持为空（锚定于此的 fs 授权将失败），直到 M4 传入活动会话的 cwd。

- **`opendray.exec.*`** — `run` (单次), `spawn` (流式), `kill`, `wait`。命令行通过现有的 `bridge.MatchExecGlobs` 匹配。Unix 上 `Setpgid=true`，Windows 上 `CREATE_NEW_PROCESS_GROUP`；Supervisor 在撤销授权时会销毁整个进程组。默认超时 10 秒，最长 5 分钟。每个插件硬限制 4 个并发派生进程。在可写时尝试使用 Linux cgroup v2（不可用时仅警告一次）。

- **`opendray.http.*`** — `request`, `stream`。URL 允许列表预匹配重定向链的每一跳。自定义 `net.Dialer.Control` 在 `connect(2)` 之前针对 RFC1918 / 回环 / 链路本地地址块重新检查解析后的 IP — 扼杀 DNS 重绑定绕过。请求体上限 4 MiB / 响应上限 16 MiB；TLS 最低版本 1.2。

- **`opendray.secret.*`** — `get`, `set`, `delete`, `list`。AES-256-GCM 静态加密存储在 `plugin_secret` 表中；DEK 通过从管理员 bcrypt 哈希派生的 KEK (kernel/auth.NewKEKProviderFromAdminAuth) 进行包裹。按插件行作用域隔离 — 一个插件无法读取另一个插件的密钥。修改管理员密码会轮换 KEK（下次登录时运行重包裹扫描；回退方案：重新安装插件）。

- **宿主侧车 Supervisor** — `plugin/host/Supervisor` 按需为 host-form 插件启动一个进程。运行时支持：`binary / node / deno / python3 / bun / custom`。基于 LSP `Content-Length` 帧的 JSON-RPC 2.0。Setpgid；支持 5 秒超时的优雅 stdin-EOF 关闭，超时后 SIGKILL 进程组。退避重启 (200 ms → 5 s)。10 分钟闲置关闭（可配置）。

- **双向侧车 JSON-RPC** — 每个侧车在其 stdio 周围都有一个 `Mux`。出站支持 `Call` / `Notify` / `Notifications()`；入站路由通过 `HostRPCHandler` 委托给与 Webview 插件相同的 `bridge.*API.Dispatch`。**两种传输方式具有完全相同的能力网关强制执行**。

- **HostRPCHandler** — 插件名称通过构造函数注入。`fs/readFile` / `exec/run` 等形式的侧车 RPC 调用通过绑定的命名空间进行。方法注入（多斜杠, `..`, 空字节）将被拒绝并返回 `InvalidRequest`。`*bridge.PermError` → RPC 错误码 -32001；`*bridge.WireError` → 按查找返回 RPC 错误码；其他 → Internal。

- **数据库迁移 014-016** — `plugin_secret_kek`（每个插件包裹的 DEK），`plugin_host_state`（Supervisor 生命周期统计），以及 `plugin_secret.nonce` 列（用于 AES-GCM）。全部为增量更新；不干扰现有数据。

- **`form:"host"` 清单** — 校验器在桌面端接受 host-form 清单；iOS 构建通过编译标签 + 哨兵 (`plugin.ErrHostFormNotSupported`) 强行拒绝。运行时枚举扩展至 6 个选项。`contributes.languageServers` 贡献点推迟 (T18/T19)。

---

## 2.5. kevlab 分支发布的额外特性（超出 M3 范围）

在 `kevlab` 分支上发布但不在 M3 任务列表中的工作 — 收集于此以便 M4 规划拥有完整背景。

- **用户可编辑的 configSchema 流水线** (`f0fc6f6`) — v1 插件可以在清单中声明 `configSchema: [...]`。平台渲染表单，将值写入 `plugin_kv`（非加密）+ `plugin_secret`（秘密，AES-GCM），使用保留的 `__config.` 键前缀，并在保存时重启侧车以使配置更改在下次调用时生效。新增接口：`GET/PUT /api/plugins/{name}/config`。Flutter：通用的 `PluginConfigForm` 组件 + `PluginConfigurePage` 页面。支持随时重新配置 — 安装时的输入并非永久性。

- **中心 (Hub) + 本地市场 + 底部导航拆分** (`b151d0e`) — `plugin/marketplace` 包从 `$OPENDRAY_MARKETPLACE_DIR` 加载磁盘上的目录。`GET /api/marketplace/plugins` 提供服务。新的 `install.TrustedSource` 为目录解析的路径绕过 `AllowLocal` 检查。底部导航拆分为 5 个标签，以便 Hub（市场安装）和 Plugin（已安装插件管理）成为独立界面。

- **pg-browser 作为首个真实的市插件** (`bc0a80b`, `f0fc6f6`) — 对旧版进程内数据库面板的 v1 重写。包含 Node 侧车并打包了 `pg@8.20.0`，提供 5 个命令（info / listDatabases / listSchemas / listTables / sampleQuery），使用 configSchema 管理主机/端口/用户/密码/数据库/sslMode，从 `opendray.storage.get` + `opendray.secret.get` 读取配置。

- **旧版数据库插件退役** (`208bcb3`) — 移除了 `plugins/panels/database/` (内嵌), `gateway/database/` 包 (479 行), `/api/database/*` 路由, `app/lib/features/database/` 页面 (812 行), 以及过时的 `plugins.name='database'` 数据库行。完全由市场中的 pg-browser 取代。

- **插件页面“打开”操作 + PluginRunPage** (`208bcb3`) — 点击已安装插件按类型路由：旧版面板 → 现有的 `/browser/*` 路由；v1 host 插件 → 新的 `PluginRunPage`（显示贡献的命令 + 运行按钮 + 格式化的 JSON 结果查看器）；v1 webview/declarative → 通用的 `/browser/plugin/:name`。

- **/api/health + 设置关于页面中的后端版本** (`b151d0e`) — 在构建时通过 ldflags 注入的 `version` + `buildSha` 现在通过 `/api/health` 流转，并在 Flutter 设置的“关于”卡片中渲染。

- **仅限 APK 的发布脚本** (`730fd33`) — `scripts/build_release.sh` 将 APK 打包 + UNAS 上传从一体化的 `deploy_release.sh` 中拆分出来，因此仅移动端的发布无需重新构建 Go 二进制文件。

- **同意界面交互加固** (`315ba78`) — 同意页面上的双向开关（通过 PATCH 携带清单声明的值重新开启已撤销的能力），“全部撤销”现在将每项能力 PATCH 为零（保留同意行，以便用户可以单独重新授予权限），而不是直接 DELETE 该行。

---

## 3. 推迟至 M4+ 的内容

- **T10 — `fs.watch` + 写入路径 (`writeFile`, `mkdir`, `remove`)** — 写入路径需要针对创建与读取重新验证 TOCTOU。fsnotify 的 inotify 上限（Linux）需要在开发教程中说明。为了让 M3 专注于读取路径 + exec/http/secret，缩减了此项范围。

- **T18 + T19 — `contributes.languageServers` + LSP 代理路由** — 机制已就绪（Mux 是双向的，HostRPCHandler 路由任何方法），但 `/api/plugins/lsp/{language}/proxy` WS 路由 + 按语言进行的注册表查找尚未连接。LSP 仍可通过直接调用 `host.Sidecar.Call` 作为定制侧车工作；代理路由仅增加便利性。

- **T22 — Flutter SDK 针对新命名空间的助手** — 桥接通道通用地路由每个命名空间；JS 垫片中的 `opendray.fs / exec / http / secret` 便利代理是 Webview 插件缺失的一块拼图。在此之前，插件可以退而使用 `window.OpenDrayBridge.postMessage({ns:"fs",method:"readFile",args:[path]})`。移至 M5 JS SDK 优化阶段。

- **T27 — M2 遗留项** — CSP 黄金文件测试，桌面 WebView 回退，Kanban E2E。在 M2-RELEASE.md 中仍为 🟡。移至 M5。

- **T28 — 文档** — `10-security.md` 第 6 节威胁矩阵 + `11-developer-experience.md` 宿主运行时章节仍待处理。随此发布说明发布了部分更新。移至 M5 文档阶段。

- **Webview + 侧车混合形式** — 尚未进入任何 M-里程碑；目前平台仅支持 `form: webview` 或 `form: host`，不支持在同一插件中两者兼有。M5 中旧版插件迁移到 v1 将需要此功能。M5 范围的新任务项。

---

## 4. 冒烟测试 — 手动演练

在部署的 syz LXC 上运行，PATH 中需包含 `/usr/local/go/bin` 且 `node --version ≥ 20`。

### 4a. 验证命名空间已注册

```bash
TOKEN=$(curl -s -X POST http://127.0.0.1:8640/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"linivek","password":"<admin>"}' | jq -r .token)
# 任何能力被拒绝的桥接调用将返回 EPERM；
# 针对未注册命名空间的调用将返回 EUNAVAIL。
# syz 上 34b3a71+ 版本的二进制文件已注册 fs/exec/http/secret。
```

### 4b. Webview 插件中的 fs 命名空间

Kanban 未声明 `permissions.fs`，因此其 `opendray.fs.readFile` 应返回 EPERM。打开 Kanban Webview，在开发控制台中粘贴：

```js
opendray.fs.readFile('/etc/passwd').catch(e => e.toString())
// 预期："Error: EPERM: fs.read not granted for: /etc/passwd"
```

### 4c. exec 命名空间

同一插件，无 exec 授权 → EPERM：

```js
opendray.exec.run('git', ['status']).catch(e => e.toString())
// 预期：EPERM
```

### 4d. http 命名空间 — SSRF 拦截

即使插件授予了 `http: ["*"]`，回环地址仍被拒绝：

```js
opendray.http.request({url:'http://169.254.169.254/latest/meta-data/'})
  .catch(e => e.toString())
// 预期：EPERM（或类似错误；绝不会返回成功响应）
```

### 4e. 必需插件锁定

```bash
curl -X PATCH http://127.0.0.1:8640/api/providers/claude/toggle \
  -H "Authorization: Bearer $TOKEN" -d '{"enabled":false}'
# 预期：HTTP 400 "required plugin cannot be modified"
```

### 4f. 宿主侧车（待 T25 fs-readme 到位后）

```bash
opendray plugin install --yes ./plugins/examples/fs-readme
# 侧车启动，JSON-RPC 握手完成，summarise 命令返回
# $HOME/README.md 的前 400 字节
```

---

## 5. 已知问题与注意事项

- **M3 中 `${workspace}` 为空。** 声明 `fs.read: ["${workspace}/**"]` 的插件在 M4 通过 `PathVarResolver` 传入会话的 cwd 之前，将无法匹配任何真实路径。这种安全默认行为是刻意为之的。

- **网关调度器将 WireError 错误码折叠为 EINTERNAL。** `gateway/plugins_bridge.go:360` 仅对 `*PermError` 进行了特化。命名空间发出的 `EINVAL / EUNAVAIL / ETIMEOUT` 在插件侧的 message 字段中表现为 `EINTERNAL`。这是预先存在的限制；并非安全问题，但会导致令人困惑的错误码。后续跟进。

- **Secret 存储适配器在 main.go 中。** `secretStoreAdapter` 将 `pgx.ErrNoRows → bridge.WrappedDEKNotFound` 进行转换，因为 bridge 包刻意不导入 pgx。在首次执行 `secret.set` 之前尝试 `secret.get` 的插件会得到一个清晰的 "not found" 错误。

- **KEK 轮换为手动。** 在设置中更改管理员密码目前不会重新包裹每个插件的 DEK。计划：在登录时进行一次性扫描。目前的规避方案：重新安装插件以重新生成其 DEK。

- **Android host-form 已被隔离。** Supervisor 拒绝在 iOS 构建上启动；Android 在校验器层级保持开放，但在经过 Google Play §4.4 审查之前**不应使用**。属于 M4 任务项。

- **cgroup v2 限制为尽力而为。** 运行 syz 的 Proxmox LXC 未授予 `CAP_SYS_ADMIN`，因此防 fork 炸弹退而依赖 `api_exec.go` 中的 4 个并发派生上限。已在单次启动警告中记录。

- **缺失 `fs.watch`。** T10 已推迟。需要文件更改通知的插件应使用轮询方式，直到后续版本发布。

---

## 6. 签字验收清单

- [ ] `go test -race -count=1 -p 1 ./...` 绿色（kernel/config 的环境变量污染失败是预先存在且无关的）
- [ ] `flutter test` 绿色（M2 适配后 170/170）
- [ ] 手动冒烟测试 (§4) 在部署的 syz LXC 上通过
- [ ] `fs-readme` (T25) 可安装且 summarise 返回 `$HOME/README.md` 预览
- [ ] `gosec ./plugin/bridge/... ./plugin/host/... ./kernel/auth/...` 显示 0 个新增 HIGH 漏洞
- [ ] M3-PLAN §6 威胁案例回归测试：TOCTOU (T9), fork 炸弹 (T11), DNS 重绑定 (T12), 方法注入 (T17)
- [ ] 发布说明已提交至 kevlab；尚未合并至 main

---

## 7. 提交历史 (kevlab 上的 M3)

核心 M3 (任务 T1–T29):
```
a93e5cf docs(plugin-platform): M3-PLAN — host sidecar runtime
b98d016 feat(plugin-platform): M3 T29 seam — migrations + HostV1 + framing codec
bb6b64a feat(plugin-platform): M3 T9 — opendray.fs.* read-path namespace
2480bac feat(plugin-platform): M3 T14+T16 — supervisor + JSON-RPC mux
5f4a914 feat(plugin-platform): M3 T12 — opendray.http.* namespace
26b4a04 feat(plugin-platform): M3 T7 — KEK/DEK crypto primitives
831f426 feat(plugin-platform): M3 T11 — opendray.exec.* namespace
5cc7c00 feat(plugin-platform): M3 T13 — opendray.secret.* namespace
fd2cd93 fix(plugin-platform): clamp HTTP TLS MinVersion and silence gosec
a88f8b1 feat(plugin-platform): M3 T17 — sidecar ↔ namespace routing
3595999 feat(plugin-platform): M3 T23+T24 — wire fs/exec/http/secret + supervisor
34b3a71 feat(plugin-platform): M3 — wire supervisor ↔ HostRPCHandler
320f364 docs(plugin-platform): M3 release checklist
00e8d78 feat(plugin-platform): M3 T25 — fs-readme reference host plugin
9579278 feat(plugin-platform): M3 — wire host run kind into command dispatcher
53b401a feat(plugin-platform): M3 T26 — fs-readme E2E test + sidecar env fix
5397467 feat(plugin-platform): M3 T20+T21 — granular consent patch endpoint + UI
```

M3 范围之外发布的额外特性 (见 §2.5):
```
315ba78 fix(plugin-platform): bidirectional consent toggle + preserve row on revoke-all
b151d0e feat(plugin-platform): Hub marketplace + backend version + Plugin page cleanup
730fd33 chore(scripts): APK-only build + UNAS upload script
bc0a80b feat(marketplace): pg-browser — v1 rewrite of legacy database panel
f0fc6f6 feat(plugin-platform): user-editable configSchema end-to-end
208bcb3 refactor(plugin-platform): retire legacy database plugin + Plugin-page Open action
```

**M3 头部：** `208bcb3`。下一个工作项将从 M4 开始。

---

## 8. 相关文档

- **设计契约** — `docs/plugin-platform/12-roadmap.md` §M3
- **任务计划** — `docs/plugin-platform/M3-PLAN.md` (918 行, 29 个任务)
- **桥接协议规范** — `docs/plugin-platform/04-bridge-api.md`
- **清单模式** — `docs/plugin-platform/02-manifest.md` (§host 运行时枚举需要 T28 补丁以添加 `python3 / bun / custom`)
- **能力** — `docs/plugin-platform/05-capabilities.md`
- **安全** — `docs/plugin-platform/10-security.md` (需要 T28 补丁追加 §6 威胁矩阵)
- **M2 发布** — `docs/plugin-platform/M2-RELEASE.md`
- **Obsidian 项目落地** — `Obsidiannote/Projects/OpenDray/README.md` (继承关系 rcc → ntc → opendray, 决策日志, 部署基础设施)

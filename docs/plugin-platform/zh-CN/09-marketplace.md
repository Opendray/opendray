# 09 — 插件市场 (Marketplace)

插件市场是一个 **Git 仓库** (`github.com/opendray/marketplace`)，任何人都可以 fork 该仓库，其 `main` 分支是权威的注册表。插件构建产物（Artifacts）托管在发布者的基础设施上（如 GitHub Releases、CDN 等）；注册表仅保存元数据和完整性哈希值。

> **已锁定：** 使用 Git 仓库作为注册表，并基于 PR 进行审批。无需运行中央服务器，无供应商锁定，且拥有免费的审计追踪。

## 仓库布局

```
opendray/marketplace/
  index.json                         # 根注册表 —— 稳定 URL
  plugins/
    acme/
      hello/
        meta.json                    # 发布者 + 描述 + 最新版本
        1.0.0.json                   # 各版本元数据 (哈希, URL, 清单)
        1.1.0.json
  publishers/
    acme.json                        # 已签名的发布者记录
  revocations.json                   # 禁用开关条目
  CODEOWNERS                         # 审核路由
```

## `index.json` (根索引) 模式

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "OpenDray Marketplace Index",
  "type": "object",
  "required": ["version", "generatedAt", "plugins"],
  "properties": {
    "version":     { "const": 1 },
    "generatedAt": { "type": "string", "format": "date-time" },
    "plugins": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["name", "publisher", "latest", "path"],
        "properties": {
          "name":        { "type": "string" },
          "publisher":   { "type": "string" },
          "displayName": { "type": "string" },
          "description": { "type": "string", "maxLength": 280 },
          "icon":        { "type": "string" },
          "categories":  { "type": "array", "items": { "type": "string" } },
          "keywords":    { "type": "array", "items": { "type": "string" } },
          "latest":      { "type": "string", "description": "语义化版本 (semver)" },
          "path":        { "type": "string", "description": "指向插件目录的相对路径" },
          "trust":       { "enum": ["official","verified","community"] },
          "downloads":   { "type": "integer" },
          "stars":       { "type": "integer" }
        }
      }
    }
  }
}
```

市场仓库中的 CI 任务会在每次推送到 `main` 时重新生成 `index.json`。客户端获取该 **唯一的** URL 来填充浏览界面。

## 版本元数据文件 (`plugins/<publisher>/<name>/<version>.json`)

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["name","publisher","version","artifact","sha256","manifest"],
  "properties": {
    "name":      { "type": "string" },
    "publisher": { "type": "string" },
    "version":   { "type": "string" },
    "releaseNotes": { "type": "string" },
    "artifact": {
      "type": "object",
      "required": ["url","size"],
      "properties": {
        "url":  { "type": "string", "format": "uri" },
        "size": { "type": "integer", "description": "字节数" },
        "mirrors": { "type": "array", "items": { "type": "string", "format": "uri" } }
      }
    },
    "sha256":    { "type": "string", "pattern": "^[a-f0-9]{64}$" },
    "signature": {
      "type": "object",
      "properties": {
        "alg":       { "enum": ["minisign","ed25519","sigstore"] },
        "publicKey": { "type": "string" },
        "value":     { "type": "string" }
      }
    },
    "manifest":  { "$ref": "https://opendray.dev/schemas/plugin-manifest-v1.json" },
    "engines":   { "$ref": "#/$defs/engines" },
    "platforms": { "type": "array",
                   "items": { "enum": ["linux-x64","linux-arm64",
                                       "darwin-x64","darwin-arm64",
                                       "windows-x64","any"] } }
  }
}
```

**规则：** `manifest` 字段是插件包中 `manifest.json` 的完整副本。客户端使用此字段渲染许可界面，而无需先下载整个 zip 包。

## 发布者记录 (`publishers/<publisher>.json`)

```json
{
  "name": "acme",
  "displayName": "Acme Inc.",
  "homepage": "https://acme.example",
  "trust": "verified",
  "keys": [
    { "alg": "ed25519", "publicKey": "base64...", "addedAt": "2025-...", "expiresAt": "2027-..." }
  ],
  "domainVerification": { "method": "dns-txt", "record": "opendray-verify=..." }
}
```

## 插件发布构建产物 (发布者服务器上的 zip 包)

### 必需内容
```
<root>/
  manifest.json                # 必须与市场中的版本记录匹配
  LICENSE
  ui/                          # 当 form=webview 时
  bin/                         # 当 form=host 时
  README.md                    # 强烈建议包含
  CHANGELOG.md                 # 强烈建议包含
```

### 规则
- 使用 zip，不使用 tar。便于跨平台工具使用。
- Web 视图最大 20 MB，宿主插件最大 200 MB。
- 禁止包含带有 setuid/setgid 位的可执行文件。
- 禁止包含指向根目录外部的符号链接。
- 根目录下的清单哈希必须等于注册表中声明的清单哈希；不匹配则中止安装。

## 发布流程 (Publish flow)

1. 插件作者通过 SDK 运行 `opendray plugin publish`。
2. CLI 工具执行以下操作：
   - 根据 v1 模式验证清单。
   - 构建 zip 包，计算 sha256，如果配置了密钥则进行签名。
   - 将产物上传到用户配置的端点（默认 GitHub Release）。
   - 如果需要，fork 市场仓库。
   - 创建分支，添加 `<version>.json` 并更新 `meta.json`。
   - 提交 PR，正文使用模板格式（包含变更日志、能力差异、截图）。
3. 市场 CI 运行：
   - 模式验证。
   - 对产物 URL 进行 SHA 检查。
   - 对插件包进行沙箱扫描（静态检查禁止的文件类型 / 可疑的二进制文件）。
   - 计算与前一版本的能力差异。
4. CODEOWNERS 审核。
5. 合并 → 重新生成 `index.json` → 客户端在轮询窗口内（默认 1 小时；用户可强制刷新）看到该插件。

## 审核标准 (PR 合并阈值)

- 清单符合 v1 模式。
- 产物可从镜像下载且 sha256 验证通过。
- 插件包中无硬编码的凭据（机密扫描器）。
- 插件包大小低于对应形式的上限。
- 图标存在且可渲染。
- PR 描述中对较上一版本新增的能力进行了说明。
- 对于“已验证 (verified)”发布者：签名可通过已注册的发布者密钥验证。
- 对于“官方 (official)”发布者：仅限 OpenDray 维护者可以修改 `publishers/opendray.json`。

## 信任等级 (Trust levels)

| 等级 | 含义 | 授予者 |
|-------|-------|-----------|
| `official` (官方) | 由 OpenDray 核心团队维护 | 手动编辑 `publishers/opendray.json` |
| `verified` (已验证) | 发布者通过了域名 / 身份验证并注册了密钥 | 市场维护者将 `publishers/<name>.json` 设为 `trust: "verified"` |
| `community` (社区) | 任何已合并 PR 的作者 | 默认值 |
| `sideloaded` (侧加载) | 从 URL / 本地路径安装，未通过市场 | 客户端标签 |

安装程序在许可界面中显示信任等级。

## 禁用开关 / 撤销 (Kill-switch / revocation)

`revocations.json`:
```json
{
  "version": 1,
  "entries": [
    { "name":"acme/evil",
      "versions":"<=1.2.3",
      "reason":"凭据外泄",
      "recordedAt":"2025-...",
      "action":"uninstall" }     // "uninstall" (卸载) | "disable" (禁用) | "warn" (警告)
  ]
}
```

客户端每 6 小时（以及在应用启动时）轮询此文件。匹配时：
- `uninstall` — 自动卸载并显示横幅。
- `disable` — 将 enabled 状态设为 false。
- `warn` — 显示红色横幅，插件继续工作直到用户采取行动。

> **已锁定：** 撤销操作是建议性的。离线安装不会自动执行操作，但在下次联网时会看到警告。我们不提供“插件持续向母站汇报”的功能。

## 镜像支持

`index.json` 和各版本的 JSON 文件必须能从至少两个 URL（在客户端设置中配置）公开访问。默认镜像为：`https://raw.githubusercontent.com/opendray/marketplace/main/…` 以及一个由 Cloudflare 代理的副本。客户端以轮询方式尝试镜像。

## 用户可见的设置

设置 → 市场：
- **注册表 URL** (默认为官方地址)。
- **自动更新插件** (默认关闭；在能力扩大的更新时会提示)。
- **允许社区插件** (默认开启；在受限安装环境下可关闭)。
- **立即刷新缓存** 按钮。

## Go 包所有权

- `plugin/market/` (新增) — 获取/缓存 `index.json`，解析 URL，验证 sha256/签名。
- `gateway/plugins_market.go` (新增) — 浏览/搜索/安装的 REST 路由。

## v1 不包含的内容

- 市场内的评分 / 评论。
- 付费插件 / 计费。
- 私有市场（目前可以通过将注册表 URL 指向私有仓库来实现，但无显式功能支持）。

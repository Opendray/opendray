# OpenDray 插件平台 v1

OpenDray 是**为 vibe coding 打造的移动端 VS Code**。插件平台允许任何开发者发布插件，并在不重新构建 Flutter 应用或 Go 后端的情况下，热安装到正在运行 Hendrick 实例中。

该目录是 **v1 设计合约**。下文中的所有模式、API 和行为都是插件作者可以依赖的。破坏性变更需要主版本升级和弃用窗口。

## 三层架构图

```
+--------------------------------------------------------+
| 工作台外壳 (Workbench Shell)   Flutter 应用, 固定 UI     |  <-- 第 1 层
|  - 渲染贡献点 (插槽)                                    |
|  - 拥有导航、主题、手势、无障碍 (a11y)                  |
+----------------------+---------------------------------+
                       |  HTTPS / WSS (localhost 或 LAN)
+----------------------v---------------------------------+
| 插件宿主 (Plugin Host)       Go 后端, 固定              |  <-- 第 2 层
|  - 安装 / 生命周期 / 能力网关                           |
|  - 桥接服务器 (opendray.* API)                          |
|  - HookBus / 事件 / 任务 + 会话所有权                   |
+----------------------+---------------------------------+
                       |  stdio JSON-RPC  /  JS bridge
+----------------------v---------------------------------+
| 插件 (Plugins)           下载的包                       |  <-- 第 3 层
|  声明式  |  WebView  |  宿主 (侧车/sidecar)             |
+--------------------------------------------------------+
```

## 文档

| # | 文档 | 用途 |
|---|-----|---------|
| 01 | [architecture.md](01-architecture.md) | 层职责、进程模型、数据流图 |
| 02 | [manifest.md](02-manifest.md) | 完整的 `manifest.json` JSON Schema v1 |
| 03 | [contribution-points.md](03-contribution-points.md) | 所有 UI 和扩展插槽 |
| 04 | [bridge-api.md](04-bridge-api.md) | `opendray.*` API 参考 |
| 05 | [capabilities.md](05-capabilities.md) | 权限分类和授权模型 |
| 06 | [plugin-formats.md](06-plugin-formats.md) | 声明式 / WebView / 宿主 — 三选一 |
| 07 | [lifecycle.md](07-lifecycle.md) | 安装、激活、更新、卸载 |
| 08 | [workbench-slots.md](08-workbench-slots.md) | UI 插槽目录及线框图 |
| 09 | [marketplace.md](09-marketplace.md) | 注册表、发布、熔断机制 |
| 10 | [security.md](10-security.md) | 威胁模型、iOS 方案、信任等级 |
| 11 | [developer-experience.md](11-developer-experience.md) | SDK、脚手架、热重载 |
| 12 | [roadmap.md](12-roadmap.md) | M1..M7 里程碑和合约冻结 |

## Hello world — 每种形式各一个

### 声明式 (纯 manifest)
```json
{
  "$schema": "https://opendray.dev/schemas/plugin-manifest-v1.json",
  "name": "hello-decl", "version": "0.1.0", "publisher": "acme",
  "form": "declarative",
  "engines": { "opendray": "^1.0.0" },
  "contributes": {
    "commands": [{ "id": "hello.say", "title": "Say Hello",
                   "run": { "kind": "notify", "message": "Hi from plugin!" } }],
    "statusBar": [{ "id": "hello.bar", "text": "hello", "command": "hello.say" }]
  }
}
```

### WebView (manifest + 静态 `ui/` 包)
```json
{
  "name": "hello-webview", "version": "0.1.0", "publisher": "acme",
  "form": "webview",
  "engines": { "opendray": "^1.0.0" },
  "ui": { "entry": "ui/index.html" },
  "contributes": {
    "views": [{ "id": "hello.view", "title": "Hello",
                "container": "activityBar", "icon": "icons/hello.svg" }]
  },
  "permissions": { "storage": true }
}
```
`ui/index.html` (包内部) 调用 `window.opendray.workbench.showMessage('hi')`。

### 宿主 (manifest + 侧车/sidecar)
```json
{
  "name": "hello-lsp", "version": "0.1.0", "publisher": "acme",
  "form": "host",
  "engines": { "opendray": "^1.0.0" },
  "host": {
    "entry": "bin/hello-lsp",
    "platforms": { "linux-x64": "bin/linux-x64/hello-lsp",
                   "darwin-arm64": "bin/darwin-arm64/hello-lsp" },
    "protocol": "jsonrpc-stdio"
  },
  "activation": ["onLanguage:hello"],
  "contributes": {
    "languageServers": [{ "id": "hello.lsp", "languages": ["hello"] }]
  },
  "permissions": { "exec": false, "fs": { "read": ["${workspace}/**"] } }
}
```

## 决策图例

- `> **Locked:**` — v1 已冻结。更改它是破坏性变更，需要 v2。
- `> **Open:**` — 未解决，必须由 Kev 选择。父编排器应跟踪这些。
- `post-v1` — 明确推迟。暂时不要进行模式化 (schematise)。

## 兼容性承诺

目前加载的所有 manifest（`plugins/agents/*` 和 `plugins/panels/*` 下的 6 个 agent 和 11 个面板）都将继续保持不变地加载。v1 是当前 `plugin.Provider` 形状的**严格超集**。升级路径请参见 [lifecycle.md](07-lifecycle.md) §Compat。

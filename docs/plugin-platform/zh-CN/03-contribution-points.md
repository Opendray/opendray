# 03 — 贡献点 (Contribution Points)

每个贡献点都是工作台（Workbench）中的一个插槽。插件在 `contributes.*` 下声明条目；外壳（Shell）负责渲染它们。宿主在加载时会验证每个条目。

图例：**MVP** = 在 M5 v1 冻结版本中发布 · **post-v1** = 仅保留命名空间；除占位符外无模式定义。

---

## 1. `activityBar` (活动栏) — MVP

位于左侧边栏（平板电脑）或底部导航溢出区（手机）的图标。

**模式** (源自 manifest §activityBarItem):
```json
{ "id": "myext.activity", "icon": "icons/star.svg",
  "title": "My Ext", "viewId": "myext.view" }
```

**Flutter 渲染：**
- 平板横屏：垂直边栏，宽度 56px，悬停时显示工具提示。
- 手机竖屏：活动栏折叠；图标移动到“更多”面板中。

**限制：** 每个插件最多 4 个活动栏条目。全系统侧边栏容量上限为 16 个项目（超出部分进入“更多”）。
**图标：** 24x24 SVG 单色（颜色随主题变化）或表情符号。

---

## 2. `views` (视图) — MVP

托管在活动栏项目、侧边面板或底部面板中的可滚动视图容器。

**模式：** 参见 manifest §view。

**渲染：**
- `render: "webview"` → Flutter 嵌入一个指向 `plugin://<name>/<ui.entry>` 的 `WebView` 组件。
- `render: "declarative"` → 宿主推送一个树结构（参见 [bridge-api §ui](04-bridge-api.md#ui)），Flutter 将其映射为原生组件。简单列表请使用此方式。

**限制：** 每个插件最多 8 个视图。一个视图的声明式树最多可包含 500 个节点。

---

## 3. `panels` (面板) — MVP

底部抽屉面板（终端、日志、问题）。在手机上向上滑动，在平板电脑上显示为选项卡。

**模式：** 参见 manifest §panel。

**渲染：** 与视图相同，但固定在底部抽屉区域；除非声明式宿主方法返回内容，否则始终为 `render: "webview"`。

**限制：** 每个插件最多 4 个面板。

---

## 4. `commands` (命令) — MVP

可通过 ID 从按键绑定、菜单、状态栏、命令面板、Telegram 和 AI 调用的命名操作。

**模式：** 参见 manifest §command。操作类型：
- `host` — 调用插件边车方法 `method` 并传递 `args`。需要 `form: "host"`。
- `notify` — 显示带有 `message` 的气泡提示（Toast）。无需代码；纯声明式。
- `openView` — 聚焦 `viewId`。
- `runTask` — 运行贡献的 `taskId`。
- `exec` — 运行 shell（需要 `permissions.exec` 与参数匹配）。
- `openUrl` — 打开外部浏览器。

**示例：**
```json
{ "id": "myext.refresh", "title": "刷新",
  "icon": "icons/refresh.svg",
  "when": "viewFocused == 'myext.view'",
  "run": { "kind": "host", "method": "refresh", "args": [] } }
```

**限制：** 每个插件最多 64 个命令。ID 命名空间为 `<插件名>.<操作名>`。

---

## 5. `settings` (设置) — MVP

在“设置 → 插件 → <名称>”中渲染为表单的字段。是遗留 `configSchema` 的超集（形状相同，在 v1 路径中添加在 `contributes.settings` 下）。

**模式：** 参见 manifest §settingField。

**渲染：** 按 `group` 分组；`dependsOn` / `dependsVal` 驱动条件显示。`secret` 类型始终被掩码处理，并通 `secret` 能力存储，不以明文配置形式存储。

**限制：** 每个插件最多 32 个字段。

---

## 6. `statusBar` (状态栏) — MVP

底部状态栏中的文本 + 可选图标。

**模式：** 参见 manifest §statusBarItem。

**渲染：**
- 平板电脑：完整的底部状态栏。
- 手机竖屏：精简模式；超过 3 个项目将折叠到“...”菜单（右对齐）或滚动（左对齐）。

**限制：** 每个插件最多 2 个项目。文本长度 ≤ 24 个字符。通过 `opendray.workbench.updateStatusBar(id, {...})` 更新。

---

## 7. `menus` (菜单) — MVP

附加到命名菜单点的上下文菜单：
- `editor/context` (编辑器上下文)
- `editor/title` (编辑器标题)
- `explorer/context` (资源管理器上下文)
- `session/toolbar` (会话工具栏)
- `view/title` (视图标题)
- `commandPalette` (命令面板，隐式 —— 每个命令都会自动注册)

```json
"menus": {
  "session/toolbar": [
    { "command": "myext.clear", "when": "sessionStatus == 'running'", "group": "navigation" }
  ]
}
```

**限制：** 每个插件每个菜单点最多 16 个条目。

---

## 8. `keybindings` (按键绑定) — MVP

和弦键/按键到命令的映射。

```json
{ "command": "myext.refresh", "key": "ctrl+shift+r", "mac": "cmd+shift+r",
  "when": "viewFocused == 'myext.view'" }
```

**规则：**
- 用户按键绑定始终覆盖插件的绑定。
- 插件间的冲突：先加载的胜出，宿主发出警告，设置 UI 显示冲突。
- 按键必须使用 VS Code 按键语法（`ctrl`, `alt`, `shift`, `meta`/`cmd`，以及 `a-z`, `0-9`, `f1..f19`, `escape`, `tab`, `enter`, `backspace`, `space`, `up`, `down`, `left`, `right`）。

---

## 9. `editorActions` (编辑器操作) — MVP

编辑器标题栏 / 侧栏（gutter）中的按钮。

```json
{ "id": "myext.format", "title": "格式化",
  "when": "resourceLangId == 'json'", "group": "navigation" }
```

在此处声明编辑器操作，并绑定到具有相同 `id` 的命令。

---

## 10. `sessionActions` (会话操作) — MVP

终端/智能体会话卡片（仪表盘）上方及会话工具栏中的按钮。

```json
{ "id": "myext.costReport", "title": "成本", "icon": "icons/$.svg",
  "command": "myext.showCost",
  "when": "sessionType == 'claude'" }
```

**限制：** 每个插件最多 4 个。

---

## 11. `telegramCommands` (Telegram 命令) — MVP

一等表面 —— OpenDray 是移动优先的，Telegram 是我们的远程控制通道。

```json
{ "command": "/deploy", "description": "部署当前分支",
  "handler": "deployHandler" }
```

- `command` 必须匹配 `^/[a-z][a-z0-9_]{0,31}$`。
- `handler` 是调用的边车方法（宿主形式）或声明式 `run` 定义（声明式形式）。
- 宿主在激活时会自动将命令添加到 BotFather 注册中。
- 需要 `permissions.telegram: true`。

**限制：** 每个插件最多 16 个。

---

## 12. `agentProviders` (智能体提供者) — MVP

在“新会话”工具选择器中添加条目。等同于现在的 `plugins/agents/*`。

```json
{ "id": "crush", "displayName": "Crush",
  "kind": "cli",
  "cli": { "command": "crush", "defaultArgs": ["--mobile"] },
  "configSchema": [ { "key":"apiKey","label":"API Key","type":"secret","envVar":"CRUSH_KEY" } ],
  "capabilities": { "supportsResume": true, "supportsStream": true }
}
```

这是无需修改 Go 代码即可添加新 AI 工具的方式。运行时使用现有的 `ResolveCLI` 路径将每个 `agentProvider` 转换为实时的 `plugin.Provider`。

---

## 13. `themes` (主题) — MVP

贡献一个颜色主题 JSON 文件。

```json
{ "id": "neon-night", "label": "Neon Night", "kind": "dark",
  "path": "themes/neon-night.json" }
```

主题 JSON 模式遵循 VS Code 的颜色标记集（子集 —— 参见 [workbench-slots.md §主题化](08-workbench-slots.md#theming)）。

---

## 14. `languages` (语言) — post-v1

保留。模式已在上方列出。在 v1 中，宿主会忽略这些条目，但在清单中接受它们，以便早期插件能够面向未来。

## 15. `taskRunners` (任务运行器) — post-v1

模式保留。目前的 `plugins/panels/task-runner` 在 M5 中仍作为面板插件存在。

## 16. `debuggers` (调试器) — post-v1

保留。DAP 集成计划在 M7 进行。

## 17. `languageServers` (语言服务器) — MVP (仅限宿主形式)

```json
{ "id": "rust-analyzer", "languages": ["rust"],
  "transport": "stdio", "initializationOptions": {} }
```

当打开所列语言的文件时，宿主激活插件（通过 `onLanguage:*`），启动边车（`form: host`），并在编辑器和边车之间代理 LSP 消息。每种语言只能有一个活跃的 LSP；插件按安装顺序尝试，第一个响应 `initialize` 的插件胜出。

---

## `when` 表达式

跨 `when` 字段使用的上下文表达式：

| 变量 | 值 |
|----------|--------|
| `activeView` | 聚焦视图的 ID |
| `viewFocused` | 聚焦视图的 ID (别名) |
| `sessionStatus` | `idle` (空闲) / `running` (运行中) / `stopped` (已停止) |
| `sessionType` | 智能体提供者 ID |
| `resourceLangId` | 活动编辑器的语言 |
| `platform` | `ios` / `android` / `web` / `desktop` |
| `orientation` | `portrait` (竖屏) / `landscape` (横屏) |

运算符：`==`, `!=`, `&&`, `||`, `!`, 括号, 单引号字符串, 数字字面量。

> **已锁定：** `when` 语法是 VS Code 语法的严格子集，因此文档和工具可以无缝迁移。

## 手机与平板自适应规则

| 插槽 | 手机竖屏 | 平板 / 横屏 |
|------|----------------|--------------------|
| 活动栏 | 底部导航 + “更多” | 左侧栏 |
| 视图 | 全屏面板 | 侧边面板 |
| 面板 | 底部面板 (滑动) | 底部选项卡栏 |
| 状态栏 | 左侧 2 个 + 右侧 1 个 + 溢出 | 完整显示 |
| 菜单 | 底部操作面板 | 弹出菜单 |
| 会话操作 | 仅图标，2 个可见 + "..." | 文本 + 图标 |

## 无障碍要求 (由外壳强制执行)

- 每个图标必须有 `title` (标题) / `tooltip` (工具提示)。
- 手机上的最小点击目标为 48x48 逻辑像素。
- 文本相对于主题背景的颜色对比度 ≥ 4.5:1。
- 声明式视图必须在每个可点击节点上提供 `semanticsLabel` (无障碍标签) (post-v1 宿主验证器强制执行；v1 仅发出警告)。

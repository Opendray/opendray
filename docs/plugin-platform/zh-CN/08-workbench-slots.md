# 08 — 工作台插槽 (Workbench Slots)

Flutter 工作台外壳是一个由命名插槽组成的固定布局。插件将项目贡献到插槽中；外壳决定如何针对当前规格尺寸渲染它们。

## 平板 / 桌面横屏

```
┌──────────────────────────────────────────────────────────────┐
│  标题栏 (TitleBar)                             [?] [•] [用户] │  <- titleBar
├──┬───────────────────────────────────────────────────────────┤
│  │                                                           │
│活│                  主区域 (Primary area)                    │
│动│           (视图 · 编辑器 · 终端)                          │
│栏│                                                           │
│  │                                                           │
│(A│                                                           │
│c│                                                           │
│t│                                                           │
│i│                                                           │
│v│                                                           │
│i│                                                           │
│t│                                                           │
│y│                                                           │
│B│                                                           │
│a│                  底部面板 (Bottom panel)                  │
│r)│        (终端 · 日志 · 问题 · 任务 · ...)                 │
│  │                                                           │
├──┴───────────────────────────────────────────────────────────┤
│  状态栏: [左侧项目]                          [右侧项目]       │  <- statusBar
└──────────────────────────────────────────────────────────────┘
```

## 手机竖屏

```
┌──────────────────────────────┐
│  标题栏          [≡] [用户]  │
├──────────────────────────────┤
│                              │
│     主区域 (一次显示一个     │
│     视图，在它们之间滑动     │
│     切换)                    │
│                              │
├──────────────────────────────┤
│  状态栏 (精简模式)           │
├──────────────────────────────┤
│ [📁] [▶] [Δ] [☰ 更多]        │  <- activityBar (底部导航, 4 + 更多)
└──────────────────────────────┘
```

手机上的底部面板：默认隐藏；底部的向上滑动操作条可将其展开为半屏面板。

## 插槽目录 (Slot catalogue)

### `titleBar` (标题栏)
- **渲染：** 应用标题、全局操作（搜索、面板切换）、用户菜单。
- **贡献：** v1 中仅限宿主贡献。
- **post-v1：** 通过 `titleBar` 贡献点提供插件标题栏按钮。

### `activityBar` (活动栏)
- **贡献点：** `contributes.activityBar[]`。
- **渲染：** 图标 + 工具提示；点击可聚焦链接的视图。
- **手机：** 带有 4 个插槽的底部导航；第 5 个及以后的项目进入“更多”。
- **平板：** 左侧垂直边栏。
- **主题标记 (Tokens)：** `activityBar.background`, `activityBar.foreground`, `activityBar.activeBorder`。

### `views` (视图)
- **贡献点：** `contributes.views[]`。
- **渲染：** Web 视图（默认）或声明式树 (`opendray.ui`)。
- **手机：** 视图占据整个主区域。向左/向右滑动可在打开的视图之间循环。返回手势可关闭。
- **平板：** 视图锚定在其活动栏插槽上；打开另一个视图将替换当前视图。多视图分屏功能在 post-v1 中提供。
- **手势：**
  - 从视图顶部向下滑动：关闭。
  - 长按活动图标：固定视图（在其他交互中保持开启）。

### `panels` (底部面板)
- **贡献点：** `contributes.panels[]`。
- **手机：** 向上滑动的底部面板；内部有用于切换多个面板的选项卡栏。
- **平板：** 始终可见的选项卡栏；拖动分隔条可调整大小。
- **关闭开关：** 点击选项卡上的 X 将从视图中移除面板（而非卸载）。

### `editorActions` (编辑器操作)
- **贡献点：** `contributes.editorActions[]`。
- **渲染：** 编辑器内容区域上方右对齐的图标按钮。
- **手机：** 折叠为 2 个图标 + 溢出菜单。

### `sessionActions` (会话操作)
- **贡献点：** `contributes.sessionActions[]`。
- **渲染：** 会话卡片（仪表盘）上及运行中的会话工具栏中的图标按钮。
- **手机：** 显示 2 个图标 + “...” 菜单。

### `statusBar` (状态栏)
- **贡献点：** `contributes.statusBar[]`。
- **渲染：** 文本 + 可选图标；如果设置了 `command`，点击将运行该命令。
- **手机：** 最多可见 3 个项目，其余折叠。
- **更新：** 通过 `opendray.workbench.updateStatusBar` 推送更新。

### `commandPalette` (命令面板)
- **自动填充：** 来自每个贡献的 `commands` 条目。
- **调用：** 平板上为标题栏按钮；手机上长按标题区域。

### `notifications` (通知)
- **自动填充：** 由 `opendray.workbench.showMessage` 填充。
- **宿主管理：** 插件无法贡献自定义通知中心。

### `settingsPane` (设置面板)
- **贡献点：** `contributes.settings[]`。
- **渲染：** 在“设置 → 插件 → <名称>”中渲染为表单。

### `menus` (菜单)
- **贡献点：** `contributes.menus.*`。
- **渲染：** 当锚点菜单点打开时，显示为底部面板（手机）或弹出菜单（平板）。

## 主题契约 (Theming contract)

主题包含一个以标记 (Token) 名称为键的 JSON 文件；外壳将标记映射到 Flutter 的 `ThemeData`。v1 所需的标记如下（缺失的键将回退到活动的浅色/深色基准主题）：

```
"editor.background", "editor.foreground", "editor.lineNumberForeground",
"activityBar.background", "activityBar.foreground", "activityBar.activeBorder",
"statusBar.background", "statusBar.foreground",
"panel.background", "panel.border",
"button.background", "button.foreground", "button.hoverBackground",
"input.background", "input.foreground", "input.border",
"list.hoverBackground", "list.activeSelectionBackground",
"terminal.background", "terminal.foreground",
"terminal.ansiBlack", ..., "terminal.ansiBrightWhite"
```

插件 Web 视图通过 CSS 变量（如 `--od-editor-background` 等）以及 `<html>` 上的 `data-theme="dark|light|high-contrast"` 属性获取这些值。

> **已锁定：** 主题标记名称是 VS Code 名称的严格子集，因此现有主题可以通过重命名轻松迁移。

## 手势与键盘映射 (手机)

| 手势 | 效果 |
|---------|--------|
| 从底部边缘向上滑动 | 展开面板 |
| 从视图顶部向下滑动 | 关闭视图 |
| 长按活动图标 | 固定视图 |
| 双指点击 | 打开命令面板 |
| 从左边缘滑动 | 上一个视图 |
| 从右边缘滑动 | 下一个视图 |

> **已锁定 (2026-04-19)：** 使用双指 + 边缘滑动。取消了三指手势 —— 这与 iOS VoiceOver 冲突且难以发现。边缘滑动是明确的，并且在开启无障碍功能时也能工作。

## 无障碍要求 (Accessibility requirements)

- 每个交互式插槽项目必须具有 `semanticsLabel` (无障碍标签)。
- 屏幕阅读器顺序遵循 `contributes.*` 数组的源顺序。
- 聚焦环感知主题，相对于背景的对比度 ≥ 3:1。
- 手机上的最小点击目标为 48x48 逻辑像素。
- 遵循减弱动态效果 (Reduced motion) 设置（Web 视图接收 `prefers-reduced-motion: reduce`）。

## 插槽所有权 (Go)

- `gateway/plugins_assets.go` (新增) — 提供 Web 视图插件包服务。
- `gateway/plugins_bridge.go` (新增) — 桥接 WebSocket。
- `plugin/slots/` (新增) — 验证贡献条目是否符合插槽限制。

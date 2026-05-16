# Multi-client session access

opendray 允许同一个 session 被多端同时打开（web admin、iOS app、Telegram），但**强烈建议同一时刻只用一端操作**。原因在底层架构，不在 opendray 自身。

## 为什么不能让 web 和 mobile 各自舒服

```
   Claude / Gemini / Codex CLI 进程
                │
                ▼
        PTY（伪终端）
        只能有 ONE 尺寸
                │
       ┌────────┼────────┐
       ▼        ▼        ▼
   手机 xterm  web xterm  Telegram
```

CLI **是单进程**，启动时通过 `TIOCGWINSZ` 询问操作系统「这个终端多宽？」，然后按这个宽度的字符 grid 进行**绝对定位排版**（"光标移动到 (45, 10)" 这种）。

所有 client 拿到的是同一份**已经排版好的字节流**。手机 xterm 和 web xterm 本质上是「同一张图」的不同显示窗口 —— **没有任何中间层能让 CLI 同时为两个不同尺寸渲染**。

这意味着只要多个 client 同时连，必有一端的视觉效果次于单独使用时。常见症状：

- **Web 拖动窗口** → 触发 FitAddon → 改 PTY 尺寸 → 手机端 TUI 跟着重排
- **手机端连接** → 设置 PTY 为手机尺寸 → web 端窗口看到右侧大片空白单元格
- **桌面 + 手机端轮流交互** → 谁最近 resize 谁赢，TUI 在两种尺寸间反复重排

## 三条可行路径

### A. 推荐：一次只用一端

最简单可靠的方案。常见工作流：

- 桌面 → 用 web admin
- 出门 / 沙发 / 临时 → 用手机 app
- 不在前面 → 让 Telegram idle 通知告诉你

opendray 的 idle 通知会推送 Claude / Gemini 的**完整 prose 回复**到 Telegram（不再退化到屏幕截图），所以「不开 web，靠 Telegram 看进度」是完全可行的。

### B. 双 session（同 cwd，独立状态）

如果真的想 web + 手机各自独立：在同一个 cwd 起**两个**不同的 session。它们各自有 PTY、各自连一份 CLI 进程，**互不干扰**。

代价：**两个对话状态不同步**。你在手机的 Gemini 跟它说的话，web 那边的 Gemini 不知道。适合「桌面跑一个长任务、手机另起一个临时问问题」的场景，不适合「想从手机继续桌面的对话」。

### C. 桌面只用 web、手机只用 mobile，session 各自独立

最自然的工作流：每个端有自己的会话集。不依赖跨端共享。idle 通知 + Telegram 命令仍可跨端使用（`/list`、`/end`、`/resume` 等）。

## 为什么不能用 tmux 解决

tmux 处理多 client 的方式是「**取最小尺寸**给所有 client」—— 仍然是单尺寸，仍然要妥协，只是把决策权从 opendray 挪到 tmux 内部。没有本质区别。

## 真正独立的多端体验需要什么

需要 **client-side conversation state**（对话状态在客户端，不在 CLI 进程里），同时配合状态同步层。这是 Claude/Gemini/Codex 的 API/web UI 的工作方式，但**不是 CLI TUI 的工作方式**。opendray 接的是 TUI CLI，所以受限于 TUI 的架构。

如果未来 Anthropic / Google 发布带 HTTP 状态 API 的 CLI 版本，opendray 可以重新评估这个问题。在那之前 —— 用方案 A 或 B。

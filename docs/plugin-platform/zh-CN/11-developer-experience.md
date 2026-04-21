# 11 — 开发者体验 (Developer Experience)

## `@opendray/plugin-sdk` npm 包

发布布局：
```
@opendray/plugin-sdk/
  package.json
  dist/
    index.js              # 在开发/测试中使用的运行时存根（空操作桥接调用）
    index.d.ts            # TS 类型 — 在 04-bridge-api.md 中逐字定义的文件
    bridge.js             # 用于 webview 包的最小有线协议实现
  schemas/
    manifest-v1.json      # 来自 02-manifest.md 的架构
  cli.js                  # 通过 `opendray plugin <cmd>` 调用
```

安装：
```
npm i -D @opendray/plugin-sdk
```

二进制文件：
```
"bin": { "opendray": "./cli.js" }
```

### SDK 形状（公共）

```ts
// Webview 入口 — 为你的插件获取一个类型化的句柄
import { getOpenDray } from '@opendray/plugin-sdk';
const od = getOpenDray();
await od.workbench.showMessage('ready');

// 宿主边车入口 — 获取一个类型化的 JSON-RPC 服务器
import { createHost } from '@opendray/plugin-sdk/host';
const host = createHost({
  async initialize(params) { return { capabilities: {} }; },
  async 'myext/refresh'(_args) { return { ok: true }; },
});
host.listen();
```

## CLI `opendray plugin <cmd>`

| 命令 | 用途 |
|---------|---------|
| `opendray plugin scaffold <name>` | 交互式；选择表单 + 垂直模板 |
| `opendray plugin validate [dir]` | 验证 manifest.json + 结构 |
| `opendray plugin dev [dir]` | 对运行中的 Opendray 宿主进行监听 + 热重载 |
| `opendray plugin build [dir]` | 生成可分发的 zip 包 |
| `opendray plugin publish [dir]` | 构建 + 签名 + 对市场发起 PR |
| `opendray plugin lint [dir]` | 静态检查（禁止的 API、能力漂移） |
| `opendray plugin sign --key <path>` | 使用 ed25519/minisign 签署包 |
| `opendray plugin doctor` | 诊断本地环境 |

## 脚手架模板

```
opendray plugin scaffold --form declarative --template statusbar
opendray plugin scaffold --form webview     --template view-react
opendray plugin scaffold --form webview     --template view-svelte
opendray plugin scaffold --form host        --template lsp
opendray plugin scaffold --form host        --template db-panel
opendray plugin scaffold --form host        --template task-runner
opendray plugin scaffold --form host        --template agent-provider
opendray plugin scaffold --form declarative --template theme
opendray plugin scaffold --form declarative --template snippets
opendray plugin scaffold --form declarative --template telegram-command
```

每个模板都会生成一个可运行的项目，包含：
- `manifest.json`：预填了必填字段、最小 `permissions` 和一个工作的贡献点。
- `README.md`：包含 3 个入门命令。
- `package.json`：包含 `opendray dev`、`opendray build`、`opendray publish`。
- 如果是 webview/host，则包含 TypeScript 设置。
- 用于 CI 验证的 GitHub Actions 工作流。

## 开发模式热重载

`opendray plugin dev`:
1. 读取 `manifest.json`。
2. 使用本地路径调用宿主的 `POST /api/plugins/dev/register`。
3. 宿主以特殊的 `sideloaded+dev` 信任级别挂载 `local:/path/to/dir/` 并自动激活。
4. 监听目录；发生任何更改时：
   - 声明式（Declarative）：重新注册清单。
   - Webview：告知 Flutter 壳 `reload` 视图的 WebView。
   - 宿主（Host）：发送 `deactivate`，重新启动边车，发送 `activate`。
5. 错误实时打印到 CLI，并显示在宿主的 “设置 → 插件 → 日志” 中。

> **已锁定：** 开发模式由宿主上的 `OPENDRAY_ALLOW_LOCAL_PLUGINS=1` 开启，在生产构建中关闭。

## 调试

### 日志抽屉
- 每个 `opendray.logger.*` 调用都会进入 “设置 → 插件 → <名称> → 日志”（实时滚动 + 搜索）。
- 可按级别过滤。
- 结构化日志（`{ msg, meta }`）呈现为可展开的行。

### 断点
- **Webview:** 通过 Android USB 使用 Chrome DevTools / 通过 iOS USB 使用 Safari Web Inspector。在桌面上，运行 OpenDray 调试构建时，“设置 → 插件 → <名称>” 中会显示 “打开 DevTools” 按钮。
- **Host (node/deno):** 由 `opendray plugin dev --inspect` 连接的 `--inspect` 标志；使用模板生成的启动配置从 VS Code 连接。
- **Host (binary):** 将你偏好的调试器（`lldb`、`gdb`、Delve）附加到 “设置 → 插件 → <名称> → 诊断” 页面显示的边车 PID。

### 桥接追踪
`opendray plugin dev --trace-bridge` 将每个桥接调用 + 响应流式传输到标准输出：
```
> fs.readFile ["/etc/hosts"]  [plugin=myext]  denied:EPERM:path not in fs.read
< 11ms
```

## 测试套件

`@opendray/plugin-sdk/test` 导出一个模拟宿主：
```ts
import { mockHost } from '@opendray/plugin-sdk/test';
const host = mockHost();
host.fs.withFile('/tmp/x', 'hi');
const od = host.asOpenDray({ plugin: { name: 'myext', version: '0.1.0', publisher: 'me' }});
expect(await od.fs.readFile('/tmp/x')).toBe('hi');
expect(host.auditLog.filter(a => a.ns === 'fs')).toHaveLength(1);
```

随附用于 webview 插件的 `vitest` 配置模板，以及用于 Node 宿主插件的 `node:test` 模板。

## 本地化

- 清单中的 `displayName`、`description` 以及每个 `title` 字段都支持 `%keys%`。
- `i18n/` 下的每个语言区域的 JSON 文件：
  ```
  i18n/
    en.json
    zh-CN.json
  ```
  ```json
  { "displayName": "看板", "view.board.title": "看板" }
  ```
- 宿主选择与应用当前语言区域匹配的语言区域，默认回退到 `en`。

## 调试输出惯例

- 错误消息必须包含插件名称前缀：`[myext] failed to load: ...`。
- 相比字符串插值，更推荐使用结构化字段。
- 发布插件中的 webview 不允许使用 `console.log` — SDK 中的 lint 规则。

## 文档生成

`opendray plugin docs` 根据以下内容生成 Markdown 页面：
- 清单字段（身份、能力、贡献）。
- 边车 JSON-RPC 方法处理程序的 JSDoc（Node 模板）。
- 合并入 README.md。

鼓励发布者将生成的 `DOC.md` 提交到其仓库。

## 参考项目

一个单独的仓库 `opendray/plugin-examples` 为每个模板包含一个工作的插件。每个示例都可以通过 `marketplace://opendray-examples/*` 独立安装。

> **已锁定 (2026-04-19):** `opendray-dev` 便携式宿主随 SDK 在 M6 中发布。插件作者必须能够在没有完整 OpenDray 实例的情况下离线开发 — 这是 `code --extensionDevelopmentPath` 的等效物，对于生态系统开发者体验（DX）至关重要。

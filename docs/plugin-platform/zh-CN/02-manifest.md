# 02 — manifest.json (v1)

每个插件包在其根目录下都包含且仅包含一个 `manifest.json` 文件。该文件是插件身份、入口点、UI 贡献和所需能力的全部事实来源。

## JSON 模式 (JSON Schema)

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://opendray.dev/schemas/plugin-manifest-v1.json",
  "title": "OpenDray Plugin Manifest v1",
  "type": "object",
  "required": ["name", "version", "publisher", "engines"],
  "additionalProperties": false,
  "properties": {

    "$schema":   { "type": "string" },
    "name":      { "type": "string", "pattern": "^[a-z0-9]([a-z0-9-]{0,62}[a-z0-9])?$" },
    "version":   { "type": "string", "pattern": "^\\d+\\.\\d+\\.\\d+(-[A-Za-z0-9.-]+)?$" },
    "publisher": { "type": "string", "pattern": "^[a-z0-9]([a-z0-9-]{0,38}[a-z0-9])?$" },
    "displayName": { "type": "string", "maxLength": 64 },
    "description": { "type": "string", "maxLength": 280 },
    "icon":        { "type": "string", "description": "Emoji 或相对于 svg/png 的相对路径 <= 256KB" },
    "homepage":    { "type": "string", "format": "uri" },
    "repository":  { "type": "string", "format": "uri" },
    "license":     { "type": "string", "description": "SPDX ID" },
    "keywords":    { "type": "array", "items": { "type": "string" }, "maxItems": 16 },
    "categories":  { "type": "array",
                     "items": { "enum": ["agent","language","linter","debugger","theme",
                                         "snippet","scm","data","devops","productivity","other"] },
                     "maxItems": 3 },

    "engines": {
      "type": "object", "required": ["opendray"],
      "properties": {
        "opendray": { "type": "string", "description": "semver 范围，例如 ^1.0.0" },
        "node":     { "type": "string" },
        "deno":     { "type": "string" }
      },
      "additionalProperties": false
    },

    "form": { "enum": ["declarative", "webview", "host"], "default": "declarative" },

    "ui": {
      "type": "object",
      "description": "当 form=webview 时必填",
      "required": ["entry"],
      "properties": {
        "entry":  { "type": "string", "description": "UI 包入口 HTML 的路径" },
        "csp":    { "type": "string", "description": "可选的额外 CSP 指令" },
        "root":   { "type": "string", "default": "ui/" }
      },
      "additionalProperties": false
    },

    "host": {
      "type": "object",
      "description": "当 form=host 时必填",
      "required": ["entry", "protocol"],
      "properties": {
        "entry":     { "type": "string" },
        "platforms": {
          "type": "object",
          "patternProperties": {
            "^(linux|darwin|windows)-(x64|arm64)$": { "type": "string" }
          },
          "additionalProperties": false
        },
        "runtime":   { "enum": ["binary", "node", "deno"] , "default": "binary" },
        "protocol":  { "enum": ["jsonrpc-stdio"] },
        "restart":   { "enum": ["on-failure", "always", "never"], "default": "on-failure" },
        "env":       { "type": "object", "additionalProperties": { "type": "string" } },
        "cwd":       { "type": "string" }
      },
      "additionalProperties": false
    },

    "activation": {
      "type": "array",
      "description": "触发插件激活的事件。为空表示惰性加载，永不自动激活。",
      "items": {
        "type": "string",
        "pattern": "^(onStartup|onCommand:[a-z0-9._-]+|onView:[a-z0-9._-]+|onSession:(start|stop|idle|output)|onLanguage:[a-z0-9_-]+|onFile:[^\\s]+|onSchedule:cron:.+)$"
      },
      "maxItems": 32
    },

    "contributes": { "$ref": "#/$defs/contributes" },
    "permissions": { "$ref": "#/$defs/permissions" },

    "configSchema": {
      "type": "array",
      "description": "遗留字段 —— 原样保留。新代码使用 contributes.settings。",
      "items": { "type": "object" }
    },

    "// ---- 来自 Provider 的锁定兼容字段 ---- ": {},
    "type":         { "enum": ["cli","local","shell","panel"], "description": "遗留 Provider.Type；缺失时由 form 暗示" },
    "category":     { "type": "string" },
    "cli":          { "$ref": "#/$defs/cliSpec" },
    "capabilities": { "$ref": "#/$defs/legacyCapabilities" },

    "v2Reserved": { "type": "object", "description": "为 v2 保留。插件宿主必须忽略此处未知的键。" }
  },

  "$defs": {
    "contributes": { "type": "object", "additionalProperties": false, "properties": {
      "commands":         { "type": "array", "items": { "$ref": "#/$defs/command" } },
      "settings":         { "type": "array", "items": { "$ref": "#/$defs/settingField" } },
      "statusBar":        { "type": "array", "items": { "$ref": "#/$defs/statusBarItem" } },
      "activityBar":      { "type": "array", "items": { "$ref": "#/$defs/activityBarItem" } },
      "views":            { "type": "array", "items": { "$ref": "#/$defs/view" } },
      "panels":           { "type": "array", "items": { "$ref": "#/$defs/panel" } },
      "menus":            { "type": "object" },
      "keybindings":      { "type": "array", "items": { "$ref": "#/$defs/keybinding" } },
      "editorActions":    { "type": "array", "items": { "$ref": "#/$defs/editorAction" } },
      "sessionActions":   { "type": "array", "items": { "$ref": "#/$defs/sessionAction" } },
      "telegramCommands": { "type": "array", "items": { "$ref": "#/$defs/telegramCommand" } },
      "agentProviders":   { "type": "array", "items": { "$ref": "#/$defs/agentProvider" } },
      "themes":           { "type": "array", "items": { "$ref": "#/$defs/theme" } },
      "languages":        { "type": "array", "items": { "$ref": "#/$defs/language" } },
      "taskRunners":      { "type": "array", "items": { "$ref": "#/$defs/taskRunner" } },
      "debuggers":        { "type": "array", "items": { "$ref": "#/$defs/debugger" } },
      "languageServers":  { "type": "array", "items": { "$ref": "#/$defs/languageServer" } }
    }},

    "permissions": {
      "type": "object", "additionalProperties": false,
      "properties": {
        "fs":        { "oneOf": [
                        {"type": "boolean"},
                        {"type": "object", "properties": {
                          "read":  {"type": "array", "items": {"type":"string"}},
                          "write": {"type": "array", "items": {"type":"string"}}
                        }, "additionalProperties": false } ] },
        "exec":      { "oneOf": [ {"type":"boolean"},
                                   {"type":"array", "items":{"type":"string"}} ] },
        "http":      { "oneOf": [ {"type":"boolean"},
                                   {"type":"array", "items":{"type":"string", "format":"uri-reference"}} ] },
        "session":   { "enum": [false,"read","write"] },
        "storage":   { "type": "boolean" },
        "secret":    { "type": "boolean" },
        "clipboard": { "enum": [false,"read","write","readwrite"] },
        "telegram":  { "type": "boolean" },
        "git":       { "enum": [false,"read","write"] },
        "llm":       { "type": "boolean" },
        "events":    { "type": "array", "items": { "type": "string" } }
      }
    },

    "command": { "type":"object", "required":["id","title"],
      "properties": {
        "id":    {"type":"string"},
        "title": {"type":"string"},
        "icon":  {"type":"string"},
        "category":{"type":"string"},
        "when":  {"type":"string", "description":"上下文表达式"},
        "run":   {"oneOf":[
          {"type":"object","required":["kind"],"properties":{
            "kind":{"enum":["host","notify","openView","runTask","exec","openUrl"]},
            "method":{"type":"string"},"args":{"type":"array"},
            "viewId":{"type":"string"},"url":{"type":"string"},
            "message":{"type":"string"},"taskId":{"type":"string"}
          }}
        ]}
      }
    },

    "settingField": {
      "type":"object","required":["key","type","label"],
      "properties":{
        "key":{"type":"string"},"label":{"type":"string"},
        "type":{"enum":["string","number","boolean","select","secret","text","args"]},
        "description":{"type":"string"},"placeholder":{"type":"string"},
        "default":{},"options":{"type":"array"},
        "required":{"type":"boolean"},"group":{"type":"string"},
        "dependsOn":{"type":"string"},"dependsVal":{"type":"string"},
        "envVar":{"type":"string"},"cliFlag":{"type":"string"},"cliValue":{"type":"boolean"}
      }
    },

    "statusBarItem":    { "type":"object","required":["id","text"],
      "properties":{"id":{"type":"string"},"text":{"type":"string"},
                    "tooltip":{"type":"string"},"command":{"type":"string"},
                    "alignment":{"enum":["left","right"],"default":"right"},
                    "priority":{"type":"integer"}}},

    "activityBarItem":  { "type":"object","required":["id","icon","title"],
      "properties":{"id":{"type":"string"},"icon":{"type":"string"},
                    "title":{"type":"string"},"viewId":{"type":"string"}}},

    "view":             { "type":"object","required":["id","title"],
      "properties":{"id":{"type":"string"},"title":{"type":"string"},
                    "container":{"enum":["activityBar","panel","sidebar"],"default":"activityBar"},
                    "icon":{"type":"string"},"when":{"type":"string"},
                    "render":{"enum":["webview","declarative"],"default":"webview"}}},

    "panel":            { "type":"object","required":["id","title"],
      "properties":{"id":{"type":"string"},"title":{"type":"string"},
                    "icon":{"type":"string"},"position":{"enum":["bottom","right"]}}},

    "keybinding":       { "type":"object","required":["command","key"],
      "properties":{"command":{"type":"string"},"key":{"type":"string"},
                    "mac":{"type":"string"},"when":{"type":"string"}}},

    "editorAction":     { "type":"object","required":["id","title"],
      "properties":{"id":{"type":"string"},"title":{"type":"string"},
                    "when":{"type":"string"},"group":{"type":"string"}}},

    "sessionAction":    { "type":"object","required":["id","title"],
      "properties":{"id":{"type":"string"},"title":{"type":"string"},
                    "icon":{"type":"string"},"command":{"type":"string"},
                    "when":{"type":"string"}}},

    "telegramCommand":  { "type":"object","required":["command","description"],
      "properties":{"command":{"type":"string","pattern":"^/[a-z][a-z0-9_]{0,31}$"},
                    "description":{"type":"string","maxLength":80},
                    "handler":{"type":"string"}}},

    "agentProvider":    { "type":"object","required":["id","displayName"],
      "properties":{"id":{"type":"string"},"displayName":{"type":"string"},
                    "kind":{"enum":["cli","api"]},
                    "cli":{"$ref":"#/$defs/cliSpec"},
                    "configSchema":{"type":"array","items":{"$ref":"#/$defs/settingField"}},
                    "capabilities":{"$ref":"#/$defs/legacyCapabilities"}}},

    "theme":            { "type":"object","required":["id","label","path"],
      "properties":{"id":{"type":"string"},"label":{"type":"string"},
                    "path":{"type":"string"},"kind":{"enum":["light","dark","high-contrast"]}}},

    "language":         { "type":"object","required":["id"],
      "properties":{"id":{"type":"string"},"extensions":{"type":"array","items":{"type":"string"}},
                    "aliases":{"type":"array","items":{"type":"string"}},
                    "grammar":{"type":"string"},"snippets":{"type":"string"}}},

    "taskRunner":       { "type":"object","required":["id","title"],
      "properties":{"id":{"type":"string"},"title":{"type":"string"},
                    "match":{"type":"string","description":"glob 通配符"},
                    "discover":{"type":"string"},"handler":{"type":"string"}}},

    "debugger":         { "type":"object","required":["id","label"],
      "properties":{"id":{"type":"string"},"label":{"type":"string"},
                    "languages":{"type":"array","items":{"type":"string"}},
                    "adapter":{"type":"string","description":"宿主方法名称"}}},

    "languageServer":   { "type":"object","required":["id","languages"],
      "properties":{"id":{"type":"string"},
                    "languages":{"type":"array","items":{"type":"string"}},
                    "transport":{"enum":["stdio"],"default":"stdio"},
                    "initializationOptions":{"type":"object"}}},

    "cliSpec": {
      "type":"object","required":["command"],
      "properties":{"command":{"type":"string"},
                    "defaultArgs":{"type":"array","items":{"type":"string"}},
                    "detectCmd":{"type":"string"}}
    },

    "legacyCapabilities": {
      "type":"object",
      "properties":{"models":{"type":"array"},"supportsResume":{"type":"boolean"},
                    "supportsStream":{"type":"boolean"},"supportsImages":{"type":"boolean"},
                    "supportsMcp":{"type":"boolean"},"dynamicModels":{"type":"boolean"}}
    }
  }
}
```

## 完整示例

### 声明式 (状态栏 + 命令 + 主题)
```json
{
  "$schema": "https://opendray.dev/schemas/plugin-manifest-v1.json",
  "name": "time-ninja",
  "version": "1.0.0",
  "publisher": "acme",
  "displayName": "Time Ninja",
  "description": "状态栏中的番茄钟。",
  "icon": "🍅",
  "engines": { "opendray": "^1.0.0" },
  "form": "declarative",
  "activation": ["onStartup"],
  "contributes": {
    "commands": [
      { "id": "time.start", "title": "开始番茄钟",
        "run": { "kind": "notify", "message": "番茄钟已开始" } }
    ],
    "statusBar": [
      { "id": "time.bar", "text": "🍅 25:00", "command": "time.start",
        "alignment": "right", "priority": 50 }
    ],
    "keybindings": [
      { "command": "time.start", "key": "ctrl+alt+p" }
    ]
  },
  "permissions": {}
}
```

### Web 视图 (活动栏中的视图)
```json
{
  "$schema": "https://opendray.dev/schemas/plugin-manifest-v1.json",
  "name": "kanban",
  "version": "0.3.0",
  "publisher": "acme",
  "description": "个人看板。",
  "icon": "📋",
  "engines": { "opendray": "^1.0.0" },
  "form": "webview",
  "ui": { "entry": "ui/index.html" },
  "activation": ["onView:kanban.board"],
  "contributes": {
    "activityBar": [
      { "id": "kanban.activity", "icon": "icons/board.svg",
        "title": "看板", "viewId": "kanban.board" }
    ],
    "views": [
      { "id": "kanban.board", "title": "看板",
        "container": "activityBar", "render": "webview" }
    ],
    "settings": [
      { "key": "syncUrl", "label": "同步 URL", "type": "string", "group": "auth" }
    ]
  },
  "permissions": {
    "storage": true,
    "http":   ["https://api.kanban.example/*"]
  }
}
```

### 宿主 (语言服务器)
```json
{
  "$schema": "https://opendray.dev/schemas/plugin-manifest-v1.json",
  "name": "rust-analyzer-od",
  "version": "0.2.0",
  "publisher": "rust-lang",
  "description": "为 OpenDray 打包的 rust-analyzer。",
  "icon": "🦀",
  "engines": { "opendray": "^1.0.0" },
  "form": "host",
  "host": {
    "entry": "bin/rust-analyzer",
    "platforms": {
      "linux-x64":   "bin/linux-x64/rust-analyzer",
      "darwin-arm64":"bin/darwin-arm64/rust-analyzer"
    },
    "runtime": "binary",
    "protocol": "jsonrpc-stdio",
    "restart": "on-failure"
  },
  "activation": ["onLanguage:rust"],
  "contributes": {
    "languageServers": [
      { "id": "rust-analyzer", "languages": ["rust"] }
    ]
  },
  "permissions": {
    "fs":   { "read": ["${workspace}/**"], "write": ["${workspace}/**"] },
    "exec": ["cargo *", "rustc *"]
  }
}
```

## 字段规则和锁定行为

> **已锁定：** `name` 在发布者范围内全局唯一且稳定。重命名需要发布一个新插件。
> **已锁定：** `version` 遵循 semver。除非用户传递 `--allow-downgrade`，否则宿主拒绝降级。
> **已锁定：** 未知的顶级键会导致验证警告（而非错误）。`v2Reserved` 下的未知键会被静默忽略。
> **已锁定：** `type`、`cli`、`category`、`capabilities`、`configSchema` 保持原样，以确保每个当前的 `plugins/agents/*` 和 `plugins/panels/*` 清单加载不受影响。当缺少 `form` 且存在 `type: "cli"|"local"|"shell"` 时，宿主将其视为隐式的 `form: "host"`，并带有一个合成的 CLI 入口点。
> **已锁定 (2026-04-19)：** v1 中没有 `extends`。推迟到 v2，且仅在出现真实用例时才考虑 —— YAGNI。

## 验证

- 宿主在安装时以及加载时根据上述模式验证清单。
- 任何字段拒绝都会终止安装，并显示指明违规路径的差异消息。
- SDK (`@opendray/plugin-sdk`) 附带该模式和一个可离线运行的 CLI 工具 `opendray plugin validate ./dist`。

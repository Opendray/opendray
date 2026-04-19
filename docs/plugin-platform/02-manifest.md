# 02 — manifest.json (v1)

Every plugin bundle contains exactly one `manifest.json` at its root. This file is the entire source of truth for identity, entrypoints, UI contributions, and required capabilities.

## JSON Schema

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
    "icon":        { "type": "string", "description": "Emoji OR relative path to svg/png <= 256KB" },
    "homepage":    { "type": "string", "format": "uri" },
    "repository":  { "type": "string", "format": "uri" },
    "license":     { "type": "string", "description": "SPDX id" },
    "keywords":    { "type": "array", "items": { "type": "string" }, "maxItems": 16 },
    "categories":  { "type": "array",
                     "items": { "enum": ["agent","language","linter","debugger","theme",
                                         "snippet","scm","data","devops","productivity","other"] },
                     "maxItems": 3 },

    "engines": {
      "type": "object", "required": ["opendray"],
      "properties": {
        "opendray": { "type": "string", "description": "semver range, e.g. ^1.0.0" },
        "node":     { "type": "string" },
        "deno":     { "type": "string" }
      },
      "additionalProperties": false
    },

    "form": { "enum": ["declarative", "webview", "host"], "default": "declarative" },

    "ui": {
      "type": "object",
      "description": "Required when form=webview",
      "required": ["entry"],
      "properties": {
        "entry":  { "type": "string", "description": "path to ui bundle entry html" },
        "csp":    { "type": "string", "description": "Optional extra CSP directives" },
        "root":   { "type": "string", "default": "ui/" }
      },
      "additionalProperties": false
    },

    "host": {
      "type": "object",
      "description": "Required when form=host",
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
      "description": "Events that trigger plugin activation. Empty = lazy, never auto-activate.",
      "items": {
        "type": "string",
        "pattern": "^(onStartup|onCommand:[a-z0-9._-]+|onView:[a-z0-9._-]+|onSession:(start|stop|idle|output)|onLanguage:[a-z0-9_-]+|onFile:[^\\s]+|onSchedule:(cron:[^\\s]+))$"
      },
      "maxItems": 32
    },

    "contributes": { "$ref": "#/$defs/contributes" },
    "permissions": { "$ref": "#/$defs/permissions" },

    "configSchema": {
      "type": "array",
      "description": "Legacy field — preserved verbatim. New code uses contributes.settings.",
      "items": { "type": "object" }
    },

    "// ---- locked-in compat fields from Provider ---- ": {},
    "type":         { "enum": ["cli","local","shell","panel"], "description": "Legacy Provider.Type; implied by form when absent" },
    "category":     { "type": "string" },
    "cli":          { "$ref": "#/$defs/cliSpec" },
    "capabilities": { "$ref": "#/$defs/legacyCapabilities" },

    "v2Reserved": { "type": "object", "description": "Reserved for v2. Plugin Host MUST ignore unknown keys here." }
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
        "when":  {"type":"string", "description":"context expression"},
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
                    "match":{"type":"string","description":"glob"},
                    "discover":{"type":"string"},"handler":{"type":"string"}}},

    "debugger":         { "type":"object","required":["id","label"],
      "properties":{"id":{"type":"string"},"label":{"type":"string"},
                    "languages":{"type":"array","items":{"type":"string"}},
                    "adapter":{"type":"string","description":"host method name"}}},

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

## Full examples

### Declarative (statusbar + commands + theme)
```json
{
  "$schema": "https://opendray.dev/schemas/plugin-manifest-v1.json",
  "name": "time-ninja",
  "version": "1.0.0",
  "publisher": "acme",
  "displayName": "Time Ninja",
  "description": "Pomodoro in the status bar.",
  "icon": "🍅",
  "engines": { "opendray": "^1.0.0" },
  "form": "declarative",
  "activation": ["onStartup"],
  "contributes": {
    "commands": [
      { "id": "time.start", "title": "Start Pomodoro",
        "run": { "kind": "notify", "message": "Pomodoro started" } }
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

### Webview (view in activity bar)
```json
{
  "$schema": "https://opendray.dev/schemas/plugin-manifest-v1.json",
  "name": "kanban",
  "version": "0.3.0",
  "publisher": "acme",
  "description": "Personal kanban board.",
  "icon": "📋",
  "engines": { "opendray": "^1.0.0" },
  "form": "webview",
  "ui": { "entry": "ui/index.html" },
  "activation": ["onView:kanban.board"],
  "contributes": {
    "activityBar": [
      { "id": "kanban.activity", "icon": "icons/board.svg",
        "title": "Kanban", "viewId": "kanban.board" }
    ],
    "views": [
      { "id": "kanban.board", "title": "Board",
        "container": "activityBar", "render": "webview" }
    ],
    "settings": [
      { "key": "syncUrl", "label": "Sync URL", "type": "string", "group": "auth" }
    ]
  },
  "permissions": {
    "storage": true,
    "http":   ["https://api.kanban.example/*"]
  }
}
```

### Host (language server)
```json
{
  "$schema": "https://opendray.dev/schemas/plugin-manifest-v1.json",
  "name": "rust-analyzer-od",
  "version": "0.2.0",
  "publisher": "rust-lang",
  "description": "rust-analyzer packaged for OpenDray.",
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

## Field rules and locked behaviour

> **Locked:** `name` is globally unique within a publisher and stable. Renaming requires publishing a new plugin.
> **Locked:** `version` is semver. Host refuses to downgrade unless `--allow-downgrade` is passed by the user.
> **Locked:** Unknown top-level keys cause validation warning (not error). Unknown keys under `v2Reserved` are silently ignored.
> **Locked:** `type`, `cli`, `category`, `capabilities`, `configSchema` stay as-is to keep every current `plugins/agents/*` and `plugins/panels/*` manifest loading unchanged. The Host treats absence of `form` + presence of `type: "cli"|"local"|"shell"` as implicit `form: "host"` with a synthetic CLI entrypoint.
> **Locked (2026-04-19):** No `extends` in v1. Deferred to v2 and only if a real use case appears — YAGNI.

## Validation

- Host validates manifest against the schema above at install and again at load.
- Any field rejection aborts install with a diff message naming the offending path.
- The SDK (`@opendray/plugin-sdk`) ships the schema and a CLI `opendray plugin validate ./dist` that runs offline.

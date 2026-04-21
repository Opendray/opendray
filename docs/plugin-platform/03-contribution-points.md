# 03 — Contribution Points

Every contribution point is a slot in the Workbench. A plugin declares entries under `contributes.*`; the shell renders them. The host validates each entry at load time.

Legend: **MVP** = ships in M5 v1 freeze · **post-v1** = reserved namespace only; no schematisation beyond placeholder.

---

## 1. `activityBar` — MVP

Icons in the left rail (tablet) or bottom navigation overflow (phone).

**Schema** (from manifest §activityBarItem):
```json
{ "id": "myext.activity", "icon": "icons/star.svg",
  "title": "My Ext", "viewId": "myext.view" }
```

**Flutter rendering:**
- Tablet landscape: vertical rail, 56px wide, tooltips on hover.
- Phone portrait: activity bar collapses; icons move into the "More" sheet.

**Limits:** max 4 activity-bar entries per plugin. Combined rail cap 16 items system-wide (overflow to "More").
**Icons:** 24x24 SVG monochrome (recolour follows theme) or emoji.

---

## 2. `views` — MVP

A scrollable view container hosted inside an activity-bar item, side panel, or bottom panel.

**Schema:** see manifest §view.

**Rendering:**
- `render: "webview"` → Flutter embeds a `WebView` widget pointing at `plugin://<name>/<ui.entry>`.
- `render: "declarative"` → host pushes a tree (see [bridge-api §ui](04-bridge-api.md#ui)) that Flutter maps to native widgets. Use this for simple lists.

**Limits:** max 8 views per plugin. A view's declarative tree may have at most 500 nodes.

---

## 3. `panels` — MVP

Bottom-drawer panels (terminal, logs, problems). Swipe up on phone, tabs on tablet.

**Schema:** see manifest §panel.

**Rendering:** identical to views but fixed to the bottom drawer region; always `render: "webview"` unless also a declarative host method returns content.

**Limits:** max 4 panels per plugin.

---

## 4. `commands` — MVP

Named actions addressable by ID from keybindings, menus, status bar, command palette, Telegram, and AI.

**Schema:** see manifest §command. Action kinds:
- `host` — call plugin sidecar method `method` with `args`. Requires `form: "host"`.
- `notify` — toast with `message`. No code required; pure declarative.
- `openView` — focus `viewId`.
- `runTask` — run a contributed `taskId`.
- `exec` — run shell (requires `permissions.exec` matching the args).
- `openUrl` — external browser.

**Example:**
```json
{ "id": "myext.refresh", "title": "Refresh",
  "icon": "icons/refresh.svg",
  "when": "viewFocused == 'myext.view'",
  "run": { "kind": "host", "method": "refresh", "args": [] } }
```

**Limits:** max 64 commands per plugin. IDs namespaced `<pluginName>.<action>`.

---

## 5. `settings` — MVP

Fields rendered as a form in Settings → Plugins → <name>. Superset of the legacy `configSchema` (same shape, added under `contributes.settings` for the v1 path).

**Schema:** see manifest §settingField.

**Rendering:** grouped by `group`; `dependsOn` / `dependsVal` drives conditional visibility. The type `secret` is always masked and stored via `secret` capability, not in plain config.

**Limits:** max 32 fields per plugin.

---

## 6. `statusBar` — MVP

Text + optional icon in the footer bar.

**Schema:** see manifest §statusBarItem.

**Rendering:**
- Tablet: full footer bar.
- Phone portrait: condensed; items past 3 collapse into a "..." menu (right alignment) or scroll (left alignment).

**Limits:** max 2 items per plugin. Text ≤ 24 chars. Updates via `opendray.workbench.updateStatusBar(id, {...})`.

---

## 7. `menus` — MVP

Contextual menus attached to named menu points:
- `editor/context`
- `editor/title`
- `explorer/context`
- `session/toolbar`
- `view/title`
- `commandPalette` (implicit — every command auto-registers)

```json
"menus": {
  "session/toolbar": [
    { "command": "myext.clear", "when": "sessionStatus == 'running'", "group": "navigation" }
  ]
}
```

**Limits:** max 16 entries per menu point per plugin.

---

## 8. `keybindings` — MVP

Chord/key-to-command map.

```json
{ "command": "myext.refresh", "key": "ctrl+shift+r", "mac": "cmd+shift+r",
  "when": "viewFocused == 'myext.view'" }
```

**Rules:**
- User keybindings always override plugin ones.
- Conflicts between plugins: first-loaded wins, Host emits a warning, Settings UI shows the collision.
- Keys must use the VS Code key syntax (`ctrl`, `alt`, `shift`, `meta`/`cmd`, plus `a-z`, `0-9`, `f1..f19`, `escape`, `tab`, `enter`, `backspace`, `space`, `up`, `down`, `left`, `right`).

---

## 9. `editorActions` — MVP

Buttons in the editor title bar / gutter.

```json
{ "id": "myext.format", "title": "Format",
  "when": "resourceLangId == 'json'", "group": "navigation" }
```

An editor action is declared here and bound to a command of the same `id`.

---

## 10. `sessionActions` — MVP

Buttons above a terminal/agent session card (dashboard) and in the session toolbar.

```json
{ "id": "myext.costReport", "title": "Cost", "icon": "icons/$.svg",
  "command": "myext.showCost",
  "when": "sessionType == 'claude'" }
```

**Limits:** max 4 per plugin.

---

## 11. `telegramCommands` — MVP

First-class surface — OpenDray is mobile-first, and Telegram is our remote-control channel.

```json
{ "command": "/deploy", "description": "Deploy current branch",
  "handler": "deployHandler" }
```

- `command` must match `^/[a-z][a-z0-9_]{0,31}$`.
- `handler` is the sidecar method invoked (Host form) or a declarative `run` definition (Declarative form).
- Host automatically adds the command to BotFather registration at activation.
- Requires `permissions.telegram: true`.

**Limits:** max 16 per plugin.

---

## 12. `agentProviders` — MVP

Adds an entry to the "new session" tool picker. Equivalent to today's `plugins/agents/*`.

```json
{ "id": "crush", "displayName": "Crush",
  "kind": "cli",
  "cli": { "command": "crush", "defaultArgs": ["--mobile"] },
  "configSchema": [ { "key":"apiKey","label":"API Key","type":"secret","envVar":"CRUSH_KEY" } ],
  "capabilities": { "supportsResume": true, "supportsStream": true }
}
```

This is how new AI tools get added without touching Go. The Runtime turns each `agentProvider` into a live `plugin.Provider` using the existing `ResolveCLI` path.

---

## 13. `themes` — MVP

Contributes a colour theme JSON file.

```json
{ "id": "neon-night", "label": "Neon Night", "kind": "dark",
  "path": "themes/neon-night.json" }
```

Theme JSON schema follows VS Code's colour token set (subset — see [workbench-slots.md §Theming](08-workbench-slots.md#theming)).

---

## 14. `languages` — post-v1

Reserved. Schema stubbed above. In v1 the host ignores entries but accepts them in manifests so early plugins can future-proof.

## 15. `taskRunners` — post-v1

Schema reserved. Current `plugins/panels/task-runner` stays as a panel plugin in M5.

## 16. `debuggers` — post-v1

Reserved. DAP integration scheduled for M7.

## 17. `languageServers` — MVP (Host form only)

```json
{ "id": "rust-analyzer", "languages": ["rust"],
  "transport": "stdio", "initializationOptions": {} }
```

When a file of a listed language opens, Host activates the plugin (via `onLanguage:*`), starts the sidecar (`form: host`), and proxies LSP messages between editor and sidecar. Only one LSP per language may be active; plugins are tried in installation order and the first one that replies to `initialize` wins.

---

## `when` expressions

Context expressions used across `when` fields:

| Variable | Values |
|----------|--------|
| `activeView` | id of focused view |
| `viewFocused` | id of focused view (alias) |
| `sessionStatus` | `idle` / `running` / `stopped` |
| `sessionType` | agent provider id |
| `resourceLangId` | language of active editor |
| `platform` | `ios` / `android` / `web` / `desktop` |
| `orientation` | `portrait` / `landscape` |

Operators: `==`, `!=`, `&&`, `||`, `!`, parentheses, single-quoted strings, numeric literals.

> **Locked:** `when` syntax is a strict subset of VS Code's, so docs and tooling transfer cleanly.

## Phone vs. tablet adaptation rules

| Slot | Phone portrait | Tablet / landscape |
|------|----------------|--------------------|
| activityBar | bottom nav + "More" | left rail |
| views | full-screen sheet | side panel |
| panels | bottom sheet (swipe) | bottom tab bar |
| statusBar | 2 left + 1 right + overflow | full |
| menus | bottom action sheet | popup menu |
| sessionActions | icon-only, 2 visible + "..." | text + icon |

## Accessibility requirements (enforced by shell)

- Every icon must have a `title` / `tooltip`.
- Minimum tap target 48x48 logical px on phone.
- Colour contrast ≥ 4.5:1 against theme background for text.
- Declarative views must provide `semanticsLabel` on every tappable node (enforced by host validator for post-v1; warning only in v1).

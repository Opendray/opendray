# 08 — Workbench Slots

The Flutter Workbench shell is a fixed layout composed of named slots. Plugins contribute items into slots; the shell decides how to render them for the current form factor.

## Tablet / desktop landscape

```
┌──────────────────────────────────────────────────────────────┐
│  TitleBar                                     [?] [•] [User] │  <- titleBar
├──┬───────────────────────────────────────────────────────────┤
│  │                                                           │
│A │                  Primary area                             │
│c │          (views · editor · terminal)                      │
│t │                                                           │
│i │                                                           │
│v │                                                           │
│i │                                                           │
│t │                                                           │
│y │                                                           │
│  ├───────────────────────────────────────────────────────────┤
│B │                                                           │
│a │                  Bottom panel                             │
│r │        (terminal · logs · problems · tasks · ...)         │
│  │                                                           │
├──┴───────────────────────────────────────────────────────────┤
│  StatusBar: [left items]                   [right items]     │  <- statusBar
└──────────────────────────────────────────────────────────────┘
```

## Phone portrait

```
┌──────────────────────────────┐
│  TitleBar        [≡] [User]  │
├──────────────────────────────┤
│                              │
│     Primary area (one        │
│     view at a time, swipe    │
│     between them)            │
│                              │
├──────────────────────────────┤
│  StatusBar (condensed)       │
├──────────────────────────────┤
│ [📁] [▶] [Δ] [☰ More]        │  <- activityBar (bottom nav, 4 + More)
└──────────────────────────────┘
```

Bottom panel on phone: hidden by default; swipe-up handle at the bottom reveals it as a half-height sheet.

## Slot catalogue

### `titleBar`
- **Renders:** app title, global actions (search, palette toggle), user menu.
- **Contributed by:** host only in v1.
- **post-v1:** plugin title-bar buttons behind a `titleBar` contribution point.

### `activityBar`
- **Contributions:** `contributes.activityBar[]`.
- **Renders:** icon + tooltip; tapping focuses the linked view.
- **Phone:** bottom nav with 4 slots; 5th+ moves to "More".
- **Tablet:** left rail, vertical.
- **Theming tokens:** `activityBar.background`, `activityBar.foreground`, `activityBar.activeBorder`.

### `views`
- **Contributions:** `contributes.views[]`.
- **Renders:** WebView (default) or declarative tree (`opendray.ui`).
- **Phone:** view takes full primary area. Swipe left/right cycles through open views. Back gesture closes.
- **Tablet:** view is anchored to its activity-bar slot; opening another view replaces the current one. Multi-view split is post-v1.
- **Gestures:**
  - Swipe down from top of view: close.
  - Long-press activity icon: pin view (kept open across other interactions).

### `panels` (bottom)
- **Contributions:** `contributes.panels[]`.
- **Phone:** swipe-up bottom sheet; tab bar inside for multiple panels.
- **Tablet:** always-visible tab bar; drag the divider to resize.
- **Kill switch:** tapping the X on the tab removes the panel from view (not uninstall).

### `editorActions`
- **Contributions:** `contributes.editorActions[]`.
- **Renders:** right-aligned icon buttons above the editor content area.
- **Phone:** collapses to 2 icons + overflow menu.

### `sessionActions`
- **Contributions:** `contributes.sessionActions[]`.
- **Renders:** icon buttons on the session card (dashboard) and in the running session toolbar.
- **Phone:** 2 icons visible + "..." menu.

### `statusBar`
- **Contributions:** `contributes.statusBar[]`.
- **Renders:** text + optional icon; tap runs `command` if set.
- **Phone:** max 3 visible, rest collapse.
- **Updates:** push via `opendray.workbench.updateStatusBar`.

### `commandPalette`
- **Auto-populated** from every contributed `commands` entry.
- **Invocation:** titlebar button on tablet; long-press title area on phone.

### `notifications`
- **Auto-populated** by `opendray.workbench.showMessage`.
- Host-managed — plugins cannot contribute a custom notification center.

### `settingsPane`
- **Contributions:** `contributes.settings[]`.
- **Renders:** form inside Settings → Plugins → <name>.

### `menus`
- **Contributions:** `contributes.menus.*`.
- **Renders:** bottom sheet (phone) or popup (tablet) when the anchor menu point opens.

## Theming contract

Themes ship a JSON file keyed by token name; the shell maps tokens to Flutter `ThemeData`. Required tokens for v1 (missing keys fall back to the active light/dark base):

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

Plugin webviews get these as CSS variables (`--od-editor-background` etc.) plus a `data-theme="dark|light|high-contrast"` attribute on `<html>`.

> **Locked:** Theme token names are a strict subset of VS Code names so existing themes can be ported with a renaming pass.

## Gesture & keyboard map (phone)

| Gesture | Effect |
|---------|--------|
| Swipe up from bottom edge | reveal panel |
| Swipe down from top of view | close view |
| Long-press activity icon | pin view |
| Two-finger tap | command palette |
| Edge-swipe from left | previous view |
| Edge-swipe from right | next view |

> **Locked (2026-04-19):** Two-finger + edge-swipe. Three-finger gestures dropped — conflict with iOS VoiceOver and harder to discover. Edge-swipe is unambiguous and works with accessibility enabled.

## Accessibility requirements

- Every interactive slot item must have `semanticsLabel`.
- Screen-reader order follows source order of `contributes.*` arrays.
- Focus ring is theme-aware, contrast ≥ 3:1 against background.
- Minimum tap target 48x48 logical px on phone.
- Reduced motion respected (webviews receive `prefers-reduced-motion: reduce`).

## Slot ownership (Go)

- `gateway/plugins_assets.go` (new) — serves webview bundles.
- `gateway/plugins_bridge.go` (new) — the bridge WebSocket.
- `plugin/slots/` (new) — validates that contribution entries fit slot limits.

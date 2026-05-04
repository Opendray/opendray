# Keyboard & theme

Two pure-cosmetic preference panels under
**Settings → Workspace**. Both persist in browser localStorage
— no server restart required, no other tab affected on a
different machine.

## Theme

Settings → **Workspace → Appearance** picker:

- **Dark** — default, matches the rest of the admin design.
- **Light** — for daylight-monitor use.
- **System** — follows the OS preference; auto-flips when your
  OS dark-mode toggles.

The theme persists in localStorage per browser. There's no
"theme per device" toggle — every browser tab on your account
shares one.

Custom theme tokens (when you want corporate branding) live
under `app/web/src/index.css` — change the CSS variables and
rebuild. opendray doesn't ship a theme editor.

## Font size

Settings → **Workspace → Font size** scales the entire UI
(titles, body text, icons, terminal characters):

| Preset | Scale |
|---|---|
| Compact | 85% |
| Default | 100% |
| Comfy | 115% |
| Large | 130% |

Use Comfy / Large on a 4K display where Default looks too small;
Compact when you have a Sessions workbench packed with tabs.
Persists per browser, same as Theme.

## Keyboard shortcuts

Shortcuts are baked-in (not yet operator-customisable). Below
is the current bindings table.

![Keyboard shortcuts editor](/tutorial/settings-shortcuts.png)

| Default | Action |
|---|---|
| `g s` | Sessions |
| `g c` | Channels |
| `g p` | Providers |
| `g i` | Integrations |
| `g a` | Activity |
| `g n` | Notes |
| `g l` | Plugins |
| `g ,` | Settings |
| `g h` | Tutorial (this page) |
| `n s` | New session (when on Sessions page) |
| `Cmd / Ctrl + K` | Command palette |
| `?` | Keyboard shortcut help dialog |
| `Esc` | Close any open dialog / popup |

## Command palette

Cmd/Ctrl + K opens a fuzzy-searchable command palette. Every
admin action that has a shortcut also lives there, plus things
that don't (e.g. *Create new note*, *Restart this session*).

For multi-step workflows, the palette beats memorising
shortcuts — it shows you what's possible inline.

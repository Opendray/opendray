# Settings — overview

The Settings page is split into three top-level groups, listed
down the left-hand sidebar:

- **Workspace** — per-browser preferences (Appearance / Font /
  Account info)
- **Server** — `config.toml` editor for every gateway-side knob,
  plus a live log tail
- **System** — read-only health view + About

Workspace settings persist in the browser. Server settings are
written back to the gateway's `config.toml` and most require a
**Restart** to take effect. The big "Restart server" button lives
at the bottom of every Server section.

## Workspace

| Item | Purpose |
|---|---|
| **Appearance** | Light / Dark / System theme |
| **Font size** | Compact / Default / Comfy / Large — scales the entire UI |
| **Account** | Logged-in operator + bearer token expiry (read-only) |

## Server (admin only)

Each Server sub-section maps 1:1 onto a `[section]` block in
`config.toml`. The toml-key for every field is shown in mono
small print under the field label so power users can correlate.

| Sub-section | What's there |
|---|---|
| **General** | `listen` address + operator account (user / password / token TTL) |
| **Logging** | level / format / optional file output + live tail console |
| **Sessions** | idle threshold + idle poll interval |
| **Vault** | notes / skills / git roots |
| **MCP registry** | MCP server registry path + secrets file |
| **Storage · Claude** | history roots + accounts dir |
| **Storage · Codex** | sessions root |
| **Storage · Gemini** | tmp root + projects.json |

Every Server form has the same operating model:

1. Edit fields — page tracks dirty state per section.
2. **Save changes** writes back to `config.toml`. Banner shows
   "unsaved" until you click; "restart required" once values
   land that need a process restart.
3. **Reset** discards unsaved changes in the current section.
4. **Restart server** triggers `syscall.Exec` self-replace; the
   UI waits for `/health` to come back, then reloads.

Sensitive fields (`database.url`, `admin.password`) are
returned as empty strings on GET — the running daemon never
echoes them back to the browser. Leaving them blank on Save
preserves the existing value; entering a value overwrites.

## System

| Item | Purpose |
|---|---|
| **Status** | Live `/health` view (status / version / uptime / db reachability) |
| **About** | Project blurb + license |

The little health dot at the bottom of the sidebar mirrors
**Status** so you don't need to navigate there to know if the
DB went away.

## Read on

| Topic | Section |
|---|---|
| Listen address, operator account, password rotation | General |
| Idle threshold and when to tune it | Session defaults |
| Customising shortcuts + theme | Keyboard & theme |
| Live log tail + file output + log levels | Logging |
| Where Claude / Codex / Gemini transcripts live on disk | Storage paths |
| Self-restart workflow + safety guards | Restart |

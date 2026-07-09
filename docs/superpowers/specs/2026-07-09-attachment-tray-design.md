# Image-attachment staging tray (Esc-to-dismiss)

**Date:** 2026-07-09
**Status:** Approved design — ready for implementation plan
**Surfaces:** web (`app/web`) + mobile (`app/mobile`). No backend change.

## Problem

When a user uploads an image to a session, opendray immediately types the returned
server path (`/tmp/opendray-uploads/{sessionId}/{hex}.jpg`) straight into the PTY —
web via `sendInput(res.path)` (`Terminal.tsx`), mobile via `_terminal.paste(...)`
(`session_terminal_view.dart`). There is no attachment UI: the path lands directly
in the running CLI's input line. So pressing **Esc** sends the escape byte to the
CLI, not to opendray, and the user cannot cancel/dismiss the attachment — which is
what they expect, since opendray's other overlays close on Esc.

## Decisions (from brainstorming)

| Decision | Choice |
|---|---|
| Commit model | **Chip + explicit insert** — upload stages a chip; nothing reaches the PTY until the user clicks **Insert**. |
| Count | **Multiple** pending chips (a tray). |
| Chip content | Image icon + filename + **✕**. **No thumbnail** → no new backend endpoint. |
| `Esc` (tray non-empty) | **Clears all** staged chips and is **swallowed** (never reaches the CLI). Empty tray → Esc passes through as today. |
| `✕` | Removes that one chip. |
| Insert | Explicit button — types all staged paths into the terminal (bare `res.path`, space-separated + trailing space), then clears the tray. Content delivered to the CLI is **unchanged** from today. |
| Backend | **None.** Reuses `/api/v1/sessions/{id}/uploads` + `sendInput`/`paste`. |
| Strings | **All new copy via the shared `app/i18n` layer** — web `t()`, mobile slang `t.` (see i18n section). No hardcoded copy. |
| Orphaned upload on cancel | Accepted — the already-uploaded `/tmp/opendray-uploads` file is left unreferenced, same as today. A delete-on-cancel endpoint is out of scope. |

## Architecture & no-hardcoding alignment

- **Shared i18n source.** `app/mobile/slang.yaml` sets `input_directory: ../i18n`, so
  both web (react-i18next) and mobile (slang codegen → `app/mobile/lib/core/i18n/strings_*.g.dart`)
  read from the same `app/i18n/{en,es,zh}.json`. New strings are added there once.
- **Styling** uses Tailwind design tokens (`bg-card`, `accent`, `text-muted-foreground`)
  via `cn`, never literal colors — matching the surrounding code.
- **Esc handling** uses the capture-phase `window` `keydown` pattern already used by
  `WikiLinkSuggestions.tsx`.
- No new hardcoded paths, formats, or magic values; the inserted text is the existing
  `res.path`. The only literal is the mobile Esc byte `'\x1b'` (an escape sequence
  already present in the code).

### Unit 1 — Web: `AttachmentTray.tsx` (new, presentational)

`app/web/src/components/sessions/AttachmentTray.tsx`. Pure, no state:

```
interface AttachmentItem { path: string; name: string }
function AttachmentTray(props: {
  items: AttachmentItem[]
  onRemove: (index: number) => void
  onInsert: () => void
  onClear: () => void
}): JSX.Element | null   // renders null when items is empty
```

Renders a chip row (icon + truncated `name` + `✕` per chip) plus an **Insert**
button. All labels via `t()`. Styled with `cn` + tokens, lucide icons
(`ImageIcon`/`Paperclip`, `X`).

### Unit 2 — Web: `Terminal.tsx` wiring

- Add state `pendingAttachments: AttachmentItem[]` (`useState`).
- Render `<AttachmentTray items={pendingAttachments} … />` directly below the xterm
  container.
- Rewire the three upload paths — `uploadFile` (~L115-140), paste (~L356-374), drop
  (~L398-411): replace `sendInput(res.path)` with
  `setPendingAttachments(a => [...a, { path: res.path, name: file.name }])`.
  Remove the current path-echoing toast (the chip is the feedback); keep error toasts.
- **Insert:** `sendInput(pendingAttachments.map(a => a.path).join(' ') + ' ')`, then
  `setPendingAttachments([])`.
- **`Esc`:** a `useEffect` registers a **capture-phase** `window` `keydown` listener;
  when `e.key === 'Escape'` and `pendingAttachments.length > 0`, call
  `e.preventDefault()` + `e.stopPropagation()` and clear the tray. Cleanup on unmount.

### Unit 3 — Mobile: `session_terminal_view.dart`

- Add widget state `List<_PendingAttachment>` (`{path, name}`).
- `_attachImage`: replace `_terminal.paste(remotePath)` with appending to the list.
- Render a chip row above the keyboard bar (✕ per chip + **Insert** action).
- **Keyboard-bar `Esc`** (currently `_send('\x1b')`, ~L657-663): becomes conditional —
  if the tray is non-empty, clear it; else `_send('\x1b')` as today.
- **Insert:** `_terminal.paste(path)` for each (space-separated), then clear.
- Strings via slang `t.` accessors (see i18n).

### i18n keys (add to `app/i18n/{en,es,zh}.json`)

Under an appropriate group (e.g. `sessions.terminal.attachments` for mobile parity;
web reads the same tree via its own namespace). Keys: `insert`, `clear`,
`removeOne` (✕ aria-label, e.g. "Remove {name}"), `attached` (feedback, e.g.
"Attached {name}"), `title`/`hint` if a tray header is shown. English base plus es/zh
translations, placeholders (`{name}`) identical across locales (i18n-parity CI). After
editing the JSON, **run the mobile slang codegen** (`dart run slang` or build_runner)
so `strings_*.g.dart` regenerates.

## Testing

- **Web:** no JS test runner (repo condition) → `pnpm --filter web exec tsc -b` +
  file-scoped `eslint` on the touched files + a manual pass (upload → chip appears →
  `Esc` clears / `✕` removes one / **Insert** types the path; empty-tray Esc still
  reaches the CLI).
- **Mobile:** `flutter analyze` (after slang codegen) + manual pass mirroring the web
  checks.
- **Backend:** none.

## Out of scope (YAGNI)

- Delete-on-cancel of the orphaned `/tmp/opendray-uploads` file (a follow-up endpoint).
- Thumbnail previews (would need a new image-serving endpoint).
- A full message composer above the terminal.

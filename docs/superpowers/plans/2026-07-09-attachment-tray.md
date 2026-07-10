# Image-attachment staging tray Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stage uploaded images as dismissable chips (Esc/✕ to cancel, explicit Insert to commit) instead of typing the path straight into the PTY, on web and mobile.

**Architecture:** No backend change. Web: a presentational `AttachmentTray` + pending-state in `Terminal.tsx`; upload stages instead of `sendInput`, a capture-phase window `keydown` swallows `Esc` to clear the tray. Mobile: pending-state in `_SessionTerminalViewState`, a chip row above the keyboard bar, and the keyboard-bar `Esc` key clears the tray when non-empty. All strings come from the shared `app/i18n` layer.

**Tech Stack:** React 19 + TypeScript + Tailwind (`cn`) + lucide (web); Flutter + slang i18n (mobile); react-i18next (web i18n). No JS test runner; no Go change.

## Global Constraints

- **No backend change**, no new endpoint. Reuse `uploadSessionFile` / `sessionsApiProvider.uploadFile` + `sendInput` (web) / `_terminal.paste` (mobile). The text delivered to the CLI is **unchanged**: the bare server path, multiples space-separated with a trailing space.
- **All new copy via `app/i18n/{en,es,zh}.json`** — web under `web.sessions.terminal.*`, mobile under `sessions.terminal.attachments.*`. **No literal English in components.**
- **Placeholders use single braces `{name}`** (both slang and this repo's react-i18next are single-brace); the SAME placeholder tokens must appear in all three locales (i18n-parity CI fails on token mismatch).
- After editing the JSON, **regenerate mobile slang**: `cd app/mobile && dart run slang` (rewrites `lib/core/i18n/strings.g.dart` + per-locale files).
- **`Esc` semantics:** clears ALL staged chips and is swallowed (never reaches the CLI) — web via capture-phase `window` listener active only while the tray is non-empty; mobile via the keyboard-bar `Esc` key. Empty tray → `Esc`/`\x1b` behaves exactly as today.
- **Styling** via Tailwind tokens + `cn` (web) and `Theme.of(context).colorScheme` (mobile) — no hardcoded colors.
- **Verify:** web `pnpm --filter web exec tsc -b` + file-scoped `pnpm --filter web exec eslint <paths>` (the full `pnpm build:web` is blocked by a pre-existing root-owned `internal/web/dist` EACCES); mobile `cd app/mobile && flutter analyze`. No unit-test runner on either surface — the gate is typecheck/analyze/lint + a manual pass.
- Commit per task. Open the PR only after all tasks pass and the operator approves; this feature **bundles with #433 into v2.11.3**.

---

### Task 1: i18n keys (shared) + slang codegen

**Files:**
- Modify: `app/i18n/en.json`, `app/i18n/es.json`, `app/i18n/zh.json`
- Regenerate: `app/mobile/lib/core/i18n/strings*.g.dart` (via slang)

**Interfaces:**
- Produces (web, under `web.sessions.terminal`): `attachInsert`, `attachClear`, `attachRemove` (has `{name}`).
- Produces (mobile, under `sessions.terminal.attachments`): `insert`, `clear`, `remove` (has `{name}`).

- [ ] **Step 1: Add the web keys** to the `web.sessions.terminal` object in each locale (place after `dropToAttach`).

`en.json`:
```json
    "attachInsert": "Insert",
    "attachClear": "Clear all",
    "attachRemove": "Remove {name}",
```
`es.json`:
```json
    "attachInsert": "Insertar",
    "attachClear": "Quitar todo",
    "attachRemove": "Quitar {name}",
```
`zh.json`:
```json
    "attachInsert": "插入",
    "attachClear": "全部清除",
    "attachRemove": "移除 {name}",
```

- [ ] **Step 2: Add the mobile keys** as a new `attachments` object inside `sessions.terminal` in each locale (place after the `keyboard` object).

`en.json`:
```json
    "attachments": {
      "insert": "Insert",
      "clear": "Clear all",
      "remove": "Remove {name}"
    },
```
`es.json`:
```json
    "attachments": {
      "insert": "Insertar",
      "clear": "Quitar todo",
      "remove": "Quitar {name}"
    },
```
`zh.json`:
```json
    "attachments": {
      "insert": "插入",
      "clear": "全部清除",
      "remove": "移除 {name}"
    },
```

- [ ] **Step 3: Validate JSON + placeholder parity**

Run:
```bash
cd "$(git rev-parse --show-toplevel)"
for f in en es zh; do python3 -c "import json;json.load(open('app/i18n/$f.json'));print('$f OK')"; done
node scripts/check-i18n-parity.mjs
```
Expected: `en OK` / `es OK` / `zh OK`, and the parity script exits 0 (every locale has the same `{name}` tokens).

- [ ] **Step 4: Regenerate slang**

Run: `cd app/mobile && dart run slang`
Expected: it reports writing `lib/core/i18n/strings.g.dart` (+ `strings_en.g.dart`, `strings_es.g.dart`, `strings_zh.g.dart`) with the new `attachments` keys. Confirm no errors about `{{` interpolation.

- [ ] **Step 5: Commit**

```bash
git add app/i18n app/mobile/lib/core/i18n
git commit -m "i18n(attachments): staging-tray strings (web + mobile) + slang codegen

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 2: Web — AttachmentTray + Terminal staging

**Files:**
- Create: `app/web/src/components/sessions/AttachmentTray.tsx`
- Modify: `app/web/src/components/sessions/Terminal.tsx`

**Interfaces:**
- Consumes: i18n keys `web.sessions.terminal.attach{Insert,Clear,Remove}` (Task 1); existing `sendInput`, `uploadSessionFile`.
- Produces: `interface AttachmentItem { path: string; name: string }` (exported from `AttachmentTray.tsx`); `<AttachmentTray items onRemove onInsert onClear />`.

- [ ] **Step 1: Create `AttachmentTray.tsx`**

```tsx
import { Paperclip, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { cn } from '@/lib/utils'

export interface AttachmentItem {
  path: string
  name: string
}

// A staging tray of pending image attachments, anchored to the bottom of
// the terminal pane. Renders nothing when empty (no layout impact on the
// xterm fit()). Esc-to-clear is owned by Terminal.tsx; this is presentational.
export function AttachmentTray({
  items,
  onRemove,
  onInsert,
  onClear,
}: {
  items: AttachmentItem[]
  onRemove: (index: number) => void
  onInsert: () => void
  onClear: () => void
}) {
  const { t } = useTranslation()
  if (items.length === 0) return null
  return (
    <div className="absolute inset-x-0 bottom-0 z-10 flex items-center gap-2 overflow-x-auto border-t border-border bg-card/95 px-2 py-1.5 backdrop-blur">
      <div className="flex min-w-0 items-center gap-1.5">
        {items.map((item, i) => (
          <span
            key={`${item.path}-${i}`}
            className="flex max-w-[180px] items-center gap-1 rounded-md border border-border bg-background px-1.5 py-0.5 text-[11px]"
          >
            <Paperclip className="size-3 shrink-0 text-muted-foreground" />
            <span className="truncate">{item.name}</span>
            <button
              type="button"
              onClick={() => onRemove(i)}
              aria-label={t('web.sessions.terminal.attachRemove', {
                name: item.name,
              })}
              className="shrink-0 rounded p-0.5 text-muted-foreground hover:bg-card hover:text-foreground"
            >
              <X className="size-3" />
            </button>
          </span>
        ))}
      </div>
      <div className="ml-auto flex shrink-0 items-center gap-1.5">
        <button
          type="button"
          onClick={onClear}
          className="rounded-md px-2 py-0.5 text-[11px] text-muted-foreground hover:bg-background hover:text-foreground"
        >
          {t('web.sessions.terminal.attachClear')}
        </button>
        <button
          type="button"
          onClick={onInsert}
          className={cn(
            'rounded-md bg-primary px-2.5 py-0.5 text-[11px] font-medium text-primary-foreground',
            'hover:opacity-95',
          )}
        >
          {t('web.sessions.terminal.attachInsert')}
        </button>
      </div>
    </div>
  )
}
```

- [ ] **Step 2: Add pending state + handlers in `Terminal.tsx`**

Add the import near the other local imports (after `import { terminalBufferText } from './terminal-text'`):
```ts
import { AttachmentTray, type AttachmentItem } from './AttachmentTray'
```

Add state next to the existing `const [dragActive, setDragActive] = useState(false)` (~L96):
```ts
  const [pendingAttachments, setPendingAttachments] = useState<AttachmentItem[]>([])
```

- [ ] **Step 3: Stage instead of inserting, in `uploadFile`**

In `uploadFile` (~L125-131), replace the success branch:
```ts
        const res = await uploadSessionFile(sessionId, file)
        sendInput(res.path)
        toast.success(t('web.sessions.terminal.uploadedToast'), {
          id: toastId,
          description: res.path,
        })
```
with:
```ts
        const res = await uploadSessionFile(sessionId, file)
        setPendingAttachments((a) => [...a, { path: res.path, name: file.name }])
        toast.dismiss(toastId)
```
(The chip is now the feedback; the loading + error toasts stay.)

- [ ] **Step 4: Add insert / remove / clear + Esc handling in `Terminal.tsx`**

Add these callbacks after `uploadFile` (they can go right before `getBufferText`):
```ts
  const insertAttachments = useCallback(() => {
    if (pendingAttachments.length === 0) return
    sendInput(pendingAttachments.map((a) => a.path).join(' ') + ' ')
    setPendingAttachments([])
  }, [pendingAttachments, sendInput])

  const removeAttachment = useCallback((index: number) => {
    setPendingAttachments((a) => a.filter((_, i) => i !== index))
  }, [])

  const clearAttachments = useCallback(() => setPendingAttachments([]), [])
```

Add an Esc effect (place it near the other `useEffect`s, e.g. after the theme-refresh effect ~L437). It only registers while the tray is non-empty, so an empty tray leaves `Esc` untouched for the CLI:
```ts
  // While attachments are staged, Esc clears them instead of reaching the
  // CLI. Capture phase so we win before xterm's own key handling (same
  // pattern WikiLinkSuggestions uses).
  useEffect(() => {
    if (pendingAttachments.length === 0) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault()
        e.stopPropagation()
        setPendingAttachments([])
      }
    }
    window.addEventListener('keydown', onKey, true)
    return () => window.removeEventListener('keydown', onKey, true)
  }, [pendingAttachments.length])
```

- [ ] **Step 5: Render the tray**

In the `return` (~L439), inside the root `<div ref={rootRef} …>`, add the tray right after the `{dragActive && …}` overlay block (still inside the relative root so it anchors to the bottom):
```tsx
      <AttachmentTray
        items={pendingAttachments}
        onRemove={removeAttachment}
        onInsert={insertAttachments}
        onClear={clearAttachments}
      />
```

- [ ] **Step 6: Typecheck + lint**

Run:
```bash
cd "$(git rev-parse --show-toplevel)"
pnpm --filter web exec tsc -b
pnpm --filter web exec eslint src/components/sessions/Terminal.tsx src/components/sessions/AttachmentTray.tsx
```
Expected: `tsc -b` 0 errors; eslint clean.

- [ ] **Step 7: Manual verification**

In a session's terminal: attach an image (button / paste / drag-drop) → a chip appears in a bottom tray, and the path is **NOT** yet in the terminal. Press **Esc** → tray clears and nothing is sent to the CLI. Attach again → click **✕** → that chip is removed. Attach one or more → click **Insert** → the path(s) are typed into the terminal and the tray clears. With an empty tray, **Esc** still reaches the CLI as before.

- [ ] **Step 8: Commit**

```bash
git add app/web/src/components/sessions/AttachmentTray.tsx app/web/src/components/sessions/Terminal.tsx
git commit -m "feat(web/sessions): stage image uploads in a dismissable tray

Upload no longer types the path straight into the PTY; it stages a chip
(Esc/✕ to cancel, Insert to commit). Esc is swallowed only while staged.

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 3: Mobile — staging tray + keyboard-bar Esc

**Files:**
- Modify: `app/mobile/lib/features/sessions/session_terminal_view.dart`

**Interfaces:**
- Consumes: slang keys `sessions.terminal.attachments.{insert,clear,remove}` (Task 1); existing `_terminal.paste`, `sessionsApiProvider.uploadFile`.
- Produces: `_PendingAttachment{path,name}`; `_AttachmentTray` widget; `_MobileKeyboardBar` gains `hasPendingAttachments` + `onEscClearPending`.

- [ ] **Step 1: Add the model + state**

Near the top of the file's private types (e.g. just above `class _ConnectionAccent`), add:
```dart
class _PendingAttachment {
  const _PendingAttachment({required this.path, required this.name});
  final String path;
  final String name;
}
```
In `_SessionTerminalViewState` (after `late final Terminal _terminal;` ~L39), add:
```dart
  final List<_PendingAttachment> _pending = [];
```

- [ ] **Step 2: Stage instead of pasting, in `_attachImage`**

In `_attachImage` (~L352-368), replace:
```dart
      _terminal.paste(remotePath);
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            t.sessions.terminal.snackbar.imageAttached(path: remotePath),
          ),
          behavior: SnackBarBehavior.floating,
          duration: const Duration(seconds: 3),
        ),
      );
```
with:
```dart
      if (!mounted) return;
      final attachmentName = file.name; // promoted non-null here (outside closure)
      setState(() => _pending.add(
            _PendingAttachment(path: remotePath, name: attachmentName),
          ));
```
(`file` is promoted to non-null after the earlier `if (file == null) return;`; capturing `attachmentName` outside the `setState` closure avoids Dart's closure-promotion ambiguity, so no `!` is needed.)

- [ ] **Step 3: Add insert / clear / remove methods**

Add to `_SessionTerminalViewState` (near `_attachImage`):
```dart
  void _insertAttachments() {
    if (_pending.isEmpty) return;
    _terminal.paste('${_pending.map((a) => a.path).join(' ')} ');
    setState(() => _pending.clear());
  }

  void _clearAttachments() => setState(() => _pending.clear());

  void _removeAttachment(int index) =>
      setState(() => _pending.removeAt(index));
```

- [ ] **Step 4: Add the `_AttachmentTray` widget**

Add near the other private widgets (e.g. above `_MobileKeyboardBar`'s class):
```dart
class _AttachmentTray extends StatelessWidget {
  const _AttachmentTray({
    required this.items,
    required this.onRemove,
    required this.onInsert,
    required this.onClear,
  });
  final List<_PendingAttachment> items;
  final void Function(int index) onRemove;
  final VoidCallback onInsert;
  final VoidCallback onClear;

  @override
  Widget build(BuildContext context) {
    if (items.isEmpty) return const SizedBox.shrink();
    final scheme = Theme.of(context).colorScheme;
    return Container(
      color: scheme.surface,
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 6),
      child: Row(
        children: [
          Expanded(
            child: SingleChildScrollView(
              scrollDirection: Axis.horizontal,
              child: Row(
                children: [
                  for (var i = 0; i < items.length; i++)
                    Padding(
                      padding: const EdgeInsets.only(right: 6),
                      child: Chip(
                        avatar: const Icon(Icons.attach_file, size: 14),
                        label: Text(
                          items[i].name,
                          overflow: TextOverflow.ellipsis,
                        ),
                        deleteIcon: const Icon(Icons.close, size: 14),
                        deleteButtonTooltipMessage: t
                            .sessions.terminal.attachments
                            .remove(name: items[i].name),
                        onDeleted: () => onRemove(i),
                      ),
                    ),
                ],
              ),
            ),
          ),
          TextButton(
            onPressed: onClear,
            child: Text(t.sessions.terminal.attachments.clear),
          ),
          const SizedBox(width: 4),
          FilledButton(
            onPressed: onInsert,
            child: Text(t.sessions.terminal.attachments.insert),
          ),
        ],
      ),
    );
  }
}
```
(`t` is the slang global — it's already imported and used throughout this file.)

- [ ] **Step 5: Render the tray + wire the keyboard bar**

In the parent `build` (~L490-497), insert the tray directly before `_MobileKeyboardBar(...)` and add the two new props to the bar:
```dart
        _AttachmentTray(
          items: _pending,
          onRemove: _removeAttachment,
          onInsert: _insertAttachments,
          onClear: _clearAttachments,
        ),
        _MobileKeyboardBar(
          onKey: _sendKey,
          onSelectText: _openSelectSheet,
          onPaste: _pasteFromClipboard,
          onAttachImage: _attachImage,
          hasPendingAttachments: _pending.isNotEmpty,
          onEscClearPending: _clearAttachments,
        ),
```

- [ ] **Step 6: Add the props to `_MobileKeyboardBar` + conditional Esc**

On the `_MobileKeyboardBar` widget class, add the fields + constructor params:
```dart
  final bool hasPendingAttachments;
  final VoidCallback onEscClearPending;
```
(and `required this.hasPendingAttachments, required this.onEscClearPending,` in its constructor).

In its `build`, change the `Esc` `_Key` (~L656-662) to:
```dart
            _Key(
              label: 'Esc',
              onTap: () {
                _haptic();
                if (widget.hasPendingAttachments) {
                  widget.onEscClearPending();
                } else {
                  _send('\x1b');
                }
              },
            ),
```

- [ ] **Step 7: Analyze**

Run: `cd app/mobile && flutter analyze`
Expected: "No issues found!" (or no new issues attributable to these files).

- [ ] **Step 8: Manual verification**

In a mobile session: attach an image → a chip appears above the keyboard bar; the path is **not** in the terminal. Tap the keyboard bar's **Esc** → the tray clears (and no escape is sent). Attach again → tap the chip's **✕** → it's removed. Tap **Insert** → the path is pasted into the terminal and the tray clears. With no chips, **Esc** sends `\x1b` as before.

- [ ] **Step 9: Commit**

```bash
git add app/mobile/lib/features/sessions/session_terminal_view.dart
git commit -m "feat(mobile/sessions): stage image uploads in a dismissable tray

Parity with web: uploads stage as chips; keyboard-bar Esc clears the tray
when staged, else sends escape.

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Notes for the implementer

- **Do not change** what the CLI receives — Insert sends the same bare path(s) that `sendInput`/`_terminal.paste` sent before, just deferred and space-joined.
- **Do not add** a delete-on-cancel endpoint, thumbnails, or a composer — all out of scope.
- The only literal escape sequence (`'\x1b'`) is pre-existing protocol, not a hardcoding-rule concern.
- No JS/Go test runner is added; the gate is `tsc -b` + `eslint` (web), `flutter analyze` (mobile), plus the manual passes.
- After all three tasks pass, this branch is ready to bundle with #433 for the **v2.11.3** release (changelog entry added at release time).

# kanban — OpenDray M2 reference plugin

Three columns, tap to delete. State saved in `opendray.storage`. Surfaces a banner when any session goes idle via `opendray.events`.

## Try it

1. `OPENDRAY_ALLOW_LOCAL_PLUGINS=1 opendray`
2. `opendray plugin install ./plugins/examples/kanban`
3. Consent screen shows: storage + events(session.idle, session.start, session.stop).
4. Tap the Kanban icon in the activity bar → "+ Add card" → the card persists across restart.
5. Revoke storage in Settings → card add fails with a "permission denied" toast within 200 ms.

## What this plugin proves

- activityBar → view → webview bundle end-to-end
- `opendray.storage.set/get` with cascade-delete on uninstall
- `opendray.workbench.showMessage` delivers a Flutter SnackBar
- `opendray.events.subscribe("session.idle")` fires via HookBus
- CSP enforces `script-src 'self' 'unsafe-eval'` — fetching `https://evil.com` fails
- Capability hot-revoke under 200 ms

## Files

```
kanban/
├── manifest.json    — v1 manifest, declares activityBar + views + storage + events permissions
├── README.md        — this file
└── ui/
    ├── index.html   — the webview entry (matches manifest.contributes.views[0].entry)
    ├── main.js      — uses window.opendray.storage / workbench / events
    └── styles.css   — dark-themed board
```

## M1 + M2 versus M3+

Kanban deliberately stays within the M2 bridge surface: **storage**, **workbench.showMessage**, **events.subscribe**. It does not call `fs.*` / `exec.*` / `http.*` (those land in M3) nor `commands.execute` / `tasks.*` (post-v1). Writing a new plugin that targets M2 should start from this one and delete what it doesn't need.

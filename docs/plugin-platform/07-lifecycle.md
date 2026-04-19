# 07 — Plugin Lifecycle

## State machine

```
                    install requested
                          │
          uninstalled ───►│
                          ▼
                    downloading ──── failure ─── uninstalled
                          │
                   verify + consent
                          │
                          ▼
                     installed ◄──── disable ───── enabled
                          │
                          ▼                          ▲
                      enabled ──────────────────────┘
                          │
                   activation event fires
                          │
                          ▼
                    activating ──── failure ──── enabled (error flag)
                          │
                          ▼
                       active  ────── idle timeout (host form) ──┐
                          │                                        │
                    deactivation event                             │
                          │                                        │
                          ▼                                        │
                   deactivating ◄────────────────────────────────┘
                          │
                          ▼
                      enabled
                          │
                     uninstall
                          │
                          ▼
                    uninstalling
                          │
                          ▼
                    uninstalled
```

## State semantics

| State | Meaning | Visible in UI |
|-------|---------|---------------|
| `uninstalled` | not on disk | Marketplace |
| `downloading` | fetching bytes | progress in Installing row |
| `installed` | bundle on disk, manifest validated, user has NOT consented yet | "Needs review" |
| `enabled` | consented, registered in DB, eligible to activate | Settings |
| `activating` | host is starting sidecar / loading webview assets | spinner |
| `active` | plugin is running; bridge calls accepted | green dot |
| `deactivating` | `shutdown` sent, waiting up to 2 s | |
| `disabled` | registered but blocked — no activation events delivered | toggle off |
| `uninstalling` | removing files and DB rows | |

## Activation events

A plugin activates when any of its manifest `activation` events fires. `onStartup` activates at host boot.

| Event | Fires when |
|-------|-----------|
| `onStartup` | host boot |
| `onCommand:<id>` | any caller runs that command |
| `onView:<id>` | user opens that view |
| `onSession:start` / `stop` / `idle` / `output` | from `HookBus` |
| `onLanguage:<id>` | editor opens a file of that language |
| `onFile:<glob>` | editor opens a file matching glob |
| `onSchedule:cron:<expr>` | scheduled; cron parsed with 5-field syntax |

A plugin can list up to 32 events. If list is empty, the plugin never auto-activates; user must activate manually.

## Idle shutdown (host form only)

If the sidecar has received no requests for `host.idleTimeoutSec` (default 300 s) AND no active subscription is open, host sends `shutdown` and the plugin returns to `enabled`. Next activation event re-starts it.

> **Locked:** iOS never activates host form plugins. On iOS, `onLanguage:*` and `onSession:output` for a host plugin silently no-op and a one-time banner explains why.

## Install flow

1. `POST /api/plugins/install` with body `{ src }`.
   `src` is one of:
   - `marketplace://<publisher>/<name>@<version>`
   - `https://...path/to/bundle.zip`
   - `local:/abs/path/to/bundle/` (dev mode only, gated by `OPENDRAY_ALLOW_LOCAL_PLUGINS=1`)
2. Host downloads, sha256-verifies (and signature-verifies if applicable).
3. Host validates manifest against JSON Schema v1.
4. Host computes capability diff vs. installed version (if any).
5. Host returns `202` with a consent token and list of caps.
6. UI shows consent screen; user confirms.
7. `POST /api/plugins/install/confirm {token}`.
8. Host extracts bundle to `plugins/.installed/<name>/<version>/`.
9. Host writes consent row pinning manifest hash.
10. Host calls `Runtime.Register()` (existing) and seeds DB.
11. If manifest lists `onStartup`, host activates immediately.

## Update flow

- New version downloaded with same flow.
- If capability diff is **equal or narrower**, update applies silently.
- If capability diff is **broader**, re-prompt for consent before activating new version.
- Old version kept at `plugins/.installed/<name>/<oldVersion>/` until next GC (24 h) so rollback is cheap.
- Manifest `engines.opendray` must satisfy current host version, else install fails with `EINCOMPAT`.

## Migration on manifest schema bump

- Manifest has an implicit `schemaVersion = 1`. v2 may add required fields; v1 plugins continue to load via a compat shim that defaults them.
- Plugins can bump `engines.opendray` to opt into newer hosts; older hosts refuse to install.
- Host never rewrites the bundle on disk. Migration is read-side only.

## Crash recovery

### Webview crash
- Flutter detects WebView process death → shell shows "Plugin crashed — reload?" action.
- `crashes` counter per plugin per day. 3+ crashes → plugin auto-disabled with a notification.

### Sidecar crash
- Supervisor restarts with exponential backoff (2s, 4s, 8s, 16s, 32s). After 5 failures in 10 minutes, plugin auto-disabled.
- Crash dumps (stderr tail ≤ 64 KB) stored in `plugins/.crash/<name>-<ts>.log`; surfaced in Settings → Plugins → Logs.

### Host crash (the whole Go process dies)
- Entire app restart. All plugins go `enabled` → reactivate according to manifest events. No plugin can prevent boot.

## Uninstall flow

1. `DELETE /api/plugins/<name>`.
2. Host calls sidecar `deactivate` + `shutdown`, waits 2 s, then SIGKILL.
3. WebView unloaded.
4. `Runtime.Remove()` — existing method.
5. Delete `plugin_kv` rows (unless user chose "keep my data"), `plugin_consent`, `plugin_audit`.
6. Remove extracted bundle dir.
7. Emit event `plugin.uninstalled`.

## Compatibility mode for current manifests

Current manifests under `plugins/agents/*` and `plugins/panels/*` do NOT have:
- `form`, `publisher`, `engines`, `contributes`, `permissions`.

At load, the Host synthesises a compat manifest:
```
form      = "host" if type in ("cli","local","shell") else "declarative"
publisher = "opendray-builtin"
engines   = { "opendray": ">=0" }
contributes.agentProviders = [Provider as-is]   // for agent manifests
contributes.views = [{ id: <name>, title: displayName, container: "activityBar", render: "webview" }]  // for panel manifests
permissions = {} // host trusts builtins
```
These synthesised manifests are held in memory only; the on-disk file is never rewritten. Builtins are treated as "trusted publisher" and skip the consent screen.

> **Locked:** Compat mode is permanent for v1. When v2 ships, builtins will be migrated one by one to native v1 manifests without breaking.

## Owner packages

- `plugin/install/` (new) — download, verify, extract, consent.
- `plugin/runtime.go` (existing) — registration + DB.
- `plugin/host/supervisor.go` (new) — sidecar lifecycle.
- `plugin/lifecycle/activation.go` (new) — activation-event dispatcher hooked to `HookBus`.

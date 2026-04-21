# 05 — Capabilities & Consent

OpenDray follows a Chrome-extension / Android model: plugins declare capabilities up front in `manifest.permissions.*`, the user sees a consent screen at install time, and every bridge call is checked at runtime.

## 1. Capability taxonomy

| Key | Grants | Default | Consent phrasing |
|-----|--------|---------|-------------------|
| `fs.read`  | Read files matching patterns | none | "Read files in: /<patterns>/" |
| `fs.write` | Write files matching patterns | none | "Create/modify files in: /<patterns>/" |
| `exec`     | Run processes (globs or full) | none | "Run commands: <globs>" or "Run any command on this machine" |
| `http`     | Outbound HTTP (URL patterns or full) | none | "Make network requests to: <hosts>" |
| `session`  | Read or write sessions | none | "Read terminal output" / "Send input to terminals" |
| `storage`  | Per-plugin KV (isolated) | none | "Store plugin-specific data" |
| `secret`   | Per-plugin secret store | none | "Store encrypted credentials for this plugin" |
| `clipboard`| Read / write device clipboard | none | "Read clipboard" / "Write clipboard" |
| `telegram` | Send via the configured bridge | none | "Send messages through your Telegram bot" |
| `git`      | Read or write git repos in workspace | none | "Read git history in your workspace" / "Commit changes" |
| `llm`      | Use your configured LLM providers (costs tokens) | none | "Call your AI models (may incur cost)" |
| `events`   | Subscribe to host events (pattern list) | none | "Listen for: <pattern list>" |

## 2. Pattern language

### Path patterns (for `fs.read`, `fs.write`)
- Base variables: `${workspace}`, `${home}`, `${dataDir}`, `${tmp}`.
- Glob syntax: doublestar (`**`), `*`, `?`, character class `[abc]`.
- Example: `["${workspace}/**/*.md", "${home}/.config/myext/**"]`.
- **Disallowed:** bare `/`, `/etc/**`, `/var/**`, `/proc/**`, `/sys/**`. Installer refuses manifest.

### URL patterns (for `http`)
- Scheme must be `https://` except when host is `localhost` or explicit LAN.
- Wildcard on host: `https://*.example.com/*`.
- Example: `["https://api.github.com/*", "https://objects.githubusercontent.com/*"]`.
- RFC1918 / link-local are denied even if pattern-matched unless the plugin is installed from a `local:` source (dev mode).

### Command patterns (for `exec`)
- Space-separated `cmd args...` globs: `"git *"`, `"npm run *"`, `"cargo test --no-run *"`.
- Single `true` grants everything and forces the consent screen to show the "run anything" red banner.

## 3. Install-time consent UI

Flutter shell renders a full-screen sheet on install:

```
┌─────────────────────────────────────────────┐
│  Install "rust-analyzer-od"                 │
│  v0.2.0 by rust-lang                        │
│  Verified publisher · signature valid       │
│─────────────────────────────────────────────│
│ This plugin wants to:                       │
│                                             │
│ • Read & modify files in your workspace     │
│ • Run commands: cargo *, rustc *            │
│ • Store plugin-specific data                │
│                                             │
│ DANGEROUS:                                  │
│ • (none)                                    │
│                                             │
│ [ Cancel ]               [ Install ]        │
└─────────────────────────────────────────────┘
```

Rules:
- Dangerous group includes `exec: true`, `fs.write` matching `${home}` root, `http` without host restriction, `session: "write"`, `telegram: true`.
- A dangerous grant forces a second-tap confirm with text "I understand the risk".
- Unsigned plugin + dangerous grant + `store: "community"` → extra "From an unverified publisher" warning.

## 4. Runtime prompts

Most capabilities are granted statically at install. Two run-time prompts exist:

1. **`clipboard: "read"`** — browsers and iOS require a user gesture to read clipboard; plugin's first `clipboard.readText()` in a session prompts.
2. **First `llm.*` call in a billing period** — shown once per 30 days with current token spend.

Other capabilities do **not** prompt at runtime; a denied call returns `EPERM` immediately.

## 5. Revocation

From Settings → Plugins → <name> → Permissions:
- Per-capability toggle (off ⇒ revoke; on ⇒ grant back).
- Revoking writes an event `plugin.permission.revoked`; active webviews see the next call fail with `EPERM`; sidecars are sent a `permissions/update` JSON-RPC notification and the process is restarted if it ignores it for 2 s.

## 6. Audit log

Every bridge call is written to a ring buffer (10k entries per plugin) with:
```
{ ts, plugin, ns, method, caps: ['fs.read'], result: 'ok|denied|error', durationMs, argsHash }
```
Viewable in Settings → Plugins → <name> → Audit. Retention 30 days on disk, then GC.

## 7. Implementation notes

- Go package owning enforcement: `plugin/bridge/capabilities.go` (new).
- DB tables (M1 migration):
  - `plugin_consents(plugin_name, granted_at, manifest_hash, perms_json)` — pinning the consent to the exact manifest hash means a manifest change that broadens capabilities re-triggers consent.
  - `plugin_audit(id, ts, plugin_name, ns, method, result, meta)`.

## 8. What plugins can never obtain

Even with every capability granted, the following stay host-only:

- Writing to the `plugins/` directory (bundles are immutable after install; updates go through the install flow).
- Writing to `kernel/store` tables other than `plugin_kv` and their own `plugin_secret` rows.
- Accessing another plugin's `storage` or `secret`.
- Spawning processes without going through `opendray.exec` (sidecar `os.exec` at the kernel level is allowed but policy-gated by `exec` capability — the host intercepts syscalls on Linux via the sidecar supervisor's rlimit/seccomp wrapper when available, and on macOS/iOS relies on policy + audit).
- Overriding another plugin's manifest-declared contributions.
- Reading another plugin's audit log.

> **Locked:** The list in §8 is the hard guarantee. Any new host subsystem must document which side of this line it lives on before shipping.

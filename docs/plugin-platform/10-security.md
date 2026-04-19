# 10 — Security

## Threat model

### Assets
- User's source code in workspace (high value).
- Secrets (API keys, tokens, cloud creds) in `secret` store and in `EnvOverrides`.
- LLM provider billing (tokens cost real money).
- Telegram bot token and allow-listed chat IDs.
- Host machine (opendray runs with user privileges).

### Adversaries
1. **Malicious plugin author** — writes a plugin that steals secrets or exfiltrates code.
2. **Compromised marketplace** — attacker gains write to the registry repo or to a publisher's artifact URL.
3. **Man-in-the-middle** — intercepts plugin download or update check.
4. **Stolen auth token** — attacker has the user's bearer token, talks to the bridge directly.
5. **Physical-access attacker** — device unlocked, installs a plugin themselves.
6. **Supply-chain** — a dependency of a host plugin (npm package, system library) is compromised.

### Out of scope
- Kernel-level escalations (opendray runs as the user; a plugin can do whatever the user could, modulo capability gate).
- Denial of service of the marketplace registry (we accept downtime; install is optional).

## Mitigations matrix

| Threat | Mitigation |
|--------|-----------|
| Malicious author | Capability-declaration + consent screen; least-privilege defaults; runtime bridge gate; audit log. |
| Compromised marketplace (registry) | Signed publisher keys; optional artifact signature verification; revocation list. |
| Compromised marketplace (artifact URL) | sha256 pinning in registry file; signature check; mirrors. |
| MITM | HTTPS-only for registry and artifacts; HSTS on official host; TOFU-pinned publisher keys after first install. |
| Stolen bearer token | Bridge WebSocket binds to origin + Host header; token is per-session and re-issued on login; rate limits. |
| Physical access | App lock (PIN/biometric) on mobile; sensitive caps prompt on first use. |
| Supply-chain | Bundle is self-contained; Host plugins cannot fetch code at runtime (CSP for webview, no network by default for sidecars unless `http` capability granted). |

## Hard guarantees (what a plugin can never do)

1. Access the `plugins/` install directory for writing.
2. Read or write another plugin's `storage` or `secret` namespace.
3. Override another plugin's contributions.
4. Escape the WebView sandbox without the user installing a host-form plugin.
5. Spawn processes without its `exec` capability and matching glob.
6. Make outbound network calls without its `http` capability and matching URL pattern.
7. Read files outside its declared `fs.read` globs.
8. Write files outside its declared `fs.write` globs.
9. Read another app's private data on iOS/Android (OS sandbox).
10. Bypass the capability gate by directly calling host HTTP endpoints — every `/api/plugins/*` endpoint binds the caller to a specific plugin via the bridge token issued at activation.

> **Locked:** The list above is the hard wall. Any new bridge method ships with a capability review and must not breach these guarantees.

## iOS App Store strategy

### Why this is NOT dynamic code under §2.5.2

- Declarative plugins contain **no executable code** — only JSON data that the native Flutter shell renders. Equivalent to a theme file.
- Webview plugins run inside `WKWebView`. Apple explicitly permits web content inside WebViews. The plugin's JS is content, not an alternative runtime. This is the same model used by every CMS, docs viewer, and Markdown app.
- Host plugins are **disabled on iOS** (§06). No second process is launched. iOS builds simply do not ship the supervisor.

### Additional iOS-specific constraints (enforced in iOS build)

- `exec` capability is unavailable on iOS (no `Process`/`posix_spawn` surface). Bridge method returns `EUNAVAIL`.
- Plugin bundles cannot ship binaries on iOS; install refuses bundles with `form: "host"`.
- Webview plugin downloads stored in the app's Documents directory, scoped to the app; Apple reviewers can see them listed.
- No plugin can disable the OS system clipboard prompt.

### App review notes to include in submission

- "Plugins are UI and data extensions. All executable plugin code is web content inside WKWebView. The app ships an empty plugin directory by default; users can browse a registry that displays descriptions and install pre-approved web content, analogous to a headless CMS or theme store. No dynamic native code is loaded at any point."

> **Locked (2026-04-19):** All 11 current core panels (`plugins/panels/*`) ship baked into the build on every platform, pinned by hash. Post-launch, winnow to the 5-7 most-used based on telemetry; the rest move to marketplace-only. This guarantees OpenDray is fully usable with zero installs on iOS and is review-safe.

## User trust levels and UX

| Level | Consent-screen treatment |
|-------|-------------------------|
| official | No warnings; green badge. |
| verified | "Verified publisher" blue badge; normal consent. |
| community | "Published by <publisher>" text; amber banner for dangerous caps. |
| sideloaded | "From a local file or URL" red banner; forces the user to type the plugin name to confirm for any dangerous cap. |

## Secrets handling

- Plugin secrets stored AES-GCM encrypted under the host's credential-store key.
- Secret values are never logged, never included in audit-log payloads (only key names are logged), never returned to a webview chrome DOM inspector hook.
- Host redacts any string matching the value of a plugin secret from log output.
- Environment variables set by a plugin's `envVar` configSchema entry are scoped to the spawned session only.

## Network policy

- Bridge WebSocket accepts connections from:
  - `app://opendray` (the Flutter app's custom scheme on mobile).
  - `https://<configured-frontend-host>`.
  - `http://localhost:<port>` only when bound to localhost.
- CSRF: every mutation endpoint requires the bearer token in `Authorization`; cookies not accepted.
- CORS: allowlist driven by the frontend host config.

## Audit and incident response

- Central log at `plugins/.audit/<date>.jsonl`.
- Emergency "all plugins disabled" flag via Settings → Plugins → Panic button and via an `OPENDRAY_PLUGINS_DISABLED=1` env var. Host boots, loads manifests, but refuses all activations.
- On detecting a revocation entry with `action: "uninstall"`, host performs the uninstall and emits a desktop/mobile notification with the stated reason.

## What CANNOT be extended (hard guarantees)

Listed for clarity to plugin authors so they don't design around it:

- The auth/login flow.
- The session spawning code path (only contribute `agentProviders` — host remains the spawner).
- The bridge protocol itself (no wire-level middleware from plugins).
- The theming token set (themes pick values, they don't add tokens).
- The capability system (no plugin can weaken or elevate another's capabilities).
- The marketplace index schema (plugins can mirror, not reinterpret).
- The OS-level sandbox of WebView / sidecar.

> **Locked:** These are the parts we never want to own the extensibility cost of. Post-v1 we may open some (e.g. custom auth providers), always behind a new capability.

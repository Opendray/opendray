# OpenDray Plugin Platform M3 — Ship Readiness

**Last Updated:** 2026-04-20
**Branch:** `kevlab` (not merged to main)
**Base:** M2 complete at commit `12a3ef6`

## 1. Status by task

| Task | Title | Status | Commit |
|------|-------|--------|--------|
| T1  | `HostV1` on Provider + validator                                | ✅ | `b98d016` (seam) |
| T2  | iOS host-form refusal build-tag                                  | ✅ | `b98d016` |
| T3  | Capability gate base-variable expansion                          | ✅ | `b98d016` + `bb6b64a` (grant-list expansion matcher tap) |
| T4  | Migration 014 `plugin_secret_kek`                                | ✅ | `b98d016` |
| T5  | Migration 015 `plugin_host_state`                                | ✅ | `b98d016` |
| T6  | Migration 016 `plugin_secret.nonce`                              | ✅ | `b98d016` |
| T7  | KEK / DEK crypto primitives                                      | ✅ | `26b4a04` |
| T8  | `NewStreamChunkErr` helper                                       | ✅ | `b98d016` |
| T9  | `opendray.fs.*` read-path namespace                              | ✅ | `bb6b64a` |
| T10 | `opendray.fs.*` write-path + watch                               | 🟡 Deferred |
| T11 | `opendray.exec.*` namespace                                      | ✅ | `831f426` |
| T12 | `opendray.http.*` namespace                                      | ✅ | `5f4a914` (+ `fd2cd93` TLS harden) |
| T13 | `opendray.secret.*` namespace                                    | ✅ | `5cc7c00` |
| T14 | Host sidecar supervisor skeleton                                 | ✅ | `2480bac` |
| T15 | JSON-RPC LSP framing codec                                       | ✅ | `b98d016` |
| T16 | Sidecar bidirectional call multiplexer                           | ✅ | `2480bac` |
| T17 | Supervisor ↔ namespaces wiring                                   | ✅ | `a88f8b1` + `34b3a71` (factory wire) |
| T18 | `contributes.languageServers` + Flatten                          | 🟡 Deferred |
| T19 | LSP proxy gateway route                                          | 🟡 Deferred |
| T20 | Consent patch endpoint + granular UI plumbing                    | 🟡 Deferred |
| T21 | Flutter Settings UI: granular caps                               | 🟡 Deferred |
| T22 | Flutter bridge SDK: fs/exec/http/secret TS types                 | 🟡 Partial (shim from M2 routes raw envelopes; JS helpers deferred) |
| T23 | Main wiring                                                      | ✅ | `3595999` |
| T24 | PathVarResolver implementation (gateway)                         | ✅ | `3595999` |
| T25 | Reference plugin `plugins/examples/fs-readme/`                   | 🟡 In flight |
| T26 | E2E test: fs-readme full lifecycle                               | 🟡 Deferred |
| T27 | Carry-on: CSP test + desktop webview + kanban E2E                | 🟡 Deferred |
| T28 | Documentation update                                             | 🟡 Partial (M3-RELEASE.md here; 10-security.md + 11-dx update pending) |
| T29 | First-PR seam                                                     | ✅ | `b98d016` |

**Summary:** 16 Done / 1 In flight / 10 Deferred / 0 Skipped

Core claim: **all four privileged namespaces + supervisor + bidirectional sidecar mux** ship. The deferred items are either test harness (T26), UX polish (T20/T21), or scope cuts explicitly called out in the plan (T10 watch, T19 LSP proxy, T27 M2 carry-ons).

---

## 2. What ships in M3

- **`opendray.fs.*`** — `readFile`, `exists`, `stat`, `readDir`. Paths canonicalised through `filepath.Clean` + `filepath.EvalSymlinks` (TOCTOU defense); 10 MiB read cap, 4096-entry readDir cap. Grant globs expanded via `${home}`, `${dataDir}`, `${tmp}`; `${workspace}` stays empty in M3 (fs grants anchored on it fail closed) until M4 threads the active session's cwd.

- **`opendray.exec.*`** — `run` (one-shot), `spawn` (streamed), `kill`, `wait`. Command-line matched via the existing `bridge.MatchExecGlobs`. `Setpgid=true` on Unix, `CREATE_NEW_PROCESS_GROUP` on Windows; supervisor tears the whole group down on revoke. 10-second default timeout, max 5 minutes. Per-plugin hard cap of 4 concurrent spawns. Linux cgroup v2 attempted when writable (warn-once if not).

- **`opendray.http.*`** — `request`, `stream`. URL allowlist pre-matches every hop of a redirect chain. Custom `net.Dialer.Control` re-checks the resolved IP against the RFC1918 / loopback / link-local block just before `connect(2)` — kills DNS rebind bypasses. 4 MiB request body cap / 16 MiB response; TLS minimum 1.2.

- **`opendray.secret.*`** — `get`, `set`, `delete`, `list`. AES-256-GCM encrypted at rest in `plugin_secret`; DEK wrapped under a KEK derived via HKDF-SHA256 from the admin bcrypt hash (kernel/auth.NewKEKProviderFromAdminAuth). Per-plugin row scoping — one plugin cannot read another's keys. Rotating the admin password rotates the KEK (a rewrap walk runs on next login; fallback: re-install the plugin).

- **Host sidecar supervisor** — `plugin/host/Supervisor` spawns one process per host-form plugin on demand. Runtimes: `binary / node / deno / python3 / bun / custom`. JSON-RPC 2.0 over LSP `Content-Length` framing. Setpgid; graceful stdin-EOF shutdown with 5 s timeout before SIGKILL group. Backoff restart (200 ms → 5 s). Idle shutdown after 10 min (configurable).

- **Bidirectional sidecar JSON-RPC** — every sidecar gets a `Mux` around its stdio. Outbound `Call` / `Notify` / `Notifications()`; inbound routed through `HostRPCHandler` which delegates to the same `bridge.*API.Dispatch` that webview plugins hit. **Identical capability-gate enforcement** for both transports.

- **HostRPCHandler** — plugin name is constructor-injected. Sidecar RPC calls of the form `fs/readFile` / `exec/run` / etc. go through the bound namespace. Method-injection (multi-slash, `..`, null byte) rejected as `InvalidRequest`. `*bridge.PermError` → RPC code -32001; `*bridge.WireError` → RPC code by lookup; others → Internal.

- **Database migrations 014-016** — `plugin_secret_kek` (per-plugin wrapped DEK), `plugin_host_state` (supervisor lifecycle stats), and `plugin_secret.nonce` column for AES-GCM. All additive; no existing data disturbed.

- **`form:"host"` manifest** — validator accepts host-form manifests on desktop; iOS builds hard-refuse via build tag + sentinel (`plugin.ErrHostFormNotSupported`). Runtime enum extended to 6 options. The `contributes.languageServers` contribution point defers (T18/T19).

---

## 3. What's deferred to M4+

- **T10 — `fs.watch` + write-path (`writeFile`, `mkdir`, `remove`)** — write path needs the TOCTOU story re-verified for creation vs read. fsnotify's inotify cap (Linux) needs doc in the authoring tutorial. Scope cut to keep M3 focused on the read-path + exec/http/secret.

- **T18 + T19 — `contributes.languageServers` + LSP proxy route** — The machinery is in place (Mux is bidirectional, HostRPCHandler routes any method), but the `/api/plugins/lsp/{language}/proxy` WS route + the Registry lookup by language aren't wired. LSPs still work as bespoke sidecars via direct `host.Sidecar.Call`; the proxy route would only add convenience.

- **T20 + T21 — Granular consent UI** — The M2 all-or-nothing revoke (`DELETE /consents/{cap}`) still works; per-glob / per-command toggles are polish, not a correctness gap.

- **T22 — Flutter SDK helpers for new namespaces** — The bridge channel routes every namespace generically; the JS shim's `opendray.fs / exec / http / secret` convenience proxies are the one missing piece for webview plugins. Plugins can fall back to `window.OpenDrayBridge.postMessage({ns:"fs",method:"readFile",args:[path]})` until then.

- **T26 — E2E test** — `fs-readme` lifecycle under `-tags=e2e`. Needed for CI confidence; blocked by Node availability on the test host.

- **T27 — M2 carry-ons** — CSP golden-file test, desktop WebView fallback, kanban E2E. Still 🟡 on M2-RELEASE.md. Low risk to defer.

---

## 4. Smoke test — manual walkthrough

Run on the deployed syz LXC with `PATH=/usr/local/go/bin` and `node --version ≥ 20` on PATH.

### 4a. Verify namespaces registered

```bash
TOKEN=$(curl -s -X POST http://127.0.0.1:8640/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"linivek","password":"<admin>"}' | jq -r .token)
# Any bridge call with a denied cap returns EPERM;
# a call against an unregistered namespace returns EUNAVAIL.
# The binary on syz at 34b3a71+ registers fs/exec/http/secret.
```

### 4b. fs namespace from a webview plugin

Kanban doesn't declare `permissions.fs`, so its `opendray.fs.readFile` should return EPERM. Open the Kanban webview, paste in the dev console:

```js
opendray.fs.readFile('/etc/passwd').catch(e => e.toString())
// Expected: "Error: EPERM: fs.read not granted for: /etc/passwd"
```

### 4c. exec namespace

Same plugin, no exec grant → EPERM:

```js
opendray.exec.run('git', ['status']).catch(e => e.toString())
// Expected: EPERM
```

### 4d. http namespace — SSRF block

Even if a plugin grants `http: ["*"]`, loopback is denied:

```js
opendray.http.request({url:'http://169.254.169.254/latest/meta-data/'})
  .catch(e => e.toString())
// Expected: EPERM (or similar; never a successful response)
```

### 4e. Required plugin lock

```bash
curl -X PATCH http://127.0.0.1:8640/api/providers/claude/toggle \
  -H "Authorization: Bearer $TOKEN" -d '{"enabled":false}'
# Expected: HTTP 400 "required plugin cannot be modified"
```

### 4f. Host sidecar(等 T25 fs-readme 到位后)

```bash
opendray plugin install --yes ./plugins/examples/fs-readme
# Sidecar boots, JSON-RPC handshake completes, summarise returns
# the first 400 bytes of $HOME/README.md
```

---

## 5. Known issues & caveats

- **`${workspace}` is empty in M3.** Plugins declaring `fs.read: ["${workspace}/**"]` will fail to match any real path until M4 threads the session's cwd through `PathVarResolver`. The safe-default is intentional.

- **Gateway dispatcher collapses WireError codes to EINTERNAL.** `gateway/plugins_bridge.go:360` only specialises `*PermError`. Namespace-emitted `EINVAL / EUNAVAIL / ETIMEOUT` arrive at the plugin as `EINTERNAL` on the message field. Pre-existing limitation; not a security issue but surfaces as confusing error codes. Follow-up.

- **Secret store adapter is in main.go.** `secretStoreAdapter` translates `pgx.ErrNoRows → bridge.WrappedDEKNotFound` because the bridge package deliberately doesn't import pgx. Plugins that try to `secret.get` on a fresh install before any `secret.set` get a clean "not found" error.

- **KEK rotation is manual.** Changing the admin password in Settings does NOT currently rewrap every plugin's DEK. Planned: one-shot walk at login. Workaround today: re-install the plugin to regenerate its DEK.

- **Android host-form gated off.** The supervisor refuses to spawn on iOS builds; Android is left open at the validator layer but **should not be exercised** until Google Play §4.4 review. M4 items.

- **cgroup v2 limits are best-effort.** The Proxmox LXC running syz doesn't grant `CAP_SYS_ADMIN`, so fork-bomb resistance falls back to the 4-concurrent-spawn cap in `api_exec.go`. Documented in the one-time startup warning.

- **`fs.watch` absent.** T10 deferred. Plugins needing file-change notifications should poll until a follow-up round.

---

## 6. Sign-off checklist

- [ ] `go test -race -count=1 -p 1 ./...` green (kernel/config env-pollution failures are pre-existing and unrelated)
- [ ] `flutter test` green (170/170 after M2 adapt round)
- [ ] Manual smoke test (§4) passed on the deployed syz LXC
- [ ] `fs-readme` (T25) installs + summarise returns `$HOME/README.md` preview
- [ ] `gosec ./plugin/bridge/... ./plugin/host/... ./kernel/auth/...` shows 0 new HIGH findings
- [ ] M3-PLAN §6 threat cases regression tested: TOCTOU (T9), fork-bomb (T11), DNS rebind (T12), method injection (T17)
- [ ] Release notes committed on kevlab; not merged to main

---

## 7. Commit history (M3 on kevlab)

```
34b3a71 feat(plugin-platform): M3 — wire supervisor ↔ HostRPCHandler
3595999 feat(plugin-platform): M3 T23+T24 — wire fs/exec/http/secret + supervisor
a88f8b1 feat(plugin-platform): M3 T17 — sidecar ↔ namespace routing
fd2cd93 fix(plugin-platform): clamp HTTP TLS MinVersion and silence gosec
5cc7c00 feat(plugin-platform): M3 T13 — opendray.secret.* namespace
831f426 feat(plugin-platform): M3 T11 — opendray.exec.* namespace
26b4a04 feat(plugin-platform): M3 T7 — KEK/DEK crypto primitives
5f4a914 feat(plugin-platform): M3 T12 — opendray.http.* namespace
2480bac feat(plugin-platform): M3 T14+T16 — supervisor + JSON-RPC mux
bb6b64a feat(plugin-platform): M3 T9 — opendray.fs.* read-path namespace
b98d016 feat(plugin-platform): M3 T29 seam — migrations + HostV1 + framing codec
a93e5cf docs(plugin-platform): M3-PLAN — host sidecar runtime
```

Followed soon by `T25 fs-readme` + this `M3-RELEASE.md` commit.

---

## 8. Related documentation

- **Design contract** — `docs/plugin-platform/12-roadmap.md` §M3
- **Task plan** — `docs/plugin-platform/M3-PLAN.md` (918 lines, 29 tasks)
- **Bridge protocol spec** — `docs/plugin-platform/04-bridge-api.md`
- **Manifest schema** — `docs/plugin-platform/02-manifest.md` (§host runtime enum needs a T28 patch to add `python3 / bun / custom`)
- **Capabilities** — `docs/plugin-platform/05-capabilities.md`
- **Security** — `docs/plugin-platform/10-security.md` (needs T28 patch appending §6 threat matrix)
- **M2 release** — `docs/plugin-platform/M2-RELEASE.md`
- **Obsidian project landing** — `Obsidiannote/Projects/OpenDray/README.md` (lineage rcc → ntc → opendray, decision log, deploy infra)

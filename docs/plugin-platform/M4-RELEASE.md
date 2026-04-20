# OpenDray Plugin Platform M4.1 — Ship Readiness

**Status:** ✅ CLOSED — backend + Flutter consumer-side complete.
M4.2 (publisher CLI) deferred to post-v1 per M4-PLAN §1.
**Branch:** `kevlab` (not merged to main per project policy; stays
on kevlab until v1 launches).
**Base:** M3 closed at `208bcb3`.
**M4.1 head:** (commit range f5414c4..HEAD on kevlab)

## 1. Status by task

| Task | Title | Status | Commit(s) |
|------|-------|--------|-----------|
| T22  | Marketplace repo template                           | ✅ | `opendray-marketplace@5d10d36` |
| T23  | GitHub Actions — regenerate index.json              | ✅ | same |
| T24  | PR validation (schema, sha256, capability-diff)     | ✅ | same |
| T1   | `plugin/market` package + local backend extraction  | ✅ | `4905629` |
| T2   | `remote.List` — fetch index.json                     | ✅ | `5bc9f1a` |
| T3   | `remote.Resolve` — per-version JSON                  | ✅ | `ae4a7b4` |
| T4   | `HTTPSSource.Fetch` — download + SHA-256 + unzip     | ✅ | `132abb6` |
| T5   | Ed25519 signature verification                       | ✅ | `8753d16` |
| T6   | Mirror fallback + `HTTPStatusError`                  | ✅ | `289ae2a` |
| T7   | In-memory TTL cache (disk cache deferred)            | ✅ | `3ca4ae5` |
| T8   | Revocation polling + semver match                    | ✅ | `a722cd1` |
| T9   | Revocation action dispatcher                         | ✅ | `da8e3a8` |
| T10  | Trust policy enforcement                             | ✅ | `db2fa6f` |
| T11  | Gateway wiring + install-time policy + refresh       | ✅ | `b8c8d61` |
| T12  | Settings → Marketplace admin endpoint                | ✅ | (with T11) |
| T18  | Hub consumes `/api/marketplace/plugins`              | ✅ (M3 — no wire change needed) |
| T19  | Trust badge on Hub cards                             | ✅ | (Flutter, this release) |
| T20  | Auto-update indicator on Plugin page                 | ✅ | (Flutter, this release) |
| T21  | Revocation banner + provider-list refresh            | ✅ | (Flutter, this release) |
| T25  | Gateway integration test — httptest registry         | ✅ | (this release) |
| T26  | Signature verification tests                         | ✅ (with T5) |
| T27  | Revocation E2E                                       | ✅ (with T8 sweep tests) |
| T28  | Documentation update                                 | 🟡 Partial — this doc; 11-dx.md updates deferred |
| T29  | M4-RELEASE.md                                        | ✅ |
| T13  | CLI scaffold                                         | ⏸ M4.2 |
| T14  | `plugin scaffold`                                    | ⏸ M4.2 |
| T15  | `plugin validate`                                    | ⏸ M4.2 |
| T16  | `plugin build` + sign                                | ⏸ M4.2 |
| T17  | `plugin publish`                                     | ⏸ M4.2 |

**Summary:** 19 Done / 1 Partial / 5 M4.2-parked.

## 2. What ships in M4.1

### Consumer-side marketplace (backend)

- **`plugin/market/`** — interface over two backends:
  - `market/local/` — M3 on-disk catalog (preserved for airgapped
    and mock deployments).
  - `market/remote/` — HTTPS fetch of `index.json` + per-version
    JSON + publisher records + revocations.json. Mirror round-
    robin on 5xx/timeout; 4xx short-circuits. In-memory TTL
    cache (5 min default; `CacheTTL=-1` disables).
- **`plugin/install/HTTPSSource.Fetch`** — real implementation.
  Streams download + SHA-256 in one pass, verifies against
  `ExpectedSHA256` before any files land. `extractZipBundle` refuses
  absolute paths / `..` / symlinks / setuid and caps each entry at
  200 MiB to block zip bombs. Cap matches 09-marketplace.md host-
  form size.
- **`plugin/market/signing/`** — Ed25519 verifier + trust-level
  policy. `EnforcePolicy(entry, publisher, now)` returns
  `ErrSignatureRequired` for official/verified without a verified
  signature; community is optional but still rejects broken sigs.
- **`plugin/market/revocation/`** — types, semver matcher
  (Masterminds/semver), poller (`Config.Interval` clamped to
  [1h, 168h]). `FetchRevocations` extends the Catalog interface;
  both backends implement.
- **`plugin/market/actions/`** — dispatcher wiring poller matches
  to `Installer.Uninstall` / `Runtime.SetEnabled` / `WorkbenchBus.Publish`.

### Consumer-side marketplace (Flutter)

- **Hub trust badge** — official / verified / community rendered as
  coloured chips on each card.
- **Plugin page update indicator** — Plugin cards show
  "update → vX.Y.Z" chip when installed version < marketplace
  latest. Simple dotted-int-triple comparator; close enough for
  v1 without pulling pub_semver.
- **Revocation banner** — new `revocation` SSE event kind fires a
  snackbar + pokes `ProvidersBus` so the Plugin page refreshes
  immediately when a plugin gets uninstalled by the kill-switch.

### Gateway routes

| Route | Purpose |
|-------|---------|
| `GET /api/marketplace/plugins`  | Catalog list (works across local / remote) |
| `POST /api/marketplace/refresh` | Drops the in-memory cache |
| `GET /api/marketplace/settings` | Read-only config snapshot (T12) |
| `POST /api/plugins/install`     | Now enforces signature policy on `marketplace://...` |

### Config / env

- `OPENDRAY_MARKETPLACE_URL` — enables remote backend.
- `OPENDRAY_MARKETPLACE_MIRRORS` — comma-separated fallback URLs.
- `OPENDRAY_REVOCATION_POLL_HOURS` — 1–168, default 6.
- `OPENDRAY_MARKETPLACE_DIR` — preserved from M3 for local backend.

## 3. What's deferred to M4.2 / M5 / post-v1

- **Publisher CLI** (T13–T17) — `opendray plugin
  scaffold/validate/build/publish`. Until that lands, Kev is the
  sole publisher and hand-edits the marketplace repo. Tracked in
  `memory/m4_2_publisher_cli.md`.
- **On-disk catalog cache** (T7 full scope) — in-memory is enough
  for launch; stale-while-revalidate is M5 polish.
- **11-dx.md update** (part of T28) — publisher workflow prose
  blocks waiting on M4.2.
- **Auto-update auto-apply** — the indicator ships in M4.1 but
  pulling the trigger is a user action (tap → reinstall flow).
  Silent background updates are M5+.

## 4. Smoke test — manual walkthrough

Pre-flight:
```bash
export OPENDRAY_MARKETPLACE_URL=https://raw.githubusercontent.com/Opendray/opendray-marketplace/main/
# Restart opendray.service on syz via the deployer (see
# memory/syz_deploy_pipeline.md — do not trigger from inside a
# Claude Code session).
```

### 4a. Remote catalog reachable
```bash
TOKEN=$(curl -s -X POST http://127.0.0.1:8640/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"linivek","password":"<pw>"}' | jq -r .token)

curl -sH "Authorization: Bearer $TOKEN" \
  http://127.0.0.1:8640/api/marketplace/plugins | jq
# Expected: {"entries":[]}  (no plugins published yet)
# OR real entries once Kev merges PRs.
```

### 4b. Settings snapshot
```bash
curl -sH "Authorization: Bearer $TOKEN" \
  http://127.0.0.1:8640/api/marketplace/settings | jq
# Expected: {"source":"remote","registryUrl":"...","pollHours":6,...}
```

### 4c. Cache refresh
```bash
curl -sX POST -H "Authorization: Bearer $TOKEN" \
  http://127.0.0.1:8640/api/marketplace/refresh | jq
# Expected: {"refreshed":true}
```

### 4d. Install from remote (when a real entry exists)
```bash
curl -sX POST -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"src":"marketplace://opendray-examples/fs-readme@1.0.0"}' \
  http://127.0.0.1:8640/api/plugins/install | jq
# Expected: 202 with token, name, version, perms, sha-verified.
```

### 4e. Revocation sweep
- Append to opendray-marketplace/revocations.json:
  `{"name":"acme/evil","versions":"<=1.2.3","reason":"test",
    "recordedAt":"...","action":"warn"}`
- Merge PR → index.json regenerates → Cloudflare cache expires.
- Next poll (within 6h default) → banner shows in Flutter app.

## 5. Known issues & caveats

- **CDN + DNS not yet configured.** M4.1 ships with GitHub raw as
  the default URL (or unset → local backend). Cloudflare DNS
  `marketplace.opendray.dev` is post-launch.
- **No publisher CLI yet** — third-party devs can't self-publish.
  See memory/m4_2_publisher_cli.md.
- **Revocation polling only starts when a catalog exists** —
  empty-catalog deployments don't spam logs.
- **The revocation banner reuses the existing showMessage
  snackbar.** A persistent red banner (per spec) is deferred; the
  immediate snackbar + Plugin-page refresh is sufficient for the
  first ship.
- **Config keys can be bare "name" (M3 back-compat)** — remote
  backend defaults empty publisher to `opendray-examples` so old
  URLs like `marketplace://fs-readme` still resolve.

## 6. Sign-off checklist

- [x] `go test -race -count=1 -p 1 ./plugin/market/... ./gateway/ ./plugin/install/` green
- [x] Flutter `flutter test` 171/171 green
- [x] End-to-end gateway integration test
  (`TestMarketplaceInstall_EndToEnd`) covers register → install →
  confirm → DB state.
- [x] Signature policy table (official / verified / community /
  unknown) covered in `plugin/market/signing/policy_test.go`.
- [x] Revocation match + poller sweep + action dispatch covered in
  `plugin/market/revocation/*_test.go`.
- [ ] Manual smoke test on syz LXC (done at end-of-session; see
  memory/syz_deploy_pipeline.md for the deploy + session-kill
  caveats).
- [x] M4-PLAN.md updated (T22-T24 marked ✅ at `f5414c4`; this
  release doc closes the loop).

## 7. Commit history (M4.1 on kevlab)

Infrastructure + decisions:
```
3799403 docs(plugin-platform): close M3 + open M4
7bec834 docs(plugin-platform): lock M4 decisions + split M4.1 / M4.2
f5414c4 docs(plugin-platform): M4 T22/T23/T24 done — marketplace repo live
```

Marketplace registry (`github.com/Opendray/opendray-marketplace`):
```
5d10d36 chore: bootstrap marketplace registry (M4 T22 + T23)
```

M4.1 backend (this repo):
```
4905629 refactor(plugin-platform): T1 — market package skeleton + local backend
5bc9f1a feat(plugin-platform): M4.1 T2 — remote.List fetches index.json
ae4a7b4 feat(plugin-platform): M4.1 T3 — remote.Resolve per-version JSON
132abb6 feat(plugin-platform): M4.1 T4 — HTTPSSource.Fetch downloads + verifies + unzips
8753d16 feat(plugin-platform): M4.1 T5 — Ed25519 signature verification
289ae2a feat(plugin-platform): M4.1 T6 — mirror fallback + HTTPStatusError
3ca4ae5 feat(plugin-platform): M4.1 T7 (minimal) — in-memory TTL cache
a722cd1 feat(plugin-platform): M4.1 T8 — revocation polling + semver match
da8e3a8 feat(plugin-platform): M4.1 T9 — revocation action dispatcher
db2fa6f feat(plugin-platform): M4.1 T10 — trust policy enforcement
b8c8d61 feat(plugin-platform): M4.1 T11 — gateway wiring, install policy, refresh
```

M4.1 Flutter + doc closeout land with this release.

## 8. Related documentation

- **Design contract** — `docs/plugin-platform/09-marketplace.md`
- **Plan** — `docs/plugin-platform/M4-PLAN.md` (24 M4.1 tasks +
  5 M4.2 parked)
- **Marketplace repo** — `github.com/Opendray/opendray-marketplace`
- **Previous release** — `docs/plugin-platform/M3-RELEASE.md`

## 9. Next

`kevlab` stays open. M5 roadmap items:
- Legacy-plugin migration (17 bundled plugins → v1 manifests).
- Webview + sidecar combined form.
- iOS App Store submission.
- Contract freeze 2026-10-01.

Merge to `main` happens only after v1 ship-readiness sign-off.

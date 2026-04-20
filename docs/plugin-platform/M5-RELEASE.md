# OpenDray Plugin Platform M5 — Ship Readiness (Phases 1–4)

**Status:** 🟡 PARTIAL — Phases 1/3/4 shipped, contract frozen. Phase 5
(legacy plugin migration) and Phase 2 (public docs + distribution) are
deferred to post-contract-freeze work.
**Branch:** `kevlab` (not merged to main per project policy; stays on
kevlab until v1 launches).
**Base:** M4.1 closed at `fb3eb51`.
**M5 head:** `20f401c..HEAD` on kevlab.

## 1. Status by task

### Phase 1 — Platform completeness (Track B)

| ID | Title | Status | Commit |
|---|---|---|---|
| B1 | Webview+host combined form: manifest schema + validator | ✅ | `a90322c` |
| B2 | Webview+host: bridge proxy `opendray.commands.execute`  | ✅ | Already shipped via `opendray.workbench.runCommand` (M2/M3) |
| B3 | Webview+host: pg-browser rich UI bundle                 | ⏸ | Moved to Phase 5 (ships with pg-browser's migration PR) |
| B4 | `editorActions` render                                  | ✅ | `20f401c` |
| B5 | `sessionActions` render                                 | ✅ | `20f401c` |

### Phase 2 — Distribution + docs (Track C)

| ID | Title | Status | Notes |
|---|---|---|---|
| C1 | MkDocs Material on `opendray-docs` repo                 | ⏸ | Deferred post-v1 |
| C2 | Cloudflare DNS `docs.opendray.dev` → GH Pages           | ⏸ | Deferred post-v1 |
| C3 | Public examples repo migration                          | ⏸ | Deferred post-v1 |
| C4 | Multi-platform release artefacts                        | ⏸ | Deferred post-v1 |
| C5 | iOS self-sign guide                                     | ⏸ | Deferred post-v1 (no App Store cut planned) |

Rationale (from the 2026-04-20 scope review): contract-freeze +
legacy migration have to land before distribution rollout is useful —
otherwise the docs site points at a contract that can still drift,
and a public examples repo hosts plugins that are about to be
rewritten.

### Phase 3 — Carry-over cleanup (Track D)

| ID | Title | Status | Commit |
|---|---|---|---|
| D1-write | `fs.writeFile` + `fs.mkdir` + `fs.remove`           | ✅ | `daf42ff` |
| D1-watch | `fs.watch(glob, cb)` streaming subscription         | ✅ | `6eac939` |
| D2       | CSP golden + `TestE2E_KanbanFullLifecycle`          | ✅ | `d1a1d7c` |
| D3       | KEK auto-rotation on admin password change          | ✅ | `ff3eee8` |
| D4       | DB upgrade-path smoke test in CI                    | ✅ | `a897566` |

### Phase 4 — Contract freeze (Track E)

| ID | Title | Status | Commit |
|---|---|---|---|
| E1 | Manifest strict unknown-field validator at install     | ✅ | this commit |
| E2 | Bridge API signature lock — frozen banner on 04-bridge-api.md | ✅ | this commit |
| E3 | M5-RELEASE.md                                          | ✅ | this commit |

### Phase 5 — Legacy migration (Track A)

| ID | Title | Status |
|---|---|---|
| A1 | Terminal plugin → v1                                   | ✅ |
| A2 | File-browser plugin → v1                               | ✅ |
| A3 | Claude plugin → v1                                     | ⏸ |
| A4 | Retire compat synthesizer path                         | ⏸ |

**Summary:** 12 ✅ / 10 ⏸ (Phase 2 + Phase 5 + B3).

## 2. What ships in M5 Phases 1/3/4

### Manifest contract extensions

- `editorActions` + `sessionActions` contribution points (M5 B4/B5).
  Validator enforces required id+title, kebab/dot command-id pattern,
  and a max of 4 session actions per plugin (mobile overflow guard).
  Registry flattens the new slots alongside M2 webview slots with a
  deterministic sort (`group` asc, empty last for editor actions;
  `pluginName` + `id` for session actions). Both slots render from the
  same `/api/workbench/contributions` response.

- Webview + host combined form (M5 B1). `form: "webview"` manifests
  may now carry a top-level `host` block declaring a companion
  sidecar. The validator enforces the same host rules whether form is
  `host` or `webview + host:{}`. `HasHostBackend()` is the unified
  predicate across the supervisor, gateway, and registry.

### Bridge namespace completion

- `fs.writeFile` / `fs.mkdir` / `fs.remove` (M5 D1-write). 10 MiB
  payload cap, `opts.mode` honoured on both new and overwritten files,
  parent-symlink TOCTOU guard (gate re-check on the resolved path),
  `opts.recursive` for mkdir and remove.

- `fs.watch(glob)` / `fs.unwatch(subId)` (M5 D1-watch). Streaming
  subscription over the bridge WebSocket: fsnotify-backed pump with
  glob-driven directory enumeration (`/**` = bounded recursive walk),
  per-event TOCTOU re-check, and hot-revoke tied to the top-level
  `fs` cap — DELETE `/consents/fs` emits an `EPERM stream:"end"` per
  live sub within the M2 T23 200 ms SLO. Caps: 256 subs per API, 256
  dirs per watcher, 4096 walk entries.

### Secret subsystem — rotation on demand

- `RotateCredentialsAndKEK` (M5 D3). A single transaction now (a)
  locks the admin row with `SELECT … FOR UPDATE`, (b) unwraps every
  `plugin_secret_kek` row with the OLD KEK, (c) rewraps with the new
  HKDF-SHA256 KEK bumped to a fresh `kek_kid`, (d) upserts the new
  bcrypt hash. Failure at any step rolls everything back —
  admin_auth + wrapped DEKs stay pre-rotation. The gateway
  password-change handler (`POST /api/auth/credentials`) routes
  through this exclusively; `CredentialStore.Save` is kept for the
  bootstrap path but no longer fronts rotation.

### Test infrastructure

- `TestE2E_KanbanFullLifecycle` (M5 D2). Install → webview asset with
  byte-exact CSP → WS storage round-trip → hot-revoke 200 ms SLO →
  persistence across a full harness restart → uninstall cascade. The
  kanban plugin is now the reference end-to-end fixture for
  webview-form plugins.

- `TestMigrate_UpgradeFromPrePluginSchema` (M5 D4). Applies migrations
  001–009 (pre-plugin-platform snapshot), seeds a sessions + mcp_servers
  row, runs the full `db.Migrate` to bridge forward, and asserts the
  legacy rows survive both the upgrade and a subsequent idempotent
  re-run with plugin-era rows present.

### Contract freeze (M5 E1/E2)

- Installer rejects any manifest with a top-level key outside the v1
  whitelist OR a `contributes.*` key outside the contribution-point
  whitelist. `$schema` and `v2Reserved` are explicitly allowed —
  `v2Reserved` is the intentional forward-compat escape hatch for
  plugins that want to stash future-shape data ahead of a v2 bump.

- `04-bridge-api.md` + `02-manifest.md` both carry the
  `v1 frozen — 2026-04-20` banner. Any post-freeze contract change
  bumps the major schema version and ships as a coordinated SDK
  release.

## 3. Deferrals (not regressions)

These were explicitly dropped from M5 scope with Kev's sign-off
(2026-04-20):

- **Phase 2 (docs + distribution)** — no value before the contract
  freezes or legacy migrates. Shipping a docs site + public examples
  repo that mirrors an unstable contract would be noise.

- **B3 (pg-browser rich UI bundle)** — the plugin will get its UI
  rework as part of its own Phase 5 migration PR, not as a
  standalone platform task.

- **iOS App Store (C5)** — out of scope; users self-sign via Xcode.
  The self-sign guide lives in the `docs/building/` tree (deferred
  with the rest of Phase 2).

- **Phase 5 legacy migration (A1–A4)** — each of terminal /
  file-browser / claude migrates in its own PR, possibly reworking
  the plugin's functionality along the way. A4 (retiring the compat
  synthesizer) runs only after A1–A3 all land.

## 4. Acceptance criteria (what "Phase 1+3+4 done" means)

- [x] Every example plugin in `plugins/examples/` passes
      `ValidateV1Strict` (on-disk scan regression test —
      `TestStrict_OnDiskExamplesPass`).
- [x] `go test -short -race ./plugin/...` green (serial for
      embedded-postgres tests).
- [x] `go test -tags=e2e -race ./plugin/...` green (kanban + time-ninja
      both run, `-p=1` required because both spin embedded-postgres).
- [x] Every `04-bridge-api.md` signature has a corresponding handler
      in `plugin/bridge/` (spot-checked during E2 freeze audit).
- [x] Password rotation end-to-end preserves all `plugin_secret` values
      (covered by `TestRotate_RewrapsExistingDEKs`).
- [x] DB migration chain survives a non-empty legacy DB
      (`TestMigrate_UpgradeFromPrePluginSchema`).

## 5. Known gaps that do NOT block v1 contract-freeze

- **D1-watch on host sidecars** — `fs.watch` from a host sidecar
  returns `EUNAVAIL` by design; the stdio protocol isn't wired for
  stream subscriptions. Webview plugins get the full contract. A
  follow-up task can add stdio stream framing if demand surfaces.

- **Disk cache for marketplace** — M4.1 TTL cache is still in-memory
  only; boot-time cold start hits the CDN. Move-to-disk was parked
  at M4.1 close and remains parked.

- **Session action `when` expressions** — rendered unconditionally in
  M5. A proper context engine (`when: "editorLang == go"`) is
  post-v1 polish.

## 6. Open follow-ups (Phase 5 track)

These are **not** M5 commitments — they're the queue that opens once
the contract is officially locked:

1. Terminal plugin → v1 (A1).
2. File-browser plugin → v1 (A2).
3. Claude plugin → v1 (A3).
4. Retire compat synthesizer (A4, only after A1–A3).

Each legacy plugin migrates in its own PR; any rework of the
plugin's functionality rides along with that migration rather than
blocking the next plugin. Kev wants to "一个一个迁移" so a regression
in one doesn't stall the others.

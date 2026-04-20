# OpenDray Plugin Platform M5 — v1 GA

**Status:** 🟢 Scope locked — ready to execute.
**Depends on:** M4.1 closed at `fb3eb51` (`kevlab`)
**Acceptance deadline:** **2026-10-01** (contract freeze per
`12-roadmap.md:75`). ≈163 days from today. Non-negotiable.

## 0. Locked decisions (2026-04-20)

Consolidated here so §2 open-questions can be ignored —
everything below is the committed M5 shape.

| Decision | Value | Notes |
|---|---|---|
| iOS App Store | ❌ OUT OF SCOPE | Platform is too sensitive (exec/http/shell capabilities) for App Store review; iOS users self-sign + side-load. C4 ships the tooling for that path. |
| Legacy migration | Tier 1 only, ORDERED LAST | Migrate `terminal` + `file-browser` + `claude` one-at-a-time at the END of M5 so platform improvements from Tracks B/C/D/E are available to lean on; some plugin functionality gets reworked during its own migration PR. |
| Webview + host form | ✅ SHIP IN M5 | Unlocks pg-browser rich UI and every future plugin that needs both UI + privileged backend. |
| Publisher CLI (M4.2) | ❌ STAYS PARKED | Sole-publisher mode through v1.0; ecosystem opens in v1.1+. |
| Docs stack | MkDocs Material | Zero content rewrite, CF DNS → GitHub Pages. |
| editorActions + sessionActions | ✅ SHIP IN M5 | Small UI wiring. |
| languageServers LSP proxy | ❌ DEFERRED TO v1.5 | 2–3 week separate milestone. |

## 0.1 Phase ordering (REVISED)

Kev's direction: **platform + infra first, legacy migration last**
so the three tier-1 plugins migrate onto a fully-ready v1
platform rather than chasing a moving target.

1. **Phase 1 — Platform (Track B)** — webview+host, editor/session actions
2. **Phase 2 — Distribution + docs (Track C)** — multi-platform builds, docs site, examples repo
3. **Phase 3 — Carry-over cleanup (Track D)** — fs.write/watch, KEK rotation, CSP test, DB upgrade test
4. **Phase 4 — Contract freeze (Track E)** — M5-RELEASE, schema version locks
5. **Phase 5 — Legacy migration (Track A)** — ONE PLUGIN AT A TIME; each PR may rework functionality beyond the manifest-only migration

## 1. Roadmap-mandated deliverables

From `12-roadmap.md` §M5:

1. All MVP contribution points shipped — **11/14 full, 3 partial**.
2. All `opendray.*` MVP namespaces implemented — **14/16 full, 2 partial**.
3. Docs published at `docs.opendray.dev` — **not live yet**.
4. Example plugins repo live — **in-tree only; no public repo**.
5. iOS build tested through App Store review — **build infra
   present; signing / CI / submission all missing**.

Per `12-roadmap.md` §Deprecation plan, legacy plugins migrate
**in-tree** during M5, PR-by-PR. Compat mode stays alive through
v1.5; removed in v2. So "migrate all 16 by freeze" is **not**
required — only "start the migration" is.

## 2. Open questions (resolve before T1)

Every question below shapes task count + sequence. Answer these
first; I won't start coding until they're pinned.

### 2.1 Legacy migration scope

Which of the 16 legacy plugins MUST ship as v1 by freeze vs.
stay on compat mode through v1.5 → retire in v2?

Candidates ranked by launch importance:

| Plugin | Category | Migration effort | Launch-critical? |
|---|---|---|---|
| `claude` | CLI agent, required | M (sidecar wraps CLI) | **Probably yes** — flagship agent |
| `terminal` | Shell, required | M (PTY sidecar) | **Probably yes** — core session |
| `file-browser` | Panel, required | S (manifest-only, keep widget) | **Yes** — required=true |
| `codex` / `gemini` / `opencode` / `qwen-code` | CLI agents | M each | Nice-to-have |
| `git` | Panel | M (gateway/git wrapper → host) | Nice-to-have |
| `task-runner` / `log-viewer` / `obsidian-reader` / `telegram` / `mcp` / `web-preview` / `simulator-preview` / `llm-providers` | Panels | S–M each | Defer to v1.5? |

**Recommendation:** **tier 1** = required plugins (`claude` +
`terminal` + `file-browser`). Tier 2 = agents (`codex` etc).
Tier 3 = everything else, defer to v1.5.

**Your call:** (a) all 16, (b) tier 1 only, (c) tier 1 + 2, or
something else.

### 2.2 Webview + host combined form

pg-browser wants a real DB browser UI. That needs `form: webview
+ host.` sidecar + `webviewUI` block. Design locked in
`06-plugin-formats.md:83` but no code yet.

- **Ship in M5:** ~1 week of work (manifest schema + validator +
  bridge routing). Unlocks pg-browser rich UI + future plugins.
- **Defer to v1.5:** pg-browser stays command-only at launch.
  Lower risk for 2026-10-01 but worse initial demo.

**Recommendation:** ship in M5. It's a one-week addition and
every plugin with real UI needs it eventually. Blocking it
post-v1 means every "good" third-party plugin has to wait.

### 2.3 iOS App Store submission

**Precondition checklist:**
- [ ] Apple Developer account active (you mentioned
  `~/.appstoreconnect/private_keys/AuthKey_BPL8QFJ8M2.p8` for
  TestFlight — still valid?)
- [ ] App ID registered on dev portal
- [ ] Provisioning profile for `com.linivek.opendray` (or chosen
  bundle id)
- [ ] App Store Connect app listing (screenshots, description,
  privacy manifest, Export Compliance)

**Scope:**
- CI workflow `flutter build ipa` + fastlane upload to TestFlight
- Phased rollout submission + Apple review (~1 week reviewer
  latency per submission)

**Recommendation:** start this track NOW in parallel with the
legacy migration. App review takes calendar time we can't
compress. Target first TestFlight build by T+4 weeks.

**Your call:** confirm the precondition checklist, or tell me
which steps you need to do manually first.

### 2.4 Publisher CLI (M4.2)

M4-PLAN parked this. M5-ship options:
- **Keep parked** — sole-publisher mode through v1; ecosystem
  opens in v1.1.
- **Include in M5** — ~1-2 weeks of work; unlocks 3rd-party
  devs at v1 launch.

**Recommendation:** keep parked. Ecosystem bootstrap doesn't
need to coincide with v1 GA; many platforms launched without
third-party self-publish (VS Code's marketplace CLI landed months
after v1).

### 2.5 Docs site — stack

`docs.opendray.dev` options:
- **MkDocs Material** — lightweight Python, good search out of
  box
- **Docusaurus** — React, heavier but better for versioned docs
- **Plain markdown on GitHub Pages** — zero config

**Recommendation:** **MkDocs Material**. The `docs/plugin-platform/`
tree is already 15 .md files — MkDocs will render them with
navigation + search without touching content. Cloudflare DNS →
GitHub Pages.

**Your call:** confirm or pick differently.

### 2.6 Deferred contribution points — ship or punt?

Three contribution points declared MVP but not rendered today:

- **editorActions** — context-menu on editor pane. Small UI wire
  up. Probably ship in M5.
- **sessionActions** — context-menu on session pane. Same scope.
  Probably ship.
- **languageServers** — full LSP proxy WS route + Flutter client
  integration. **Significant** — could be 2-3 weeks alone.

**Recommendation:** ship editorActions + sessionActions in M5;
move languageServers to v1.5 (compat spec already allows clients
to degrade gracefully when the route 404s).

## 3. Proposed M5 scope (given my recommendations above)

### Track A — Legacy migration (tier 1 only)

| ID | Task | Effort |
|---|---|---|
| A1 | Migrate `terminal` manifest → v1 (publisher, engines) | S |
| A2 | Migrate `file-browser` manifest → v1 | S |
| A3 | Migrate `claude` → v1 host plugin (Node sidecar wrapping CLI, activation=onSession, permissions.exec) | M |
| A4 | Retire compat synthesizer path for migrated plugins | S |
| A5 | Doc update + examples README | S |

### Track B — Platform completeness

| ID | Task | Effort | Notes |
|---|---|---|---|
| B1 | Webview+host combined form: manifest schema + validator | S | ✅ `a90322c` |
| B2 | Webview+host: webview → own sidecar method proxy | — | Already ships via `opendray.workbench.runCommand` (M2/M3) — verified with B1 |
| B3 | Webview+host: pg-browser rich UI bundle | M | **Moved to Phase 5** — pg-browser's v1 migration PR will also rework its UI (confirmed 2026-04-20) |
| B4 | editorActions render | S | ✅ contract + registry + Flutter editor title-bar slot |
| B5 | sessionActions render | S | ✅ contract + registry + Flutter session toolbar slot |

### Track C — Distribution + docs (replaces original iOS track)

| ID | Task | Effort |
|---|---|---|
| C1 | MkDocs Material on `opendray-docs` repo | S |
| C2 | Cloudflare DNS `docs.opendray.dev` → GH Pages | S |
| C3 | Public examples repo `github.com/Opendray/opendray-examples` — migrate `plugins/examples/` out | S |
| C4 | Multi-platform release artefacts: Linux amd64 / macOS arm64 / Windows x64 tarballs + APK + iOS self-sign guide | M |
| C5 | `docs/building/self-hosting-ios.md` — fastlane + Xcode side-load instructions for iOS users | S |

### Track D — Carry-over cleanup

| ID | Task | Effort | Notes |
|---|---|---|---|
| D1.write | `fs.writeFile` + `fs.mkdir` + `fs.remove` | S | ✅ parent-symlink TOCTOU guard + 10 MiB write cap + mode bits |
| D1.watch | `fs.watch(glob, cb)` — streaming namespace method | M | Pending — needs stream-capable Conn + subscription lifecycle |
| D2 | CSP golden-file test + kanban E2E (T27 M3 deferral) | S | ✅ CSP golden was already in plugins_assets_test.go; added TestE2E_KanbanFullLifecycle with 200 ms hot-revoke SLO + persistence-across-restart + asset 404 post-uninstall |
| D3 | KEK auto-rotation on admin password change | M | ✅ RotateCredentialsAndKEK — atomic tx: rewrap every plugin_secret_kek row with new KEK + bump kid, then upsert admin_auth. Gateway password-change path now uses it. Tests cover fresh install, rewrap-preserves-plaintext, and failure rollback. |
| D4 | DB upgrade-path smoke test in CI | S | ✅ TestMigrate_UpgradeFromPrePluginSchema — 001–009 → seed legacy data → full Migrate → idempotent re-run w/ plugin rows |

### Track E — Contract freeze

| ID | Task | Effort |
|---|---|---|
| E1 | Manifest schema version lock + validator error on unknown fields post-freeze | S |
| E2 | Bridge API signature lock — finalise `04-bridge-api.md`, any post-freeze change needs major bump | S |
| E3 | M5-RELEASE.md | S |

**Totals:** 18 tasks (5 tracks). Track A (legacy) = 4 of those,
run LAST and each may also rework plugin functionality.

## 4. Dependency chain (high level)

```
Phase 1 — Platform:
    B1 (webview+host manifest) ◀── B2 (bridge proxy) ◀── B3 (pg-browser rich UI)
    B4 (editorActions) — independent
    B5 (sessionActions) — independent

Phase 2 — Distribution + docs:
    C1 (MkDocs) ◀── C2 (DNS) — pre-req for C3-C5
    C3 (examples repo) — independent
    C4 (multi-platform builds) ◀── C5 (iOS self-sign guide)

Phase 3 — Cleanup:
    D1-D4 all independent, can interleave

Phase 4 — Contract freeze:
    E1 + E2 before E3 (release notes lock the contract)

Phase 5 — Legacy migration:
    A1-A3 independent, one PR each, with optional rework
    A4 (retire compat) AFTER A1-A3
```

## 5. Rollout order (revised for Phase 1→5 sequencing)

1. **Phase 1 (platform):** B1 → B2 → B3 + B4 + B5 in parallel
2. **Phase 2 (distribution/docs):** C1 → C2 → C3 → C4 → C5
3. **Phase 3 (cleanup):** D1-D4 as time allows; D1 is biggest
4. **Phase 4 (contract):** E1 → E2 → E3 (release)
5. **Phase 5 (legacy):** A1 → A2 → A3 → A4; each PR stands alone,
   any rework to the plugin's functionality lands in its own
   migration PR rather than blocking the next plugin

## 6. Acceptance criteria for "M5 done"

- All three tier-1 legacy plugins migrate cleanly — fresh install
  from marketplace works identically to compat-synthesized flow.
- `form: webview + host` accepts. pg-browser rich UI renders on
  device.
- editorActions + sessionActions produce context menus.
- iOS app passes App Store review + available via TestFlight
  phased rollout.
- `docs.opendray.dev` resolves + renders the full `docs/plugin-
  platform/` tree with navigation + search.
- Example plugin tree either in a dedicated repo OR well-
  organised in `plugins/examples/` with README.
- No P0/P1 bugs against bridge API or manifest schema.
- Contract freeze effective 2026-10-01 — manifest schema + bridge
  API signature changes now require major version.

## 7. Non-goals (punted to v1.5 / M6)

- Tier 2 + tier 3 legacy plugin migration.
- Publisher CLI (M4.2 scope).
- languageServers LSP proxy route.
- Disk-based marketplace cache (stale-while-revalidate).
- KEK auto-rotation (stays manual through v1.5).
- Telegram bot `onCommand` routing.
- Declarative UI tree E2E tests.

## 8. Contract-freeze risk

If tracks slip:
- **iOS review** is the longest-latency item. Lose 1 week →
  slip everything.
- **Legacy migration** is the highest-volume — even tier 1 has
  `claude` which is complex (sidecar + session activation).
- **docs.opendray.dev** is gating on Cloudflare DNS propagation.

**Mitigation if we overrun:**
1. Drop pg-browser rich UI from M5 (keep command-only) → B3 out.
2. Drop editorActions + sessionActions → B4+B5 to v1.5.
3. Sole-publisher mode through v1.0 (already in the plan).

## 9. Decisions (locked 2026-04-20)

See §0. Summary:
- Legacy migration: tier 1 only, run LAST, one PR per plugin
- Webview+host form: ship in M5
- iOS App Store: OUT — side-load + self-sign path
- Publisher CLI: parked
- Docs: MkDocs Material
- editorActions + sessionActions: ship; languageServers deferred
- Phase ordering: platform → distribution → cleanup → freeze → legacy

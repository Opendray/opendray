# OpenDray Plugin Platform M4 — Marketplace client + publisher

**Status:** Planning
**Depends on:** M3 closed at commit `208bcb3` (`kevlab`)
**Unlocks per roadmap:** plugin ecosystem — third parties can publish, users can discover/install/auto-update.
**Contract freeze date (immovable):** 2026-10-01 — M4 + M5 must fit inside the remaining window.

## 1. Scope boundary

### In scope

- A real remote marketplace: `plugin/market/` package fetches
  `index.json` + per-version JSON from a public registry URL,
  verifies SHA-256, verifies Ed25519 signatures, caches locally.
- `HTTPSSource.Fetch` wired — `marketplace://` refs resolve against
  the remote registry (M3 shipped the local-catalog stub; M4
  replaces it with the real one).
- Revocation list polling (6 h default + on launch). Actions:
  `uninstall` / `disable` / `warn`.
- Publisher CLI `opendray plugin` with subcommands `scaffold`,
  `validate`, `build`, `publish`. `publish` forks the marketplace
  repo, commits, and opens a PR.
- Flutter: Hub migrates from reading the local-catalog JSON to
  calling a new `/api/marketplace/registry/*` endpoint backed by
  the remote fetch; trust badges + auto-update indicator surface.
- Marketplace repo template (`github.com/opendray/marketplace`)
  with a CI workflow that regenerates `index.json` on every merge
  to `main`.

### Out of scope (push to M5)

- Legacy-plugin migration to v1 — still happens in M5 per
  12-roadmap §Deprecation plan. Marketplace exists in M4 so M5's
  migration has somewhere to publish to.
- Webview + sidecar combined form. New work item flagged in
  M3-RELEASE §3; scope in M5.
- iOS App Store submission — M5.
- Paid plugins / billing — M7 per roadmap.
- Private marketplaces — M7.
- In-marketplace ratings / reviews — post-v1.

## 2. Open questions (resolve before T1)

1. **Registry URL default.** Roadmap suggests
   `https://raw.githubusercontent.com/opendray/marketplace/main/`
   OR a Cloudflare-fronted copy. Which ships as the default?
   Mirror list is client-configurable — pick which public URL is
   authoritative.
2. **Signing-key bootstrap.** `opendray plugin build` needs a
   publisher private key. Options:
   (a) file in `~/.config/opendray/keys/<publisher>.ed25519`;
   (b) macOS Keychain / Linux Secret Service; (c) passphrase-
   encrypted file. (a) is simplest and portable; (b) is nicer.
3. **CI infrastructure for the marketplace repo.** GitHub Actions
   (public, native to the source of truth) or self-hosted Gitea
   (matches existing `tea.linivek.online`)? Probably GitHub so
   contributors can submit PRs without a Gitea account.
4. **Revocation-list poll cadence override.** 6 h matches the
   spec; should we expose `OPENDRAY_REVOCATION_POLL_HOURS`? Nice
   for airgapped deployments but could confuse users — lean "no,
   use the spec value" unless someone asks.
5. **Hub page vs Settings → Marketplace.** M3 shipped Hub as the
   install entry. Spec lists a separate "Settings → Marketplace"
   for registry URL + auto-update toggle. Resolution: keep Hub
   as the browse/install surface, add a Settings → Marketplace
   subpage for admin knobs only.

## 3. Task graph (29 tasks)

### Backend — marketplace client (T1–T12)

| ID | Title | Depends on | Effort |
|----|-------|------------|--------|
| T1 | Refactor `plugin/marketplace` → `plugin/market/local` + introduce `plugin/market/remote` skeleton | — | S |
| T2 | `market/remote.FetchIndex` — HTTP GET index.json + schema validate | T1 | S |
| T3 | `market/remote.FetchVersion` — per-version JSON + SHA256 check | T2 | S |
| T4 | `install.HTTPSSource.Fetch` — download ZIP, verify SHA256, extract to staging | T3 + T1 | M |
| T5 | Ed25519 signature verification + publisher key resolution | T3 | M |
| T6 | Mirror fallback — round-robin retry on 5xx/timeout | T2, T3, T4 | S |
| T7 | Registry cache — filesystem persistence, stale-while-revalidate | T2 | S |
| T8 | `market/revocations.go` — poll loop (6 h + on launch), `plugin_revocation_seen` table | T2 | M |
| T9 | Revocation action dispatcher — uninstall / disable / warn wired through Installer + provider runtime | T8 | M |
| T10 | Trust level propagation — Entry.Trust flows from publisher record through Hub | T5 | S |
| T11 | Gateway `GET /api/marketplace/registry` (index + per-version) + `POST /refresh` | T7, T10 | S |
| T12 | Settings → Marketplace admin subpage backing — registry URL + mirrors in config + auto-update toggle in user prefs | T11 | S |

### Publisher CLI (T13–T17)

`cmd/opendray-plugin/` is a new binary separate from `cmd/opendray`
so plugin authors don't need the full gateway.

| ID | Title | Depends on | Effort |
|----|-------|------------|--------|
| T13 | `cmd/opendray-plugin/` skeleton + cobra-style subcommand dispatch | — | S |
| T14 | `scaffold` — interactive manifest wizard (form / permissions / configSchema) | T13 | M |
| T15 | `validate` — runs `ValidateV1` against local manifest + lint bundle rules (zip size, setuid bit, symlink escape) | T13 | S |
| T16 | `build` — zip + SHA256 + optional Ed25519 sign | T15, T5 | M |
| T17 | `publish` — fork marketplace repo, create branch, write per-version JSON, open PR with templated body | T16, T11 | M |

### Flutter (T18–T21)

| ID | Title | Depends on | Effort |
|----|-------|------------|--------|
| T18 | Hub fetches from `/api/marketplace/registry` (not local catalog) + cache-aware refresh | T11 | M |
| T19 | Trust badge on Hub cards (official / verified / community / sideloaded) + legend | T18, T10 | S |
| T20 | Auto-update — list in Plugin page shows "update available" chip; Settings toggle for capability-broadening prompts | T18, T11 | M |
| T21 | Revocation banner + system dialog on `uninstall` / `disable` actions | T9 | S |

### Marketplace repo infra (T22–T24)

These land on `github.com/opendray/marketplace`, not the main repo.

| ID | Title | Depends on | Effort |
|----|-------|------------|--------|
| T22 | Template repo layout (plugins/ / publishers/ / CODEOWNERS / revocations.json) | — | S |
| T23 | GitHub Actions: regenerate index.json on push to main + upload to CDN mirror | T22 | S |
| T24 | CI validation: manifest schema + SHA matches artifact URL + sandbox scan (forbidden file types) + capability-diff comment on PR | T23 | M |

### Tests (T25–T27)

| ID | Title | Depends on | Effort |
|----|-------|------------|--------|
| T25 | Integration test: gateway against a `file://` test-registry fixture | T4, T5 | M |
| T26 | Signature verification unit + integration tests | T5 | S |
| T27 | Revocation E2E: fake registry marks plugin revoked → client polls → action fires | T8, T9 | M |

### Docs (T28–T29)

| ID | Title | Depends on | Effort |
|----|-------|------------|--------|
| T28 | Update `11-developer-experience.md` with publisher workflow + new CLI | T17 | S |
| T29 | M4-RELEASE.md — status table + smoke test + commit history | all | S |

**Totals:** 29 tasks · 10 M / 18 S / 1 parked. Expected duration: 3–5 weeks of focused work at the rate M3 shipped.

## 4. Dependency chain

```
T1 (refactor)
 └▶ T2 (FetchIndex) ──┬▶ T3 (FetchVersion) ──┬▶ T4 (HTTPSSource) ─┐
                     │                       │                     │
                     │                       ├▶ T5 (signatures) ──┤
                     │                       │                     │
                     │                       └▶ T6 (mirrors) ─────┤
                     │                                              │
                     ├▶ T7 (cache) ────────┬▶ T11 (registry API) ─┼▶ T18 (Hub) ─┬▶ T19 (badge)
                     │                     │                      │             ├▶ T20 (update)
                     │                     └▶ T12 (settings) ─────┘             │
                     │                                                           │
                     └▶ T8 (revocations) ──▶ T9 (actions) ────────────────────── └▶ T21 (banner)

T13 (CLI skel) ──▶ T14 (scaffold)
               └─▶ T15 (validate) ──▶ T16 (build) ──▶ T17 (publish)

T22 (repo tmpl) ──▶ T23 (CI regen) ──▶ T24 (PR checks)

Tests/docs depend on everything above; rollout order below.
```

## 5. Rollout order

1. **T1 → T2 → T7 → T3 → T11** — thinnest useful slice: gateway
   can talk to a remote registry and return the index. Flutter
   still shows the M3 local-catalog fallback; no user-visible
   change yet.
2. **T4 → T6** — HTTPS install path wired. At this point
   `marketplace://fs-readme` works from the real registry (with
   an Ed25519-less dev bypass).
3. **T5 → T10** — signatures + trust propagation.
4. **T8 → T9 → T21** — revocation infra shipped even before UI
   migrates, so it protects existing users first.
5. **T18 → T19 → T20** — Flutter migration. At this point the
   Hub is live against the real registry.
6. **T12** — Settings → Marketplace admin knobs.
7. **T13 → T17** — publisher CLI independent of the above.
8. **T22 → T24** — marketplace repo bootstrapped. Parallel track
   that doesn't gate code shipping.
9. **T25 → T27** — tests fill in as each slice lands; T27 is the
   highest-value acceptance test for M4.
10. **T28 → T29** — docs + release notes.

## 6. Acceptance criteria for "M4 done"

- End-to-end: `opendray plugin publish` from a dev machine → PR
  opens on `github.com/opendray/marketplace` → CI passes → human
  merges → Hub on a second device shows the plugin within 10
  minutes and installs it via one tap with signature verified.
- Revocation: mark an installed plugin as `uninstall` in
  `revocations.json` → within the poll window the client
  auto-uninstalls with a banner.
- Trust badges render on every Hub card. `sideloaded` label
  appears when installing from `local:` or bare absolute path.
- Gateway `/api/marketplace/registry` responses carry the same
  Entry shape the M3 local catalog used (Flutter stays
  back-compatible).
- `opendray plugin validate` exits 0 on every example in
  `plugins/examples/*`.
- No P0/P1 bugs against the marketplace wire format.

## 7. Security notes (carry-over from §6 of 10-security.md)

- **Signature verification is required** for `trust: verified` or
  `official`. A missing/invalid signature on those trust levels
  fails install with a clear message. `community` and
  `sideloaded` are not required to sign.
- **Publisher key rotation** — publisher record lists multiple
  Ed25519 keys with `addedAt` / `expiresAt`. A signature
  verifies if it matches any non-expired key.
- **TLS pinning is NOT attempted** for the registry URL —
  configurable mirrors make pinning brittle and revocation-list
  checks + PR audit trail are the real defences.
- **SSRF stays blocked** — HTTPSSource.Fetch uses the same
  `net.Dialer.Control` RFC1918-deny path as `opendray.http.*`.
  Registry + artifact URLs MUST resolve to public IPs (minor
  annoyance for tests — fixture uses `file://`, not HTTP).
- **Revocation is advisory.** Airgapped installs won't auto-act
  but will show the warning on next network contact. No phone-
  home beacon.

## 8. Non-goals

- **No vendor-run approval workflow.** The marketplace is a Git
  repo reviewed via PR. Merge = published.
- **No in-app publishing UI.** Publishing is a developer
  workflow; the app is for installing.
- **No paid billing.** Out of scope for v1; M7.
- **No private registries first-class.** You can point
  `OPENDRAY_MARKETPLACE_REGISTRY_URL` at any URL already; first-
  class private-marketplace features are M7.

## 9. Relevant file paths

- `plugin/market/` — new package (partially repurposes M3's
  `plugin/marketplace` as the `local` backend).
- `plugin/install/source.go` — `HTTPSSource.Fetch` ENotImpl →
  implemented.
- `cmd/opendray-plugin/` — new binary.
- `gateway/plugins_market.go` — new (replaces M3
  `gateway/marketplace.go` eventually; kept parallel during
  migration).
- `app/lib/features/hub/hub_page.dart` — rewire to
  `/api/marketplace/registry`.
- `docs/plugin-platform/09-marketplace.md` — already the
  authoritative spec (no rewrite needed; small updates for
  concrete file paths once code lands).

## 10. Open risk — contract freeze

The v1 contract freeze is **2026-10-01**. M4 + M5 + iOS review
must fit inside that window. M4 at the pace M3 shipped = ~4
weeks; M5 migration of 17 legacy plugins (even at 2–3 days each)
plus iOS review = 8–10 weeks. That leaves zero buffer.

**Mitigations if we slip:**

- Ship M4 without T13–T17 (publisher CLI) — all plugins stay
  first-party until post-freeze. Users can still install from
  the marketplace repo; only third-party plugin publishing is
  deferred.
- Combine T8 + T9 (revocation) with manual-only revocation
  (Kev toggles it client-side on the LXC) instead of remote
  polling, if signing infra takes too long.

Review the freeze date monthly. Per roadmap §Locked, we can
only slip if "M1–M4 slip by more than one calendar month
combined". M3 shipped on time so we still have the full M4
budget.

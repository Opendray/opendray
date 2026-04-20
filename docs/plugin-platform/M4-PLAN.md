# OpenDray Plugin Platform M4 — Marketplace (consumer side)

**Status:** Ready to start
**Depends on:** M3 closed at commit `208bcb3` (`kevlab`)
**Scope:** M4.1 only — consumer-side marketplace. Third-party
publisher CLI (M4.2) deferred until after v1 launch; Kev is the
sole publisher in the meantime and edits the marketplace repo by
hand.
**Contract freeze date (immovable):** 2026-10-01 — M4.1 + M5 must
fit inside the remaining window.

## 1. Scope boundary

### In scope (M4.1 — this milestone)

- Real remote marketplace: `plugin/market/` package fetches
  `index.json` + per-version JSON from a public registry URL,
  verifies SHA-256, verifies Ed25519 signatures, caches locally.
- `HTTPSSource.Fetch` wired — `marketplace://` refs resolve against
  the remote registry (M3 shipped the local-catalog stub; M4
  replaces it with the real one).
- Revocation list polling (6 h default + on launch). Actions:
  `uninstall` / `disable` / `warn`.
- Flutter: Hub migrates from reading the local catalog to calling
  a new `/api/marketplace/registry/*` endpoint backed by the
  remote fetch; trust badges + auto-update indicator surface.
- Marketplace repo template (`github.com/Opendray/opendray-marketplace`)
  with a CI workflow that regenerates `index.json` on every merge
  to `main`. Needed even in sole-publisher mode so Kev's hand-
  edited PRs get validated + published consistently.

### Deferred to M4.2 (post-launch)

- **Publisher CLI `opendray plugin {scaffold,validate,build,
  publish}`.** Third-party developers can't self-publish yet;
  until M4.2 lands they either submit a PR against the
  marketplace repo manually or go through Kev. Rationale: the
  app ship date is tight, publisher workflow is a non-blocking
  ergonomics layer on top of the same wire format. Tracked in
  project memory `m4_2_publisher_cli.md`.

### Out of scope (push to M5 or later)

- Legacy-plugin migration to v1 — M5 per 12-roadmap §Deprecation.
- Webview + sidecar combined form — M5, flagged in M3-RELEASE §3.
- iOS App Store submission — M5.
- Paid plugins / billing — M7.
- Private marketplaces first-class — M7.
- In-marketplace ratings / reviews — post-v1.

## 2. Resolved design decisions

All five open questions resolved 2026-04-20. Recorded here so T1
starts from a fixed set of constraints; future ambiguity resolves
by reading this section.

### 2.1 Registry URL — CDN-fronted custom domain

**Primary:** `https://marketplace.opendray.dev/` (Cloudflare CDN).
**Fallback mirror 1:** `https://raw.githubusercontent.com/opendray/marketplace/main/`
**Reserved mirror slot:** community-run mirror (configurable, no
default).

Why not raw GitHub: unauth rate limit is 60 req/hr per IP. At
ecosystem scale every app launch pulls `index.json` + any plugin
install pulls a per-version JSON — one office building on
shared-NAT hits the limit in minutes. CDN absorbs that plus gives
global latency. Custom domain also reads as legitimate in the
consent sheet ("marketplace.opendray.dev said this plugin…").

Infra: Cloudflare DNS CNAME → GitHub Pages or R2. Kev already
runs CF tunnels; added surface area ≈ zero.

### 2.2 Signing keys — three-tier lookup

Publisher CLI + server's signing verifier resolve the private key
in this priority:

1. `OPENDRAY_SIGNING_KEY` env var — for CI / GitHub Actions.
2. OS keychain (macOS Keychain / Linux Secret Service / Windows
   Credential Manager) — default for devs. Service name
   `opendray`, account `<publisher>`. Use `github.com/zalando/go-keyring`
   for cross-platform.
3. `~/.config/opendray/keys/<publisher>.age` — passphrase-
   encrypted file for edge cases (shared workstation, headless
   Linux where Secret Service isn't available).

Mirrors the `age` / `sops` / `gpg` / `vsce` pattern every plugin
author already recognises.

### 2.3 Marketplace CI — GitHub Actions

Public repo with third-party PR contributors. Self-hosted Gitea
would require contributors to register on `tea.linivek.online` —
enough friction to kill the ecosystem.

Budget: public repos get unlimited GitHub Actions minutes on the
standard runners.

### 2.4 Revocation poll cadence — env-overridable

Default 6 h (per spec). Expose `OPENDRAY_REVOCATION_POLL_HOURS`
with floor 1 / ceiling 168.

- `1 h` — high-security (finance, healthcare).
- `24 h` — mobile-battery sensitive.
- `168 h` (1 week) — enterprise fleets with local mirror.

Ceiling 168 prevents "never" — revocation is security
infrastructure, must poll eventually.

### 2.5 Hub vs Settings → Marketplace — both

- **Hub** (exists from M3): consumer surface. Browse cards,
  install button, consent dialog, config dialog.
- **Settings → Marketplace** (new subpage in M4): admin surface.
  Registry URL + mirrors, auto-update toggle, trust-level filter
  ("only show verified+"), "Refresh cache now", revocation log.

Two roles, two pages. Same split as App Store "browse" vs iOS
Settings → "App Store" preferences.

## 3. Additional ecosystem-level decisions

### 3.1 Trust levels — semantics

| Level | Means | Granted by |
|-------|-------|-----------|
| `official` | OpenDray core team maintained or audited | Manual edit to `publishers/opendray.json` |
| `verified` | Publisher passed DNS TXT + identity check, Ed25519 key registered | CI verifies DNS TXT; human sets `trust: verified` on merge |
| `community` | Any merged PR author | Default |
| `sideloaded` | Client-side label when install source isn't marketplace | Client attaches automatically on `local:` / bare path |

### 3.2 Publisher verification flow (for reference; implemented in M4.2)

1. Third-party dev forks `github.com/Opendray/opendray-marketplace`.
2. Adds `publishers/<name>.json` with `keys: [ed25519 pubkey]` +
   `domainVerification.record: "opendray-verify=<token>"`.
3. Adds DNS TXT `opendray-verify=<token>` to the claimed domain.
4. Opens PR.
5. CI (T24) verifies the TXT record resolves + matches the token.
6. Human maintainer reviews → merges → trust starts at
   `community`. Upgrade to `verified` is a separate manual edit
   after identity check.

Documented in the marketplace repo README. Publisher CLI in M4.2
automates steps 1–2 and 4.

### 3.3 Version pinning — no "latest"

Install refs MUST carry a specific version:
`marketplace://<publisher>/<name>@<version>`. Bare `<name>`
refs resolve client-side to the latest at-browse-time, but the
install call carries the pinned version. Prevents silent upgrade
on PR merge. Auto-update is a separate per-plugin opt-in.

### 3.4 Namespace — `publisher/name` from day one

Install refs use `publisher/name` format. Reserving a shape
compatible with the third-party ecosystem avoids a breaking
change once M4.2 lands. M3's bare-`name` refs (e.g.
`marketplace://fs-readme`) get backwards-compat resolved to
`opendray-examples/fs-readme` during T1's refactor.

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

### Publisher CLI (T13–T17) — **deferred to M4.2**

`cmd/opendray-plugin/` is a new binary separate from `cmd/opendray`
so plugin authors don't need the full gateway. **Not in M4.1
scope** — until M4.2 lands, the sole publisher (Kev) hand-edits
the marketplace repo + relies on CI (T22–T24) to validate.

Tracked in memory `m4_2_publisher_cli.md` so it's not forgotten.

| ID | Title | Depends on | Effort |
|----|-------|------------|--------|
| T13 | `cmd/opendray-plugin/` skeleton + cobra-style subcommand dispatch | — | S |
| T14 | `scaffold` — interactive manifest wizard (form / permissions / configSchema) | T13 | M |
| T15 | `validate` — runs `ValidateV1` against local manifest + lint bundle rules (zip size, setuid bit, symlink escape) | T13 | S |
| T16 | `build` — zip + SHA256 + optional Ed25519 sign | T15, T5 | M |
| T17 | `publish` — fork marketplace repo, create branch, write per-version JSON, open PR with templated body (+ DNS TXT verification helper for publisher onboarding) | T16, T11 | M |

### Flutter (T18–T21)

| ID | Title | Depends on | Effort |
|----|-------|------------|--------|
| T18 | Hub fetches from `/api/marketplace/registry` (not local catalog) + cache-aware refresh | T11 | M |
| T19 | Trust badge on Hub cards (official / verified / community / sideloaded) + legend | T18, T10 | S |
| T20 | Auto-update — list in Plugin page shows "update available" chip; Settings toggle for capability-broadening prompts | T18, T11 | M |
| T21 | Revocation banner + system dialog on `uninstall` / `disable` actions | T9 | S |

### Marketplace repo infra (T22–T24)

These land on `github.com/Opendray/opendray-marketplace`, not the main repo.

| ID | Title | Depends on | Effort |
|----|-------|------------|--------|
| T22 | Template repo layout (plugins/ / publishers/ / CODEOWNERS / revocations.json) | — | ✅ `opendray-marketplace@5d10d36` |
| T23 | GitHub Actions: regenerate index.json on push to main + upload to CDN mirror | T22 | ✅ `opendray-marketplace@5d10d36` |
| T24 | CI validation: manifest schema + SHA matches artifact URL + sandbox scan (forbidden file types) + capability-diff comment on PR | T23 | ✅ `opendray-marketplace@5d10d36` (shipped together with T22/T23) |

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

**Totals (M4.1):** 24 tasks · 8 M / 16 S. Expected duration: ~3 weeks at M3's rate.
**Totals (M4.2):** 5 tasks · 3 M / 2 S. Runs after v1 launch.

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

T13–T17 (publisher CLI) — **deferred to M4.2**, unblocks after v1 launch.

T22 (repo tmpl) ──▶ T23 (CI regen) ──▶ T24 (PR checks)

Tests/docs depend on everything above; rollout order below.
```

## 5. Rollout order (M4.1)

1. **T22 → T23** — create the marketplace repo + CI that
   regenerates `index.json`. Even sole-publisher mode needs this
   running before T2 has anything to fetch from.
2. **T1 → T2 → T7 → T3 → T11** — thinnest useful slice: gateway
   can talk to a remote registry and return the index. Flutter
   still shows the M3 local-catalog fallback; no user-visible
   change yet.
3. **T4 → T6** — HTTPS install path wired. At this point
   `marketplace://opendray-examples/fs-readme@1.0.0` works from
   the real registry (Ed25519 dev-bypass until T5).
4. **T5 → T10** — signatures + trust propagation.
5. **T8 → T9 → T21** — revocation infra shipped even before UI
   migrates so it protects existing users first.
6. **T18 → T19 → T20** — Flutter migration. At this point the
   Hub is live against the real registry.
7. **T12** — Settings → Marketplace admin subpage.
8. **T24** — marketplace repo PR-validation job (SHA check,
   schema validation, capability diff).
9. **T25 → T27** — tests fill in as each slice lands; T27
   (revocation E2E) is the highest-value acceptance test.
10. **T28 → T29** — docs + release notes.

**M4.2 (after v1 launch):** T13 → T17 in order.

## 6. Acceptance criteria for "M4.1 done"

- End-to-end: Kev hand-edits a PR on `github.com/Opendray/opendray-marketplace`
  adding `plugins/opendray-examples/fs-readme/1.0.0.json` with a
  real signed artifact URL → CI (T23/T24) validates + regenerates
  `index.json` on merge → Hub on a second device shows the
  plugin within 10 minutes and installs it via one tap with
  SHA-256 + Ed25519 signature verified.
- Revocation: mark an installed plugin as `uninstall` in
  `revocations.json` → within the poll window the client
  auto-uninstalls with a banner.
- Trust badges render on every Hub card. `sideloaded` label
  appears when installing from `local:` or bare absolute path.
- Settings → Marketplace shows Registry URL, mirror list,
  auto-update toggle, Refresh button, and a revocation log.
- Gateway `/api/marketplace/registry` responses carry the same
  Entry shape the M3 local catalog used (Flutter stays
  back-compatible on the wire).
- No P0/P1 bugs against the marketplace wire format.

## 6.2 Acceptance criteria for "M4.2 done" (future)

- `opendray plugin scaffold` generates a working manifest.
- `opendray plugin validate` exits 0 on every example in
  `plugins/examples/*`.
- `opendray plugin publish` from a third-party dev machine →
  forks repo + opens PR → CI passes → maintainer merges → Hub
  shows the plugin.
- DNS TXT verification flow documented + tested.

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

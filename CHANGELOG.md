# Changelog

All notable changes to OpenDray v2 are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Version numbers follow this project's own **major-as-generation**
strategy — major version = product generation, minor = feature
iteration, patch = fix / polish. See [VERSIONING.md](./VERSIONING.md)
for the full rationale and what triggers a major bump.

## [Unreleased]

## [v2.4.0] — 2026-05-31

Multi-Claude-account UX, two-way Telegram channel, and a clutch of
session-quality fixes. The big new capability: a single OpenDray
gateway can now manage multiple Anthropic identities side-by-side and
let an operator switch a live Claude session between them without
losing the conversation.

### Added

- **Claude accounts: filesystem watcher.** `~/.claude-accounts/<name>/`
  is now monitored with fsnotify; a new `.credentials.json` (the
  result of `CLAUDE_CONFIG_DIR=… claude login`) registers an account
  row automatically. 500ms debounce, backoff-on-error reattach loop,
  symlink rejection at every level.
- **Claude accounts: synthetic `default` row.** `~/.claude/.credentials.json`
  (the CLI's own home) now surfaces as a row named `default` so the
  primary identity is visible in the panel without forcing the
  named-account login flow.
- **Claude accounts: capacity chips.** Each row now shows
  `subscription_type`, `rate_limit_tier`, `active_sessions`,
  `last_used_at`, and `oauth_email` — all derived server-side from
  `<configDir>/.credentials.json` + `<configDir>/.claude.json` + a
  single JOIN against the sessions table. No new chrome.
- **Claude accounts: least-loaded auto-assign at session create.**
  When `POST /sessions` arrives with provider=claude and empty
  `claude_account_id` (and ≥2 accounts are enabled), the gateway
  picks the enabled account with the fewest non-terminal sessions
  (alphabetical tiebreaker). Removes the "everything piles onto
  default" bias. Explicit operator pin still wins.
- **Claude accounts: identity drift detection.** First-seen
  `oauthAccount.emailAddress` per account is recorded under
  `~/.opendray/cliacct-identity.json` (chmod 0600). On every List/Get,
  the current on-disk email is compared; mismatch surfaces
  `identity_drift=true` and `previous_email` on the Account row,
  rendered as a red "identity changed: was X · accept" chip.
  `POST /api/v1/claude-accounts/{id}/accept-identity` updates the
  baseline so the chip clears.
- **Session switch preserves conversation.**
  `PATCH /api/v1/sessions/{id}/claude-account` now hard-links the
  Claude transcript JSONL from `<old_config_dir>/projects/<workspace>/
  <session_id>.jsonl` into `<new_config_dir>/projects/<workspace>/`
  before respawning. Claude `--resume` then finds and replays the
  conversation under the new account. Hard-link shares one inode so
  switching back-and-forth keeps both views synchronized.
- **Telegram: two-way conversational chat.** Typing indicator, turn
  replies, persistent control keyboard acting on the current session,
  configurable from the dashboard.
- **Catalog: warn + confirm before CLI upgrade.** The in-app CLI
  upgrade button now warns when sessions are using the CLI it's about
  to replace, with a new `scripts/enable-cli-updates.sh` helper for
  the non-root install path.
- **Web: MRU session ordering + Cmd/Ctrl+K palette search**.

### Changed

- `claude_account_id` validation is now enforced at session create
  AND at switch — bogus or disabled ids return HTTP 400 BEFORE the
  row is persisted (create) or BEFORE the live PTY is stopped (switch).
- Default idle threshold raised 30s → 5m so long-running tool
  invocations don't get killed by the idle reaper.
- The "Switch Account" confirmation dialog now says "conversation
  history is preserved" instead of "in-progress conversation state
  will be lost" — accurate description of what now happens.

### Fixed

- `token_filled` previously only checked the legacy
  `<accountsDir>/tokens/<name>.token` file, so every config-dir
  account (the documented flow!) showed "NO TOKEN YET" despite having
  working credentials. Now reports true when either source has usable
  credentials.
- Gemini reply parsing now reads `chats/*.jsonl` instead of scraping
  the screen, eliminating screen-dump noise in Telegram forwards.
- Session 'shell' provider's chrome stripper is now shell-aware so
  raw prompt characters don't leak into the channel forwarders.
- Web: copy now works over plain-HTTP LAN (Clipboard API requires
  HTTPS otherwise), terminal selection-driven copy works, copy pill
  is anchored at the selection with neutral styling.

### Security

- All disk reads in the cliacct path use `os.Lstat` and reject
  symlinks (`<accountsDir>/<name>/`, `<configDir>/.credentials.json`,
  `<configDir>/.claude.json`, the legacy token file). Defense in
  depth against an attacker who can write under the accounts tree.
- `migrateClaudeTranscript` Lstat-rejects symlinked sources before
  `os.Link` so a planted symlink can't be hardlinked into the new
  account's tree and read as conversation history by `claude --resume`.
- Telegram inbound is gated to the configured owner across all
  message types, not just control commands.

### API

- New: `POST /api/v1/claude-accounts/{id}/accept-identity` — clears
  the identity-drift baseline by recording the current on-disk email
  as the new accepted identity.

### Config

- New: `[providers.claude] watcher_enabled` (default true). Set to
  false to disable the fsnotify watcher; the Import-local button
  still works on demand.

## [v2.3.4] — 2026-05-29

### Fixed

- **Language toggle in the web Topbar moved its checkmark but UI
  strings didn't switch.** The zustand → i18next bridge ran as a
  module-level `useLocale.subscribe(...)` in `i18n.ts` that mounted
  before React. Under React 19 StrictMode + Vite HMR + zustand persist
  hydration the subscription could end up registered against a store
  snapshot React never re-reconciled with, so picking a language moved
  the dropdown's checkmark (which reads from the store) without
  triggering `i18n.changeLanguage()`. Moved the bridge into a
  `<LocaleSync />` React effect under `QueryClientProvider` so it
  shares the same lifecycle as every other `useTranslation()`
  consumer and they update in lockstep (#267).

- **Nine UI strings rendered their placeholders literally** —
  "update available → {{version}}", "Suggested ({{count}})", "Updated
  {{from}} → {{to}}", "connected · {{count}} tools", and the three
  About-panel version-toaster lines all showed the `{{var}}` template
  instead of the substituted value. The web i18next interpolation is
  configured for single-brace `{name}` but those particular keys were
  authored with the i18next default `{{name}}`. Normalized them across
  both locales (#261).

- **Mobile `flutter build apk` failed with hundreds of parser errors
  after slang codegen.** Mobile's slang config uses
  `string_interpolation: braces` (matching the web) but the same
  `{{var}}` typos that produced literal placeholders on web produced
  invalid Dart on mobile — `({required Object {version})` and
  `${{version}}` — that wouldn't compile. Same normalization as #261,
  plus a refresh of the generated `strings*.g.dart` outputs and
  alignment of `app/mobile/pubspec.yaml` to the product version
  (#264).

### Changed

- **App icons now show the new wooden-cart wordmark glyph instead of
  the old pink-gradient "D".** README was already updated to the
  opendray.dev wordmark, but the running surfaces — web favicon,
  Android launcher mipmaps, the full iOS `AppIcon.appiconset`, and
  the repo-root `assets/icons/logo/` set — hadn't caught up, so a
  fresh install showed the new brand on GitHub and the old brand on
  the device. Regenerated every square icon surface from a single
  1024×1024 source so proportions stay consistent across sizes
  (#266).

- **The Providers page now asks for confirmation before upgrading a
  CLI that has live sessions on it.** Linux file-replacement
  semantics mean an already-loaded session keeps the old binary in
  memory, but a long session with lazy / dynamic imports or in-flight
  subprocess work can pick up new code mid-run. When `n > 0`
  non-terminal sessions are using the provider, clicking Update opens
  a dialog with the count and an honest explanation of the trade-off;
  with no live sessions Update still fires immediately, as before.
  Update-check responses also stay fresh for an hour now (matching
  the server-side npm cache) instead of being re-fetched on every tab
  switch (#263).

## [v2.3.3] — 2026-05-24

### Fixed

- **About panel showed no version and the self-update button did
  nothing.** The dashboard called the version / self-update API at
  `/version` and `/version/update` instead of `/api/v1/...`, so the
  requests 404'd. Added the `/api/v1` prefix (#251).

## [v2.3.2] — 2026-05-24

### Fixed

- **Cross-session memory injection rendered every fact as `- ---`.**
  The "Recent project memory" banner took the first line of each
  memory, which for frontmatter-authored facts is the `---` YAML
  delimiter. It now skips the frontmatter and surfaces the
  `description` (falling back to the first body line) (#250).

## [v2.3.1] — 2026-05-24

### Fixed

- **Copy buttons silently failed over plain HTTP (LAN IP / mobile).**
  `navigator.clipboard` is only exposed in a secure context. Added a
  shared `copyText()` helper that falls back to `execCommand('copy')`
  and routed the existing copy callsites through it (#249).

## [v2.3.0] — 2026-05-23

### Fixed

- **Live sessions were destroyed by a daemon restart (e.g. a
  self-update).** Sessions are now marked `interrupted` on a gateway
  shutdown and auto-resumed on the next startup via their stored agent
  session id (`--resume`), with bounded-concurrency spawning and an
  optional `OPENDRAY_AUTO_RESUME_MAX` cap. A drain gate warns before a
  self-update interrupts running work (#247).
- **404 page instead of the login screen after a restart.** The 401
  redirect now respects the dashboard base path (→ `/admin/login`)
  and keeps `next` router-relative (#248).
- **Brand icons broke under a non-`/admin` base path** (#246).

## [v2.2.2] — 2026-05-23

### Added

- **Memory: global-scope injection fallback + recency default** — a
  fact told to one session surfaces in another regardless of cwd
  (#244).
- **Transport-aware MCP editor template + "unsupported" badge for
  Codex** (#242).

### Fixed

- **Memory endpoints are now scope-gated** (admin or
  `memory:read` / `memory:write`) (#245).

## [v2.2.1] — 2026-05-22

### Added

- **Always-visible "Check for updates" + re-install action in the
  About panel** (#243).

### Fixed

- **Remote MCP URL normalization** (#230).

## [v2.2.0] — 2026-05-22

### Added

- **In-dashboard update notification + one-click background
  self-update** (#241).
- **Startup warning when W^X (MemoryDenyWriteExecute) blocks
  executable memory** (#240).

### Changed

- **Repository renamed `opendray_v2` → `opendray`** across
  code/config/docs; install / uninstall URLs updated (#238, #237).

### Fixed

- **Dropped `MemoryDenyWriteExecute`** from the systemd unit — it
  broke Codex / Gemini sessions (#218).

## [v2.1.1] — 2026-05-22

### Added

- **Responsive mobile web layout** — slide-over nav + inspector with
  edge handles (#236).

### Fixed

- **Telegram channel:** handle `/start`, and a clearer `/list` header
  for terminated sessions (#235).

## [v2.1.0] — 2026-05-22

### Added

- **Per-provider model management from the dashboard** (#229).
- **Real CLI version + "update available" surfaced in the providers
  API/UI** (#227).
- **Interactive session switching via `/select` + Talk-to buttons**
  in channels (#226).
- **Validate MCP servers from the Plugins page** (#233).
- **Windows installer: a true one-liner** — auto-installs WSL2 +
  Ubuntu, runs the installer, and persists across reboot (#213).

### Changed

- **Hardened the merged Update action** — provider mutations are gated
  and the update path degrades gracefully (#234).

### Fixed

- **Session list shows session names in `/list` instead of bare ids**
  (#224).
- **Spawned CLIs get a color-capable `TERM`** so Claude/Codex/Gemini
  render in color (#225).
- **macOS installer hardening** — robust local Postgres provisioning,
  configured-port binding, idempotent launchd reload, bash 3.2
  compatibility, and a launchd PATH that finds brew-installed CLIs
  (#208, #209, #211, #212, #231, #232).
- **Windows installer:** OS-build guard, auto-resume after a WSL
  reboot, PowerShell 5.1 compatibility (#214).
- **Installer:** validate DB identifiers and don't abort on a free /
  commented-out Postgres port (#210).

### Security

- **Scrubbed dev-internal docs + personal-network references from the
  public repository** (#204).

## [v2.0.5] — 2026-05-18

### Added

- **Flutter mobile session terminal now has the URL detector
  badge.** Same model as the web admin: the PTY byte stream is
  scanned for http(s) URLs with the same state-machine extractor
  that re-assembles CLI-soft-wrapped OAuth URLs. A floating pill
  in the top-right corner of the terminal — primary tap opens the
  most recent URL in the OS browser via `url_launcher`, secondary
  `⋯` button opens a bottom-sheet with every URL (newest first)
  for picking older ones. Closes the OAuth-on-Flutter-app gap
  reported alongside the web fix.

### Changed

- **Web login no longer pre-fills the username with "admin".** The
  install wizard lets operators pick any username, so seeding the
  field forced everyone-who-didn't-keep-the-default to backspace
  before typing. The field is now empty by default and autofocused.

## [v2.0.4] — 2026-05-18

### Fixed

- **URL extractor now re-assembles CLI-soft-wrapped URLs.** AI CLIs
  (claude-code, codex, gemini) hard-wrap long OAuth URLs at the
  terminal column width by emitting literal `\n` characters every
  ~55 chars. The v2.0.1 / v2.0.2 / v2.0.3 extractor used a `[^\s]+`
  regex that stops at `\n`, so it captured only the first wrapped
  segment (e.g. `https://...&client_`). Tapping the badge opened a
  truncated URL, the OAuth provider rejected it, and the operator
  couldn't authenticate.

  The extractor is now a state-machine walker that anchors on
  `https?://`, consumes URL-body characters, and treats a single
  internal `\n` as a soft-wrap when the current line is ≥ 40 chars
  long (matches real CLI wrap width; well above "<intro phrase>\n
  <url>" prose patterns). Paragraph breaks (`\n\n`), single
  newlines followed by non-URL characters, and short prose lines
  still terminate the URL correctly.

  Verified against the actual 450-char claude-code OAuth URL that
  was failing in production: extractor now produces ONE complete
  URL (vs. two truncated segments).

## [v2.0.3] — 2026-05-18

### Fixed

- **Terminal URL badge always opens with one tap, regardless of how
  many URLs the session has accumulated.** v2.0.2 made the `N = 1`
  case one-tap, but real sessions usually have ≥ 2 URLs by the time
  the auth flow runs (the CLI's welcome banner often prints a docs
  link before the OAuth URL), and that fell back to the two-tap
  dialog flow. The badge now ALWAYS opens the **most recent** URL
  on a single tap — which is the OAuth URL in 100% of the
  `claude login` / `gemini auth login` / `codex login` cases.

  Multi-URL access stays available via a small `⋯` button beside
  the primary anchor — tapping it opens the same list dialog as
  before, so operators can still grab an older URL when they need
  it. The dialog row Open buttons are also real anchors (not
  `window.open()`) for the same popup-blocker reason.

  This is a web-admin-only fix. The Flutter mobile app's terminal
  surface doesn't have URL detection yet — separate follow-up.

## [v2.0.2] — 2026-05-18

### Added

- **Service-control subcommands**: `opendray start`, `opendray stop`,
  `opendray restart`, `opendray status`. Thin wrappers over
  `systemctl` (Linux) and `launchctl` (macOS) so operators don't
  have to remember the platform-native incantation. On Linux, the
  binary auto-prepends `sudo` if the caller isn't root. On macOS,
  defaults to the user LaunchAgent (`gui/$UID/com.opendray.opendray`);
  pass `--system` to target the LaunchDaemon scope.

### Fixed

- **One-tap link open for the OAuth URL badge.** When a session has
  exactly one detected URL (the common AI-CLI auth case: `claude
  login` / `gemini auth login` / `codex login` each print one OAuth
  URL), the floating "🔗 1 link" badge is now itself an
  `<a target="_blank">` — a single tap goes straight to the
  browser, no intermediate dialog. The dialog still appears when
  ≥ 2 URLs are detected, so multi-link sessions still get the
  disambiguating UI. In the dialog, the "Open" button is also a
  real anchor now, which avoids popup-blocker gating on some
  mobile browsers.

## [v2.0.1] — 2026-05-18

### Removed

- **Docker deployment path.** opendray is a host-resident gateway —
  it spawns AI CLIs via PTYs and shares process state (`~/.claude`,
  ssh-agent, project files) with them, which is incompatible with the
  container isolation a production Docker deploy would impose.
  Removed `Dockerfile`, `docker-compose.yml`, `docker-compose.test.yml`,
  `.dockerignore`, `.env.example`, the GHCR push job from the release
  workflow, and the Docker-Compose sections from README / docs.
- **In-app Tutorial page.** All 84 markdown sections plus
  `Tutorial.tsx` removed; docs now live in a dedicated repo that will
  publish independently. Sidebar entry, `/tutorial` route, and i18n
  keys (`nav.tutorial`, `web.providers.claudeAccounts.tutorialTooltip`,
  `web.providers.claudeAccounts.architectureLink`) removed in parallel.

### Fixed

- **"No Claude accounts" empty state** (Providers page + Spawn dialog,
  web + mobile) now tells operators the actual setup path: spawn a
  session and run `claude login` in the terminal. The previous wording
  pointed at the gateway-host shell workflow (works only for SSH-
  capable operators) and incorrectly implied a system
  `ANTHROPIC_API_KEY` fallback. The shell workflow remains available
  in the Providers page text for power-users juggling multiple
  identities; it's just no longer the headline instruction.

### Changed

- Brand: web favicon, docs hero, iOS `AppIcon.appiconset` (15 sizes),
  Android mipmap (5 densities), and `app/mobile/assets/brand/`
  launcher source refreshed from a new canonical set in
  `assets/icons/logo/`. Now tracked in-repo so a future refresh is
  one `cp` + the existing `sips` resize loop.

### Added — install / uninstall / update tooling

Lifecycle scripts and binary subcommands that grew out of a fresh-
LXC end-to-end install test. Everything below is `curl | bash`–
reachable, idempotent, and works on Linux (Ubuntu / Debian) +
macOS; Windows is funneled through WSL2.

- **One-line installer wizard** (#185 #186)
  - `scripts/install.sh` — dual-mode entry: dispatches to the OS
    installer in a local checkout, or shallow-clones the repo and
    re-execs when piped from `curl`.
  - `scripts/install-linux.sh` — apt + systemd; walks the operator
    through Postgres (existing or fresh `postgresql-16` +
    `pgvector` install), AI-CLI choice, admin credentials, listen
    address, release-tarball binary install, schema migration,
    and a hardened systemd unit. Optional `--from-source` builds
    the binary + web bundle from a checkout instead.
  - `scripts/install-macos.sh` — brew + LaunchAgent (or
    `--launchd-daemon` for system-wide), same flow. Detects Apple
    Silicon vs Intel for the right release asset.
  - `scripts/install-windows.ps1` — PowerShell helper for WSL2:
    detects existing WSL, otherwise prints the install command +
    reboot guidance, then hands off to the Linux installer
    inside Ubuntu.
- **One-line uninstaller** (#191)
  - Default mode stops + removes the gateway runtime but keeps
    `config.toml`, data directory (bcrypt keyfile, sessions,
    notes, vault), logs, and the PostgreSQL database — so a
    re-install picks up where you left off.
  - `--purge` (or `OPENDRAY_PURGE=1`) drops the DB + role,
    deletes config / data / logs, removes the service user.
  - Post-purge verification step: walks the standard install
    paths and bails loudly with `ls -la` output if anything
    survived. "No trace left" gets *checked*, not assumed.
- **`opendray update` subcommand** (#194)
  - Fetches the latest GitHub release, picks the goreleaser
    asset matching this host's `GOOS/GOARCH`, verifies SHA-256
    against the release's `SHA256SUMS`, then atomically replaces
    `/proc/self/exe` via temp+rename.
  - Flags: `--check` (probe only), `--force` (re-install same
    version), `--yes` (skip confirm), `--restart` (`systemctl
    restart opendray` after replace, Linux only).
  - Fails fast with a "try with sudo" hint when it can't write
    the install directory — no silent no-op.
- **`opendray providers <list|update>`** (#194)
  - Detects installed AI CLIs (`claude`, `gemini`, `codex`),
    prints versions + paths.
  - `update` re-runs `npm install -g` per CLI; `--check` shells
    out to `npm view <pkg> version` to compare current vs
    npm-latest.
  - `--only claude,gemini` restricts to a subset; `--json` on
    `list` for scripted consumers.

### Security

- **Secrets out of `config.toml`** (#192). The wizard now writes
  the database URL + admin bootstrap password to a separate file:
    - Linux: `/etc/opendray/opendray.env` (mode `0640 root:opendray`),
      consumed by systemd via `EnvironmentFile=`.
    - macOS: `~/.opendray/opendray.env` (mode `0600`), consumed
      by a tiny launcher wrapper (`~/.opendray/bin/opendray-launcher.sh`)
      that the LaunchAgent's `ProgramArguments` invokes — launchd
      has no `EnvironmentFile` equivalent.
  - `config.toml` is now `0644` and contains only non-secrets
    (listen, log config, `[admin].user`, runtime data dir).
  - Existing opendray env-var override layer
    (`OPENDRAY_DATABASE_URL`, `OPENDRAY_ADMIN_PASSWORD`, etc.)
    does the actual wiring — no Go changes needed.

### Fixed (install wizard, all reported during the LXC walkthrough)

- `curl | bash` prompts work — wizard re-attaches stdin to
  `/dev/tty` so EOF on the pipe doesn't make every `read` fail
  under `set -e` (#187).
- `run_priv -E …` / `run_priv -u …` no longer trip "command not
  found" when running as root — new `run_priv_env` /
  `run_priv_as` helpers handle both root + non-root paths (#188).
- pnpm moved to the `--from-source` branch only; default-path
  Node install no longer hangs on corepack's silent download
  (#189).
- AI CLI install shows npm's progress bar instead of `--silent
  >/dev/null` (so a 90-second download doesn't look like a hang)
  (#189).
- Admin login works after install: wizard writes `[admin].user`
  in addition to the password; matches opendray's auth contract
  (#190).
- Customisable admin username (was hard-coded to "admin") (#190).
- Final-summary URL resolves the host's LAN IP for `0.0.0.0`
  listens instead of printing the `<this-host>` placeholder
  (#190).
- Colour codes render in the summary block — colour vars use
  ANSI-C quoting so heredoc interpolation carries real ESC
  bytes (#190).
- `uninstall --purge` deletions are unconditional now; survived
  the previous flag-gated logic that occasionally left
  `config.toml` on disk (#192).
- Env-var alternative for the purge flag (`OPENDRAY_PURGE=1
  bash`) — survives `bash -s -- --flag` paste-newline weirdness
  (#193).

### Documentation

- README hero: typographic v2 logo + status / license / CI /
  GHCR badges + "What is opendray?" five-bullet section + paired
  EN / ZH `README.md` / `README.zh.md` (#180 #181 #182).
- One-liner install / uninstall snippets at the top of
  `## Install` on both READMEs (#186 #192 #193).
- `docs/getting-started.md` (+ `.zh.md`) — 15-minute end-to-end
  walkthrough that mirrors what the wizard does (#183).
- `docs/operator-guide.md` strengthened on Docker-deploy scope —
  decision-question framing makes the "no session spawn" limit
  unmissable (#184).
- `scripts/README.md` documents the wizard, file layout (now
  including the secrets / config split), troubleshooting table,
  and the env-var alternatives for the purge / yes flags.

### Branding

- Unified launcher icons across web favicon, iOS
  `AppIcon.appiconset` (15 sizes), and Android mipmap densities
  (5) using the cropped typographic v2 logo (#182).

## [v2.0.0] — 2026-05-17

### Versioning realignment

- **Re-tagged from the previous `v1.0.0` tag** (issue #165). The
  major version now reflects this codebase's identity as the second
  generation of the opendray product (`opendray_v2`). The previous
  `v1.0.0` tag was deleted (had three duplicate draft releases on
  GitHub, all deleted; no published release; no downstream
  installers depend on it).
- New [VERSIONING.md](./VERSIONING.md) documents the
  major-as-generation policy and what triggers future bumps.

### Added

- Per-session bypass toggle in the Spawn dialog (mobile + web).
  Provider-aware: Claude → `--dangerously-skip-permissions`,
  Codex → `--ask-for-approval never`, Gemini → `--yolo`. Off by
  default; the previous all-or-nothing provider config setting
  still works for "always bypass" deployments.

### Changed

- Spawn dialog's Claude account picker now appears immediately on
  open (mobile + web). Previously it waited for the operator to
  re-tap the provider dropdown because the parent state's
  provider id stayed unset.
- When 2+ Claude accounts are registered, the `Default (env /
  system)` option disappears from the Claude account picker; the
  first enabled account auto-selects. Single-account setups
  retain the Default option.

### Fixed

- Release workflow's `ghcr` job now produces image tags on
  `workflow_dispatch`. `docker/metadata-action` was reading
  `github.ref` (a branch when dispatched manually), so `type=semver`
  rules emitted zero tags and buildx failed with "tag is needed when
  pushing to registry". Each rule now passes `value=${{ env.TAG }}`
  so the same ruleset works for both `push:tags` and
  `workflow_dispatch` entry points.

### Added

- Release workflow gains a `ghcr` job that builds the multi-arch
  Dockerfile (linux/amd64 + linux/arm64) and pushes to
  `ghcr.io/opendray/opendray` on every tag release. Job-scoped
  `packages: write` (the parent `release` job stays at
  contents+id-token least-privilege). Tag set covers `:1.0.0`,
  `:1.0`, `:v1.0.0`, plus `:latest` for non-prerelease semver.
  SHA-pinned actions throughout, matching the existing release-
  pipeline pattern.

- `.github/workflows/release.yml` — automated release pipeline.
  Triggers on `v*` tag push (or manually via workflow_dispatch with a
  tag input). Produces a goreleaser draft release with:
    * cross-compiled archives (linux/darwin × amd64/arm64) +
      `SHA256SUMS`
    * cosign keyless OIDC signatures (`SHA256SUMS.sig`,
      `SHA256SUMS.pem`) via Sigstore Fulcio — no long-lived key
    * SPDX SBOM via anchore/sbom-action
  Permissions limited to `contents: write` (release upload) and
  `id-token: write` (cosign OIDC). Supply-chain hardening: SHA-pinned
  cosign-installer, sbom-action, and goreleaser-action; fail-fast
  tag-format validation on workflow_dispatch.
- `deploy/` directory with reference deploy artefacts:
  - `deploy/systemd/opendray.service` — production-ready systemd unit
    with sandboxing (`NoNewPrivileges`, `ProtectSystem=strict`, etc.),
    `migrate`-then-`serve` startup, 20s graceful-stop window.
  - `deploy/lxc/proxmox-pty-notes.md` — Proxmox-specific guide covering
    privileged vs unprivileged container PTY behaviour, the cgroup +
    bind-mount config required for unprivileged LXCs, networking +
    pgvector + pg_dump-version checks, and a pre-go-live checklist.
  - `deploy/README.md` — index pointing operators at the right artefact
    for their topology.
  - operator-guide.md "Where to look next" section now links to `deploy/`.
- ADR 0016 (Proposed): backup-format v2 design for per-install PBKDF2
  salt. Captures the four binding decisions (in-header storage,
  version-byte bump 1→2, per-Seal salt provenance, indefinite v1
  read compat) and the three-PR rollout. Implementation pending.
- LICENSE file (Apache 2.0) — previously declared in README only.
- SECURITY.md — threat model, default posture, deployment checklist, report channel.
- CONTRIBUTING.md — dev setup, test commands, PR + commit conventions.
- CHANGELOG.md — this file.

### Changed
- `internal/backup/cipher.go`: 6-line comment on `kdfSalt` flagging it
  as a frozen v1 protocol constant and pointing at ADR 0016. No code
  behaviour change.
- Renumbered ADR `0011-memory-subsystem.md` → `0014-memory-subsystem.md` to
  resolve the duplicate-0011 collision with `0011-channel-rich-content-and-bridge.md`.
  Updated cross-references in README, ADR 0013, and the embed-onnx stub.

## [v1.0.0 — retracted] — 2026-05-09

> **Note.** This tag was retracted on 2026-05-17 and the work it
> covered is folded into [v2.0.0](#v200--2026-05-17) above. See
> issue #165 and [VERSIONING.md](./VERSIONING.md) for the rationale.
> Original section preserved verbatim below for historical context.

First stable release. Tagged at commit `fe96fd8` on `main`. Web frontend
+ backend feature-complete; mobile + Slack inbound + automated release
workflow deferred to v1.x per the post-v1.0 roadmap. v1
(`Opendray/opendray`) keeps running in production through this quarter
per ADR 0001.

The feature inventory below was originally captured under
`[v1.0-rc] — 2026-05-05`; section was promoted to `[v1.0.0]` on tag.

### Added (since the greenfield start)

- **M0 — composition root:** `internal/app/`, config loader (`internal/config/`),
  pgx pool + hand-rolled migration runner (`internal/store/`), event bus
  (`internal/eventbus/`), structured logging via slog.
- **M1 — sessions:** PTY lifecycle, ring-buffer streaming, WS handler,
  resume-via-reconnect (per ADR 0003).
- **M2 — CLI catalog:** provider manifests + per-id user config
  (`internal/catalog/`).
- **M2.5 — admin auth:** bearer tokens with constant-time password compare
  and 24h TTL (`internal/auth/`).
- **M3 — integrations:** external-app registry, `/api/v1/proxy/{prefix}/*`
  reverse proxy, integration call log (`internal/integration/`, ADR 0006,
  ADR 0010).
- **M4 — channels:** channel hub + Telegram, Slack, Discord, DingTalk,
  Feishu, WeChat, WeCom (`internal/channel/`, ADR 0005, ADR 0011-channel).
- **Memory:** built-in pgvector cross-CLI memory layer
  (`internal/memory/`, ADR 0014). Three-CLI mirror keeps Claude / Codex /
  Gemini transcripts aligned. ONNX local-embedding optional via
  `-tags local_onnx`.
- **Ambient memory:** auto-capture from active sessions + auto-injection
  on session start (ADR 0013).
- **Backup + export:** AES-256-GCM encrypted PostgreSQL dumps,
  S3/WebDAV/SFTP/rclone targets, admin export/import bundles
  (`internal/backup/`, ADR 0012).
- **Web admin (W0–W5):** React 19 + Vite + Tailwind v4 + shadcn/ui +
  TanStack Router/Query + Zustand + xterm.js. Single SPA bundled into
  the Go binary via `go:embed` (ADR 0007, ADR 0008).
- **Events stream:** admin-bearer-authed `/api/v1/integrations/_events`
  WebSocket (ADR 0009).

### Deferred to post-v1.0

- Mobile (Flutter) client — replaced by responsive web in v2 phase 2.
- Slack inbound (M5+).
- Deploy automation (release toolchain — goreleaser, Dockerfile,
  systemd unit) lands in a follow-up PR.
- e2e Playwright harness.

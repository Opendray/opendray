# Changelog

All notable changes to OpenDray v2 are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Version numbers follow this project's own **major-as-generation**
strategy — major version = product generation, minor = feature
iteration, patch = fix / polish. See [VERSIONING.md](./VERSIONING.md)
for the full rationale and what triggers a major bump.

## [Unreleased]

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

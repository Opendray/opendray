# README Refresh Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refresh `README.md` + 9 translations, add `llms.txt`, sync GitHub repo description + topics so opendray accurately represents its current 5-CLI capability surface, looks comparable to top OSS repos, and is discoverable via both search engines and LLM answers.

**Architecture:** One atomic PR on branch `docs/readme-refresh`. Sequential commits: screenshots first (visual assets are prerequisites for the README changes that reference them), then English canonical README, then `llms.txt`, then the 9 translations. Then PR + CI + merge. Then GitHub metadata update (after merge, so description on main is consistent with README on main). No version bump (this is a docs change per VERSIONING.md).

**Tech Stack:** Playwright (headless Chromium) for screenshot capture; Node.js for scripts; `gh` CLI for GitHub metadata + PR ops; Markdown (GitHub-flavored) for content.

**Reference documents:**
- Spec: `docs/superpowers/specs/2026-07-02-readme-refresh-design.md` (committed on this branch as `f5738c1`)
- Current README: `README.md` (474 lines)
- Existing translations: `README.zh.md`, `README.fa.md`, `README.es.md`, `README.pt-BR.md`, `README.ja.md`, `README.ko.md`, `README.fr.md`, `README.de.md`, `README.ru.md`
- Provider catalog (source of truth for which CLIs to name): `internal/catalog/builtin/{claude,codex,antigravity,grok,opencode,shell}.json`

## Global Constraints

- **No em-dashes** (`—` U+2014, `–` U+2013) anywhere in the output. Use periods, commas, parens, colons, or semicolons instead. Rule applies to English AND all 9 translation files including RU, FR, DE, FA, ZH, JA, KO, ES, PT-BR.
- **No Claude attribution** in commit messages, PR title, PR body, or issue text. No "Co-Authored-By: Claude", no "Generated with Claude Code" footer.
- **No version bump.** VERSIONING.md classifies docs as patch or no-bump; this is a docs-only refresh, no CHANGELOG entry required unless the merging maintainer wants one.
- **Do not touch translations manually beyond section-parity requirements.** Match the existing translation style per language. Do not "improve" pre-existing prose.
- **Single atomic PR.** All commits land on branch `docs/readme-refresh` and merge in one PR. No amending, no force-push, no split PRs.
- **CI must be green before merge.** The `check-i18n-parity` job runs on every PR; `check-readme-drift` is advisory but should show clean (all 9 translations updated in this PR).
- **Base branch:** `main`. Do not target any other branch.

---

## File Structure

**Created:**
- `docs/assets/screenshots/hero-web-admin.png` (wide dashboard shot)
- `docs/assets/screenshots/session-live.png` (live session view with memory rail)
- `docs/assets/screenshots/mobile-session.png` (mobile viewport view)
- `docs/assets/screenshots/channels-telegram.png` (channels page)
- `llms.txt` (root)
- `scripts/capture-screenshots.mjs` (throwaway; deleted at end)

**Modified:**
- `README.md` (English canonical, full rewrite per spec)
- `README.zh.md`, `README.fa.md`, `README.es.md`, `README.pt-BR.md`, `README.ja.md`, `README.ko.md`, `README.fr.md`, `README.de.md`, `README.ru.md` (9 files, structural sync + translated new sections)

**External state changed:**
- GitHub repo description on `Opendray/opendray`
- GitHub repo topics on `Opendray/opendray`

---

### Task 1: Capture product screenshots

**Privacy constraint (operator-set):** Screenshots must come from a fresh
seeded gateway, NOT from the production instance on `127.0.0.1:8770`. The
production instance carries real user data; the shots go into a public README.

**Files:**
- Create: `docs/assets/screenshots/hero-web-admin.png`
- Create: `docs/assets/screenshots/session-live.png`
- Create: `docs/assets/screenshots/mobile-session.png`
- Create: `docs/assets/screenshots/channels-telegram.png`
- Create then delete: `scripts/capture-screenshots.mjs`
- Create then delete: `/tmp/opendray-shots/config.toml`, `/tmp/opendray-shots/data/`

**Interfaces:**
- Consumes: none (first task)
- Produces: four PNG files at the paths above, each ≤500KB, referenced by name in later tasks.

- [ ] **Step 1: Spin up a seeded scratch gateway on `127.0.0.1:8771`**

Do NOT reuse the production instance on `:8770`. Create a fresh scratch
instance:

```bash
cd /var/lib/opendray/rcc/opendray
mkdir -p /tmp/opendray-shots/data
export OPENDRAY_SHOTS_PG_DB=opendray_shots
export OPENDRAY_SHOTS_PG_USER=opendray_shots
export OPENDRAY_SHOTS_PG_PASS=$(openssl rand -base64 24 | tr -d '=+/')
export OPENDRAY_SHOTS_ADMIN=$(openssl rand -base64 24 | tr -d '=+/')

# Create a scratch Postgres database + role (assumes local PG accessible
# from the operator user).
sudo -u postgres psql -c "CREATE ROLE $OPENDRAY_SHOTS_PG_USER LOGIN PASSWORD '$OPENDRAY_SHOTS_PG_PASS';" 2>&1 || true
sudo -u postgres psql -c "CREATE DATABASE $OPENDRAY_SHOTS_PG_DB OWNER $OPENDRAY_SHOTS_PG_USER;" 2>&1 || true
sudo -u postgres psql -d $OPENDRAY_SHOTS_PG_DB -c "CREATE EXTENSION IF NOT EXISTS vector;" 2>&1

cat > /tmp/opendray-shots/config.toml <<EOF
[server]
listen = "127.0.0.1:8771"

[database]
url = "postgres://$OPENDRAY_SHOTS_PG_USER:$OPENDRAY_SHOTS_PG_PASS@127.0.0.1:5432/$OPENDRAY_SHOTS_PG_DB?sslmode=disable"

[admin]
password = "$OPENDRAY_SHOTS_ADMIN"

[storage]
data_dir = "/tmp/opendray-shots/data"
EOF
```

Apply schema, seed fake data (see Step 2), then start:

```bash
go run ./cmd/opendray migrate -config /tmp/opendray-shots/config.toml
go run ./cmd/opendray serve -config /tmp/opendray-shots/config.toml &
SHOTS_PID=$!
sleep 5
curl -s -o /dev/null -w "shots gateway HTTP=%{http_code}\n" http://127.0.0.1:8771/admin/
```

Expected: HTTP 200.

- [ ] **Step 1b: Seed fake data**

Write `/tmp/opendray-shots/seed.sql` with fake but plausible fixtures. Use
generic names ("Prompt refactor", "Docs update", "Bug triage") and never
real personal info. Seed:

1. Four sessions in Sessions view, showing multi-provider coverage:
   Claude Code / Codex / Grok Build / shell. Each with a fake project name
   ("app/web", "internal/session", "docs/", "misc/").
2. 8-12 memory entries with generic technical text (no personal data).
3. Three channels configured but disabled: Telegram, Slack, Discord (do not
   put real webhook URLs or tokens; use `sk_fake_xxx` style placeholders).

Apply:

```bash
sudo -u postgres psql -d $OPENDRAY_SHOTS_PG_DB -f /tmp/opendray-shots/seed.sql
```

If direct SQL is fragile against the migration schema, fall back to using
the REST API against `127.0.0.1:8771` with the admin credentials to POST
the fixtures. Either path lands the same data.

- [ ] **Step 2: Install Playwright + Chromium**

Run:

```bash
cd /var/lib/opendray/rcc/opendray
mkdir -p scripts
npx --yes playwright@1.48 install chromium --with-deps 2>&1 | tail -5
```

Expected: no error, chromium browser downloaded to `~/.cache/ms-playwright/`.

If `apt-get` prompts for sudo on `--with-deps` in a restricted env, retry without it: `npx --yes playwright@1.48 install chromium`.

- [ ] **Step 3: Write the capture script**

Create `scripts/capture-screenshots.mjs`:

```javascript
#!/usr/bin/env node
// Throwaway: captures the 4 screenshots the README refresh needs.
// Deleted at the end of the README-refresh task chain.

import { chromium } from 'playwright'
import { mkdirSync } from 'node:fs'
import { resolve } from 'node:path'

const BASE = process.env.OPENDRAY_URL || 'http://127.0.0.1:8770'
const OUT = resolve(process.cwd(), 'docs/assets/screenshots')
mkdirSync(OUT, { recursive: true })

const ADMIN_USER = process.env.OPENDRAY_ADMIN || 'admin'
const ADMIN_PASS = process.env.OPENDRAY_ADMIN_PASSWORD

if (!ADMIN_PASS) {
  console.error('OPENDRAY_ADMIN_PASSWORD env var required')
  process.exit(1)
}

const shots = [
  { name: 'hero-web-admin', path: '/admin/', viewport: { width: 1440, height: 900 } },
  { name: 'session-live', path: '/admin/sessions', viewport: { width: 1440, height: 900 } },
  { name: 'mobile-session', path: '/admin/sessions', viewport: { width: 390, height: 844 } },
  { name: 'channels-telegram', path: '/admin/channels', viewport: { width: 1440, height: 900 } },
]

const browser = await chromium.launch()
const context = await browser.newContext({
  httpCredentials: { username: ADMIN_USER, password: ADMIN_PASS },
})
const page = await context.newPage()

for (const s of shots) {
  await page.setViewportSize(s.viewport)
  await page.goto(`${BASE}${s.path}`, { waitUntil: 'networkidle' })
  await page.waitForTimeout(1500)
  await page.screenshot({ path: `${OUT}/${s.name}.png`, fullPage: false })
  console.log(`captured ${s.name}.png`)
}

await browser.close()
```

- [ ] **Step 4: Run the capture script**

```bash
cd /var/lib/opendray/rcc/opendray
export OPENDRAY_URL="http://127.0.0.1:8771"
export OPENDRAY_ADMIN_PASSWORD="$OPENDRAY_SHOTS_ADMIN"
node scripts/capture-screenshots.mjs
```

Expected output:

```
captured hero-web-admin.png
captured session-live.png
captured mobile-session.png
captured channels-telegram.png
```

If the admin uses cookie auth rather than HTTP basic (check by opening `/admin/` in a browser and observing the login form), replace the `httpCredentials` block in the script with an explicit login flow:

```javascript
const context = await browser.newContext()
const page = await context.newPage()
await page.goto(`${BASE}/admin/login`)
await page.fill('input[type=text], input[name=username]', ADMIN_USER)
await page.fill('input[type=password], input[name=password]', ADMIN_PASS)
await page.click('button[type=submit]')
await page.waitForURL(/\/admin\//)
```

- [ ] **Step 5: Verify screenshots**

```bash
ls -la docs/assets/screenshots/
file docs/assets/screenshots/*.png
```

Expected: 4 PNG files, each between 30KB and 500KB, each identified as `PNG image data, 1440 x 900` (desktop shots) or `390 x 844` (mobile shot). If any file is >500KB, compress with `optipng -o5 <file>` or reduce viewport.

- [ ] **Step 6: Tear down the scratch gateway and delete the throwaway script**

```bash
kill $SHOTS_PID 2>/dev/null || true
rm scripts/capture-screenshots.mjs
rm -rf /tmp/opendray-shots
sudo -u postgres psql -c "DROP DATABASE IF EXISTS $OPENDRAY_SHOTS_PG_DB;"
sudo -u postgres psql -c "DROP ROLE IF EXISTS $OPENDRAY_SHOTS_PG_USER;"
```

- [ ] **Step 7: Commit**

```bash
cd /var/lib/opendray/rcc/opendray
git add docs/assets/screenshots/
git commit -m "docs(readme): add product screenshots for the refresh

Four shots captured from a local instance for the README refresh:
- hero-web-admin.png (dashboard)
- session-live.png (live session with memory rail)
- mobile-session.png (mobile viewport)
- channels-telegram.png (chat channels page)"
```

Expected: commit succeeds, `git log --oneline -1` shows the new commit.

---

### Task 2: Rewrite `README.md` (English canonical)

**Files:**
- Modify: `README.md` (full rewrite per spec section 1 + section 2)

**Interfaces:**
- Consumes: PNG file names from Task 1 (`docs/assets/screenshots/{hero-web-admin,session-live,mobile-session,channels-telegram}.png`)
- Produces: canonical structure that all 9 translations in Task 4 will mirror. New section headings (verbatim, English) that translations must match:
  - `## What is opendray?`
  - `## Screenshots`
  - `## Features`
  - `## Architecture at a glance`
  - `## Comparison`
  - `## Who is this for?`
  - `## Install`
  - `## Quickstart (5-minute dev path)`
  - `## Production deploy`
  - `## Web frontend`
  - `## Mobile app`
  - `## FAQ`
  - `## Documentation`
  - `## Status`
  - `## Tests`
  - `## Relationship to v1`
  - `## License`

- [ ] **Step 1: Read the current README and the spec side by side**

```bash
cd /var/lib/opendray/rcc/opendray
wc -l README.md docs/superpowers/specs/2026-07-02-readme-refresh-design.md
```

Read both files fully before writing. The spec covers structural changes; the current README carries prose that mostly stays (install, quickstart, production deploy).

- [ ] **Step 2: Write the new hero (top of file through the language switcher)**

Replace lines 1-34 of `README.md` with:

```markdown
<p align="center">
  <a href="https://opendray.dev"><img src="docs/assets/logo.png" alt="opendray" width="180"></a>
</p>

<h1 align="center">opendray</h1>

<p align="center">
  <strong>Self-hosted gateway for Claude Code, Codex, Antigravity, Grok Build, and OpenCode. Run agent sessions on your own infrastructure. Drive from web, mobile, or chat.</strong>
</p>

<p align="center">
  <strong><a href="https://opendray.dev">opendray.dev</a></strong>
</p>

<p align="center">
  <a href="https://opendray.dev"><img alt="Website" src="https://img.shields.io/badge/website-opendray.dev-F43F5E"></a>
  <a href="https://github.com/Opendray/opendray/releases/latest"><img alt="Latest release" src="https://img.shields.io/github/v/release/Opendray/opendray?label=release&color=4f46e5"></a>
  <a href="LICENSE"><img alt="License Apache 2.0" src="https://img.shields.io/github/license/Opendray/opendray?color=blue"></a>
  <a href="https://github.com/Opendray/opendray/actions/workflows/ci.yml"><img alt="CI" src="https://img.shields.io/github/actions/workflow/status/Opendray/opendray/ci.yml?branch=main&label=CI"></a>
  <a href="https://github.com/Opendray/opendray/discussions"><img alt="Discussions" src="https://img.shields.io/github/discussions/Opendray/opendray?color=ec4899"></a>
  <br/>
  <img alt="Go" src="https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go&logoColor=white">
  <img alt="React" src="https://img.shields.io/badge/React-19-61DAFB?logo=react&logoColor=black">
  <img alt="Flutter" src="https://img.shields.io/badge/Flutter-mobile-02569B?logo=flutter&logoColor=white">
  <img alt="Postgres" src="https://img.shields.io/badge/PostgreSQL-15%2F16%2F17-336791?logo=postgresql&logoColor=white">
</p>

<p align="center">
  <strong>English</strong> · <a href="README.zh.md">简体中文</a> · <a href="README.fa.md">فارسی</a> · <a href="README.es.md">Español</a> · <a href="README.pt-BR.md">Português</a> · <a href="README.ja.md">日本語</a> · <a href="README.ko.md">한국어</a> · <a href="README.fr.md">Français</a> · <a href="README.de.md">Deutsch</a> · <a href="README.ru.md">Русский</a>
</p>

<p align="center">
  <a href="docs/getting-started.md"><img alt="Get started" src="https://img.shields.io/badge/get%20started-4f46e5?style=for-the-badge"></a>
  <a href="#screenshots"><img alt="See screenshots" src="https://img.shields.io/badge/see%20screenshots-ec4899?style=for-the-badge"></a>
  <a href="https://opendray.dev"><img alt="Live demo" src="https://img.shields.io/badge/live%20demo-F43F5E?style=for-the-badge"></a>
</p>

![opendray web admin: live agent sessions on your own infrastructure](docs/assets/screenshots/hero-web-admin.png)

---

Running Claude Code or Codex over SSH means the agent dies the moment your laptop closes. opendray runs it on a host that stays awake (a Mac mini under your desk, a NAS, a VPS) and lets you reattach from a web admin, a mobile app, or a chat message. Sessions keep executing whether or not anyone is connected. Multiple accounts get pooled with per-tier balancing and live account-switch. A local-first memory layer keeps every embedding on your network.

---
```

- [ ] **Step 3: Write "What is opendray?" section**

Replace the existing "What is opendray?" block (lines around 48-58) with:

```markdown
## What is opendray?

**opendray** wraps the AI coding CLIs you already use (Claude Code, Codex, Antigravity, Grok Build, OpenCode, plus any shell) and turns them into something you can drive from anywhere. Run sessions on your home server, NAS, or VPS. Get notified on Telegram when a session goes idle. Reply from your phone to feed the next prompt back in. All over a self-hosted gateway you control end to end.

- **One backend, three surfaces.** Single Go binary serving a React web admin and a Flutter mobile app, with every action also exposed over a REST + WebSocket API for third-party integrations.
- **Six bidirectional channels, no walled gardens.** Telegram, Slack, Discord, Feishu (飞书), DingTalk (钉钉), WeCom (企业微信), plus a Bridge adapter for anything custom. Replies on any channel route back into the right session.
- **Local-first memory.** ONNX / Ollama / LM Studio embeddings with three-scope retrieval (user, project, session), smart ranking, and cross-layer conflict detection. No vector data leaves your network.
- **Integration-grade API.** Scoped API keys, per-call audit log, reverse-proxy mounts. Treat opendray as the gateway behind your own product or just as a personal command centre.
- **Multi-account fleet for Claude, Codex, Antigravity.** Drop multiple logged-in accounts into the gateway; opendray auto-discovers them via a filesystem watcher, balances new sessions across enabled accounts, and lets you switch a live session between accounts **without losing the conversation** (transcript migrates under the hood). Each account row shows live capacity (subscription tier, rate-limit tier, active sessions, last-used, current login email).
- **Self-hosted, licence-clear.** Apache 2.0, one static binary, cosign-signed releases with SPDX SBOM. No telemetry, no cloud account, no subscription.
```

- [ ] **Step 4: Add Screenshots section**

Insert after "What is opendray?", before "Architecture at a glance":

```markdown
## Screenshots

<table>
  <tr>
    <td width="50%"><img alt="opendray dashboard showing multiple concurrent AI CLI sessions" src="docs/assets/screenshots/hero-web-admin.png"><br/><sub><b>Web admin.</b> Every running Claude Code, Codex, Antigravity, Grok Build, and OpenCode session on your host in one view.</sub></td>
    <td width="50%"><img alt="opendray live session with xterm terminal and memory retrieval rail" src="docs/assets/screenshots/session-live.png"><br/><sub><b>Live session.</b> Attach to any running agent from the browser. Full xterm PTY, transcript overlay for TUIs that skip wheel input, memory rail showing what got recalled for the current prompt.</sub></td>
  </tr>
  <tr>
    <td width="50%"><img alt="opendray mobile app showing a Claude Code session on a phone" src="docs/assets/screenshots/mobile-session.png"><br/><sub><b>Mobile.</b> Flutter app for iOS and Android. Feature parity with the web admin. Sideloaded, no App Store account required.</sub></td>
    <td width="50%"><img alt="opendray channels page with Telegram, Slack, Discord adapters wired in" src="docs/assets/screenshots/channels-telegram.png"><br/><sub><b>Chat channels.</b> Wire Telegram, Slack, Discord, Feishu, DingTalk, or WeCom. Any reply routes back to the right session.</sub></td>
  </tr>
</table>
```

- [ ] **Step 5: Add Features matrix**

Insert after Screenshots, before "Architecture at a glance":

```markdown
## Features

| | |
|---|---|
| **Sessions** | Attach to a running Claude Code, Codex, Antigravity, Grok Build, OpenCode, or shell session from web / mobile / chat. Sessions survive client disconnect and host reboot. Live transcript overlay for TUIs that skip wheel input. |
| **Providers** | 5 first-class AI coding CLIs plus arbitrary shell. Adding a new CLI is a JSON descriptor drop-in under `internal/catalog/builtin/`. Per-provider MCP-server injection (Vault, memory, integrations). |
| **Memory** | Three-scope retrieval (user, project, session). Local-first embeddings via ONNX, Ollama, or LM Studio. Cross-layer conflict detection. Global knowledge pages injected at spawn. Compiler flywheel distils episodes into reusable playbooks. |
| **Channels** | Telegram, Slack, Discord, Feishu, DingTalk, WeCom. Bridge adapter for custom transports. Bidirectional: sessions notify, replies feed back. |
| **Integrations** | REST + WebSocket API with scoped API keys, per-call audit log, and reverse-proxy mounts. Vault MCP for secret access. Public `docs/integration-guide.md`. |
| **Ops** | Single Go binary. One-line installer (Linux, macOS, WSL2). Self-managing (`opendray update / start / stop / providers update`). Encrypted PostgreSQL backups + data exports. Goreleaser pipeline with cosign-signed releases + SPDX SBOM. |
| **Security** | Apache 2.0. No telemetry, no cloud account. Cosign keyless (Sigstore) signing. `ProtectSystem=strict` systemd hardening. Multi-tenant-safe scoped tokens. |
```

- [ ] **Step 6: Update the Architecture Mermaid diagram to name all 5 CLIs**

Replace the `subgraph cli` block in the existing Mermaid (currently naming `cc`, `co`, `ag`, `sh`) with:

```mermaid
    subgraph cli [AI CLIs · spawned via PTY]
        cc[Claude Code]
        co[Codex]
        ag[Antigravity]
        gb[Grok Build]
        oc[OpenCode]
        sh[Shell]
    end
```

And add the corresponding edges:

```mermaid
    sess --> cc
    sess --> co
    sess --> ag
    sess --> gb
    sess --> oc
    sess --> sh
```

- [ ] **Step 7: Add Comparison section (two tables)**

Insert after "Architecture at a glance", before "Install":

```markdown
## Comparison

### opendray vs known AI clients

| | opendray | Claude Desktop | Cursor | CLI over SSH | ChatGPT Desktop |
|---|---|---|---|---|---|
| Session survives client disconnect | ✅ | ❌ | ❌ | ⚠️ (needs tmux / screen) | ❌ |
| Multi-account pool with live switch | ✅ | ❌ | ❌ | ❌ | ❌ |
| Cross-session memory layer | ✅ | ❌ | Partial | ❌ | Partial |
| Host filesystem + tool use | ✅ | Limited | ✅ | ✅ | Limited |
| Mobile client with feature parity | ✅ | ❌ | ❌ | ⚠️ (SSH client) | Partial |
| Chat channel adaptors | ✅ (6) | ❌ | ❌ | ❌ | ❌ |
| Self-hosted | ✅ | ❌ | ❌ | ✅ | ❌ |
| Licence | Apache 2.0 | Proprietary | Proprietary | (varies) | Proprietary |

### opendray vs self-hosted chat frontends

| | opendray | Open WebUI | LibreChat | Dify |
|---|---|---|---|---|
| Runs actual agent CLI (not just chat) | ✅ | ❌ | ❌ | Partial |
| Tool use + file writes on host | ✅ | ❌ | ❌ | Sandboxed |
| Multiple AI coding CLIs in one gateway | ✅ (5) | ❌ | ❌ | ❌ |
| Cross-session memory | ✅ | Basic | Basic | ✅ |
| PTY session with terminal reattach | ✅ | ❌ | ❌ | ❌ |
| Chat channel adaptors | ✅ (6) | Partial | Partial | ✅ |
| Licence | Apache 2.0 | MIT | MIT | Apache 2.0 |
```

- [ ] **Step 8: Add "Who is this for?" section**

Insert after Comparison, before Install:

```markdown
## Who is this for?

**Solo dev running a homelab.** You already have a Mac mini, NAS, or Proxmox box running 24/7. You've been running Claude Code over SSH but the session dies every time your laptop sleeps. You want the CLI to keep going, and you want to reattach from your phone on the train. opendray is the gateway that puts your host between you and the CLI.

**Small-team lead standing up shared AI infrastructure.** Your team has 3-5 Anthropic accounts spread across work + personal. You want to pool them, watch usage per account, and let anyone on the team drive a session from the browser. opendray gives you multi-account pooling, per-account observability, scoped API keys per teammate, and a mobile app they can install without an App Store submission.

**Integrator building on top of a session-runner.** You're building a product that needs to spawn Claude Code / Codex / Grok Build sessions with tool use, and you don't want to reimplement session lifecycle, PTY handling, memory, or channel routing. opendray exposes every action over REST + WebSocket with scoped keys, per-call audit logs, and reverse-proxy mounts. Treat it as your agent runtime.
```

- [ ] **Step 9: Preserve Install, Quickstart, Production deploy sections**

These sections (lines ~127-391 in the current file) stay word-for-word. Only touch: any em-dash present. Search:

```bash
grep -n "—\|–" README.md
```

Replace each hit with a period, comma, colon, or parens as context dictates. Do not add or remove any information; only swap the punctuation.

- [ ] **Step 10: Preserve Web frontend + Mobile app sections**

Keep the existing content. Add one deep-link at the end of the Mobile app section:

```markdown
**[→ See a mobile screenshot](#screenshots)**
```

- [ ] **Step 11: Write the FAQ section**

Insert after Mobile app, before Documentation:

```markdown
## FAQ

### What is opendray?

opendray is a self-hosted gateway that wraps the AI coding CLIs you already use (Claude Code, Codex, Antigravity, Grok Build, OpenCode, and shell) and turns them into sessions you can drive from a web admin, a Flutter mobile app, or six chat channels (Telegram, Slack, Discord, Feishu, DingTalk, WeCom). One Go binary. Apache 2.0. Your infra, your data, your tokens.

### Which AI CLIs does opendray support?

Five first-class providers as of v2.10.x: **Claude Code** (Anthropic), **Codex** (OpenAI), **Antigravity** (Google `agy`), **Grok Build** (xAI), and **OpenCode**. Plus arbitrary shell for anything else. Adding a new CLI is a JSON descriptor under `internal/catalog/builtin/`; no adapter code required for common cases.

### How is opendray different from Claude Desktop or ChatGPT Desktop?

Claude Desktop and ChatGPT Desktop are chat clients that run on your laptop and die when the laptop closes. opendray runs the actual agentic CLI on a host that stays awake and lets you reattach from anywhere. Sessions survive client disconnect, laptop sleep, and network drops. Multiple accounts get pooled with live switch between them.

### How is opendray different from running Claude Code over SSH?

Four things SSH does not give you: (1) session survives when you disconnect (no `tmux` gymnastics required, though you can still use tmux inside), (2) attach from a phone or a chat channel, not just a terminal, (3) shared memory layer across every session on the host, (4) multi-account pool with per-tier balancing and live account-switch mid-conversation.

### How is opendray different from Open WebUI, LibreChat, or Dify?

Those are chat frontends against a model API. They send prompts to `api.openai.com` (or similar) and render the response. opendray runs the actual agent CLI process on your host, complete with tool use, file writes, memory, and MCP servers. If a task needs `Read` / `Edit` / `Bash` on your host filesystem, opendray does it; chat frontends do not.

### Can I use multiple Claude, Codex, or Antigravity accounts?

Yes. Drop the logged-in credential directories on the host (Claude uses `CLAUDE_CONFIG_DIR`, Antigravity uses `$HOME` isolation) and opendray auto-discovers them via a filesystem watcher. New sessions balance across enabled accounts by tier + capacity. You can switch a live session between accounts without losing the conversation (transcript migrates under the hood). Rate-limit auto-failover carries context by default.

### Where is my data stored?

PostgreSQL on your host (bring your own instance, or use the one the installer bootstraps). Embeddings come from your own provider (ONNX bundled, Ollama, or LM Studio). No vector data, transcripts, or memory entries leave your network. No telemetry. No cloud account. `opendray` never phones home.

### Can I run this in Docker?

Not currently (v2.x). opendray spawns AI CLIs via PTYs and shares host process state (credential directories, ssh-agent, project files) with them. That is incompatible with the container isolation production Docker imposes. Use the pre-built binary and systemd or launchd (Linux + macOS both have one-line installers). See [Production deploy](#production-deploy).

### Does opendray work on a NAS, Mac mini, or Raspberry Pi?

NAS: yes on Synology / QNAP / TrueNAS-Scale (anything with Linux + Postgres). Mac mini: yes, this is a common deploy (LaunchDaemon shipped). Raspberry Pi: works on Pi 4 / Pi 5 but underpowered for concurrent sessions; single-user hobby use only.

### Is opendray free? What is the licence?

Apache 2.0. Free forever. No paid tier, no telemetry, no phone-home. Sponsors welcomed (see `.github/FUNDING.yml`).

### How do I contribute?

Read [`CONTRIBUTING.md`](CONTRIBUTING.md) and [`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md). Concrete ways in: (1) translate a README or docs page into a language we already ship, (2) add a provider descriptor for a new AI coding CLI under `internal/catalog/builtin/`, (3) write a channel adaptor for a chat platform we do not cover, (4) contribute screenshots for the docs, (5) file a bug or a feature request. PRs need CI green; translations are advisory-only; no CLA.
```

- [ ] **Step 12: Move Status/Version section to just before Tests**

Delete the current "Status" block near the top of the file (the one that says `v2.7.6`). Insert a new block just before the "Tests" section:

```markdown
## Status

Current generation: **v2.10.x**. See [`CHANGELOG.md`](CHANGELOG.md) for release history and [`VERSIONING.md`](VERSIONING.md) for the major-as-generation policy (major = product generation, not strict SemVer "breaking change").
```

- [ ] **Step 13: Preserve Documentation, Tests, Relationship to v1, License**

These sections stay word-for-word. Sweep any em-dashes.

- [ ] **Step 14: Final em-dash sweep**

```bash
cd /var/lib/opendray/rcc/opendray
grep -n "—\|–" README.md
```

Expected output: (empty). If any hit remains, replace and re-check.

- [ ] **Step 15: Verify Markdown renders**

```bash
node -e "const fs=require('fs'); const t=fs.readFileSync('README.md','utf8'); const brokenLinks=[...t.matchAll(/\]\(([^)]+)\)/g)].map(m=>m[1]).filter(u=>!u.startsWith('http')&&!u.startsWith('#')&&!u.startsWith('mailto:')); const missing=brokenLinks.filter(u=>{try{fs.accessSync(u.split('#')[0]);return false}catch{return true}}); console.log('missing local links:', missing);"
```

Expected: `missing local links: []`.

- [ ] **Step 16: Commit**

```bash
git add README.md
git commit -m "docs(readme): rewrite hero, add screenshots, features, comparison, personas, FAQ

Refresh the English canonical README:
- hero names all 5 first-class CLIs (Claude Code, Codex, Antigravity,
  Grok Build, OpenCode) instead of the stale 3-CLI list;
- add screenshots gallery (4 shots);
- add feature matrix, 2 comparison tables (vs known AI clients, vs
  self-hosted chat frontends), personas, and an 11-question FAQ;
- move stale version reference (v2.7.6) out of the status line;
  replace with 'current generation: v2.10.x, see CHANGELOG';
- preserve install / quickstart / production-deploy / mobile /
  documentation sections word-for-word.

Translations will follow in the next commits on this branch."
```

---

### Task 3: Add `llms.txt` at repo root

**Files:**
- Create: `llms.txt`

**Interfaces:**
- Consumes: nothing (independent from other tasks)
- Produces: canonical LLM-facing summary at repo root

- [ ] **Step 1: Write `llms.txt`**

Create `llms.txt` at repo root:

```markdown
# opendray

> Self-hosted gateway for the AI coding CLIs you already use (Claude Code, Codex, Antigravity, Grok Build, OpenCode, plus shell). Runs agent sessions on your own infrastructure. Drive from web, mobile, or chat channels. Open REST + WebSocket API for integrations. Apache 2.0.

## What it is

opendray is a Go binary that runs on your host (Linux, macOS, or Proxmox LXC). It spawns AI coding CLIs in PTYs, exposes them over a web admin (React), a mobile app (Flutter, iOS + Android), a REST + WebSocket API, and six chat channel adaptors (Telegram, Slack, Discord, Feishu, DingTalk, WeCom). A local-first memory layer keeps embeddings and transcripts on your network. Postgres + pgvector. No telemetry, no cloud account.

## Supported providers

- Claude Code (Anthropic)
- Codex (OpenAI)
- Antigravity (Google, `agy`)
- Grok Build (xAI)
- OpenCode
- Shell (arbitrary)

## Key differentiators

1. Sessions survive client disconnect. The agent keeps running when your laptop closes.
2. Multi-account pool with live account-switch mid-conversation for Claude, Codex, Antigravity.
3. Cross-session memory layer with three-scope retrieval (user, project, session) and local-first embeddings (ONNX, Ollama, LM Studio).
4. Bidirectional chat channels: reply on Telegram, the message routes back into the session.
5. One binary, cosign-signed releases with SPDX SBOM.

## Not what it is

- Not a chat frontend (compare Open WebUI, LibreChat). opendray runs the actual agentic CLI process with tool use and host filesystem access, not just prompts against a model API.
- Not a hosted service. There is no opendray SaaS. You run the binary on your host.
- Not a Docker deployment. PTY + host process state sharing is incompatible with production container isolation. Deploy via the pre-built binary + systemd or launchd.

## Install

One-line installer (Linux, macOS, WSL2):

```
curl -fsSL https://raw.githubusercontent.com/Opendray/opendray/main/scripts/install.sh | bash
```

Or via npm (`npm install -g opendray`), or a pre-built release binary.

## Links

- Repository: https://github.com/Opendray/opendray
- Website: https://opendray.dev
- Documentation: https://github.com/Opendray/opendray/tree/main/docs
- Getting started: https://github.com/Opendray/opendray/blob/main/docs/getting-started.md
- Integration guide: https://github.com/Opendray/opendray/blob/main/docs/integration-guide.md
- Changelog: https://github.com/Opendray/opendray/blob/main/CHANGELOG.md
- Licence: Apache 2.0
```

- [ ] **Step 2: Em-dash sweep**

```bash
grep -n "—\|–" llms.txt || echo "clean"
```

Expected: `clean`.

- [ ] **Step 3: Commit**

```bash
git add llms.txt
git commit -m "docs(seo): add llms.txt for AI-answer discoverability

Canonical LLM-friendly summary at repo root. Names all 5 supported CLIs
verbatim, states the differentiators against chat-frontend alternatives
(Open WebUI, LibreChat) and native desktop clients, and points at the
install one-liner + docs.

Emerging pattern: llms.txt is the robots.txt analogue for AI-crawl
answers. Model providers picking up this file get a canonical citable
source instead of stitching from README fragments."
```

---

### Task 4a: Translate to CJK (zh, ja, ko)

**Files:**
- Modify: `README.zh.md`, `README.ja.md`, `README.ko.md`

**Interfaces:**
- Consumes: new English README from Task 2 (section headings, prose, tables)
- Produces: three translated files with identical structure

- [ ] **Step 1: Read the current translation to learn the file's style**

For each file (zh, ja, ko):

```bash
head -200 README.zh.md
head -200 README.ja.md
head -200 README.ko.md
```

Note the tone, honorifics, technical-term handling (English kept vs localized), and header casing conventions per language.

- [ ] **Step 2: For each of zh, ja, ko: mirror the new English structure**

For each language, rewrite the file to match the new English README's section order and content. Keep:

- The existing language-switcher row unchanged (they already list all 10 languages).
- Existing code blocks and command examples verbatim (English).
- Screenshot references and image paths verbatim.
- Feature-matrix category labels in the target language; capability values in the target language.
- Comparison-table column headers in the target language; product names verbatim (Claude Desktop, Cursor, Open WebUI, etc.).
- FAQ questions translated verbatim (they are literal user queries in the target language too).

Style expectations per language:

- **Chinese (zh):** simplified, no honorifics. Match the existing tone.
- **Japanese (ja):** です・ます tone (matches existing file). Technical terms often kept in Katakana or English (session → セッション, memory → メモリ or memory).
- **Korean (ko):** 존댓말, technical terms often in English within parentheses.

- [ ] **Step 3: Em-dash sweep for each file**

```bash
for f in README.zh.md README.ja.md README.ko.md; do
  echo "$f:"; grep -n "—\|–" "$f" || echo "  clean"
done
```

Expected: all three clean.

- [ ] **Step 4: Section-parity check**

```bash
node -e "
const fs=require('fs');
const en=fs.readFileSync('README.md','utf8').match(/^##\s.*$/gm)||[];
for (const f of ['README.zh.md','README.ja.md','README.ko.md']) {
  const lang=fs.readFileSync(f,'utf8').match(/^##\s.*$/gm)||[];
  console.log(f+': '+lang.length+' H2 headings (EN has '+en.length+')');
}
"
```

Expected: each translation has the same H2 count as the English canonical.

- [ ] **Step 5: Commit**

```bash
git add README.zh.md README.ja.md README.ko.md
git commit -m "docs(readme): sync zh / ja / ko translations to the refreshed structure

Mirrors README.md: 5-CLI hero, screenshots gallery, feature matrix,
2 comparison tables, personas, 11-question FAQ, moved status block.

Code blocks and command examples kept verbatim in English (matches
existing translation policy in these files). Product names in
comparison tables (Claude Desktop, Cursor, Open WebUI, etc.) kept
verbatim, per convention for named products."
```

---

### Task 4b: Translate to Romance (es, pt-BR, fr)

**Files:**
- Modify: `README.es.md`, `README.pt-BR.md`, `README.fr.md`

**Interfaces:**
- Consumes: new English README from Task 2
- Produces: three translated files with identical structure

- [ ] **Step 1: Read the current translation to learn each file's style**

```bash
head -200 README.es.md
head -200 README.pt-BR.md
head -200 README.fr.md
```

Note tone (formal vs informal), diacritic usage, and technical-term policy (usually keep English for CLI names).

- [ ] **Step 2: For each of es, pt-BR, fr: mirror the new English structure**

Follow the same procedure as Task 4a Step 2. Style expectations:

- **Spanish (es):** neutral (no vos/vosotros regional slant), technical terms in English.
- **Portuguese (pt-BR):** Brazilian conventions.
- **French (fr):** vouvoiement, technical terms in English with occasional French gloss.

- [ ] **Step 3: Em-dash sweep**

```bash
for f in README.es.md README.pt-BR.md README.fr.md; do
  echo "$f:"; grep -n "—\|–" "$f" || echo "  clean"
done
```

Expected: all three clean. Note that Romance languages sometimes tempt em-dashes in dialogue-style prose. Use commas or parens instead.

- [ ] **Step 4: Section-parity check**

```bash
node -e "
const fs=require('fs');
const en=fs.readFileSync('README.md','utf8').match(/^##\s.*$/gm)||[];
for (const f of ['README.es.md','README.pt-BR.md','README.fr.md']) {
  const lang=fs.readFileSync(f,'utf8').match(/^##\s.*$/gm)||[];
  console.log(f+': '+lang.length+' H2 headings (EN has '+en.length+')');
}
"
```

Expected: all three match the English H2 count.

- [ ] **Step 5: Commit**

```bash
git add README.es.md README.pt-BR.md README.fr.md
git commit -m "docs(readme): sync es / pt-BR / fr translations to the refreshed structure

Mirrors README.md: 5-CLI hero, screenshots gallery, feature matrix,
2 comparison tables, personas, 11-question FAQ, moved status block."
```

---

### Task 4c: Translate to remaining (de, ru, fa)

**Files:**
- Modify: `README.de.md`, `README.ru.md`, `README.fa.md`

**Interfaces:**
- Consumes: new English README from Task 2
- Produces: three translated files with identical structure

- [ ] **Step 1: Read the current translation to learn each file's style**

```bash
head -200 README.de.md
head -200 README.ru.md
head -200 README.fa.md
```

Note: `README.fa.md` is right-to-left. Preserve `<p align="center">` and `<img>` wrappers (they align in the Markdown-rendered flow); Farsi body prose reads RTL naturally in GitHub's renderer without extra markup.

- [ ] **Step 2: For each of de, ru, fa: mirror the new English structure**

Follow the same procedure. Style expectations:

- **German (de):** formal Sie-form, compound nouns typical in tech writing. Technical terms usually kept in English (Session, Memory).
- **Russian (ru):** formal вы-form. Technical terms often kept in English with a Russian gloss on first mention.
- **Farsi (fa):** RTL. Technical terms often kept in English. Do not add `<div dir="rtl">` wrappers (existing file does not use them and GitHub renders correctly without).

- [ ] **Step 3: Em-dash sweep**

```bash
for f in README.de.md README.ru.md README.fa.md; do
  echo "$f:"; grep -n "—\|–" "$f" || echo "  clean"
done
```

Expected: all three clean. German prose sometimes tempts em-dashes; use commas or parens instead. Russian and Farsi rarely use em-dashes in this style.

- [ ] **Step 4: Section-parity check**

```bash
node -e "
const fs=require('fs');
const en=fs.readFileSync('README.md','utf8').match(/^##\s.*$/gm)||[];
for (const f of ['README.de.md','README.ru.md','README.fa.md']) {
  const lang=fs.readFileSync(f,'utf8').match(/^##\s.*$/gm)||[];
  console.log(f+': '+lang.length+' H2 headings (EN has '+en.length+')');
}
"
```

Expected: all three match.

- [ ] **Step 5: Commit**

```bash
git add README.de.md README.ru.md README.fa.md
git commit -m "docs(readme): sync de / ru / fa translations to the refreshed structure

Mirrors README.md: 5-CLI hero, screenshots gallery, feature matrix,
2 comparison tables, personas, 11-question FAQ, moved status block.

Farsi (fa) preserves the existing RTL rendering (no dir wrappers
added; the file relies on GitHub's Markdown renderer to handle RTL
paragraphs)."
```

---

### Task 6 (reordered from 5): Update GitHub repo description + topics AFTER merge

**Files:**
- No local files. External state change on `Opendray/opendray` via `gh api`.

**Interfaces:**
- Consumes: nothing local
- Produces: updated `description` and `topics` on the GitHub repo, verifiable via `gh repo view`

- [ ] **Step 1: Update the repo description**

```bash
export PATH=$PATH:/var/lib/opendray/.local/bin
gh api -X PATCH repos/Opendray/opendray -f description="Self-hosted gateway for Claude Code, Codex, Antigravity, Grok Build, OpenCode. Run AI coding agents on your own infra with a shared local-first memory layer. Drive from web, mobile, Telegram / Slack / Discord / Feishu / DingTalk / WeCom. Open REST + WebSocket API. Apache 2.0."
```

Expected: 200 OK, JSON response echoing the new description.

- [ ] **Step 2: Fetch current topics**

```bash
gh api repos/Opendray/opendray/topics | jq -r '.names[]' | sort > /tmp/current-topics.txt
cat /tmp/current-topics.txt
```

Expected: current topic list. Note the presence of `gemini-cli`.

- [ ] **Step 3: Compute the new topic list**

```bash
{
  cat /tmp/current-topics.txt | grep -v '^gemini-cli$'
  echo grok
  echo opencode
  echo antigravity
  echo pgvector
  echo ai-gateway
  echo agent-runtime
  echo homelab-ai
  echo local-first
} | sort -u > /tmp/new-topics.txt
cat /tmp/new-topics.txt
```

Expected: same list minus `gemini-cli`, plus the 8 new topics, no duplicates.

- [ ] **Step 4: Apply the new topics**

```bash
NAMES=$(jq -R -s -c 'split("\n") | map(select(length > 0))' < /tmp/new-topics.txt)
gh api -X PUT repos/Opendray/opendray/topics -f "names=$NAMES" 2>&1 | tail -5
```

If the array form of `-f` fails, use `--input`:

```bash
jq -n --argjson names "$NAMES" '{names: $names}' | gh api -X PUT repos/Opendray/opendray/topics --input -
```

Expected: 200 OK, JSON echo of new topics list.

- [ ] **Step 5: Verify from a fresh fetch**

```bash
gh repo view Opendray/opendray --json description,repositoryTopics | jq
```

Expected: description matches the new string exactly. Topics list contains `grok`, `opencode`, `antigravity`, `pgvector`, `ai-gateway`, `agent-runtime`, `homelab-ai`, `local-first`, and does NOT contain `gemini-cli`.

- [ ] **Step 6: Note the change in the plan status**

No commit here (no local files touched). The change is auditable via GitHub's activity log.

---

### Task 5 (reordered from 6): Open PR, wait for CI, merge

**Files:**
- No local files. PR creation, CI wait, merge on green.

**Interfaces:**
- Consumes: all commits from Tasks 1-4 on branch `docs/readme-refresh`
- Produces: merged commit on `main`

- [ ] **Step 1: Push all commits**

```bash
cd /var/lib/opendray/rcc/opendray
export PATH=$PATH:/var/lib/opendray/.local/bin
git push origin docs/readme-refresh 2>&1 | tail -5
```

Expected: no error, branch updated on remote.

- [ ] **Step 2: Open the PR**

```bash
gh pr create \
  --base main \
  --head docs/readme-refresh \
  --title "docs(readme): refresh hero, add comparison, FAQ, screenshots, sync 5-CLI provider list" \
  --body "$(cat <<'EOF'
## Summary

Refresh `README.md` and the 9 translations to accurately represent the current 5-CLI shipping surface (Claude Code, Codex, Antigravity, Grok Build, OpenCode, plus shell), look comparable to top-tier OSS repos, and improve discoverability via both search engines and LLM answers.

## What changed

- **Hero rewritten.** New tagline names all 5 wrapped CLIs. Three markdown-badge action buttons (Get started, See screenshots, Live demo). Hero screenshot above the fold.
- **Screenshots gallery added.** 4 shots: dashboard, live session with memory rail, mobile viewport, chat channels page. Captured from a local instance.
- **Feature matrix added.** 7 category rows (Sessions, Providers, Memory, Channels, Integrations, Ops, Security) with concrete keyword-dense bullets.
- **Two comparison tables added.**
  - opendray vs known AI clients (Claude Desktop, Cursor, CLI over SSH, ChatGPT Desktop)
  - opendray vs self-hosted chat frontends (Open WebUI, LibreChat, Dify)
- **Personas section added.** Solo dev / homelab, small team lead, integrator.
- **11-question FAQ added.** GEO-shaped questions covering positioning ("vs Claude Desktop", "vs Open WebUI"), practical concerns ("Docker?", "Raspberry Pi?"), and contribution paths.
- **Structural reorder.** Status/version block moved from top to just before Tests (removes the stale `v2.7.6` reference; new block says `current generation: v2.10.x, see CHANGELOG`). Architecture Mermaid diagram updated to name all 5 CLIs.
- **`llms.txt` added at repo root.** Canonical LLM-friendly summary for AI-answer discoverability.
- **9 translations synced.** All 9 language files mirror the new structure.
- **GitHub repo description + topics updated.** Description names the 5 CLIs verbatim. Topic list drops `gemini-cli` (retired), adds `grok`, `opencode`, `antigravity`, `pgvector`, `ai-gateway`, `agent-runtime`, `homelab-ai`, `local-first`.

## Design + rationale

Design doc: [`docs/superpowers/specs/2026-07-02-readme-refresh-design.md`](docs/superpowers/specs/2026-07-02-readme-refresh-design.md)
Plan: [`docs/superpowers/plans/2026-07-02-readme-refresh.md`](docs/superpowers/plans/2026-07-02-readme-refresh.md)

## Test plan

- [ ] `check-i18n-parity` CI job green
- [ ] `check-readme-drift` advisory shows all 9 translations updated (clean)
- [ ] All 4 screenshots load on GitHub's rendered README preview
- [ ] `llms.txt` renders as raw text at `https://raw.githubusercontent.com/Opendray/opendray/main/llms.txt` after merge
- [ ] Repo description on `github.com/Opendray/opendray` matches the new string
- [ ] Repo topics on `github.com/Opendray/opendray` include `grok` / `opencode` / `antigravity`, do NOT include `gemini-cli`
EOF
)"
```

Expected: PR URL printed. Note the PR number.

- [ ] **Step 3: Wait for CI to complete**

Set `PR=<the number from step 2>` then:

```bash
until state=$(gh pr view $PR --json mergeStateStatus,statusCheckRollup 2>/dev/null); \
  pending=$(echo "$state" | grep -oE '"status":"IN_PROGRESS"|"status":"QUEUED"' | wc -l); \
  [ "$pending" = "0" ]; do sleep 30; done
echo "CI done."
echo "$state" | jq '.statusCheckRollup | map({name, conclusion})'
```

Expected: every check has `"conclusion":"SUCCESS"`. If any check is `FAILURE` or `CANCELLED`, stop and report.

- [ ] **Step 4: Merge on green**

```bash
gh pr merge $PR --squash --delete-branch
```

Expected: `✓ Squashed and merged pull request #<N>`. Branch deleted.

If the PR is `BEHIND` main, update first:

```bash
gh pr update-branch $PR
```

Then re-run Step 3 to wait for CI on the updated branch, then re-run Step 4.

- [ ] **Step 5: Verify the merge landed**

```bash
gh pr view $PR --json state,mergeCommit
```

Expected: `"state":"MERGED"`, non-null `mergeCommit.oid`.

- [ ] **Step 6: Sanity-check the live README + metadata**

Open `https://github.com/Opendray/opendray` in a browser (or `gh browse`). Confirm:

1. Hero mentions Grok Build and OpenCode.
2. Hero screenshot renders.
3. Comparison tables render.
4. FAQ section renders.
5. Repo description at the top of the page matches the new string.
6. Topics chips show `grok`, `opencode`, `antigravity`, and do not show `gemini-cli`.
7. `llms.txt` is fetchable at `https://raw.githubusercontent.com/Opendray/opendray/main/llms.txt`.

If any check fails, open a follow-up PR to fix. Do NOT amend the merged commit.

---

## Success Criteria

- README hero names all 5 wrapped CLIs (Claude Code, Codex, Antigravity, Grok Build, OpenCode).
- 4 screenshots render on the GitHub-rendered README.
- Comparison tables + FAQ + personas + feature matrix all render correctly.
- Status/version block no longer contains `v2.7.6`.
- `llms.txt` present at repo root, fetchable via raw.githubusercontent.
- All 9 translations updated (check-readme-drift advisory clean, i18n parity green).
- Repo description on GitHub matches the new string exactly.
- Repo topics include `grok`, `opencode`, `antigravity`, `pgvector`, `ai-gateway`, `agent-runtime`, `homelab-ai`, `local-first`; do NOT include `gemini-cli`.
- All CI checks green.
- Zero em-dashes across all changed files.
- PR merged into `main`.

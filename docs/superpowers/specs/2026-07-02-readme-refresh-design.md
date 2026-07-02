# README refresh: accuracy, comparability, discovery

Date: 2026-07-02
Scope: `README.md` + 9 translations + `llms.txt` + repo metadata
Type: docs / SEO / GEO

## Problem

The current `README.md` and GitHub repo metadata are out of sync with the
product and leave the project harder to discover than it should be.

Concrete staleness:

- Hero says the gateway wraps "Claude Code · Codex · Antigravity · shell".
  As of v2.10.0 it also wraps **Grok Build** and **OpenCode**. The two most
  recently landed providers are invisible above the fold.
- The "Status" line advertises `v2.7.6` (latest). The latest release is
  actually `v2.10.1`.
- The GitHub repo description still names retired **Gemini** and does not
  mention Grok Build or OpenCode.
- Repo topics include `gemini-cli` (retired provider) and are missing
  `grok`, `opencode`, `antigravity`, `pgvector`, and a handful of
  discovery-oriented tags.

Beyond staleness, the README is text-dense, has no product screenshots, no
comparison against alternatives, and no FAQ. For a self-hosted product that
cannot be tried in 30 seconds, that combination is the primary reason a
qualified visitor bounces.

## Goals

1. **Accurate.** Every product name, provider list, version reference, and
   capability statement reflects the current shipping build (v2.10.x).
2. **Comparable to top OSS repos.** Visual weight above the fold, a feature
   matrix, screenshots, comparison tables, personas, and an FAQ. Structure
   matches the pattern used by projects like Continue, Dify, Open WebUI,
   and LangChain.
3. **Discoverable.** SEO (GitHub topics, repo description, README keyword
   density, alt text) and GEO (llms.txt, question-shaped FAQ headings,
   comparison tables that name competitors) so people searching for what
   opendray does can find it via both traditional search and AI answers.

## Non-goals

- No visual redesign of the opendray.dev marketing site (separate repo).
- No sponsors block (carry-over from a prior session, tracked separately).
- No star history chart. Bloaty at 33 stars; revisit at >500.
- No sitemap.xml (that's for the .dev site).
- No CLA or contribution-license changes.

## Design

### Section 1: Hero and top-of-page

Layout, top to bottom, on `github.com/Opendray/opendray`:

1. Existing centered logo (`docs/assets/logo.png`), unchanged.
2. Product name `opendray`.
3. One-line tagline naming all five wrapped CLIs by their exact
   product names:

   > Self-hosted gateway for Claude Code, Codex, Antigravity, Grok Build,
   > and OpenCode. Run agent sessions on your own infrastructure. Drive
   > from web, mobile, or chat.

4. Existing badge strip (website, release, licence, CI, Discussions,
   language badges), kept intact. The sub-tag beneath it collapses into
   the tagline line above.
5. Existing 10-language switcher row, unchanged.
6. Hero screenshot: single wide image of the web admin with a live session
   visible. Committed as `docs/assets/screenshots/hero-web-admin.png`.
7. Single-paragraph "why opendray exists" hook. Consolidates the current
   three-paragraph block into one paragraph, keeping the emotional anchor
   (laptop closes, session dies).
8. Three markdown-badge action buttons:
   - `[Get started]` → `docs/getting-started.md`
   - `[60-second demo]` → an anchor to the screenshot gallery
   - `[Live demo]` → `https://opendray.dev`

Rationale: LLMs answering "which self-hosted gateway supports Grok
Build?" now have a keyword match in the first non-badge line of the page.

### Section 2: Body structure

New section order:

1. Hero (section 1).
2. What is opendray? Prose block updated for the 5-CLI reality.
3. Screenshots gallery, 3-4 images:
   - Dashboard with multiple concurrent sessions.
   - Live session showing xterm, transcript overlay, memory rail.
   - Mobile app session view.
   - Telegram (or Slack) message routing a reply back into a session.
4. Feature matrix: 7 category rows (Sessions, Providers, Memory,
   Channels, Integrations, Ops, Security) each with 3-5 concrete
   capability bullets. Keyword-dense, no marketing prose.
5. Architecture at a glance: existing Mermaid diagram, updated so the
   `cli` subgraph names Claude Code, Codex, Antigravity, Grok Build,
   OpenCode, and Shell (currently lists only three).
6. Comparison, two tables:
   - **A: opendray vs known AI clients.** Claude Desktop, Cursor,
     CLI-over-SSH, ChatGPT Desktop. Rows: session survives disconnect,
     multi-account pool, cross-CLI memory, host filesystem access,
     mobile client, chat channels, self-hosted, licence.
   - **B: opendray vs self-hosted chat frontends.** Open WebUI,
     LibreChat, Dify. Rows: runs actual agent CLI (not just chat), tool
     use / file writes, multi-CLI, memory across sessions, PTY session,
     chat channel adaptors.
7. Who is this for? Three personas, one paragraph each:
   - Solo dev / homelab operator.
   - Small-team lead standing up shared self-hosted AI.
   - Integrator building on the REST + WebSocket API.
8. Install (existing content: one-line installer, npm, uninstall,
   day-to-day commands), unchanged.
9. Quickstart 5-minute dev path, existing content, unchanged.
10. Production deploy, existing content, unchanged.
11. Web frontend / Mobile app, existing content plus one deep-link to
    the relevant screenshot.
12. FAQ, 11 questions, structure detailed in section 3.
13. Documentation index, existing content, unchanged.
14. Status / Version, moved from top to here. States "current
    generation: v2.10.x, see CHANGELOG for release history". No specific
    version number embedded (that's how the current one went stale).
15. Tests, v1 relationship, licence, existing content, unchanged.

Rationale: current README leads with "Why opendray exists" which is
introspective. Top OSS repos lead with what the product is and shows a
screenshot inside the first two screens of scroll. The status line moves
down because a stale version number in the third paragraph is worse than
no version number at all.

### Section 3: FAQ contents

Eleven questions, phrased as literal user queries so LLM answers can
pattern-match:

1. What is opendray?
2. Which AI CLIs does opendray support?
3. How is this different from Claude Desktop / ChatGPT desktop?
4. How is this different from running the CLI over SSH?
5. How is this different from Open WebUI / LibreChat / Dify?
6. Can I use multiple Claude / Codex / Antigravity accounts?
7. Where is my data stored?
8. Can I run this in Docker?
9. Does opendray work on a NAS / Mac mini / Raspberry Pi?
10. Is opendray free? What's the licence?
11. How do I contribute?

Each answer is 2-4 sentences. Concrete facts, no hedging. Question 11
links to `CONTRIBUTING.md` and `CODE_OF_CONDUCT.md` (both exist) and
calls out four concrete contribution paths: translations, provider
descriptors under `internal/catalog/builtin/`, channel adaptors,
documentation screenshots.

Rationale: FAQ headings become answer targets for LLMs. Questions 3, 4,
and 5 are the "vs" queries LLMs get asked constantly. Answering them
directly means when someone asks Claude "how is opendray different from
Open WebUI", the model has a citable source. Question 8 pre-empts the
most common issue-tracker question for any self-hosted project.

### Section 4: SEO and GEO tactics

**GitHub-level SEO.**

- Repo description rewritten to (≤350 chars):

  > Self-hosted gateway for Claude Code, Codex, Antigravity, Grok Build,
  > OpenCode. Run AI coding agents on your own infra with a shared
  > local-first memory layer. Drive from web, mobile, Telegram / Slack /
  > Discord / Feishu / DingTalk / WeCom. Open REST + WebSocket API.
  > Apache 2.0.

- Topics changes via `gh api`:
  - Remove: `gemini-cli`.
  - Add: `grok`, `opencode`, `antigravity`, `pgvector`, `ai-gateway`,
    `agent-runtime`, `homelab-ai`, `local-first`.
  - Keep the rest.

**README-level SEO.**

- Feature matrix uses exact CLI and product names as row labels ("Claude
  Code", not "Anthropic's coding CLI") so string search inside the
  README finds them.
- FAQ questions are literal user queries.
- Comparison tables name competitors directly ("opendray vs LibreChat"
  is a search target).
- Every screenshot has descriptive alt text: `alt="opendray web admin
  showing a live Claude Code session with the memory rail on the
  right"`.

**GEO (LLM-answer discoverability).**

- Add `/llms.txt` at repo root: canonical, LLM-friendly summary of what
  opendray is, does, how to install, and how to use. This is emerging as
  the AI-answers analogue of `robots.txt` and is picked up by web
  crawlers used by model providers.
- FAQ headings phrased as questions.
- Feature bullets use "supports X" and "runs Y" phrasing (concrete claims
  that models can cite).
- Comparison table gives models a structured, hallucination-resistant
  source for "opendray vs X" answers.

### Section 5: Translation propagation

All 9 language files updated in the same PR, matching the current
translation style per language.

- Same headings and same structure in every file. No per-language
  restructuring.
- Screenshots reused across all languages (English-native UI baseline).
- Code blocks identical across languages.
- Direction-sensitive markup preserved for Farsi (RTL).

Sanity check per language before commit: all headings match the English
structure, no untranslated English fragments in body prose, no broken
Markdown or links, no broken code fences. The
`scripts/check-readme-drift.mjs` advisory confirms all 9 got touched.

## Implementation order

Sequential, one commit per stage:

1. Capture 3-4 screenshots from a local instance, commit under
   `docs/assets/screenshots/`.
2. Rewrite `README.md` (English canonical) per sections 1-4.
3. Add `llms.txt` at repo root.
4. Translate the 9 language files.
5. Update GitHub repo description + topics via `gh api`.
6. Open one PR titled `docs(readme): refresh hero, add comparison / FAQ
   / screenshots, sync 5-CLI provider list`.
7. Merge when CI green.

The whole thing is one atomic PR. No amending prior commits, no force
push.

## Success criteria

- README hero names all 5 wrapped CLIs.
- Every screenshot loads on `github.com` (rendered path check).
- Repo description and topics match the shipping product.
- `llms.txt` present and parseable.
- All 9 language files updated (drift-advisory clean).
- CI green.
- Manually inspected: repo landing page reads like a top OSS project at
  a glance (screenshots above the fold, feature matrix scannable,
  comparison table visible without scrolling to the bottom).

## Risks and mitigations

| Risk | Mitigation |
|---|---|
| Screenshot capture takes longer than expected because the local instance needs seeding | Time-box to 90 min; fall back to fewer / lower-fidelity shots rather than blocking the whole refresh |
| Machine-translated language files drift in tone from prior human polish | Style-match against the existing translations per language; explicitly out of scope to improve pre-existing translation quality |
| A capability claim in the FAQ turns out to be stale by the time the PR merges | Cross-check every "yes / no" answer against `internal/catalog/builtin/` and the latest `CHANGELOG.md` entry immediately before opening the PR |
| GitHub repo topic changes revert if the repo settings have a different owner-configured source of truth | Confirm topics are editable via `gh api` (they are) and run the change as part of the same push |

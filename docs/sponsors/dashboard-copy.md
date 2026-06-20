# Sponsor dashboard copy

This file holds the text to paste into the two GitHub Sponsors dashboards:

- Org: <https://github.com/sponsors/Opendray/dashboard>
- Personal: <https://github.com/sponsors/navidrast/dashboard>

Each section maps to a specific field in the dashboard UI. Copy the block, paste into the field, save. Update this file when copy changes so both dashboards stay in sync.

---

## A. Opendray org account

### Short bio (160 chars max)

> Self-hosted gateway for AI coding agents (Claude Code, Codex, Antigravity, shell) with one local-first memory layer. Apache-2.0. opendray.dev

### Detailed bio (markdown, profile page)

```markdown
**opendray** wraps the AI coding CLIs you already use (Claude Code, Codex, Antigravity, plus any shell) and turns them into something you can drive from anywhere. Sessions live on your own infra (NAS, Mac mini, VPS), survive your laptop sleeping, and reattach from a web admin, a Flutter mobile app, or a Telegram / Slack / Discord / Feishu / DingTalk / WeCom message.

Built in the open under Apache-2.0. One static Go binary, cosign-signed releases, SPDX SBOM, no telemetry, no cloud account, no subscription.

Sponsorship pays for the time that keeps it that way: roadmap work, the release pipeline, all 10 README translations, the channel adaptors, the local-first memory layer (ONNX / Ollama / LM Studio embeddings), and the answers under your "how do I" issues.

See [`SPONSORS.md`](https://github.com/Opendray/opendray/blob/main/SPONSORS.md) on the repo for tiers and what each one buys.
```

### Header / cover image

Use `docs/assets/social-preview.png` from the repo. 1280×640, fits the dashboard's cover slot.

### Goal

**Title**: Keep opendray full-time

**Amount**: $4,000 / month

**Description**:
```
$4k/month covers one full-time maintainer at sustainable open-source rates in our region. It funds: 5 release trains a quarter, all 10 README languages kept in sync, weekly issue triage, the integrations API, and one new channel adaptor per quarter. Above $4k, sponsorships fund a second maintainer.
```

### Tiers

Add each tier below in the dashboard. Set "single sponsorship" off for all (recurring only) unless noted.

---

**Tier name**: Backer
**Amount**: $5/month
**Welcome message**:
```
Welcome to opendray. Your name is on the Backers wall in SPONSORS.md and at opendray.dev/sponsors. If you opted out of attribution, no problem, the thank-you stands either way.
```
**Public description**:
```
Name on the Backers wall in SPONSORS.md and at opendray.dev/sponsors. Thank-you in the next release notes. Helps keep the lights on.
```

---

**Tier name**: Hobbyist
**Amount**: $25/month
**Welcome message**:
```
Welcome! You now have access to release-candidate builds (linked in the private sponsor Discussions thread) and you can post questions to me directly there. Reply to this welcome with any first question and I'll get back within ~24h.
```
**Public description**:
```
Everything in Backer, plus early access to release-candidate builds and a private "ask the maintainer" thread in Discussions. Good if you run opendray at home and want a faster line to me.
```

---

**Tier name**: Pro
**Amount**: $100/month
**Welcome message**:
```
Thanks for backing opendray. Your name or handle goes into the README sponsor footer (small). Reply with the exact text you'd like shown, and a link if you want one. You also get a shout-out in the next social post.
```
**Public description**:
```
Everything in Hobbyist, plus your name or handle in the README sponsor footer (small) and a thank-you in social channels. For independent devs and small consultancies who run opendray professionally.
```

---

**Tier name**: Team
**Amount**: $500/month
**Welcome message**:
```
Welcome! Three onboarding steps:
1. Send a logo (SVG preferred, PNG 400×100 fallback) for the README sponsor section and opendray.dev.
2. Pick a recurring 30-min office-hours slot, monthly. Reply with your timezone and I'll send options.
3. Tag any issue you open with the `sponsor:team` label and I'll triage it within the day.
```
**Public description**:
```
Everything in Pro, plus your logo in the README sponsor section (medium), a monthly office-hours call (30 min), and priority triage on issues you open. For teams running opendray in production.
```

---

**Tier name**: Founding
**Amount**: $2,500/month
**Welcome message**:
```
Thank you, this matters a lot. Reply to schedule the first roadmap review (60 min) and we'll pick the first quarterly feature / integration to schedule. Logo placement at the top of the README and on opendray.dev front page goes live within the week.
```
**Public description**:
```
Everything in Team, plus top-of-README placement (large), one feature or integration per quarter scheduled with you, and a quarterly roadmap review call. Limited to a small number of slots so the maintainer can actually deliver.
```

---

### Newsletter draft (first sponsor update)

```markdown
**Subject**: Welcome to the opendray sponsor list

You're the reason this stays a full-time project. Quick orientation:

- **Release cadence**: roughly one minor every 2 weeks, one patch a week. Watch the repo's Releases page or `CHANGELOG.md`.
- **Roadmap**: tracked in the project's `project_plan` and surfaced in Discussions → Announcements.
- **Direct line**: Team and Founding tiers get a private thread, Hobbyist+ get the private Discussions thread, all tiers can email `navid@opendray.dev`.
- **What's next this quarter**: voice channels (TTS + STT) so you can talk to a session from Telegram / Slack, and the v3 generation R&D kickoff.

Thanks for being early. Reply with what you're using opendray for, I read every one.

— navid, opendray
```

---

## B. Personal account (navidrast)

Most of the org copy carries over. The differences:

### Short bio (160 chars max)

> I maintain opendray, a self-hosted gateway for AI coding agents (Claude Code, Codex, Antigravity, shell) with a local-first memory layer. Apache-2.0.

### Detailed bio (markdown)

```markdown
I'm Navid. I build [opendray](https://github.com/Opendray/opendray), an Apache-2.0 self-hosted gateway that wraps the AI coding CLIs you already use (Claude Code, Codex, Antigravity, plus shell) and lets you drive sessions from web, mobile, or chat (Telegram / Slack / Discord / Feishu / DingTalk / WeCom).

Sponsorship at this personal account is the fallback when an employer's procurement can only pay individuals, not organizations. If your company can sponsor an organization, please use [the project account](https://github.com/sponsors/Opendray) instead.

Either way, the money funds the same thing: time spent on opendray (roadmap, releases, docs in 10 languages, support).
```

### Goal

Same as org account, $4,000/month. Keep them synced so total funding rolls up cleanly.

### Tiers

Use the same five tiers as the org account, same prices, same descriptions. The only line to swap is in the Backer welcome message: replace "opendray.dev/sponsors" with "your name on the Backers wall under github.com/sponsors/navidrast/sponsors".

---

## Field-by-field upload checklist

When you sit at the dashboard for the first time, walk this in order:

1. **Profile** → Short bio → paste section A.1 or B.1
2. **Profile** → Detailed bio → paste section A.2 or B.2
3. **Profile** → Header image → upload `docs/assets/social-preview.png`
4. **Goals** → Add goal → paste from section A.4
5. **Tiers** → Add tier (×5) → paste each from section A.5
6. **Newsletter** → Draft → paste section A.6 (org only, not personal)
7. **Settings** → Payouts → connect Stripe (you only)
8. **Settings** → Matching → check the box if you want GitHub's matching program (free)

After save, the public page is live at `github.com/sponsors/Opendray` and the Sponsor button on the repo starts working.

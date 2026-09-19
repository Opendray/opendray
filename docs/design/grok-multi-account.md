# Grok Multi-Account (pool + live switch)

Status: **Draft — RFC for review (no implementation yet)**
Owner: opendray gateway
Last updated: 2026-08-31

Bring grok to parity with Claude and Antigravity: a pool of grok accounts,
pick one per session, and switch a live session from one account to another
without dropping the session. Grok is currently the only agent provider with
**no** account support — every grok session shares the single `~/.grok`
login.

---

## 1. What exists today

Account support is built per-provider; two providers have it, grok has none:

- **Claude — `cliacct`.** Isolates accounts via `CLAUDE_CONFIG_DIR`; injects
  `CLAUDE_CODE_OAUTH_TOKEN` + `CLAUDE_CONFIG_DIR` at spawn. Live switch:
  `PATCH /sessions/{id}/claude-account` (+ carry-context — the prior
  transcript is injected into the fresh session).
- **Antigravity — `agyacct`.** `agy` keys its entire state off `$HOME`, so an
  account is a dedicated HOME dir holding its own OAuth token
  (`<HOME>/.gemini/antigravity-cli/antigravity-oauth-token`). The adapter
  sets `out.Env["HOME"] = <account home>`; live switch:
  `PATCH /sessions/{id}/antigravity-account`. opendray discovers account
  dirs and points spawns at them — it never mints tokens.
- **Grok — nothing.** No pool, no switch endpoint, no isolation. All grok
  sessions read `~/.grok` (whatever `grok login` last wrote).

## 2. Grok maps onto the Antigravity blueprint — more cleanly

Grok, like `agy`, keys its state off a home directory. But grok has its own
`GROK_HOME` env var (verified: it relocates grok's home — `auth.json`,
`config.toml`, `trusted_folders.toml`, sessions), so opendray can relocate
**only grok's state**, never the whole `HOME`. That is strictly better than
the Antigravity model, which has to move the entire HOME and carries the
"don't write integration MCP into a shared HOME" caveats.

A grok account is a dedicated `GROK_HOME` directory containing:

| File                          | Role                                             |
|-------------------------------|--------------------------------------------------|
| `auth.json` (0600)            | the xAI login token — the account identity       |
| `config.toml`                 | per-account grok config                          |
| `trusted_folders.toml`        | repo-local MCP trust (grok reads this from home) |
| `agent_id`, `sessions/`, …    | per-account runtime state                        |

Heavy, account-independent dirs (`bin/` ~317 MB, `bundled/`, `vendor/`,
`downloads/`, `installed-plugins/`, `marketplace-cache/`) are **symlinked to
a shared source**, not copied — the exact pattern `ensureAgySharedCache`
already uses for Antigravity's Playwright cache — so N accounts don't cost N×
hundreds of MB.

## 3. Components (mirror agyacct)

- **C1 — `grokacct` service + store + handler.** Account rows hold metadata
  only (id, name, display name, `GROK_HOME` dir, token-present). CRUD under
  `/api/v1/grok-accounts`. Discovers account homes; never mints tokens.
- **C2 — spawn binding.** In the adapter: when `providerID=="grok"` and an
  account is selected, `out.Env["GROK_HOME"] = ResolveSpawnHome(accountID)`.
  Symlink shared assets into the home if missing (C4).
- **C3 — live switch.** `SwitchGrokAccount(ctx, id, accountID)` +
  `PATCH /sessions/{id}/grok-account`: validate target, stop the PTY, rebind
  `grok_account_id`, respawn under the new `GROK_HOME`. Mirrors
  `SwitchAntigravityAccount`.
- **C4 — shared-asset symlinker.** `ensureGrokSharedAssets(home)` — symlink
  `bin/`, `bundled/`, `vendor/`, caches to a shared source; best-effort,
  logs + continues on failure. Modeled on `ensureAgySharedCache`.
- **C5 — MCP trust per home.** `renderGrokMCP` currently writes trust to
  `~/.grok/trusted_folders.toml`; when account-bound it must target
  `<GROK_HOME>/trusted_folders.toml` so repo-local MCP stays trusted under
  the account's home.
- **C6 — token-present detection.** `<GROK_HOME>/auth.json` exists + non-empty
  → account is "logged in" (like agyacct's `TokenFilled`). Surface it in the
  account row so the UI shows which accounts need `grok login`.

## 4. Login flow (out-of-band, same as agy)

opendray never handles xAI credentials. The operator logs each account in
once, under its home:

```sh
GROK_HOME=<account home> grok login    # sign in with the xAI account
```

opendray then discovers `auth.json` and points spawns at that home.

## 5. Carry-context on switch

Claude's switch injects the prior transcript so the conversation continues;
Antigravity's does not. For grok, carry-context is achievable via the grok
system-prompt surface (`--rules` / `<GROK_HOME>/AGENTS.md`, the same surface
the skills/global-instruction work uses) — inject a recap of the prior
conversation into the fresh account's spawn. **Recommend deferring** to keep
v1 aligned with the Antigravity model (rebind + respawn); add carry-context
as a fast-follow.

## 6. Fundamentals compliance

| Ref | Fundamental          | Verdict                                                  |
|-----|----------------------|----------------------------------------------------------|
| F1  | Cross-CLI parity     | **Reinforces** — grok reaches the account parity Claude/agy already have |
| F3  | Self-hosted          | Neutral — all on-disk, no external service               |
| F5  | Multi-user isolation | Per-account `GROK_HOME`; account rows scoped by principal |
| F6  | Secrets never leak   | `auth.json` stays 0600 on disk; opendray never mints or transports tokens |

No new coupling — this follows an existing, proven pattern (`agyacct`).

## 7. Phasing

1. **P1 — Pool + bind at create.** `grokacct` CRUD + discovery + token-present;
   adapter sets `GROK_HOME` when a session is created against an account.
   Ships the multi-account pool (pick account when spawning).
2. **P2 — Live switch.** `SwitchGrokAccount` + `PATCH /grok-account` + web/mobile
   switch UI. This is the "switch without losing the session" ask.
3. **P3 — Polish.** Shared-asset symlinks (C4), MCP-trust-per-home (C5),
   optional carry-context (§5).

## 8. Open decisions

- **D1 — Relocation lever.** `GROK_HOME` (grok-scoped, recommended) vs full
  `HOME` like agy. **Recommend `GROK_HOME`** — it isolates only grok, avoiding
  agy's shared-HOME hazards.
- **D2 — Shared heavy dirs.** Symlink to a shared source (recommended, ~saves
  hundreds of MB/account) vs full per-account copy. **Recommend symlink**,
  reusing the `ensureAgySharedCache` pattern.
- **D3 — Carry-context in v1?** **Recommend defer** (match agy); add later via
  the grok `--rules`/AGENTS.md surface.

## 9. Review asks (linivek)

1. Is the `grokacct`-mirrors-`agyacct` shape right, using `GROK_HOME` instead
   of full-HOME relocation?
2. Ship P1 (pool + bind-at-create) before P2 (live switch), or land them
   together since the switch is the headline ask?
3. Shared-asset symlinking (D2) — acceptable, or must each account be fully
   self-contained on disk?

## 10. Non-goals

- Minting or refreshing xAI tokens — login stays out-of-band (`grok login`).
- Changing Claude/Antigravity account behavior — this only adds grok.

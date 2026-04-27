# Plugin-Driven Architecture Refactor

**Status:** In progress · Phase 0 landed · Phase 1 next  
**Branch:** `refactor/plugin-driven`  
**Owner:** Linivek  
**Last updated:** 2026-04-27

This document is the single source of truth for the multi-phase refactor
that turns OpenDray from a UI-only plugin host into a fully
capability-driven plugin system. Update this file with every phase PR.

---

## 1. Problem statement

OpenDray today ships a complete plugin lifecycle (manifest validation,
install/uninstall, signing, consent, bridge protocol, three runtime
forms — declarative / webview / host). However the **contribution
surface is UI-only** (commands, status bar, keybindings, menus,
activity bar, views, panels, editor/session actions). All
*capability-flavoured* features — LLM providers, source-control forges,
MCP servers, messaging channels, account/credential stores, DB browser
— are hardcoded in `gateway/` and have no equivalent registration
pathway for third-party plugins.

The bridge API surface confirms the problem: `plugin/bridge/api_llm.go`
has `llm.invoke` going **plugin → host** (the plugin asks the host to
call the LLM). This means *only the host knows what providers exist* —
plugins cannot supply new ones. That is the precise technical
definition of "pseudo-plugin": insiders have a privileged path that
outsiders cannot replicate.

## 2. Goals (the Definition of Done)

The refactor is complete when **all** of the following are true:

1. **No vendor names in `gateway/`** — `claude_accounts.go`,
   `llm_providers.go`, `telegram.go`, `mcp/`, `sourcecontrol*/`,
   `pg_handlers.go`, `forge/` no longer exist as such. `gateway/`
   contains only the router, middleware, plugin runtime, workbench,
   and marketplace.
2. **Builtin and external plugins use the same registration surface** —
   in-process Go calls vs JSON-RPC bridge are transports; both end up
   in the same registry; consumers cannot tell them apart.
3. **Manifests describe capabilities** — every capability the host
   exposes (`provider`, `channel`, `forge`, `mcpServer`, …) can be
   declared statically in `contributes.*`, allowing the host to know
   what a plugin owns without loading code.
4. **Host calls plugin capabilities through the registry** — bridge
   protocol gains host→plugin direction so externals can supply
   providers, channels, etc., not just consume host services.
5. **External plugins can supply at least one core capability** —
   end-to-end demo: install a third-party host plugin that registers
   a new LLM provider, list/use it through unmodified gateway code.

## 3. Locked architectural decisions

| # | Decision | Choice | Rationale |
|---|----------|--------|-----------|
| 1 | Builtin runtime form | **In-process Go** | Same as OpenClaw bundled. Subprocess for 15–20 builtins is unacceptable startup/memory overhead; in-process collapses to function calls. |
| 2 | Phase-1 capability set | **Provider, Channel, Forge, McpServer, Hook** | Smallest surface that proves the contract. Phase 3+ adds Tool, Service, HttpRoute, Auth as needed. |
| 3 | Manifest schema strategy | **Extend `ContributesV1`** with capability fields | Strict whitelist already exists; adding documented fields is backwards-compatible. Avoids the cost of a parallel V2 codepath. |
| 4 | Migration strategy | **Big bang per vendor** | No external API stability requirement at v0.3.x. One PR per vendor: build new path, switch, delete old. |
| 5 | Frontend impact during migration | **Keep all `/api/*` endpoints stable** | Endpoints stay; backend swaps from hardcoded module to `registry.Get*` lookup. Flutter zero-change until Phase 5. |

If any of these need to change, edit this table and call out the
flip-back cost in the change PR.

## 4. Target architecture

```
┌──────────────────────────────────────────────────────────────┐
│  Flutter App  (workbench/contributions, bridge ws)            │
└──────────────────────────────────────────────────────────────┘
                            ▲
                    HTTP / WS │
┌───────────────────────────┴──────────────────────────────────┐
│  gateway/   (router, middleware, plugin runtime, workbench)   │
│  Reads from registry; never imports vendor packages.          │
└───────────────────────────┬──────────────────────────────────┘
                            │  reads/writes via
                            ▼
┌──────────────────────────────────────────────────────────────┐
│  plugin/api/   ── Stable Go contract (PluginAPI interface)   │
│   RegisterProvider / Channel / Forge / McpServer / Hook       │
└──────┬───────────────────────────────────────┬───────────────┘
       │                                       │
   in-process                              bridge JSON-RPC
       │                                       │
       ▼                                       ▼
┌─────────────────────────┐          ┌────────────────────────┐
│ plugins/builtin/<x>/    │          │ ~/.opendray/plugins/<y>/│
│   manifest.json         │          │   manifest.json         │
│   register.go (Go pkg)  │          │   host-darwin-arm64     │
└────────────┬────────────┘          └────────────┬───────────┘
             │                                    │
             └────────────► plugin/capreg ◄───────┘
                          (single registry)
```

## 5. Per-module migration plan

| Phase | Status | Module | Target | Notes |
|-------|--------|--------|--------|-------|
| 0 | ✅ landed | `plugin/api/`, `plugin/capreg/`, `ContributesV1` extension | New SSOT contract | Pure additions; no behaviour change |
| 1 | ⏳ next | `gateway/llm_providers.go`, `gateway/llm_proxy/` (vendor parts) | `plugins/builtin/llm-anthropic/`, `llm-openai/`, `llm-gemini/` + `gateway/llm_proxy/router.go` reads from registry | Vertical slice. Validates contract + bridge bidirectional. |
| 2 | — | bridge `host → plugin` direction | Same protocol envelope, server-initiated | Sample external host plugin providing a mock provider |
| 3 | — | `gateway/sourcecontrol/`, `sourcecontrol_*.go` | `plugins/builtin/forge-github/`, `forge-gitea/`, `forge-gitlab/` | Webhook routing reuses Phase 2 |
| 3 | — | `gateway/mcp/` | `plugins/builtin/mcp-runner/` | `supportsMcp` capability flag preserved |
| 3 | — | `gateway/telegram/`, `telegram.go` | `plugins/builtin/telegram/register.go` (manifest already exists) | Hookbus calls move to `api.Hooks().Subscribe(...)` |
| 3 | — | `gateway/pg_handlers.go`, `pg/` | `plugins/builtin/pg-browser/register.go` (manifest exists) | DB schema queries via `api.DB()` |
| 3 | — | `gateway/forge/` | merged into forge plugins | — |
| 3 | — | `gateway/claude_accounts.go` | `plugins/builtin/llm-anthropic/accounts.go` | Belongs to anthropic plugin; table stays in `kernel/store` |
| 4 | — | `plugin/hooks.go` expansion | More hook events + return semantics | Adds `before_tool_call`, `before_agent_reply`, etc. |
| 4 | — | Manifest `activation` field | Lazy-load support | Marketplace can list capabilities without loading code |
| 5 | — | `plugin/sdk/` extraction | Separate Go module for plugin authors | Public API for third parties |
| 5 | — | Marketplace capability filter | `?capability=provider` etc. | Backend consumes capreg listings |
| 5 | — | Compat shim cleanup | Remove legacy v0 paths | After all builtins migrated |

## 6. Phase 0 deliverables (landed)

Pure additions; no existing code path modified.

```
plugin/
  api/
    version.go           — APIVersion constant ("1.0.0")
    api.go               — PluginAPI interface + PluginInfo
    errors.go            — sentinel errors (ErrPluginUnloaded, ...)
    provider.go          — Provider interface + request/response types
    channel.go           — Channel interface + ChannelMessage
    forge.go             — Forge interface + Repository/PullRequest
    mcp.go               — McpServer interface + Tool/Result
    hook.go              — HookBus + event names
  capreg/
    registry.go          — capability registry (Set/Get/Remove/List)
    registry_test.go     — full coverage incl. concurrency
  manifest_capabilities.go         — ProviderContributionV1 etc.
  manifest_capabilities_test.go    — JSON round-trip + strict accept
  manifest.go            — ContributesV1 +4 fields (M6 block)
  manifest_strict.go     — whitelist +4 keys
docs/architecture/plugin-driven-refactor.md  — this file
```

**Verification:**
- `go build ./plugin/...` green
- `go vet ./plugin/...` clean
- `go test ./plugin/api/... ./plugin/capreg/...` pass
- `go test ./plugin -run TestContributesV1\|TestStrict` pass
- Pre-existing `plugin/bridge` macOS-symlink failures unchanged (not our regression)

## 7. Risks & rollback

| Risk | Impact | Mitigation |
|------|--------|------------|
| Phase-1 contract turns out wrong | Days–week of rework | Phase 0/1 only used internally. Public API stamped only after Phase 2 demonstrates external-plugin viability. |
| Removing `gateway/<vendor>.go` masks a hidden coupling | Broken endpoint in prod | One PR per vendor; manual smoke of corresponding endpoints before merge. Revertable via `git revert`. |
| Frontend depends on hardcoded route | Flutter feature breaks | Endpoints (`/api/llm-providers`, `/api/sourcecontrol/...`) are kept stable; only their backend handlers change. |
| `plugin_registry` / `claude_accounts` / `llm_providers` DB tables: ownership unclear | Confused uninstall semantics | Tables stay in `kernel/store`; plugins access via scoped `api.DB()` helper. Uninstalling a plugin does NOT delete user data (matches OpenClaw). |
| Builtin in-process registrar conflicts with existing `plugin/runtime.go` install flow | Boot break | Phase 1 introduces a discriminator: `plugin_registry.source = "builtin"` rows skip install/consent flow but still appear in lists. |

## 8. Inspirations / prior art

- [OpenClaw plugin architecture](https://docs.openclaw.ai/plugins/architecture.md)
- [OpenClaw plugin manifest](https://docs.openclaw.ai/plugins/manifest.md)
- [OpenClaw SDK overview](https://docs.openclaw.ai/plugins/sdk-overview.md)
- [OpenClaw hooks](https://docs.openclaw.ai/plugins/hooks.md)

The `plugin/api/hook.go` event names (`before_tool_call`,
`before_agent_reply`, etc.) are deliberately aligned with OpenClaw's
hook taxonomy so the two systems converge on shared vocabulary.

## 9. Working agreement

- One phase, one branch direction. PRs land into `refactor/plugin-driven`.
- After Phase 5, this branch is rebased on `main` and merged.
- This document is updated by every phase PR — the migration table
  status column especially.
- Any decision change goes through "edit this doc with rationale + flip
  cost" before code.

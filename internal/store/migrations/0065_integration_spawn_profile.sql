-- 0065 — per-integration spawn profile (MCP servers + system prompt +
-- bypass-permissions), the provider-AGNOSTIC half of integration spawn
-- config that complements 0064's provider/model/account defaults.
--
-- 0064 let an operator pin WHICH agent (provider + model + account) an
-- integration's sessions spawn against. But a third-party app usually
-- also needs the SAME two things injected into every session it spawns,
-- regardless of which CLI ends up running:
--
--   • its own MCP tool servers (so the agent can actually do the app's
--     work — e.g. PDA's invoicing tools), and
--   • a boot system prompt (the app's operating contract — e.g. "you are
--     the secretary, reply only via reply_to_user").
--
-- Until now the consumer had to hand-build per-CLI args (claude's
-- --mcp-config + --append-system-prompt + --dangerously-skip-permissions)
-- into the POST /sessions `args`, which hard-locks the integration to one
-- provider and breaks the moment the operator picks a different agent in
-- 0064's default-agent config.
--
-- These columns move that intent onto the integration, provider-agnostic.
-- opendray translates each at spawn time through its EXISTING per-provider
-- machinery: MCP via renderMCP (claude --mcp-config / gemini
-- .gemini/settings.json / codex|antigravity config), the prompt via the
-- same per-provider system-prompt surface used for ambient memory
-- (claude --append-system-prompt / gemini GEMINI.md / codex|antigravity
-- AGENTS.md), and bypass via each manifest's bypassPermissions/yolo flag.
--
--   mcp_servers:         '[]' (none) | JSON array of {name,command,args,env,url,transport}
--   system_prompt:       '' (none) | markdown injected into every spawned session
--   bypass_permissions:  false | true → auto-approve (the app drives the
--                        session unattended; there is no human at the TUI)
--
-- Applied ONLY to sessions an integration creates (origin=integration).
-- Operator/dev sessions are untouched, so the app's tools never leak into
-- the operator's own CLI sessions.

ALTER TABLE integrations
    ADD COLUMN mcp_servers        JSONB   NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN system_prompt      TEXT    NOT NULL DEFAULT '',
    ADD COLUMN bypass_permissions BOOLEAN NOT NULL DEFAULT FALSE;

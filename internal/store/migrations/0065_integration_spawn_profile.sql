-- 0065 — per-integration spawn profile: the provider-AGNOSTIC half of an
-- integration's spawn config (MCP servers + system prompt + permission
-- mode), complementing 0064's provider/model/account identity defaults.
-- Together they are ONE logical "spawn profile" on the integration row;
-- the split across two migrations is only an accident of time.
--
-- A third-party app declares, ONCE on its integration:
--   • its own MCP tool servers (so the agent can do the app's work), and
--   • a boot system prompt (its operating contract, e.g. "you are the
--     secretary; reply only via reply_to_user"), and
--   • a permission mode (default = the operator-attended TUI gates tool
--     calls; bypass = auto-approve for an unattended app-driven session).
--
-- opendray translates each at spawn through its EXISTING per-provider
-- machinery: MCP via renderMCP (claude --mcp-config / gemini settings /
-- codex|antigravity|opencode config), the prompt via the per-provider
-- system-prompt surface (claude --append-system-prompt / gemini GEMINI.md
-- / codex|antigravity AGENTS.md), and the permission mode via each
-- manifest's bypass flag. Applied ONLY to origin=integration sessions, so
-- the app's tools/prompt never leak into the operator's own CLI sessions.
--
--   mcp_servers:      '[]' (none) | JSON array of {name,command,args,env,url,transport}
--   system_prompt:    '' (none) | markdown injected into every spawned session
--   permission_mode:  'default' (TUI gates) | 'bypass' (auto-approve, unattended)
--   agent_id:         '' (unused) | RESERVED forward-compat FK slot for a future
--                     named, reusable Agent entity that many integrations can share.
--                     Not read at runtime yet — reserved so adopting a named-Agent
--                     model later needs no table reshape.
--
-- Idempotent (IF [NOT] EXISTS) so it converges when re-applied: this
-- migration was reshaped after an earlier form shipped a boolean
-- `bypass_permissions`; permission_mode replaces it (true → 'bypass'),
-- giving room for future modes (plan / acceptEdits) without another
-- migration. An environment that applied the old form can re-run this
-- (delete its schema_migrations row) to converge.

ALTER TABLE integrations
    ADD COLUMN IF NOT EXISTS mcp_servers     JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS system_prompt   TEXT  NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS permission_mode TEXT  NOT NULL DEFAULT 'default',
    ADD COLUMN IF NOT EXISTS agent_id        TEXT  NOT NULL DEFAULT '';

-- Carry a previously-shipped boolean forward, then drop it.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'integrations' AND column_name = 'bypass_permissions'
    ) THEN
        UPDATE integrations SET permission_mode = 'bypass' WHERE bypass_permissions;
        ALTER TABLE integrations DROP COLUMN bypass_permissions;
    END IF;
END $$;

ALTER TABLE integrations DROP CONSTRAINT IF EXISTS integrations_permission_mode_check;
ALTER TABLE integrations ADD CONSTRAINT integrations_permission_mode_check
    CHECK (permission_mode IN ('default', 'bypass'));

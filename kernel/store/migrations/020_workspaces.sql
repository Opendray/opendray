-- 020_workspaces — multi-tenant foundation.
--
-- v0.5 ships the schema only. Behaviour is unchanged: every existing
-- row gets backfilled to a single "default" workspace, and the new
-- workspace_id columns stay NULLABLE so the existing INSERT queries
-- (which don't yet write workspace_id) keep working without change.
--
-- v0.5.1 will update queries to write workspace_id explicitly.
-- v0.5.2 will set NOT NULL once all writers are migrated.
-- v0.6 adds the workspace switcher UI + per-workspace plugin configs.
-- v0.7+ adds workspace_users + RBAC.
--
-- Why this staging: dropping a fully-multitenant model in one PR risks
-- breaking every existing INSERT/SELECT path. Doing it column-by-column
-- with NULLABLE+backfill lets us land the schema today, update callers
-- piecemeal, and tighten constraints last.

CREATE TABLE IF NOT EXISTS workspaces (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    -- is_default marks the workspace new resources fall back to until
    -- the v0.6 switcher UI lets users pick one explicitly. Exactly one
    -- row should have is_default = TRUE; enforced by the partial index
    -- below.
    is_default  BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_workspaces_one_default
    ON workspaces (is_default) WHERE is_default = TRUE;

-- Bootstrap a single default workspace if none exists. The slug
-- "default" is reserved (we never let users create another "default").
INSERT INTO workspaces (name, slug, is_default)
SELECT 'Default', 'default', TRUE
WHERE NOT EXISTS (SELECT 1 FROM workspaces);

-- ── workspace_id columns + backfill on existing resources ────────────

-- sessions
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS workspace_id UUID
    REFERENCES workspaces(id) ON DELETE CASCADE;
UPDATE sessions SET workspace_id = (SELECT id FROM workspaces WHERE is_default LIMIT 1)
    WHERE workspace_id IS NULL;
CREATE INDEX IF NOT EXISTS idx_sessions_workspace ON sessions(workspace_id);

-- claude_accounts: also relax the UNIQUE(name) → UNIQUE(workspace_id, name)
-- so two workspaces can have an account named "personal" each.
ALTER TABLE claude_accounts ADD COLUMN IF NOT EXISTS workspace_id UUID
    REFERENCES workspaces(id) ON DELETE CASCADE;
UPDATE claude_accounts SET workspace_id = (SELECT id FROM workspaces WHERE is_default LIMIT 1)
    WHERE workspace_id IS NULL;
ALTER TABLE claude_accounts DROP CONSTRAINT IF EXISTS claude_accounts_name_key;
-- Use a function-defaulted partial: workspace+name unique only when both set.
-- After v0.5.2 makes workspace_id NOT NULL we can drop the WHERE clause.
CREATE UNIQUE INDEX IF NOT EXISTS idx_claude_accounts_ws_name
    ON claude_accounts(workspace_id, name)
    WHERE workspace_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_claude_accounts_workspace
    ON claude_accounts(workspace_id);

-- llm_providers
ALTER TABLE llm_providers ADD COLUMN IF NOT EXISTS workspace_id UUID
    REFERENCES workspaces(id) ON DELETE CASCADE;
UPDATE llm_providers SET workspace_id = (SELECT id FROM workspaces WHERE is_default LIMIT 1)
    WHERE workspace_id IS NULL;
CREATE INDEX IF NOT EXISTS idx_llm_providers_workspace ON llm_providers(workspace_id);

-- plugin_kv (per-plugin key/value config). Scoping by workspace lets
-- the same plugin (e.g. qdrant) be configured with a different
-- collection per workspace.
ALTER TABLE plugin_kv ADD COLUMN IF NOT EXISTS workspace_id UUID
    REFERENCES workspaces(id) ON DELETE CASCADE;
UPDATE plugin_kv SET workspace_id = (SELECT id FROM workspaces WHERE is_default LIMIT 1)
    WHERE workspace_id IS NULL;
-- The existing UNIQUE(plugin, key) becomes UNIQUE(workspace_id, plugin, key).
ALTER TABLE plugin_kv DROP CONSTRAINT IF EXISTS plugin_kv_plugin_key_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_plugin_kv_ws_plugin_key
    ON plugin_kv(workspace_id, plugin, key)
    WHERE workspace_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_plugin_kv_workspace ON plugin_kv(workspace_id);

-- plugin_secret (per-plugin encrypted values). Same scoping rationale.
ALTER TABLE plugin_secret ADD COLUMN IF NOT EXISTS workspace_id UUID
    REFERENCES workspaces(id) ON DELETE CASCADE;
UPDATE plugin_secret SET workspace_id = (SELECT id FROM workspaces WHERE is_default LIMIT 1)
    WHERE workspace_id IS NULL;
ALTER TABLE plugin_secret DROP CONSTRAINT IF EXISTS plugin_secret_plugin_key_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_plugin_secret_ws_plugin_key
    ON plugin_secret(workspace_id, plugin, key)
    WHERE workspace_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_plugin_secret_workspace ON plugin_secret(workspace_id);

-- plugin_consents (per-plugin permission grants).
ALTER TABLE plugin_consents ADD COLUMN IF NOT EXISTS workspace_id UUID
    REFERENCES workspaces(id) ON DELETE CASCADE;
UPDATE plugin_consents SET workspace_id = (SELECT id FROM workspaces WHERE is_default LIMIT 1)
    WHERE workspace_id IS NULL;
CREATE INDEX IF NOT EXISTS idx_plugin_consents_workspace ON plugin_consents(workspace_id);

-- Tables intentionally NOT scoped (server-global):
--   plugins              — installation registry; same install for all workspaces
--   admin_auth           — root login lives at server level
--   plugin_audit         — event stream; can filter by workspace_id of the
--                          target resource client-side
--   plugin_secret_kek    — server-wide key encryption key
--   plugin_host_state    — server-wide sidecar state
--   plugin_tombstone     — server-wide uninstall record
--   source_control_baselines — per-session, transitively scoped via sessions.workspace_id

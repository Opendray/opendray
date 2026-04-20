-- 018_plugin_tombstone.sql
--
-- Records plugin names the user has explicitly uninstalled so
-- Runtime.LoadAll doesn't re-seed them from the embedded/filesystem
-- bundle on every gateway restart.
--
-- Why a separate table vs. a soft-delete column on plugins:
--   1. The plugin row's DELETE cascades (plugin_kv, plugin_secret,
--      plugin_consent, plugin_audit, …) are the intended semantics of
--      uninstall — the user's config should not haunt a reinstall. A
--      soft-delete column would require adding WHERE clauses to every
--      downstream read and would leak into FK designs.
--   2. The tombstone survives uninstall->wait->(reinstall via Hub)
--      because Install.Confirm explicitly deletes the row before
--      seeding, which is an explicit user action.
--
-- Storing uninstalled_at (rather than a plain name list) gives us a
-- cheap "recently uninstalled" view later without a schema change.

CREATE TABLE IF NOT EXISTS plugin_tombstone (
    name           TEXT PRIMARY KEY,
    uninstalled_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

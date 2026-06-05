-- M-U Phase 1 — Collapse the memory scope model from three to two.
--
-- session | project | global  ->  project | global
--
-- A session is always built on a project, so a session-scoped memory is
-- a redundant partition that fragments recall. This migration folds every
-- session-scoped row into its project and removes 'session' as a legal
-- scope. It is LOSSLESS: nothing is deleted.
--
--   * A session row whose session still exists becomes a project row
--     keyed by that session's cwd.
--   * A session row whose session is gone (an orphan) is soft-archived
--     rather than dropped: scope flips to 'project' to satisfy the new
--     CHECK, archived_at is stamped so it is excluded once the Phase 4
--     archive filter lands, and its text/embedding are preserved. (Its
--     scope_key keeps the old session id, which matches no cwd query, so
--     it is already unreachable by normal project search.)
--
-- The archived_at / archived_reason columns are the soft-delete primitive
-- the rest of the M-U arc (Phase 4) builds on; they are introduced here
-- so the fold has a lossless place to put orphans.

-- 1. Soft-delete primitive (idempotent — Phase 4 reuses these columns).
ALTER TABLE memories
    ADD COLUMN IF NOT EXISTS archived_at     TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS archived_reason TEXT;

-- 2. Fold session rows whose session still exists into their project,
--    keyed by the session's cwd.
UPDATE memories m
   SET scope     = 'project',
       scope_key = s.cwd,
       updated_at = NOW()
  FROM sessions s
 WHERE m.scope = 'session'
   AND s.id = m.scope_key;

-- 3. Archive orphans (session gone) instead of deleting them. After
--    step 2 every remaining scope='session' row is an orphan.
UPDATE memories
   SET scope           = 'project',
       archived_at     = NOW(),
       archived_reason = 'session scope retired (M-U Phase 1); originating session not found',
       updated_at      = NOW()
 WHERE scope = 'session';

-- 4. Swap the CHECK constraint now that no row is scope='session'.
ALTER TABLE memories
    DROP CONSTRAINT IF EXISTS memories_scope_check;

ALTER TABLE memories
    ADD CONSTRAINT memories_scope_check
        CHECK (scope IN ('project', 'global'));

-- 0049 — purge ephemeral-cwd project footprint (Cortex).
--
-- Sessions spawned in throwaway temp dirs — third-party consumers and
-- tests constantly run in /tmp, /var/folders, .cache — accumulated
-- journal entries, scanner docs, lifecycle rows, and (since 0046)
-- seeded blueprint sections, all of which made them show up as
-- "projects" in the Cortex UI. The knowledge anchorer has always
-- skipped these cwds; as of this release the journaler, capture
-- pipeline, scanners, blueprint seeding, and doc writes do too
-- (projectdoc.IsEphemeralCwd). This migration removes the residue.
--
-- The SQL pattern below mirrors projectdoc.IsEphemeralCwd exactly:
--   ''            | '/tmp' | '/tmp/%' | '/private/tmp%'
--   '/var/folders/%' | '/private/var/folders/%' | '%/tmp.%' | '%/.cache/%'
--
-- Doc/journal/blueprint rows are deleted outright (throwaway by
-- definition); episodic memories are SOFT-ARCHIVED instead so anything
-- mistakenly swept is restorable within the cleaner's grace window.

-- Blueprint sections (0046 seeded 5 per known cwd, tmp dirs included).
DELETE FROM doc_blueprint_sections
 WHERE lower(cwd) = '/tmp'
    OR lower(cwd) LIKE '/tmp/%'
    OR lower(cwd) LIKE '/private/tmp%'
    OR lower(cwd) LIKE '/var/folders/%'
    OR lower(cwd) LIKE '/private/var/folders/%'
    OR lower(cwd) LIKE '%/tmp.%'
    OR lower(cwd) LIKE '%/.cache/%';

-- Scanner docs / goal / plan / overview rows for tmp cwds.
DELETE FROM project_docs
 WHERE lower(cwd) = '/tmp'
    OR lower(cwd) LIKE '/tmp/%'
    OR lower(cwd) LIKE '/private/tmp%'
    OR lower(cwd) LIKE '/var/folders/%'
    OR lower(cwd) LIKE '/private/var/folders/%'
    OR lower(cwd) LIKE '%/tmp.%'
    OR lower(cwd) LIKE '%/.cache/%';

DELETE FROM project_doc_proposals
 WHERE lower(cwd) = '/tmp'
    OR lower(cwd) LIKE '/tmp/%'
    OR lower(cwd) LIKE '/private/tmp%'
    OR lower(cwd) LIKE '/var/folders/%'
    OR lower(cwd) LIKE '/private/var/folders/%'
    OR lower(cwd) LIKE '%/tmp.%'
    OR lower(cwd) LIKE '%/.cache/%';

-- Journal entries — the main reason tmp dirs appeared in ListProjects
-- (it unions project_docs with session_logs cwds).
DELETE FROM session_logs
 WHERE lower(cwd) = '/tmp'
    OR lower(cwd) LIKE '/tmp/%'
    OR lower(cwd) LIKE '/private/tmp%'
    OR lower(cwd) LIKE '/var/folders/%'
    OR lower(cwd) LIKE '/private/var/folders/%'
    OR lower(cwd) LIKE '%/tmp.%'
    OR lower(cwd) LIKE '%/.cache/%';

DELETE FROM project_lifecycle
 WHERE lower(cwd) = '/tmp'
    OR lower(cwd) LIKE '/tmp/%'
    OR lower(cwd) LIKE '/private/tmp%'
    OR lower(cwd) LIKE '/var/folders/%'
    OR lower(cwd) LIKE '/private/var/folders/%'
    OR lower(cwd) LIKE '%/tmp.%'
    OR lower(cwd) LIKE '%/.cache/%';

DELETE FROM memory_cleanup_decisions
 WHERE lower(memory_scope_key) = '/tmp'
    OR lower(memory_scope_key) LIKE '/tmp/%'
    OR lower(memory_scope_key) LIKE '/private/tmp%'
    OR lower(memory_scope_key) LIKE '/var/folders/%'
    OR lower(memory_scope_key) LIKE '/private/var/folders/%'
    OR lower(memory_scope_key) LIKE '%/tmp.%'
    OR lower(memory_scope_key) LIKE '%/.cache/%';

-- Episodic memories captured under tmp cwds: SOFT-archive (reversible
-- within the grace window) rather than delete.
UPDATE memories
   SET archived_at = NOW(),
       archived_reason = 'ephemeral cwd cleanup (0049)',
       updated_at = NOW()
 WHERE scope = 'project'
   AND archived_at IS NULL
   AND (lower(scope_key) = '/tmp'
     OR lower(scope_key) LIKE '/tmp/%'
     OR lower(scope_key) LIKE '/private/tmp%'
     OR lower(scope_key) LIKE '/var/folders/%'
     OR lower(scope_key) LIKE '/private/var/folders/%'
     OR lower(scope_key) LIKE '%/tmp.%'
     OR lower(scope_key) LIKE '%/.cache/%');

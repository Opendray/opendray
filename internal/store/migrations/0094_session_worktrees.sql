-- 0094 — worktree isolation: split "where the CLI runs" from "which
-- project this session belongs to".
--
--   cwd             (existing) — the LOGICAL project path. Keys memory,
--                   project docs, Cortex, journal, custom tasks, UI
--                   grouping. Its meaning does not change.
--   work_dir        (new) — the PHYSICAL execution path. '' means "not
--                   isolated; the process runs in cwd" (every existing
--                   row). For isolated sessions it points at the
--                   session's managed git worktree; child shell
--                   sessions inherit the parent's work_dir.
--   worktree_branch (new) — the branch the managed worktree was created
--                   on (opendray/<session-id>). Set only on the OWNING
--                   session (the one whose create carried
--                   isolation=worktree); '' for children and normal
--                   sessions. Doubles as "this row owns a worktree" for
--                   reclamation.
--
-- The sessions table is also the worktree registry: scope-key
-- canonicalization (work_dir -> cwd) is rebuilt from these columns at
-- boot. See docs/design/worktree-isolation.md.

ALTER TABLE sessions ADD COLUMN IF NOT EXISTS work_dir        TEXT NOT NULL DEFAULT '';
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS worktree_branch TEXT NOT NULL DEFAULT '';

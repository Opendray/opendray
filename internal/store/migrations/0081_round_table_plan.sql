-- 0081_round_table_plan — a role-based execution plan for the Round Table
-- (experimental). After the discussion, the group's work is broken into an
-- ordered list of steps, each assigned to the member whose strength fits
-- (claude=code, antigravity=UI, codex=review). The operator runs steps one at
-- a time; each spawns a real agent session in the shared project cwd, so the
-- specialists collaborate through the working tree.
--
--   plan = [{assignee, model, account_id, task, status, session_id}]
--
-- Forward-only, idempotent, non-destructive: a plain ADD COLUMN IF NOT EXISTS
-- with a default. Rolls back with
-- `ALTER TABLE round_tables DROP COLUMN plan;` (see ROLLBACK.md).
ALTER TABLE round_tables
    ADD COLUMN IF NOT EXISTS plan JSONB NOT NULL DEFAULT '[]'::jsonb;

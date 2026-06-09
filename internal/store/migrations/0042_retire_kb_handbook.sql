-- 0042_retire_kb_handbook — Experience Flywheel: delete the per-project
-- handbook pages. Knowledge is cross-project only; the per-project document is
-- Notes (goal/plan/tech/journal). kb_handbook duplicated that, so it is
-- removed. The kind stays in the CHECK constraint (no constraint change) so
-- this is a pure data cleanup; the service layer no longer creates the kind.
DELETE FROM project_docs WHERE kind = 'kb_handbook';
DELETE FROM project_doc_proposals WHERE kind = 'kb_handbook';

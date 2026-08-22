-- Allow 'approved' as a project_docs author.
--
-- Approving a KB proposal stamps the resulting doc updated_by='approved'
-- (an AI draft the operator sanctioned — locks the page like 'operator'
-- but keeps deletion-as-signal mining from reading AI-draft diffs as
-- human deletions). The CHECK constraint from 0027 predates that value,
-- so every approval hit a constraint violation and the operator's
-- "approve" click failed. Same relaxation pattern 0027 itself used.

ALTER TABLE project_docs DROP CONSTRAINT IF EXISTS project_docs_updated_by_check;
ALTER TABLE project_docs ADD CONSTRAINT project_docs_updated_by_check
    CHECK (updated_by IN ('operator', 'agent', 'scanner', 'approved'));

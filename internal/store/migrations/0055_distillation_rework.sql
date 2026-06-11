-- 0055 — distillation rework: clean slate for playbooks (Cortex).
--
-- Pre-gate playbooks were one-liner "memories" with no procedure, no
-- evidence, no value — promoting them produced junk skills. Playbooks
-- are DERIVED data: archive them all (reversible) and let the next
-- reflect cycle re-distill under the new structural quality gate
-- (≥3 steps, concrete commands/paths, verbatim evidence quotes from
-- the work log, a real trigger).
--
-- Skills are NOT touched — the operator retires those by hand from
-- the workbench (some may have been hand-curated).

UPDATE knowledge_nodes
   SET archived_at = NOW(),
       archived_reason = 'distillation rework (0055) — re-distilled under quality gates'
 WHERE kind = 'playbook'
   AND archived_at IS NULL;

-- Clear every project's reflect signature so the dirty-check doesn't
-- skip re-distillation of unchanged feedstock.
UPDATE knowledge_nodes
   SET provenance = provenance - 'reflect_sig'
 WHERE kind = 'entity'
   AND provenance ? 'reflect_sig';

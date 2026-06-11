-- 0056 — experience compiler (Cortex distillation rework, round 2).
--
-- Skills must now come from REPETITION + SUCCESS evidence: the compiler
-- mines session journals ACROSS projects, clusters similar episodes, and
-- only drafts a candidate when the same procedure succeeded in ≥2
-- sessions. Closing the loop needs outcome counters on skills: when a
-- session that referenced a skill ends, we record whether it succeeded
-- (exit code 0) — skills that get loaded but keep ending in failure are
-- retirement candidates alongside the never-used ones.

ALTER TABLE knowledge_nodes
    ADD COLUMN success_count INT NOT NULL DEFAULT 0,
    ADD COLUMN failure_count INT NOT NULL DEFAULT 0;

-- Un-promoted playbooks from the per-project reflector are superseded:
-- they were distilled from single-project logs without recurrence
-- evidence, which is exactly what the rework removes. Archive (never
-- delete — reversible, lineage preserved); the compiler re-distills
-- anything that genuinely recurs. Skills are NOT touched.
UPDATE knowledge_nodes
   SET archived_at = NOW(),
       provenance = jsonb_set(COALESCE(provenance, '{}'::jsonb),
                              '{archived_reason}',
                              '"experience compiler (0056) — superseded by recurrence-evidence mining"')
 WHERE kind = 'playbook'
   AND archived_at IS NULL
   AND provenance->>'source' = 'reflector';

-- The per-project reflect dirty-check signature is dead state now.
UPDATE knowledge_nodes
   SET provenance = provenance - 'reflect_sig'
 WHERE kind = 'entity'
   AND provenance ? 'reflect_sig';

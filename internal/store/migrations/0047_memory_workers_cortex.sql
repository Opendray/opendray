-- Cortex — add the blueprint proposer + curation conversation
-- touchpoints to the memory_workers task catalogue.
--
--   blueprint — classifies a project from repo signals and proposes a
--               doc section set (operator-triggered, Phase 3).
--   curation  — the conversational channel for updating doc sections
--               and re-drafting Foundational/Emergent knowledge pages
--               with the AI (Phase 4).
--
-- Both seed kind='summarizer' so existing deployments route through
-- their default worker until the operator flips a row to Agent.

ALTER TABLE memory_workers
    DROP CONSTRAINT IF EXISTS memory_workers_task_check;

ALTER TABLE memory_workers
    ADD CONSTRAINT memory_workers_task_check
        CHECK (task IN (
            'gatekeeper','cleaner','gitactivity','transcript',
            'plan_drift','conflict_detector','capture',
            'blueprint','curation'
        ));

INSERT INTO memory_workers (task, kind) VALUES
    ('blueprint', 'summarizer'),
    ('curation', 'summarizer')
ON CONFLICT (task) DO NOTHING;

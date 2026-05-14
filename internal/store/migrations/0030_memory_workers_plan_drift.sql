-- M-PA — Add plan_drift to the memory_workers task catalogue.
--
-- Plan-drift detection runs after each session.ended event: given
-- the session transcript summary + recent journal + current plan,
-- decide whether the plan needs an update and file a proposal. Same
-- pluggable surface as the other LLM touchpoints (summarizer vs
-- agent) so operators can route it independently.
--
-- The check constraint is recreated with the new task value. Seed
-- a summarizer default so existing deployments get the feature on
-- next start without operator action.

ALTER TABLE memory_workers
    DROP CONSTRAINT IF EXISTS memory_workers_task_check;

ALTER TABLE memory_workers
    ADD CONSTRAINT memory_workers_task_check
        CHECK (task IN ('gatekeeper','cleaner','gitactivity','transcript','plan_drift'));

INSERT INTO memory_workers (task, kind) VALUES
    ('plan_drift', 'summarizer')
ON CONFLICT (task) DO NOTHING;

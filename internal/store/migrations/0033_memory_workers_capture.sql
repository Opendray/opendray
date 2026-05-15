-- M-PE — Add capture to the memory_workers task catalogue.
--
-- Pre M-PE the capture engine talked to summarizer.Registry
-- directly, so operators could only point it at HTTP summarizer
-- endpoints (Ollama / LM Studio / Anthropic / OpenAI / Integration).
-- The other five touchpoints (gatekeeper / cleaner / gitactivity /
-- transcript / plan_drift / conflict_detector) had already gone
-- through the worker fabric, giving them the choice of Agent (CLI
-- --print) for higher-quality Claude / Gemini runs.
--
-- This migration lets operators pick the same Agent mode for
-- capture. The default seed stays kind='summarizer' so existing
-- deployments behave identically until an operator flips it.

ALTER TABLE memory_workers
    DROP CONSTRAINT IF EXISTS memory_workers_task_check;

ALTER TABLE memory_workers
    ADD CONSTRAINT memory_workers_task_check
        CHECK (task IN (
            'gatekeeper','cleaner','gitactivity','transcript',
            'plan_drift','conflict_detector','capture'
        ));

INSERT INTO memory_workers (task, kind) VALUES
    ('capture', 'summarizer')
ON CONFLICT (task) DO NOTHING;

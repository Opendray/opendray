-- 0051 — per-task model selection for agent workers (Cortex).
--
-- The operator can pick WHICH agent drives each Cortex touchpoint
-- (claude / gemini / local summarizer) but not WHICH MODEL — so basic
-- chores (capture, gatekeeper) burned frontier-model money. model is
-- passed to the agent CLI (`claude --model …` / `gemini --model …`);
-- empty keeps the CLI's default. Summarizer-kind workers ignore it
-- (their model lives on the summarizer provider row).

ALTER TABLE memory_workers
    ADD COLUMN model TEXT;

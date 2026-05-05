-- 0022_capture_more_triggers — extend trigger_kind enum.
--
-- Phase A shipped only after_messages. Phase B adds:
--   on_idle:   fire when the session has been idle for N seconds
--              since the last user message. trigger_config: {"seconds": 60}
--   k_chars:   fire when the cumulative character count of new
--              user messages crosses K. trigger_config: {"k": 4000}
--   manual:    never auto-fire; only triggered via
--              POST /memory-capture-rules/{id}/run-now.

ALTER TABLE memory_capture_rules
    DROP CONSTRAINT IF EXISTS memory_capture_rules_trigger_kind_check;

ALTER TABLE memory_capture_rules
    ADD CONSTRAINT memory_capture_rules_trigger_kind_check
    CHECK (trigger_kind IN ('after_messages','on_idle','k_chars','manual'));

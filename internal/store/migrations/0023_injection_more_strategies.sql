-- 0023_injection_more_strategies — extend strategy_kind enum.
--
-- Phase A shipped only 'none' and 'top_k_recent'. Phase B adds:
--   top_k_relevant: spawn-time semantic search using cwd as query
--                   for top K most-relevant memories (instead of
--                   most-recent). config: {"k": 5}
--   on_keyword:     reserved — UI lets the operator pick keywords;
--                   actual hook into message stream is Phase C.
--   manual_only:    no auto-injection; only via API/UI button.
--   hybrid:         inject one ultra-short summary line at spawn
--                   (top-1 by recency, 80 chars max), leaving room
--                   for the model to memory_search if it wants more.

ALTER TABLE memory_injection_profiles
    DROP CONSTRAINT IF EXISTS memory_injection_profiles_strategy_kind_check;

ALTER TABLE memory_injection_profiles
    ADD CONSTRAINT memory_injection_profiles_strategy_kind_check
    CHECK (strategy_kind IN ('none','top_k_recent','top_k_relevant','on_keyword','manual_only','hybrid'));

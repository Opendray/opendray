-- Polarity: what KIND of utterance a memory is.
--
-- The store held every memory as an undifferentiated statement, and the
-- KB drafter consumed each one as "evidence to fold into the page".
-- That is a category error with a sharp edge: a directive about the
-- documentation itself ("docs must not mention X") is feedstock whose
-- text contains X, so the drafter reads it as evidence ABOUT X and
-- writes X back — and every time the operator repeats the instruction,
-- the loop gets another feedstock line to feed on.
--
--   fact        statement about the world (the default; today's behaviour)
--   rule        binding directive about how we WORK — belongs on pages,
--               tagged so the drafter files it as a rule, not a narrative
--   meta        directive about the docs/knowledge system ITSELF — must
--               never enter feedstock at all
--   correction  supersedes an earlier claim; enters feedstock as a fact
--
-- NULL means not yet classified. A background classifier fills it in
-- batches; consumers treat NULL as 'fact', so an unclassified backlog —
-- or a classifier outage — is exactly the pre-migration behaviour and
-- nothing is ever lost to it.

ALTER TABLE memories
  ADD COLUMN IF NOT EXISTS polarity TEXT
  CHECK (polarity IN ('fact', 'rule', 'meta', 'correction'));

CREATE INDEX IF NOT EXISTS idx_memories_polarity_unclassified
  ON memories (created_at)
  WHERE polarity IS NULL AND archived_at IS NULL;

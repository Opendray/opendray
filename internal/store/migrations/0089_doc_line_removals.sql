-- Deletion-as-signal: remember what the operator removed from a page.
--
-- An operator deleting a line from a knowledge page is the highest-
-- confidence negative signal the system can receive, and until now it
-- was discarded entirely: the drafter regenerates the page from
-- feedstock, so a deleted line whose evidence still exists upstream is
-- re-derived on the next sweep, and the operator finds themselves in a
-- tug-of-war with their own curation loop.
--
-- Each row is one normalized line the operator deleted from one page.
-- removal_count is the escalation ladder: at 1 the line becomes soft
-- negative context for the drafter ("the operator removed these — do
-- not reintroduce them"); at 2 — which can only mean the system put
-- the line back and the operator deleted it AGAIN — the row flips to
-- 'banned' and the line is deterministically scrubbed from every
-- future draft. A reworded line hashes differently and starts a fresh
-- row, so escalation cannot fire on an ordinary rewrite.
--
-- status: 'active' (recorded, soft), 'banned' (hard-scrubbed),
-- 'dismissed' (operator unbanned / told us to forget it).

CREATE TABLE IF NOT EXISTS doc_line_removals (
  id               TEXT PRIMARY KEY,
  cwd              TEXT NOT NULL,
  kind             TEXT NOT NULL,
  line_hash        TEXT NOT NULL,
  line_text        TEXT NOT NULL,
  removal_count    INT  NOT NULL DEFAULT 1,
  status           TEXT NOT NULL DEFAULT 'active'
                   CHECK (status IN ('active', 'banned', 'dismissed')),
  first_removed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_removed_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (cwd, kind, line_hash)
);

CREATE INDEX IF NOT EXISTS idx_doc_line_removals_page
  ON doc_line_removals (cwd, kind, status);

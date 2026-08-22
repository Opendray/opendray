-- Per-page exclusions: subjects the KB drafter must never write about.
--
-- A knowledge page is regenerated from feedstock (memory fact titles +
-- graph entities), so deleting a line from the page changes nothing
-- upstream — the next sweep re-derives it from the same source and the
-- operator's edit looks like it was "reverted". Worse, a fact that says
-- "never mention X" is itself feedstock whose title contains X, so the
-- drafter reads it as evidence about X and writes X back. Repeating the
-- instruction makes the loop tighter, not looser.
--
-- The pipeline had no way to express a NEGATIVE: prompt_hint steers form
-- and scope, but every other input is material to fold in. This column
-- is that missing channel. Each entry is matched case-insensitively on
-- word boundaries and applied at BOTH ends of the draft — offending
-- feedstock lines and offending lines of the current page are dropped
-- before the model sees them, and any that survive into the generated
-- body are stripped after. Filtering the input is what actually breaks
-- the loop; the output pass is the backstop.
--
-- Empty array = today's behaviour exactly, so every existing row is
-- already correct and nothing needs re-entering.

ALTER TABLE doc_blueprint_sections
  ADD COLUMN IF NOT EXISTS exclusions TEXT[] NOT NULL DEFAULT '{}';

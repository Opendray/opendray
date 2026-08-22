-- Rule suggestions in curation conversations.
--
-- A page's steering rules (Guidance / Excluded subjects) are operator-owned
-- settings: the AI drafter must not edit them, but the DISCUSSION is exactly
-- where a needed rule change surfaces ("stop mentioning X" → an exclusion).
-- Making the operator re-type the suggestion into the settings dialog loses
-- the value of the AI having produced it. So an AI turn may carry a
-- structured rule suggestion; the operator applies it with one click and the
-- backend merges it into doc_blueprint_sections — approval stays human, the
-- typing doesn't.
--
-- rule_suggestion: the suggestion payload (guidance replacement +
-- exclusions to add/remove + reason); NULL for messages without one.
-- rule_applied_at: set when the operator applied it — the UI flips the
-- Apply button to an applied badge and the backend refuses a second apply.

ALTER TABLE cortex_conversation_messages
  ADD COLUMN IF NOT EXISTS rule_suggestion JSONB,
  ADD COLUMN IF NOT EXISTS rule_applied_at TIMESTAMPTZ;

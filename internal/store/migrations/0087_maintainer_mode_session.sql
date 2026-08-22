-- Allow maintainer_mode = 'session' on doc blueprint sections.
--
-- The mode answers "who keeps this section current". The three 0046
-- values cover automation (ai), the operator (human) and a mechanical
-- rebuilder (scanner) — but not the agent doing the work. Some knowledge
-- only that agent ever holds: the container it just provisioned, the
-- database role it just created, the port it chose. The background
-- drafter distils episodic facts and cannot reconstruct any of it, so a
-- page that records such things has no viable maintainer among the three.
--
-- 'session' hands the pen to the in-session agent, which writes the page
-- through the kb_page_set MCP tool. Which pages carry the mode is the
-- operator's choice, stored here and read at tools/list time, so no page
-- slug is special-cased anywhere in the code.
--
-- Widening a CHECK only: every existing row stays valid and nothing is
-- rewritten. Dropping by the generated name is safe — it is what
-- CREATE TABLE ... CHECK produced in 0046 — and IF EXISTS keeps a re-run
-- harmless.

ALTER TABLE doc_blueprint_sections
  DROP CONSTRAINT IF EXISTS doc_blueprint_sections_maintainer_mode_check;

ALTER TABLE doc_blueprint_sections
  ADD CONSTRAINT doc_blueprint_sections_maintainer_mode_check
  CHECK (maintainer_mode IN ('ai', 'human', 'scanner', 'session'));

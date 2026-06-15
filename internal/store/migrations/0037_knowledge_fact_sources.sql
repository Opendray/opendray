-- M-KG Phase 1 — link fact nodes to the episodic memory rows they are
-- distilled from.
--
-- A fact node REFERENCES its source memory row(s) here; it does not copy the
-- text (the full content stays in `memories`, the fact node carries only a
-- short title label). Decoupling: this FKs knowledge_nodes, but only
-- SOFT-references memories by id (NO foreign key to memories) so the memory
-- schema stays untouched and the dependency remains one-way.

CREATE TABLE IF NOT EXISTS knowledge_fact_sources (
    node_id   TEXT NOT NULL REFERENCES knowledge_nodes(id) ON DELETE CASCADE,
    memory_id TEXT NOT NULL,
    PRIMARY KEY (node_id, memory_id)
);

CREATE INDEX IF NOT EXISTS knowledge_fact_sources_memory_idx
    ON knowledge_fact_sources (memory_id);

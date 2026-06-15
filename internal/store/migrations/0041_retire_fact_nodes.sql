-- 0041_retire_fact_nodes — P-G: retire the knowledge-graph `fact` node layer.
--
-- Fact nodes were a 1:1 mirror of episodic Memory rows. The KB pages + the
-- reflector now distil straight from Memory (the canonical fact store), so the
-- duplicate layer is removed. The graph keeps entity / playbook / skill nodes.
--
-- knowledge_fact_sources is repurposed as the anchorer's "memory processed"
-- marker, now pointing a memory_id at the project ENTITY instead of a fact
-- node. We clear the old fact-node-keyed rows here; the next consolidation
-- sweep re-derives entities straight from Memory and re-marks processed
-- memories against their project entity. Entity nodes themselves survive, so
-- the re-derivation is mostly idempotent (findOrCreateEntity dedups by name) —
-- a bounded one-time re-extraction, the same cost profile as a graph reset.
--
-- Done as deletes (not an in-place re-point) so the migration can never trip
-- the (node_id, memory_id) unique constraint and block startup.

-- 1. Drop the fact-node-keyed processed markers (re-derived next sweep).
DELETE FROM knowledge_fact_sources fs
 USING knowledge_nodes f
 WHERE f.id = fs.node_id AND f.kind = 'fact';

-- 2. Delete edges touching fact nodes, then the fact nodes themselves.
DELETE FROM knowledge_edges
 WHERE src_id IN (SELECT id FROM knowledge_nodes WHERE kind = 'fact')
    OR dst_id IN (SELECT id FROM knowledge_nodes WHERE kind = 'fact');

DELETE FROM knowledge_nodes WHERE kind = 'fact';

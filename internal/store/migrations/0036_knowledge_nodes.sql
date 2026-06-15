-- M-KG Phase 0 — structured knowledge tier substrate.
--
-- A DB-native typed knowledge graph that grows ON TOP of the M-U episodic
-- memory. Decoupling: knowledge_* may reference memories, but memory never
-- references knowledge_* (strict one-way dependency). No intelligence yet —
-- just the tables + closed-vocabulary CHECK constraints. The whole tier sits
-- behind the [knowledge].enabled flag; when off, these tables stay empty.
--
-- `vector` is the pgvector type already used by memories / project_docs;
-- embedding is left NULL in Phase 0 and picked up by the Phase 2 backfill.

CREATE TABLE IF NOT EXISTS knowledge_nodes (
    id           TEXT PRIMARY KEY,
    kind         TEXT NOT NULL CHECK (kind IN ('entity','fact','playbook','skill')),
    entity_type  TEXT CHECK (entity_type IN ('service','host','project','tool','decision','tech','person')),
    title        TEXT NOT NULL,
    body         TEXT NOT NULL DEFAULT '',
    scope        TEXT NOT NULL DEFAULT 'project' CHECK (scope IN ('project','domain','global')),
    scope_key    TEXT NOT NULL DEFAULT '',
    maturity     TEXT NOT NULL DEFAULT 'candidate' CHECK (maturity IN ('candidate','fact','playbook','skill')),
    confidence   DOUBLE PRECISION,
    provenance   JSONB NOT NULL DEFAULT '{}'::jsonb,
    embedding    vector,
    embedder     TEXT,
    embedding_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    archived_at  TIMESTAMPTZ,
    -- entity_type is required for entity nodes and forbidden otherwise.
    CONSTRAINT knowledge_nodes_entity_type_ck CHECK (
        (kind = 'entity' AND entity_type IS NOT NULL)
        OR (kind <> 'entity' AND entity_type IS NULL)
    )
);

CREATE INDEX IF NOT EXISTS knowledge_nodes_kind_idx
    ON knowledge_nodes (kind) WHERE archived_at IS NULL;
CREATE INDEX IF NOT EXISTS knowledge_nodes_scope_idx
    ON knowledge_nodes (scope, scope_key) WHERE archived_at IS NULL;
-- Drives the Phase 2 embedding backfill (mirrors memories / project_docs).
CREATE INDEX IF NOT EXISTS knowledge_nodes_needs_embed_idx
    ON knowledge_nodes (updated_at) WHERE embedding IS NULL AND archived_at IS NULL;

CREATE TABLE IF NOT EXISTS knowledge_edges (
    src_id     TEXT NOT NULL REFERENCES knowledge_nodes(id) ON DELETE CASCADE,
    edge_type  TEXT NOT NULL CHECK (edge_type IN (
                   'runs_on','uses','about','part_of',
                   'depends_on','supersedes','derived_from','used_by')),
    dst_id     TEXT NOT NULL REFERENCES knowledge_nodes(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (src_id, edge_type, dst_id),
    CONSTRAINT knowledge_edges_no_self CHECK (src_id <> dst_id)
);

CREATE INDEX IF NOT EXISTS knowledge_edges_src_idx ON knowledge_edges (src_id, edge_type);
CREATE INDEX IF NOT EXISTS knowledge_edges_dst_idx ON knowledge_edges (dst_id, edge_type);

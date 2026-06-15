-- 0048 — curation conversations (Cortex Phase 4).
--
-- The conversational channel the operator uses to actively maintain
-- the system: "更新技术栈", "根据最近的工作重写计划", or — for the
-- Foundational knowledge pages — discuss and re-draft the standing
-- infrastructure/conventions policies with the AI. Each conversation
-- is bound to one target (a project doc section, a global knowledge
-- page, or a project's blueprint) and its AI replies may carry a
-- structured revision that is either applied directly (ai-maintained,
-- unlocked target) or filed as a proposal (human-locked target).
--
-- Escalation: a conversation can be promoted to a full agent session
-- (codebase-grounded); the link is recorded in escalated_session_id.

CREATE TABLE cortex_conversations (
    id                   TEXT        PRIMARY KEY,
    target_kind          TEXT        NOT NULL
        CHECK (target_kind IN ('doc_section', 'kb_page', 'blueprint')),
    -- target_cwd: the project cwd, or '__global__' for kb_page.
    target_cwd           TEXT        NOT NULL,
    -- target_slug: section slug / kb kind; 'blueprint' for blueprints.
    target_slug          TEXT        NOT NULL,
    status               TEXT        NOT NULL DEFAULT 'open'
        CHECK (status IN ('open', 'closed', 'escalated')),
    escalated_session_id TEXT,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cortex_conversations_target
    ON cortex_conversations (target_cwd, target_slug);

CREATE TABLE cortex_conversation_messages (
    id              TEXT        PRIMARY KEY,
    conversation_id TEXT        NOT NULL
        REFERENCES cortex_conversations(id) ON DELETE CASCADE,
    role            TEXT        NOT NULL
        CHECK (role IN ('operator', 'ai', 'system')),
    content         TEXT        NOT NULL,
    -- revision_action: '' | 'applied' | 'proposed' — what the AI's
    -- revision did. revision_ref holds the doc id / proposal id.
    revision_action TEXT        NOT NULL DEFAULT '',
    revision_ref    TEXT        NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cortex_conversation_messages_conv
    ON cortex_conversation_messages (conversation_id, created_at);

-- 0019_memory_summarizer_providers — first-class provider config.
--
-- Each row is one summarizer LLM that opendray can call to extract
-- durable facts from a conversation transcript. kind picks the
-- implementation (anthropic | ollama in Phase A; openai +
-- integration reserved for Phase B). config holds non-sensitive
-- knobs (model name, base_url for ollama). api_key_ciphertext
-- holds an AES-GCM envelope encrypted with the same backup
-- passphrase derived key as backup_targets — the plaintext is
-- never persisted; reads decrypt on demand inside service code.
--
-- enabled lets operators stage a provider without making it
-- selectable. is_default marks the implicit fallback for capture
-- rules that don't pin a specific provider.

CREATE TABLE IF NOT EXISTS memory_summarizer_providers (
    id                    TEXT PRIMARY KEY,
    name                  TEXT NOT NULL UNIQUE,
    kind                  TEXT NOT NULL
        CHECK (kind IN ('anthropic','ollama')),
    model                 TEXT NOT NULL,
    base_url              TEXT NOT NULL DEFAULT '',
    api_key_ciphertext    TEXT,
    api_key_fingerprint   TEXT,
    extra_config          JSONB NOT NULL DEFAULT '{}'::jsonb,
    enabled               BOOLEAN NOT NULL DEFAULT TRUE,
    is_default            BOOLEAN NOT NULL DEFAULT FALSE,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Partial unique index: at most one row may be is_default=TRUE
-- across the whole table. Lets operators flip the default flag
-- atomically without storing a sentinel.
CREATE UNIQUE INDEX IF NOT EXISTS memory_summarizer_providers_default_idx
    ON memory_summarizer_providers((is_default))
    WHERE is_default = TRUE;

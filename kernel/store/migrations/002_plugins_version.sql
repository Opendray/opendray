ALTER TABLE plugins ADD COLUMN IF NOT EXISTS version TEXT NOT NULL DEFAULT '0.0.0';
ALTER TABLE plugins ADD COLUMN IF NOT EXISTS manifest JSONB NOT NULL DEFAULT '{}'::jsonb;

-- Drop old columns that no longer exist in new schema
ALTER TABLE plugins DROP COLUMN IF EXISTS base_url;
ALTER TABLE plugins DROP COLUMN IF EXISTS route_prefix;
ALTER TABLE plugins DROP COLUMN IF EXISTS health_endpoint;

-- Rollback migration: Remove slug and aliases columns

BEGIN;

DROP INDEX IF EXISTS idx_pipeline_slug;

ALTER TABLE pipeline
DROP COLUMN IF EXISTS aliases;

ALTER TABLE pipeline
DROP COLUMN IF EXISTS slug;

COMMIT;

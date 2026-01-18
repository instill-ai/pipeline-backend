-- Rollback migration: Remove slug, aliases, and display_name columns from pipeline_release table

BEGIN;

DROP INDEX IF EXISTS idx_pipeline_release_slug;
ALTER TABLE pipeline_release DROP COLUMN IF EXISTS slug;
ALTER TABLE pipeline_release DROP COLUMN IF EXISTS aliases;
ALTER TABLE pipeline_release DROP COLUMN IF EXISTS display_name;

COMMIT;

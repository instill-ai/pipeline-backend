-- Rollback: Revert pipeline_release ID to version-based format
-- Note: This is a best-effort rollback. The original version-based IDs
-- are preserved in the slug column.

BEGIN;

-- Drop the unique constraint
DROP INDEX IF EXISTS idx_pipeline_release_pipeline_uid_id_unique;

-- Restore id from slug (best effort - original version ID was stored there)
UPDATE pipeline_release
SET id = slug
WHERE slug IS NOT NULL AND slug != '';

COMMIT;

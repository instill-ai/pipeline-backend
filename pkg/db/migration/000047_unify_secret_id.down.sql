-- Rollback: Restore old id values from slug
-- Note: This is a best-effort rollback. The original id values may not be fully recoverable
-- if slug was already populated before migration.
BEGIN;

-- Drop the unique constraint added in up migration
DROP INDEX IF EXISTS idx_secret_namespace_id;

-- Restore id from slug (best effort)
UPDATE secret
SET id = slug
WHERE slug IS NOT NULL AND slug != '';

COMMIT;

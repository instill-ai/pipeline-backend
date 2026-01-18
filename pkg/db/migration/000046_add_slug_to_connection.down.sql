-- Migration: Remove slug, aliases, display_name, and description columns from connection table
BEGIN;
DROP INDEX IF EXISTS idx_connection_slug;
ALTER TABLE connection DROP COLUMN IF EXISTS slug;
ALTER TABLE connection DROP COLUMN IF EXISTS aliases;
ALTER TABLE connection DROP COLUMN IF EXISTS display_name;
ALTER TABLE connection DROP COLUMN IF EXISTS description;
COMMIT;

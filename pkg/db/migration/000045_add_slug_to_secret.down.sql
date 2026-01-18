-- Migration: Remove slug, aliases, and display_name columns from secret table
BEGIN;
DROP INDEX IF EXISTS idx_secret_slug;
ALTER TABLE secret DROP COLUMN IF EXISTS slug;
ALTER TABLE secret DROP COLUMN IF EXISTS aliases;
ALTER TABLE secret DROP COLUMN IF EXISTS display_name;
COMMIT;

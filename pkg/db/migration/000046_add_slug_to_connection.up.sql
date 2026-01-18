-- Migration: Add slug, aliases, display_name, and description columns to connection table for AIP standard
BEGIN;
-- Add slug column to connection table
ALTER TABLE connection
ADD COLUMN IF NOT EXISTS slug VARCHAR(255);
-- Add aliases column to connection table (stores previous slugs)
ALTER TABLE connection
ADD COLUMN IF NOT EXISTS aliases TEXT [];
-- Add display_name column if not exists
ALTER TABLE connection
ADD COLUMN IF NOT EXISTS display_name VARCHAR(255);
-- Add description column if not exists
ALTER TABLE connection
ADD COLUMN IF NOT EXISTS description TEXT;
-- Populate display_name from id for existing records if empty
UPDATE connection
SET display_name = id
WHERE display_name IS NULL
    OR display_name = '';
-- Generate slug from display_name for existing connection records
UPDATE connection
SET slug = LOWER(
        REGEXP_REPLACE(
            REGEXP_REPLACE(
                REGEXP_REPLACE(
                    COALESCE(display_name, id),
                    '[^a-zA-Z0-9\s-]',
                    '',
                    'g'
                ),
                '\s+',
                '-',
                'g'
            ),
            '-+',
            '-',
            'g'
        )
    )
WHERE slug IS NULL
    OR slug = '';
-- Create index for slug lookups
CREATE INDEX IF NOT EXISTS idx_connection_slug ON connection(slug);
COMMIT;

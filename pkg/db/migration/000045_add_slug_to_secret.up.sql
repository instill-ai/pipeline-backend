-- Migration: Add slug, aliases, and display_name columns to secret table for AIP standard
BEGIN;
-- Add slug column to secret table
ALTER TABLE secret
ADD COLUMN IF NOT EXISTS slug VARCHAR(255);
-- Add aliases column to secret table (stores previous slugs)
ALTER TABLE secret
ADD COLUMN IF NOT EXISTS aliases TEXT [];
-- Add display_name column if not exists
ALTER TABLE secret
ADD COLUMN IF NOT EXISTS display_name VARCHAR(255);
-- Populate display_name from id for existing records if empty
UPDATE secret
SET display_name = id
WHERE display_name IS NULL
    OR display_name = '';
-- Generate slug from display_name for existing secret records
UPDATE secret
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
CREATE INDEX IF NOT EXISTS idx_secret_slug ON secret(slug);
COMMIT;

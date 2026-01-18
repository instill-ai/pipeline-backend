-- Migration: Add slug, aliases, and display_name columns to pipeline_release table for AIP standard
BEGIN;
-- Add slug column to pipeline_release table
ALTER TABLE pipeline_release
ADD COLUMN IF NOT EXISTS slug VARCHAR(255);
-- Add aliases column to pipeline_release table (stores previous slugs)
ALTER TABLE pipeline_release
ADD COLUMN IF NOT EXISTS aliases TEXT [];
-- Add display_name column if not exists
ALTER TABLE pipeline_release
ADD COLUMN IF NOT EXISTS display_name VARCHAR(255);
-- Populate display_name from id for existing records if empty
UPDATE pipeline_release
SET display_name = id
WHERE display_name IS NULL
    OR display_name = '';
-- Generate slug from display_name for existing pipeline_release records
UPDATE pipeline_release
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
CREATE INDEX IF NOT EXISTS idx_pipeline_release_slug ON pipeline_release(slug);
COMMIT;

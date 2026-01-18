-- Migration: Unify pipeline ID to AIP standard (hash-based canonical ID)
-- This migration converts the user-provided ID to a hash-based canonical ID
-- and preserves the original ID in the slug column for backward compatibility.

-- Ensure pgcrypto extension is available for digest function
CREATE EXTENSION IF NOT EXISTS pgcrypto;

BEGIN;

-- Step 1: Ensure slug is populated from the old user-provided ID if empty
UPDATE pipeline
SET slug = LOWER(
    REGEXP_REPLACE(
        REGEXP_REPLACE(
            REGEXP_REPLACE(
                COALESCE(id, ''),
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
WHERE slug IS NULL OR slug = '';

-- Step 2: Ensure display_name is populated from the old user-provided ID if empty
UPDATE pipeline
SET display_name = id
WHERE (display_name IS NULL OR display_name = '') AND id IS NOT NULL AND id != '';

-- Step 3: Create a function to generate hash-based canonical ID
-- This matches the Go implementation: prefix-{base62(sha256(uid)[:10])}
CREATE OR REPLACE FUNCTION generate_pipeline_canonical_id(uid UUID)
RETURNS TEXT AS $$
DECLARE
    hash_bytes BYTEA;
    result TEXT := '';
    i INT;
    num INT;
    base62_chars TEXT := '0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz';
BEGIN
    -- Get SHA-256 hash of the UID string
    hash_bytes := digest(uid::TEXT, 'sha256');

    -- Take first 10 bytes and encode to base62
    FOR i IN 0..4 LOOP
        -- Combine 2 bytes into a 16-bit number
        num := (get_byte(hash_bytes, i*2) << 8) | get_byte(hash_bytes, i*2 + 1);
        -- Encode to 2 base62 characters
        result := result || substr(base62_chars, (num % 62) + 1, 1);
        num := num / 62;
        result := result || substr(base62_chars, (num % 62) + 1, 1);
    END LOOP;

    RETURN 'pip-' || result;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Step 4: Update the id column with hash-based canonical IDs
-- Only update records that don't already have hash-based IDs
UPDATE pipeline
SET id = generate_pipeline_canonical_id(uid)
WHERE id IS NULL OR id = '' OR id NOT LIKE 'pip-%';

-- Step 5: Add unique constraint on (owner, id) for canonical IDs
-- First drop any existing constraint if it exists
ALTER TABLE pipeline DROP CONSTRAINT IF EXISTS pipeline_owner_id_unique;
CREATE UNIQUE INDEX IF NOT EXISTS idx_pipeline_owner_id_unique ON pipeline(owner, id);

-- Step 6: Clean up the helper function
DROP FUNCTION IF EXISTS generate_pipeline_canonical_id(UUID);

COMMIT;

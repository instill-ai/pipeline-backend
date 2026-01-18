-- Migration: Unify connection schema to use hash-based canonical ID
-- This migration converts the user-provided `id` to use the AIP standard:
-- - The old user-provided `id` values are moved to `slug` (if slug is empty)
-- - The old user-provided `id` values are moved to `display_name` (if display_name is empty)
-- - The `id` column is updated with hash-based canonical IDs generated from UID

-- Ensure pgcrypto extension is available for digest function
CREATE EXTENSION IF NOT EXISTS pgcrypto;

BEGIN;

-- Step 1: Populate slug from old id if slug is empty
-- The old id was user-provided and works well as a slug
UPDATE connection
SET slug = id
WHERE (slug IS NULL OR slug = '')
  AND id IS NOT NULL
  AND id != '';

-- Step 2: Populate display_name from old id if display_name is empty
-- The old id was user-provided and can serve as a display name
UPDATE connection
SET display_name = id
WHERE (display_name IS NULL OR display_name = '')
  AND id IS NOT NULL
  AND id != '';

-- Step 3: Create a function to generate base62 encoded hash from UUID
-- This matches the Go implementation in utils.GeneratePrefixedResourceID
CREATE OR REPLACE FUNCTION generate_connection_canonical_id(uid UUID) RETURNS VARCHAR AS $$
DECLARE
    hash_bytes BYTEA;
    base62_chars TEXT := '0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz';
    result TEXT := '';
    byte_val INT;
    i INT;
BEGIN
    -- SHA256 hash of the UUID string, take first 10 bytes (80 bits)
    hash_bytes := substring(digest(uid::text, 'sha256') from 1 for 10);

    -- Convert each byte to base62
    FOR i IN 1..10 LOOP
        byte_val := get_byte(hash_bytes, i - 1);
        IF byte_val = 0 THEN
            result := result || substr(base62_chars, 1, 1);
        ELSE
            WHILE byte_val > 0 LOOP
                result := result || substr(base62_chars, (byte_val % 62) + 1, 1);
                byte_val := byte_val / 62;
            END LOOP;
        END IF;
    END LOOP;

    RETURN 'con-' || result;
END;
$$ LANGUAGE plpgsql;

-- Step 4: Update id column with hash-based canonical IDs
UPDATE connection
SET id = generate_connection_canonical_id(uid);

-- Step 5: Add unique constraint on (namespace_uid, id) if not exists
CREATE UNIQUE INDEX IF NOT EXISTS idx_connection_namespace_id ON connection(namespace_uid, id);

-- Step 6: Drop the helper function
DROP FUNCTION IF EXISTS generate_connection_canonical_id(UUID);

COMMIT;

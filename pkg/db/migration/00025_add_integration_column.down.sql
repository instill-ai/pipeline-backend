BEGIN;

DROP INDEX IF EXISTS integration_index;
ALTER TABLE component_definition_index
  DROP COLUMN IF EXISTS has_integration,
  DROP COLUMN IF EXISTS vendor;

COMMIT;

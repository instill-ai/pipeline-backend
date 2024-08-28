BEGIN;

ALTER TABLE component_definition_index
  ADD COLUMN IF NOT EXISTS has_integration BOOL DEFAULT FALSE NOT NULL,
  ADD COLUMN IF NOT EXISTS vendor VARCHAR(255) DEFAULT '' NOT NULL;

CREATE INDEX IF NOT EXISTS integration_index ON component_definition_index (feature_score DESC, uid DESC) WHERE is_visible IS TRUE AND has_integration IS TRUE;

COMMIT;

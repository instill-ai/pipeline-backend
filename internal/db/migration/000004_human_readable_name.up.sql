BEGIN;

CREATE TABLE IF NOT EXISTS pipelines_migration AS TABLE pipelines;

ALTER TABLE pipelines DROP COLUMN IF EXISTS ext_id;
ALTER TABLE pipelines DROP COLUMN IF EXISTS creator_id;

ALTER TABLE pipelines ADD namespace varchar(39) NOT NULL DEFAULT 'undefined';
COMMENT ON COLUMN pipelines.namespace IS 'namespace in which the pipeline belongs to';

ALTER TABLE pipelines ADD CONSTRAINT unique_name_namespace UNIQUE (name, deleted_at, namespace);

COMMIT;

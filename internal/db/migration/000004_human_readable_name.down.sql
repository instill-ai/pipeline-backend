BEGIN;

ALTER TABLE pipelines DROP CONSTRAINT unique_name_namespace;

ALTER TABLE pipelines DROP COLUMN IF EXISTS namespace;

ALTER TABLE pipelines ADD ext_id varchar(20) NOT NULL DEFAULT '000004';
COMMENT ON COLUMN pipelines.ext_id IS 'the hash of the id';

ALTER TABLE pipelines ADD creator_id varchar(36) NOT NULL DEFAULT '000004';

UPDATE pipelines
SET ext_id = migration.ext_id, creator_id = migration.creator_id
FROM pipelines_migration AS migration
WHERE pipelines.id = migration.id;

DROP TABLE IF EXISTS pipelines_migration;

COMMIT;

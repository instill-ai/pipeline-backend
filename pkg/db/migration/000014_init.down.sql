BEGIN;

DROP TABLE IF EXISTS tag;
DROP INDEX IF EXISTS tag_pipeline_uid;
DROP INDEX IF EXISTS tag_tag_name;
DROP INDEX IF EXISTS tag_unique_pipeline_tag;

COMMIT;

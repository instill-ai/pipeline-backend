BEGIN;

ALTER TABLE pipeline DROP COLUMN IF EXISTS template_overrides;
ALTER TABLE pipeline DROP COLUMN IF EXISTS template_pipeline_release_uid;
ALTER TABLE pipeline DROP COLUMN IF EXISTS template_pipeline_uid;
ALTER TABLE pipeline DROP COLUMN IF EXISTS use_template;

COMMIT;

BEGIN;

ALTER TABLE component_run DROP COLUMN IF EXISTS total_duration;
ALTER TABLE pipeline_run DROP COLUMN IF EXISTS total_duration;

COMMIT;

BEGIN;

ALTER TABLE component_run DROP COLUMN IF EXISTS blob_data_expiration_time;
ALTER TABLE pipeline_run DROP COLUMN IF EXISTS blob_data_expiration_time;

COMMIT;

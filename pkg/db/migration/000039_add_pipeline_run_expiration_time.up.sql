BEGIN;

ALTER TABLE pipeline_run ADD COLUMN IF NOT EXISTS blob_data_expiration_time TIMESTAMPTZ;
ALTER TABLE component_run ADD COLUMN IF NOT EXISTS blob_data_expiration_time TIMESTAMPTZ;

COMMIT;

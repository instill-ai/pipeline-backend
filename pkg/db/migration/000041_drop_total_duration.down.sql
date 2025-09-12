BEGIN

ALTER TABLE pipeline_run ADD COLUMN IF NOT EXISTS total_duration BIGINT DEFAULT 0;
COMMENT ON COLUMN pipeline_run.total_duration IS 'in milliseconds';

ALTER TABLE component_run ADD COLUMN IF NOT EXISTS total_duration BIGINT DEFAULT 0;
COMMENT ON COLUMN component_run.total_duration IS 'in milliseconds';

COMMIT;

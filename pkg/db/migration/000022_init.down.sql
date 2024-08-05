BEGIN;

-- Drop indexes
DROP INDEX IF EXISTS idx_component_runs_completed_time;
DROP INDEX IF EXISTS idx_component_runs_started_time;
DROP INDEX IF EXISTS idx_component_runs_status;
DROP INDEX IF EXISTS idx_pipeline_runs_namespace;
DROP INDEX IF EXISTS idx_pipeline_runs_completed_time;
DROP INDEX IF EXISTS idx_pipeline_runs_started_time;
DROP INDEX IF EXISTS idx_pipeline_runs_triggered_time;
DROP INDEX IF EXISTS idx_pipeline_runs_status;
DROP INDEX IF EXISTS idx_pipeline_runs_pipeline_uid;

-- Drop foreign key constraint
ALTER TABLE IF EXISTS component_runs
DROP CONSTRAINT IF EXISTS fk_component_runs_pipeline_run;

-- Drop tables
DROP TABLE IF EXISTS component_runs;
DROP TABLE IF EXISTS pipeline_runs;

COMMIT;

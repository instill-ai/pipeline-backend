-- Drop indexes
DROP INDEX IF EXISTS idx_run_components_run_uid_component_id;
DROP INDEX IF EXISTS idx_pipeline_runs_pipeline_uid_triggered_time;
DROP INDEX IF EXISTS idx_pipeline_runs_pipeline_uid_status;

DROP INDEX IF EXISTS idx_run_components_completed_time;
DROP INDEX IF EXISTS idx_run_components_started_time;
DROP INDEX IF EXISTS idx_run_components_status;
DROP INDEX IF EXISTS idx_run_components_component_id;
DROP INDEX IF EXISTS idx_run_components_run_uid;

DROP INDEX IF EXISTS idx_pipeline_runs_completed_time;
DROP INDEX IF EXISTS idx_pipeline_runs_started_time;
DROP INDEX IF EXISTS idx_pipeline_runs_triggered_time;
DROP INDEX IF EXISTS idx_pipeline_runs_status;
DROP INDEX IF EXISTS idx_pipeline_runs_pipeline_uid;

-- Drop tables
DROP TABLE IF EXISTS run_components;
DROP TABLE IF EXISTS pipeline_runs;

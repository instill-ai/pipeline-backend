BEGIN;

CREATE TABLE IF NOT EXISTS pipeline_runs (
    pipeline_uid UUID NOT NULL,
    pipeline_trigger_uid UUID PRIMARY KEY,
    pipeline_version VARCHAR(255),
    status VARCHAR(50),
    source VARCHAR(50),
    total_duration BIGINT,
    triggered_by VARCHAR(255),
    namespace VARCHAR(255),
    inputs JSONB,
    outputs JSONB,
    recipe_snapshot JSONB,
    triggered_time TIMESTAMP WITH TIME ZONE NOT NULL,
    started_time TIMESTAMP WITH TIME ZONE,
    completed_time TIMESTAMP WITH TIME ZONE,
    error_msg TEXT
);

CREATE TABLE IF NOT EXISTS component_runs (
    pipeline_trigger_uid UUID NOT NULL,
    component_id VARCHAR(255) NOT NULL,
    status VARCHAR(50),
    total_duration BIGINT,
    started_time TIMESTAMP WITH TIME ZONE,
    completed_time TIMESTAMP WITH TIME ZONE,
    error_msg TEXT,
    inputs JSONB,
    outputs JSONB,
    PRIMARY KEY (pipeline_trigger_uid, component_id)
);

CREATE INDEX IF NOT EXISTS idx_pipeline_runs_pipeline_uid ON pipeline_runs(pipeline_uid);
CREATE INDEX IF NOT EXISTS idx_pipeline_runs_status ON pipeline_runs(status);
CREATE INDEX IF NOT EXISTS idx_pipeline_runs_triggered_time ON pipeline_runs(triggered_time);
CREATE INDEX IF NOT EXISTS idx_pipeline_runs_started_time ON pipeline_runs(started_time);
CREATE INDEX IF NOT EXISTS idx_pipeline_runs_completed_time ON pipeline_runs(completed_time);
CREATE INDEX IF NOT EXISTS idx_pipeline_runs_namespace ON pipeline_runs(namespace);  -- Add this line
CREATE INDEX IF NOT EXISTS idx_component_runs_status ON component_runs(status);
CREATE INDEX IF NOT EXISTS idx_component_runs_started_time ON component_runs(started_time);
CREATE INDEX IF NOT EXISTS idx_component_runs_completed_time ON component_runs(completed_time);

-- Add foreign key constraint
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'fk_component_runs_pipeline_run'
        AND table_name = 'component_runs'
    ) THEN
        ALTER TABLE component_runs
        ADD CONSTRAINT fk_component_runs_pipeline_run
        FOREIGN KEY (pipeline_trigger_uid) REFERENCES pipeline_runs(pipeline_trigger_uid);
    END IF;
END $$;

COMMIT;
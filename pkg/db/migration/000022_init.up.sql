-- Create PipelineRun table
CREATE TABLE pipeline_runs (
    uid UUID PRIMARY KEY,
    pipeline_uid UUID NOT NULL,
    pipeline_version VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    source VARCHAR(50) NOT NULL,
    total_duration BIGINT NOT NULL,
    triggered_by VARCHAR(255) NOT NULL,
    credits INT NOT NULL,
    inputs JSONB,
    outputs JSONB,
    recipe_snapshot JSONB,
    triggered_time TIMESTAMP WITH TIME ZONE NOT NULL,
    started_time TIMESTAMP WITH TIME ZONE,
    completed_time TIMESTAMP WITH TIME ZONE,
    create_time TIMESTAMP WITH TIME ZONE NOT NULL,
    update_time TIMESTAMP WITH TIME ZONE NOT NULL,
    delete_time TIMESTAMP WITH TIME ZONE,
    error_msg TEXT,
    CONSTRAINT fk_pipeline_runs_pipeline FOREIGN KEY (pipeline_uid) REFERENCES pipeline(uid)
);

-- Create RunComponent table
CREATE TABLE run_components (
    uid UUID PRIMARY KEY,
    run_uid UUID NOT NULL,
    component_id VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    total_duration BIGINT NOT NULL,
    started_time TIMESTAMP WITH TIME ZONE NOT NULL,
    completed_time TIMESTAMP WITH TIME ZONE NOT NULL,
    credits INT NOT NULL,
    error_msg TEXT,
    inputs JSONB,
    outputs JSONB,
    create_time TIMESTAMP WITH TIME ZONE NOT NULL,
    update_time TIMESTAMP WITH TIME ZONE NOT NULL,
    delete_time TIMESTAMP WITH TIME ZONE,
    CONSTRAINT fk_run_components_pipeline_run FOREIGN KEY (run_uid) REFERENCES pipeline_runs(uid)
);

-- Create indexes for efficient lookups

-- PipelineRun indexes
CREATE INDEX idx_pipeline_runs_pipeline_uid ON pipeline_runs(pipeline_uid);
CREATE INDEX idx_pipeline_runs_status ON pipeline_runs(status);
CREATE INDEX idx_pipeline_runs_triggered_time ON pipeline_runs(triggered_time);
CREATE INDEX idx_pipeline_runs_started_time ON pipeline_runs(started_time);
CREATE INDEX idx_pipeline_runs_completed_time ON pipeline_runs(completed_time);

-- RunComponent indexes
CREATE INDEX idx_run_components_run_uid ON run_components(run_uid);
CREATE INDEX idx_run_components_component_id ON run_components(component_id);
CREATE INDEX idx_run_components_status ON run_components(status);
CREATE INDEX idx_run_components_started_time ON run_components(started_time);
CREATE INDEX idx_run_components_completed_time ON run_components(completed_time);

-- Composite indexes for common query patterns
CREATE INDEX idx_pipeline_runs_pipeline_uid_status ON pipeline_runs(pipeline_uid, status);
CREATE INDEX idx_pipeline_runs_pipeline_uid_triggered_time ON pipeline_runs(pipeline_uid, triggered_time);
CREATE INDEX idx_run_components_run_uid_component_id ON run_components(run_uid, component_id);

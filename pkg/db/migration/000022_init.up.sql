-- Create PipelineRun table
CREATE TABLE IF NOT EXISTS pipeline_runs (
                               uid UUID PRIMARY KEY,
                               pipeline_uid UUID NOT NULL,
                               pipeline_version VARCHAR(255) NOT NULL,
                               status VARCHAR(50) NOT NULL,
                               source VARCHAR(50) NOT NULL,
                               total_duration BIGINT NOT NULL,
                               triggered_by VARCHAR(255) NOT NULL,
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

COMMENT ON COLUMN pipeline_runs.inputs IS 'Array of FileReference: [{name: string, type: string, size: bigint, url: string}]';
COMMENT ON COLUMN pipeline_runs.outputs IS 'Array of FileReference: [{name: string, type: string, size: bigint, url: string}]';

-- Create RunComponent table
CREATE TABLE IF NOT EXISTS run_components (
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

COMMENT ON COLUMN run_components.inputs IS 'Array of FileReference: [{name: string, type: string, size: bigint, url: string}]';
COMMENT ON COLUMN run_components.outputs IS 'Array of FileReference: [{name: string, type: string, size: bigint, url: string}]';

-- The rest of your CREATE INDEX statements remain unchanged
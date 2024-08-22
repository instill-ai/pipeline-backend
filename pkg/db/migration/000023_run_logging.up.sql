BEGIN;

create type valid_trigger_status as enum ('RUN_STATUS_COMPLETED', 'RUN_STATUS_FAILED', 'RUN_STATUS_PROCESSING', 'RUN_STATUS_QUEUED');
create type valid_trigger_source as enum ('RUN_SOURCE_CONSOLE', 'RUN_SOURCE_API');

CREATE TABLE IF NOT EXISTS pipeline_run
(
    pipeline_trigger_uid UUID PRIMARY KEY,
    pipeline_uid         UUID                     NOT NULL,
    pipeline_version     VARCHAR(255)             NOT NULL,
    status               valid_trigger_status     NOT NULL,
    source               valid_trigger_source     NOT NULL,
    total_duration       BIGINT,
    triggered_by         VARCHAR(255)             NOT NULL,
    inputs               JSONB,
    outputs              JSONB,
    recipe_snapshot      JSONB,
    started_time         TIMESTAMP WITH TIME ZONE NOT NULL,
    completed_time       TIMESTAMP WITH TIME ZONE,
    error                TEXT
);

comment on column pipeline_run.total_duration is 'in milliseconds';

CREATE TABLE IF NOT EXISTS component_run
(
    pipeline_trigger_uid UUID                     NOT NULL,
    component_id         VARCHAR(255)             NOT NULL,
    status               valid_trigger_status     NOT NULL,
    total_duration       BIGINT,
    started_time         TIMESTAMP WITH TIME ZONE NOT NULL,
    completed_time       TIMESTAMP WITH TIME ZONE,
    error                TEXT,
    inputs               JSONB,
    outputs              JSONB,
    PRIMARY KEY (pipeline_trigger_uid, component_id)
);

comment on column component_run.total_duration is 'in milliseconds';

CREATE INDEX IF NOT EXISTS idx_pipeline_run_pipeline_uid ON pipeline_run (pipeline_uid);
CREATE INDEX IF NOT EXISTS idx_pipeline_run_status ON pipeline_run (status);
CREATE INDEX IF NOT EXISTS idx_pipeline_run_started_time ON pipeline_run (started_time);
CREATE INDEX IF NOT EXISTS idx_pipeline_run_completed_time ON pipeline_run (completed_time);

CREATE INDEX IF NOT EXISTS idx_component_run_status ON component_run (status);
CREATE INDEX IF NOT EXISTS idx_component_run_started_time ON component_run (started_time);
CREATE INDEX IF NOT EXISTS idx_component_run_completed_time ON component_run (completed_time);

-- Add foreign key constraint
DO
$$
    BEGIN
        IF NOT EXISTS (SELECT 1
                       FROM information_schema.table_constraints
                       WHERE constraint_name = 'fk_component_run_pipeline_run'
                         AND table_name = 'component_run') THEN
            ALTER TABLE component_run
                ADD CONSTRAINT fk_component_run_pipeline_run
                    FOREIGN KEY (pipeline_trigger_uid) REFERENCES pipeline_run (pipeline_trigger_uid);
        END IF;
    END
$$;

COMMIT;

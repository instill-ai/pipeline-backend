BEGIN;

CREATE TABLE IF NOT EXISTS public.pipeline_run_on (
  uid UUID NOT NULL,
  pipeline_uid UUID NOT NULL,
  release_uid UUID NOT NULL,
  event_id VARCHAR(255) NOT NULL,
  run_on_type VARCHAR(255) NOT NULL,
  identifier JSONB NOT NULL,
  create_time TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
  update_time TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
  delete_time TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NULL,
  CONSTRAINT pipeline_run_on_pkey PRIMARY KEY (uid)
);

CREATE INDEX pipeline_run_on_pipeline_uid_release_uid_create_time_pagination ON public.pipeline_run_on (pipeline_uid, release_uid, create_time);
CREATE INDEX pipeline_run_on_run_on_type_identifier_pagination ON public.pipeline_run_on (run_on_type, identifier) WHERE delete_time IS NULL;
CREATE UNIQUE INDEX pipeline_run_on_pipeline_uid_release_uid_event_id_run_on_type ON public.pipeline_run_on (pipeline_uid, release_uid, event_id, run_on_type, identifier) WHERE delete_time IS NULL;

COMMIT;

BEGIN;

CREATE TABLE IF NOT EXISTS public.pipeline_release (
  uid UUID NOT NULL,
  id VARCHAR(255) NOT NULL,
  description VARCHAR(1023) NULL,
  pipeline_uid UUID NOT NULL,
  recipe JSONB NOT NULL,
  create_time TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
  update_time TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
  delete_time TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NULL,
  CONSTRAINT pipeline_release_pkey PRIMARY KEY (uid)
);
CREATE UNIQUE INDEX unique_pipeline_uid_id_delete_time ON public.pipeline_release (pipeline_uid, id) WHERE delete_time IS NULL;
CREATE INDEX release_uid_create_time_pagination ON public.pipeline_release (uid, create_time);

ALTER TABLE public.pipeline ADD COLUMN "default_release_uid" UUID DEFAULT NULL;

COMMIT;

BEGIN;

CREATE TABLE IF NOT EXISTS public.tag(
    uid UUID NOT NULL,
    id VARCHAR(255) NOT NULL,
    pipeline_uid UUID NOT NULL,
    tag_name VARCHAR(255) NOT NULL
    create_time TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    update_time TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    delete_time TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NULL,
);


CREATE INDEX tag_uid_create_time_pagination ON public.tag(uid, create_time)
CREATE INDEX tag_name on public.tag(tag_name)
CREATE UNIQUE INDEX unique_pipeline_tag ON public.tag(pipeline_uid, tag_name) WHERE delete_time IS NULL;

COMMIT;
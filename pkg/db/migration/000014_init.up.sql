BEGIN;

CREATE TABLE IF NOT EXISTS public.tag(
    pipeline_uid UUID NOT NULL,
    tag_name VARCHAR(255) NOT NULL,
    create_time TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    update_time TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL
);


CREATE INDEX tag_pipeline_uid ON public.tag(pipeline_uid);
CREATE INDEX tag_tag_name on public.tag(tag_name);
CREATE UNIQUE INDEX tag_unique_pipeline_tag ON public.tag(pipeline_uid, tag_name);

COMMIT;

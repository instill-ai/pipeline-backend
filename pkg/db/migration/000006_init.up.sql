BEGIN;

ALTER TABLE public.pipeline ADD COLUMN "metadata" JSONB DEFAULT '{}';
ALTER TABLE public.pipeline_release ADD COLUMN "metadata" JSONB DEFAULT '{}';

COMMIT;

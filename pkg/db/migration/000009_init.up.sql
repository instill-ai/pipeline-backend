BEGIN;

ALTER TABLE public.pipeline ADD COLUMN "readme" TEXT DEFAULT '';
ALTER TABLE public.pipeline_release ADD COLUMN "readme" TEXT DEFAULT '';

COMMIT;

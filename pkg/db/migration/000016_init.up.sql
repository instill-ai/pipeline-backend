BEGIN;

ALTER TABLE public.pipeline ADD COLUMN recipe_yaml TEXT DEFAULT '';
ALTER TABLE public.pipeline_release ADD COLUMN recipe_yaml TEXT DEFAULT '';
ALTER TABLE public.pipeline ALTER COLUMN recipe DROP NOT NULL;
ALTER TABLE public.pipeline_release ALTER COLUMN recipe DROP NOT NULL;

COMMIT;

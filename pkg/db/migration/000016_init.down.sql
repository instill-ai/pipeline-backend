BEGIN;

ALTER TABLE public.pipeline DROP COLUMN recipe_yaml;
ALTER TABLE public.pipeline_release DROP COLUMN recipe_yaml;
COMMIT;

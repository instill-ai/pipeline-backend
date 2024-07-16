BEGIN;

ALTER TABLE public.pipeline DROP COLUMN source_url;
ALTER TABLE public.pipeline DROP COLUMN documentation_url;
ALTER TABLE public.pipeline DROP COLUMN license;
ALTER TABLE public.pipeline DROP COLUMN profile_image;

UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'jq-filter:', 'jqFilter:');
UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'json-string:', 'jsonInput:');
UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'jq-filter:', 'jqFilter:');
UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'json-string:', 'jsonInput:');

COMMIT;

BEGIN;

ALTER TABLE public.pipeline ADD COLUMN IF NOT EXISTS source_url VARCHAR(255) DEFAULT '';
ALTER TABLE public.pipeline ADD COLUMN IF NOT EXISTS documentation_url VARCHAR(255) DEFAULT '';
ALTER TABLE public.pipeline ADD COLUMN IF NOT EXISTS license VARCHAR(255) DEFAULT '';

-- `profile_image` stores the profile image of the pipeline in base64 format.
ALTER TABLE public.pipeline ADD COLUMN IF NOT EXISTS profile_image TEXT DEFAULT NULL;

UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'jqFilter:', 'jq-filter:');
UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'jsonInput:', 'json-string:');
UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'jqFilter:', 'jq-filter:');
UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'jsonInput:', 'json-string:');

COMMIT;

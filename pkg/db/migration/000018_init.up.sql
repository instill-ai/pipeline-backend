BEGIN;

ALTER TABLE public.pipeline ADD COLUMN IF NOT EXISTS source_url VARCHAR(255) DEFAULT '';
ALTER TABLE public.pipeline ADD COLUMN IF NOT EXISTS documentation_url VARCHAR(255) DEFAULT '';
ALTER TABLE public.pipeline ADD COLUMN IF NOT EXISTS license VARCHAR(255) DEFAULT '';

-- `profile_image` stores the profile image of the pipeline in base64 format.
ALTER TABLE public.pipeline ADD COLUMN IF NOT EXISTS profile_image TEXT DEFAULT NULL;

COMMIT;

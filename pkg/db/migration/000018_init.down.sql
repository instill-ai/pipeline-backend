BEGIN;

ALTER TABLE public.pipeline DROP COLUMN source_url;
ALTER TABLE public.pipeline DROP COLUMN documentation_url;
ALTER TABLE public.pipeline DROP COLUMN license;
ALTER TABLE public.pipeline DROP COLUMN profile_image;

COMMIT;

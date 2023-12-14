BEGIN;

ALTER TABLE public.pipeline RENAME COLUMN permission TO sharing;

COMMIT;

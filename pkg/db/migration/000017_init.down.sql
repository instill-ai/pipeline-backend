BEGIN;

ALTER TABLE public.pipeline DROP COLUMN number_of_clones;
DROP INDEX IF EXISTS pipeline_number_of_clones;

COMMIT;

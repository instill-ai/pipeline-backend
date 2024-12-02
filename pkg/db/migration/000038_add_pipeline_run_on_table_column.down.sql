BEGIN;

ALTER TABLE public.pipeline_run_on
DROP COLUMN config,
DROP COLUMN setup;

COMMIT;

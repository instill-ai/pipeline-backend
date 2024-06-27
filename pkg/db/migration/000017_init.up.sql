BEGIN;

ALTER TABLE public.pipeline ADD COLUMN number_of_clones INTEGER DEFAULT 0;
CREATE INDEX pipeline_number_of_clones ON public.pipeline (number_of_clones);

COMMIT;

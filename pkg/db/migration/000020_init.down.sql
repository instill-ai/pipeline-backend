BEGIN;

ALTER TABLE public.pipeline DROP COLUMN namespace_id;
ALTER TABLE public.pipeline DROP COLUMN namespace_type;
ALTER TABLE public.secret DROP COLUMN namespace_id;
ALTER TABLE public.secret DROP COLUMN namespace_type;

COMMIT;

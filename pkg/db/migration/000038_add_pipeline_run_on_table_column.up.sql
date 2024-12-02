BEGIN;

-- Store config and setup for pipeline run events to enable proper event unregistration with original settings
ALTER TABLE public.pipeline_run_on
ADD COLUMN config JSONB,
ADD COLUMN setup JSONB;

COMMIT;

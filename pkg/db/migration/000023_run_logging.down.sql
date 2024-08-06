BEGIN;

DROP TABLE IF EXISTS component_run;
DROP TABLE IF EXISTS pipeline_run;

DROP TYPE IF EXISTS valid_trigger_status;
DROP TYPE IF EXISTS valid_trigger_source;

COMMIT;

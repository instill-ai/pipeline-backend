BEGIN;

alter table pipeline_run
    drop column namespace;

COMMIT;
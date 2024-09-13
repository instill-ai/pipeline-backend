BEGIN;

alter table pipeline_run
    add namespace varchar(255) not null default '';

comment on column pipeline_run.namespace is 'run by namespace, which is the credit owner';

update pipeline_run
set namespace=triggered_by
where pipeline_run.namespace = '';

COMMIT;

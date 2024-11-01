BEGIN;

alter table pipeline_run
    rename column runner_uid to triggered_by;

alter table pipeline_run
    rename column requester_uid to namespace;

comment on column pipeline_run.namespace is 'run by namespace, which is the credit owner';

alter table pipeline_run
    alter column namespace type varchar(255) using ''::character varying;

alter table pipeline_run
    alter column triggered_by type varchar(255);

COMMIT;

BEGIN;

alter table pipeline_run
    rename column triggered_by to runner_uid;

alter table pipeline_run
    rename column namespace to requester_uid;

comment on column pipeline_run.requester_uid is null;

alter table pipeline_run
    alter column runner_uid type uuid using runner_uid::uuid;

alter table pipeline_run
    alter column requester_uid drop default;

alter table pipeline_run
    alter column requester_uid type uuid using requester_uid::uuid;

COMMIT;

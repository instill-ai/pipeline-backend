BEGIN;

ALTER TABLE pipelines ADD creator_id varchar(36) NOT NULL;

COMMIT;

BEGIN;

DROP TABLE IF EXISTS secret;
DROP INDEX IF EXISTS secret_unique_owner_id;
DROP INDEX IF EXISTS secret_uid_create_time_pagination;

COMMIT;

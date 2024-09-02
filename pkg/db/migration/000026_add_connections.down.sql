BEGIN;

DROP INDEX IF EXISTS unique_connection_id_namespace;
DROP TABLE IF EXISTS connection;
DROP TYPE valid_connection_method;

COMMIT;


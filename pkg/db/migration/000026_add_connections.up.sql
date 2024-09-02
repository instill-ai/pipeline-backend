BEGIN;

CREATE TYPE valid_connection_method AS ENUM (
  'METHOD_DICTIONARY',
  'METHOD_OAUTH'
);

CREATE TABLE IF NOT EXISTS connection (
  uid             UUID                    PRIMARY KEY,
  id              VARCHAR(255)            NOT NULL,
  namespace_uid   UUID                    NOT NULL,
  integration_uid UUID                    NOT NULL REFERENCES component_definition_index,
  method          valid_connection_method NOT NULL,
  setup           JSONB                   NOT NULL,
  create_time     TIMESTAMPTZ             NOT NULL DEFAULT CURRENT_TIMESTAMP,
  update_time     TIMESTAMPTZ             NOT NULL DEFAULT CURRENT_TIMESTAMP,
  delete_time     TIMESTAMPTZ
);

CREATE UNIQUE INDEX unique_connection_id_namespace ON connection (id, namespace_uid) WHERE delete_time IS NULL;

COMMIT;

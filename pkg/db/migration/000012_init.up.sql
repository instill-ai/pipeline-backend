BEGIN;

CREATE TYPE COMPONENT_DEFINITION_VALID_COMPONENT_TYPE AS ENUM (
  'COMPONENT_TYPE_UNSPECIFIED',
  'COMPONENT_TYPE_CONNECTOR_AI',
  'COMPONENT_TYPE_CONNECTOR_DATA',
  'COMPONENT_TYPE_OPERATOR',
  'COMPONENT_TYPE_CONNECTOR_APPLICATION'
);

CREATE TYPE VALID_RELEASE_STAGE AS ENUM (
  'RELEASE_STAGE_UNSPECIFIED',
  'RELEASE_STAGE_OPEN_FOR_CONTRIBUTION',
  'RELEASE_STAGE_COMING_SOON',
  'RELEASE_STAGE_ALPHA',
  'RELEASE_STAGE_BETA',
  'RELEASE_STAGE_GA'
);

/*
The source of truth for component definitions is in memory, extracted from the
different definitions.json files. We keep a table in the database that will be
synced with the inmem definitions on startup and that allows us to compute
pagination and filtering for component lists.
*/
CREATE TABLE IF NOT EXISTS component_definition_index (
  uid            UUID                                      PRIMARY KEY,
  id             VARCHAR(255)                              NOT NULL,
  title          VARCHAR(255)                              NOT NULL,
  component_type COMPONENT_DEFINITION_VALID_COMPONENT_TYPE DEFAULT 'COMPONENT_TYPE_UNSPECIFIED' NOT NULL,
  version        VARCHAR(255)                              NOT NULL,
  release_stage  VALID_RELEASE_STAGE                       DEFAULT 'RELEASE_STAGE_UNSPECIFIED' NOT NULL,
  -- is_visible is computed from a combination of fields (e.g. tombstone,
  -- public, deprecated), and is used to hide components from the list
  -- endpoint.
  is_visible     BOOL                                      NOT NULL,
  -- feature_score is used to position results in a page, i.e., to give more
  -- visibility to certain components.
  feature_score  INT                                       DEFAULT 0 NOT NULL
);

CREATE INDEX IF NOT EXISTS component_definition_index_filter ON component_definition_index (release_stage, component_type, feature_score DESC) WHERE is_visible IS TRUE;

COMMIT;

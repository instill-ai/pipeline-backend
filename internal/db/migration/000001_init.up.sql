BEGIN;

CREATE TABLE IF NOT EXISTS pipelines (
  id SERIAL PRIMARY KEY,
  ext_id varchar(20) NOT NULL,
  name varchar(256) NOT NULL,
  description text,
  enabled boolean NOT NULL DEFAULT (false),
  created_at timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
  updated_at timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
  deleted_at timestamp,
  recipe JSONB,
  crontab varchar(13)
);

COMMENT ON COLUMN pipelines.ext_id IS 'the hash of the id';
COMMENT ON COLUMN pipelines.name IS 'name of this pipeline';
COMMENT ON COLUMN pipelines.description IS 'description of this pipeline';
COMMENT ON COLUMN pipelines.enabled IS 'activate/deactivate pipeline';
COMMENT ON COLUMN pipelines.recipe IS 'pipeline configration';
COMMENT ON COLUMN pipelines.crontab IS 'the 6 * crontab format';

COMMIT;

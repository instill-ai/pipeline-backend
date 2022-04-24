BEGIN;

CREATE TYPE valid_status AS ENUM ('STATUS_INACTIVE', 'STATUS_ACTIVE', 'STATUS_ERROR');

ALTER TABLE pipeline RENAME COLUMN active TO status;

ALTER TABLE pipeline ALTER COLUMN status DROP DEFAULT;

ALTER TABLE pipeline ALTER COLUMN status TYPE valid_status
USING CASE WHEN status=FALSE THEN 'STATUS_INACTIVE'::valid_status ELSE 'STATUS_ACTIVE'::valid_status END;

ALTER TABLE pipeline ALTER COLUMN status SET DEFAULT 'STATUS_INACTIVE'::valid_status;

COMMENT ON COLUMN pipeline.status IS 'pipeline status';

COMMIT;

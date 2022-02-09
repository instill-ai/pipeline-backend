BEGIN;

ALTER TABLE "pipelines" ADD "creator_id" varchar(36) NOT NULL;
COMMENT ON COLUMN "pipelines"."creator_id" IS 'the creator (should be retired once Keto is set)';

COMMIT;
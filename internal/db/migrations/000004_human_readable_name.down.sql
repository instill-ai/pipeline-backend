BEGIN;

ALTER TABLE "pipelines" ALTER COLUMN "id" SET DATA TYPE INT;

ALTER TABLE "pipelines" DROP CONSTRAINT "unique_name_namespace";

ALTER TABLE "pipelines" DROP COLUMN IF EXISTS "namespace";

ALTER TABLE "pipelines" ADD "ext_id" varchar(20) NOT NULL DEFAULT '000004';
COMMENT ON COLUMN "pipelines"."ext_id" IS 'the hash of the id';

ALTER TABLE "pipelines" ADD "creator_id" varchar(36) NOT NULL DEFAULT '000004';
COMMENT ON COLUMN "pipelines"."creator_id" IS 'the creator (should be retired once Keto is set)';

UPDATE "pipelines"
SET "ext_id" = bak."ext_id", "creator_id" = bak."creator_id"
FROM "pipelines_bak" AS bak
WHERE "pipelines"."id" = bak."id";

DROP TABLE IF EXISTS "pipelines_bak";

COMMIT;

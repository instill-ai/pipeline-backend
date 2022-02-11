BEGIN;

CREATE TABLE IF NOT EXISTS "pipelines_bak" AS TABLE "pipelines";

ALTER TABLE "pipelines" DROP COLUMN IF EXISTS "ext_id";
ALTER TABLE "pipelines" DROP COLUMN IF EXISTS "creator_id";

ALTER TABLE "pipelines" ADD "namespace" varchar(39) NOT NULL DEFAULT 'undefined';
COMMENT ON COLUMN "pipelines"."namespace" IS 'a set of pipelines';

ALTER TABLE "pipelines" ADD CONSTRAINT "unique_name_namespace" UNIQUE ("name", "deleted_at", "namespace");

ALTER TABLE "pipelines" ALTER COLUMN "id" SET DATA TYPE BIGINT;

COMMIT;

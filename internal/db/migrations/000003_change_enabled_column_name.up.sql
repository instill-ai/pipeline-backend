BEGIN;

ALTER TABLE "pipelines" RENAME COLUMN "enabled" TO "active";

COMMIT;
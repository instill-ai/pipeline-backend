BEGIN;

ALTER TABLE "pipelines" RENAME COLUMN "active" TO "enabled";

COMMIT;

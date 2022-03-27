BEGIN;

ALTER TABLE "pipeline_history" DROP CONSTRAINT IF EXISTS "pipeline_history_fk_pipeline_id";

DROP TABLE IF EXISTS "pipeline_history";

DROP TABLE IF EXISTS "pipelines";

COMMIT;

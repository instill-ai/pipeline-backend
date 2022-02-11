BEGIN;

CREATE TABLE IF NOT EXISTS "pipelines" (
  "id" SERIAL PRIMARY KEY,
  "ext_id" varchar(20) NOT NULL,
  "name" varchar(256) NOT NULL,
  "description" text,
  "enabled" boolean NOT NULL DEFAULT (false),
  "created_at" timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
  "updated_at" timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
  "deleted_at" timestamp,
  "recipe" JSONB,
  "crontab" varchar(13)
);

CREATE TABLE IF NOT EXISTS "pipeline_history" (
  "pipeline_id" int NOT NULL,
  "recipe" JSONB NOT NULL,
  "version" int NOT NULL,
  "created_at" timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
  PRIMARY KEY ("pipeline_id", "version")
);

COMMENT ON COLUMN "pipelines"."ext_id" IS 'the hash of the id';
COMMENT ON COLUMN "pipelines"."name" IS 'name of this pipeline';
COMMENT ON COLUMN "pipelines"."description" IS 'description of this pipeline';
COMMENT ON COLUMN "pipelines"."enabled" IS 'activate/deactivate pipeline';
COMMENT ON COLUMN "pipelines"."recipe" IS 'describe what the pipeline looks like';
COMMENT ON COLUMN "pipelines"."crontab" IS 'the 6 * crontab format';

COMMENT ON COLUMN "pipeline_history"."recipe" IS 'describe what the pipeline looks like';

ALTER TABLE "pipeline_history" ADD CONSTRAINT "pipeline_history_fk_pipeline_id" FOREIGN KEY ("pipeline_id") REFERENCES "pipelines"("id");

COMMIT;

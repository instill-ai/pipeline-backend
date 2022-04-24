BEGIN;

ALTER TABLE pipelines RENAME TO pipeline;

ALTER TABLE pipeline RENAME CONSTRAINT pipelines_pkey TO pipeline_pkey;

COMMIT;

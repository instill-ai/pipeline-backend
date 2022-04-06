BEGIN;

ALTER TABLE pipeline RENAME TO pipelines;

ALTER TABLE pipelines RENAME CONSTRAINT pipeline_pkey TO pipelines_pkey;

COMMIT;

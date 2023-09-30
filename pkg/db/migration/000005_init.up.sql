BEGIN;

ALTER TABLE public.pipeline ADD COLUMN "permission" JSONB DEFAULT {};
ALTER TABLE public.pipeline ADD COLUMN "share_code" VARCHAR(255) DEFAULT '' NOT NULL;

COMMIT;

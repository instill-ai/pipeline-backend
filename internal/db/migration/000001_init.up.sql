BEGIN;
CREATE TYPE valid_status AS ENUM (
  'STATUS_UNSPECIFIED',
  'STATUS_INACTIVATED',
  'STATUS_ACTIVATED',
  'STATUS_ERROR'
);
CREATE TABLE IF NOT EXISTS public.pipeline (
  id UUID NOT NULL,
  owner_id UUID NOT NULL,
  name VARCHAR(255) NOT NULL,
  description VARCHAR(1023) NULL,
  recipe JSONB NOT NULL,
  status VALID_STATUS NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
  deleted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NULL,
  CONSTRAINT pipeline_pkey PRIMARY KEY (id)
);
ALTER TABLE public.pipeline
ADD CONSTRAINT unique_owner_id_name_deleted_at UNIQUE (owner_id, name, deleted_at);
CREATE INDEX pipeline_id_created_at_pagination ON public.pipeline (id, created_at);
COMMIT;

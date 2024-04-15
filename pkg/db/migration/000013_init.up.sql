BEGIN;
CREATE TABLE IF NOT EXISTS public.secret (
  uid UUID NOT NULL,
  id VARCHAR(255) NOT NULL,
  owner VARCHAR(255) NOT NULL,
  description VARCHAR(1023) NULL,
  value VARCHAR(1023) NULL,
  create_time TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
  update_time TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL
);
CREATE UNIQUE INDEX secret_unique_owner_id ON public.secret (owner, id);
CREATE INDEX secret_uid_create_time_pagination ON public.secret (uid, create_time);

COMMIT;

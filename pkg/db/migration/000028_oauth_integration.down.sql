BEGIN;

ALTER TABLE connection DROP COLUMN IF EXISTS o_auth_access_details;
ALTER TABLE connection DROP COLUMN IF EXISTS scopes;

COMMIT;

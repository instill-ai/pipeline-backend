BEGIN;

-- Add column to indicate if pipeline is using a template
ALTER TABLE pipeline ADD COLUMN IF NOT EXISTS use_template BOOLEAN DEFAULT FALSE;

-- Add column to store the UUID of the template pipeline being used
ALTER TABLE pipeline ADD COLUMN IF NOT EXISTS template_pipeline_uid UUID;

-- Add column to store the UUID of the specific template pipeline release being used
ALTER TABLE pipeline ADD COLUMN IF NOT EXISTS template_pipeline_release_uid UUID;

-- Add column to store JSON overrides for template customization
ALTER TABLE pipeline ADD COLUMN IF NOT EXISTS template_overrides JSONB;

COMMIT;

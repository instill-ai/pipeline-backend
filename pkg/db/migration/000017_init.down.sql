BEGIN;

ALTER TABLE public.pipeline DROP COLUMN number_of_clones;
DROP INDEX IF EXISTS pipeline_number_of_clones;

-- The commands are generated via
-- curl -s 'https://INSTILL-CLOUD-HOST/v1beta/component-definitions?pageSize=50' | jq -r ".componentDefinitions.[] | \"UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'type: \" + .id +\"', 'type: \"+ .uid +\"');\""

UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'type: instill-model', 'type: ddcf42c3-4c30-4c65-9585-25f1c89b2b48');
UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'type: openai', 'type: 9fb6a2cb-bff5-4c69-bc6d-4538dd8e3362');
UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'type: numbers', 'type: 70d8664a-d512-4517-a5e8-5d4da81756a7');
UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'type: pinecone', 'type: 4b1dcf82-e134-4ba7-992f-f9a02536ec2b');
UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'type: redis', 'type: fd0ad325-f2f7-41f3-b247-6c71d571b1b8');
UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'type: json', 'type: 28f53d15-6150-46e6-99aa-f76b70a926c0');
UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'type: image', 'type: e9eb8fc8-f249-4e11-ad50-5035d79ffc18');
UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'type: document', 'type: e5b290ae-ad53-47c9-a64e-efbc5358520b');
UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'type: archetype-ai', 'type: e414a1f8-5fdf-4292-b050-9f9176254a4b');
UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'type: bigquery', 'type: e2ffe076-ab2c-4e5e-9587-a613a6b1c146');
UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'type: stability-ai', 'type: c86a95cc-7d32-4e22-a290-8c699f6705a4');
UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'type: website', 'type: 98909958-db7d-4dfe-9858-7761904be17e');
UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'type: restapi', 'type: 5ee55a5c-6e30-4c7a-80e8-90165a729e0a');
UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'type: text', 'type: 5b7aca5b-1ae3-477f-bf60-d34e1c993c87');
UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'type: base64', 'type: 3a836447-c211-4134-9cc5-ad45e1cc467e');
UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'type: google-search', 'type: 2b1da686-878a-462c-b2c6-a9690199939c');
UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'type: gcs', 'type: 205cbeff-6f45-4abe-b0a8-cec1a310137f');
UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'type: slack', 'type: 1e9f469e-da5e-46eb-8a89-23466627e3b5');
UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'type: hugging-face', 'type: 0255ef87-33ce-4f88-b9db-8897f8c17233');

-- The commands are generated via
-- curl -s 'https://INSTILL-CLOUD-HOST/v1beta/component-definitions?pageSize=50' | jq -r ".componentDefinitions.[] | \"UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'type: \" + .id +\"', 'type: \"+ .uid +\"');\""

UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'type: instill-model', 'type: ddcf42c3-4c30-4c65-9585-25f1c89b2b48');
UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'type: openai', 'type: 9fb6a2cb-bff5-4c69-bc6d-4538dd8e3362');
UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'type: numbers', 'type: 70d8664a-d512-4517-a5e8-5d4da81756a7');
UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'type: pinecone', 'type: 4b1dcf82-e134-4ba7-992f-f9a02536ec2b');
UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'type: redis', 'type: fd0ad325-f2f7-41f3-b247-6c71d571b1b8');
UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'type: json', 'type: 28f53d15-6150-46e6-99aa-f76b70a926c0');
UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'type: image', 'type: e9eb8fc8-f249-4e11-ad50-5035d79ffc18');
UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'type: document', 'type: e5b290ae-ad53-47c9-a64e-efbc5358520b');
UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'type: archetype-ai', 'type: e414a1f8-5fdf-4292-b050-9f9176254a4b');
UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'type: bigquery', 'type: e2ffe076-ab2c-4e5e-9587-a613a6b1c146');
UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'type: stability-ai', 'type: c86a95cc-7d32-4e22-a290-8c699f6705a4');
UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'type: website', 'type: 98909958-db7d-4dfe-9858-7761904be17e');
UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'type: restapi', 'type: 5ee55a5c-6e30-4c7a-80e8-90165a729e0a');
UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'type: text', 'type: 5b7aca5b-1ae3-477f-bf60-d34e1c993c87');
UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'type: base64', 'type: 3a836447-c211-4134-9cc5-ad45e1cc467e');
UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'type: google-search', 'type: 2b1da686-878a-462c-b2c6-a9690199939c');
UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'type: gcs', 'type: 205cbeff-6f45-4abe-b0a8-cec1a310137f');
UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'type: slack', 'type: 1e9f469e-da5e-46eb-8a89-23466627e3b5');
UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'type: hugging-face', 'type: 0255ef87-33ce-4f88-b9db-8897f8c17233');

COMMIT;

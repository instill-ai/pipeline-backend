BEGIN;

ALTER TABLE public.pipeline ADD COLUMN number_of_clones INTEGER DEFAULT 0;
CREATE INDEX pipeline_number_of_clones ON public.pipeline (number_of_clones);

UPDATE public.pipeline SET recipe = replace(recipe::TEXT, '', '')::jsonb;


-- The commands are generated via
-- curl -s 'https://INSTILL-CLOUD-HOST/v1beta/component-definitions?pageSize=50' | jq -r ".componentDefinitions.[] | \"UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'type: \" + .uid +\"', 'type: \"+ .id +\"');\""

UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'type: ddcf42c3-4c30-4c65-9585-25f1c89b2b48', 'type: instill-model');
UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'type: 9fb6a2cb-bff5-4c69-bc6d-4538dd8e3362', 'type: openai');
UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'type: 70d8664a-d512-4517-a5e8-5d4da81756a7', 'type: numbers');
UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'type: 4b1dcf82-e134-4ba7-992f-f9a02536ec2b', 'type: pinecone');
UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'type: fd0ad325-f2f7-41f3-b247-6c71d571b1b8', 'type: redis');
UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'type: 28f53d15-6150-46e6-99aa-f76b70a926c0', 'type: json');
UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'type: e9eb8fc8-f249-4e11-ad50-5035d79ffc18', 'type: image');
UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'type: e5b290ae-ad53-47c9-a64e-efbc5358520b', 'type: document');
UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'type: e414a1f8-5fdf-4292-b050-9f9176254a4b', 'type: archetype-ai');
UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'type: e2ffe076-ab2c-4e5e-9587-a613a6b1c146', 'type: bigquery');
UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'type: c86a95cc-7d32-4e22-a290-8c699f6705a4', 'type: stability-ai');
UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'type: 98909958-db7d-4dfe-9858-7761904be17e', 'type: website');
UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'type: 5ee55a5c-6e30-4c7a-80e8-90165a729e0a', 'type: restapi');
UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'type: 5b7aca5b-1ae3-477f-bf60-d34e1c993c87', 'type: text');
UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'type: 3a836447-c211-4134-9cc5-ad45e1cc467e', 'type: base64');
UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'type: 2b1da686-878a-462c-b2c6-a9690199939c', 'type: google-search');
UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'type: 205cbeff-6f45-4abe-b0a8-cec1a310137f', 'type: gcs');
UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'type: 1e9f469e-da5e-46eb-8a89-23466627e3b5', 'type: slack');
UPDATE public.pipeline SET recipe_yaml = replace(recipe_yaml, 'type: 0255ef87-33ce-4f88-b9db-8897f8c17233', 'type: hugging-face');

-- The commands are generated via
-- curl -s 'https://INSTILL-CLOUD-HOST/v1beta/component-definitions?pageSize=50' | jq -r ".componentDefinitions.[] | \"UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'type: \" + .uid +\"', 'type: \"+ .id +\"');\""

UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'type: ddcf42c3-4c30-4c65-9585-25f1c89b2b48', 'type: instill-model');
UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'type: 9fb6a2cb-bff5-4c69-bc6d-4538dd8e3362', 'type: openai');
UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'type: 70d8664a-d512-4517-a5e8-5d4da81756a7', 'type: numbers');
UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'type: 4b1dcf82-e134-4ba7-992f-f9a02536ec2b', 'type: pinecone');
UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'type: fd0ad325-f2f7-41f3-b247-6c71d571b1b8', 'type: redis');
UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'type: 28f53d15-6150-46e6-99aa-f76b70a926c0', 'type: json');
UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'type: e9eb8fc8-f249-4e11-ad50-5035d79ffc18', 'type: image');
UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'type: e5b290ae-ad53-47c9-a64e-efbc5358520b', 'type: document');
UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'type: e414a1f8-5fdf-4292-b050-9f9176254a4b', 'type: archetype-ai');
UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'type: e2ffe076-ab2c-4e5e-9587-a613a6b1c146', 'type: bigquery');
UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'type: c86a95cc-7d32-4e22-a290-8c699f6705a4', 'type: stability-ai');
UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'type: 98909958-db7d-4dfe-9858-7761904be17e', 'type: website');
UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'type: 5ee55a5c-6e30-4c7a-80e8-90165a729e0a', 'type: restapi');
UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'type: 5b7aca5b-1ae3-477f-bf60-d34e1c993c87', 'type: text');
UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'type: 3a836447-c211-4134-9cc5-ad45e1cc467e', 'type: base64');
UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'type: 2b1da686-878a-462c-b2c6-a9690199939c', 'type: google-search');
UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'type: 205cbeff-6f45-4abe-b0a8-cec1a310137f', 'type: gcs');
UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'type: 1e9f469e-da5e-46eb-8a89-23466627e3b5', 'type: slack');
UPDATE public.pipeline_release SET recipe_yaml = replace(recipe_yaml, 'type: 0255ef87-33ce-4f88-b9db-8897f8c17233', 'type: hugging-face');
COMMIT;

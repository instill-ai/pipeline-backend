BEGIN;

UPDATE public.component_definition_index SET component_type='COMPONENT_TYPE_APPLICATION' WHERE id='restapi';

COMMIT;

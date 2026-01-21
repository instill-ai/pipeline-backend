package handler

// *RequiredFields are Protobuf message fields with REQUIRED field_behavior annotation
// Per AIP standards: display_name, visibility, and recipe are required for pipeline creation
var createPipelineRequiredFields = []string{"display_name", "visibility"}
var lookUpPipelineRequiredFields = []string{"permalink"}
var renamePipelineRequiredFields = []string{"name", "new_pipeline_id"}
var triggerPipelineRequiredFields = []string{"name", "data"}

// immutableFields are Protobuf message fields with IMMUTABLE field_behavior annotation
// Per AIP standards: id is now OUTPUT_ONLY (server-generated), not immutable
var immutablePipelineFields = []string{}

// outputOnlyFields are Protobuf message fields with OUTPUT_ONLY field_behavior annotation
// Per AIP standards: id, aliases, slug are server-generated
var outputOnlyPipelineFields = []string{"name", "id", "slug", "aliases", "owner", "create_time", "update_time"}

var releaseCreateRequiredFields = []string{}

// outputOnlyFields are Protobuf message fields with OUTPUT_ONLY field_behavior annotation
// Note: PipelineRelease has no uid field - id is IMMUTABLE, not OUTPUT_ONLY
var releaseOutputOnlyFields = []string{"name", "slug", "aliases", "create_time", "update_time"}

var createSecretRequiredFields = []string{"id", "value"}
var outputOnlySecretFields = []string{"name", "uid", "create_time", "update_time"}
var immutableSecretFields = []string{"id"}

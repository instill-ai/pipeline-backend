package handler

// *RequiredFields are Protobuf message fields with REQUIRED field_behavior annotation
// Per AIP standards: display_name, visibility, and recipe are required for pipeline creation
var createPipelineRequiredFields = []string{"display_name", "visibility"}
var lookUpPipelineRequiredFields = []string{"permalink"}
var renamePipelineRequiredFields = []string{"pipeline_id", "new_pipeline_id"}
var triggerPipelineRequiredFields = []string{"pipeline_id", "data"}

// immutableFields are Protobuf message fields with IMMUTABLE field_behavior annotation
// Per AIP standards: id is now OUTPUT_ONLY (server-generated), not immutable
var immutablePipelineFields = []string{}

// outputOnlyFields are Protobuf message fields with OUTPUT_ONLY field_behavior annotation
// Per AIP standards: id, aliases are server-generated
// Note: slug is OPTIONAL - can be provided by client, or server generates from display_name
var outputOnlyPipelineFields = []string{"name", "id", "aliases", "owner", "create_time", "update_time"}

var releaseCreateRequiredFields = []string{}

// outputOnlyFields are Protobuf message fields with OUTPUT_ONLY field_behavior annotation
var releaseOutputOnlyFields = []string{"name", "uid", "create_time", "update_time"}

var createSecretRequiredFields = []string{"id", "value"}
var outputOnlySecretFields = []string{"name", "uid", "create_time", "update_time"}
var immutableSecretFields = []string{"id"}

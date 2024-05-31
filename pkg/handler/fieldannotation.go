package handler

// *RequiredFields are Protobuf message fields with REQUIRED field_behavior annotation
var createPipelineRequiredFields = []string{}
var lookUpPipelineRequiredFields = []string{"permalink"}
var renamePipelineRequiredFields = []string{"name", "new_pipeline_id"}
var triggerPipelineRequiredFields = []string{"name", "data"}

// immutableFields are Protobuf message fields with IMMUTABLE field_behavior annotation
var immutablePipelineFields = []string{"id"}

// outputOnlyFields are Protobuf message fields with OUTPUT_ONLY field_behavior annotation
var outputOnlyPipelineFields = []string{"name", "uid", "owner", "create_time", "update_time"}

var releaseCreateRequiredFields = []string{}
var releaseRenameRequiredFields = []string{"name", "new_pipeline_release_id"}

// outputOnlyFields are Protobuf message fields with OUTPUT_ONLY field_behavior annotation
var releaseOutputOnlyFields = []string{"name", "uid", "create_time", "update_time"}

var createSecretRequiredFields = []string{"id", "value"}
var outputOnlySecretFields = []string{"name", "uid", "create_time", "update_time"}
var immutableSecretFields = []string{"id"}

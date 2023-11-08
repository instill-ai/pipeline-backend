package handler

// *RequiredFields are Protobuf message fields with REQUIRED field_behavior annotation
var createRequiredFields = []string{}
var lookUpRequiredFields = []string{"permalink"}
var renameRequiredFields = []string{"name", "new_pipeline_id"}
var triggerRequiredFields = []string{"name", "inputs"}

// immutableFields are Protobuf message fields with IMMUTABLE field_behavior annotation
var immutableFields = []string{"id"}

// outputOnlyFields are Protobuf message fields with OUTPUT_ONLY field_behavior annotation
var outputOnlyFields = []string{"name", "uid", "owner", "create_time", "update_time"}

var releaseCreateRequiredFields = []string{}
var releaseRenameRequiredFields = []string{"name", "new_pipeline_release_id"}

// outputOnlyFields are Protobuf message fields with OUTPUT_ONLY field_behavior annotation
var releaseOutputOnlyFields = []string{"name", "uid", "create_time", "update_time"}

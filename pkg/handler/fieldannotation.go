package handler

// *RequiredFields are Protobuf message fields with REQUIRED field_behavior annotation
var createRequiredFields = []string{}
var lookUpRequiredFields = []string{"permalink"}
var activateRequiredFields = []string{"name"}
var deactivateRequiredFields = []string{"name"}
var renameRequiredFields = []string{"name", "new_pipeline_id"}
var triggerRequiredFields = []string{"name", "inputs"}
var triggerBinaryRequiredFields = []string{"name", "file_lengths", "content"}

// immutableFields are Protobuf message fields with IMMUTABLE field_behavior annotation
var immutableFields = []string{"id", "recipe", "recipe.source", "recipe.model_instances", "recipe.destination"}

// outputOnlyFields are Protobuf message fields with OUTPUT_ONLY field_behavior annotation
var outputOnlyFields = []string{"name", "uid", "mode", "state", "owner", "create_time", "update_time"}

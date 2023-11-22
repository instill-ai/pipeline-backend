package handler

// *RequiredFields are Protobuf message fields with REQUIRED field_behavior annotation
var createPipelineRequiredFields = []string{}
var lookUpPipelineRequiredFields = []string{"permalink"}
var renamePipelineRequiredFields = []string{"name", "new_pipeline_id"}
var triggerPipelineRequiredFields = []string{"name", "inputs"}

// immutableFields are Protobuf message fields with IMMUTABLE field_behavior annotation
var immutablePipelineFields = []string{"id"}

// outputOnlyFields are Protobuf message fields with OUTPUT_ONLY field_behavior annotation
var outputOnlyPipelineFields = []string{"name", "uid", "owner", "create_time", "update_time"}

var releaseCreateRequiredFields = []string{}
var releaseRenameRequiredFields = []string{"name", "new_pipeline_release_id"}

// outputOnlyFields are Protobuf message fields with OUTPUT_ONLY field_behavior annotation
var releaseOutputOnlyFields = []string{"name", "uid", "create_time", "update_time"}

// *RequiredFields are Protobuf message fields with REQUIRED field_behavior annotation
var createConnectorRequiredFields = []string{"configuration"}
var lookUpConnectorRequiredFields = []string{"permalink"}
var connectConnectorRequiredFields = []string{"name"}
var disconnectConnectorRequiredFields = []string{"name"}
var renameConnectorRequiredFields = []string{"name", "new_connector_id"}

// *ImmutableFields* are Protobuf message fields with IMMUTABLE field_behavior annotation
var immutableConnectorFields = []string{"id", "connector_definition_name"}

// *OutputOnlyFields are Protobuf message fields with OUTPUT_ONLY field_behavior annotation
var outputOnlyConnectorFields = []string{"name", "uid", "state", "tombstone", "owner", "create_time", "update_time"}

package datamodel

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/instill-ai/pipeline-backend/internal/logger"

	pipelinePB "github.com/instill-ai/protogen-go/pipeline/v1alpha"
)

// PipelineJSONSchema represents the Pipeline JSON Schema for validating the payload
var PipelineJSONSchema *jsonschema.Schema

// InitJSONSchema initialise JSON Schema instances with the given files
func InitJSONSchema() {

	logger, _ := logger.GetZapLogger()

	var err error
	PipelineJSONSchema, err = jsonschema.Compile("config/models/pipeline.json")
	if err != nil {
		logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
	}

}

//ValidatePipelineJSONSchema validates the Protobuf message data
func ValidatePipelineJSONSchema(pbPipeline *pipelinePB.Pipeline) error {

	data, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(pbPipeline)
	if err != nil {
		return err
	}

	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		log.Fatal(err)
	}

	if err := PipelineJSONSchema.Validate(v); err != nil {
		b, _ := json.MarshalIndent(err.(*jsonschema.ValidationError).DetailedOutput(), "", "  ")
		return fmt.Errorf(string(b))
	}

	return nil
}

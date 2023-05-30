package datamodel

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/instill-ai/pipeline-backend/pkg/logger"

	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

// PipelineJSONSchema represents the Pipeline JSON Schema for validating the payload
var PipelineJSONSchema *jsonschema.Schema

// InitJSONSchema initialise JSON Schema instances with the given files
func InitJSONSchema(ctx context.Context) {

	logger, _ := logger.GetZapLogger(ctx)

	var err error
	PipelineJSONSchema, err = jsonschema.Compile("config/model/pipeline.json")
	if err != nil {
		logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
	}

}

// ValidatePipelineJSONSchema validates the Protobuf message data
func ValidatePipelineJSONSchema(pbPipeline *pipelinePB.Pipeline) error {

	b, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(pbPipeline)
	if err != nil {
		return err
	}
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	if err := PipelineJSONSchema.Validate(v); err != nil {
		switch e := err.(type) {
		case *jsonschema.ValidationError:
			b, err := json.MarshalIndent(e.DetailedOutput(), "", "  ")
			if err != nil {
				return err
			}
			return fmt.Errorf(string(b))
		case jsonschema.InvalidJSONTypeError:
			return e
		default:
			return e
		}
	}

	return nil
}

package instillmodel

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"

	modelpb "github.com/instill-ai/protogen-go/model/v1alpha"
)

func (e *execution) executeVisionTask(grpcClient modelpb.ModelPublicServiceClient, nsID string, modelID string, version string, inputs []*structpb.Struct) ([]*structpb.Struct, error) {
	if len(inputs) <= 0 {
		return nil, fmt.Errorf("invalid input: %v for model: %s/%s/%s", inputs, nsID, modelID, version)
	}

	if grpcClient == nil {
		return nil, fmt.Errorf("uninitialized client")
	}

	// Transform inputs to standardized format
	taskInputs := []*structpb.Struct{}
	for _, input := range inputs {
		// Parse the input using VisionInput struct
		var inputStruct VisionInput
		err := base.ConvertFromStructpb(input, &inputStruct)
		if err != nil {
			return nil, fmt.Errorf("failed to convert input to VisionInput struct: %w", err)
		}

		// Create request data
		requestData := &VisionRequestData{
			ImageBase64: util.TrimBase64Mime(inputStruct.ImageBase64),
			Type:        "image-base64",
		}

		// Create standardized request wrapper
		requestWrapper := &RequestWrapper{
			Data: requestData,
		}

		// Convert to structpb
		taskInput, err := base.ConvertToStructpb(requestWrapper)
		if err != nil {
			return nil, fmt.Errorf("failed to convert request wrapper to structpb: %w", err)
		}
		taskInputs = append(taskInputs, taskInput)
	}

	taskOutputs, err := trigger(grpcClient, e.SystemVariables, nsID, modelID, version, taskInputs)
	if err != nil {
		return nil, err
	}
	if len(taskOutputs) <= 0 {
		return nil, fmt.Errorf("invalid output: %v for model: %s/%s/%s", taskOutputs, nsID, modelID, version)
	}

	// For vision tasks, we return the raw output data structure
	// since different vision tasks have different output formats
	outputs := []*structpb.Struct{}
	for idx := range inputs {
		outputs = append(outputs, taskOutputs[idx].Fields["data"].GetStructValue())
	}
	return outputs, nil
}

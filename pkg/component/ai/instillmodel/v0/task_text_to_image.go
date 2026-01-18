package instillmodel

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"

	modelpb "github.com/instill-ai/protogen-go/model/v1alpha"
)

func (e *execution) executeTextToImage(grpcClient modelpb.ModelPublicServiceClient, nsID string, modelID string, version string, inputs []*structpb.Struct) ([]*structpb.Struct, error) {
	if len(inputs) <= 0 {
		return nil, fmt.Errorf("invalid input: %v for model: %s/%s/%s", inputs, nsID, modelID, version)
	}

	if grpcClient == nil {
		return nil, fmt.Errorf("uninitialized client")
	}

	// Transform inputs to standardized format
	taskInputs := []*structpb.Struct{}
	for _, input := range inputs {
		// Parse the input using TextToImageInput struct
		var inputStruct TextToImageInput
		err := base.ConvertFromStructpb(input, &inputStruct)
		if err != nil {
			return nil, fmt.Errorf("failed to convert input to TextToImageInput struct: %w", err)
		}

		// Create request data
		requestData := &TextToImageRequestData{
			Prompt: inputStruct.Prompt,
		}

		// Create request parameters
		requestParam := &TextToImageRequestParameter{}
		if inputStruct.Samples != nil {
			requestParam.N = *inputStruct.Samples
		}
		if inputStruct.Seed != nil {
			requestParam.Seed = *inputStruct.Seed
		}
		if inputStruct.AspectRatio != nil {
			requestParam.AspectRatio = *inputStruct.AspectRatio
		}
		if inputStruct.NegativePrompt != nil {
			requestParam.NegativePrompt = *inputStruct.NegativePrompt
		}

		// Create standardized request wrapper
		requestWrapper := &RequestWrapper{
			Data:      requestData,
			Parameter: requestParam,
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

	// Transform raw outputs to standardized TextToImageOutput format
	outputs := []*structpb.Struct{}
	for idx := range inputs {
		choices := taskOutputs[idx].Fields["data"].GetStructValue().Fields["choices"].GetListValue()

		// For now, return a simple structure to avoid the hanging issue
		// TODO: Implement proper image handling
		outputStruct := &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"images": structpb.NewListValue(&structpb.ListValue{
					Values: make([]*structpb.Value, len(choices.Values)),
				}),
			},
		}

		outputs = append(outputs, outputStruct)
	}

	return outputs, nil
}

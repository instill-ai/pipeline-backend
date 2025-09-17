package instillmodel

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"

	modelpb "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

func (e *execution) executeTextGeneration(grpcClient modelpb.ModelPublicServiceClient, nsID string, modelID string, version string, inputs []*structpb.Struct) ([]*structpb.Struct, error) {
	if len(inputs) <= 0 {
		return nil, fmt.Errorf("invalid input: %v for model: %s/%s/%s", inputs, nsID, modelID, version)
	}

	if grpcClient == nil {
		return nil, fmt.Errorf("uninitialized client")
	}

	// Transform inputs to standardized format
	taskInputs := []*structpb.Struct{}
	for _, input := range inputs {
		// Parse the input using TextGenerationInput struct
		var inputStruct TextGenerationInput
		err := base.ConvertFromStructpb(input, &inputStruct)
		if err != nil {
			return nil, fmt.Errorf("failed to convert input to TextGenerationInput struct: %w", err)
		}

		// Create request data
		requestData := &TextGenerationRequestData{
			Prompt: inputStruct.Prompt,
		}
		if inputStruct.SystemMessage != nil {
			requestData.SystemMessage = *inputStruct.SystemMessage
		}

		// Create request parameters
		requestParam := &TextGenerationRequestParameter{
			N: 1,
		}
		if inputStruct.Seed != nil {
			requestParam.Seed = *inputStruct.Seed
		}
		if inputStruct.MaxNewTokens != nil {
			requestParam.MaxTokens = *inputStruct.MaxNewTokens
		}
		if inputStruct.Temperature != nil {
			requestParam.Temperature = *inputStruct.Temperature
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

	// Transform raw outputs to standardized TextGenerationOutput format
	outputs := []*structpb.Struct{}
	for idx := range inputs {
		choices := taskOutputs[idx].Fields["data"].GetStructValue().Fields["choices"].GetListValue()

		// Create standardized output structure
		textGenOutput := TextGenerationOutput{
			Text: choices.GetValues()[0].GetStructValue().Fields["content"].GetStringValue(),
		}

		// Convert to structpb
		outputStruct, err := base.ConvertToStructpb(textGenOutput)
		if err != nil {
			return nil, fmt.Errorf("failed to convert text generation output to structpb: %w", err)
		}

		outputs = append(outputs, outputStruct)
	}

	return outputs, nil
}

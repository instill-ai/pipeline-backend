package instillmodel

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	modelpb "github.com/instill-ai/protogen-go/model/v1alpha"
)

func (e *execution) executeEmbedding(grpcClient modelpb.ModelPublicServiceClient, nsID string, modelID string, version string, inputs []*structpb.Struct) ([]*structpb.Struct, error) {

	if len(inputs) <= 0 {
		return nil, fmt.Errorf("invalid input: %v for model: %s/%s/%s", inputs, nsID, modelID, version)
	}

	if grpcClient == nil {
		return nil, fmt.Errorf("uninitialized client")
	}

	// Transform inputs to standardized format
	taskInputs := []*structpb.Struct{}
	for _, input := range inputs {
		// Parse the input using EmbeddingInput struct
		var inputStruct EmbeddingInput
		err := base.ConvertFromStructpb(input, &inputStruct)
		if err != nil {
			return nil, fmt.Errorf("failed to convert input to EmbeddingInput struct: %w", err)
		}

		// Create standardized request wrapper
		requestWrapper := &RequestWrapper{
			Data:      inputStruct.Data,
			Parameter: inputStruct.Parameter,
		}

		// Convert to structpb
		taskInput, err := base.ConvertToStructpb(requestWrapper)
		if err != nil {
			return nil, fmt.Errorf("failed to convert request wrapper to structpb: %w", err)
		}
		taskInputs = append(taskInputs, taskInput)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	ctx = metadata.NewOutgoingContext(ctx, getRequestMetadata(e.SystemVariables))

	res, err := grpcClient.TriggerNamespaceModel(ctx, &modelpb.TriggerNamespaceModelRequest{
		NamespaceId: nsID,
		ModelId:     modelID,
		Version:     version,
		TaskInputs:  taskInputs,
	})

	if err != nil || res == nil {
		return nil, fmt.Errorf("error triggering model: %v", err)
	}

	if len(res.TaskOutputs) == 0 {
		return nil, fmt.Errorf("no output from model")
	}

	// Transform raw outputs to standardized EmbeddingOutput format
	outputs := make([]*structpb.Struct, len(res.TaskOutputs))
	for idx, taskOutput := range res.TaskOutputs {
		// Extract embeddings from the raw response
		rawEmbeddings := taskOutput.Fields["data"].GetStructValue().Fields["embeddings"].GetListValue()

		// Convert to standardized format
		standardizedEmbeddings := make([]OutputEmbedding, 0, len(rawEmbeddings.Values))
		for i, rawEmb := range rawEmbeddings.Values {
			embStruct := rawEmb.GetStructValue()

			// Extract vector data
			vectorList := embStruct.Fields["embedding"].GetListValue()
			vector := make([]any, len(vectorList.Values))
			for j, v := range vectorList.Values {
				vector[j] = v.GetNumberValue()
			}

			// Create standardized embedding
			outputEmb := OutputEmbedding{
				Index:   i,
				Vector:  vector,
				Created: int(time.Now().Unix()), // Use current timestamp if not provided
			}

			// Use created timestamp from response if available
			if createdField, ok := embStruct.Fields["created"]; ok {
				outputEmb.Created = int(createdField.GetNumberValue())
			}

			standardizedEmbeddings = append(standardizedEmbeddings, outputEmb)
		}

		// Create standardized output structure
		embeddingOutput := EmbeddingOutput{
			Data: EmbeddingOutputData{
				Embeddings: standardizedEmbeddings,
			},
		}

		// Convert to structpb
		outputStruct, err := base.ConvertToStructpb(embeddingOutput)
		if err != nil {
			return nil, fmt.Errorf("failed to convert embedding output to structpb: %w", err)
		}

		outputs[idx] = outputStruct
	}

	return outputs, nil
}

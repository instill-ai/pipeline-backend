package instillmodel

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"

	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

func (e *execution) trigger(ctx context.Context, job *base.Job) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	ctx = metadata.NewOutgoingContext(ctx, getRequestMetadata(e.SystemVariables))

	// Read() is deprecated and will be removed in a future version.
	// However, Instill Model is still using structpb.Struct for input and output.
	// When we move out Read(), we will need to update the Instill Model as well.
	input, err := job.Input.Read(ctx)

	if err != nil {
		return fmt.Errorf("reading input data: %w", err)
	}

	triggerInfo, err := getTriggerInfo(input)

	if err != nil {
		return fmt.Errorf("getting trigger info: %w", err)
	}

	grpcClient := e.client

	res, err := grpcClient.TriggerNamespaceModel(ctx, &modelPB.TriggerNamespaceModelRequest{
		NamespaceId: triggerInfo.nsID,
		ModelId:     triggerInfo.modelID,
		Version:     triggerInfo.version,
		TaskInputs:  []*structpb.Struct{input},
	})

	if err != nil {
		return fmt.Errorf("triggering model: %v", err)
	}

	if res == nil || len(res.TaskOutputs) == 0 {
		return fmt.Errorf("triggering model: get empty task outputs")
	}

	// Write() is deprecated and will be removed in a future version.
	// However, Instill Model is still using structpb.Struct for input and output.
	// When we move out Write(), we will need to update the Instill Model as well.
	err = job.Output.Write(ctx, res.TaskOutputs[0])

	if err != nil {
		return fmt.Errorf("writing output data: %w", err)
	}

	return nil
}

func getTriggerInfo(input *structpb.Struct) (*triggerInfo, error) {
	if input == nil {
		return nil, fmt.Errorf("input is nil")
	}
	data, ok := input.Fields["data"]
	if !ok {
		return nil, fmt.Errorf("data field not found")
	}
	model, ok := data.GetStructValue().Fields["model"]

	if !ok {
		return nil, fmt.Errorf("model field not found")
	}
	modelNameSplits := strings.Split(model.GetStringValue(), "/")
	if len(modelNameSplits) != 3 {
		return nil, fmt.Errorf("model name should be in the format of <namespace>/<model>/<version>")
	}
	return &triggerInfo{
		nsID:    modelNameSplits[0],
		modelID: modelNameSplits[1],
		version: modelNameSplits[2],
	}, nil
}
